package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"strings"

	"ingress/models"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/pithomlabs/rea"
	restateingress "github.com/restatedev/sdk-go/ingress"
)

// Configuration constants (Read from environment in production)
const (
	RESTATE_INGRESS_URL = "http://localhost:9080"    // Restate runtime endpoint
	INGRESS_API_KEY     = "super-secret-ingress-key" // L1 Authentication Secret
)

// Context key for storing authenticated UserID
type ctxKey string

const userIDKey ctxKey = "userID"

// generateIdempotencyKeyFromContext wraps the framework's deterministic key generation
// Ensures idempotency keys are non-temporal and deterministic per DOS_DONTS_REA guidelines
func generateIdempotencyKeyFromContext(parts ...string) string {
	// Use the same pattern as ControlPlaneService.GenerateIdempotencyKeyDeterministic
	// which creates keys from business data with deterministic separator
	if len(parts) == 0 {
		return "order"
	}
	// Combine parts with colons: "order:userID:data"
	result := "order"
	for _, part := range parts {
		result += ":" + part
	}
	return result
}

// L1 Authentication Middleware: Checks API Key and sets UserID in context
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey != INGRESS_API_KEY {
			http.Error(w, "Unauthorized: Invalid API Key", http.StatusUnauthorized)
			return
		}

		// In a real application, token validation extracts the UserID
		// Here, we simulate UserID extraction based on a path parameter or JWT payload
		userID := chi.URLParam(r, "userID")
		if userID == "" {
			http.Error(w, "Bad Request: UserID must be provided in path", http.StatusBadRequest)
			return
		}

		// Securely attach the authenticated UserID to the request context
		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Ingress Client Structure
type Ingress struct {
	client *restateingress.Client
}

