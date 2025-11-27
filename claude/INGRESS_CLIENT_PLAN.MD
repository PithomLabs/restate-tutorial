# Ingress Client Implementation Plan

## Objective

Implement the framework's `IngressClient` wrappers using the actual `github.com/restatedev/sdk-go/ingress` package, with support for Attach semantics and comprehensive examples.

## Background

**Current State**: The framework has stub implementations of `IngressClient`, `IngressServiceClient`, `IngressObjectClient`, and `IngressWorkflowClient` that return "requires restateingress package" errors.

**Gap**: External applications cannot actually use these clients to call Restate handlers.

**Recommendation (from DOS guidance)**: 
-Implement the wrapper using the `ingress` package from the SDK
- Include examples for Attach / AttachFuture semantics
- **Critical**: Attach only works if an idempotency key was used
- Add tests demonstrating external callers using Attach

## Proposed Changes

### 1. Update Imports

Add `github.com/restatedev/sdk-go/ingress` to the framework's imports.

### 2. Update `IngressClient` Structure

```go
type IngressClient struct {
    client *ingress.Client  // Actual SDK client
    log    *slog.Logger
}
```

### 3. Implement Service Client Methods

#### `IngressServiceClient.Call()`
- Use `ingress.Service[I, O](client, serviceName, handlerName).Request(ctx, input, opts...)`
- Apply ValidationMode from `IngressCallOption`
- Return actual result or error

#### `IngressServiceClient.Send()`
- Use `ingress.ServiceSend[I](client, serviceName, handlerName).Send(ctx, input, opts...)`
- Apply ValidationMode
- Return invocation ID

#### `IngressServiceClient.AttachByIdempotencyKey()`
**NEW METHOD** - Attach to existing invocation by idempotency key:
- Use `ingress.ServiceInvocationByIdempotencyKey[O](client, serviceName, handlerName, idempotencyKey).Attach(ctx)`
- Only works if invocation was started with idempotency key
- Returns result when invocation completes

### 4. Implement Object Client Methods

#### `IngressObjectClient.Call()` / `Send()`
- Use `ingress.Object[I, O](...)` and `ingress.ObjectSend[I](...)`

#### `IngressObjectClient.AttachByIdempotencyKey()`
-Use `ingress.ObjectInvocationByIdempotencyKey[O](...).Attach(ctx)`

### 5. Implement Workflow Client Methods

#### `IngressWorkflowClient.Submit()`
- Use `ingress.WorkflowSend[I](client, serviceName, workflowID, handlerName).Send(ctx, input, opts...)`

#### `IngressWorkflowClient.Attach()`
- Use `ingress.WorkflowHandle[O](client, serviceName, workflowID).Attach(ctx)`
- Waits for workflow run handler to complete

#### `IngressWorkflowClient.GetOutput()`
- Use `ingress.Workflow[encoding.Void, O](client, serviceName, workflowID, outputHandler).Request(ctx, encoding.Void{}, opts...)`

### 6. Add IngressCallOption.ValidationMode

Update `IngressCallOption` to include `ValidationMode` field (consistent with `CallOption`):

```go
type IngressCallOption struct {
    IdempotencyKey string
    Delay          time.Duration
    ValidationMode IdempotencyValidationMode  // NEW
}
```

### 7. Apply Validation in Ingress Methods

Similar to internal clients, apply configurable validation:
- `IdempotencyValidationWarn`: Log warning, proceed
- `IdempotencyValidationFail`: Return error
- `IdempotencyValidationDisabled`: Skip validation

### 8. Helper Methods for Attach Patterns

Add convenience methods to clarify attach usage:

```go
// AttachToServiceInvocation attaches to a service invocation by idempotency key
func (c *IngressClient) AttachToServiceInvocation[O any](
    ctx context.Context,
    serviceName string,
    handlerName string,
    idempotencyKey string,
) (O, error)

// AttachToWorkflow attaches to a workflow and waits for completion
func (c *IngressClient)AttachToWorkflow[O any](
    ctx context.Context,
    serviceName string,
    workflowID string,
) (O, error)
```

## Examples to Create

### Example 1: External Service Invocation with Attach

```go
// external_caller_example.go
func main() {
    // Create ingress client for external application
    ingressClient := framework.NewIngressClient("http://localhost:8080", "")
    
    // Submit order with idempotency key
    serviceClient := ingressClient.Service[OrderRequest, OrderResult]("OrderService", "Process")
    
    idempotencyKey := "order-12345"
    invocationID, err := serviceClient.Send(
        context.Background(),
order{CustomerID: "cust-1", Amount: 100.00},
        framework.IngressCallOption{
            IdempotencyKey: idempotencyKey,
            ValidationMode: framework.IdempotencyValidationFail,
        },
    )
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Submitted order: %s\\n", invocationID)
    
    // Later: Attach to get result (using idempotency key)
    result, err := serviceClient.AttachByIdempotencyKey(
        context.Background(),
        idempotencyKey,
    )
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Order result: %+v\\n", result)
}
```

