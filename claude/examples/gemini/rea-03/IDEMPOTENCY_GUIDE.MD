# Idempotency Key Usage Guide

> **When to use (and when NOT to use) idempotency keys in Restate**

## Overview

Idempotency keys are crucial for distributed systems, but **using them unnecessarily adds complexity and overhead**. This guide explains when idempotency keys provide value and when Restate's built-in guarantees make them redundant.

## Key Principle

**Idempotency keys are for CROSS-INVOCATION protection, not WITHIN-INVOCATION protection.**

Restate's journaling already provides exactly-once execution within a handler invocation. Idempotency keys are needed when you want to:
1. **Attach** to an existing invocation from outside
2. **Deduplicate** calls across different handler invocations
3. **Retry-protect** external API calls

## When to Use Idempotency Keys

### ✅ 1. External Calls (Ingress)

**Use Case**: Calling Restate from outside a handler (web app, CLI, cron job)

**Why**: No journaling protection; need deduplication across retries

```go
// ✅ GOOD: External call with idempotency key
client := NewIngressClient("http://localhost:8080", "auth-key")
serviceClient := IngressService[OrderRequest, OrderResponse](
    client, "OrderService", "processOrder",
)

result, err := serviceClient.Call(
    ctx,
    OrderRequest{OrderID: "order-123"},
    IngressCallOption{
        IdempotencyKey: "user-123-order-123", // Prevents duplicate orders
    },
)
```

**Benefits**:
- Prevents duplicate invocations on client retry
- Enables attach semantics (get result without re-execution)
- Cross-network failure protection

### ✅ 2. Cross-Handler Calls with Attach Semantics

**Use Case**: Need to attach to a workflow/service from another handler

**Why**: Want to retrieve result without starting a new invocation

```go
// ✅ GOOD: Cross-handler call with idempotency key for attach
func ProcessOrder(ctx restate.Context, req OrderRequest) error {
    workflowClient := NewWorkflowClient[WorkflowInput, WorkflowOutput](
        "PaymentWorkflow", "run",
    )
    
    // Submit workflow with deterministic key
    idempotencyKey := GenerateDeterministicKey("payment", req.OrderID)
    workflowClient.Submit(
        ctx,
        req.OrderID,
        WorkflowInput{Amount: req.Total},
        CallOption{IdempotencyKey: idempotencyKey},
    )
    
    // Later (maybe in a different handler), attach to check status
    result, err := workflowClient.Attach(ctx, req.OrderID)
    return err
}
```

**Benefits**:
- Can attach from different handlers
- Avoid starting duplicate workflows
- Query workflow status without re-execution

### ✅ 3. Deduplication Across Handler Invocations

**Use Case**: Same logical operation might be invoked multiple times

**Why**: External retries or business logic might trigger duplicates

```go
// ✅ GOOD: Deduplication across separate handler invocations
func HandleWebhook(ctx restate.Context, webhook WebhookEvent) error {
    // External system might retry webhook delivery
    orderClient := NewServiceClient[OrderRequest, OrderResponse](
        "OrderService", "createOrder",
    )
    
    // Use webhook ID as idempotency key
    result, err := orderClient.Call(
        ctx,
        OrderRequest{WebhookID: webhook.ID},
        CallOption{
            IdempotencyKey: fmt.Sprintf("webhook-%s", webhook.ID),
        },
    )
    return err
}
```

**Benefits**:
- Prevents duplicate processing of the same webhook
- Safe against external system retries
- Idempotent business logic

## When NOT to Use Idempotency Keys

### ❌ 1. Same-Handler Execution Calls

**Anti-Pattern**: Adding idempotency keys to calls within the same handler

**Why**: Restate's journaling already guarantees exactly-once execution

