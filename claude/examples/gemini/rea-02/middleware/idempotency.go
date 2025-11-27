package middleware

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/pithomlabs/rea"
)

// IdempotencyValidationMiddleware validates Idempotency-Key headers on ingress requests
type IdempotencyValidationMiddleware struct {
	policy  rea.FrameworkPolicy
	metrics *rea.MetricsCollector
	hooks   *rea.ObservabilityHooks
	logger  *slog.Logger
}

// NewIdempotencyValidationMiddleware creates a new idempotency validation middleware
func NewIdempotencyValidationMiddleware(
	logger *slog.Logger,
	metrics *rea.MetricsCollector,
	hooks *rea.ObservabilityHooks,
) *IdempotencyValidationMiddleware {
	return &IdempotencyValidationMiddleware{
		policy:  rea.GetFrameworkPolicy(),
		metrics: metrics,
		hooks:   hooks,
		logger:  logger,
	}
}

// Middleware wraps an http.Handler with idempotency validation
func (m *IdempotencyValidationMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract Idempotency-Key header
		idempotencyKey := r.Header.Get("Idempotency-Key")

		// Validate key presence
		if idempotencyKey == "" {
			violation := rea.GuardrailViolation{
				Check:    "idempotency_key_missing",
				Message:  "Idempotency-Key header is required",
				Severity: "error",
			}

			// Route violation per policy
			err := rea.HandleGuardrailViolation(violation, m.logger, m.policy)

			// Record metrics
			m.metrics.RecordInvocation("ingress", "validate_idempotency", 0, err)

			// Call hook
			if m.hooks != nil && m.hooks.OnError != nil {
				m.hooks.OnError("idempotency_validation_missing_key", err)
			}

			// If strict policy, reject request
			if m.policy == rea.PolicyStrict && err != nil {
				http.Error(w, "Missing required Idempotency-Key header", http.StatusBadRequest)
				return
			}
		} else {
			// Validate key format
			if err := rea.ValidateIdempotencyKey(idempotencyKey); err != nil {
				violation := rea.GuardrailViolation{
					Check:    "idempotency_key_invalid_format",
					Message:  "Idempotency-Key format is invalid: " + err.Error(),
					Severity: "warning",
				}

				// Route violation per policy
				errResult := rea.HandleGuardrailViolation(violation, m.logger, m.policy)

				// Record metrics
				m.metrics.RecordInvocation("ingress", "validate_idempotency_format", 0, errResult)

				// Call hook
				if m.hooks != nil && m.hooks.OnError != nil {
					m.hooks.OnError("idempotency_validation_format", errResult)
				}

				// If strict policy, reject
				if m.policy == rea.PolicyStrict && errResult != nil {
					http.Error(w, "Invalid Idempotency-Key format", http.StatusBadRequest)
					return
				}
			}
		}

		// Log successful validation
		m.logger.Info("idempotency_key_validated",
			"key", idempotencyKey,
			"path", r.URL.Path,
		)

		// Call hook for successful validation
		if m.hooks != nil && m.hooks.OnInvocationStart != nil {
			m.hooks.OnInvocationStart("ingress", r.URL.Path, map[string]string{
				"idempotency_key": idempotencyKey,
			})
		}

		// Attach to context for downstream handlers
		r.Header.Set("X-Idempotency-Key", idempotencyKey)

		// Continue to next handler
		next.ServeHTTP(w, r)
	})
}

// RequestWithIdempotencyKey extracts idempotency key from request
func RequestWithIdempotencyKey(r *http.Request) string {
	// First check our internal header (set by middleware)
	if key := r.Header.Get("X-Idempotency-Key"); key != "" {
		return key
	}
	// Fall back to client header
	return r.Header.Get("Idempotency-Key")
}

// IsIdempotencyKeyDeterministic checks if a key appears to be deterministically generated
func IsIdempotencyKeyDeterministic(key string) bool {
	// Keys should follow patterns like: "order:userid:orderid:v1" or "exec:hash"
	// Should NOT contain timestamps or random strings
	deterministicPatterns := []string{
		"order:", "exec:", "result:", "payment:", "shipment:",
	}

	for _, pattern := range deterministicPatterns {
		if strings.HasPrefix(key, pattern) {
			return true
		}
	}

	// Check if it's a SHA256 hash (deterministic)
	if len(key) == 64 { // SHA256 hex output length
		err := rea.ValidateIdempotencyKey(key)
		return err == nil
	}

	return false
}
