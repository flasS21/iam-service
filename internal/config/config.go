package config

import (
	"os"
)

type Config struct {
	AppPort string

	// ----------------------------
	// Google OAuth
	// ----------------------------
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string

	// ============================
	// NEW: Keycloak OAuth
	// ============================
	KeycloakIssuer        string
	KeycloakClientID      string
	KeycloakRedirectURL   string
	KeycloakPublicBaseURL string

	// ----------------------------
	// Redis
	// ----------------------------
	RedisAddr     string
	RedisPassword string

	// ----------------------------
	// Database
	// ----------------------------
	DatabaseDSN string
}

func Load() Config {

	cfg := Config{

		AppPort: os.Getenv("APP_PORT"),

		// ----------------------------
		// Google OAuth
		// ----------------------------
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),

		// ============================
		// NEW: Keycloak OAuth
		// ============================
		KeycloakIssuer:        os.Getenv("KEYCLOAK_ISSUER"),
		KeycloakClientID:      os.Getenv("KEYCLOAK_CLIENT_ID"),
		KeycloakRedirectURL:   os.Getenv("KEYCLOAK_REDIRECT_URL"),
		KeycloakPublicBaseURL: os.Getenv("KEYCLOAK_PUBLIC_BASE_URL"),

		// ----------------------------
		// Redis
		// ----------------------------
		RedisAddr:     os.Getenv("REDIS_ADDR"),
		RedisPassword: os.Getenv("REDIS_PASSWORD"),

		// ----------------------------
		// Database
		// ----------------------------
		DatabaseDSN: os.Getenv("DATABASE_DSN"),
	}

	return cfg
}
