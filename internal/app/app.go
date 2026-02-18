package app

import (
	"context"
	"net/http"

	"iam-service/internal/config"
)

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
