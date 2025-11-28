// package restorm
package main

import (
	"context"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/server"
)

// ============================================================================
// CORE FRAMEWORK - Event-Driven Architecture inspired by Delphi 7
// ============================================================================

// Event represents a domain event with metadata
type Event struct {
	Name      string
	Data      interface{}
	Timestamp time.Time
	Source    string
}

// ============================================================================
// CONTEXT WRAPPERS - Type-safe context handling
// ============================================================================

// ServiceContext wraps restate.Context for services
type ServiceContext struct {
	restate.Context
	eventBus *EventBus
}

// ObjectContext wraps restate.ObjectContext for virtual objects
type ObjectContext struct {
	restate.ObjectContext
	eventBus *EventBus
}

// SharedContext wraps restate.ObjectSharedContext for concurrent handlers
type SharedContext struct {
	restate.ObjectSharedContext
	eventBus *EventBus
}

// Emit publishes an event to be handled asynchronously
func (sc *ServiceContext) Emit(eventName string, data interface{}) {
	event := Event{
		Name:      eventName,
		Data:      data,
		Timestamp: time.Now(),
		Source:    "service",
	}
	sc.eventBus.Dispatch(sc.Context, event)
}

func (oc *ObjectContext) Emit(eventName string, data interface{}) {
	event := Event{
		Name:      eventName,
		Data:      data,
		Timestamp: time.Now(),
		Source:    restate.Key(oc.ObjectContext),
	}
	oc.eventBus.Dispatch(oc.ObjectContext, event)
}

// ============================================================================
// STATE MANAGER - Elegant property-style state access
// ============================================================================

type StateManager struct {
	ctx restate.ObjectContext
	key string
}

func (oc *ObjectContext) State(key string) *StateManager {
	return &StateManager{ctx: oc.ObjectContext, key: key}
}

func (sm *StateManager) GetInt(defaultValue int) int {
	val, err := restate.Get[int](sm.ctx, sm.key)
	if err != nil || val == 0 {
		return defaultValue
	}
	return val
}

func (sm *StateManager) GetString(defaultValue string) string {
	val, err := restate.Get[string](sm.ctx, sm.key)
	if err != nil || val == "" {
		return defaultValue
	}
	return val
}

func (sm *StateManager) GetBool(defaultValue bool) bool {
	val, err := restate.Get[bool](sm.ctx, sm.key)
	if err != nil {
		return defaultValue
	}
	return val
}

func (sm *StateManager) SetInt(value int) {
	restate.Set(sm.ctx, sm.key, value)
}

func (sm *StateManager) SetString(value string) {
	restate.Set(sm.ctx, sm.key, value)
}

func (sm *StateManager) SetBool(value bool) {
	restate.Set(sm.ctx, sm.key, value)
}

func (sm *StateManager) Increment(delta int) int {
	current := sm.GetInt(0)
	newVal := current + delta
	sm.SetInt(newVal)
	return newVal
}

func (sm *StateManager) Decrement(delta int) int {
	return sm.Increment(-delta)
}

// ============================================================================
// EVENT BUS - Central event routing (like Delphi's message bus)
// ============================================================================

type EventHandler func(restate.Context, Event) error

type EventBus struct {
	handlers map[string][]EventHandler
}

func NewEventBus() *EventBus {
	return &EventBus{
		handlers: make(map[string][]EventHandler),
	}
}

func (eb *EventBus) On(eventName string, handler EventHandler) {
	eb.handlers[eventName] = append(eb.handlers[eventName], handler)
}

func (eb *EventBus) Dispatch(ctx restate.Context, event Event) {
	handlers, exists := eb.handlers[event.Name]
	if !exists {
		return
	}

	// Events are handled asynchronously (fire and forget)
	for _, handler := range handlers {
		go func(h EventHandler) {
			h(ctx, event)
		}(handler)
	}
}

// ============================================================================
// WORKFLOW - Delphi-style component for orchestrating multi-step processes
// ============================================================================

type WorkflowStep struct {
	Name        string
	Execute     func(*ServiceContext) error
	OnSuccess   func(*ServiceContext) error
	OnFailure   func(*ServiceContext, error) error
	Compensate  func(*ServiceContext) error
	RetryPolicy *RetryPolicy
}

type RetryPolicy struct {
	MaxAttempts int
	Delay       time.Duration
}

type Workflow struct {
	Name  string
	Steps []WorkflowStep
}

