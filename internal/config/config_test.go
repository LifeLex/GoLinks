package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Save original environment
	originalEnv := map[string]string{
		"PORT":          os.Getenv("PORT"),
		"DATABASE_PATH": os.Getenv("DATABASE_PATH"),
		"BASE_URL":      os.Getenv("BASE_URL"),
		"ENVIRONMENT":   os.Getenv("ENVIRONMENT"),
	}

	// Clean up after test
	defer func() {
		for key, value := range originalEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	tests := []struct {
		name     string
		envVars  map[string]string
		expected *Config
	}{
		{
			name:    "default values",
			envVars: map[string]string{},
			expected: &Config{
				Port:         8080,
				DatabasePath: "golinks.db",
				BaseURL:      "http://localhost:8080",
				Environment:  "development",
			},
		},
		{
			name: "custom values from environment",
			envVars: map[string]string{
				"PORT":          "9090",
				"DATABASE_PATH": "/custom/path/db.sqlite",
				"BASE_URL":      "https://golinks.company.com",
				"ENVIRONMENT":   "production",
			},
			expected: &Config{
				Port:         9090,
				DatabasePath: "/custom/path/db.sqlite",
				BaseURL:      "https://golinks.company.com",
				Environment:  "production",
			},
		},
		{
			name: "partial custom values",
			envVars: map[string]string{
				"PORT":     "3000",
				"BASE_URL": "https://custom.example.com",
			},
			expected: &Config{
				Port:         3000,
				DatabasePath: "golinks.db", // default
				BaseURL:      "https://custom.example.com",
				Environment:  "development", // default
			},
		},
		{
			name: "invalid port falls back to default",
			envVars: map[string]string{
				"PORT": "invalid",
			},
			expected: &Config{
				Port:         8080, // default due to invalid value
				DatabasePath: "golinks.db",
				BaseURL:      "http://localhost:8080",
				Environment:  "development",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			for key := range originalEnv {
				os.Unsetenv(key)
			}

			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			cfg, err := Load()
			if err != nil {
				t.Errorf("Load() error = %v", err)
				return
			}

			if cfg.Port != tt.expected.Port {
				t.Errorf("Load() Port = %v, want %v", cfg.Port, tt.expected.Port)
			}

			if cfg.DatabasePath != tt.expected.DatabasePath {
				t.Errorf("Load() DatabasePath = %v, want %v", cfg.DatabasePath, tt.expected.DatabasePath)
			}

			if cfg.BaseURL != tt.expected.BaseURL {
				t.Errorf("Load() BaseURL = %v, want %v", cfg.BaseURL, tt.expected.BaseURL)
			}

			if cfg.Environment != tt.expected.Environment {
				t.Errorf("Load() Environment = %v, want %v", cfg.Environment, tt.expected.Environment)
			}
		})
	}
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		fallback string
		envValue string
		expected string
	}{
		{
			name:     "environment variable exists",
			key:      "TEST_VAR",
			fallback: "default",
			envValue: "custom",
			expected: "custom",
		},
		{
			name:     "environment variable does not exist",
			key:      "NONEXISTENT_VAR",
			fallback: "default",
			envValue: "",
			expected: "default",
		},
		{
			name:     "empty environment variable",
			key:      "EMPTY_VAR",
			fallback: "default",
			envValue: "",
			expected: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up
			defer os.Unsetenv(tt.key)

			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
			}

			result := getEnv(tt.key, tt.fallback)
			if result != tt.expected {
				t.Errorf("getEnv() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetEnvAsInt(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		fallback int
		envValue string
		expected int
	}{
		{
			name:     "valid integer",
			key:      "TEST_INT",
			fallback: 8080,
			envValue: "9090",
			expected: 9090,
		},
		{
			name:     "invalid integer",
			key:      "TEST_INT",
			fallback: 8080,
			envValue: "invalid",
			expected: 8080,
		},
		{
			name:     "empty value",
			key:      "TEST_INT",
			fallback: 8080,
			envValue: "",
			expected: 8080,
		},
		{
			name:     "negative integer",
			key:      "TEST_INT",
			fallback: 8080,
			envValue: "-1",
			expected: -1,
		},
		{
			name:     "zero",
			key:      "TEST_INT",
			fallback: 8080,
			envValue: "0",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up
			defer os.Unsetenv(tt.key)

			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
			}

			result := getEnvAsInt(tt.key, tt.fallback)
			if result != tt.expected {
				t.Errorf("getEnvAsInt() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConfigValidation(t *testing.T) {
	// Test that Load() always returns a valid config
	cfg, err := Load()
	if err != nil {
		t.Errorf("Load() should not return error, got %v", err)
	}

	if cfg == nil {
		t.Error("Load() should not return nil config")
		return
	}

	// Test that all fields have reasonable values
	if cfg.Port <= 0 || cfg.Port > 65535 {
		t.Errorf("Port should be valid, got %d", cfg.Port)
	}

	if cfg.DatabasePath == "" {
		t.Error("DatabasePath should not be empty")
	}

	if cfg.BaseURL == "" {
		t.Error("BaseURL should not be empty")
	}

	if cfg.Environment == "" {
		t.Error("Environment should not be empty")
	}
}
