package provider

import (
	"context"

	"iam-service/internal/auth"
)

type OAuthProvider interface {
	Name() string
	AuthCodeURL(state string, codeChallenge string) string
	ExchangeCode(
		ctx context.Context,
		code string,
		codeVerifier string,
	) (*auth.Identity, error)
}
