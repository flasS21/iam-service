package utils

import (
	"crypto/rand"
	"encoding/base64"
)

func RandomString(bytes int) string {
	b := make([]byte, bytes)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
