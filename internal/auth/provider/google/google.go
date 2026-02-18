package google

import (
	"context"
	"errors"
	"fmt"

	"iam-service/internal/auth"
	"iam-service/internal/logger"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

const providerName = "google"

type Provider struct {
	oauthConfig *oauth2.Config
	verifier    *oidc.IDTokenVerifier
}

func New(
	ctx context.Context,
	clientID string,
	clientSecret string,
	redirectURL string,
) (*Provider, error) {

	if clientID == "" || clientSecret == "" || redirectURL == "" {
		return nil, errors.New("google oauth config missing required fields")
	}

	oidcProvider, err := oidc.NewProvider(ctx, "https://accounts.google.com")
	if err != nil {
		return nil, fmt.Errorf("failed to init google oidc provider: %w", err)
	}

	verifier := oidcProvider.Verifier(&oidc.Config{
		ClientID: clientID,
	})

	oauthCfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Endpoint:     oidcProvider.Endpoint(),
		Scopes: []string{
			oidc.ScopeOpenID,
			"profile",
			"email",
		},
	}

	return &Provider{
		oauthConfig: oauthCfg,
		verifier:    verifier,
	}, nil
}

// Name returns the provider identifier used by the registry.
func (p *Provider) Name() string {
	return providerName
}

// AuthCodeURL builds the OAuth authorization URL with PKCE parameters.
func (p *Provider) AuthCodeURL(state string, codeChallenge string) string {
	return p.oauthConfig.AuthCodeURL(
		state,
		oauth2.AccessTypeOnline,
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
}

func (p *Provider) ExchangeCode(
	ctx context.Context,
	code string,
	codeVerifier string,
) (*auth.Identity, error) {

	token, err := p.oauthConfig.Exchange(
		ctx,
		code,
		oauth2.SetAuthURLParam("code_verifier", codeVerifier),
	)
	if err != nil {
		return nil, fmt.Errorf("google token exchange failed: %w", err)
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return nil, errors.New("google did not return id_token")
	}

	idToken, err := p.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("google id_token verification failed: %w", err)
	}

	var claims struct {
		Subject       string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
	}

	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("google id_token claims parse failed: %w", err)
	}

	if claims.Subject == "" || claims.Email == "" {
		return nil, errors.New("google id_token missing required claims")
	}

	logger.Info("google oidc verified", map[string]any{
		"issuer":          idToken.Issuer,
		"subject_present": claims.Subject != "",
		"email_present":   claims.Email != "",
		"email_verified":  claims.EmailVerified,
		"audience":        idToken.Audience,
		"expiry_unix":     idToken.Expiry.Unix(),
	})

	return &auth.Identity{
		Provider:       providerName,
		ProviderUserID: claims.Subject,
		Email:          claims.Email,
		EmailVerified:  claims.EmailVerified,
	}, nil
}
