# DOS_DONTS MEGA - Framework Edition

This document rewrites all examples from DOS_DONTS_MEGA.MD using the custom framework layer built on top of the standard Restate Go SDK.

## Key Differences from Standard Restate Go SDK

The **framework** adds a high-level abstraction layer that:

1. **Enforces Best Practices**: Automatically prevents common mistakes (e.g., writing state from shared context)
2. **Reduces Boilerplate**: Provides helpers for common patterns (sagas, workflows, external signals)
3. **Type Safety**: Strong compile-time guarantees through generic wrappers
4. **Explicit Context Separation**: Clear distinction between Control Plane (orchestration) and Data Plane (business logic)
5. **Built-in Observability**: Integrated metrics, tracing, and logging utilities
6. **Saga Framework**: Durable compensation with automatic retry, DLQ, and rollback strategies
7. **Workflow Automation**: Pre-built utilities for timers, loops, promise racing, and status tracking

---

## 1. Wrap Non-Deterministic Operations in `restate.Run`

### Standard SDK

```go
// Standard SDK approach
result, err := restate.Run(ctx, func(ctx restate.RunContext) (string, error) {
    return doDbRequest()
})
if err != nil {
    return err
}
```

### Framework Approach

```go
// Framework provides RunWithRetry with automatic retry logic
cfg := framework.DefaultRunConfig("db-request")
result, err := framework.RunWithRetry(ctx, cfg, func(rc restate.RunContext) (string, error) {
    return doDbRequest()
})
if err != nil {
    return err
}

// Or use DataPlaneService for pure business logic
dbService := framework.NewStatelessService("db-request", func(rc restate.RunContext, input string) (string, error) {
    return doDbRequest()
})
result, err := dbService.Execute(ctx, "input")
```

**Difference**: Framework adds automatic retry with exponential backoff, terminal error detection, and better logging. The `RunWithRetry` function handles transient errors gracefully.

---

## 2. Use Context Logger

### Standard SDK

```go
ctx.Log().Info("This will not be printed again during replays")
```

### Framework Approach

```go
// Same as standard SDK - framework uses the same context logger
ctx.Log().Info("This will not be printed again during replays")

// But framework also provides structured observability hooks
hooks := framework.DefaultObservabilityHooks(ctx.Log())
hooks.OnInvocationStart("MyService", "MyHandler", input)
```

**Difference**: Framework adds optional observability hooks for custom monitoring, metrics collection, and distributed tracing integration.

---

## 3. Generate Deterministic IDs and Time

### Standard SDK

```go
uuid := restate.UUID(ctx)
```

### Framework Approach

```go
// Framework provides helpers for common deterministic operations
helpers := framework.NewDeterministicHelpers(ctx)
uuid := helpers.UUID()
randInt := helpers.RandInt(1, 100)
randFloat := helpers.RandFloat()

// For deterministic time capture
timeHelper := framework.NewTime(ctx)
now := timeHelper.Now() // Captured once, deterministic on replay
duration := timeHelper.Since(startTime)
```

**Difference**: Framework wraps all deterministic operations in a consistent API with additional utilities like random number generation and time helpers that automatically use `restate.Run` for determinism.

---

## 4. Parallelize Tasks Safely (Fan-Out/Fan-In)

### Standard SDK

```go
callFuture1 := restate.Service[string](ctx, "MyService", "MyHandler").RequestFuture("hi")
callFuture2 := restate.Service[string](ctx, "MyService", "MyHandler").RequestFuture("hi again")

for fut, err := range restate.Wait(ctx, callFuture1, callFuture2) {
    // process results
}
```

### Framework Approach

```go
// Framework uses the same underlying mechanism but with service abstractions
client1 := framework.NewServiceClient[string, string](ctx, "MyService", "MyHandler")
client2 := framework.NewServiceClient[string, string](ctx, "MyService", "MyHandler")

future1 := client1.CallAsync("hi")
future2 := client2.CallAsync("hi again")

// Use standard SDK wait mechanisms
for fut, err := range restate.Wait(ctx, future1, future2) {
    // process results
}

// Or use ForEach for batch operations (in workflows)
items := []string{"task1", "task2", "task3"}
err := framework.ForEach(workflowCtx, items, func(item string, index int) error {
    // Process each item durably
    return nil
})
```

**Difference**: Framework provides `ForEach` helper for workflow-based batch processing with automatic error handling and logging.

---

