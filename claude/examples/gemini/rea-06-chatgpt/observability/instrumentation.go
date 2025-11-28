// https://docs.restate.dev/use-cases/microservice-orchestration#comparison-with-other-solutions
// Restate has built-in UI & execution tracing
// this package is left as an exercise if you want to integrate observability
// to third-party SaaS
package observability

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/pithomlabs/rea"

	restate "github.com/restatedev/sdk-go"
)

// IdempotencyInstrumentedHandler wraps a handler with comprehensive idempotency metrics and tracing
type IdempotencyInstrumentedHandler[I, O any] struct {
	handler func(restate.Context, I) (O, error)
	name    string
	metrics *rea.MetricsCollector
	hooks   *rea.ObservabilityHooks
	tracing *rea.TracingContext
	logger  *slog.Logger
}

// NewIdempotencyInstrumentedHandler creates a new instrumented handler wrapper
func NewIdempotencyInstrumentedHandler[I, O any](
	name string,
	handler func(restate.Context, I) (O, error),
	metrics *rea.MetricsCollector,
	hooks *rea.ObservabilityHooks,
	logger *slog.Logger,
) *IdempotencyInstrumentedHandler[I, O] {
	return &IdempotencyInstrumentedHandler[I, O]{
		handler: handler,
		name:    name,
		metrics: metrics,
		hooks:   hooks,
		logger:  logger,
	}
}

// Handle executes the wrapped handler with instrumentation
func (h *IdempotencyInstrumentedHandler[I, O]) Handle(
	ctx restate.Context,
	input I,
) (O, error) {
	// Record start time
	start := time.Now()

	// Create trace span
	var span *rea.OpenTelemetrySpan
	if h.tracing != nil {
		span = h.tracing.StartSpan(h.name, map[string]string{
			"input_type": fmt.Sprintf("%T", input),
		})
	}

	// Call invocation start hook
	if h.hooks != nil && h.hooks.OnInvocationStart != nil {
		h.hooks.OnInvocationStart("handler", h.name, input)
	}

	// Log invocation start
	h.logger.Info("handler_execution_started",
		"handler", h.name,
		"timestamp", start,
	)

	// Increment active invocations
	if h.metrics != nil {
		h.metrics.IncrementActiveInvocations(h.name)
	}

	// Execute handler
	result, err := h.handler(ctx, input)
	duration := time.Since(start)

	// Decrement active invocations
	if h.metrics != nil {
		h.metrics.DecrementActiveInvocations(h.name)
	}

	// Record metrics
	if h.metrics != nil {
		h.metrics.RecordInvocation(h.name, "execute", duration, err)
	}

	// Record end span
	if h.tracing != nil && span != nil {
		if err != nil {
			span.Status = "error"
		} else {
			span.Status = "success"
		}
		span.Attributes["duration_ms"] = fmt.Sprintf("%.2f", duration.Seconds()*1000)
		h.tracing.EndSpan(span, err)
	}

	// Call invocation end hook
	if h.hooks != nil && h.hooks.OnInvocationEnd != nil {
		h.hooks.OnInvocationEnd("handler", h.name, result, err, duration)
	}

	// Call error hook if error occurred
	if err != nil && h.hooks != nil && h.hooks.OnError != nil {
		h.hooks.OnError("handler_execution", err)
	}

	// Log invocation end
	h.logger.Info("handler_execution_completed",
		"handler", h.name,
		"duration", duration,
		"success", err == nil,
	)

	return result, err
}

// IdempotencyMetrics tracks idempotency-specific metrics
type IdempotencyMetrics struct {
	ValidationAttempts  int64
	ValidationPassed    int64
	ValidationFailed    int64
	DeduplicatedCalls   int64
	KeyGenerationErrors int64
	DuplicateDetections int64
}

// CollectIdempotencyMetrics extracts idempotency-specific metrics from collector
func CollectIdempotencyMetrics(mc *rea.MetricsCollector) IdempotencyMetrics {
	if mc == nil {
		return IdempotencyMetrics{}
	}

	allMetrics := mc.GetMetrics()

	return IdempotencyMetrics{
		ValidationAttempts:  getInt64(allMetrics, "idempotency_validations_total"),
		ValidationPassed:    getInt64(allMetrics, "idempotency_validations_passed"),
		ValidationFailed:    getInt64(allMetrics, "idempotency_validations_failed"),
		DeduplicatedCalls:   getInt64(allMetrics, "idempotency_deduped_calls"),
		KeyGenerationErrors: getInt64(allMetrics, "idempotency_key_generation_errors"),
		DuplicateDetections: getInt64(allMetrics, "idempotency_duplicate_detections"),
	}
}

// getInt64 safely extracts int64 from metrics map
func getInt64(metrics map[string]interface{}, key string) int64 {
	if val, ok := metrics[key]; ok {
		if i, ok := val.(int64); ok {
			return i
		}
	}
	return 0
}

// IdempotencyMetricsCollector aggregates idempotency-specific metrics
type IdempotencyMetricsCollector struct {
	baseCollector *rea.MetricsCollector
	logger        *slog.Logger
}

// NewIdempotencyMetricsCollector creates a new idempotency-focused metrics collector
func NewIdempotencyMetricsCollector(
	baseCollector *rea.MetricsCollector,
	logger *slog.Logger,
) *IdempotencyMetricsCollector {
	return &IdempotencyMetricsCollector{
		baseCollector: baseCollector,
		logger:        logger,
	}
}

// RecordValidation records an idempotency validation attempt
func (c *IdempotencyMetricsCollector) RecordValidation(passed bool, err error) {
	c.baseCollector.RecordInvocation(
		"idempotency",
		"validation",
		0,
		err,
	)

	if passed {
		c.logger.Info("idempotency_validation_passed")
	} else {
		c.logger.Warn("idempotency_validation_failed", "error", err)
	}
}

// RecordDuplicate records a detected duplicate idempotency key
func (c *IdempotencyMetricsCollector) RecordDuplicate(key string) {
	c.logger.Info("idempotency_duplicate_detected", "key", key)
	c.baseCollector.RecordInvocation(
		"idempotency",
		"duplicate_detected",
		0,
		nil,
	)
}

// RecordDeduplication records a successful deduplication
func (c *IdempotencyMetricsCollector) RecordDeduplication(key string) {
	c.logger.Info("idempotency_deduplication_successful", "key", key)
	c.baseCollector.RecordInvocation(
		"idempotency",
		"deduplication",
		0,
		nil,
	)
}

// RecordKeyGeneration records idempotency key generation
func (c *IdempotencyMetricsCollector) RecordKeyGeneration(
	source string,
	isDeterministic bool,
	err error,
) {
	c.logger.Info("idempotency_key_generated",
		"source", source,
		"deterministic", isDeterministic,
		"error", err,
	)

	c.baseCollector.RecordInvocation(
		"idempotency",
		"key_generation",
		0,
		err,
	)
}

// GetIdempotencyMetrics returns aggregated idempotency metrics
func (c *IdempotencyMetricsCollector) GetIdempotencyMetrics() IdempotencyMetrics {
	return CollectIdempotencyMetrics(c.baseCollector)
}
