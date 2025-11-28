package config

import (
	"log/slog"
	"os"
)

// IdempotencyConfig holds configuration for idempotency features
type IdempotencyConfig struct {
	// Strict mode: reject requests without valid Idempotency-Key
	Strict bool

	// Logger for structured logging
	Logger *slog.Logger
}

// LoadIdempotencyConfig loads configuration from environment and defaults
func LoadIdempotencyConfig() *IdempotencyConfig {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Determine strict mode from environment
	strict := os.Getenv("RESTATE_STRICT_IDEMPOTENCY") == "true"

	return &IdempotencyConfig{
		Strict: strict,
		Logger: logger,
	}
}

// ApplyConfig applies the configuration globally
func (c *IdempotencyConfig) ApplyConfig() {
	c.Logger.Info("IdempotencyConfig applied",
		"strict_mode", c.Strict,
	)
}

// GetStrictMode returns the strict mode setting
func (c *IdempotencyConfig) GetStrictMode() bool {
	return c.Strict
}

// GetLogger returns the configured logger
func (c *IdempotencyConfig) GetLogger() *slog.Logger {
	return c.Logger
}


