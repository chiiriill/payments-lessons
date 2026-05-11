package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	AppName            string
	HTTPAddr           string
	ShutdownTimeout    time.Duration
	DatabaseURL        string
	ProviderBaseURL    string
	ProviderTimeout    time.Duration
	ProviderRetryCount int
	WebhookSecret      string
}

func Load() Config {
	return Config{
		AppName:            env("APP_NAME", "payments-service"),
		HTTPAddr:           env("HTTP_ADDR", ":8081"),
		ShutdownTimeout:    durationEnv("SHUTDOWN_TIMEOUT", 10*time.Second),
		DatabaseURL:        env("DATABASE_URL", "postgres://postgres:postgres@localhost:5433/payments?sslmode=disable"),
		ProviderBaseURL:    env("PROVIDER_BASE_URL", "http://localhost:8082"),
		ProviderTimeout:    durationEnv("PROVIDER_TIMEOUT", 3*time.Second),
		ProviderRetryCount: intEnv("PROVIDER_RETRY_COUNT", 2),
		WebhookSecret:      env("WEBHOOK_SECRET", "local-secret"),
	}
}

func env(key string, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func intEnv(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	var n int
	if _, err := fmt.Sscanf(v, "%d", &n); err != nil {
		return fallback
	}
	return n
}
