# TL;DR

The only requirement is your business logic resides in Restate. The rest of your application can be built with any Web framework in Go, databases, message queues, and other services.


# Rea Framework

> **High-level abstraction framework for building robust, durable distributed applications with Restate**

The Rea ("Restate Enhanced Abstractions") framework is a comprehensive Go library that wraps the Restate SDK with opinionated best practices, type safety, and developer-friendly APIs. It eliminates boilerplate, prevents common anti-patterns, and provides production-ready patterns for distributed orchestration.

---

## 🎯 Core Philosophy

**Control Plane vs Data Plane Separation**

```
┌─────────────────────────────────────────────────────────────┐
│                    CONTROL PLANE                             │
│  (Orchestration Layer - Workflows, Sagas, Coordination)     │
├─────────────────────────────────────────────────────────────┤
│  • Workflow Orchestrators (WorkflowContext)                  │
│  • Virtual Object Coordinators (ObjectContext)               │
│  • Saga Controllers (compensation logic)                     │
│  • Promise/Awakeable Coordination                            │
└─────────────────┬───────────────────────────────────────────┘
                  │ Invokes via Request/RequestFuture/Send
                  ▼
┌─────────────────────────────────────────────────────────────┐
│                     DATA PLANE                               │
│     (Execution Layer - Business Logic, Side Effects)        │
├─────────────────────────────────────────────────────────────┤
│  • Stateless Services (Basic Service - Context)              │
│  • Stateful Services (Virtual Objects - read-only queries)   │
│  • External Integration Services (Run wrappers)              │
│  • Compute/Transform Services                                │
└─────────────────────────────────────────────────────────────┘
```

---

## 📚 Categories & Features

### 1. **Service Classification & Service Types**

Clear separation between orchestration and business logic with type-safe abstractions.

**Service Types:**
- **Control Plane Services** - Workflow orchestration, saga management, coordination
- **Data Plane Services** - Business logic execution, external calls, side effects
- **Virtual Objects** - Stateful, key-addressable services with single-writer semantics
- **Workflows** - Long-running orchestrations with durable promises and human-in-the-loop

**Client Types:**
- `ServiceClient[I, O]` - For stateless services
- `ObjectClient[I, O]` - For Virtual Objects (key-based addressing)
- `WorkflowClient[I, O]` - For workflow lifecycle (Submit, Attach, Signal)

📖 Service Patterns Guide

---

### 2. **Durable Saga Framework**

Production-ready distributed transaction compensation with automatic retry, dead-letter queue, and compensation strategies.

**Features:**
- Pre-action compensation registration (enforced)
- Automatic LIFO rollback on errors
- Exponential backoff with configurable retry policies
- Dead-letter queue for irrecoverable failures
- Idempotent compensation validation helpers
- Multiple compensation strategies (all, completed, best-effort, until-success)

**Example:**
```go
saga := NewSaga(ctx, "payment-flow", nil)
saga.Register("charge_card", func(rc restate.RunContext, payload []byte) error {
    return refundCard(payload) // Must be idempotent!
})

// Compensation persisted BEFORE action
saga.Add("charge_card", chargeData, false)

// If error occurs, saga automatically compensates
defer saga.CompensateIfNeeded(&err)
```

📖 Saga Guide

---

### 3. **Type-Safe State Management**

Runtime-enforced read/write permissions with compile-time type safety.

**Features:**
- `State[T]` - Type-safe state accessor with runtime validation
- Read-only contexts (ObjectSharedContext, WorkflowSharedContext) reject writes
- Exclusive contexts (ObjectContext, WorkflowContext) allow writes
- Automatic error handling for invalid context usage

**Example:**
```go
// Read-safe from any context
balance := NewState[int](ctx, "balance")
value, err := balance.Get()

// Write requires exclusive context
if err := balance.Set(1000); err != nil {
    // Returns error if called from shared context
}
```

📖 Type Safety Guide

---

### 4. **Workflow Automation & Retention**

Utilities for long-running workflows with state retention policies.

**Features:**
- `WorkflowTimer` - Durable timers and sleep utilities
- `PromiseRacer` - Race promises against timeouts
- `WorkflowLoop` - Safe looping with iteration limits
- `WorkflowStatus` - Progress tracking via shared handlers
- `WorkflowConfig` - State retention and cleanup policies

