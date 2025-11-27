// main.go
//
// Single-file example using github.com/pithomlabs/rea framework
// Demonstrates services that use the rea framework for:
//  - State management (rea.State)
//  - Saga framework (rea.NewSaga)
//  - External/Internal signaling (rea.WaitForExternalSignal, rea.GetInternalSignal)
//  - Run helpers (rea.RunDo, rea.RunDoVoid)
//
// This follows all DOs & DON'Ts from the Restate framework and best practices.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	rea "github.com/pithomlabs/rea"
	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/server"
)

// -----------------------------------------------------------------------------
// Types used across services
// -----------------------------------------------------------------------------

type OrderRequest struct {
	OrderID string
	Item    string
	Amount  float64
	UserID  string
}

type OrderResponse struct {
	OrderID string
	Status  string
	Message string
}

// -----------------------------------------------------------------------------
// Service Implementations
// -----------------------------------------------------------------------------

// Inventory service (sync Request)
type InventoryService struct{}

func (s *InventoryService) CheckAvailability(ctx restate.Context, item string) (bool, error) {
	ctx.Log().Info("InventoryService: check", "item", item)
	// Simple deterministic logic (no Run needed)
	if item == "unavailable_item" {
		return false, nil
	}
	return true, nil
}

// Payment service: uses restate.Run to call external payment gateway simulation
type PaymentService struct{}

func (p *PaymentService) ProcessPayment(ctx restate.Context, req OrderRequest) (bool, error) {
	ctx.Log().Info("PaymentService: initiating payment", "order", req.OrderID, "amount", req.Amount)

	// Use rea.RunDo to perform the external HTTP call (simulated).
	// This ensures the result is recorded deterministically and replay-safe.
	res, err := rea.RunDo(ctx, func(rc restate.RunContext) (string, error) {
		// This closure runs as a single durable side-effect step.
		// Simulate non-deterministic external call:
		rc.Log().Info("Run: contacting external payment gateway (simulated)", "order", req.OrderID)

		// Simulated behavior:
		// - if amount > 1000 -> fail payment
		// - else succeed
		if req.Amount > 1000 {
			return "DECLINED", nil
		}
		return "APPROVED", nil
	}, restate.WithName("payment.gateway.call"))

	if err != nil {
		ctx.Log().Error("PaymentService: run failed", "err", err.Error())
		return false, err
	}

	ctx.Log().Info("PaymentService: gateway result", "result", res)
	return res == "APPROVED", nil
}

// AnalyticsService demonstrates using rea.RunDo (sync) to send a final order event.
type AnalyticsService struct{}

func (a *AnalyticsService) LogOrderEvent(ctx restate.Context, event map[string]any) (restate.Void, error) {
	ctx.Log().Info("AnalyticsService: logging order event")

	_, err := rea.RunDo(ctx, func(rc restate.RunContext) (restate.Void, error) {
		// Simulated HTTP call to analytics provider (side-effect).
		rc.Log().Info("Run: sending analytics event (simulated)", "order", event["order_id"])
		// pretend send; may fail in a real world call
		return restate.Void{}, nil
	}, restate.WithName("analytics.send"))
	return restate.Void{}, err
}

// ShippingService demonstrates RunAsync (background side-effect)
type ShippingService struct{}

func (s *ShippingService) RequestShipment(ctx restate.Context, orderID string) (string, error) {
	ctx.Log().Info("ShippingService: requesting shipment", "order", orderID)

	// We want to perform the external shipment scheduling in background to avoid blocking.
	fut := restate.RunAsync(ctx, func(rc restate.RunContext) (string, error) {
		rc.Log().Info("RunAsync: calling shipping provider (simulated)", "order", orderID)
		// Simulate provider returning a tracking id after some delay
		tracking := "TRACK-" + orderID
		rc.Log().Info("RunAsync: provider returned tracking", "tracking", tracking)
		return tracking, nil
	}, restate.WithName("shipping.request"))

	// Wait for result
	tracking, err := fut.Result()
	if err != nil {
		ctx.Log().Error("ShippingService: shipping request failed", "err", err.Error())
		return "", err
	}
	ctx.Log().Info("ShippingService: got tracking", "tracking", tracking)
	return tracking, nil
}

