package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"iam-service/internal/app"
	"iam-service/internal/config"
	"iam-service/internal/logger"
)

func main() {
	logger.Init()
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	application, err := app.New(ctx, cfg)
	if err != nil {
		logger.Fatal("failed to initialize app", map[string]any{
			"error": err.Error(),
		})
	}

	go func() {
		if err := application.Run(); err != nil {
			logger.Fatal("http server failed", map[string]any{
				"error": err.Error(),
			})
		}
	}()

	logger.Info("auth-service started", map[string]any{
		"port": cfg.AppPort,
	})

	<-ctx.Done() // wait for Ctrl+C

	logger.Info("shutdown signal received", nil)

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := application.Shutdown(shutdownCtx); err != nil {
		logger.Fatal("graceful shutdown failed", map[string]any{
			"error": err.Error(),
		})
	}

	logger.Info("auth-service stopped cleanly", nil)
}
