package config

import (
	"strconv"
	"time"
)

const (
	defaultRestaurantHTTPAddress = ":8081"
	defaultKitchenBaseURL        = "http://localhost:8080"
	defaultRestaurantAPIKey      = "demo-restaurant-key"
	defaultPollInterval          = 5 * time.Second

	defaultMaxOrderKopecks = 500_000
)

type Restaurant struct {
	HTTPAddress     string
	KitchenBaseURL  string
	APIKey          string
	PollInterval    time.Duration
	MaxOrderKopecks int64
	RequestTimeout  time.Duration
	ShutdownTimeout time.Duration
}

func LoadRestaurant() Restaurant {
	return Restaurant{
		HTTPAddress:     envOrDefault("RESTAURANT_HTTP_ADDR", defaultRestaurantHTTPAddress),
		KitchenBaseURL:  envOrDefault("KITCHEN_BASE_URL", defaultKitchenBaseURL),
		APIKey:          envOrDefault("RESTAURANT_API_KEY", defaultRestaurantAPIKey),
		PollInterval:    durationOrDefault("RESTAURANT_POLL_INTERVAL", defaultPollInterval),
		MaxOrderKopecks: int64OrDefault("RESTAURANT_MAX_ORDER_KOPECKS", defaultMaxOrderKopecks),
		RequestTimeout:  5 * time.Second,
		ShutdownTimeout: 10 * time.Second,
	}
}

func durationOrDefault(key string, fallback time.Duration) time.Duration {
	value, err := time.ParseDuration(envOrDefault(key, ""))
	if err != nil || value <= 0 {
		return fallback
	}

	return value
}

func int64OrDefault(key string, fallback int64) int64 {
	value, err := strconv.ParseInt(envOrDefault(key, ""), 10, 64)
	if err != nil || value <= 0 {
		return fallback
	}

	return value
}
