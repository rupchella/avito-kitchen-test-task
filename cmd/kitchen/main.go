package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-flow-2-rupchella-a6429326/internal/config"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-flow-2-rupchella-a6429326/internal/httpserver"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-flow-2-rupchella-a6429326/internal/postgres"
	postgresrepository "github.com/talense-tasks/backend-trainee-assignment-autumn-2026-flow-2-rupchella-a6429326/internal/repository/postgres"
	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-flow-2-rupchella-a6429326/internal/service"
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

	restaurantRepository := postgresrepository.NewRestaurantRepository(pool)
	menuRepository := postgresrepository.NewMenuRepository(pool)
	orderRepository := postgresrepository.NewOrderRepository(pool)
	integrationRepository := postgresrepository.NewIntegrationRepository(pool)

	restaurantService := service.NewRestaurantService(
		restaurantRepository,
		menuRepository,
	)
	orderService := service.NewOrderService(
		orderRepository,
		menuRepository,
		restaurantRepository,
	)
	partnerService := service.NewPartnerService(
		integrationRepository,
		menuRepository,
		orderRepository,
	)

	openAPISpec, err := os.ReadFile(cfg.OpenAPIPath)
	if err != nil {
		logger.Warn(
			"API documentation is disabled",
			"path", cfg.OpenAPIPath,
			"error", err,
		)
	}

	router := httpserver.NewRouter(httpserver.RouterDependencies{
		Pinger:           pool,
		ReadinessTimeout: cfg.ReadinessTimeout,
		OpenAPISpec:      openAPISpec,
		Restaurants: httpserver.NewRestaurantHandler(
			restaurantService,
			logger,
		),
		Orders:  httpserver.NewOrderHandler(orderService, logger),
		Partner: httpserver.NewPartnerHandler(partnerService, logger),
	})

	server := httpserver.New(
		cfg.HTTPAddress,
		httpserver.LogRequests(logger, router),
	)

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
