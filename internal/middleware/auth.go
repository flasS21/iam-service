package middleware

import (
	"context"
	"log"
	"net/http"
	"time"

	"iam-service/internal/session"
)

// unexported, collision-proof context key
type userIDContextKeyType struct{}

var userIDKey = userIDContextKeyType{}

const (
	IdleTimeout = 30 * time.Minute
	// IdleTimeout = 100 * time.Second
)

// UserIDFromContext extracts the authenticated user ID from context.
func UserIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDKey).(string)
	return id, ok
}

type AuthMiddleware struct {
	Store session.Store
}

func NewAuthMiddleware(store session.Store) *AuthMiddleware {
	return &AuthMiddleware{Store: store}
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
			// http.Error(w, "unauthorized", http.StatusUnauthorized)
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

		if sess != nil {
			log.Printf("event=session_loaded sid=%s user_id=%s expires_at=%s",
				sess.SessionID,
				sess.UserID,
				sess.ExpiresAt.UTC(),
			)
		}

		// 3. Keystone fix: enforce session expiry

		// now := time.Now()
		// if now.After(sess.AbsoluteExpiresAt) {
		// 	_ = a.Store.Delete(r.Context(), sessionID)
		// 	w.WriteHeader(http.StatusUnauthorized)
		// 	return
		// }

		if time.Now().After(sess.ExpiresAt) {

			log.Printf("event=session_expired sid=%s user_id=%s now=%s expires_at=%s",
				sess.SessionID,
				sess.UserID,
				time.Now().UTC(),
				sess.ExpiresAt.UTC(),
			)

			_ = a.Store.Delete(r.Context(), sessionID)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// 3.5 Sliding window: always extend on activity
		newExpiry := time.Now().Add(IdleTimeout)

		log.Printf("event=session_extend sid=%s user_id=%s old_expiry=%s new_expiry=%s",
			sess.SessionID,
			sess.UserID,
			sess.ExpiresAt.UTC(),
			newExpiry.UTC(),
		)

		sess.ExpiresAt = newExpiry
		_ = a.Store.Update(r.Context(), *sess)

		// 4. Attach user_id to context
		ctx := context.WithValue(r.Context(), userIDKey, sess.UserID)

		log.Printf("event=session_authorized sid=%s user_id=%s path=%s",
			sess.SessionID,
			sess.UserID,
			r.URL.Path,
		)

		// 5. Continue request
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
