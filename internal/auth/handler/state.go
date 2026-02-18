package handler

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	stateCookieName = "__oauth_state"
	stateTTL        = 5 * time.Minute
)

func generateState(c *gin.Context) string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)

	state := base64.RawURLEncoding.EncodeToString(b)

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     stateCookieName,
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(stateTTL.Seconds()),
	})

	return state
}

func validateState(c *gin.Context) bool {
	stateQuery := c.Query("state")
	if stateQuery == "" {
		return false
	}

	cookie, err := c.Request.Cookie(stateCookieName)
	if err != nil {
		return false
	}

	return cookie.Value == stateQuery
}
