# Restate Go SDK vs Rea Framework: A Comprehensive Comparison

> Understanding when to use the standard SDK versus the rea framework for building resilient applications

## Table of Contents
- [Overview](#overview)
- [Quick Comparison](#quick-comparison)
- [Standard Restate Go SDK](#standard-restate-go-sdk)
- [Rea Framework](#rea-framework)
- [Side-by-Side Examples](#side-by-side-examples)
- [When to Use Which](#when-to-use-which)
- [Migration Guide](#migration-guide)

---

## Overview

**Restate Go SDK** is the official low-level SDK providing direct access to all Restate primitives. It's the foundation for building durable applications.

**Rea Framework** (`github.com/pithomlabs/rea`) is a high-level abstraction built **on top of** the Restate SDK. It provides utility functions, enhanced patterns, and guardrails to prevent common anti-patterns.

**Key Insight**: Rea doesn't replace the SDK—it enhances it. You use both together.

---

## Quick Comparison

| Feature | Standard SDK | Rea Framework |
|---------|-------------|---------------|
| **Service Registration** | `restate.Reflect()` | Same (`restate.Reflect()`) |
| **Retry Logic** | `restate.Run()` | `rea.RunWithRetry()` with configurable backoff |
| **State Clearing** | `restate.ClearAll(ctx)` | `rea.ClearAll(ctx)` with error handling |
| **Concurrency** | `restate.Wait()` | `rea.Gather()`, `rea.FanOutFail()` |
| **Batch Processing** | Manual implementation | `rea.ProcessBatch()` with concurrency control |
| **Anti-Pattern Detection** | None | Built-in guards and logging |
| **Learning Curve** | Moderate | Lower (safer defaults) |
| **Control Level** | Full control | Opinionated patterns |
| **Dependencies** | Restate SDK only | Restate SDK + Rea |

---

## Standard Restate Go SDK

### Installation
```bash
go get github.com/restatedev/sdk-go
```

### Core Primitives

**Service Registration**
```go
server.NewRestate().
    Bind(restate.Reflect(MyService{})).
    Start(ctx, ":9080")
```

**Basic Run Block**
```go
result, err := restate.Run(ctx, func(ctx restate.RunContext) (string, error) {
    return callExternalAPI()
})
```

**State Management**
```go
value, err := restate.Get[string](ctx, "key")
restate.Set(ctx, "key", "value")
restate.ClearAll(ctx)
```

**Concurrency**
```go
f1 := restate.Service[string](ctx, "Svc1", "Method1").RequestFuture("input")
f2 := restate.Service[string](ctx, "Svc2", "Method2").RequestFuture("input")

for fut, err := range restate.Wait(ctx, f1, f2) {
    result, _ := fut.(restate.ResponseFuture[string]).Response()
    // Process result
}
```

### Pros
✅ **Full control** over execution  
✅ **Minimal abstraction** overhead  
✅ **Official support** from Restate team  
✅ **Direct access** to all primitives  
✅ **Lightweight** - no extra dependencies  

### Cons
❌ **Manual retry configuration** (verbose)  
❌ **No anti-pattern guards** (easy to make mistakes)  
❌ **Boilerplate** for common patterns  
❌ **Learning curve** for advanced patterns  

---

## Rea Framework

### Installation
```bash
go get github.com/pithomlabs/rea
```

### Enhanced Patterns

**Service Registration** (Same as SDK)
```go
// Rea doesn't change service registration
server.NewRestate().
    Bind(restate.Reflect(MyService{})).
    Start(ctx, ":9080")
```

**Enhanced Run with Retry**
```go
cfg := rea.RunConfig{
    MaxRetries:     3,
    InitialDelay:   100 * time.Millisecond,
    MaxDelay:       2 * time.Second,
    BackoffFactor:  2.0,
    Name:           "APICall",
}

result, err := rea.RunWithRetry(ctx, cfg, func(ctx restate.RunContext) (string, error) {
    return callExternalAPI()
})
```

**Safe State Clearing**
```go
// Returns error instead of panicking
if err := rea.ClearAll(ctx); err != nil {
    return err
}
```

**Advanced Concurrency**
```go
// Gather - Wait for all futures
results, err := rea.Gather(ctx, future1, future2, future3)

// FanOutFail - Fail fast on first error
operations := []func() (string, error){op1, op2, op3}
results, err := rea.FanOutFail(ctx, operations)

// ProcessBatch - Controlled concurrency
results, err := rea.ProcessBatch(ctx, items, processor, 
    rea.BatchConfig{MaxConcurrency: 5})
```

**Workflow Iteration**
```go
// ForEach with durable iteration
err := rea.ForEach(ctx, items, func(item Item, index int) error {
    return processItem(ctx, item)
})
```

### Pros
✅ **Reduced boilerplate** for common patterns  
✅ **Built-in retry logic** with exponential backoff  
✅ **Anti-pattern guards** prevent common mistakes  
✅ **Better error handling** (errors vs panics)  
✅ **Batch processing** helpers  
✅ **Type-safe** concurrent operations  

### Cons
❌ **Extra dependency** to maintain  
❌ **Less flexible** (opinionated patterns)  
❌ **Community-maintained** (not official Restate)  
❌ **Learning curve** for rea-specific APIs  

---

## Side-by-Side Examples

### Example 1: Retry with Backoff

**Standard SDK** (Manual)
```go
func callWithRetry(ctx restate.Context) (string, error) {
    var lastErr error
    delay := 100 * time.Millisecond
    
    for attempt := 0; attempt < 3; attempt++ {
        result, err := restate.Run(ctx, func(ctx restate.RunContext) (string, error) {
            return callAPI()
        })
        if err == nil {
            return result, nil
        }
        lastErr = err
        
        if attempt < 2 {
            restate.Sleep(ctx, delay)
            delay *= 2  // Exponential backoff
        }
    }
    return "", lastErr
}
```

**Rea Framework** (Declarative)
```go
func callWithRetry(ctx restate.Context) (string, error) {
    cfg := rea.RunConfig{
        MaxRetries:     3,
        InitialDelay:   100 * time.Millisecond,
        BackoffFactor:  2.0,
    }
    return rea.RunWithRetry(ctx, cfg, callAPI)
}
```

**Result**: ~15 lines → 5 lines

---

### Example 2: Concurrent Operations

**Standard SDK**
```go
func processConcurrent(ctx restate.Context, items []string) ([]Result, error) {
    var futures []restate.Future
    
    for _, item := range items {
        f := restate.Service[Result](ctx, "Processor", "Process").
            RequestFuture(item)
        futures = append(futures, f)
    }
    
    var results []Result
    for fut, err := range restate.Wait(ctx, futures...) {
        if err != nil {
            return nil, err
        }
        result, err := fut.(restate.ResponseFuture[Result]).Response()
        if err != nil {
            return nil, err
        }
        results = append(results, result)
    }
    return results, nil
}
```

**Rea Framework**
```go
func processConcurrent(ctx restate.Context, items []string) ([]Result, error) {
    return rea.MapConcurrent(ctx, items, func(item string) (Result, error) {
        return restate.Service[Result](ctx, "Processor", "Process").
            Request(item)
    })
}
```

**Result**: ~20 lines → 5 lines

---

### Example 3: State Clearing

**Standard SDK**
```go
func clearUserState(ctx restate.ObjectContext) error {
    restate.ClearAll(ctx)  // Panics on error
    return nil
}
```

**Rea Framework**
```go
func clearUserState(ctx restate.ObjectContext) error {
    return rea.ClearAll(ctx)  // Returns error
}
```

**Result**: Better error propagation

---

## When to Use Which

### Use **Standard Restate SDK** When:

✅ **Building simple services** with minimal business logic  
✅ **Learning Restate** (understand fundamentals first)  
✅ **Need full control** over execution flow  
✅ **Avoiding dependencies** (minimal surface area)  
✅ **Contributing to Restate ecosystem** (SDK is the standard)  
✅ **Working with official examples** and documentation  

**Example Use Cases:**
- Simple CRUD services
- Proof-of-concept projects
- Educational/tutorial code
- Libraries intended for wide distribution

---

### Use **Rea Framework** When:

✅ **Building production systems** with complex workflows  
✅ **Team wants guardrails** against anti-patterns  
✅ **Reducing boilerplate** is a priority  
✅ **Advanced concurrency** patterns needed  
✅ **Batch processing** is common  
✅ **Saga orchestration** with compensations  

**Example Use Cases:**
- E-commerce order fulfillment (this example!)
- Payment processing systems
- Multi-step approval workflows
- Data pipeline orchestration
- Microservice choreography

---

## Migration Guide

### From SDK to Rea

**Step 1**: Add dependency
```bash
go get github.com/pithomlabs/rea
```

**Step 2**: Replace patterns incrementally
```go
// Before (SDK)
result, err := restate.Run(ctx, func(ctx restate.RunContext) (T, error) {
    return operation()
})

// After (Rea)
cfg := rea.DefaultRunConfig("operation")
result, err := rea.RunWithRetry(ctx, cfg, operation)
```

**Step 3**: Keep service registration unchanged
```go
// No change needed - rea uses same registration
server.NewRestate().Bind(restate.Reflect(Service{}))
```

**Step 4**: Adopt rea utilities gradually
- Start with `RunWithRetry` for external calls
- Use `Gather` for concurrent operations
- Apply `ClearAll` for safer state management

### From Rea to SDK

Simply replace rea utility calls with equivalent SDK code. Service registration remains identical.

---

## Best Practices

### Hybrid Approach (Recommended)

**Use SDK for:**
- Service registration
- Basic state operations
- Simple workflows

**Use Rea for:**
- Complex retry logic
- Batch processing
- Advanced concurrency
- Saga orchestration

**Example**:
```go
// Use SDK for basic operations
basket, _ := restate.Get[[]string](ctx, "basket")
basket = append(basket, item)
restate.Set(ctx, "basket", basket)

// Use Rea for complex external calls
cfg := rea.RunConfig{MaxRetries: 3, InitialDelay: 100*time.Millisecond}
receipt, err := rea.RunWithRetry(ctx, cfg, func(ctx restate.RunContext) (Receipt, error) {
    return paymentGateway.Charge(order)
})
```

---

## Rea Framework Features Reference

### Retry Patterns
- `rea.RunWithRetry()` - Configurable exponential backoff
- `rea.RunDo()` - Single execution with context
- `rea.RunDoVoid()` - Void return type helper

### Concurrency
- `rea.Gather()` - Wait for all futures
- `rea.FanOutFail()` - Fail-fast parallel execution
- `rea.MapConcurrent()` - Parallel map operation
- `rea.ProcessBatch()` - Controlled concurrency batching

### Workflow Utilities
- `rea.ForEach()` - Durable iteration
- `rea.RacePromiseWithTimeout()` - Promise with timeout
- `rea.GetInternalSignal()` - Promise helper

### State Management
- `rea.ClearAll()` - Safe state clearing

### Saga Support
- `rea.NewControlPlaneService()` - Saga orchestration
- `rea.AddCompensationStep()` - Register compensations
- `rea.AwaitHumanApproval()` - Human-in-the-loop

### Ingress Helpers
- `rea.NewIngressClient()` - Enhanced ingress client
- `rea.IngressObject()` - Type-safe object calls
- `rea.IngressService()` - Type-safe service calls

---

## Real-World Example

### This Repository Structure

```
rea/
├── ingress/
│   └── ingress.go       # Uses standard SDK (simple HTTP handlers)
└── services/
    └── svcs.go          # Uses Rea framework (complex workflows)
```

**Why?**
- **Ingress**: Simple HTTP → Restate bridge, no retry logic needed
- **Services**: Complex orchestration, benefits from rea utilities

**Pattern**:
```go
// ingress.go - Standard SDK
func (i *Ingress) handleCheckout(w http.ResponseWriter, r *http.Request) {
    restateingress.ObjectSend[string](i.client, "UserSession", userID, "Checkout").
        Send(ctx, orderID)
}

// svcs.go - Rea Framework
func (ShippingService) InitiateShipment(ctx restate.Context, req ShipmentRequest) (bool, error) {
    cfg := rea.RunConfig{MaxRetries: 3, InitialDelay: 100*time.Millisecond}
    return rea.RunWithRetry(ctx, cfg, func(ctx restate.RunContext) (bool, error) {
        return callExternalShippingAPI(req)
    })
}
```

---

## Performance Considerations

### Overhead

**Rea Framework** adds minimal overhead:
- Function call indirection (~nanoseconds)
- Configuration struct allocation (~bytes)
- No runtime reflection beyond SDK

**Benchmark** (rough estimate):
```
SDK Direct:           100ns/op
Rea RunWithRetry:     110ns/op  (+10% overhead)
```

**Verdict**: Negligible for I/O-bound operations

---

## Community and Support

### Standard SDK
- **Official Restate** support
- Complete documentation at https://docs.restate.dev
- Active Discord community
- GitHub: https://github.com/restatedev/sdk-go

### Rea Framework  
- **Community-maintained** by pithomlabs
- Source: https://github.com/pithomlabs/rea
- Documentation: https://pkg.go.dev/github.com/pithomlabs/rea
- Issues and PRs welcome

---

## Summary

**Choose Standard SDK** if you want:
- Minimal dependencies
- Full control
- Official support
- Learning Restate fundamentals

**Choose Rea Framework** if you want:
- Production-grade patterns
- Reduced boilerplate
- Anti-pattern guards
- Advanced utilities

**Best Approach**:
Start with the SDK to learn Restate, then adopt Rea utilities where they add value. Many teams use both together, leveraging SDK for simplicity and Rea for complexity.

---

## Additional Resources

- **Restate Docs**: https://docs.restate.dev
- **SDK Reference**: https://pkg.go.dev/github.com/restatedev/sdk-go
- **Rea Reference**: https://pkg.go.dev/github.com/pithomlabs/rea
- **Examples**: This repository demonstrates both approaches

**Questions?** File an issue or join the Restate Discord!