func (w *Workflow) Run(ctx *ServiceContext) error {
	completedSteps := []string{}

	for _, step := range w.Steps {
		ctx.Log().Info("Executing workflow step", "workflow", w.Name, "step", step.Name)

		var err error
		attempts := 1
		if step.RetryPolicy != nil {
			attempts = step.RetryPolicy.MaxAttempts
		}

		for i := 0; i < attempts; i++ {
			err = step.Execute(ctx)
			if err == nil {
				break
			}

			if step.RetryPolicy != nil && i < attempts-1 {
				time.Sleep(step.RetryPolicy.Delay) // Use standard sleep in workflow coordination
			}
		}

		if err != nil {
			ctx.Log().Error("Step failed", "step", step.Name, "error", err)

			if step.OnFailure != nil {
				step.OnFailure(ctx, err)
			}

			// Compensate completed steps in reverse (saga pattern)
			for i := len(completedSteps) - 1; i >= 0; i-- {
				compensateStep := w.Steps[i]
				if compensateStep.Compensate != nil {
					ctx.Log().Info("Compensating step", "step", compensateStep.Name)
					compensateStep.Compensate(ctx)
				}
			}

			return fmt.Errorf("workflow failed at step %s: %w", step.Name, err)
		}

		if step.OnSuccess != nil {
			step.OnSuccess(ctx)
		}

		completedSteps = append(completedSteps, step.Name)
	}

	ctx.Log().Info("Workflow completed successfully", "workflow", w.Name)
	return nil
}

// ============================================================================
// ENTITY - Virtual Object wrapper with event support (Delphi-style component)
// ============================================================================

type Entity struct {
	name     string
	onCreate func(*ObjectContext, string) error
	eventBus *EventBus
}

func NewEntity(name string) *Entity {
	return &Entity{
		name:     name,
		eventBus: NewEventBus(),
	}
}

func (e *Entity) OnCreate(handler func(*ObjectContext, string) error) *Entity {
	e.onCreate = handler
	return e
}

func (e *Entity) OnEvent(eventName string, handler EventHandler) *Entity {
	e.eventBus.On(eventName, handler)
	return e
}

// Build creates the underlying Restate virtual object
// Uses reflection-based approach from the SDK
func (e *Entity) Build() interface{} {
	// Return a struct that will be reflected by restate.Reflect()
	// The framework user will define actual handlers as methods on their entity struct
	return e
}

// ============================================================================
// SERVICE - Stateless service wrapper (Delphi-style component)
// ============================================================================

type Service struct {
	name     string
	eventBus *EventBus
}

func NewService(name string) *Service {
	return &Service{
		name:     name,
		eventBus: NewEventBus(),
	}
}

func (s *Service) OnEvent(eventName string, handler EventHandler) *Service {
	s.eventBus.On(eventName, handler)
	return s
}

// Build creates the underlying Restate service
func (s *Service) Build() interface{} {
	return s
}

// ============================================================================
// APPLICATION - Main application container (like Delphi's TApplication)
// ============================================================================

type Application struct {
	components []interface{}
	port       string
}

func NewApplication(port string) *Application {
	return &Application{
		port:       port,
		components: []interface{}{},
	}
}

func (app *Application) Register(component interface{}) {
	app.components = append(app.components, component)
}

func (app *Application) Run() error {
	r := server.NewRestate()

	for _, component := range app.components {
		r.Bind(restate.Reflect(component))
	}

	fmt.Printf("🚀 Application starting on %s\n", app.port)
	return r.Start(context.Background(), app.port)
}

// ============================================================================
// EXAMPLE USAGE - Demonstrates the elegant Delphi-style API
// ============================================================================

// OrderRequest represents an incoming order
type OrderRequest struct {
	ProductId string `json:"productId"`
	Quantity  int    `json:"quantity"`
	Customer  string `json:"customer"`
}