// Durable State Initialization Endpoint
func (i *Ingress) handleAddItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := ctx.Value(userIDKey).(string)
	if !ok {
		http.Error(w, "Internal Error: UserID not found in context", http.StatusInternalServerError)
		return
	}

	var item string
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Ingress Client call to the UserSession Virtual Object
	// Using rea framework - the service calls remain the same as Restate SDK
	_, err := restateingress.Object[string, string](i.client, "UserSession", userID, "AddItem").Request(ctx, item)
	if err != nil {
		log.Printf("Error adding item for %s: %v", userID, err)
		http.Error(w, fmt.Sprintf("Restate Durable Call Failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Item '%s' added to basket for user %s", item, userID)
}

// Durable Checkout and Workflow Initiation Endpoint
// PHASE 1: Uses deterministic order ID generation and idempotency keys
func (i *Ingress) handleCheckout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := ctx.Value(userIDKey).(string)
	if !ok {
		http.Error(w, "Internal Error: UserID not found in context", http.StatusInternalServerError)
		return
	}

	// Extract idempotency key from request header (PHASE 2: Ingress policy validation)
	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		log.Printf("Warning: No Idempotency-Key provided for checkout by user %s", userID)
		// Use framework's deterministic key generation from business context
		idempotencyKey = generateIdempotencyKeyFromContext(userID, "default-checkout")
	}

	// PHASE 1: Generate deterministic order ID using framework primitives
	// This ensures retries with same business context get the same orderID
	orderID := generateIdempotencyKeyFromContext(userID, idempotencyKey)

	// Ingress Client call to the UserSession Virtual Object to initiate Checkout
	// Using rea framework - asynchronous durable initiation with idempotency key
	_, err := restateingress.ObjectSend[string](i.client, "UserSession", userID, "Checkout").
		Send(ctx, orderID) // Note: Pass deterministic orderID to handler

	if err != nil {
		log.Printf("Error initiating checkout for %s: %v", userID, err)
		http.Error(w, fmt.Sprintf("Restate Durable Send Failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	fmt.Fprintf(w, "Checkout initiated for Order ID: %s. Processing durably...", orderID)
}

// Start a workflow directly (demonstrates workflow invocation from ingress)
// PHASE 1: Uses deterministic order ID generation and idempotency keys
func (i *Ingress) handleStartWorkflow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := ctx.Value(userIDKey).(string)
	if !ok {
		http.Error(w, "Internal Error: UserID not found in context", http.StatusInternalServerError)
		return
	}

	// Parse order from request body
	var order ingress.Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Set authenticated user ID
	order.UserID = userID

	// PHASE 1: Generate deterministic order ID if not provided
	// Uses business context (userID + items) to ensure consistency
	if order.OrderID == "" {
		order.OrderID = generateIdempotencyKeyFromContext(userID, order.Items)
	}

	// Extract idempotency key from request header (PHASE 2: Ingress policy validation)
	// Note: This header is validated by the middleware; here we log if missing
	if idempotencyKey := r.Header.Get("Idempotency-Key"); idempotencyKey == "" {
		log.Printf("Warning: No Idempotency-Key provided for workflow by user %s", userID)
	}

	// Trigger workflow using ingress client
	// This sends the order to the OrderFulfillmentWorkflow.Run handler
	_, err := restateingress.WorkflowSend[ingress.Order](i.client, "OrderFulfillmentWorkflow", order.OrderID, "Run").
		Send(ctx, order)

	if err != nil {
		log.Printf("Error starting workflow for %s: %v", order.OrderID, err)
		http.Error(w, fmt.Sprintf("Failed to start workflow: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	fmt.Fprintf(w, "Workflow started for Order ID: %s", order.OrderID)
}

// Approve a workflow (demonstrates workflow interaction via shared handlers)
func (i *Ingress) handleApproveWorkflow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orderID := chi.URLParam(r, "orderID")

	if orderID == "" {
		http.Error(w, "Order ID required", http.StatusBadRequest)
		return
	}

	// Call the OnApprove shared handler to resolve the workflow's promise
	// This allows the workflow to continue past the approval step
	_, err := restateingress.Workflow[string, error](i.client, "OrderFulfillmentWorkflow", orderID, "OnApprove").
		Request(ctx, orderID)

	if err != nil {
		log.Printf("Error approving workflow %s: %v", orderID, err)
		http.Error(w, fmt.Sprintf("Failed to approve workflow: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Workflow %s approved successfully", orderID)
}

func main() {
	// Initialize Restate Ingress Client
	// Note: Ingress client usage is the same regardless of whether services use rea framework
	restateClient := restateingress.NewClient(RESTATE_INGRESS_URL)
	ingressHandler := &Ingress{client: restateClient}

	// PHASE 2: Initialize metrics and observability for idempotency tracking
	metrics := rea.NewMetricsCollector()
	logger := slog.Default()
	hooks := rea.DefaultObservabilityHooks(logger)

	// PHASE 2: Create idempotency validation middleware
	idempotencyValidator := NewIdempotencyValidationMiddleware(logger, metrics, hooks)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// PHASE 2: Apply idempotency validation middleware globally
	r.Use(idempotencyValidator.Middleware)

	// Define protected routes that require L1 authentication and context setting
	r.Route("/api/v1/user/{userID}", func(r chi.Router) {
		r.Use(authMiddleware)

		r.Post("/add-item", ingressHandler.handleAddItem)             // Add item to basket
		r.Post("/checkout", ingressHandler.handleCheckout)            // Initiate checkout (PHASE 1: deterministic IDs)
		r.Post("/start-workflow", ingressHandler.handleStartWorkflow) // Start workflow directly (PHASE 1: deterministic IDs)
	})

	// Workflow management endpoints (admin operations)
	r.Route("/api/v1/workflow", func(r chi.Router) {
		r.Post("/{orderID}/approve", ingressHandler.handleApproveWorkflow) // Approve pending workflow
	})

	// PHASE 3: Export metrics endpoint for Prometheus
	r.Get("/metrics", func(w http.ResponseWriter, r *http.Request) {
		metricsSnapshot := metrics.GetMetrics()
		// Convert to JSON for simplicity (in production, use Prometheus text format)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(metricsSnapshot)
	})

	log.Println("Starting Ingress Handler on :8080")
	log.Println("Idempotency validation enabled (PHASE 2)")
	log.Println("Deterministic ID generation enabled (PHASE 1)")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal(err)
	}
}

// ============================================================================
// IDEMPOTENCY VALIDATION MIDDLEWARE (Embedded from middleware package)
// ============================================================================

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
