# Ingress Client Type Safety Enhancement

> **Type-safe ingress client creation for external calls to Restate services**

## Overview

Enhanced the `IngressClient` to support **type-safe generic client creation**, allowing external applications to call Restate services with compile-time type checking for both input and output types.

## Problem Solved

### Before: Unsafe `any` Types

```go
// Old approach - no type safety
client := ingressClient.Service("OrderService", "processOrder")
result, err := client.Call(ctx, input, opts...) // result is 'any', requires type assertion
```

**Issues**:
- Return type is always `any`, requiring manual type assertions
- No compile-time validation of request/response types
- Easy to make mistakes with type mismatches
- Runtime errors instead of compile-time errors

### After: Type-Safe Generics

```go
// New approach - full type safety
client := IngressService[OrderRequest, OrderResponse](
    ingressClient, 
    "OrderService", 
    "processOrder",
)
result, err := client.Call(ctx, input, opts...) // result is OrderResponse, no assertion needed
```

**Benefits**:
- ✅ Compile-time type checking
- ✅ No manual type assertions required
- ✅ IDE autocomplete for request/response types
- ✅ Catch type mismatches at compile time, not runtime

## Technical Implementation

### Challenge: Go Generic Methods Limitation

Go **does not support generic methods on non-generic types**. This code is invalid:

```go
// ❌ INVALID - generic methods not allowed
func (ic *IngressClient) Service[I, O any](name, handler string) *IngressServiceClient[I, O] {
    // ...
}
```

**Error**: `method must have no type parameters`

### Solution: Standalone Generic Functions

We use **standalone generic constructor functions** instead:

```go
// ✅ VALID - standalone generic function
func IngressService[I, O any](
    ic *IngressClient, 
    serviceName, 
    handlerName string,
) *IngressServiceClient[I, O] {
    return &IngressServiceClient[I, O]{
        ingress:     ic,
        serviceName: serviceName,
        handlerName: handlerName,
    }
}
```

## API Reference

### IngressService

Creates a type-safe client for calling a **stateless service**.

```go
func IngressService[I, O any](
    ic *IngressClient,
    serviceName string,
    handlerName string,
) *IngressServiceClient[I, O]
```

**Type Parameters**:
- `I`: Input request type
- `O`: Output response type

**Example**:
```go
client := IngressService[OrderRequest, OrderResponse](
    ingressClient,
    "OrderService",
    "processOrder",
)
```

### IngressObject

Creates a type-safe client for calling a **Virtual Object**.

```go
func IngressObject[I, O any](
    ic *IngressClient,
    serviceName string,
    handlerName string,
) *IngressObjectClient[I, O]
```

**Type Parameters**:
- `I`: Input request type
- `O`: Output response type

**Example**:
```go
client := IngressObject[UpdateRequest, UpdateResponse](
    ingressClient,
    "UserObject",
    "updateProfile",
)
```

### IngressWorkflow

Creates a type-safe client for calling a **Workflow**.

```go
func IngressWorkflow[I, O any](
    ic *IngressClient,
    serviceName string,
    handlerName string,
) *IngressWorkflowClient[I, O]
```

**Type Parameters**:
- `I`: Workflow input type
- `O`: Workflow output type

**Example**:
```go
client := IngressWorkflow[WorkflowInput, WorkflowOutput](
    ingressClient,
    "OrderWorkflow",
    "run",
)
```

## Complete Usage Example

### Setup

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    
    . "github.com/restatedev/examples/rea2/claude"
)

// Define your request/response types
type OrderRequest struct {
    OrderID string   `json:"orderId"`
    Items   []string `json:"items"`
}

type OrderResponse struct {
    OrderID string `json:"orderId"`
    Status  string `json:"status"`
}
```

### 1. Create Ingress Client

```go
// Create base ingress client
ingressClient := NewIngressClient("http://localhost:8080", "your-auth-key")
```

### 2. Call Service with Idempotency Key

**Type-safe service client**:

```go
// Create type-safe client
serviceClient := IngressService[OrderRequest, OrderResponse](
    ingressClient,
    "OrderService",
    "processOrder",
)

// Call with idempotency key (enables Attach semantics)
idempotencyKey := "order-12345"
result, err := serviceClient.Call(
    context.Background(),
    OrderRequest{
        OrderID: "order-12345",
        Items:   []string{"item1", "item2"},
    },
    IngressCallOption{
        IdempotencyKey: idempotencyKey,
        ValidationMode: IdempotencyValidationWarn,
    },
)
if err != nil {
    log.Fatalf("Service call failed: %v", err)
}
fmt.Printf("Order processed: %+v\n", result) // result is OrderResponse
```

### 3. Attach to Existing Invocation

**Use the idempotency key to attach**:

```go
// Attach to the same invocation using idempotency key
attachedResult, err := serviceClient.AttachByIdempotencyKey(
    context.Background(),
    idempotencyKey,
)
if err != nil {
    log.Fatalf("Attach failed: %v", err)
}
fmt.Printf("Attached result: %+v\n", attachedResult) // Same OrderResponse
```

**How it works**:
- Idempotency key uniquely identifies the invocation
- If invocation is still running, attach waits for result
- If invocation completed, returns cached result immediately
- No duplicate execution, guaranteed exactly-once semantics

### 4. Submit and Attach to Workflow

**Submit workflow for async execution**:

```go
type WorkflowInput struct {
    Data string `json:"data"`
}

