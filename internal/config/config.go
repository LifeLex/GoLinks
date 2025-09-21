package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	Port         int    `json:"port"`
	DatabasePath string `json:"database_path"`
	BaseURL      string `json:"base_url"`
	Environment  string `json:"environment"`
}

// Load loads configuration from environment variables and .env file
func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if file doesn't exist)
	_ = godotenv.Load()

	cfg := &Config{
		Port:         getEnvAsInt("PORT", 8080),
		DatabasePath: getEnv("DATABASE_PATH", "golinks.db"),
		BaseURL:      getEnv("BASE_URL", "http://localhost:8080"),
		Environment:  getEnv("ENVIRONMENT", "development"),
	}

	return cfg, nil
}

// getEnv gets an environment variable with a fallback value
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// getEnvAsInt gets an environment variable as integer with a fallback value
func getEnvAsInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return fallback
}
