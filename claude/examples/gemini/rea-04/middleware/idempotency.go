package middleware

import (
	"log/slog"
	"net/http"
	"strings"
)

// IdempotencyValidationMiddleware validates Idempotency-Key headers on ingress requests
type IdempotencyValidationMiddleware struct {
	logger *slog.Logger
	strict bool // If true, reject requests without valid Idempotency-Key
}

// NewIdempotencyValidationMiddleware creates a new idempotency validation middleware
func NewIdempotencyValidationMiddleware(
	logger *slog.Logger,
	strict bool,
) *IdempotencyValidationMiddleware {
	return &IdempotencyValidationMiddleware{
		logger: logger,
		strict: strict,
	}
}

// Middleware wraps an http.Handler with idempotency validation
func (m *IdempotencyValidationMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract Idempotency-Key header
		idempotencyKey := r.Header.Get("Idempotency-Key")

		// Validate key presence
		if idempotencyKey == "" {
			m.logger.Warn("idempotency_key_missing",
				"path", r.URL.Path,
				"method", r.Method,
			)

			// If strict policy, reject request
			if m.strict {
				http.Error(w, "Missing required Idempotency-Key header", http.StatusBadRequest)
				return
			}
		} else {
			// Validate key format (UUID-like or alphanumeric with dashes/underscores)
			if !isValidIdempotencyKeyFormat(idempotencyKey) {
				m.logger.Warn("idempotency_key_invalid_format",
					"key", idempotencyKey,
					"path", r.URL.Path,
				)

				// If strict policy, reject
				if m.strict {
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

		// Attach to context for downstream handlers
		r.Header.Set("X-Idempotency-Key", idempotencyKey)

		// Continue to next handler
		next.ServeHTTP(w, r)
	})
}

// isValidIdempotencyKeyFormat checks if a key is in valid format
func isValidIdempotencyKeyFormat(key string) bool {
	// Accept:
	// - UUIDs (36 chars with dashes)
	// - SHA256 hashes (64 hex chars)
	// - Deterministic patterns like "order:userid:orderid"
	// - Alphanumeric with dashes, underscores, colons
	
	if len(key) == 0 {
		return false
	}
	
	// Check length constraints
	if len(key) > 256 {
		return false
	}
	
	// UUID pattern
	if len(key) == 36 && isUUID(key) {
		return true
	}
	
	// SHA256 hash (64 hex chars)
	if len(key) == 64 && isHexString(key) {
		return true
	}
	
	// Alphanumeric with dashes, underscores, colons
	for _, ch := range key {
		if !((ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '-' || ch == '_' || ch == ':') {
			return false
		}
	}
	
	return true
}

// isUUID checks if a string is a valid UUID format
func isUUID(s string) bool {
	// Pattern: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	if len(s) != 36 {
		return false
	}
	if s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return false
	}
	for _, ch := range s {
		if ch == '-' {
			continue
		}
		if !((ch >= '0' && ch <= '9') ||
			(ch >= 'a' && ch <= 'f') ||
			(ch >= 'A' && ch <= 'F')) {
			return false
		}
	}
	return true
}

// isHexString checks if a string contains only hex characters
func isHexString(s string) bool {
	for _, ch := range s {
		if !((ch >= '0' && ch <= '9') ||
			(ch >= 'a' && ch <= 'f') ||
			(ch >= 'A' && ch <= 'F')) {
			return false
		}
	}
	return true
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
		return isHexString(key) // Valid hex = valid SHA256
	}

	return false
}
