package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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

// Data structures matching services
type Order struct {
	OrderID     string
	UserID      string
	Items       string
	AmountCents int
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
func (i *Ingress) handleCheckout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := ctx.Value(userIDKey).(string)
	if !ok {
		http.Error(w, "Internal Error: UserID not found in context", http.StatusInternalServerError)
		return
	}

	// In a real scenario, orderID would be generated or received.
	orderID := fmt.Sprintf("ORDER-%d", time.Now().UnixNano())

	// Ingress Client call to the UserSession Virtual Object to initiate Checkout
	// Using rea framework - asynchronous durable initiation
	_, err := restateingress.ObjectSend[string](i.client, "UserSession", userID, "Checkout").Send(ctx, orderID)

	if err != nil {
		log.Printf("Error initiating checkout for %s: %v", userID, err)
		http.Error(w, fmt.Sprintf("Restate Durable Send Failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	fmt.Fprintf(w, "Checkout initiated for Order ID: %s. Processing durably...", orderID)
}

// Start a workflow directly (demonstrates workflow invocation from ingress)
func (i *Ingress) handleStartWorkflow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := ctx.Value(userIDKey).(string)
	if !ok {
		http.Error(w, "Internal Error: UserID not found in context", http.StatusInternalServerError)
		return
	}

	// Parse order from request body
	var order Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Set authenticated user ID
	order.UserID = userID

	// Generate order ID if not provided
	if order.OrderID == "" {
		order.OrderID = fmt.Sprintf("ORDER-%d", time.Now().UnixNano())
	}

	// Trigger workflow using ingress client
	// This sends the order to the OrderFulfillmentWorkflow.Run handler
	_, err := restateingress.WorkflowSend[Order](i.client, "OrderFulfillmentWorkflow", order.OrderID, "Run").
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

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Define protected routes that require L1 authentication and context setting
	r.Route("/api/v1/user/{userID}", func(r chi.Router) {
		r.Use(authMiddleware)

		r.Post("/add-item", ingressHandler.handleAddItem)             // Add item to basket
		r.Post("/checkout", ingressHandler.handleCheckout)            // Initiate checkout
		r.Post("/start-workflow", ingressHandler.handleStartWorkflow) // Start workflow directly
	})

	// Workflow management endpoints (admin operations)
	r.Route("/api/v1/workflow", func(r chi.Router) {
		r.Post("/{orderID}/approve", ingressHandler.handleApproveWorkflow) // Approve pending workflow
	})

	log.Println("Starting Ingress Handler on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal(err)
	}
}
