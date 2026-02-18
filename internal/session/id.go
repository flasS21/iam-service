package session

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// GenerateID generates a cryptographically secure session ID.
// 32 bytes = 256 bits of entropy.
func GenerateID() (string, error) {

	const size = 32 // 256 bits

	b := make([]byte, size)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("session: failed to generate id: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(b), nil

}
