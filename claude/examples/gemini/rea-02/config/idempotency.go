package config

import (
	"log/slog"
	"os"

	"github.com/pithomlabs/rea"
)

// IdempotencyConfig holds configuration for idempotency features
type IdempotencyConfig struct {
	// ValidationMode controls how idempotency key validation failures are handled
	ValidationMode rea.IdempotencyValidationMode

	// FrameworkPolicy controls the strictness of framework guardrails
	FrameworkPolicy rea.FrameworkPolicy

	// EnableMetrics controls whether metrics collection is enabled
	EnableMetrics bool

	// EnableTracing controls whether distributed tracing is enabled
	EnableTracing bool

	// Logger for structured logging
	Logger *slog.Logger
}

// LoadIdempotencyConfig loads configuration from environment and defaults
func LoadIdempotencyConfig() *IdempotencyConfig {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Determine policy from environment or auto-detect from CI
	var policy rea.FrameworkPolicy
	if policyEnv := os.Getenv("RESTATE_FRAMEWORK_POLICY"); policyEnv != "" {
		policy = rea.FrameworkPolicy(policyEnv)
	} else if os.Getenv("CI") == "true" {
		// Auto-detect: CI environment → strict policy
		policy = rea.PolicyStrict
	} else {
		// Development: warn by default
		policy = rea.PolicyWarn
	}

	// Determine validation mode from environment
	var validationMode rea.IdempotencyValidationMode
	if modeEnv := os.Getenv("RESTATE_IDEMPOTENCY_VALIDATION"); modeEnv != "" {
		validationMode = rea.IdempotencyValidationMode(modeEnv)
	} else {
		validationMode = rea.IdempotencyValidationWarn // Default to warn mode
	}

	return &IdempotencyConfig{
		ValidationMode:  validationMode,
		FrameworkPolicy: policy,
		EnableMetrics:   true,
		EnableTracing:   os.Getenv("ENABLE_TRACING") == "true",
		Logger:          logger,
	}
}

// ApplyConfig applies the configuration globally
func (c *IdempotencyConfig) ApplyConfig() {
	// Set global framework policy
	rea.SetFrameworkPolicy(c.FrameworkPolicy)

	c.Logger.Info("IdempotencyConfig applied",
		"policy", c.FrameworkPolicy,
		"validation_mode", c.ValidationMode,
		"metrics_enabled", c.EnableMetrics,
		"tracing_enabled", c.EnableTracing,
	)
}

// InitializeMiddleware creates and returns configured middleware components
func (c *IdempotencyConfig) InitializeMiddleware() (
	*rea.MetricsCollector,
	*rea.ObservabilityHooks,
	*rea.IdempotencyValidationMiddleware,
) {
	// Create metrics collector
	metrics := rea.NewMetricsCollector()

	// Create observability hooks with logger
	hooks := rea.DefaultObservabilityHooks(c.Logger)

	// Create idempotency validation middleware
	idempotencyValidator := rea.NewIdempotencyValidationMiddleware(c.Logger, metrics, hooks)

	return metrics, hooks, idempotencyValidator
}
