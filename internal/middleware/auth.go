package middleware

import (
	"context"
	"log"
	"net/http"
	"time"

	"iam-service/internal/auth/resolver"
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
	// IdleTimeout = 30 * time.Minute
	// IdleTimeout = 100 * time.Second
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

		// 1. Read session cookie
		cookie, err := r.Cookie(session.CookieName)

		log.Printf("event=session_cookie_read path=%s ip=%s has_cookie=%t",
			r.URL.Path,
			r.RemoteAddr,
			err == nil && cookie.Value != "",
		)

		if err != nil || cookie.Value == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		sessionID := cookie.Value

		// 2. Load session
		sess, err := a.Store.Get(r.Context(), sessionID)
		if err != nil || sess == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		log.Printf("event=session_loaded sid=%s user_id=%s expires_at=%s absolute_expires_at=%s",
			sess.SessionID,
			sess.UserID,
			sess.ExpiresAt.UTC(),
			sess.AbsoluteExpiresAt.UTC(),
		)

		// 🔐 Session version enforcement
		currentVersion, err := a.Resolver.GetSessionVersion(r.Context(), sess.UserID)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if sess.Version != currentVersion {
			log.Printf(
				"event=session_version_mismatch sid=%s user_id=%s session_version=%d current_version=%d",
				sess.SessionID,
				sess.UserID,
				sess.Version,
				currentVersion,
			)

			_ = a.Store.Delete(r.Context(), sessionID)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// 🔐 CSRF enforcement for state-changing methods
		if r.Method == http.MethodPost ||
			r.Method == http.MethodPut ||
			r.Method == http.MethodPatch ||
			r.Method == http.MethodDelete {

			headerToken := r.Header.Get("X-CSRF-Token")

			if headerToken == "" || headerToken != sess.CSRFToken {
				log.Printf(
					"event=csrf_mismatch sid=%s user_id=%s method=%s",
					sess.SessionID,
					sess.UserID,
					r.Method,
				)

				w.WriteHeader(http.StatusForbidden)
				return
			}
		}

		now := time.Now()

		// 3. Hard absolute expiry
		if now.After(sess.AbsoluteExpiresAt) {
			log.Printf("event=session_absolute_expired sid=%s user_id=%s now=%s absolute_expiry=%s",
				sess.SessionID,
				sess.UserID,
				now.UTC(),
				sess.AbsoluteExpiresAt.UTC(),
			)

			_ = a.Store.Delete(r.Context(), sessionID)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// 4. Idle expiry check
		if now.After(sess.ExpiresAt) {

			log.Printf("event=session_idle_expired sid=%s user_id=%s now=%s expires_at=%s",
				sess.SessionID,
				sess.UserID,
				now.UTC(),
				sess.ExpiresAt.UTC(),
			)

			_ = a.Store.Delete(r.Context(), sessionID)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// 5. Sliding window (bounded by absolute cap)
		newIdleExpiry := now.Add(IdleTimeout)

		var newExpiry time.Time
		if newIdleExpiry.After(sess.AbsoluteExpiresAt) {
			newExpiry = sess.AbsoluteExpiresAt
		} else {
			newExpiry = newIdleExpiry
		}

		log.Printf("event=session_extend sid=%s user_id=%s old_expiry=%s new_expiry=%s absolute_expiry=%s",
			sess.SessionID,
			sess.UserID,
			sess.ExpiresAt.UTC(),
			newExpiry.UTC(),
			sess.AbsoluteExpiresAt.UTC(),
		)

		sess.ExpiresAt = newExpiry
		_ = a.Store.Update(r.Context(), *sess)

		// 6. Attach user_id to context
		ctx := context.WithValue(r.Context(), userIDKey, sess.UserID)

		log.Printf("event=session_authorized sid=%s user_id=%s path=%s",
			sess.SessionID,
			sess.UserID,
			r.URL.Path,
		)

		// 7. Continue request
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
