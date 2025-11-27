package main

import (
	"context"
	"fmt"
	"os"
	"rea-05/models"
	"time"

	"github.com/pithomlabs/rea"
	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/server"
)

// --- 1. Stateless Service: ShippingService (External Integration) ---

type ShippingService struct{}

func (ShippingService) InitiateShipment(ctx restate.Context, shipment models.ShipmentRequest) (bool, error) {
	ctx.Log().Info("ShippingService received request", "order_id", shipment.OrderID)

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
			ctx.Log().Error("Shipping failed due to external error", "order_id", shipment.OrderID)
			return false, fmt.Errorf("external shipping API error")
		}
		ctx.Log().Info("Successfully registered shipment", "order_id", shipment.OrderID)
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
		ctx.Log().Info("COMPENSATION: Cancelling shipment", "order_id", orderID)
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

	ctx.Log().Info("User basket updated", "user_id", userID, "item_count", len(basket))
	return true, nil
}

// Durable waiting using Awakeables (Virtual Object primitive)
// PHASE 1: Uses State[T] for deduplication of idempotency keys
func (UserSession) Checkout(ctx restate.ObjectContext, orderID string) (bool, error) {
	userID := restate.Key(ctx)
	ctx.Log().Info("User initiating checkout", "user_id", userID, "order_id", orderID)

	// PHASE 1: Check for duplicate execution using State[T]
	// This implements Pattern C: Explicit State-Based Deduplication
	dedupKey := fmt.Sprintf("checkout:exec:%s", orderID)

	// Check if this checkout was already executed (using restate.Get from SDK)
	executed, err := restate.Get[bool](ctx, dedupKey)
	if err == nil && executed {
		ctx.Log().Info("Duplicate checkout detected, returning cached result", "order_id", orderID)
		// Return success without re-executing
		return true, nil
	}

	// Step 1: Reserve Inventory (Durable RPC Call)
	// Placeholder: This call would typically go to an Inventory Virtual Object

	// Step 2: Create Awakeable for Payment Completion
	awakeable := restate.Awakeable[models.PaymentReceipt](ctx)
	id := awakeable.Id()

	// Data Plane: Send the awakeable ID to an external payment system
	_, err = restate.Run(ctx, func(ctx restate.RunContext) (bool, error) {
		ctx.Log().Info("Notifying external Payment Gateway", "awakeable_id", id)
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
		ctx.Log().Error("Payment failed", "order_id", orderID, "error", err)
		return false, restate.TerminalError(fmt.Errorf("Payment failure: %v", err), 500)
	}

	ctx.Log().Info("Payment successful", "order_id", orderID, "transaction_id", receipt.TransactionID)

	// PHASE 1: Mark this checkout as executed for deduplication (using restate.Set from SDK)
	restate.Set(ctx, dedupKey, true)
	ctx.Log().Info("Marked checkout as executed", "order_id", orderID)

	// Step 3: Clear State using rea framework helper
	if err := rea.ClearAll(ctx); err != nil {
		return false, err
	}
	ctx.Log().Info("Session state cleared")

	// Launch Saga Workflow (asynchronous)
	orderPayload := models.Order{
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
func (OrderFulfillmentWorkflow) Run(ctx restate.WorkflowContext, order models.Order) error {
	ctx.Log().Info("Workflow started", "order_id", order.OrderID)

	// L2: Use authenticated user ID from order (set by ingress)
	userID := order.UserID
	ctx.Log().Info("Processing order with authenticated user", "user_id", userID, "order_id", order.OrderID)

	// Compensation stack setup: ensures bulletproof transactions using Go defer
	// Compensations are registered but are only executed if the handler exits with a non-recoverable error (panic/TerminalError)

	// 1. Process Reservation Step (Simulated)
	defer func() {
		// This compensation is run last (FILO order)
		ctx.Log().Info("COMPENSATION: Inventory reservation released", "order_id", order.OrderID)
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
	ctx.Log().Info("Waiting for admin approval", "order_id", order.OrderID)

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
	ctx.Log().Info("Admin approved order", "order_id", order.OrderID)

	// Step 3: Initiate Shipping (Durable Call)
	shipmentReq := models.ShipmentRequest{OrderID: order.OrderID, Address: order.ShippingAddress}

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
	ctx.Log().Info("Shipping initiated, waiting for delivery confirmation")
	restate.Sleep(ctx, 5*time.Second) // Durable Sleep
	ctx.Log().Info("Delivery timer expired, order assumed delivered")

	// Final Step: Complete workflow
	ctx.Log().Info("Order fulfillment complete", "order_id", order.OrderID)
	return nil
}

// Handler to interact concurrently with the Workflow using Durable Promises
func (OrderFulfillmentWorkflow) OnApprove(ctx restate.WorkflowSharedContext, orderID string) error {
	// Resolves the named Durable Promise, allowing the Run handler to continue
	ctx.Log().Info("Shared context handler received approval signal, resolving promise", "order_id", orderID)

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
		fmt.Fprintf(os.Stderr, "Restate server failed to start: %v\n", err)
		os.Exit(1)
	}
}

// Helper to manually complete an Awakeable (used for simulating external payment callback)
func ResolveAwakeableManually(restateURL, id string) {
	// This would use the ingress client to resolve awakeables from external systems
	// For demonstration purposes only
	fmt.Printf("Would resolve awakeable %s at %s\n", id, restateURL)
}
