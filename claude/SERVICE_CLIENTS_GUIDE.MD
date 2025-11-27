# Service Type-Specific Clients - Complete Guide

## Overview

The framework now provides dedicated client types for each Restate service type, offering better type safety and clearer APIs compared to the generic `ServiceClient`. This ensures proper usage patterns and prevents common mistakes.

## Client Types

### 1. ServiceClient[I, O] - Basic Services
For stateless services that don't require keys.

### 2. ObjectClient[I, O] - Virtual Objects
For stateful services with key-based addressing.

### 3. WorkflowClient[I, O] - Workflows
For orchestration workflows with lifecycle management.

## ObjectClient - Virtual Objects

### Overview
Virtual Objects are stateful services where each instance is identified by a unique key. The `ObjectClient` ensures you always provide the required key parameter.

### Definition
```go
type ObjectClient[I, O any] struct {
    ServiceName string
    HandlerName string
}
```

### Methods

#### Call() - Request-Response
```go
func (c ObjectClient[I, O]) Call(
    ctx restate.Context,
    key string,          // Required: Object instance key
    input I,
    opts ...CallOption,
) (O, error)
```

#### Send() - One-Way (Fire-and-Forget)
```go
func (c ObjectClient[I, O]) Send(
    ctx restate.Context,
    key string,
    input I,
    opts ...CallOption,
) restate.Invocation
```

#### RequestFuture() - Concurrent Execution
```go
func (c ObjectClient[I, O]) RequestFuture(
    ctx restate.Context,
    key string,
    input I,
) restate.Future
```

### Usage Examples

#### Example 1: Simple Request-Response
```go
type UserService struct{}

func (UserService) UpdateProfile(ctx restate.Context, profile Profile) error {
    // Create client for calling another Virtual Object
    accountClient := ObjectClient[ChargeRequest, ChargeResult]{
        ServiceName: "BillingAccount",
        HandlerName: "Charge",
    }
    
    // Call with user's account key
    result, err := accountClient.Call(
        ctx,
        profile.UserID, // Object key
        ChargeRequest{Amount: 100},
    )
    if err != nil {
        return err
    }
    
    ctx.Log().Info("Charged account", "result", result)
    return nil
}
```

#### Example 2: One-Way Invocation
```go
func (OrderService) PlaceOrder(ctx restate.ObjectContext, order Order) error {
    // Send notification without waiting for response
    notificationClient := ObjectClient[Notification, restate.Void]{
        ServiceName: "NotificationService",
        HandlerName: "SendEmail",
    }
    
    notificationClient.Send(
        ctx,
        order.UserID, // User's notification object key
        Notification{
            Type:    "order_placed",
            OrderID: order.ID,
        },
    )
    
    return nil
}
```

#### Example 3: Concurrent Calls with RequestFuture
```go
func (OrderService) ProcessOrder(ctx restate.Context, order Order) error {
    inventoryClient := ObjectClient[CheckRequest, bool]{
        ServiceName: "Inventory",
        HandlerName: "Check",
    }
    
    // Start concurrent checks for multiple items
    var futures []restate.Future
    for _, item := range order.Items {
        fut := inventoryClient.RequestFuture(
            ctx,
            item.ProductID, // Each product is an object instance
            CheckRequest{Quantity: item.Quantity},
        )
        futures = append(futures, fut)
    }
    
    // Wait for all checks to complete
    for fut, err := range restate.Wait(ctx, futures...) {
        if err != nil {
            return fmt.Errorf("inventory check failed: %w", err)
        }
        
        // Extract result (type assertion needed with restate.Future)
        available, ok := fut.(bool)
        if !ok || !available {
            return restate.TerminalError(
                fmt.Errorf("insufficient inventory"),
                400,
            )
        }
    }
    
    return nil
}
```

#### Example 4: With Idempotency Keys
```go
func (PaymentService) Charge(ctx restate.Context, req ChargeRequest) error {
    accountClient := ObjectClient[DebitRequest, DebitResult]{
        ServiceName: "Account",
        HandlerName: "Debit",
    }
    
    // Generate deterministic idempotency key
    cp := NewControlPlaneService(ctx.(restate.WorkflowContext), "payment", "pay")
    idempKey := cp.GenerateIdempotencyKey(ctx, "debit")
    
    result, err := accountClient.Call(
        ctx,
        req.AccountID,
        DebitRequest{Amount: req.Amount},
        CallOption{IdempotencyKey: idempKey},
    )
    
    return err
}
```

---

## WorkflowClient - Workflows

### Overview
Workflows are long-running orchestrations with a unique workflow ID. The `WorkflowClient` provides methods tailored to workflow lifecycle management.

