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

		// Securely attach the authenticated UserID to the request context [6]
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

	// L2 Security: Propagate authenticated identity to the durable core [1]
	// Note: In production, use proper authentication headers at infrastructure level

	// Ingress Client call to the UserSession Virtual Object (Service: Object, Key: UserID, Method: AddItem)
	// This uses Request() for a synchronous immediate responsecheck.
	// Note: Headers are managed at the Restate infrastructure level, not via SDK options
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

	// Note: In production, use proper authentication headers at infrastructure level

	// Ingress Client call to the UserSession Virtual Object to initiate Checkout.
	// We use ObjectSend() for durable, asynchronous initiation, acknowledging receipt immediately.
	// The rest of the durable execution (saga, payment, shipping) proceeds in the background. [1]
	// Note: Headers are managed at the Restate infrastructure level
	_, err := restateingress.ObjectSend[string](i.client, "UserSession", userID, "Checkout").Send(ctx, orderID)

	if err != nil {
		log.Printf("Error initiating checkout for %s: %v", userID, err)
		http.Error(w, fmt.Sprintf("Restate Durable Send Failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	fmt.Fprintf(w, "Checkout initiated for Order ID: %s. Processing durably...", orderID)
}

func main() {
	// Initialize Restate Ingress Client
	restateClient := restateingress.NewClient(RESTATE_INGRESS_URL)
	ingressHandler := &Ingress{client: restateClient}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Define protected routes that require L1 authentication and context setting
	r.Route("/api/v1/user/{userID}", func(r chi.Router) {
		r.Use(authMiddleware)

		r.Post("/add-item", ingressHandler.handleAddItem)  // Example: curl -X POST -H 'X-API-Key: super-secret-ingress-key' http://localhost:8080/api/v1/user/azmy/add-item -d '"ticket-3"'
		r.Post("/checkout", ingressHandler.handleCheckout) // Example: curl -X POST -H 'X-API-Key: super-secret-ingress-key' http://localhost:8080/api/v1/user/azmy/checkout
	})

	log.Println("Starting Ingress Handler on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal(err)
	}
}