**Workflow Configuration:**
```go
cfg := WorkflowConfig{
    StateRetentionDays:      30,  // Restate retention limit
    AutoCleanupOnCompletion: true, // Purge state on success
    MaxStateSizeBytes:       1_000_000, // Warn threshold
}
```

📖 Workflow Automation Guide

---

### 5. **Idempotency Key Management**

Auto-detection of unnecessary idempotency keys with validation and framework policy controls.

**Key Principle:** Use idempotency keys for **cross-invocation protection**, not within journaled handlers.

**When to use:**
✅ External calls (ingress)  
✅ Cross-handler attach semantics  
✅ Deduplication across invocations  

**When NOT to use:**
❌ Same-handler execution (journaling provides guarantees)  
❌ Sequential calls within same handler  
❌ Fire-and-forget Send within handler  

**Auto-Detection:**
```go
// Framework detects and warns about unnecessary keys
client.Call(ctx, input, CallOption{
    IdempotencyKey: "unnecessary", // ⚠️ Logged as redundant
})
```

📖 Idempotency Guide

---

### 6. **Side Effects (Run Wrappers)**

Context-capture prevention and retry utilities for durable side effects.

**Features:**
- `RunDo[T]` - Execute side effects with result
- `RunDoVoid` - Execute void side effects
- `RunWithRetry` - Automatic retry with exponential backoff
- `RunAsync` / `RunAsyncWithRetry` - Asynchronous execution
- Anti-pattern guards (prevents accidental context capture)

**Example:**
```go
// ✅ Correct: Only uses RunContext inside Run block
user, err := RunDo(ctx, func(rc restate.RunContext) (User, error) {
    return fetchUserFromDB(userID) // External call
}, restate.WithName("fetch-user"))

// ❌ Wrong: Captures outer ctx (causes non-determinism)
// restate.Run(ctx, func(rc restate.RunContext) {
//     ctx.Sleep(time.Second) // ❌ Uses outer ctx!
// })
```

📖 Run Side Effects Guide

---

### 7. **Concurrency Patterns**

Type-safe concurrent execution with deterministic coordination.

**Features:**
- `RequestFuture` - Start concurrent calls, wait later
- `restate.Wait()` - Wait for all futures (fan-in)
- `restate.WaitFirst()` - Race multiple futures
- `RunAsync` - Concurrent side effects
- Iterator-based result collection (no channels needed)

**Example:**
```go
// Fan-out: Start concurrent calls
fut1 := inventoryClient.RequestFuture(ctx, "product-1", req)
fut2 := paymentClient.RequestFuture(ctx, "account-1", req)

// Fan-in: Collect results deterministically
for fut, err := range restate.Wait(ctx, fut1, fut2) {
    if err != nil {
        return err
    }
    // Process result
}
```

📖 Concurrency Patterns Guide

---

### 8. **Security & Validation**

Cryptographic request validation and input/output validation.

**Security Features:**
- Ed25519 signature verification
- HTTPS enforcement
- Origin whitelisting
- Request replay protection
- Configurable validation modes (strict, permissive, disabled)

**Input Validation:**
```go
type OrderRequest struct {
    Amount   float64 `validate:"required,gt=0"`
    Quantity int     `validate:"required,min=1,max=100"`
}

// Automatic validation with framework
if err := ValidateInput(req); err != nil {
    return restate.TerminalError(err, 400)
}
```

📖 Security Guide

---

### 9. **Ingress Client (External Calls)**

Type-safe HTTP client for calling Restate from outside handler context.

**Client Types:**
- `IngressClient` - Base HTTP client with auth
- `IngressServiceClient` - For stateless services
- `IngressObjectClient` - For Virtual Objects
- `IngressWorkflowClient` - For workflows

**Usage:**
```go
// External application calling Restate
ingressClient := NewIngressClient("http://localhost:8080", "auth-key")

// Call service from outside
svcClient := IngressService[Order, OrderResult](
    ingressClient, "OrderService", "create",
)
result, err := svcClient.Call(context.Background(), order, 
    IngressCallOption{
        IdempotencyKey: "order-123", // ✅ Required for ingress
    },
)
```

⚠️ **Critical:** Never use `IngressClient` inside a Restate handler - bypasses journaling!

📖 Ingress Client Guide

---

### 10. **Observability & Metrics**

Prometheus-compatible metrics, OpenTelemetry tracing, and custom hooks.

