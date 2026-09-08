package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-flow-2-rupchella-a6429326/internal/config"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-flow-2-rupchella-a6429326/internal/httpserver"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-flow-2-rupchella-a6429326/internal/postgres"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := run(logger); err != nil {
		logger.Error("kitchen API stopped with error", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg := config.LoadKitchen()

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	connectCtx, cancel := context.WithTimeout(
		ctx,
		cfg.DatabaseConnectTimeout,
	)

	pool, err := postgres.NewPool(connectCtx, cfg.DatabaseURL)
	cancel()

	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer pool.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", httpserver.Liveness)
	mux.HandleFunc(
		"GET /readyz",
		httpserver.Readiness(pool, cfg.ReadinessTimeout),
	)

	server := httpserver.New(cfg.HTTPAddress, mux)

	logger.Info(
		"starting kitchen API",
		"address",
		cfg.HTTPAddress,
	)

	if err := httpserver.Run(
		ctx,
		server,
		cfg.ShutdownTimeout,
	); err != nil {
		return fmt.Errorf("run HTTP server: %w", err)
	}

	logger.Info("kitchen API stopped")

	return nil
}
