package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-flow-2-rupchella-a6429326/internal/config"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-flow-2-rupchella-a6429326/internal/httpserver"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-flow-2-rupchella-a6429326/internal/restaurant"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := run(logger); err != nil {
		logger.Error("restaurant service stopped with error", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg := config.LoadRestaurant()

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	client := restaurant.NewClient(
		cfg.KitchenBaseURL,
		cfg.APIKey,
		&http.Client{
			Timeout: cfg.RequestTimeout,
		},
	)

	menu := restaurant.DemoMenu()

	worker := restaurant.NewWorker(
		client,
		logger,
		menu,
		cfg.PollInterval,
		cfg.MaxOrderKopecks,
	)

	var waitGroup sync.WaitGroup

	waitGroup.Add(1)

	go func() {
		defer waitGroup.Done()

		worker.Run(ctx)
	}()

	handler := restaurant.NewHandler(client, menu, logger)
	server := httpserver.New(
		cfg.HTTPAddress,
		httpserver.LogRequests(logger, handler.Routes()),
	)

	logger.Info(
		"starting restaurant service",
		"address", cfg.HTTPAddress,
		"kitchen", cfg.KitchenBaseURL,
		"poll_interval", cfg.PollInterval.String(),
	)

	err := httpserver.Run(ctx, server, cfg.ShutdownTimeout)

	waitGroup.Wait()

	if err != nil {
		return fmt.Errorf("run HTTP server: %w", err)
	}

	logger.Info("restaurant service stopped")

	return nil
}
