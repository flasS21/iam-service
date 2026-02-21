package app

import (
	"context"
	"net/http"

	"iam-service/internal/config"
)

/*
New initializes the application with HTTP server and dependencies.
It sets up the router and any required resources through setupHTTP,
then configures an HTTP server listening on the port specified in config.
Returns the initialized App or an error if setup fails.
*/

type App struct {
	httpServer *http.Server
	cleanup    func() error
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	router, cleanup, err := setupHTTP(ctx, cfg)
	if err != nil {
		return nil, err
	}

	server := &http.Server{
		Addr:    ":" + cfg.AppPort,
		Handler: router,
	}

	return &App{
		httpServer: server,
		cleanup:    cleanup,
	}, nil
}

func (a *App) Run() error {
	return a.httpServer.ListenAndServe()
}

func (a *App) Shutdown(ctx context.Context) error {
	if err := a.httpServer.Shutdown(ctx); err != nil {
		return err
	}
	if a.cleanup != nil {
		return a.cleanup()
	}
	return nil
}
