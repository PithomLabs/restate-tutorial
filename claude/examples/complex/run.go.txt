// main.go
//
// Single-file example (Option 2): framework block first, then example services that
// demonstrate restate.Run, restate.RunVoid, restate.RunAsync, plus the 7 patterns.
// This file intentionally follows the DOs & DON'Ts from your framework and Restate docs.
//
// NOTE: This code assumes github.com/restatedev/sdk-go and its server package are available.
// It's illustrative and aligned with the framework you provided.

package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"path"
	"sort"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/server"
)

// ============================================================================
// FRAMEWORK BLOCK (copied / adapted from user's provided framework)
// This block provides:
//  - WaitForExternalSignal / ResolveExternalSignal / RejectExternalSignal
//  - GetInternalSignal (promises)
//  - State[T] wrapper (enforces write only from writable contexts)
//  - Saga framework: SagaFramework, Register, Add, CompensateIfNeeded, etc.
// ============================================================================

// WaitForExternalSignal creates a durable 'awakeable' that can be resolved
// or rejected by an external system (e.g., a human-in-the-loop UI, a
// payment gateway callback).
func WaitForExternalSignal[T any](ctx restate.Context) restate.AwakeableFuture[T] {
	ctx.Log().Info("framework: waiting for external signal (awakeable)")
	return restate.Awakeable[T](ctx)
}

// ResolveExternalSignal resolves an 'awakeable' (created by
// WaitForExternalSignal) from another service.
func ResolveExternalSignal[T any](ctx restate.Context, id string, value T) {
	ctx.Log().Info("framework: resolving external signal", "awakeable_id", id)
	restate.ResolveAwakeable[T](ctx, id, value)
}

// RejectExternalSignal rejects an 'awakeable' (created by
// WaitForExternalSignal) from another service.
func RejectExternalSignal(ctx restate.Context, id string, reason error) {
	ctx.Log().Error("framework: rejecting external signal", "awakeable_id", id, "reason", reason.Error())
	restate.RejectAwakeable(ctx, id, reason)
}

// GetInternalSignal gets a handle to a durable 'promise' used for
// coordination *within* a Restate workflow.
func GetInternalSignal[T any](ctx restate.WorkflowSharedContext, name string) restate.DurablePromise[T] {
	ctx.Log().Info("framework: getting internal signal (promise)", "name", name)
	return restate.Promise[T](ctx, name)
}

// State[T] wrapper enforcing Set/Clear only allowed from mutable contexts.
type State[T any] struct {
	ctx restate.ObjectSharedContext
	key string
}

func NewState[T any](ctx restate.ObjectSharedContext, key string) *State[T] {
	return &State[T]{ctx: ctx, key: key}
}

func (s *State[T]) Get() (T, error) {
	return restate.Get[T](s.ctx, s.key)
}

func (s *State[T]) Set(value T) error {
	// Try casting to ObjectContext for write capability
	if writerCtx, ok := any(s.ctx).(restate.ObjectContext); ok {
		restate.Set(writerCtx, s.key, value)
		return nil
	}
	// Try WorkflowContext as well (some states might be workflow-scoped)
	if wctx, ok := any(s.ctx).(restate.WorkflowContext); ok {
		restate.Set(wctx, s.key, value)
		return nil
	}
	err := fmt.Errorf("framework: Set(key=%s) called from read-only context", s.key)
	// Terminal error because this is a programming error (do/don't)
	s.ctx.Log().Error(err.Error())
	return restate.TerminalError(err)
}

func (s *State[T]) Clear() error {
	if writerCtx, ok := any(s.ctx).(restate.ObjectContext); ok {
		restate.Clear(writerCtx, s.key)
		return nil
	}
	if wctx, ok := any(s.ctx).(restate.WorkflowContext); ok {
		restate.Clear(wctx, s.key)
		return nil
	}
	err := fmt.Errorf("framework: Clear(key=%s) called from read-only context", s.key)
	s.ctx.Log().Error(err.Error())
	return restate.TerminalError(err)
}

// ============================================================================
// Saga framework (abridged & faithful to user's posted implementation)
// ============================================================================

type SagaCompensationFunc func(rc restate.RunContext, payload []byte) error

type SagaEntry struct {
	Name      string    `json:"name"`
	Payload   []byte    `json:"payload"`
	StepID    string    `json:"step_id"`
	Timestamp time.Time `json:"timestamp"`
	Attempt   int       `json:"attempt"`
}

type SagaFrameworkConfig struct {
	MaxCompensationRetries int
	InitialRetryDelay      time.Duration
	MaxRetryDelay          time.Duration
	FailOnCleanupError     bool
	EscalationDLQKey       string
}