**Features:**
- `MetricsCollector` - Counters, gauges, histograms
- `TracingContext` - OpenTelemetry span creation
- `ObservabilityHooks` - Custom event callbacks
- `InstrumentedServiceClient` - Automatic observability for calls

**Example:**
```go
metrics := NewMetricsCollector()
tracing := NewTracingContext(ctx)
hooks := DefaultObservabilityHooks(logger)

// Wrap client with automatic observability
instrumentedClient := NewInstrumentedClient(
    client, metrics, tracing, hooks,
)

// All calls automatically traced and metered
result, err := instrumentedClient.Call(ctx, req)

// Export metrics for Prometheus
http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
    json.NewEncoder(w).Encode(metrics.GetMetrics())
})
```

📖 Observability Guide

---

### 11. **Framework Policy & Guardrails**

Unified policy control for all runtime validations.

**Policy Modes:**
- `PolicyStrict` - Fail-fast on violations (CI/production)
- `PolicyWarn` - Log warnings but continue (development)
- `PolicyDisabled` - Skip validation (testing only)

**Configuration:**
```go
// Environment-based auto-detection
export RESTATE_FRAMEWORK_POLICY=strict  # CI=true defaults to strict

// Or set programmatically
SetFrameworkPolicy(PolicyStrict)
```

**Guardrail Coverage:**
- Idempotency key validation
- State write permission checks
- Security validation
- Saga compensation registration
- Context usage validation

📖 Framework Policy

---

### 12. **Microservices Orchestration**

High-level patterns for coordinating multiple services.

**Patterns:**
- **Saga Pattern** - Distributed transactions with compensation
- **Circuit Breaker** - Fault tolerance for external services
- **Request Aggregation** - Combine multiple service calls
- **Service Mesh Integration** - Distributed tracing context propagation
- **Human-in-the-Loop** - Approval workflows with awakeables

📖 Microservices Orchestration Guide

---

## 🚀 Quick Start

### Installation

```go
import framework "github.com/your-org/rea-framework"
```

### Basic Service

```go
type GreetingService struct{}

func (GreetingService) Greet(ctx restate.Context, name string) (string, error) {
    return fmt.Sprintf("Hello, %s!", name), nil
}

func main() {
    server := restate.NewRestate()
    server.Bind(restate.Reflect(GreetingService{}))
    server.Start(context.Background(), ":9080")
}
```

### Workflow with Saga

```go
type OrderWorkflow struct{}

func (OrderWorkflow) Run(ctx restate.WorkflowContext, order Order) (err error) {
    saga := framework.NewSaga(ctx, "order-flow", nil)
    defer saga.CompensateIfNeeded(&err)
    
    // Register compensations
    saga.Register("charge_payment", refundPayment)
    saga.Add("charge_payment", order.PaymentData, false)
    
    // Call services
    inventoryClient := framework.ServiceClient[Item, bool]{
        ServiceName: "Inventory",
        HandlerName: "Reserve",
    }
    
    available, err := inventoryClient.Call(ctx, order.Item)
    if err != nil || !available {
        return err // Saga auto-compensates
    }
    
    return nil
}
```

### Virtual Object

```go
type CartObject struct{}

func (CartObject) AddItem(ctx restate.ObjectContext, item Item) error {
    cartState := framework.NewState[Cart](ctx, "cart")
    
    cart, err := cartState.Get()
    if err != nil {
        cart = Cart{Items: []Item{}}
    }
    
    cart.Items = append(cart.Items, item)
    return cartState.Set(cart)
}
```

---

## 📖 Complete Documentation