### Definition
```go
type WorkflowClient[I, O any] struct {
    ServiceName string
    HandlerName string // Usually "run" for main workflow handler
}
```

### Methods

#### Submit() - Start New Workflow
```go
func (c WorkflowClient[I, O]) Submit(
    ctx restate.Context,
    workflowID string,   // Required: Unique workflow instance ID
    input I,
    opts ...CallOption,
) restate.Invocation
```

#### Attach() - Attach to Running Workflow
```go
func (c WorkflowClient[I, O]) Attach(
    ctx restate.Context,
    workflowID string,
) (O, error)
```

#### AttachFuture() - Attach Concurrently
```go
func (c WorkflowClient[I, O]) AttachFuture(
    ctx restate.Context,
    workflowID string,
) restate.Future
```

#### Signal() - Send Signal to Workflow
```go
func (c WorkflowClient[I, O]) Signal(
    ctx restate.Context,
    workflowID string,
    signalHandler string, // Name of shared handler
    input I,
    opts ...CallOption,
) restate.Invocation
```

#### GetOutput() - Query Workflow State
```go
func (c WorkflowClient[I, O]) GetOutput(
    ctx restate.Context,
    workflowID string,
    outputHandler string, // Shared handler name
) (O, error)
```

### Usage Examples

#### Example 1: Submit New Workflow
```go
type OrderOrchestrator struct{}

func (OrderOrchestrator) CreateOrder(ctx restate.Context, req CreateOrderRequest) error {
    // Submit new workflow instance
    fulfillmentWorkflow := WorkflowClient[FulfillmentInput, FulfillmentResult]{
        ServiceName: "FulfillmentWorkflow",
        HandlerName: "run",
    }
    
    // Use order ID as workflow ID for idempotency
    workflowID := fmt.Sprintf("order-%s", req.OrderID)
    
    fulfillmentWorkflow.Submit(
        ctx,
        workflowID,
        FulfillmentInput{
            OrderID:  req.OrderID,
            Items:    req.Items,
            Address:  req.ShippingAddress,
        },
    )
    
    ctx.Log().Info("Fulfillment workflow started", "workflow_id", workflowID)
    return nil
}
```

#### Example 2: Attach to Existing Workflow
```go
func (OrderService) GetOrderStatus(ctx restate.Context, orderID string) (OrderStatus, error) {
    fulfillmentWorkflow := WorkflowClient[FulfillmentInput, FulfillmentResult]{
        ServiceName: "FulfillmentWorkflow",
        HandlerName: "run",
    }
    
    // Attach to the workflow and wait for completion
    workflowID := fmt.Sprintf("order-%s", orderID)
    result, err := fulfillmentWorkflow.Attach(ctx, workflowID)
    if err != nil {
        return OrderStatus{}, err
    }
    
    return OrderStatus{
        OrderID:     orderID,
        Status:      result.Status,
        TrackingNum: result.TrackingNumber,
    }, nil
}
```

#### Example 3: Signal Workflow (Human-in-the-Loop)
```go
func (ApprovalService) SubmitApproval(
    ctx restate.Context,
    workflowID string,
    approved bool,
) error {
    approvalWorkflow := WorkflowClient[ApprovalRequest, ApprovalResult]{
        ServiceName: "ApprovalWorkflow",
        HandlerName: "run",
    }
    
    // Send approval signal to workflow's shared handler
    approvalWorkflow.Signal(
        ctx,
        workflowID,
        "SubmitApproval", // Shared handler name
        approved,
    )
    
    ctx.Log().Info("Approval submitted", "workflow_id", workflowID, "approved", approved)
    return nil
}
```

#### Example 4: Query Workflow Status
```go
func (MonitoringService) CheckWorkflowProgress(
    ctx restate.Context,
    workflowID string,
) (WorkflowStatus, error) {
    dataProcessingWorkflow := WorkflowClient[ProcessingInput, ProcessingOutput]{
        ServiceName: "DataProcessingWorkflow",
        HandlerName: "run",
    }
    
    // Query status via shared handler
    status, err := dataProcessingWorkflow.GetOutput(
        ctx,
        workflowID,
        "GetStatus", // Shared handler that returns StatusData
    )
    if err != nil {
        return WorkflowStatus{}, err
    }
    
    return WorkflowStatus{
        WorkflowID: workflowID,
        Phase:      status.Phase,
        Progress:   status.Progress,
    }, nil
}
```