## 5. Implement Timeouts and Racing

### Standard SDK

```go
sleepFuture := restate.After(ctx, 30*time.Second)
callFuture := restate.Service[string](ctx, "MyService", "MyHandler").RequestFuture("hi")
fut, err := restate.WaitFirst(ctx, sleepFuture, callFuture)
// switch on fut to determine winner
```

### Framework Approach

```go
// Framework provides dedicated promise/awakeable racing utilities
awakeable := framework.WaitForExternalSignal[string](ctx)
result, timedOut, err := framework.RaceAwakeableWithTimeout(
    workflowCtx,
    awakeable,
    30*time.Second,
    "default-value",
)

if timedOut {
    // Handle timeout
}

// Or race a promise against timeout
result, err := framework.RacePromiseWithTimeout[string](
    workflowCtx,
    "approval-signal",
    24*time.Hour,
)

if result.TimedOut {
    // Handle timeout
} else {
    // Use result.Value
}
```

**Difference**: Framework provides typed, ergonomic helpers specifically for racing awakeables and promises, with built-in timeout handling and clear result types. The `PromiseRaceResult` struct explicitly indicates what happened.

---

## 6. Manage State in Virtual Objects/Workflows

### Standard SDK

```go
restate.Set(ctx, "my-key", "my-new-value")
myNumber, err := restate.Get[int](ctx, "my-number-key")
restate.ClearAll(ctx)
```

### Framework Approach

```go
// Framework provides type-safe state wrappers with runtime guards
state := framework.NewState[string](ctx, "my-key")

// Get is safe from any context
value, err := state.Get()

// Set is only allowed from exclusive contexts (enforced at runtime)
err = state.Set("my-new-value") // Returns error if called from shared context

// Clear operations
err = state.Clear()
err = framework.ClearAll(ctx) // Also runtime-guarded
```

**Difference**: Framework enforces read-only vs read-write context separation at runtime, preventing accidental state mutations from shared handlers. The SDK allows this but will fail at runtime; the framework makes it explicit with error returns.

---

## 7. Use Awakeables vs. Durable Promises

### Standard SDK

```go
// Awakeable for external systems
awakeable := restate.Awakeable[string](ctx)
awakeableID := awakeable.Id()
// ... pass ID to external system ...
result, err := awakeable.Result()

// Promise for workflow signaling
promise := restate.Promise[string](workflowCtx, "review")
err := promise.Resolve("approved")
```

### Framework Approach

```go
// Awakeable with clear naming
awakeable := framework.WaitForExternalSignal[string](ctx)
awakeableID := awakeable.Id()
// ... pass ID to external system ...

// Resolve from callback handler
framework.ResolveExternalSignal(ctx, awakeableID, "response")
// Or reject
framework.RejectExternalSignal(ctx, awakeableID, errors.New("failed"))

// Promise for internal workflow signaling
promise := framework.GetInternalSignal[string](workflowSharedCtx, "review")
result, err := promise.Result()
```

**Difference**: Framework uses semantic naming that makes the intent clear: `WaitForExternalSignal` vs `GetInternalSignal`. This prevents confusion about when to use awakeables (external) vs promises (internal workflow).

---

## 8. Throw Terminal Errors

### Standard SDK

```go
return restate.TerminalError(fmt.Errorf("Something went wrong."), 500)
```

### Framework Approach

```go
// Same as standard SDK
return restate.TerminalError(fmt.Errorf("Something went wrong."), 500)

// Framework's RunWithRetry automatically detects terminal errors and stops retrying
cfg := framework.DefaultRunConfig("operation")
result, err := framework.RunWithRetry(ctx, cfg, func(rc restate.RunContext) (string, error) {
    if invalidInput {
        // This will stop retries
        return "", restate.TerminalError(errors.New("invalid input"), 400)
    }
    return doOperation()
})
```

**Difference**: Framework utilities like `RunWithRetry` are aware of terminal errors and won't retry them, unlike raw `restate.Run` which requires manual handling.

---

## 9. Implement Sagas (Distributed Compensation)

### Standard SDK

```go
func (BookingWorkflow) Run(ctx restate.Context, req BookingRequest) (err error) {
    var compensations []func() (restate.Void, error)
    defer func() {
        if err != nil {
            for _, compensation := range slices.Backward(compensations) {
                if _, compErr := compensation(); compErr != nil {
                    err = compErr
                }
            }
        }
    }()

    compensations = append(compensations, func() (restate.Void, error) {
        // Compensation logic (CancelFlight)
        return restate.Void{}, nil
    })

    if _, err = restate.Run(ctx, /* BookFlight logic */); err != nil {
        return err
    }

    return nil
}
```