```go
// ❌ BAD: Unnecessary idempotency key within same handler
func ProcessOrder(ctx restate.Context, req OrderRequest) error {
    // This call is ALREADY exactly-once via journaling
    paymentClient := NewServiceClient[PaymentRequest, PaymentResponse](
        "PaymentService", "charge",
    )
    
    // ❌ WASTEFUL: Idempotency key adds no value here
    result, err := paymentClient.Call(
        ctx,
        PaymentRequest{Amount: req.Total},
        CallOption{
            IdempotencyKey: fmt.Sprintf("payment-%s", req.OrderID), // UNNECESSARY!
        },
    )
    
    return err
}

// ✅ GOOD: Rely on Restate's journaling
func ProcessOrder(ctx restate.Context, req OrderRequest) error {
    paymentClient := NewServiceClient[PaymentRequest, PaymentResponse](
        "PaymentService", "charge",
    )
    
    // No idempotency key needed - journaling handles it
    result, err := paymentClient.Call(ctx, PaymentRequest{Amount: req.Total})
    return err
}
```

**Why It's Wasteful**:
- Restate journals this call
- On replay, uses journaled result (doesn't re-execute)
- Idempotency key adds overhead with zero benefit
- Extra validation, storage, and network bytes

### ❌ 2. Sequential Calls in Same Handler

**Anti-Pattern**: Multiple calls with different idempotency keys in same handler

**Why**: Each call is already journaled individually

```go
// ❌ BAD: Unnecessary keys for sequential operations
func ProcessOrder(ctx restate.Context, req OrderRequest) error {
    // Step 1: Validate inventory
    inventoryClient := NewServiceClient[CheckRequest, CheckResponse](
        "InventoryService", "check",
    )
    _, err := inventoryClient.Call(
        ctx,
        CheckRequest{Items: req.Items},
        CallOption{
            IdempotencyKey: fmt.Sprintf("inv-%s", req.OrderID), // UNNECESSARY!
        },
    )
    
    // Step 2: Charge payment
    paymentClient := NewServiceClient[PaymentRequest, PaymentResponse](
        "PaymentService", "charge",
    )
    _, err = paymentClient.Call(
        ctx,
        PaymentRequest{Amount: req.Total},
        CallOption{
            IdempotencyKey: fmt.Sprintf("pay-%s", req.OrderID), // UNNECESSARY!
        },
    )
    
    return err
}

// ✅ GOOD: Journaling provides all needed guarantees
func ProcessOrder(ctx restate.Context, req OrderRequest) error {
    // Each call is journaled independently
    inventoryClient := NewServiceClient[CheckRequest, CheckResponse](
        "InventoryService", "check",
    )
    _, err := inventoryClient.Call(ctx, CheckRequest{Items: req.Items})
    
    paymentClient := NewServiceClient[PaymentRequest, PaymentResponse](
        "PaymentService", "charge",
    )
    _, err = paymentClient.Call(ctx, PaymentRequest{Amount: req.Total})
    
    return err
}
```

### ❌ 3. Send (Fire-and-Forget) Within Handler

**Anti-Pattern**: Idempotency keys on Send calls within same handler

**Why**: Send is journaled, idempotency key doesn't affect anything

```go
// ❌ BAD: Idempotency key on Send does nothing useful
func ProcessOrder(ctx restate.Context, req OrderRequest) error {
    notificationClient := NewServiceClient[NotifyRequest, NotifyResponse](
        "NotificationService", "send",
    )
    
    // ❌ WASTEFUL: Send is journaled, key adds no value
    notificationClient.Send(
        ctx,
        NotifyRequest{Email: req.Email},
        CallOption{
            IdempotencyKey: fmt.Sprintf("notify-%s", req.OrderID), // UNNECESSARY!
        },
    )
    
    return nil
}

// ✅ GOOD: No key needed
func ProcessOrder(ctx restate.Context, req OrderRequest) error {
    notificationClient := NewServiceClient[NotifyRequest, NotifyResponse](
        "NotificationService", "send",
    )
    
    // Journaling ensures exactly-once Send
    notificationClient.Send(ctx, NotifyRequest{Email: req.Email})
    return nil
}
```

## Auto-Detection Pattern

The framework can detect same-handler execution and skip automatic key injection:

```go
// Framework implementation
func (c ServiceClient[I, O]) Call(ctx restate.Context, input I, opts ...CallOption) (O, error) {
    opt := mergeOptions(opts)
    
    // Auto-detect same-handler execution
    if opt.IdempotencyKey != "" && isSameHandlerExecution(ctx) {
        // Log warning
        logger := getLogger(ctx)
        logger.Warn("idempotency key provided for same-handler call",
            "key", opt.IdempotencyKey,
            "service", c.serviceName,
            "handler", c.handlerName,
            "reason", "Restate journaling already provides exactly-once execution")
        
        // Remove unnecessary key (opt-in behavior)
        if opt.AutoRemoveRedundantKeys {
            opt.IdempotencyKey = ""
        }
    }
    
    // Continue with call...
}

// Helper to detect same-handler execution
func isSameHandlerExecution(ctx restate.Context) bool {
    // Check if ctx is a journaled context (not RunContext or external)
    switch ctx.(type) {
    case restate.Context, restate.ObjectContext, restate.WorkflowContext:
        return true // Within handler execution
    case restate.RunContext:
        return false // Side effect, key might be useful
    default:
        return false
    }
}
```

## Configuration Options

### Global Policy

```go
// Set framework-wide behavior
SetFrameworkPolicy(PolicyStrict) // Reject redundant keys
SetFrameworkPolicy(PolicyWarn)   // Log warnings (default)
SetFrameworkPolicy(PolicyDisabled) // Allow everything
```

### Per-Call Override

```go
// Explicitly allow idempotency key (bypass auto-detection)
client.Call(ctx, input, CallOption{
    IdempotencyKey: "my-key",
    AllowRedundantKey: true, // Override auto-detection
})
```

## Decision Tree

```
Are you calling from OUTSIDE a Restate handler?
├─ YES → Use idempotency key ✅
└─ NO  → Are you calling from WITHIN a handler?
         ├─ YES → Do you need Attach semantics?
         │        ├─ YES → Use idempotency key ✅
         │        └─ NO  → DON'T use idempotency key ❌
         └─ NO  → You're not in Restate context?
                  └─ Use idempotency key ✅
```

## Quick Reference Table

| Call Type | From Where | Idempotency Key? | Reason |
|-----------|------------|------------------|--------|
| Service.Call | Outside handler (ingress) | ✅ YES | No journaling protection |
| Service.Call | Inside handler | ❌ NO | Journaling provides exactly-once |
| Service.Send | Outside handler (ingress) | ✅ YES | Deduplication needed |
| Service.Send | Inside handler | ❌ NO | Already journaled |
| Workflow.Submit | Outside handler | ✅ YES | Attach semantics |
| Workflow.Submit | Inside handler (no attach) | ❌ NO | Journaled submission |
| Workflow.Submit | Inside handler (need attach) | ✅ YES | Cross-handler attach |
| Object.Call | Outside handler | ✅ YES | No journaling |
| Object.Call | Inside handler | ❌ NO | Journaled |

## Best Practices

### ✅ DO

1. **Use idempotency keys for all ingress calls**:
   ```go
   // External API calling Restate
   client.Call(ctx, input, IngressCallOption{
       IdempotencyKey: "unique-key",
   })
   ```

2. **Use deterministic keys for workflow submission**:
   ```go
   key := GenerateDeterministicKey("workflow", entityID, operation)
   ```

3. **Document why you're using a key**:
   ```go
   // Need attach semantics to query from different handler
   workflowClient.Submit(ctx, id, input, CallOption{
       IdempotencyKey: key,
   })
   ```

4. **Let framework auto-detect and warn**:
   ```go
   // Enable warnings for unnecessary keys
   SetFrameworkPolicy(PolicyWarn)
   ```

### ❌ DON'T

1. **Don't add keys "just in case"**:
   ```go
   // ❌ Adds no value within handler
   client.Call(ctx, input, CallOption{
       IdempotencyKey: "unnecessary",
   })
   ```

2. **Don't use random keys within handlers**:
   ```go
   // ❌ Random keys within handler are wasteful
   client.Call(ctx, input, CallOption{
       IdempotencyKey: uuid.New().String(),
   })
   ```

3. **Don't assume "more keys = safer"**:
   - Journaling already provides exactly-once
   - Extra keys add overhead without benefit

## Performance Impact

### With Unnecessary Keys

| Metric | Impact |
|--------|--------|
| Storage | +~50 bytes per key |
| Network | +~50 bytes per request |
| Validation | +~10μs CPU per call |
| Complexity | Higher debugging overhead |

### Without Keys (journaling only)

| Metric | Impact |
|--------|--------|
| Storage | Minimal (journal entries only) |
| Network | Minimal overhead |
| Validation | None |
| Complexity | Lower |

**Recommendation**: Only use keys when they provide actual value.

## Migration Guide

### From: Always Using Keys

```go
// OLD: Unnecessary keys everywhere
func ProcessOrder(ctx restate.Context, req OrderRequest) error {
    client1.Call(ctx, input1, CallOption{IdempotencyKey: "key1"})
    client2.Call(ctx, input2, CallOption{IdempotencyKey: "key2"})
    client3.Send(ctx, input3, CallOption{IdempotencyKey: "key3"})
}
```

### To: Keys Only When Needed

```go
// NEW: No keys within handler (journaling provides guarantees)
func ProcessOrder(ctx restate.Context, req OrderRequest) error {
    client1.Call(ctx, input1) // Journaled
    client2.Call(ctx, input2) // Journaled
    client3.Send(ctx, input3) // Journaled
}

// Keys only for ingress or cross-invocation attach
func ExternalAPI(w http.ResponseWriter, r *http.Request) {
    client := NewIngressClient(...)
    svcClient := IngressService[I, O](client, "Service", "handler")
    
    // ✅ Key needed here (no journaling)
    svcClient.Call(ctx, input, IngressCallOption{
        IdempotencyKey: extractIdempotencyKey(r),
    })
}
```

## Monitoring

### Metrics to Track

```go
// Track unnecessary key usage
idempotency_key_usage{
    location="same_handler",    // Where key was used
    action="warning_logged",    // What framework did
    service="OrderService",
    handler="process",
}

// Track key effectiveness
idempotency_key_hits{
    type="deduplication",  // Key prevented duplicate
    source="ingress",
}
```

### Alerts

```yaml
# High rate of unnecessary keys
- alert: UnnecessaryIdempotencyKeys
  expr: rate(idempotency_key_warnings[5m]) > 10
  annotations:
    summary: "High rate of unnecessary idempotency keys"
    description: "Consider removing keys from same-handler calls"
```

## Related Documentation

- [Framework Policy](file:///home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/FRAMEWORK_POLICY.MD) - Validation modes
- [Ingress Client](file:///home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/INGRESS_CLIENT_TYPE_SAFETY.MD) - External calls
- [Interservice Communication](file:///home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/INTERSERVICE_COMM_UPDATES.MD) - Internal vs external

## Summary

| Concept | Key Takeaway |
|---------|--------------|
| **When to use** | External calls, cross-invocation attach, deduplication |
| **When NOT to use** | Same-handler execution (journaling provides guarantees) |
| **Auto-detection** | Framework can detect and warn about unnecessary keys |
| **Performance** | Unnecessary keys add ~50 bytes storage + 10μs validation |
| **Best practice** | Use keys sparingly, only when they provide actual value |

**Remember**: Restate's journaling is your friend. Trust it for within-handler guarantees, use idempotency keys for cross-boundary protection! 🎯
