package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GinRequireAuth adapts the net/http AuthMiddleware to Gin.
// It preserves the Keystone rule that auth decisions are
// session-based and provider-agnostic.
func GinRequireAuth(auth *AuthMiddleware) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Bridge handler to allow net/http middleware execution
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Request = r
			c.Next()
		})

		// Wrap Gin request with net/http auth middleware
		handler := auth.RequireAuth(next)

		// Execute middleware chain
		handler.ServeHTTP(c.Writer, c.Request)

		// If auth middleware already handled the response, stop Gin chain
		if c.Writer.Written() {
			c.Abort()
			return
		}
	}
}
