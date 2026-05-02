package middleware

import (
	"context"
	"net/http"
	"time"

	"iam-service/internal/auth/resolver"
	"iam-service/internal/logger"
	"iam-service/internal/session"
)

/*
auth.go provides session-based authentication middleware with idle timeout and sliding window expiry.
RequireAuth() validates session cookies, enforces 30-minute idle timeout, extends session on activity,
and attaches user ID to request context. UserIDFromContext() extracts authenticated user ID from context.
*/

// unexported, collision-proof context key
type userIDContextKeyType struct{}

var userIDKey = userIDContextKeyType{}

const (
	IdleTimeout = 5 * time.Minute
)

// UserIDFromContext extracts the authenticated user ID from context.
func UserIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDKey).(string)
	return id, ok
}

type AuthMiddleware struct {
	Store    session.Store
	Resolver resolver.Resolver
}

func NewAuthMiddleware(store session.Store, resolver resolver.Resolver) *AuthMiddleware {
	return &AuthMiddleware{
		Store:    store,
		Resolver: resolver,
	}
}

func (a *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		cookie, err := r.Cookie(session.CookieName)

		logger.Info("session cookie read", map[string]any{
			"path":       r.URL.Path,
			"ip":         r.RemoteAddr,
			"has_cookie": err == nil && cookie.Value != "",
		})

		if err != nil || cookie.Value == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		sessionID := cookie.Value

		sess, err := a.Store.Get(r.Context(), sessionID)
		if err != nil || sess == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		logger.Info("session loaded", map[string]any{
			"session_id":        sess.SessionID,
			"user_id":           sess.UserID,
			"expires_at":        sess.ExpiresAt.UTC(),
			"absolute_expires":  sess.AbsoluteExpiresAt.UTC(),
		})

		currentVersion, err := a.Resolver.GetSessionVersion(r.Context(), sess.UserID)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if sess.Version != currentVersion {
			logger.Warn("session version mismatch", map[string]any{
				"session_id":      sess.SessionID,
				"user_id":         sess.UserID,
				"session_version": sess.Version,
				"current_version": currentVersion,
			})

			_ = a.Store.Delete(r.Context(), sessionID)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		status, err := a.Resolver.GetUserStatus(r.Context(), sess.UserID)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if status != "active" {
			logger.Warn("user disabled", map[string]any{
				"session_id": sess.SessionID,
				"user_id":    sess.UserID,
				"status":     status,
			})

			_ = a.Store.Delete(r.Context(), sessionID)

			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if r.Method == http.MethodPost ||
			r.Method == http.MethodPut ||
			r.Method == http.MethodPatch ||
			r.Method == http.MethodDelete {

			headerToken := r.Header.Get("X-CSRF-Token")

			if headerToken == "" || headerToken != sess.CSRFToken {
				logger.Warn("csrf mismatch", map[string]any{
					"session_id": sess.SessionID,
					"user_id":    sess.UserID,
					"method":     r.Method,
				})

				w.WriteHeader(http.StatusForbidden)
				return
			}
		}

		now := time.Now()

		if now.After(sess.AbsoluteExpiresAt) {
			logger.Info("session absolute expired", map[string]any{
				"session_id":      sess.SessionID,
				"user_id":         sess.UserID,
				"now":             now.UTC(),
				"absolute_expiry": sess.AbsoluteExpiresAt.UTC(),
			})

			_ = a.Store.Delete(r.Context(), sessionID)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if now.After(sess.ExpiresAt) {
			logger.Info("session idle expired", map[string]any{
				"session_id": sess.SessionID,
				"user_id":    sess.UserID,
				"now":        now.UTC(),
				"expires_at": sess.ExpiresAt.UTC(),
			})

			_ = a.Store.Delete(r.Context(), sessionID)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		newIdleExpiry := now.Add(IdleTimeout)

		var newExpiry time.Time
		if newIdleExpiry.After(sess.AbsoluteExpiresAt) {
			newExpiry = sess.AbsoluteExpiresAt
		} else {
			newExpiry = newIdleExpiry
		}

		logger.Info("session extend", map[string]any{
			"session_id":      sess.SessionID,
			"user_id":         sess.UserID,
			"old_expiry":      sess.ExpiresAt.UTC(),
			"new_expiry":      newExpiry.UTC(),
			"absolute_expiry": sess.AbsoluteExpiresAt.UTC(),
		})

		sess.ExpiresAt = newExpiry
		_ = a.Store.Update(r.Context(), *sess)

		ctx := context.WithValue(r.Context(), userIDKey, sess.UserID)

		logger.Info("session authorized", map[string]any{
			"session_id": sess.SessionID,
			"user_id":    sess.UserID,
			"path":       r.URL.Path,
		})

		next.ServeHTTP(w, r.WithContext(ctx))

	})
}