// OrderResponse represents the order processing result
type OrderResponse struct {
	OrderId string `json:"orderId"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// ============================================================================
// INVENTORY ENTITY - Virtual Object with state
// ============================================================================

type Inventory struct {
	*Entity
}

func NewInventory() *Inventory {
	entity := NewEntity("Inventory")
	inv := &Inventory{Entity: entity}

	// Configure initialization
	entity.OnCreate(func(ctx *ObjectContext, productId string) error {
		ctx.State("stock").SetInt(100)
		ctx.Log().Info("Inventory initialized", "product", productId, "stock", 100)
		return nil
	})

	return inv
}

// Reserve is an exclusive handler that reserves inventory
func (inv *Inventory) Reserve(ctx restate.ObjectContext, quantity int) (map[string]interface{}, error) {
	// Wrap context with our framework
	objCtx := &ObjectContext{ObjectContext: ctx, eventBus: inv.eventBus}

	// Auto-initialize on first call
	initialized := objCtx.State("_initialized").GetBool(false)
	if !initialized {
		if inv.onCreate != nil {
			inv.onCreate(objCtx, restate.Key(ctx))
		}
		objCtx.State("_initialized").SetBool(true)
	}

	stock := objCtx.State("stock")
	current := stock.GetInt(0)

	objCtx.Log().Info("Reserve request", "current", current, "requested", quantity)

	if current >= quantity {
		newStock := stock.Decrement(quantity)

		// Emit event for low stock warning
		if newStock < 20 {
			objCtx.Emit("LowStock", map[string]interface{}{
				"product":  restate.Key(ctx),
				"stock":    newStock,
				"quantity": quantity,
			})
		}

		return map[string]interface{}{
			"success":  true,
			"newStock": newStock,
			"message":  fmt.Sprintf("Reserved %d items", quantity),
		}, nil
	}

	return map[string]interface{}{
		"success": false,
		"message": fmt.Sprintf("Insufficient stock. Available: %d, Requested: %d", current, quantity),
	}, nil
}

// CheckStock is a shared handler (can run concurrently)
func (inv *Inventory) CheckStock(ctx restate.ObjectSharedContext) (int, error) {
	stock, err := restate.Get[int](ctx, "stock")
	if err != nil {
		return 100, nil // Default stock
	}
	return stock, nil
}

// ============================================================================
// ORDER PROCESSOR SERVICE - Orchestrates order workflow
// ============================================================================

type OrderProcessor struct {
	*Service
}

func NewOrderProcessor() *OrderProcessor {
	service := NewService("OrderProcessor")

	// Register event handlers
	service.OnEvent("PaymentSuccessful", func(ctx restate.Context, event Event) error {
		ctx.Log().Info("Payment successful event received", "order", event.Data)
		return nil
	})

	return &OrderProcessor{Service: service}
}

// ProcessOrder handles the main order processing workflow
func (op *OrderProcessor) ProcessOrder(ctx restate.Context, req OrderRequest) (OrderResponse, error) {
	// Wrap context with our framework
	svcCtx := &ServiceContext{Context: ctx, eventBus: op.eventBus}

	svcCtx.Log().Info("Processing order", "product", req.ProductId, "quantity", req.Quantity)

	// Create workflow with saga pattern
	workflow := &Workflow{
		Name: "OrderProcessing",
		Steps: []WorkflowStep{
			{
				Name: "ReserveInventory",
				Execute: func(ctx *ServiceContext) error {
					ctx.Log().Info("Reserving inventory", "product", req.ProductId)

					// Call inventory service using correct SDK pattern
					inventoryClient := restate.Object[map[string]interface{}](ctx.Context, "Inventory", req.ProductId, "Reserve")
					result, err := inventoryClient.Request(req.Quantity)
					if err != nil {
						return err
					}

					if !result["success"].(bool) {
						return fmt.Errorf("reservation failed: %s", result["message"])
					}

					ctx.Log().Info("Inventory reserved", "newStock", result["newStock"])
					return nil
				},
				Compensate: func(ctx *ServiceContext) error {
					ctx.Log().Info("Releasing inventory")
					// Would call inventory.Release() here
					return nil
				},
			},
			{
				Name: "ChargePayment",
				Execute: func(ctx *ServiceContext) error {
					ctx.Log().Info("Processing payment")
					// Simulate payment processing
					return nil
				},
				OnSuccess: func(ctx *ServiceContext) error {
					ctx.Emit("PaymentSuccessful", req)
					return nil
				},
				RetryPolicy: &RetryPolicy{
					MaxAttempts: 3,
					Delay:       1 * time.Second,
				},
			},
			{
				Name: "SendConfirmation",
				Execute: func(ctx *ServiceContext) error {
					ctx.Log().Info("Sending confirmation email", "customer", req.Customer)
					return nil
				},
			},
		},
	}

	err := workflow.Run(svcCtx)
	if err != nil {
		return OrderResponse{
			Status:  "FAILED",
			Message: err.Error(),
		}, err
	}

	return OrderResponse{
		OrderId: fmt.Sprintf("order-%d", time.Now().UnixNano()),
		Status:  "COMPLETED",
		Message: "Order processed successfully!",
	}, nil
}

// ============================================================================
// MAIN APPLICATION
// ============================================================================

// func ExampleUsage() {
func main() {
	// Create components
	inventory := NewInventory()
	orderProcessor := NewOrderProcessor()

	// Create and configure application
	app := NewApplication(":3333")
	app.Register(inventory)
	app.Register(orderProcessor)

	// Run application
	if err := app.Run(); err != nil {
		panic(err)
	}
}
