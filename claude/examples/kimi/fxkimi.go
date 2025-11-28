// fxkimi - Elegant Restate.dev Workflow Framework
// A clean, intuitive framework for building resilient workflows with mixed sync/async steps
package fxkimi

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/encoding"
	"github.com/restatedev/sdk-go/server"
)

// ===== Core Framework Types =====

// WorkflowInput represents the input to a workflow
type WorkflowInput struct {
	WorkflowName string                 `json:"workflow_name"`
	Data         map[string]interface{} `json:"data"`
}

// WorkflowOutput represents the output from a workflow
type WorkflowOutput struct {
	Result interface{} `json:"result"`
}

// WorkflowStatus represents the current status of a workflow
type WorkflowStatus struct {
	Input   WorkflowInput  `json:"input"`
	Results WorkflowOutput `json:"results"`
	Status  string         `json:"status"`
}

// Step represents a single workflow step that can be synchronous or asynchronous
type Step interface {
	// Execute runs the step and returns a future for async steps, or nil for sync steps
	Execute(ctx restate.Context) (restate.Future, error)
	// IsAsync returns true if this step should run asynchronously
	IsAsync() bool
	// Name returns the step name for logging and debugging
	Name() string
}

// StepResult represents the result of a workflow step
type StepResult struct {
	Data  interface{}
	Error error
}

// WorkflowBuilder provides a fluent API for building workflows
type WorkflowBuilder struct {
	name  string
	steps []Step
}

// WorkflowContext wraps Restate context with additional workflow-specific functionality
type WorkflowContext struct {
	ctx    restate.WorkflowContext
	logger *slog.Logger
}

// ===== Step Implementations =====

// SyncStep represents a synchronous workflow step
type SyncStep struct {
	name string
	fn   func(restate.Context) (interface{}, error)
}

func (s *SyncStep) Execute(ctx restate.Context) (restate.Future, error) {
	_, err := s.fn(ctx)
	return nil, err
}

func (s *SyncStep) IsAsync() bool { return false }
func (s *SyncStep) Name() string  { return s.name }

// AsyncStep represents an asynchronous workflow step
type AsyncStep struct {
	name string
	fn   func(restate.Context) (restate.Future, error)
}

func (s *AsyncStep) Execute(ctx restate.Context) (restate.Future, error) {
	return s.fn(ctx)
}

func (s *AsyncStep) IsAsync() bool { return true }
func (s *AsyncStep) Name() string  { return s.name }

// RunStep represents a durable side-effect step
type RunStep struct {
	name string
	fn   func(restate.RunContext) (interface{}, error)
}

func (s *RunStep) Execute(ctx restate.Context) (restate.Future, error) {
	_, err := restate.Run(ctx, s.fn)
	return nil, err
}

func (s *RunStep) IsAsync() bool { return false }
func (s *RunStep) Name() string  { return s.name }

// ===== Fluent Builder API =====

// NewWorkflow creates a new workflow builder
func NewWorkflow(name string) *WorkflowBuilder {
	return &WorkflowBuilder{
		name:  name,
		steps: make([]Step, 0),
	}
}

// Step adds a synchronous step to the workflow
func (w *WorkflowBuilder) Step(name string, fn func(restate.Context) (interface{}, error)) *WorkflowBuilder {
	w.steps = append(w.steps, &SyncStep{name: name, fn: fn})
	return w
}

// AsyncStep adds an asynchronous step to the workflow
func (w *WorkflowBuilder) AsyncStep(name string, fn func(restate.Context) (restate.Future, error)) *WorkflowBuilder {
	w.steps = append(w.steps, &AsyncStep{name: name, fn: fn})
	return w
}

// Run adds a durable side-effect step to the workflow
func (w *WorkflowBuilder) Run(name string, fn func(restate.RunContext) (interface{}, error)) *WorkflowBuilder {
	w.steps = append(w.steps, &RunStep{name: name, fn: fn})
	return w
}

// ServiceCall adds a service call step (async by default)
func (w *WorkflowBuilder) ServiceCall(name, service, method string, input interface{}) *WorkflowBuilder {
	return w.AsyncStep(name, func(ctx restate.Context) (restate.Future, error) {
		return restate.Service[interface{}](ctx, service, method).RequestFuture(input), nil
	})
}

// ObjectCall adds an object call step (async by default)
func (w *WorkflowBuilder) ObjectCall(name, service, key, method string, input interface{}) *WorkflowBuilder {
	return w.AsyncStep(name, func(ctx restate.Context) (restate.Future, error) {
		return restate.Object[interface{}](ctx, service, key, method).RequestFuture(input), nil
	})
}

// Sleep adds a sleep step
func (w *WorkflowBuilder) Sleep(name string, duration time.Duration) *WorkflowBuilder {
	return w.Step(name, func(ctx restate.Context) (interface{}, error) {
		return nil, restate.Sleep(ctx, duration)
	})
}