### Example 2: Workflow Submission and Attach

```go
func main() {
    ingressClient := framework.NewIngressClient("http://localhost:8080", "")
    
    workflowClient := ingressClient.Workflow[WorkflowInput, WorkflowOutput](
        "ApprovalWorkflow",
        "run",
    )
    
    workflowID := "approval-wf-001"
    
    // Submit workflow
    _, err := workflowClient.Submit(
        context.Background(),
        workflowID,
        WorkflowInput{RequestID: "req-123"},
    )
    if err != nil {
        log.Fatal(err)
    }
    
    // Attach to wait for completion
    output, err := workflowClient.Attach(context.Background(), workflowID)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Workflow completed: %+v\\n", output)
}
```

### Example 3: Demonstrating Attach Requirement

```go
// Shows that Attach requires idempotency key for services
func demonstrateAttachRequirement() {
    ingressClient := framework.NewIngressClient("http://localhost:8080", "")
    serviceClient := ingressClient.Service[Input, Output]("MyService", "MyHandler")
    
    // ❌ BAD: Send without idempotency key
    invID1, _ := serviceClient.Send(context.Background(), Input{})
    
    // Cannot attach to retrieve result later - no idempotency key!
    // This will fail or hang indefinitely
    
    // ✅ GOOD: Send with idempotency key
    idempKey := "unique-key-123"
    _, _ := serviceClient.Send(
        context.Background(),
        Input{},
        framework.IngressCallOption{IdempotencyKey: idempKey},
    )
    
    // Can attach to retrieve result
    result, _ := serviceClient.AttachByIdempotencyKey(context.Background(), idempKey)
    fmt.Printf("Result: %+v\\n", result)
}
```

## Tests to Create

### Test 1: Service Send and Attach
```go
func TestIngressServiceSendAndAttach(t *testing.T) {
    // Start Restate test environment
    // Register test service
    // Create ingress client
    // Send with idempotency key
    // Attach to get result
    // Assert result matches expected
}
```

### Test 2: Workflow Submit and Attach
```go
func TestIngressWorkflowAttach(t *testing.T) {
    // Submit workflow
    // Attach to wait for completion
    // Assert output is correct
}
```

### Test 3: Validation Mode Enforcement
```go
func TestIngressValidationModes(t *testing.T) {
    // Test with Warn mode - should log but proceed
    // Test with Fail mode - should return error
    // Test with Disabled mode - should skip validation
}
```

### Test 4: Attach Without Idempotency Key Fails
```go
func TestAttachWithoutIdempotencyKeyFails(t *testing.T) {
    // Send without idempotency key
    // Attempt to attach by key
    // Assert error occurs
}
```

## Documentation Updates

1. Update `INGRESS_CLIENT_GUIDE.MD` with:
   - Complete API reference
   - Attach semantics explanation
   - Idempotency key requirement for Attach
   - Examples for all client types

2. Update `INTERSERVICE_COMM_UPDATES.MD` with:
   - Ingress client implementation section
   - Attach pattern examples

3. Create `INGRESS_ATTACH_PATTERNS.MD` with:
   - Detailed explanation of Attach vs Send
   - When to use each pattern
   - Idempotency key requirements
   - Common pitfalls

## Implementation Order

1. ✅ Add `ingress` import
2. ✅ Update `IngressClient` struct to wrap SDK client
3. ✅ Update `NewIngressClient()` to create SDK client
4. ✅ Implement `IngressCallOption.ValidationMode`
5. ✅ Implement `IngressServiceClient.Call()` and `Send()`
6.✅ Add `IngressServiceClient.AttachByIdempotencyKey()`
7. ✅ Implement `IngressObjectClient` methods
8. ✅ Implement `IngressWorkflowClient` methods
9. ✅ Create example files
10. ✅ Create test files
11. ✅ Update documentation

## Success Criteria

- [ ] All ingress client methods use actual SDK `ingress` package
- [ ] ValidationMode is respected in all client methods
- [ ] Attach methods work correctly with idempotency keys
- [ ] Examples clearly demonstrate Attach patterns
- [ ] Tests verify Attach requires idempotency keys
- [ ] Documentation explains when/how to use Attach
- [ ] No "requires restateingress package" errors
