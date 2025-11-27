package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/pithomlabs/rea"
	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/server"
)

// --- Configuration and Data Structures ---

type Order struct {
	OrderID     string
	UserID      string
	Items       string
	AmountCents int
}

type ShipmentRequest struct {
	OrderID string
	Address string
}

type PaymentReceipt struct {
	TransactionID string
	Success       bool
}

// --- 1. Stateless Service: ShippingService (External Integration) ---

type ShippingService struct{}

func (ShippingService) InitiateShipment(ctx restate.Context, shipment ShipmentRequest) (bool, error) {
	log.Printf("ShippingService received request for Order %s", shipment.OrderID)

	// Data Plane Operation: using rea.RunWithRetry for enhanced retry patterns
	cfg := rea.RunConfig{
		MaxRetries:    3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      2 * time.Second,
		BackoffFactor: 2.0,
		Name:          "InitiateShipment",
	}

	success, err := rea.RunWithRetry(ctx, cfg, func(ctx restate.RunContext) (bool, error) {
		// Simulate non-deterministic external I/O (e.g., HTTP request to FedEx/UPS)
		time.Sleep(100 * time.Millisecond)
		if shipment.OrderID == "FAIL_SHIP" {
			log.Println("Shipping failed due to external error")
			return false, fmt.Errorf("external shipping API error")
		}
		log.Printf("Successfully registered shipment for Order %s", shipment.OrderID)
		return true, nil
	})

	if err != nil {
		// Non-terminal error leads to retry of the whole handler (including Run block)
		return false, err
	}
	if !success {
		// If the external API reports a non-retryable business error, we throw a TerminalError
		return false, restate.TerminalError(fmt.Errorf("Shipping company rejected shipment"), 400)
	}

	return true, nil
}

func (ShippingService) CancelShipment(ctx restate.Context, orderID string) (bool, error) {
	// Durable execution guarantees this compensation runs until completion
	_, err := restate.Run(ctx, func(ctx restate.RunContext) (bool, error) {
		log.Printf("COMPENSATION: Cancelling shipment for Order %s", orderID)
		//... Actual call to external API to cancel shipment...
		return true, nil
	})
	return err == nil, err
}

// --- 2. Virtual Object: UserSession (Stateful Actor) ---

type UserSession struct{}

func (UserSession) AddItem(ctx restate.ObjectContext, item string) (bool, error) {
	// Simplified: In production, validate authenticated user matches object key
	userID := restate.Key(ctx)

	// Durable State Management
	basket, err := restate.Get[[]string](ctx, "basket")
	if err != nil {
		return false, err
	}
	if basket == nil {
		basket = make([]string, 0)
	}

	basket = append(basket, item)

	// SetState is a control plane operation, journaled for durability
	restate.Set(ctx, "basket", basket)

	log.Printf("User %s basket now has %d items.", userID, len(basket))
	return true, nil
}

// Durable waiting using Awakeables (Virtual Object primitive)
// PHASE 1: Uses State[T] for deduplication of idempotency keys
func (UserSession) Checkout(ctx restate.ObjectContext, orderID string) (bool, error) {
	userID := restate.Key(ctx)
	log.Printf("User %s initiating checkout for %s. State is locked.", userID, orderID)

	// PHASE 1: Check for duplicate execution using State[T]
	// This implements Pattern C: Explicit State-Based Deduplication
	dedupKey := fmt.Sprintf("checkout:exec:%s", orderID)
	
	// Check if this checkout was already executed (using restate.Get from SDK)
	executed, err := restate.Get[bool](ctx, dedupKey)
	if err == nil && executed {
		log.Printf("Duplicate checkout detected for order %s. Returning cached result.", orderID)
		// Return success without re-executing
		return true, nil
	}

	// Step 1: Reserve Inventory (Durable RPC Call)
	// Placeholder: This call would typically go to an Inventory Virtual Object

	// Step 2: Create Awakeable for Payment Completion
	awakeable := restate.Awakeable[PaymentReceipt](ctx)
	id := awakeable.Id()

	// Data Plane: Send the awakeable ID to an external payment system
	_, err = restate.Run(ctx, func(ctx restate.RunContext) (bool, error) {
		log.Printf("Notifying external Payment Gateway with Awakeable ID: %s", id)
		// In a real system, this would be an HTTP POST to a payment processor, including the ID
		return true, nil
	})
	if err != nil {
		return false, err
	}

	// Execution Suspends: Wait for external system to resolve the awakeable
	receipt, err := awakeable.Result()
	if err != nil {
		// If awakeable failed, throw terminal error
		log.Printf("Payment failed for %s: %v", orderID, err)
		return false, restate.TerminalError(fmt.Errorf("Payment failure: %v", err), 500)
	}

	log.Printf("Payment successful for %s. Tx ID: %s", orderID, receipt.TransactionID)

	// PHASE 1: Mark this checkout as executed for deduplication (using restate.Set from SDK)
	restate.Set(ctx, dedupKey, true)
	log.Printf("Marked checkout as executed for order %s", orderID)

	// Step 3: Clear State using rea framework helper
	if err := rea.ClearAll(ctx); err != nil {
		return false, err
	}
	log.Printf("Session state cleared.")

	// Launch Saga Workflow (asynchronous)
	orderPayload := Order{
		OrderID:     orderID,
		UserID:      userID,
		Items:       "item-1,item-2", // Placeholder items
		AmountCents: 10000,
	}

	// Durable Send to Workflow entry point
	restate.WorkflowSend(ctx, "OrderFulfillmentWorkflow", orderID, "Run").Send(orderPayload)

	return true, nil
}