// WaitFirst adds a step that waits for the first of multiple futures to complete
func (w *WorkflowBuilder) WaitFirst(name string, futures ...restate.Future) *WorkflowBuilder {
	return w.Step(name, func(ctx restate.Context) (interface{}, error) {
		first, err := restate.WaitFirst(ctx, futures...)
		return first, err
	})
}

// WaitAll adds a step that waits for all futures to complete
func (w *WorkflowBuilder) WaitAll(name string, futures ...restate.Future) *WorkflowBuilder {
	return w.Step(name, func(ctx restate.Context) (interface{}, error) {
		results := make([]interface{}, 0, len(futures))
		for fut, err := range restate.Wait(ctx, futures...) {
			if err != nil {
				return nil, err
			}
			result, err := fut.(restate.ResponseFuture[interface{}]).Response()
			if err != nil {
				return nil, err
			}
			results = append(results, result)
		}
		return results, nil
	})
}

// Build creates the final workflow definition
func (w *WorkflowBuilder) Build() *WorkflowDefinition {
	return &WorkflowDefinition{
		name:  w.name,
		steps: w.steps,
	}
}

// ===== Workflow Definition and Execution =====

// WorkflowDefinition represents a complete workflow
type WorkflowDefinition struct {
	name  string
	steps []Step
}

// Name returns the workflow name
func (w *WorkflowDefinition) Name() string {
	return w.name
}

// Execute runs the workflow with the given context and input
func (w *WorkflowDefinition) Execute(ctx restate.WorkflowContext, input interface{}) (interface{}, error) {
	wfCtx := &WorkflowContext{
		ctx:    ctx,
		logger: slog.Default().With("workflow", w.name),
	}

	wfCtx.logger.Info("Starting workflow execution", "input", input)

	// Store input in workflow state
	restate.Set(ctx, "workflow_input", input)

	// Execute all steps
	results := make(map[string]interface{})
	asyncFutures := make(map[string]restate.Future)

	for i, step := range w.steps {
		stepName := step.Name()
		wfCtx.logger.Info("Executing step", "step", stepName, "index", i)

		if step.IsAsync() {
			// Execute async step and store future
			future, err := step.Execute(ctx)
			if err != nil {
				return nil, fmt.Errorf("step %s failed: %w", stepName, err)
			}
			asyncFutures[stepName] = future
			wfCtx.logger.Info("Async step started", "step", stepName)
		} else {
			// Execute sync step immediately
			_, err := step.Execute(ctx)
			if err != nil {
				return nil, fmt.Errorf("step %s failed: %w", stepName, err)
			}
			wfCtx.logger.Info("Sync step completed", "step", stepName)
		}
	}

	// Wait for all async steps to complete
	if len(asyncFutures) > 0 {
		wfCtx.logger.Info("Waiting for async steps to complete", "count", len(asyncFutures))

		futures := make([]restate.Future, 0, len(asyncFutures))
		for _, fut := range asyncFutures {
			futures = append(futures, fut)
		}

		for fut, err := range restate.Wait(ctx, futures...) {
			if err != nil {
				return nil, fmt.Errorf("async step failed: %w", err)
			}

			// Find which step this future belongs to
			for stepName, future := range asyncFutures {
				if future == fut {
					result, err := fut.(restate.ResponseFuture[interface{}]).Response()
					if err != nil {
						return nil, fmt.Errorf("step %s returned error: %w", stepName, err)
					}
					results[stepName] = result
					wfCtx.logger.Info("Async step completed", "step", stepName, "result", result)
					break
				}
			}
		}
	}

	// Store final results
	restate.Set(ctx, "workflow_results", results)

	wfCtx.logger.Info("Workflow completed successfully")
	return results, nil
}

// ===== Example Workflow Implementations =====

// OrderProcessingWorkflow demonstrates a complete order processing workflow
func OrderProcessingWorkflow() *WorkflowDefinition {
	return NewWorkflow("OrderProcessing").
		// Step 1: Validate order (sync)
		Run("validate_order", func(ctx restate.RunContext) (interface{}, error) {
			// Validate order data
			return "order_validated", nil
		}).
		// Step 2: Check inventory (async)
		ServiceCall("check_inventory", "InventoryService", "CheckAvailability", map[string]string{
			"product_id": "123",
			"quantity":   "1",
		}).
		// Step 3: Process payment (durable side-effect)
		Run("process_payment", func(ctx restate.RunContext) (interface{}, error) {
			// Process payment with external payment gateway
			return "payment_processed", nil
		}).
		// Step 4: Reserve inventory (async)
		ServiceCall("reserve_inventory", "InventoryService", "Reserve", map[string]string{
			"product_id": "123",
			"quantity":   "1",
		}).
		// Step 5: Create shipment (async)
		ServiceCall("create_shipment", "ShippingService", "CreateShipment", map[string]string{
			"order_id": "order_123",
		}).
		// Step 6: Send confirmation email (durable side-effect)
		Run("send_confirmation", func(ctx restate.RunContext) (interface{}, error) {
			// Send confirmation email
			return "email_sent", nil
		}).
		Build()
}