// CRMService demonstrates Run for a side-effect with no return.
type CRMService struct{}

type CRMRequest struct {
	UserID string
}

func (c *CRMService) SyncCustomer(ctx restate.Context, req CRMRequest) (restate.Void, error) {
	ctx.Log().Info("CRMService: sync customer", "user", req.UserID)

	err := rea.RunDoVoid(ctx, func(rc restate.RunContext) error {
		rc.Log().Info("Run: calling CRM provider (simulated)", "user", req.UserID)
		return nil
	}, restate.WithName("crm.sync"))

	return restate.Void{}, err
}

// NotificationService demonstrates Send (fire-and-forget) pattern.
type NotificationService struct{}

type EmailRequest struct {
	To   string
	Body string
}

func (n *NotificationService) SendEmail(ctx restate.Context, req EmailRequest) (restate.Void, error) {
	ctx.Log().Info("NotificationService: send email", "to", req.To)
	// Typically would call external email provider with Run/RunAsync, but as it's fire-and-forget
	// we show a quick synchronous log and return.
	return restate.Void{}, nil
}

// OrderWorkflow demonstrates usage of Saga, Promise and Run inside a workflow.
type OrderWorkflow struct{}

// Run is the main exclusive handler.
func (w *OrderWorkflow) Run(ctx restate.WorkflowContext, req OrderRequest) (string, error) {
	var err error
	// Build a saga using rea framework to allow compensation if downstream fails
	saga := rea.NewSaga(ctx, "order-saga", nil)
	defer saga.CompensateIfNeeded(&err)

	// Register compensations
	saga.Register("refund-payment", func(rc restate.RunContext, payload []byte) error {
		// In a real system we would call payment refund API inside rc (using rc.Run semantics),
		// but for this example we just log.
		var p OrderRequest
		json.Unmarshal(payload, &p)
		rc.Log().Warn("Compensation: refund-payment", "order", p.OrderID)
		return nil
	})
	saga.Register("remove-cart-item", func(rc restate.RunContext, payload []byte) error {
		var item string
		json.Unmarshal(payload, &item)
		rc.Log().Warn("Compensation: remove-cart-item", "item", item)
		return nil
	})

	// Persist steps
	_ = saga.Add("remove-cart-item", req.Item, true)
	_ = saga.Add("refund-payment", req, true)

	// Example: call shipping provider using rea.RunDo (synchronous side-effect) to reserve slot
	shipResult, runErr := rea.RunDo(ctx, func(rc restate.RunContext) (string, error) {
		rc.Log().Info("Run: calling shipping provider (simulated)", "order", req.OrderID)
		// Simulate possible transient error: choose success
		return "SHIP-OK", nil
	}, restate.WithName("workflow.shipping.reserve"))

	if runErr != nil {
		err = fmt.Errorf("shipping reservation failed: %w", runErr)
		return "", err
	}
	ctx.Log().Info("OrderWorkflow: shipping reserved", "res", shipResult)

	// Use a promise: wait for internal verification
	emailPromise := rea.GetInternalSignal[bool](ctx, "email-verified")
	verified, pErr := emailPromise.Result()

	// When the asynchronous operation fails to produce a result due to a system
	// error, timeout, or other exceptional circumstance.
	if pErr != nil {
		err = pErr
		return "", err
	}

	if !verified {
		err = fmt.Errorf("email not verified") // A business logic error
		return "", err
	}

	// Finalize workflow
	return "workflow: completed", nil
}

type VerifyEmailRequest struct {
	Verified bool `json:"verified"`
}

