package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/talense-tasks/backend-trainee-assignment-autumn-2026-flow-2-rupchella-a6429326/internal/httpserver"
)

const (
	defaultHTTPAddress = ":8081"
	shutdownTimeout    = 10 * time.Second
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	address := os.Getenv("RESTAURANT_HTTP_ADDR")
	if address == "" {
		address = defaultHTTPAddress
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthHandler)

	server := httpserver.New(address, mux)

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	logger.Info("starting restaurant service", "address", address)

	if err := httpserver.Run(ctx, server, shutdownTimeout); err != nil {
		logger.Error("restaurant service stopped with error", "error", err)
		os.Exit(1)
	}

	logger.Info("restaurant service stopped")
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