### Core Concepts
- [Service Patterns Guide](file:///home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/SERVICE_PATTERNS_GUIDE.MD)
- [Service Clients Guide](file:///home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/SERVICE_CLIENTS_GUIDE.MD)
- [Type Safety Guide](file:///home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/TYPE_SAFETY_GUIDE.MD)

### Workflows & Orchestration
- [Workflow Automation Guide](file:///home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/WORKFLOW_AUTOMATION_GUIDE.MD)
- [Saga Guide](file:///home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/SAGA_GUIDE.MD)
- [Enhanced Saga Guide](file:///home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/ENHANCED_SAGA_GUIDE.MD)
- [Microservices Orchestration](file:///home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/MICROSERVICES_ORCHESTRATION_GUIDE.MD)

### Concurrency & Side Effects
- [Concurrency Patterns Guide](file:///home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/CONCURRENCY_PATTERNS_GUIDE.MD)
- [Run Side Effects Guide](file:///home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/RUN_SIDE_EFFECTS_GUIDE.MD)

### Best Practices
- [Idempotency Guide](file:///home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/IDEMPOTENCY_GUIDE.MD)
- [Validation Guide](file:///home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/VALIDATION_GUIDE.MD)
- [Security Guide](file:///home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/SECURITY_GUIDE.MD)

### External Integration
- [Ingress Client Guide](file:///home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/INGRESS_CLIENT_GUIDE.MD)
- [Ingress Type Safety](file:///home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/INGRESS_CLIENT_TYPE_SAFETY.MD)

### Observability
- [Observability Guide](file:///home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/OBSERVABILITY_GUIDE.MD)

### Configuration & Policy
- [Framework Policy](file:///home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/FRAMEWORK_POLICY.MD)
- [Workflow Retention Guide](file:///home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/WORKFLOW_RETENTION_GUIDE.MD)

### Reference
- [DOS and DON'Ts](file:///home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/DOS_DONTS_MEGA.MD)


---

## ✅ Best Practices Summary

### ✅ DO

1. **Use type-specific clients** - `ObjectClient` for Virtual Objects, `WorkflowClient` for workflows
2. **Wrap external calls in `RunDo`** - Ensures determinism and durability
3. **Register saga compensations BEFORE actions** - Framework enforces this
4. **Use idempotency keys for external/ingress calls** - Cross-invocation protection
5. **Use framework policy in CI** - `export RESTATE_FRAMEWORK_POLICY=strict`
6. **Instrument service clients** - Automatic observability with `InstrumentedServiceClient`

### ❌ DON'T

1. **Don't use `IngressClient` inside handlers** - Bypasses journaling, breaks durability
2. **Don't use idempotency keys within same handler** - Redundant, journaling provides guarantees
3. **Don't capture outer context in `restate.Run`** - Use `RunDo` to prevent mistakes
4. **Don't use `time.Now()` or `rand` directly** - Use deterministic helpers
5. **Don't sleep in exclusive object handlers** - Blocks all requests to that key
6. **Don't forget workflow retention limits** - Default 24 hours, configure appropriately

---

## 🎓 Anti-Patterns Prevented

The framework actively guards against common Restate anti-patterns:

| Anti-Pattern | Protection | Reference |
|-------------|-----------|-----------|
| Using outer context in `restate.Run` | `RunDo` / `RunDoVoid` wrappers | Run Guide |
| Unnecessary idempotency keys | Auto-detection with warnings | Idempotency Guide |
| State writes from read-only contexts | Runtime validation | Type Safety Guide |
| Missing saga compensations | Enforced registration | Saga Guide |
| Non-deterministic iteration | Ordered map helpers | Concurrency Guide |
| Blocking exclusive handlers | Static analysis warnings | Service Patterns

---

## 🏗️ Architecture Decisions

### Why Control Plane / Data Plane?

**Control Plane** (Orchestration):
- Manages "what to do" and "in what order"
- Uses state for coordination, not business data
- Calls Data Plane services for actual work
- Implements compensation logic

**Data Plane** (Execution):
- Implements "how to do it"
- Contains business logic and side effects
- No orchestration concerns
- Stateless where possible

This separation ensures:
- Clear responsibility boundaries
- Easier testing (mock data plane in control plane tests)
- Better observability (orchestration vs execution metrics)
- Simpler reasoning about failures

### Why Type-Specific Clients?

Before:
```go
// Unclear what this service is
client := ServiceClient[I, O]{...}
client.Call(ctx, input) // Missing key for objects!
```

After:
```go
// Clear this is a Virtual Object
client := ObjectClient[I, O]{...}
client.Call(ctx, key, input) // Key required at compile time
```

**Benefits:**
- Compiler enforces correct parameters
- Self-documenting code
- Fewer runtime errors
- Better IDE autocomplete


---

## 🤝 Contributing

This framework embodies best practices learned from production Restate deployments. Contributions should:
1. Follow the Control Plane / Data Plane separation
2. Add guardrails for new anti-patterns discovered
3. Include comprehensive documentation
4. Provide both positive and negative examples
---

## 📝 License

MIT License

---

## 🔗 Resources

- [Restate Documentation](https://docs.restate.dev/)
- [Restate Go SDK](https://github.com/restatedev/sdk-go)


---

**Made with ❤️ for building reliable distributed systems**