// Workflow shared handler to resolve promise
func (w *OrderWorkflow) VerifyEmail(ctx restate.WorkflowSharedContext, req VerifyEmailRequest) (restate.Void, error) {
	p := rea.GetInternalSignal[bool](ctx, "email-verified") // workflow promise is name-based

	// Imagine we had some logic that could fail
	if req.Verified {
		return restate.Void{}, p.Resolve(true)
	} else {
		// Let's say a false result means the token was invalid/expired
		// This is a system failure, not a simple "not verified" state.
		return restate.Void{}, p.Reject(fmt.Errorf("verification token is invalid or expired"))
	}
}

// ApprovalService demonstrates Awakeable usage for human-in-the-loop approval
type ApprovalService struct{}

type ApprovalRequest struct {
	Id      string
	Approve bool
}

func (a *ApprovalService) RequestApproval(ctx restate.Context, action string) (string, error) {
	ctx.Log().Info("ApprovalService: creating awakeable for action", "action", action)
	aw := rea.WaitForExternalSignal[bool](ctx).Id()
	// Return ID to caller so they can give it to external system (UI)
	return aw, nil
}

func (a *ApprovalService) HandleApprovalCallback(ctx restate.Context, req ApprovalRequest) (restate.Void, error) {
	ctx.Log().Info("ApprovalService: resolving awakeable", "id", req.Id, "approve", req.Approve)
	rea.ResolveExternalSignal(ctx, req.Id, req.Approve)
	return restate.Void{}, nil
}

// -----------------------------------------------------------------------------
// CONTROL PLANE: OrderOrchestrator (Workflow)
// Handles orchestration, coordination, saga management
// -----------------------------------------------------------------------------

type OrderOrchestrator struct{}

// Run is the control plane handler that orchestrates the order processing flow
func (o *OrderOrchestrator) Run(ctx restate.WorkflowContext, req OrderRequest) (r OrderResponse, err error) {
	ctx.Log().Info("OrderOrchestrator: starting order flow", "order", req.OrderID)

	// Initialize saga for compensation
	saga := rea.NewSaga(ctx, "order-saga", nil)
	defer saga.CompensateIfNeeded(&err)

	// Register compensations for rollback scenarios
	saga.Register("refund-payment", func(rc restate.RunContext, payload []byte) error {
		var order OrderRequest
		json.Unmarshal(payload, &order)
		rc.Log().Warn("Saga: refunding payment", "order", order.OrderID)
		// In production: call PaymentService.Refund
		return nil
	})
	saga.Register("release-inventory", func(rc restate.RunContext, payload []byte) error {
		var item string
		json.Unmarshal(payload, &item)
		rc.Log().Warn("Saga: releasing inventory", "item", item)
		// In production: call InventoryService.Release
		return nil
	})

	// Step 1: Check inventory (data plane call)
	invClient := rea.ServiceClient[string, bool]{
		ServiceName: "InventoryService",
		HandlerName: "CheckAvailability",
	}
	available, err := invClient.Call(ctx, req.Item)
	if err != nil {
		return OrderResponse{req.OrderID, "FAILED", "inventory check failed"}, err
	}
	if !available {
		return OrderResponse{req.OrderID, "FAILED", "item unavailable"}, nil
	}
	_ = saga.Add("release-inventory", req.Item, true)

	// Step 2: Process payment (data plane call) - using future for async execution
	payClient := restate.Service[bool](ctx, "PaymentService", "ProcessPayment")
	payFuture := payClient.RequestFuture(req)

	// Step 3: Fire notification (data plane call - fire-and-forget)
	notifClient := rea.ServiceClient[map[string]any, restate.Void]{
		ServiceName: "NotificationService",
		HandlerName: "SendEmail",
	}
	_ = notifClient.Send(ctx, map[string]any{
		"to":      req.UserID,
		"message": fmt.Sprintf("Order %s received", req.OrderID),
	})

	// Step 4: Sync CRM (data plane call)
	crmClient := rea.ServiceClient[CRMRequest, restate.Void]{
		ServiceName: "CRMService",
		HandlerName: "SyncCustomer",
	}
	_, crmErr := crmClient.Call(ctx, CRMRequest{UserID: req.UserID})
	if crmErr != nil {
		ctx.Log().Warn("OrderOrchestrator: CRM sync failed (non-fatal)", "err", crmErr.Error())
	}

	// Step 5: Wait for payment result (control plane coordination)
	paymentOk, payErr := payFuture.Response()
	if payErr != nil {
		return OrderResponse{req.OrderID, "FAILED", "payment error"}, payErr
	}
	if !paymentOk {
		return OrderResponse{req.OrderID, "FAILED", "payment declined"}, nil
	}
	_ = saga.Add("refund-payment", req, true)

	// Step 6: Log analytics (data plane call)
	analyticsClient := rea.ServiceClient[map[string]any, restate.Void]{
		ServiceName: "AnalyticsService",
		HandlerName: "LogOrderEvent",
	}
	_, _ = analyticsClient.Call(ctx, map[string]any{
		"order_id": req.OrderID,
		"amount":   req.Amount,
		"user":     req.UserID,
	})

	// Step 7: Request shipment (data plane call)
	shipClient := rea.ServiceClient[string, string]{
		ServiceName: "ShippingService",
		HandlerName: "RequestShipment",
	}
	tracking, serr := shipClient.Call(ctx, req.OrderID)
	if serr != nil {
		return OrderResponse{req.OrderID, "FAILED", "shipment failed"}, serr
	}

	ctx.Log().Info("OrderOrchestrator: order completed", "order", req.OrderID, "tracking", tracking)
	return OrderResponse{req.OrderID, "COMPLETED", fmt.Sprintf("tracking=%s", tracking)}, nil
}

