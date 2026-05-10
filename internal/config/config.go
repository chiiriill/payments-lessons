package config

import "os"

type Config struct {
	HTTPAddr    string
	DatabaseURL string
}

func Load() Config {
	cfg := Config{
		HTTPAddr:    ":8081",
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}
	if v := os.Getenv("HTTP_ADDR"); v != "" {
		cfg.HTTPAddr = v
	}
	return cfg
}