// UserSignupWorkflow demonstrates a user signup workflow with email verification
func UserSignupWorkflow() *WorkflowDefinition {
	return NewWorkflow("UserSignup").
		// Step 1: Create user account (durable)
		Run("create_account", func(ctx restate.RunContext) (interface{}, error) {
			// Create user in database
			return map[string]string{"user_id": "user_123", "status": "created"}, nil
		}).
		// Step 2: Generate verification token
		Step("generate_token", func(ctx restate.Context) (interface{}, error) {
			// Generate verification token
			return "verification_token_123", nil
		}).
		// Step 3: Send verification email (durable)
		Run("send_verification_email", func(ctx restate.RunContext) (interface{}, error) {
			// Send verification email
			return "email_sent", nil
		}).
		// Step 4: Wait for email verification (async - uses promise)
		AsyncStep("wait_verification", func(ctx restate.Context) (restate.Future, error) {
			return restate.Promise[string](ctx.(restate.WorkflowContext), "email_verified"), nil
		}).
		// Step 5: Activate user account
		Run("activate_account", func(ctx restate.RunContext) (interface{}, error) {
			// Activate user account
			return "account_activated", nil
		}).
		Build()
}

// ===== Restate Service Integration =====

// FxKimiService provides the main service interface for Restate
type FxKimiService struct {
	workflows map[string]*WorkflowDefinition
}

// NewFxKimiService creates a new service instance
func NewFxKimiService() *FxKimiService {
	return &FxKimiService{
		workflows: make(map[string]*WorkflowDefinition),
	}
}

// RegisterWorkflow registers a workflow with the service
func (s *FxKimiService) RegisterWorkflow(workflow *WorkflowDefinition) {
	s.workflows[workflow.name] = workflow
}

// Run executes a workflow
func (s *FxKimiService) Run(ctx restate.WorkflowContext, input WorkflowInput) (WorkflowOutput, error) {
	workflowName := input.WorkflowName
	workflow, ok := s.workflows[workflowName]
	if !ok {
		return WorkflowOutput{}, restate.TerminalError(fmt.Errorf("workflow %s not found", workflowName))
	}

	result, err := workflow.Execute(ctx, input.Data)
	if err != nil {
		return WorkflowOutput{}, err
	}

	return WorkflowOutput{
		Result: result,
	}, nil
}

// SignalEmailVerified handles the email verification signal
func (s *FxKimiService) SignalEmailVerified(ctx restate.WorkflowSharedContext, token string) error {
	return restate.Promise[string](ctx, "email_verified").Resolve(token)
}

// QueryStatus returns the current workflow status
func (s *FxKimiService) QueryStatus(ctx restate.WorkflowSharedContext, _ interface{}) (WorkflowStatus, error) {
	input, err := restate.Get[WorkflowInput](ctx, "workflow_input")
	if err != nil {
		return WorkflowStatus{}, err
	}

	results, err := restate.Get[WorkflowOutput](ctx, "workflow_results")
	if err != nil {
		return WorkflowStatus{}, err
	}

	return WorkflowStatus{
		Input:   input,
		Results: results,
		Status:  "completed",
	}, nil
}

// ===== Main Application =====

func main() {
	// Create the service
	service := NewFxKimiService()

	// Register workflows
	service.RegisterWorkflow(OrderProcessingWorkflow())
	service.RegisterWorkflow(UserSignupWorkflow())

	// Create and start the server
	// Disable JSON schema generation by providing a custom schema generator
	// that returns an empty schema for all types
	customCodec := encoding.JSONCodecWithCustomSchemaGenerator(func(v any) interface{} {
		// Return a generic empty object schema for all types
		// This prevents the nil pointer dereference by always returning a valid schema
		return map[string]interface{}{
			"type": "object",
		}
	})

	server := server.NewRestate().
		Bind(restate.Reflect(service,
			restate.WithPayloadCodec(customCodec)))

	slog.Info("Starting FxKimi workflow service on :9080")
	if err := server.Start(context.Background(), ":9080"); err != nil {
		slog.Error("application exited unexpectedly", "err", err.Error())
		os.Exit(1)
	}
}

/*
Usage Examples:

1. Start an order processing workflow:
curl -X POST localhost:8080/FxKimiService/workflow-id-123/Run \
  -H "Content-Type: application/json" \
  -d '{
    "workflow_name": "OrderProcessing",
    "order_id": "order_123",
    "product_id": "product_456",
    "customer_id": "customer_789"
  }'

2. Signal email verification:
curl -X POST localhost:8080/FxKimiService/workflow-id-123/SignalEmailVerified \
  -H "Content-Type: application/json" \
  -d '"verification_token_123"'

3. Query workflow status:
curl -X GET localhost:8080/FxKimiService/workflow-id-123/QueryStatus

4. Attach to workflow to get results:
curl -X GET localhost:8080/restate/workflow/FxKimiService/workflow-id-123/attach
*/