#### Example 5: Concurrent Workflow Attachments
```go
func (ReportService) GenerateReport(ctx restate.Context, reportReq ReportRequest) error {
    reportWorkflow := WorkflowClient[ReportInput, ReportOutput]{
        ServiceName: "ReportGenerationWorkflow",
        HandlerName: "run",
    }
    
    // Start multiple report generation workflows
    var futures []restate.Future
    for _, dataset := range reportReq.Datasets {
        workflowID := fmt.Sprintf("report-%s-%s", reportReq.ReportID, dataset.ID)
        
        // Submit workflow
        reportWorkflow.Submit(ctx, workflowID, ReportInput{
            Dataset: dataset,
            Format:  reportReq.Format,
        })
        
        // Attach for results
        fut := reportWorkflow.AttachFuture(ctx, workflowID)
        futures = append(futures, fut)
    }
    
    // Wait for all reports to complete
    var reports []ReportOutput
    for fut, err := range restate.Wait(ctx, futures...) {
        if err != nil {
            return fmt.Errorf("report generation failed: %w", err)
        }
        
        report, ok := fut.(ReportOutput)
        if ok {
            reports = append(reports, report)
        }
    }
    
    // Combine reports
    return combineReports(ctx, reports)
}
```

---

## Comparison: ServiceClient vs ObjectClient vs WorkflowClient

### ServiceClient - Stateless Services
```go
// No key required - service is stateless
client := ServiceClient[Input, Output]{
    ServiceName: "PaymentProcessor",
    HandlerName: "ProcessPayment",
}

result, err := client.Call(ctx, input)
```

### ObjectClient - Virtual Objects
```go
// Key required - identifies specific object instance
client := ObjectClient[Input, Output]{
    ServiceName: "UserAccount",
    HandlerName: "UpdateBalance",
}

result, err := client.Call(
    ctx,
    "user-123", // Required key
    input,
)
```

### WorkflowClient - Workflows
```go
// Workflow ID required - identifies workflow instance
client := WorkflowClient[Input, Output]{
    ServiceName: "OrderWorkflow",
    HandlerName: "run",
}

// Submit new workflow
client.Submit(ctx, "order-456", input)

// Or attach to existing
result, err := client.Attach(ctx, "order-456")
```

## Best Practices

### ✅ DO: Use Type-Specific Clients

```go
// ✓ Clear intent - this is a Virtual Object call
objectClient := ObjectClient[Request, Response]{
    ServiceName: "Account",
    HandlerName: "Debit",
}
objectClient.Call(ctx, accountKey, request)

// ✓ Clear intent - this is a workflow
workflowClient := WorkflowClient[Input, Output]{
    ServiceName: "OrderWorkflow",
    HandlerName: "run",
}
workflowClient.Submit(ctx, workflowID, input)
```

### ✅ DO: Use RequestFuture for Concurrent Calls

```go
// ✓ Concurrent execution
futures := make([]restate.Future, 0)
for _, userID := range userIDs {
    fut := userClient.RequestFuture(ctx, userID, request)
    futures = append(futures, fut)
}
// Wait for all
for fut, err := range restate.Wait(ctx, futures...) {
    // Process results
}
```

### ✅ DO: Use Workflow Submit vs Attach Correctly

```go
// ✓ Submit - Start new workflow (idempotent)
workflow.Submit(ctx, workflowID, input)

// ✓ Attach - Get result from existing/running workflow
result, _ := workflow.Attach(ctx, workflowID)
```

### ❌ DON'T: Mix Up Service Types

```go
// ✗ DON'T: Use ServiceClient for objects (missing key)
badClient := ServiceClient[Request, Response]{
    ServiceName: "Account", // This is a Virtual Object!
    HandlerName: "Debit",
}
badClient.Call(ctx, request) // ERROR: No key provided

// ✓ DO: Use ObjectClient
goodClient := ObjectClient[Request, Response]{
    ServiceName: "Account",
    HandlerName: "Debit",
}
goodClient.Call(ctx, accountKey, request)
```

### ❌ DON'T: Forget Idempotency for Important Operations

```go
// ✗ DON'T: No idempotency key for critical operations
workflow.Submit(ctx, workflowID, input) // What if this retries?

// ✓ DO: Add idempotency key
idempKey := cp.GenerateIdempotencyKey(ctx, "submit")
workflow.Submit(ctx, workflowID, input, CallOption{
    IdempotencyKey: idempKey,
})
```

## Advanced Patterns

### Pattern 1: Fan-Out/Fan-In with ObjectClient

