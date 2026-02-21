package app

import (
	"context"

	"iam-service/internal/auth/handler"
	"iam-service/internal/auth/provider"
	"iam-service/internal/auth/provider/keycloak"

	"iam-service/internal/auth/resolver"
	"iam-service/internal/config"
	"iam-service/internal/middleware"
	"iam-service/internal/session"

	"github.com/gin-gonic/gin"
)

/*
setupHTTP configures the HTTP router with all dependencies, middleware, and routes.
It delegates infrastructure initialization (database, Redis, Keycloak) to setupInfra,
then creates session store and identity resolver. Registers public routes, health check,
and protected API/web routes with authentication middleware. Returns configured Gin engine,
cleanup function for database closure, and any initialization error.
*/

func setupHTTP(ctx context.Context, cfg config.Config) (*gin.Engine, func() error, error) {

	infra, err := setupInfra(ctx, cfg)
	if err != nil {
		return nil, nil, err
	}

	// ----------------------------
	// Dependencies
	// ----------------------------

	sessionStore := session.NewRedisStore(infra.Redis.Client)
	identityResolver := resolver.NewDBResolver(infra.DB)

	keycloakProvider, err := keycloak.New(
		ctx,
		cfg.KeycloakIssuer,
		cfg.KeycloakClientID,
		cfg.KeycloakClientSecret,
		cfg.KeycloakRedirectURL,
		cfg.KeycloakPublicBaseURL,
	)
	if err != nil {
		return nil, nil, err
	}

	registry := provider.NewRegistry(
		keycloakProvider,
	)

	authHandler := handler.NewHandler(
		registry,
		sessionStore,
		identityResolver,
	)

	authMiddleware := middleware.NewAuthMiddleware(sessionStore)

	// ----------------------------
	// Router
	// ----------------------------

	router := gin.New()
	router.Use(gin.Recovery())

	// ----------------------------
	// Public Routes
	// ----------------------------

	authHandler.RegisterRoutes(router)

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	router.Static("/mvp-test", "./mvp-test")

	router.GET("/", func(c *gin.Context) {
		c.File("./mvp-test/index.html")
	})

	// ----------------------------
	// Protected API Routes
	// ----------------------------

	api := router.Group("/api")
	api.Use(middleware.GinRequireAuth(authMiddleware))

	api.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	api.GET("/me", func(c *gin.Context) {
		userID := c.GetString("userID")
		c.JSON(200, gin.H{
			"user_id": userID,
		})
	})

	// ----------------------------
	// Protected Web Routes
	// ----------------------------

	web := router.Group("/")
	web.Use(middleware.GinRequireAuth(authMiddleware))

	web.GET("/dashboard", func(c *gin.Context) {
		c.File("./mvp-test/dashboard.html")
	})

	// -----------------------------------
	// Demo Frontend (web-test)
	// -----------------------------------
	// router.Static("/web-test", "./web-test")

	// router.GET("/", func(c *gin.Context) {
	// 	c.File("./web-test/index.html")
	// })

	// router.GET("/dashboard.html", func(c *gin.Context) {
	// 	c.File("./web-test/dashboard.html")
	// })

	// ----------------------------
	// Cleanup
	// ----------------------------

	return router, func() error {
		return infra.DB.Close()
	}, nil
}