// --- 3. Workflow: OrderFulfillmentWorkflow (Saga and Coordination) ---

type OrderFulfillmentWorkflow struct{}

// The main, exactly-once execution handler for the workflow
func (OrderFulfillmentWorkflow) Run(ctx restate.WorkflowContext, order Order) error {
	log.Printf("Workflow started for Order ID: %s", order.OrderID)

	// Simplified: In production, validate L2 authenticated user identity
	userID := "admin"
	log.Printf("Validated L2 Identity: %s", userID)

	// Compensation stack setup: ensures bulletproof transactions using Go defer
	// Compensations are registered but are only executed if the handler exits with a non-recoverable error (panic/TerminalError)

	// 1. Process Reservation Step (Simulated)
	defer func() {
		// This compensation is run last (FILO order)
		log.Printf("COMPENSATION: Inventory reservation released for %s", order.OrderID)
	}()

	// 2. Shipping Initiation Step
	shippingCompensated := false
	defer func() {
		if shippingCompensated {
			// Compensation must be guarded to run only if the step succeeded initially
			restate.ServiceSend(ctx, "ShippingService", "CancelShipment").Send(order.OrderID)
		}
	}()

	// Simulate Admin Approval using Durable Promise (Coordination)
	// at this stage, the workflow is paused pending approval
	approval := restate.Promise[bool](ctx, "admin_approval")
	log.Printf("Waiting for admin approval for Order %s...", order.OrderID)

	// human-in-the-loop approval process is ongoing
	// when the approver is done, human clicks a link which triggers the
	// handleApproveWorkflow handler in ingress.go
	// which then triggers the OrderFulfillmentWorkflow OnApprove handler
	// in svcs.go

	approved, err := approval.Result()
	if err != nil {
		return fmt.Errorf("failed to wait for approval: %w", err)
	}
	if !approved {
		// If admin rejects, we stop the workflow and trigger compensations
		// TerminalError means no need for retry because this is THE business logic
		return restate.TerminalError(fmt.Errorf("Order rejected by administrator"), 400)
	}
	log.Printf("Admin approved Order %s.", order.OrderID)

	// Step 3: Initiate Shipping (Durable Call)
	shipmentReq := ShipmentRequest{OrderID: order.OrderID, Address: "123 Durable Way"}

	// interservice communication, an RPC within this microservices package (svcs.go)
	// it is not a durable promise since we don't want to pause the workflow
	// we will get a response immediately since it's an HTTP POST API to the shipping company
	// the InitiateShipment service manages all the error handling in its business logic
	_, err = restate.Service[bool](ctx, "ShippingService", "InitiateShipment").Request(shipmentReq)
	if err != nil {
		// Non-recoverable failure in the step triggers deferred compensations
		return restate.TerminalError(fmt.Errorf("Shipment initiation failed: %v", err), 500)
	}
	shippingCompensated = true // Mark step as successful, enabling its compensation if later steps fail

	// Step 4: Durable Timer (Simulate wait for delivery confirmation)
	log.Printf("Shipping initiated. Sleeping for 15 minutes (simulated 5 seconds)...")
	restate.Sleep(ctx, 5*time.Second) // Durable Sleep
	log.Printf("Delivery timer expired. Order assumed delivered.")

	// Final Step: Complete workflow
	log.Printf("Order fulfillment complete for %s. State is now committed.", order.OrderID)
	return nil
}

// Handler to interact concurrently with the Workflow using Durable Promises
func (OrderFulfillmentWorkflow) OnApprove(ctx restate.WorkflowSharedContext, orderID string) error {
	// Resolves the named Durable Promise, allowing the Run handler to continue
	log.Printf("Shared context handler received approval signal for %s. Resolving promise.", orderID)

	// this is where the business logic goes
	// if disapproved, return restate.Promise[bool](ctx, "admin_approval").Resolve(false)

	return restate.Promise[bool](ctx, "admin_approval").Resolve(true)
}

func main() {
	// Rea framework demonstration: Service registration uses standard Restate SDK
	// Rea provides utility functions used INSIDE handlers:
	// - RunWithRetry: Enhanced retry patterns with backoff
	// - Gather: Wait for multiple futures concurrently
	// - ClearAll: Safe state clearing with error handling
	// - FanOutFail: Fail-fast concurrent operations
	// - ProcessBatch: Batch processing with concurrency control

	if err := server.NewRestate().
		Bind(restate.Reflect(ShippingService{})).
		Bind(restate.Reflect(UserSession{})).
		Bind(restate.Reflect(OrderFulfillmentWorkflow{})).
		Start(context.Background(), "0.0.0.0:9080"); err != nil {
		log.Fatalf("Restate server failed to start: %v", err)
	}
}

// Helper to manually complete an Awakeable (used for simulating external payment callback)
func ResolveAwakeableManually(restateURL, id string) {
	// This would use the ingress client to resolve awakeables from external systems
	// For demonstration purposes only
	fmt.Printf("Would resolve awakeable %s at %s\n", id, restateURL)
}