func DefaultSagaConfig() SagaFrameworkConfig {
	return SagaFrameworkConfig{
		MaxCompensationRetries: 5,
		InitialRetryDelay:      1 * time.Second,
		MaxRetryDelay:          5 * time.Minute,
		FailOnCleanupError:     false,
		EscalationDLQKey:       "",
	}
}

type SagaFramework struct {
	wctx              restate.WorkflowContext
	nsKey             string
	escalatedSagasKey string
	registry          map[string]SagaCompensationFunc
	log               *slog.Logger
	cfg               SagaFrameworkConfig
}

func NewSaga(ctx restate.WorkflowContext, name string, cfg *SagaFrameworkConfig) *SagaFramework {
	instance := restate.Key(ctx)
	ns := path.Join(instance, "saga", name)
	dlq := path.Join(instance, "saga-dlq", name)
	if cfg == nil {
		c := DefaultSagaConfig()
		cfg = &c
	}
	logger := ctx.Log()
	if cfg.EscalationDLQKey != "" {
		dlq = cfg.EscalationDLQKey
	}
	return &SagaFramework{
		wctx:              ctx,
		nsKey:             ns,
		escalatedSagasKey: dlq,
		registry:          make(map[string]SagaCompensationFunc),
		log:               logger,
		cfg:               *cfg,
	}
}

func (s *SagaFramework) Register(name string, fn SagaCompensationFunc) {
	if fn == nil {
		return
	}
	s.registry[name] = fn
}

func (s *SagaFramework) Add(name string, payload any, dedupe bool) error {
	raw, err := canonicalJSON(payload)
	if err != nil {
		return fmt.Errorf("saga: marshal payload: %w", err)
	}
	stepID := deterministicStepID(name, raw)

	entries, _ := restate.Get[[]SagaEntry](s.wctx, s.nsKey)
	if dedupe {
		for _, e := range entries {
			if e.StepID == stepID {
				return nil
			}
		}
	}
	entry := SagaEntry{
		Name:      name,
		Payload:   raw,
		StepID:    stepID,
		Timestamp: time.Now(),
		Attempt:   0,
	}
	entries = append(entries, entry)
	restate.Set(s.wctx, s.nsKey, entries)
	s.log.Info("saga.added", "step", name, "step_id", stepID)
	return nil
}

func (s *SagaFramework) CompensateIfNeeded(errPtr *error) {
	if errPtr == nil || *errPtr == nil {
		return
	}
	origErr := *errPtr
	entries, _ := restate.Get[[]SagaEntry](s.wctx, s.nsKey)
	if len(entries) == 0 {
		s.log.Info("saga.no_compensations")
		return
	}
	total := len(entries)
	for idx := total - 1; idx >= 0; idx-- {
		entry := entries[idx]
		handler, ok := s.registry[entry.Name]
		if !ok || handler == nil {
			msg := fmt.Sprintf("missing compensation handler: %s", entry.Name)
			s.log.Error("saga.missing_handler", "name", entry.Name, "idx", idx)
			s.persistDLQ(origErr, fmt.Errorf(msg), entries, idx)
			*errPtr = restate.TerminalError(fmt.Errorf("%s: original=%w", msg, origErr))
			return
		}
		for {
			ents, _ := restate.Get[[]SagaEntry](s.wctx, s.nsKey)
			if idx >= len(ents) {
				msg := fmt.Errorf("compensation index out of range: idx=%d len=%d", idx, len(ents))
				s.log.Error("saga.index_oob", "idx", idx, "len", len(ents))
				s.persistDLQ(origErr, msg, entries, idx)
				*errPtr = restate.TerminalError(fmt.Errorf("compensation state corrupted: %w", origErr))
				return
			}
			cur := ents[idx]
			_, runErr := restate.Run(s.wctx, func(rc restate.RunContext) (restate.Void, error) {
				return restate.Void{}, handler(rc, cur.Payload)
			}, restate.WithName(fmt.Sprintf("saga.compensate.%s.%d", cur.Name, idx)))
			if runErr == nil {
				ents = removeIndex(ents, idx)
				restate.Set(s.wctx, s.nsKey, ents)
				s.log.Info("saga.compensation.succeeded", "name", cur.Name, "idx", idx)
				break
			}
			cur.Attempt++
			ents[idx] = cur
			restate.Set(s.wctx, s.nsKey, ents)
			s.log.Warn("saga.compensation.failed", "name", cur.Name, "idx", idx, "attempt", cur.Attempt, "err", runErr.Error())
			if s.cfg.MaxCompensationRetries >= 0 && cur.Attempt >= s.cfg.MaxCompensationRetries {
				msg := fmt.Errorf("max retries exceeded for %s (attempts=%d): last_err=%w", cur.Name, cur.Attempt, runErr)
				s.log.Error("saga.compensation.max_retries", "name", cur.Name, "attempts", cur.Attempt)
				s.persistDLQ(origErr, msg, ents, idx)
				*errPtr = restate.TerminalError(fmt.Errorf("compensation failed irrecoverably: %w", origErr))
				return
			}
			delay := computeBackoff(s.cfg.InitialRetryDelay, s.cfg.MaxRetryDelay, cur.Attempt-1)
			s.log.Info("saga.compensation.retry_scheduled", "name", cur.Name, "attempt", cur.Attempt, "delay", delay.String())
			if sleepErr := restate.Sleep(s.wctx, delay); sleepErr != nil {
				s.log.Error("saga.sleep_failed", "err", sleepErr.Error())
				s.persistDLQ(origErr, sleepErr, ents, idx)
				*errPtr = restate.TerminalError(fmt.Errorf("saga sleep failed: %w", origErr))
				return
			}
		}
	}
	restate.Clear(s.wctx, s.nsKey)
	s.log.Info("saga.compensations_completed")
}

