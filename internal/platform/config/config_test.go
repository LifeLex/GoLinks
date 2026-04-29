package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	keys := []string{"PORT", "DATABASE_DRIVER", "DATABASE_URL", "DATABASE_PATH", "BASE_URL", "ENVIRONMENT"}
	originalEnv := snapshotEnv(keys)
	defer restoreEnv(originalEnv)

	tests := []struct {
		name              string
		envVars           map[string]string
		wantDriver        string
		wantURL           string
		wantPath          string
		wantPort          int
		wantBaseURL       string
		wantEnvironment   string
	}{
		{
			name:            "default values",
			envVars:         map[string]string{},
			wantDriver:      "sqlite",
			wantURL:         "golinks.db", // synthesised from DATABASE_PATH default
			wantPath:        "golinks.db",
			wantPort:        8080,
			wantBaseURL:     "http://localhost:8080",
			wantEnvironment: "development",
		},
		{
			name: "explicit DATABASE_URL wins",
			envVars: map[string]string{
				"DATABASE_DRIVER": "sqlite",
				"DATABASE_URL":    "file:custom.db?_pragma=journal_mode(WAL)",
				"DATABASE_PATH":   "should-be-ignored.db",
			},
			wantDriver:      "sqlite",
			wantURL:         "file:custom.db?_pragma=journal_mode(WAL)",
			wantPath:        "should-be-ignored.db",
			wantPort:        8080,
			wantBaseURL:     "http://localhost:8080",
			wantEnvironment: "development",
		},
		{
			name: "DATABASE_PATH synthesises URL when DATABASE_URL is empty",
			envVars: map[string]string{
				"DATABASE_PATH": "/var/lib/golinks/db.sqlite",
			},
			wantDriver:      "sqlite",
			wantURL:         "/var/lib/golinks/db.sqlite",
			wantPath:        "/var/lib/golinks/db.sqlite",
			wantPort:        8080,
			wantBaseURL:     "http://localhost:8080",
			wantEnvironment: "development",
		},
		{
			name: "postgres driver requires explicit URL",
			envVars: map[string]string{
				"DATABASE_DRIVER": "postgres",
				"DATABASE_URL":    "postgres://u:p@db:5432/golinks?sslmode=disable",
			},
			wantDriver:      "postgres",
			wantURL:         "postgres://u:p@db:5432/golinks?sslmode=disable",
			wantPath:        "golinks.db", // default; ignored when driver != sqlite
			wantPort:        8080,
			wantBaseURL:     "http://localhost:8080",
			wantEnvironment: "development",
		},
		{
			name:            "invalid port falls back to default",
			envVars:         map[string]string{"PORT": "invalid"},
			wantDriver:      "sqlite",
			wantURL:         "golinks.db",
			wantPath:        "golinks.db",
			wantPort:        8080,
			wantBaseURL:     "http://localhost:8080",
			wantEnvironment: "development",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, k := range keys {
				os.Unsetenv(k)
			}
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}

			checks := []struct {
				name string
				got  interface{}
				want interface{}
			}{
				{"DatabaseDriver", cfg.DatabaseDriver, tt.wantDriver},
				{"DatabaseURL", cfg.DatabaseURL, tt.wantURL},
				{"DatabasePath", cfg.DatabasePath, tt.wantPath},
				{"Port", cfg.Port, tt.wantPort},
				{"BaseURL", cfg.BaseURL, tt.wantBaseURL},
				{"Environment", cfg.Environment, tt.wantEnvironment},
			}
			for _, c := range checks {
				if c.got != c.want {
					t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
				}
			}
		})
	}
}

func TestGetEnv(t *testing.T) {
	defer os.Unsetenv("TEST_VAR")
	os.Setenv("TEST_VAR", "custom")
	if got := getEnv("TEST_VAR", "default"); got != "custom" {
		t.Errorf("getEnv() = %v, want custom", got)
	}
	os.Unsetenv("TEST_VAR")
	if got := getEnv("TEST_VAR", "default"); got != "default" {
		t.Errorf("getEnv() = %v, want default", got)
	}
}

func TestGetEnvAsInt(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		fallback int
		expected int
	}{
		{"valid integer", "9090", 8080, 9090},
		{"invalid integer", "invalid", 8080, 8080},
		{"empty value", "", 8080, 8080},
		{"negative integer", "-1", 8080, -1},
		{"zero", "0", 8080, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := "TEST_INT"
			defer os.Unsetenv(key)
			if tt.envValue != "" {
				os.Setenv(key, tt.envValue)
			}
			if got := getEnvAsInt(key, tt.fallback); got != tt.expected {
				t.Errorf("getEnvAsInt() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func snapshotEnv(keys []string) map[string]string {
	out := make(map[string]string, len(keys))
	for _, k := range keys {
		out[k] = os.Getenv(k)
	}
	return out
}

func restoreEnv(env map[string]string) {
	for k, v := range env {
		if v == "" {
			os.Unsetenv(k)
		} else {
			os.Setenv(k, v)
		}
	}
}