type WorkflowOutput struct {
    Result string `json:"result"`
}

// Create workflow client
workflowClient := IngressWorkflow[WorkflowInput, WorkflowOutput](
    ingressClient,
    "OrderWorkflow",
    "run",
)

// Submit workflow (returns immediately)
workflowID := "workflow-789"
invocationID, err := workflowClient.Submit(
    context.Background(),
    workflowID,
    WorkflowInput{Data: "some data"},
    IngressCallOption{
        IdempotencyKey: "workflow-idempotency-123",
        Delay:          5 * time.Second, // Optional delay
    },
)

// Attach to get result (blocks until workflow completes)
workflowResult, err := workflowClient.Attach(
    context.Background(),
    workflowID,
)
fmt.Printf("Workflow result: %+v\n", workflowResult) // WorkflowOutput
```

### 5. Fire-and-Forget (Send)

**Send without waiting for response**:

```go
type NotificationRequest struct {
    Email   string `json:"email"`
    Message string `json:"message"`
}

type NotificationResponse struct {
    Sent bool `json:"sent"`
}

sendClient := IngressService[NotificationRequest, NotificationResponse](
    ingressClient,
    "NotificationService",
    "sendEmail",
)

invocationID, err := sendClient.Send(
    context.Background(),
    NotificationRequest{
        Email:   "user@example.com",
        Message: "Hello!",
    },
    IngressCallOption{
        IdempotencyKey: "notification-456",
    },
)
fmt.Printf("Notification sent with invocation ID: %s\n", invocationID)
```

### 6. Virtual Object Calls

**Call object handlers with object key**:

```go
type UpdateRequest struct {
    Name string `json:"name"`
}

type UpdateResponse struct {
    Updated bool `json:"updated"`
}

objectClient := IngressObject[UpdateRequest, UpdateResponse](
    ingressClient,
    "UserObject",
    "updateProfile",
)

// Call with object key
objectResult, err := objectClient.Call(
    context.Background(),
    "user-123", // Object key
    UpdateRequest{Name: "John Doe"},
    IngressCallOption{
        IdempotencyKey: "update-profile-789",
    },
)

// Attach to object invocation by idempotency key
attachedObjResult, err := objectClient.AttachByIdempotencyKey(
    context.Background(),
    "user-123", // Object key
    "update-profile-789", // Idempotency key
)
```

## Comparison with Internal Clients

### Internal Clients (Inside Handlers)

```go
// Used INSIDE Restate handlers
func ProcessOrder(ctx restate.Context, req OrderRequest) (OrderResponse, error) {
    // Create internal client
    client := NewServiceClient[PaymentRequest, PaymentResponse](
        "PaymentService",
        "charge",
    )
    
    // Call other service from within handler
    result, err := client.Call(ctx, paymentReq)
    return orderResp, nil
}
```

**Characteristics**:
- Uses `restate.Context` from handler
- Automatic durability and replay protection
- Part of Restate's execution journal
- Cannot be used outside handlers

### Ingress Clients (Outside Handlers)

```go
// Used OUTSIDE Restate handlers (external apps)
func main() {
    ingressClient := NewIngressClient("http://localhost:8080", "auth-key")
    
    // Create ingress client
    client := IngressService[OrderRequest, OrderResponse](
        ingressClient,
        "OrderService",
        "processOrder",
    )
    
    // Call from external application
    result, err := client.Call(context.Background(), orderReq)
}
```

**Characteristics**:
- Uses `context.Context` (standard Go context)
- Calls via HTTP to Restate ingress
- Used from external applications, CLIs, web servers
- Requires auth key for security
- Supports Attach semantics via idempotency keys

## Idempotency Key Validation

All ingress clients support **configurable idempotency key validation**:

```go
IngressCallOption{
    IdempotencyKey: "my-key",
    ValidationMode: IdempotencyValidationWarn, // Options below
}
```

**Validation Modes**:

| Mode | Behavior | Use Case |
|------|---------|----------|
| `IdempotencyValidationWarn` | Log warnings but continue | Development, staging |
| `IdempotencyValidationFail` | Fail with error on invalid key | Production, CI |
| `IdempotencyValidationDisabled` | Skip validation entirely | Testing only |

**Best Practices**:
- Use `Fail` mode in production for safety
- Use `Warn` mode in development to catch issues early
- Never use `Disabled` mode in production

## Migration Guide

### From Old Non-Generic API

**Before**:
```go
// Old: Returns 'any', requires type assertion
client := ingressClient.Service("OrderService", "processOrder")
result, err := client.Call(ctx, input)
orderResp := result.(OrderResponse) // Manual assertion, runtime check
```

**After**:
```go
// New: Type-safe, no assertion needed
client := IngressService[OrderRequest, OrderResponse](
    ingressClient,
    "OrderService",
    "processOrder",
)
result, err := client.Call(ctx, input) // result is OrderResponse
```

### Update Import Pattern

If using **dot import** for convenience:

```go
import (
    . "github.com/restatedev/examples/rea2/claude"
)

