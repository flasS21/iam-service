package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func GinRequireAuth(auth *AuthMiddleware) gin.HandlerFunc {
	return func(c *gin.Context) {

		authorized := false

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authorized = true
			c.Request = r
		})

		handler := auth.RequireAuth(next)

		handler.ServeHTTP(c.Writer, c.Request)

		if !authorized {
			// Auth failed → stop chain
			c.Abort()
			return
		}

		// Auth succeeded → continue Gin chain
		c.Next()
	}
}
