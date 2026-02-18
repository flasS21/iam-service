package handler

import (
	"log"
	"net/http"
	"time"

	"iam-service/internal/auth/provider"
	"iam-service/internal/auth/resolver"
	"iam-service/internal/logger"
	"iam-service/internal/session"

	"github.com/gin-gonic/gin"
)

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
	r.POST("/auth/logout", h.Logout)
	r.POST("/auth/logout-all", h.LogoutAll)

	for _, route := range r.Routes() {
		log.Printf("[ROUTE] %s %s", route.Method, route.Path)
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
		// c.JSON(http.StatusUnauthorized, gin.H{
		// 	"error": "invalid state",
		// })
		clearAuthArtifacts(c)
		h.redirectToKeycloakLogin(c)
		return
	}

	errParam := c.Query("error")
	errDesc := c.Query("error_description")

	// CASE 1: OAuth error (very common during registration)
	if errParam != "" {
		logger.Warn("oidc callback returned error", map[string]any{
			"error": errParam,
			"desc":  errDesc,
		})
		// 	c.Redirect(http.StatusFound, "/login")
		// 	return
		clearAuthArtifacts(c)
		h.redirectToKeycloakLogin(c)
		return
	}

	// CASE 2: Normal OAuth callback
	code := c.Query("code")
	if code == "" {
		// logger.Error("oidc callback missing code and error", nil)
		// c.AbortWithStatus(http.StatusBadRequest)
		logger.Error("oidc callback missing code and error", nil)
		clearAuthArtifacts(c)
		h.redirectToKeycloakLogin(c)
		return
	}

	codeVerifier := getPKCEVerifier(c)
	if codeVerifier == "" {
		// c.JSON(http.StatusUnauthorized, gin.H{
		// 	"error": "missing pkce verifier",
		// })
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
		// c.JSON(http.StatusUnauthorized, gin.H{
		// 	"error": "authentication failed",
		// })
		clearAuthArtifacts(c)
		h.redirectToKeycloakLogin(c)
		return
	}

	userID, err := h.resolver.Resolve(c.Request.Context(), identity)
	if err != nil {
		// c.JSON(http.StatusInternalServerError, gin.H{
		// 	"error": "failed to resolve user",
		// })
		clearAuthArtifacts(c)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	sessionID, err := session.GenerateID()
	if err != nil {
		// c.JSON(http.StatusInternalServerError, gin.H{
		// 	"error": "failed to create session",
		// })
		clearAuthArtifacts(c)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	now := time.Now()
	absoluteExpiry := now.Add(24 * time.Hour)

	sess := session.Session{
		SessionID:         sessionID,
		UserID:            userID,
		CreatedAt:         now,
		AbsoluteExpiresAt: absoluteExpiry,
		ExpiresAt:         absoluteExpiry,
	}

	if err := h.sessionStore.Create(c.Request.Context(), sess); err != nil {
		// c.JSON(http.StatusInternalServerError, gin.H{
		// 	"error": "failed to persist session",
		// })
		clearAuthArtifacts(c)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	session.SetCookie(c.Writer, sessionID, absoluteExpiry, session.CookieOptions{
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	log.Printf("[LOGIN_SUCCESS] user_id=%s sid=%s ip=%s",
		userID,
		sessionID,
		c.ClientIP(),
	)

	// c.JSON(http.StatusOK, gin.H{
	// 	"status": "authenticated",
	// })
	clearAuthArtifacts(c)
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Redirect(http.StatusFound, "/dashboard")

	// W E B - T E S T
	// c.Redirect(http.StatusFound, "/dashboard.html")

}

func (h *Handler) Logout(c *gin.Context) {

	log.Printf("[REQ] %s %s", c.Request.Method, c.Request.URL.Path)

	// 1. Read session cookie (same pattern as auth middleware)
	cookie, err := c.Request.Cookie(session.CookieName)
	if err == nil && cookie.Value != "" {
		// 2. Delete session from store (best-effort)
		_ = h.sessionStore.Delete(c.Request.Context(), cookie.Value)
		// D E B U G - L O G O U T
		log.Printf(
			"[LOGOUT] session_id=%s ip=%s",
			cookie.Value,
			c.ClientIP(),
		)
	}

	// 3. Clear cookie (must pass options)
	session.ClearCookie(c.Writer, session.CookieOptions{
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	// 4. Idempotent response
	c.Status(http.StatusNoContent)
}

func (h *Handler) LogoutAll(c *gin.Context) {

	log.Printf("[REQ] %s %s", c.Request.Method, c.Request.URL.Path)

	// Read current session to determine user
	cookie, err := c.Request.Cookie(session.CookieName)
	if err != nil || cookie.Value == "" {
		c.Status(http.StatusNoContent)
		return
	}

	ctx := c.Request.Context()

	// Load current session to get userID
	sess, err := h.sessionStore.Get(ctx, cookie.Value)
	if err != nil || sess == nil {
		c.Status(http.StatusNoContent)
		return
	}

	userID := sess.UserID

	// Type assert to RedisStore
	redisStore, ok := h.sessionStore.(*session.RedisStore)
	if !ok {
		c.Status(http.StatusInternalServerError)
		return
	}

	userKey := "user_sessions:" + userID

	// Get all session IDs
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

	log.Printf("[LOGOUT_ALL] user_id=%s sessions=%d ip=%s",
		userID,
		len(sids),
		c.ClientIP(),
	)

	// Clear current cookie
	session.ClearCookie(c.Writer, session.CookieOptions{
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
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
