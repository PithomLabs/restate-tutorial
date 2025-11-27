## Based off rea2/kimi framework.go and main.go

### Prompt: add examples to attached main.go illustrating stateful services using durable object like cart services using the attached framework.go following constraints from previously attached 2 markdown files (agents.md and dos_donts_mega.md)

# High-Level Framework Extension for Control Plane and Data Plane Services

Based on the Restate SDK documentation and the existing framework, here's how I would extend the abstraction to minimize boilerplate and clearly separate concerns:

## Architecture Overview

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

## 1. Control Plane Abstractions

### 1.1 Orchestrator Base

```go
// Orchestrator provides common patterns for control plane services
type Orchestrator struct {
    ctx  restate.WorkflowContext
    saga *SagaFramework
    log  *slog.Logger
}

// Key capabilities:
// - Automatic saga setup and cleanup
// - Typed service invocation helpers
// - Promise/Awakeable management
// - Parallel execution patterns (fan-out/fan-in)
// - Timeout and retry configuration
```

**Why:** Workflows and Virtual Objects in control plane mode need:
- State management (exclusive handlers)
- Coordination primitives (promises, awakeables)
- Compensation logic (sagas)
- Deterministic concurrency (Wait/WaitFirst)

### 1.2 Coordinator for Virtual Objects

```go
// Coordinator wraps ObjectContext for stateful orchestration
type Coordinator struct {
    ctx   restate.ObjectContext
    key   string
    state StateManager // typed state access
    log   *slog.Logger
}

// Key capabilities:
// - Type-safe state access (Get/Set/Clear)
// - Service call helpers (Request/RequestFuture/Send)
// - Awakeable management for external coordination
// - Automatic state persistence
```

**Why:** Virtual Objects acting as coordinators need:
- Single-writer state consistency
- Entity-scoped coordination (per key)
- Blocking protection (avoid sleep/awakeable in exclusive handlers)

## 2. Data Plane Abstractions

### 2.1 Service Base

```go
// Service wraps Context for stateless business logic
type Service struct {
    ctx restate.Context
    log *slog.Logger
}

// Key capabilities:
// - Run/RunAsync wrappers with naming
// - Error handling (Terminal vs Retryable)
// - Logging helpers
// - UUID/Time deterministic helpers
```

**Why:** Basic Services need:
- Durable execution of side effects (Run blocks)
- No state management
- Simple request-response or fire-and-forget patterns

### 2.2 ExternalIntegration

```go
// ExternalIntegration wraps common patterns for calling external systems
type ExternalIntegration struct {
    service  *Service
    timeout  time.Duration
    retries  int
}

// Key capabilities:
// - Automatic Run wrapping for HTTP/DB calls
// - Configurable timeouts and retries
// - Response caching/deduplication
// - Circuit breaker pattern
```

**Why:** External calls must:
- Always be wrapped in `restate.Run` for determinism
- Handle transient failures with retries
- Support timeout patterns (race against sleep)

### 2.3 AsyncTask

```go
// AsyncTask for background/concurrent work
type AsyncTask struct {
    service *Service
}

// Key capabilities:
// - RunAsync wrapper
// - Future management
// - Parallel execution helpers (fan-out)
// - Result aggregation
```

**Why:** Async operations need:
- Non-blocking execution (RequestFuture, RunAsync)
- Deterministic waiting (Wait, WaitFirst)
- Proper error propagation

## 3. Key Design Patterns

### 3.1 Control Plane → Data Plane Invocation

```go
// Orchestrator calling data plane service
func (o *Orchestrator) InvokeService[I, O any](
    serviceName string,
    handlerName string,
    input I,
) (O, error) {
    // Enforces proper context usage
    // Automatic logging
    // Error wrapping
    return restate.Service[O](o.ctx, serviceName, handlerName).Request(input)
}

// Parallel invocation with timeout
func (o *Orchestrator) InvokeParallel[I, O any](
    calls []ServiceCall[I, O],
    timeout time.Duration,
) ([]O, error) {
    // Fan-out pattern
    // Automatic Wait() coordination
    // Timeout via race with After()
}
```