// No package prefix needed
client := IngressService[OrderRequest, OrderResponse](...)
```

If using **named import**:

```go
import (
    framework "github.com/restatedev/examples/rea2/claude"
)

// Use package prefix
client := framework.IngressService[OrderRequest, OrderResponse](...)
```

## Advanced Patterns

### 1. Reusable Client Factory

```go
type ClientFactory struct {
    ingress *IngressClient
}

func NewClientFactory(baseURL, authKey string) *ClientFactory {
    return &ClientFactory{
        ingress: NewIngressClient(baseURL, authKey),
    }
}

func (cf *ClientFactory) OrderService() *IngressServiceClient[OrderRequest, OrderResponse] {
    return IngressService[OrderRequest, OrderResponse](
        cf.ingress,
        "OrderService",
        "processOrder",
    )
}

func (cf *ClientFactory) UserObject() *IngressObjectClient[UpdateRequest, UpdateResponse] {
    return IngressObject[UpdateRequest, UpdateResponse](
        cf.ingress,
        "UserObject",
        "updateProfile",
    )
}
```

**Usage**:
```go
factory := NewClientFactory("http://localhost:8080", "auth-key")
orderClient := factory.OrderService()
result, err := orderClient.Call(ctx, orderReq)
```

### 2. Retry with Exponential Backoff

```go
func CallWithRetry[I, O any](
    client *IngressServiceClient[I, O],
    ctx context.Context,
    input I,
    opts IngressCallOption,
) (O, error) {
    var result O
    var err error
    
    for attempt := 0; attempt < 3; attempt++ {
        result, err = client.Call(ctx, input, opts)
        if err == nil {
            return result, nil
        }
        
        backoff := time.Duration(1<<uint(attempt)) * time.Second
        time.Sleep(backoff)
    }
    
    return result, fmt.Errorf("after 3 retries: %w", err)
}
```

### 3. Parallel Calls

```go
func ParallelServiceCalls(ctx context.Context) error {
    ingressClient := NewIngressClient("http://localhost:8080", "auth-key")
    
    // Create clients
    orderClient := IngressService[OrderRequest, OrderResponse](
        ingressClient, "OrderService", "processOrder",
    )
    paymentClient := IngressService[PaymentRequest, PaymentResponse](
        ingressClient, "PaymentService", "charge",
    )
    
    // Execute in parallel
    type result struct {
        order   OrderResponse
        payment PaymentResponse
        err     error
    }
    
    ch := make(chan result, 2)
    
    go func() {
        order, err := orderClient.Call(ctx, orderReq)
        ch <- result{order: order, err: err}
    }()
    
    go func() {
        payment, err := paymentClient.Call(ctx, paymentReq)
        ch <- result{payment: payment, err: err}
    }()
    
    // Collect results
    for i := 0; i < 2; i++ {
        r := <-ch
        if r.err != nil {
            return r.err
        }
    }
    
    return nil
}
```

## Troubleshooting

### Error: "method must have no type parameters"

**Cause**: Trying to use generic methods on `IngressClient`

**Solution**: Use standalone generic functions instead:

```go
// ❌ Wrong
client := ingressClient.Service[I, O](...)

// ✅ Correct
client := IngressService[I, O](ingressClient, ...)
```

### Error: "cannot infer type arguments"

**Cause**: Go cannot infer the generic type parameters

**Solution**: Explicitly specify both input and output types:

```go
// ❌ Wrong - types not specified
client := IngressService(ingressClient, "OrderService", "processOrder")

// ✅ Correct - both types specified
client := IngressService[OrderRequest, OrderResponse](
    ingressClient, "OrderService", "processOrder",
)
```

### Import Path Issues

If you get import errors, ensure the framework is in your `go.mod`:

```bash
go mod init your-module
go get github.com/restatedev/examples/rea2/claude
```

Or use a replace directive for local development:

```go
// go.mod
replace github.com/restatedev/examples/rea2/claude => ./path/to/framework
```

## Related Documentation

- [Interservice Communication](file:///home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/INTERSERVICE_COMM_UPDATES.MD) - Internal vs external clients
- [Framework Policy](file:///home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/FRAMEWORK_POLICY.MD) - Validation mode configuration
- [Framework vs SDK](file:///home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/FRAMEWORK_VS_SDK_SUMMARY.MD) - Overall comparison

## Summary

| Aspect | Details |
|--------|---------|
| **Purpose** | Type-safe ingress client creation for external calls |
| **Implementation** | Standalone generic constructor functions (Go limitation workaround) |
| **Type Safety** | Full compile-time checking of request/response types |
| **Use Cases** | External apps, CLIs, web servers calling Restate services |
| **Key Functions** | `IngressService`, `IngressObject`, `IngressWorkflow` |
| **Backward Compatible** | Yes, old non-generic API still works |

The type-safe ingress client API provides a robust, compile-time validated way to call Restate services from external applications while maintaining the full power of Attach semantics and idempotency guarantees! 🎉
