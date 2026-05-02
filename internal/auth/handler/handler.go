package handler

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"time"

	"iam-service/internal/auth/provider"
	"iam-service/internal/auth/resolver"
	"iam-service/internal/logger"
	"iam-service/internal/middleware"
	"iam-service/internal/session"

	"github.com/gin-gonic/gin"
)

/*
Handler manages OAuth/OIDC authentication flow and session lifecycle.
Orchestrates login initiation, OAuth callback handling with state/PKCE validation,
code exchange with identity provider, session creation in Redis, and logout operations.
Supports single session logout and logout-all across user devices via Redis pipeline.
*/
type Handler struct {
	providers    *provider.Registry
	sessionStore session.Store
	resolver     resolver.Resolver
}

func NewHandler(
	registry *provider.Registry,
	sessionStore session.Store,
	resolver resolver.Resolver,
) *Handler {
	return &Handler{
		providers:    registry,
		sessionStore: sessionStore,
		resolver:     resolver,
	}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.GET("/oauth/login", h.login)
	r.GET("/oauth/callback", h.callback)
	for _, route := range r.Routes() {
		logger.Info("route registered", map[string]any{
			"method": route.Method,
			"path":   route.Path,
		})
	}
}

func (h *Handler) login(c *gin.Context) {

	p := h.providers.Get()

	state := generateState(c)
	_, codeChallenge := generatePKCE(c)

	authURL := p.AuthCodeURL(state, codeChallenge)
	c.Redirect(http.StatusFound, authURL)
}

func (h *Handler) callback(c *gin.Context) {
	p := h.providers.Get()

	if !validateState(c) {
		clearAuthArtifacts(c)
		h.redirectToKeycloakLogin(c)
		return
	}

	errParam := c.Query("error")
	errDesc := c.Query("error_description")

	if errParam != "" {
		logger.Warn("oidc callback returned error", map[string]any{
			"error": errParam,
			"desc":  errDesc,
		})
		clearAuthArtifacts(c)
		h.redirectToKeycloakLogin(c)
		return
	}

	code := c.Query("code")
	if code == "" {
		logger.Error("oidc callback missing code and error", nil)
		clearAuthArtifacts(c)
		h.redirectToKeycloakLogin(c)
		return
	}

	codeVerifier := getPKCEVerifier(c)
	if codeVerifier == "" {
		clearAuthArtifacts(c)
		h.redirectToKeycloakLogin(c)
		return
	}

	identity, err := p.ExchangeCode(
		c.Request.Context(),
		code,
		codeVerifier,
	)
	if err != nil {
		clearAuthArtifacts(c)
		h.redirectToKeycloakLogin(c)
		return
	}

	userID, err := h.resolver.Resolve(c.Request.Context(), identity)
	if err != nil {
		clearAuthArtifacts(c)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	status, err := h.resolver.GetUserStatus(c.Request.Context(), userID)
	if err != nil {
		clearAuthArtifacts(c)
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	if status != "active" {
		logger.Warn("login blocked for disabled user", logger.WithRequestID(c, map[string]any{
			"user_id": userID,
			"status":  status,
			"ip":      c.ClientIP(),
		}))

		clearAuthArtifacts(c)
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	version, err := h.resolver.GetSessionVersion(c.Request.Context(), userID)
	if err != nil {
		clearAuthArtifacts(c)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	sessionID, err := session.GenerateID()
	if err != nil {
		clearAuthArtifacts(c)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	csrfToken, err := generateCSRFToken()
	if err != nil {
		clearAuthArtifacts(c)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	now := time.Now()
	absoluteExpiry := now.Add(10 * time.Minute)
	idleExpiry := now.Add(middleware.IdleTimeout)

	sess := session.Session{
		SessionID:         sessionID,
		UserID:            userID,
		CreatedAt:         now,
		AbsoluteExpiresAt: absoluteExpiry,
		ExpiresAt:         idleExpiry,
		Version:           version,
		CSRFToken:         csrfToken,
	}

	if err := h.sessionStore.Create(c.Request.Context(), sess); err != nil {
		clearAuthArtifacts(c)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	session.SetCookie(c.Writer, sessionID, absoluteExpiry, session.CookieOptions{
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "csrf_token",
		Value:    csrfToken,
		Path:     "/",
		HttpOnly: false, // must be readable by JS
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Expires:  absoluteExpiry,
	})

	logger.Info("login success", logger.WithRequestID(c, map[string]any{
		"user_id":    userID,
		"session_id": sessionID,
		"ip":         c.ClientIP(),
	}))

	clearAuthArtifacts(c)
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Redirect(http.StatusFound, "/dashboard")
}

func (h *Handler) Logout(c *gin.Context) {
	logger.Info("logout request", logger.WithRequestID(c, map[string]any{
		"method": c.Request.Method,
		"path":   c.Request.URL.Path,
	}))

	cookie, err := c.Request.Cookie(session.CookieName)
	if err == nil && cookie.Value != "" {
		_ = h.sessionStore.Delete(c.Request.Context(), cookie.Value)
		logger.Info("logout success", logger.WithRequestID(c, map[string]any{
			"session_id": cookie.Value,
			"ip":         c.ClientIP(),
		}))
	}

	session.ClearCookie(c.Writer, session.CookieOptions{
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "csrf_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: false,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	c.Status(http.StatusNoContent)
}

func (h *Handler) LogoutAll(c *gin.Context) {
	logger.Info("logout-all request", logger.WithRequestID(c, map[string]any{
		"method": c.Request.Method,
		"path":   c.Request.URL.Path,
	}))

	cookie, err := c.Request.Cookie(session.CookieName)
	if err != nil || cookie.Value == "" {
		c.Status(http.StatusNoContent)
		return
	}

	ctx := c.Request.Context()

	sess, err := h.sessionStore.Get(ctx, cookie.Value)
	if err != nil || sess == nil {
		c.Status(http.StatusNoContent)
		return
	}

	userID := sess.UserID

	redisStore, ok := h.sessionStore.(*session.RedisStore)
	if !ok {
		c.Status(http.StatusInternalServerError)
		return
	}

	userKey := "user_sessions:" + userID

	sids, err := redisStore.Client().SMembers(ctx, userKey).Result()
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	pipe := redisStore.Client().TxPipeline()

	for _, sid := range sids {
		pipe.Del(ctx, "session:"+sid)
	}

	pipe.Del(ctx, userKey)

	_, _ = pipe.Exec(ctx)

	logger.Info("logout-all success", logger.WithRequestID(c, map[string]any{
		"user_id":  userID,
		"sessions": len(sids),
		"ip":       c.ClientIP(),
	}))

	session.ClearCookie(c.Writer, session.CookieOptions{
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "csrf_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: false,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	c.Status(http.StatusNoContent)
}

func clearAuthArtifacts(c *gin.Context) {

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "pkce_verifier",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) redirectToKeycloakLogin(c *gin.Context) {

	keycloakProvider := h.providers.Get()

	state := generateState(c)
	_, codeChallenge := generatePKCE(c)

	authURL := keycloakProvider.AuthCodeURL(state, codeChallenge)
	c.Redirect(http.StatusFound, authURL)
}

func generateCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
