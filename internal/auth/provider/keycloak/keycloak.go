package keycloak

import (
	"context"
	"errors"
	"fmt"

	"iam-service/internal/auth"
	"iam-service/internal/logger"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

const providerName = "keycloak"

type Provider struct {
	oauthConfig *oauth2.Config
	verifier    *oidc.IDTokenVerifier
}

func New(
	ctx context.Context,
	issuer string,
	clientID string,
	clientSecret string,
	redirectURL string,
	publicBaseURL string,
) (*Provider, error) {

	if issuer == "" || clientID == "" || clientSecret == "" || redirectURL == "" || publicBaseURL == "" {
		return nil, errors.New("keycloak oauth config missing required fields")
	}

	oidcProvider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("failed to init keycloak oidc provider: %w", err)
	}

	verifier := oidcProvider.Verifier(&oidc.Config{
		ClientID: clientID,
	})

	// Endpoint: oidcProvider.Endpoint(),
	// ep := oidcProvider.Endpoint()
	// ep.AuthURL = publicBaseURL + "/protocol/openid-connect/auth"

	oauthCfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Endpoint:     oidcProvider.Endpoint(),
		Scopes: []string{
			oidc.ScopeOpenID,
			"email",
			"profile",
		},
	}

	return &Provider{
		oauthConfig: oauthCfg,
		verifier:    verifier,
	}, nil
}

func (p *Provider) Name() string {
	return providerName
}

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
		// return nil, fmt.Errorf("keycloak token exchange failed: %w", err)
		logger.Error("keycloak token exchange failed", map[string]any{
			"error": err.Error(),
		})
		return nil, err
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return nil, errors.New("keycloak did not return id_token")
	}

	logger.Info("keycloak id_token received", map[string]any{
		"present": rawIDToken != "",
	})

	idToken, err := p.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		// return nil, fmt.Errorf("keycloak id_token verification failed: %w", err)
		logger.Error("keycloak id_token verification failed", map[string]any{
			"error": err.Error(),
		})
		return nil, err
	}

	var claims struct {
		Subject           string `json:"sub"`
		Email             string `json:"email"`
		EmailVerified     bool   `json:"email_verified"`
		PreferredUsername string `json:"preferred_username"`
	}

	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("keycloak id_token claims parse failed: %w", err)
	}

	if claims.Subject == "" || claims.Email == "" {
		return nil, errors.New("keycloak id_token missing required claims")
	}

	logger.Info("keycloak claims", map[string]any{
		"sub":            claims.Subject,
		"email":          claims.Email,
		"email_verified": claims.EmailVerified,
	})

	logger.Info("keycloak oidc verified", map[string]any{
		"issuer":             idToken.Issuer,
		"subject_present":    claims.Subject != "",
		"email_present":      claims.Email != "",
		"email_verified":     claims.EmailVerified,
		"preferred_username": claims.PreferredUsername,
		"audience":           idToken.Audience,
		"expiry_unix":        idToken.Expiry.Unix(),
	})

	return &auth.Identity{
		KeycloakSub:   claims.Subject,
		Email:         claims.Email,
		EmailVerified: claims.EmailVerified,
	}, nil
}
