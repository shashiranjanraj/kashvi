package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shashiranjanraj/kashvi/config"
	"github.com/shashiranjanraj/kashvi/internal/kernel"
	"github.com/shashiranjanraj/kashvi/pkg/cache"
	"github.com/shashiranjanraj/kashvi/pkg/database"
	"github.com/shashiranjanraj/kashvi/pkg/logger"
	"github.com/shashiranjanraj/kashvi/pkg/queue"
	"github.com/shashiranjanraj/kashvi/pkg/storage"
)

// Start boots the server, runs until SIGINT/SIGTERM, then shuts down gracefully.
func Start() error {
	if err := config.Load(); err != nil {
		return fmt.Errorf("config: %w", err)
	}

	// Guard: refuse to start in production with the default JWT secret.
	if (config.AppEnv() == "production" || config.AppEnv() == "prod") &&
		config.JWTSecret() == "change-me-in-production" {
		return fmt.Errorf("refusing to start: JWT_SECRET must be changed in production")
	}

	if err := database.Connect(); err != nil {
		return fmt.Errorf("database: %w", err)
	}

	// Redis is non-fatal — app degrades gracefully without it.
	if err := cache.Connect(); err != nil {
		logger.Warn("cache: Redis unavailable, continuing without cache", "error", err)
	}

	// Wire DB into queue for persistent failed jobs.
	queue.UseDB(database.DB)

	storage.Connect()

	httpKernel := kernel.NewHTTPKernel()

	addr := ":" + config.AppPort()
	srv := &http.Server{
		Addr:         addr,
		Handler:      httpKernel.Handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Channel to receive OS signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		fmt.Printf("🚀 Kashvi running on %s  [env: %s]\n", addr, config.AppEnv())
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case sig := <-quit:
		fmt.Printf("\n⚡ Signal %s received — shutting down gracefully…\n", sig)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return srv.Shutdown(ctx)
}
