// Package config loads runtime configuration from environment variables, with
// optional .env support. The Config struct is passed around as a value; mutate
// only at startup.
package config

import (
	"os"
	"strconv"

	"golinks/internal/platform/logger"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application.
type Config struct {
	Port           int           `json:"port"`
	DatabaseDriver string        `json:"database_driver"` // "sqlite" or "postgres"
	DatabaseURL    string        `json:"database_url"`
	BaseURL        string        `json:"base_url"`
	Environment    string        `json:"environment"`
	Logging        logger.Config `json:"logging"`

	// DatabasePath is deprecated — kept so existing deployments setting
	// only DATABASE_PATH continue to work. If DATABASE_URL is empty and
	// the driver is sqlite, DatabasePath is used to synthesise the URL.
	DatabasePath string `json:"database_path"`
}

// Load reads configuration from environment variables (and .env if present).
// Missing or invalid values fall back to sensible development defaults.
func Load() (*Config, error) {
	_ = godotenv.Load() // Ignore error if .env doesn't exist.

	cfg := &Config{
		Port:           getEnvAsInt("PORT", 8080),
		DatabaseDriver: getEnv("DATABASE_DRIVER", "sqlite"),
		DatabaseURL:    getEnv("DATABASE_URL", ""),
		DatabasePath:   getEnv("DATABASE_PATH", "golinks.db"),
		BaseURL:        getEnv("BASE_URL", "http://localhost:8080"),
		Environment:    getEnv("ENVIRONMENT", "development"),
		Logging: logger.Config{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: "text",
		},
	}

	// Synthesise a SQLite URL from the deprecated DATABASE_PATH when no
	// DATABASE_URL was provided. Postgres requires an explicit URL — there's
	// no sensible default we can fabricate.
	if cfg.DatabaseURL == "" && cfg.DatabaseDriver == "sqlite" {
		cfg.DatabaseURL = cfg.DatabasePath
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return fallback
}
