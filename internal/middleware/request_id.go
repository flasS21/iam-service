package middleware

import (
	"github.com/gin-gonic/gin"
)

const RequestIDKey = "request_id"

func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		requestID := c.GetHeader("X-Request-ID")

		if requestID == "" {
			requestID = "unknown"
		}

		c.Set(RequestIDKey, requestID)

		c.Writer.Header().Set("X-Request-ID", requestID)

		c.Next()
	}
}
