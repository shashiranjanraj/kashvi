package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/shashiranjanraj/kashvi/config"
	"github.com/shashiranjanraj/kashvi/pkg/cache"
	"github.com/shashiranjanraj/kashvi/pkg/database"
	kashvigrpc "github.com/shashiranjanraj/kashvi/pkg/grpc"
	"github.com/shashiranjanraj/kashvi/pkg/logger"
	"github.com/shashiranjanraj/kashvi/pkg/queue"
	"github.com/shashiranjanraj/kashvi/pkg/storage"
)

// Start boots the HTTP + gRPC servers, runs until SIGINT/SIGTERM, then shuts
// down gracefully.
//
// handler is the application's root http.Handler (built by pkg/app.buildHandler).
// Passing nil uses a minimal default handler (useful for quick smoke tests).
func Start(handler http.Handler) error {
	if err := config.Load(); err != nil {
		return fmt.Errorf("config: %w", err)
	}

	// Log runtime concurrency level.
	procs := runtime.GOMAXPROCS(0)
	logger.Info("runtime", "GOMAXPROCS", procs, "NumCPU", runtime.NumCPU())

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

	// ── HTTP server ─────────────────────────────────────────────────────────

	if handler == nil {
		handler = http.NotFoundHandler()
	}

	addr := ":" + config.AppPort()
	srv := &http.Server{
		Addr:    addr,
		Handler: handler,
		// Tuned for high-throughput (100k req/min target).
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   10 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 2)

	go func() {
		fmt.Printf("🚀 Kashvi HTTP  on %s  [env: %s]  [workers: %d]\n",
			addr, config.AppEnv(), runtime.GOMAXPROCS(0))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// ── gRPC server ─────────────────────────────────────────────────────────

	grpcSrv, _, grpcErr := kashvigrpc.Start(config.GRPCPort())
	if grpcErr != nil {
		logger.Warn("grpc: server failed to start, HTTP-only mode", "error", grpcErr)
	} else {
		fmt.Printf("🔌 Kashvi gRPC  on :%s\n", config.GRPCPort())
	}

	// ── Wait for shutdown signal ─────────────────────────────────────────────

	select {
	case err := <-errCh:
		return err
	case sig := <-quit:
		fmt.Printf("\n⚡ Signal %s received — shutting down gracefully…\n", sig)
	}

	// Graceful HTTP shutdown (10 s deadline).
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	httpErr := srv.Shutdown(ctx)

	// Graceful gRPC shutdown.
	kashvigrpc.Stop(grpcSrv)

	// Flush MongoDB log handler.
	logger.CloseMongoHandler()

	return httpErr
}