func (s *SagaFramework) persistDLQ(originalErr, escalationErr error, entries []SagaEntry, cursor int) {
	_, _ = restate.Run(s.wctx, func(rc restate.RunContext) (restate.Void, error) {
		record := map[string]any{
			"saga_key":         s.nsKey,
			"original_error":   originalErr.Error(),
			"escalation_error": escalationErr.Error(),
			"entries":          entries,
			"cursor":           cursor,
			"timestamp":        time.Now(),
			"manual_fix":       true,
		}
		restate.Set(s.wctx, s.escalatedSagasKey, record)
		return restate.Void{}, nil
	}, restate.WithName("saga.persist_dlq"))
}

func computeBackoff(initial, max time.Duration, attempt int) time.Duration {
	if initial <= 0 {
		initial = time.Second
	}
	if max <= 0 {
		max = 5 * time.Minute
	}
	d := initial * (1 << uint(attempt))
	if d > max {
		return max
	}
	return d
}

func deterministicStepID(name string, payload []byte) string {
	h := sha256.Sum256(append([]byte(name+":"), payload...))
	return fmt.Sprintf("%x", h[:])
}

func canonicalJSON(v any) ([]byte, error) {
	var iv any
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &iv); err != nil {
		return nil, err
	}
	canon := canonicalizeValue(iv)
	return json.Marshal(canon)
}

func canonicalizeValue(v any) any {
	switch vv := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(vv))
		for k := range vv {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		m := make(map[string]any, len(keys))
		for _, k := range keys {
			m[k] = canonicalizeValue(vv[k])
		}
		return m
	case []any:
		out := make([]any, len(vv))
		for i := range vv {
			out[i] = canonicalizeValue(vv[i])
		}
		return out
	default:
		return vv
	}
}

func removeIndex[T any](s []T, i int) []T {
	if i < 0 || i >= len(s) {
		return s
	}
	return append(s[:i], s[i+1:]...)
}

// ============================================================================
// END FRAMEWORK BLOCK
// ============================================================================

// -----------------------------------------------------------------------------
// Now the example services and workflows that use the framework and demonstrate
// restate.Run, restate.RunVoid, restate.RunAsync, and all patterns.
// -----------------------------------------------------------------------------

// Types used across services
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

// Payment service: uses a mix of RunAsync to call external payment gateway simulation
type PaymentService struct{}

// ProcessPayment is a durable handler that simulates invoking a real payment gateway.
// We implement it using restate.Run (to record result) inside the service itself
// to show usage (a service may need to call external system).
func (p *PaymentService) ProcessPayment(ctx restate.Context, req OrderRequest) (bool, error) {
	ctx.Log().Info("PaymentService: initiating payment", "order", req.OrderID, "amount", req.Amount)

	// Use restate.Run to perform the external HTTP call (simulated). This ensures
	// the result is recorded deterministically and replay-safe.
	res, err := restate.Run(ctx, func(rc restate.RunContext) (string, error) {
		// This closure runs as a single durable side-effect step.
		// Simulate non-deterministic external call:
		rc.Log().Info("Run: contacting external payment gateway (simulated)", "order", req.OrderID)

		// Simulated behavior:
		// - if amount > 1000 -> fail payment
		// - else succeed
		if req.Amount > 1000 {
			return "DECLINED", nil
		}
		// simulate latency and external state change (but do not actually perform)
		//time.Sleep(200 * time.Millisecond)
		return "APPROVED", nil
	}, restate.WithName("payment.gateway.call"))
	if err != nil {
		ctx.Log().Error("PaymentService: run failed", "err", err.Error())
		return false, err
	}

	ctx.Log().Info("PaymentService: gateway result", "result", res)
	return res == "APPROVED", nil
}