// -----------------------------------------------------------------------------
// DATA PLANE: OrderProcessorService (kept for backward compatibility)
// This is now a simple facade that delegates to the orchestrator
// -----------------------------------------------------------------------------

type OrderProcessorService struct{}

func (o *OrderProcessorService) ProcessOrder(ctx restate.Context, req OrderRequest) (OrderResponse, error) {
	// Delegate to the workflow orchestrator (control plane)
	// This shows how a service can trigger a workflow
	workflowClient := rea.WorkflowClient[OrderRequest, OrderResponse]{
		ServiceName: "OrderOrchestrator",
		HandlerName: "Run",
	}

	// Attach to the workflow (or start it if not exists)
	response, err := workflowClient.Attach(ctx, req.OrderID)
	if err != nil {
		ctx.Log().Error("OrderProcessor: orchestrator failed", "err", err.Error())
		return OrderResponse{req.OrderID, "FAILED", "orchestration error"}, err
	}

	return response, nil
}

// -----------------------------------------------------------------------------
// main: bind services and start Restate runtime (single-file runnable)
// -----------------------------------------------------------------------------
func main() {
	rt := server.NewRestate().
		Bind(restate.Reflect(&InventoryService{})).
		Bind(restate.Reflect(&PaymentService{})).
		Bind(restate.Reflect(&NotificationService{})).
		Bind(restate.Reflect(&ShippingService{})).
		Bind(restate.Reflect(&CRMService{})).
		Bind(restate.Reflect(&AnalyticsService{})).
		Bind(restate.Reflect(&OrderWorkflow{})).
		Bind(restate.Reflect(&ApprovalService{})).
		Bind(restate.Reflect(&OrderOrchestrator{})). // Control plane orchestrator
		Bind(restate.Reflect(&OrderProcessorService{}))

	log.Println("Starting Restate runtime (examples) on :2223")
	if err := rt.Start(context.Background(), ":2223"); err != nil {
		log.Fatalf("restate runtime: %v", err)
	}
}
