// Package logger is a thin wrapper around log/slog that exposes Printf-style
// formatting helpers. It owns a process-wide default logger that is initialised
// once at startup via Initialize, then read via Default.
package logger

import (
	"fmt"
	"log/slog"
	"os"
)

// Logger wraps *slog.Logger so callers can pass a single concrete type around.
type Logger struct {
	*slog.Logger
}

// Config controls the logger's level. Format is reserved for future use.
type Config struct {
	Level  string `json:"level"`  // debug, info, warn, error
	Format string `json:"format"` // not used in simple logger
}

// New constructs a logger from the given Config. Output goes to stdout as text;
// switch to slog.NewJSONHandler when an observability backend lands.
func New(cfg Config) *Logger {
	var level slog.Level
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level, AddSource: true}
	handler := slog.NewTextHandler(os.Stdout, opts)
	return &Logger{Logger: slog.New(handler)}
}

// Debug logs at debug level.
func (l *Logger) Debug(msg string, args ...interface{}) {
	l.Logger.Debug(formatMessage(msg, args...))
}

// Info logs at info level.
func (l *Logger) Info(msg string, args ...interface{}) {
	l.Logger.Info(formatMessage(msg, args...))
}

// Warn logs at warn level.
func (l *Logger) Warn(msg string, args ...interface{}) {
	l.Logger.Warn(formatMessage(msg, args...))
}

// Error logs at error level.
func (l *Logger) Error(msg string, args ...interface{}) {
	l.Logger.Error(formatMessage(msg, args...))
}

func formatMessage(msg string, args ...interface{}) string {
	if len(args) == 0 {
		return msg
	}
	return fmt.Sprintf(msg, args...)
}

var defaultLogger *Logger

// Initialize sets the package-level default logger. Call once at startup.
func Initialize(cfg Config) {
	defaultLogger = New(cfg)
}

// Default returns the package-level logger, creating a fallback if Initialize
// was never called (so test code doesn't need to).
func Default() *Logger {
	if defaultLogger == nil {
		defaultLogger = New(Config{Level: "info", Format: "text"})
	}
	return defaultLogger
}