### Framework Approach

```go
func (MyWorkflow) Run(ctx restate.WorkflowContext, req BookingRequest) (err error) {
    // Create control plane service with built-in saga
    cp := framework.NewControlPlaneService(ctx, "booking-saga", "booking")

    // Register all compensation handlers upfront
    cp.RegisterCompensation("book-flight", func(rc restate.RunContext, payload []byte) error {
        return cancelFlight(rc, payload)
    })

    cp.RegisterCompensation("book-hotel", func(rc restate.RunContext, payload []byte) error {
        return cancelHotel(rc, payload)
    })

    // Execute within saga context (automatic compensation on error)
    return cp.Orchestrate(func() error {
        // Add compensation BEFORE action
        err := cp.AddCompensationStep("book-flight", req.FlightDetails, true)
        if err != nil {
            return err
        }

        // Execute action
        _, err = restate.Run(ctx, func(rc restate.RunContext) (string, error) {
            return bookFlight(rc, req.FlightDetails)
        })
        if err != nil {
            return err // Triggers automatic compensation
        }

        // Next step
        err = cp.AddCompensationStep("book-hotel", req.HotelDetails, true)
        if err != nil {
            return err
        }

        _, err = restate.Run(ctx, func(rc restate.RunContext) (string, error) {
            return bookHotel(rc, req.HotelDetails)
        })

        return err
    })
}
```

**Difference**: Framework provides a full saga framework with:
- Automatic compensation execution in LIFO order
- Exponential backoff retry for compensation failures
- Dead Letter Queue (DLQ) for irrecoverable errors
- Deduplication of compensation steps
- Multiple compensation strategies (best effort, until success, etc.)
- State persistence for compensation metadata

---

## 10. Workflow with Durable Promises (Signal Pattern)

### Standard SDK

```go
func (SignupWorkflow) Run(ctx restate.WorkflowContext, user User) (bool, error) {
    // ... send verification email step ...

    // Wait for user to click verification link
    clickedSecret, err := restate.Promise[string](ctx, "email-verified").Result()
    if err != nil {
        return false, err
    }

    return clickedSecret == secret, nil
}

func (SignupWorkflow) VerifyEmail(ctx restate.WorkflowSharedContext, req VerifyEmailRequest) error {
    // Resolve the promise to continue the main workflow
    return restate.Promise[string](ctx, "email-verified").Resolve(req.Secret)
}
```

### Framework Approach

```go
func (SignupWorkflow) Run(ctx restate.WorkflowContext, user User) (bool, error) {
    // ... send verification email step ...

    // Framework provides semantic naming for internal signals
    emailPromise := framework.GetInternalSignal[string](ctx, "email-verified")

    // Can race against timeout using framework utility
    result, err := framework.RacePromiseWithTimeout[string](
        ctx,
        "email-verified",
        24*time.Hour,
    )

    if err != nil {
        return false, err
    }

    if result.TimedOut {
        return false, restate.TerminalError(errors.New("email verification timeout"), 408)
    }

    return result.Value == secret, nil
}

func (SignupWorkflow) VerifyEmail(ctx restate.WorkflowSharedContext, req VerifyEmailRequest) error {
    // Resolve internal signal
    promise := framework.GetInternalSignal[string](ctx, "email-verified")
    return promise.Resolve(req.Secret)
}
```

**Difference**: Framework adds built-in timeout racing for promises and uses semantic naming (`GetInternalSignal`) to clarify this is for workflow-internal communication.

---

## 11. Durable Timers and Loops

### Standard SDK

```go
clickedPromise := restate.Promise[string](ctx, "email-verified")
verificationTimeoutFuture := restate.After(ctx, 24*time.Hour)

for {
    reminderTimerFuture := restate.After(ctx, 15*time.Second)
    resFut, err := restate.WaitFirst(ctx, clickedPromise, reminderTimerFuture, verificationTimeoutFuture)
    // ... switch on resFut to handle promise result, reminder email, or timeout ...
}
```

### Framework Approach

