package logger

import (
	"fmt"
	"log/slog"
	"os"
)

// Logger is a simple wrapper around slog
type Logger struct {
	*slog.Logger
}

// Config holds logger configuration
type Config struct {
	Level  string `json:"level"`  // debug, info, warn, error
	Format string `json:"format"` // not used in simple logger
}

// New creates a new simple slog logger
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

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	}

	handler := slog.NewTextHandler(os.Stdout, opts)
	logger := slog.New(handler)

	return &Logger{
		Logger: logger,
	}
}

// Debug logs debug messages
func (l *Logger) Debug(msg string, args ...interface{}) {
	l.Logger.Debug(formatMessage(msg, args...))
}

// Info logs info messages
func (l *Logger) Info(msg string, args ...interface{}) {
	l.Logger.Info(formatMessage(msg, args...))
}

// Warn logs warning messages
func (l *Logger) Warn(msg string, args ...interface{}) {
	l.Logger.Warn(formatMessage(msg, args...))
}

// Error logs error messages
func (l *Logger) Error(msg string, args ...interface{}) {
	l.Logger.Error(formatMessage(msg, args...))
}

// formatMessage formats the message with args using Printf-style formatting
func formatMessage(msg string, args ...interface{}) string {
	if len(args) == 0 {
		return msg
	}
	// Use Go's fmt package for printf-style formatting
	return fmt.Sprintf(msg, args...)
}

// Global logger instance
var defaultLogger *Logger

// Initialize sets up the global logger
func Initialize(cfg Config) {
	defaultLogger = New(cfg)
}

// Default returns the default logger instance
func Default() *Logger {
	if defaultLogger == nil {
		// Fallback to a basic logger if not initialized
		defaultLogger = New(Config{Level: "info", Format: "text"})
	}
	return defaultLogger
}