### 3.2 Data Plane Service Template

```go
// ExternalCall wraps restate.Run with best practices
func (s *Service) ExternalCall[T any](
    name string,
    fn func(ctx restate.RunContext) (T, error),
) (T, error) {
    return restate.Run(s.ctx, fn, restate.WithName(name))
}

// AsyncExternalCall for non-blocking
func (s *Service) AsyncExternalCall[T any](
    name string,
    fn func(ctx restate.RunContext) (T, error),
) restate.RunAsyncFuture[T] {
    return restate.RunAsync(s.ctx, fn, restate.WithName(name))
}
```

### 3.3 State Management Enforcement

```go
// StateManager enforces read/write rules
type StateManager struct {
    ctx restate.ObjectSharedContext // base type
}

// Get is always safe
func (sm *StateManager) Get[T any](key string) (T, error)

// Set requires exclusive context (compile-time check via type system)
func (sm *StateManager) Set[T any](key string, value T) error {
    if writerCtx, ok := sm.ctx.(restate.ObjectContext); ok {
        restate.Set(writerCtx, key, value)
        return nil
    }
    return restate.TerminalError(fmt.Errorf("Set requires exclusive context"))
}
```

## 4. Service Classification Rules

### Control Plane Services Use:
- ✅ `WorkflowContext` (workflows)
- ✅ `ObjectContext` (exclusive virtual objects)
- ✅ `WorkflowSharedContext` (signals/queries)
- ✅ `ObjectSharedContext` (concurrent reads)
- ✅ Saga framework
- ✅ Promises (workflows) or Awakeables (virtual objects)
- ✅ Request/RequestFuture/Send to data plane
- ❌ **NO** direct `restate.Run` (orchestration, not execution)

### Data Plane Services Use:
- ✅ `Context` (stateless services)
- ✅ `ObjectSharedContext` (read-only virtual objects)
- ✅ `restate.Run` / `restate.RunAsync` (side effects)
- ✅ Simple request-response patterns
- ❌ **NO** Workflows
- ❌ **NO** State mutation (unless explicit read-only query service)
- ❌ **NO** Saga logic (orchestration concern)

## 5. Example Usage

### Control Plane (Orchestrator)

```go
type OrderOrchestrator struct{}

func (o *OrderOrchestrator) Run(ctx restate.WorkflowContext, req Order) (err error) {
    orch := framework.NewOrchestrator(ctx, "order-flow")
    defer orch.CompensateIfNeeded(&err)
    
    // Register compensations
    orch.RegisterCompensation("payment", refundPayment)
    
    // Call data plane services
    inv, err := orch.InvokeService[Item, bool]("Inventory", "Check", req.Item)
    if err != nil || !inv { return err }
    
    // Parallel calls with timeout
    results, err := orch.InvokeParallel([]Call{
        {Service: "Payment", Handler: "Process", Input: req.Payment},
        {Service: "Shipping", Handler: "Reserve", Input: req.Address},
    }, 30*time.Second)
    
    return nil
}
```

### Data Plane (Service)

```go
type PaymentService struct{}

func (p *PaymentService) Process(ctx restate.Context, req Payment) (Result, error) {
    svc := framework.NewService(ctx)
    
    // External call (automatically wrapped in Run)
    result, err := svc.ExternalCall("payment.gateway", func(rc restate.RunContext) (Result, error) {
        return callPaymentGateway(req)
    })
    
    return result, err
}
```

## 6. Benefits

1. **Clear Separation**: Control plane handles "what to do," data plane handles "how to do it"
2. **Boilerplate Reduction**: Common patterns (Run wrapping, saga setup, state access) are encapsulated
3. **Type Safety**: Enforces correct context usage at compile time where possible
4. **Best Practices**: Framework guides users toward correct patterns (e.g., no Run in orchestrators)
5. **Error Prevention**: Catches common mistakes (state writes in shared contexts, blocking in exclusive handlers)
6. **Observability**: Automatic logging and tracing built into framework methods

This design follows the Restate philosophy: **orchestration in workflows/virtual objects, execution in services**, while making the distinction explicit and easy to follow.