```go
// Framework provides WorkflowTimer utilities
timer := framework.NewWorkflowTimer(ctx)

// Sleep with better logging
err := timer.Sleep(15 * time.Second)

// Or sleep until specific time
err := timer.SleepUntil(targetTime)

// Framework provides loop utilities with safety guards
loop := framework.NewWorkflowLoop(ctx, 10000) // max 10k iterations

err := loop.While(
    func() (bool, error) {
        // Check if promise is completed (non-blocking peek)
        promise := framework.GetInternalSignal[string](ctx, "email-verified")
        completed, _ := promise.Peek()
        return !completed, nil
    },
    func(iteration int) error {
        ctx.Log().Info("sending reminder", "iteration", iteration)

        // Sleep between reminders
        timer := framework.NewWorkflowTimer(ctx)
        return timer.Sleep(15 * time.Second)
    },
)

// Or use retry with backoff
loop.Retry(func(attempt int) error {
    return sendEmail(ctx)
}, 3, 1*time.Second)
```

**Difference**: Framework provides:
- `WorkflowTimer` for cleaner timer operations
- `WorkflowLoop` with built-in safety limits to prevent infinite loops
- `While`, `Retry`, and `ForEach` loop constructs with automatic iteration counting
- Built-in logging and error handling

---

## 12. Workflow Status Tracking

### Standard SDK

```go
// Manual status tracking with state
restate.Set(ctx, "status", WorkflowStatus{
    Phase: "processing",
    Progress: 0.5,
})

// Query from shared handler
status, err := restate.Get[WorkflowStatus](sharedCtx, "status")
```

### Framework Approach

```go
// In the exclusive run handler
err := framework.UpdateStatus(ctx, "workflow_status", framework.StatusData{
    Phase:          "processing",
    Progress:       0.5,
    CurrentStep:    "payment",
    CompletedSteps: []string{"validation", "inventory"},
    IsComplete:     false,
    Metadata:       map[string]interface{}{"order_id": "12345"},
})

// In shared handler
statusTracker := framework.NewWorkflowStatus(sharedCtx, "workflow_status")
status, err := statusTracker.GetStatus()
fmt.Printf("Progress: %.0f%%, Phase: %s\n", status.Progress*100, status.Phase)
```

**Difference**: Framework provides structured status tracking with:
- Standardized `StatusData` format
- Automatic timestamp management
- Progress tracking (0.0-1.0)
- Completed steps tracking
- Extensible metadata
- Type-safe access from shared context

---

## 13. Virtual Object State Management

### Standard SDK

```go
func (UserAccount) UpdateBalance(ctx restate.ObjectContext, amount float64) (float64, error) {
    balance, err := restate.Get[float64](ctx, "balance")
    // ... error handling ...

    newBalance := balance + amount
    if newBalance < 0.0 {
        return 0.0, restate.TerminalError(errors.New("insufficient funds"), 400)
    }

    restate.Set(ctx, "balance", newBalance)
    return newBalance, nil
}
```

### Framework Approach

```go
func (UserAccount) UpdateBalance(ctx restate.ObjectContext, amount float64) (float64, error) {
    // Framework provides type-safe state wrapper
    balanceState := framework.NewState[float64](ctx, "balance")

    balance, err := balanceState.Get()
    if err != nil {
        return 0.0, err
    }

    newBalance := balance + amount
    if newBalance < 0.0 {
        return 0.0, restate.TerminalError(errors.New("insufficient funds"), 400)
    }

    // Set is runtime-guarded (only works with exclusive context)
    err = balanceState.Set(newBalance)
    if err != nil {
        return 0.0, err
    }

    return newBalance, nil
}

// Read-only handler (shared context)
func (UserAccount) GetBalance(ctx restate.ObjectSharedContext) (float64, error) {
    balanceState := framework.NewState[float64](ctx, "balance")

    // Get works from any context
    balance, err := balanceState.Get()
    if err != nil {
        return 0.0, err
    }

    // This would return an error at runtime
    // err = balanceState.Set(100.0) // ERROR: Set called from read-only context

    return balance, nil
}
```

**Difference**: Framework's `NewState` wrapper:
- Enforces exclusive vs shared context rules at runtime with clear errors
- Provides type safety through generics
- Makes state mutations explicit
- Prevents accidental writes from shared handlers with helpful error messages

---

## 14. Idempotency Keys

### Standard SDK

```go
restate.ServiceSend(ctx, "MyService", "MyHandler").
    Send("Hi", restate.WithIdempotencyKey("my-idempotency-key"))
```

