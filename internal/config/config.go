package config

import (
	"os"
	"strconv"

	"golinks/internal/logger"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	Port         int           `json:"port"`
	DatabasePath string        `json:"database_path"`
	BaseURL      string        `json:"base_url"`
	Environment  string        `json:"environment"`
	Logging      logger.Config `json:"logging"`
	Auth         AuthConfig    `json:"auth"`
}

// AuthConfig holds authentication-related settings.
type AuthConfig struct {
	// SessionTTLHours is how long a login session stays valid. Default 720 (30d).
	SessionTTLHours int `json:"session_ttl_hours"`
	// CookieSecure marks the session cookie Secure (HTTPS-only). Defaults to true
	// in production and false in development so http://localhost works.
	CookieSecure bool `json:"cookie_secure"`
	// BcryptCost is the bcrypt work factor for password hashing. Default 12.
	BcryptCost int `json:"bcrypt_cost"`
	// MinPasswordLen is the minimum accepted password length. Default 8.
	MinPasswordLen int `json:"min_password_len"`
}

// Load loads configuration from environment variables and .env file
func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if file doesn't exist)
	_ = godotenv.Load()

	environment := getEnv("ENVIRONMENT", "development")

	cfg := &Config{
		Port:         getEnvAsInt("PORT", 8080),
		DatabasePath: getEnv("DATABASE_PATH", "golinks.db"),
		BaseURL:      getEnv("BASE_URL", "http://localhost:8080"),
		Environment:  environment,
		Logging: logger.Config{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: "text", // Not used in simple logger
		},
		Auth: AuthConfig{
			SessionTTLHours: getEnvAsInt("SESSION_TTL_HOURS", 720),
			CookieSecure:    getEnvAsBool("COOKIE_SECURE", environment == "production"),
			BcryptCost:      getEnvAsInt("BCRYPT_COST", 12),
			MinPasswordLen:  getEnvAsInt("MIN_PASSWORD_LEN", 8),
		},
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

// getEnvAsBool gets an environment variable as a bool with a fallback value.
func getEnvAsBool(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return fallback
}