```go
func processBatch(ctx restate.Context, userIDs []string) error {
    userClient := ObjectClient[UpdateRequest, UpdateResult]{
        ServiceName: "User",
        HandlerName: "Update",
    }
    
    // Fan-out: Start all updates concurrently
    futures := make([]restate.Future, 0, len(userIDs))
    for _, userID := range userIDs {
        fut := userClient.RequestFuture(ctx, userID, UpdateRequest{
            Action: "sync",
        })
        futures = append(futures, fut)
    }
    
    // Fan-in: Collect results
    results := make([]UpdateResult, 0, len(userIDs))
    for fut, err := range restate.Wait(ctx, futures...) {
        if err != nil {
            return err
        }
        if result, ok := fut.(UpdateResult); ok {
            results = append(results, result)
        }
    }
    
    return aggregateResults(ctx, results)
}
```

### Pattern 2: Workflow with Progress Tracking

```go
func startTrackedWorkflow(ctx restate.Context, taskID string) error {
    workflow := WorkflowClient[TaskInput, TaskOutput]{
        ServiceName: "TaskWorkflow",
        HandlerName: "run",
    }
    
    // Submit workflow
    workflowID := fmt.Sprintf("task-%s", taskID)
    workflow.Submit(ctx, workflowID, TaskInput{TaskID: taskID})
    
    // Poll for progress periodically
    timer := NewWorkflowTimer(ctx.(restate.WorkflowContext))
    
    for i := 0; i < 10; i++ {
        timer.Sleep(30 * time.Second)
        
        // Query progress
        status, err := workflow.GetOutput(ctx, workflowID, "GetStatus")
        if err != nil {
            continue
        }
        
        ctx.Log().Info("Workflow progress", "progress", status.Progress)
        
        if status.IsComplete {
            break
        }
    }
    
    // Attach for final result
    return workflow.Attach(ctx, workflowID)
}
```

### Pattern 3: Workflow Cancellation Signal

```go
func cancelWorkflow(ctx restate.Context, workflowID string, reason string) error {
    workflow := WorkflowClient[WorkflowInput, WorkflowOutput]{
        ServiceName: "LongRunningWorkflow",
        HandlerName: "run",
    }
    
    // Send cancellation signal
    workflow.Signal(
        ctx,
        workflowID,
        "Cancel", // Shared handler that sets a cancellation flag
        CancelRequest{Reason: reason},
    )
    
    ctx.Log().Info("Workflow cancellation requested", 
        "workflow_id", workflowID,
        "reason", reason)
    
    return nil
}
```

## Testing Examples

```go
func TestObjectClient_Call(t *testing.T) {
    ctx := mockContext()
    
    client := ObjectClient[string, string]{
        ServiceName: "Echo",
        HandlerName: "Echo",
    }
    
    result, err := client.Call(ctx, "instance-1", "hello")
    
    assert.NoError(t, err)
    assert.Equal(t, "hello", result)
}

func TestWorkflowClient_Submit(t *testing.T) {
    ctx := mockContext()
    
    client := WorkflowClient[WorkflowInput, WorkflowOutput]{
        ServiceName: "TestWorkflow",
        HandlerName: "run",
    }
    
    invocation := client.Submit(ctx, "wf-123", WorkflowInput{Data: "test"})
    
    assert.NotNil(t, invocation)
}

func TestObjectClient_RequestFuture(t *testing.T) {
    ctx := mockContext()
    
    client := ObjectClient[int, int]{
        ServiceName: "Math",
        HandlerName: "Square",
    }
    
    fut1 := client.RequestFuture(ctx, "key-1", 5)
    fut2 := client.RequestFuture(ctx, "key-2", 10)
    
    for fut, err := range restate.Wait(ctx, fut1, fut2) {
        assert.NoError(t, err)
        // Verify results
    }
}
```

## Migration from Generic ServiceClient

### Before (Generic)
```go
// Unclear what type of service this is
client := ServiceClient[Request, Response]{
    ServiceName: "Account",
    HandlerName: "Debit",
}
// Missing key parameter - will fail at runtime
result, err := client.Call(ctx, request)
```

### After (Type-Specific)
```go
// Clear this is a Virtual Object
client := ObjectClient[Request, Response]{
    ServiceName: "Account",
    HandlerName: "Debit",
}
// Key required at compile time - catches errors early
result, err := client.Call(ctx, accountKey, request)
```

## Conclusion

Service type-specific clients provide:

- ✅ **Better type safety** - Compiler enforces key/ID parameters
- ✅ **Clearer intent** - Code self-documents service types
- ✅ **Workflow lifecycle** - Submit, Attach, Signal methods
- ✅ **Fewer runtime errors** - Catch mistakes at compile time
- ✅ **Better IDE support** - Autocomplete knows required parameters
- ✅ **Automatic validation** - Idempotency keys validated on use

**Recommendation**: Use `ObjectClient` for Virtual Objects and `WorkflowClient` for Workflows. Reserve `ServiceClient` for stateless services only.