// AnalyticsService demonstrates using restate.Run (sync) to send a final order event.
type AnalyticsService struct{}

func (a *AnalyticsService) LogOrderEvent(ctx restate.Context, event map[string]any) (restate.Void, error) {
	ctx.Log().Info("AnalyticsService: logging order event")

	_, err := restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
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
		//time.Sleep(250 * time.Millisecond)
		err := restate.Sleep(ctx, 250*time.Millisecond)
		if err != nil {
			return "", err
		}
		tracking := "TRACK-" + orderID
		rc.Log().Info("RunAsync: provider returned tracking", "tracking", tracking)
		return tracking, nil
	}, restate.WithName("shipping.request"))

	// Optionally, we can wait for result later or return invocation meta. Here we wait deterministically.
	tracking, err := fut.Result()
	if err != nil {
		ctx.Log().Error("ShippingService: shipping request failed", "err", err.Error())
		return "", err
	}
	ctx.Log().Info("ShippingService: got tracking", "tracking", tracking)
	return tracking, nil
}

// CRMService demonstrates RunVoid for a side-effect with no return.
type CRMService struct{}

type CRMRequest struct {
	UserID string
}

func (c *CRMService) SyncCustomer(ctx restate.Context, req CRMRequest) (restate.Void, error) {
	ctx.Log().Info("CRMService: sync customer", "user", req.UserID)

	_, err := restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
		rc.Log().Info("Run: calling CRM provider (simulated)", "user", req.UserID)
		//time.Sleep(100 * time.Millisecond)
		err := restate.Sleep(ctx, 100*time.Millisecond)
		if err != nil {
			return restate.Void{}, err
		}
		return restate.Void{}, nil
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

// For the workflow we must import both WorkflowContext and WorkflowSharedContext handlers.
// 'Run' is the main exclusive handler.
func (w *OrderWorkflow) Run(ctx restate.WorkflowContext, req OrderRequest) (string, error) {
	var err error
	// Build a saga to allow compensation if downstream fails
	saga := NewSaga(ctx, "order-saga", nil)
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

	// Example: call shipping provider using Run (synchronous side-effect) to reserve slot
	shipResult, runErr := restate.Run(ctx, func(rc restate.RunContext) (string, error) {
		rc.Log().Info("Run: calling shipping provider (simulated)", "order", req.OrderID)
		// Simulate possible transient error: choose success
		//time.Sleep(100 * time.Millisecond)
		// Sleep
		err := restate.Sleep(ctx, 100*time.Millisecond)
		if err != nil {
			return "", err
		}
		return "SHIP-OK", nil
	}, restate.WithName("workflow.shipping.reserve"))
	if runErr != nil {
		err = fmt.Errorf("shipping reservation failed: %w", runErr)
		return "", err
	}
	ctx.Log().Info("OrderWorkflow: shipping reserved", "res", shipResult)

	// Use a promise: wait for internal verification
	emailPromise := GetInternalSignal[bool](ctx, "email-verified")
	verified, pErr := emailPromise.Result()

	//when the asynchronous operation fails to produce a result due to a system
	//error, timeout, or other exceptional circumstance.
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

/*
func (w *OrderWorkflow) VerifyEmail(ctx restate.WorkflowSharedContext, v bool) (restate.Void, error) {
	p := GetInternalSignal[bool](ctx, "email-verified")
	return restate.Void{}, p.Resolve(v)
}
*/

type VerifyEmailRequest struct {
	Verified bool `json:"verified"`
}

// Workflow shared handler to resolve promise
func (w *OrderWorkflow) VerifyEmail(ctx restate.WorkflowSharedContext, req VerifyEmailRequest) (restate.Void, error) {
	p := GetInternalSignal[bool](ctx, "email-verified") //workflow promise is name-based

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
	aw := WaitForExternalSignal[bool](ctx).Id()
	// Return ID to caller so they can give it to external system (UI)
	return aw, nil
}

func (a *ApprovalService) HandleApprovalCallback(ctx restate.Context, req ApprovalRequest) (restate.Void, error) {
	ctx.Log().Info("ApprovalService: resolving awakeable", "id", req.Id, "approve", req.Approve)
	ResolveExternalSignal(ctx, req.Id, req.Approve)
	return restate.Void{}, nil
}

// OrderProcessorService ties all pieces together and demonstrates Run/RunVoid/RunAsync usage.
type OrderProcessorService struct{}

func (o *OrderProcessorService) ProcessOrder(ctx restate.Context, req OrderRequest) (OrderResponse, error) {
	ctx.Log().Info("OrderProcessor: start", "order", req.OrderID)

	// 1) Synchronous inter-service call: check inventory (Request)
	invClient := restate.Service[bool](ctx, "InventoryService", "CheckAvailability")
	available, err := invClient.Request(req.Item)
	if err != nil {
		ctx.Log().Error("OrderProcessor: inventory call failed", "err", err.Error())
		return OrderResponse{req.OrderID, "FAILED", "inventory error"}, nil
	}
	if !available {
		ctx.Log().Info("OrderProcessor: item not available", "item", req.Item)
		return OrderResponse{req.OrderID, "FAILED", "item unavailable"}, nil
	}

	// 2) Asynchronous payment processing: RequestFuture + continue (RequestFuture)
	payClient := restate.Service[bool](ctx, "PaymentService", "ProcessPayment")
	payFuture := payClient.RequestFuture(req) // returns future immediately

	// 3) Fire-and-forget notification: ServiceSend.Send
	notifSend := restate.ServiceSend(ctx, "NotificationService", "SendEmail")
	_ = notifSend.Send(map[string]any{
		"to":      req.UserID,
		"message": fmt.Sprintf("Order %s received", req.OrderID),
	})

	// 4) While payment is processing, perform CRM sync as a side-effect using RunVoid
	_, crmErr := restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
		rc.Log().Info("Run: syncing CRM for user", "user", req.UserID)
		//time.Sleep(80 * time.Millisecond)
		err := restate.Sleep(ctx, 100*time.Millisecond)
		if err != nil {
			return restate.Void{}, err
		}
		return restate.Void{}, nil
	}, restate.WithName("crm.sync"))

	if crmErr != nil {
		ctx.Log().Warn("OrderProcessor: crm sync failed (non-fatal)", "err", crmErr.Error())
		// non-fatal; continue — but if you want to fail, return an error.
	}

	// 5) Wait for payment result deterministically
	paymentOk, payErr := payFuture.Response()
	if payErr != nil {
		ctx.Log().Error("OrderProcessor: payment failed", "err", payErr.Error())
		return OrderResponse{req.OrderID, "FAILED", "payment error"}, nil
	}
	if !paymentOk {
		ctx.Log().Info("OrderProcessor: payment declined", "order", req.OrderID)
		return OrderResponse{req.OrderID, "FAILED", "payment declined"}, nil
	}

	// 6) After successful payment, perform analytics logging using restate.Run (synchronous record)
	analyticsClient := restate.Service[restate.Void](ctx, "AnalyticsService", "LogOrderEvent")
	_, aerr := analyticsClient.Request(map[string]any{
		"order_id": req.OrderID,
		"amount":   req.Amount,
		"user":     req.UserID,
	})
	if aerr != nil {
		// Analytics failure should not break order processing in this example; log and continue.
		ctx.Log().Warn("OrderProcessor: analytics logging failed", "err", aerr.Error())
	}

	// 7) Request shipment using ShippingService which uses RunAsync internally
	shipClient := restate.Service[string](ctx, "ShippingService", "RequestShipment")
	tracking, serr := shipClient.Request(req.OrderID)
	if serr != nil {
		ctx.Log().Error("OrderProcessor: shipment request failed", "err", serr.Error())
		// Try to continue; in production you might register saga steps to compensate later
		return OrderResponse{req.OrderID, "FAILED", "shipment failed"}, nil
	}

	// 8) Optionally start a workflow to coordinate long-running fulfillment + promise resolution
	workflowClient := restate.Workflow[string](ctx, "OrderWorkflow", req.OrderID, "Run")
	wres, werr := workflowClient.Request(req)
	if werr != nil {
		ctx.Log().Error("OrderProcessor: workflow start failed", "err", werr.Error())
		// record but continue
	}

	ctx.Log().Info("OrderProcessor: completed", "order", req.OrderID, "tracking", tracking, "workflow", wres)
	return OrderResponse{req.OrderID, "COMPLETED", fmt.Sprintf("tracking=%s workflow=%s", tracking, wres)}, nil
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
		Bind(restate.Reflect(&OrderProcessorService{}))

	log.Println("Starting Restate runtime (examples) on :2223")
	if err := rt.Start(context.Background(), ":2223"); err != nil {
		log.Fatalf("restateruntime: %v", err)
	}
}