### Framework Approach

```go
// Framework provides deterministic idempotency key generation
cp := framework.NewControlPlaneService(ctx, "orchestrator", "order")

// Generate deterministic key using context UUID (same on replay)
idempotencyKey := cp.GenerateIdempotencyKey(ctx, "payment")

// Use with service calls
restate.ServiceSend(ctx, "PaymentService", "Charge").
    Send(chargeRequest, restate.WithIdempotencyKey(idempotencyKey))

// Or generate from business data (fully deterministic)
idempotencyKey := cp.GenerateIdempotencyKeyDeterministic("user123", "order456")
```

**Difference**: Framework provides:
- `GenerateIdempotencyKey`: Uses deterministic UUID seeded by invocation
- `GenerateIdempotencyKeyDeterministic`: Creates keys from business data
- Consistent key format with prefixes for organization
- Automatic deduplication in saga compensation steps

---

## 15. Human-in-the-Loop with Timeout

### Standard SDK

```go
awakeable := restate.Awakeable[bool](ctx)
awakeableID := awakeable.Id()

ctx.Log().Info("Awaiting approval", "awakeable_id", awakeableID)

timeoutFuture := restate.After(ctx, 24*time.Hour)
winner, err := restate.WaitFirst(ctx, awakeable, timeoutFuture)

if winner == timeoutFuture {
    return restate.TerminalError(errors.New("approval timeout"), 408)
}

approved, err := awakeable.Result()
return approved, err
```

### Framework Approach

```go
cp := framework.NewControlPlaneService(ctx, "approval-workflow", "approval")

approved, err := cp.AwaitHumanApproval(ctx, "approval-123", 24*time.Hour)
if err != nil {
    return err
}

if !approved {
    return restate.TerminalError(errors.New("approval denied"), 403)
}

// In callback handler
framework.ResolveExternalSignal(ctx, awakeableID, true)
```

**Difference**: Framework provides `AwaitHumanApproval` which:
- Combines awakeable creation, timeout racing, and logging
- Returns terminal error on timeout with appropriate HTTP status
- Clear boolean result for approval status
- Structured logging with approval ID and awakeable ID

---

## 16. Security and Request Validation

### Standard SDK

```go
// Manual request validation setup
server := restate.NewServer(/* ... */)
// Limited built-in security features
```

### Framework Approach

```go
// Production security config
securityConfig := framework.DefaultSecurityConfig()
securityConfig.SigningKeys = []ed25519.PublicKey{publicKey1, publicKey2}
securityConfig.RequireHTTPS = true
securityConfig.AllowedOrigins = []string{"https://restate.prod.example.com"}
securityConfig.RestrictPublicAccess = true
securityConfig.ValidationMode = framework.SecurityModeStrict

// Development config (permissive)
devConfig := framework.DevelopmentSecurityConfig()

// Apply to server (framework extension)
// This would be implemented in server setup
```

**Difference**: Framework adds comprehensive security configuration:
- Ed25519 signature validation
- HTTPS enforcement
- Origin allowlisting
- Public access restrictions
- Multiple validation modes (strict, permissive, disabled)
- Pre-configured production and development profiles

---

## Summary of Key Framework Benefits

### 1. **Safety Through Type System**
- Runtime guards prevent state mutations from shared contexts
- Generic wrappers provide compile-time type safety
- Explicit error returns for violations instead of panics

### 2. **Reduced Boilerplate**
- Saga framework eliminates manual compensation tracking
- Workflow utilities provide common patterns (timers, loops, racing)
- Structured observability reduces custom logging code

### 3. **Enforced Best Practices**
- Compensation-before-action pattern enforced by `SafeStep`
- Semantic naming prevents awakeable/promise confusion
- Built-in retry logic with terminal error detection

### 4. **Better Observability**
- Integrated metrics collection
- OpenTelemetry-compatible tracing
- Structured logging with context
- Customizable hooks for monitoring

### 5. **Production-Ready Features**
- DLQ for irrecoverable compensation failures
- Multiple compensation strategies
- Security configuration presets
- Comprehensive error handling

### 6. **Clear Separation of Concerns**
- Control Plane: Orchestration, sagas, human workflows
- Data Plane: Business logic, external calls
- Explicit context types prevent misuse

The framework doesn't replace the SDK—it builds on top of it to provide a more ergonomic, safer, and production-ready development experience while preserving the full power of Restate's durable execution model.
