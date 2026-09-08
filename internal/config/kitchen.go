package config

import (
	"os"
	"time"
)

const (
	defaultKitchenHTTPAddress = ":8080"
	defaultDatabaseURL        = "postgres://kitchen:kitchen@localhost:5432/kitchen?sslmode=disable"
)

type Kitchen struct {
	HTTPAddress            string
	DatabaseURL            string
	DatabaseConnectTimeout time.Duration
	ReadinessTimeout       time.Duration
	ShutdownTimeout        time.Duration
}

func LoadKitchen() Kitchen {
	return Kitchen{
		HTTPAddress:            envOrDefault("KITCHEN_HTTP_ADDR", defaultKitchenHTTPAddress),
		DatabaseURL:            envOrDefault("DATABASE_URL", defaultDatabaseURL),
		DatabaseConnectTimeout: 5 * time.Second,
		ReadinessTimeout:       time.Second,
		ShutdownTimeout:        10 * time.Second,
	}
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
