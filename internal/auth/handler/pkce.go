package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	pkceCookieName = "__oauth_pkce"
	pkceTTL        = 5 * time.Minute
)

func generatePKCE(c *gin.Context) (verifier string, challenge string) {
	b := make([]byte, 32)
	_, _ = rand.Read(b)

	verifier = base64.RawURLEncoding.EncodeToString(b)

	hash := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(hash[:])

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     pkceCookieName,
		Value:    verifier,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(pkceTTL.Seconds()),
	})

	return verifier, challenge
}

func getPKCEVerifier(c *gin.Context) string {
	cookie, err := c.Request.Cookie(pkceCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}
