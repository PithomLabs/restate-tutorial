# Concepts: Understanding Durable Execution

> **Learn the foundational concepts that make Restate different from traditional microservices**

## 🎯 What is Durable Execution?

### Traditional Microservices (The Problem)

In traditional microservices, when your code fails mid-execution:

```go
func ProcessOrder(orderID string) error {
    // Step 1: Reserve inventory ✅ Succeeds
    ReserveInventory(orderID)
    
    // Step 2: Charge payment ✅ Succeeds
    ChargePayment(orderID)
    
    // Step 3: Send confirmation ❌ CRASH! (service restarts)
    SendConfirmation(orderID)
    
    // When the service restarts, what happens?
    // - Do we retry from the beginning? (duplicate charge!)
    // - Do we skip completed steps? (how do we know which ones?)
    // - Manual cleanup? (error-prone and complex)
}
```

**Problems:**
- 😱 Lost execution state
- 🔄 Manual retry logic everywhere
- ❌ Risk of duplicate operations
- 🐛 Complex error handling
- 😓 Difficult to reason about

### Restate Solution: Durable Execution

With Restate, your code **automatically resumes from where it left off**:

```go
func (s *OrderService) ProcessOrder(ctx restate.Context, orderID string) error {
    // Step 1: Reserve inventory ✅ Journaled
    _, err := restate.Service[string](ctx, "Inventory", "Reserve").
        Request(orderID)
    if err != nil {
        return err
    }
    
    // Step 2: Charge payment ✅ Journaled
    _, err = restate.Service[string](ctx, "Payment", "Charge").
        Request(orderID)
    if err != nil {
        return err
    }
    
    // Step 3: Send confirmation ❌ CRASH!
    // Restate automatically replays from here ↓
    _, err = restate.Service[string](ctx, "Email", "Send").
        Request(orderID)
    return err
}
```

**How Restate Fixes This:**
1. **Journals** each completed step
2. **Replays** from the last successful step on retry
3. **Guarantees** exactly-once execution per step
4. **Automatic** - no manual retry logic needed!

## 🏛️ Core Concepts

### 1. The Restate Context

The `ctx restate.Context` is your gateway to durable operations:

```go
func MyHandler(ctx restate.Context, input string) (string, error) {
    // ctx provides:
    // - Durable service calls
    // - State management (for Virtual Objects)
    // - Logging that doesn't duplicate on replay
    // - Deterministic random/UUID generation
    
    ctx.Log().Info("Processing", "input", input) // Smart logging
    
    return "done", nil
}
```

**Key Point:** Always use `ctx` for operations you want to be durable!

### 2. Journaling and Replay

Restate maintains a **journal** of your handler's execution:

```
Invocation ID: 12345
┌─────────────────────────────────────┐
│ Journal Entry 1: Started handler    │
│ Journal Entry 2: Called Service A   │
│ Journal Entry 3: Result from A: "ok"│
│ Journal Entry 4: Called Service B   │
│ ... CRASH ...                       │
└─────────────────────────────────────┘

On Retry (automatic):
┌─────────────────────────────────────┐
│ ⏭️ Skip: Already completed entries  │
│ ⏯️ Resume: From Journal Entry 5     │
│ Journal Entry 5: Result from B: "ok"│
│ Journal Entry 6: Handler completed  │
└─────────────────────────────────────┘
```

**Benefits:**
- No duplicate operations
- Instant state recovery
- Exactly-once guarantees

### 3. Three Service Types

Restate offers three service types for different use cases:

#### Basic Service (Stateless)

```go
type GreetingService struct{}

// Stateless, high concurrency, no state
func (GreetingService) Greet(ctx restate.Context, name string) (string, error) {
    return fmt.Sprintf("Hello, %s!", name), nil
}
```

**Use When:**
- Stateless business logic
- API calls and transformations
- High concurrency needed
- No state between requests

#### Virtual Object (Stateful, Key-Based)

```go
type ShoppingCart struct{}

// Stateful per cart ID, single-writer per key
func (ShoppingCart) AddItem(ctx restate.ObjectContext, item Item) error {
    // Get state for THIS cart's key
    cart, _ := restate.Get[Cart](ctx, "cart")
    cart.Items = append(cart.Items, item)
    restate.Set(ctx, "cart", cart)
    return nil
}
```

**Use When:**
- Modeling entities (users, carts, sessions)
- Need consistent state per entity
- Single-writer consistency required

#### Workflow (Long-Running Orchestration)

```go
type SignupWorkflow struct{}

// Runs exactly once per workflow ID
func (SignupWorkflow) Run(ctx restate.WorkflowContext, user User) (bool, error) {
    // Send email
    SendEmail(user.Email)
    
    // Wait for user to click link (hours/days!)
    confirmed, _ := restate.Promise[bool](ctx, "email-confirmed").Result()
    
    return confirmed, nil
}
```

**Use When:**
- Multi-step processes
- Human-in-the-loop
- Long-running operations (hours/days)
- Need exactly-once guarantee

### Comparison Table

| Feature | Basic Service | Virtual Object | Workflow |
|---------|--------------|----------------|----------|
| **State** | None | Per key | Per workflow ID |
| **Concurrency** | Unlimited | Single-writer per key | One `run` handler |
| **Context Type** | `Context` | `ObjectContext` / `ObjectSharedContext` | `WorkflowContext` / `WorkflowSharedContext` |
| **Use Case** | API logic | Stateful entities | Orchestration |
| **Examples** | Email sender, calculator | User account, cart | Signup flow, approval |

## 🔄 Request-Response Communication

Restate acts as a reliable RPC framework:

```go
// Calling another service (durable)
result, err := restate.Service[OutputType](ctx, "ServiceName", "HandlerName").
    Request(inputData)
if err != nil {
    return err // Automatic retry on transient errors
}

// Use the result
fmt.Println(result)
```

**What Restate Does:**
1. **Journals** the call
2. **Proxies** the request to the target service
3. **Journals** the response
4. **Retries** on transient failures
5. **Replays** from journal on handler restart

## ⚠️ Error Handling: Terminal vs Retriable

Not all errors should be retried forever!

### Retriable Errors (Default)

```go
func MyHandler(ctx restate.Context, input string) (string, error) {
    // Regular error - Restate will retry
    return "", fmt.Errorf("database temporarily unavailable")
    // ↑ Restate retries with exponential backoff
}
```

**Use For:**
- Network timeouts
- Temporary service unavailability
- Transient database errors

### Terminal Errors (Stop Retrying)

```go
func MyHandler(ctx restate.Context, input string) (string, error) {
    if input == "" {
        // Terminal error - no retry, fail immediately
        return "", restate.TerminalError(
            fmt.Errorf("input cannot be empty"), 
            400, // HTTP status code
        )
    }
    return processInput(input), nil
}
```

**Use For:**
- Invalid input (validation errors)
- Business logic failures (insufficient funds)
- Permanent failures (resource not found)

### Error Handling Decision Tree

```
Is this error temporary?
├─ YES → Regular error (retriable)
│        return fmt.Errorf("...")
│
└─ NO → Terminal error (stop retry)
         return restate.TerminalError(...)
```

## 📊 The Big Picture

```
┌─────────────────────────────────────────────────────────┐
│                    Your Application                      │
│                                                          │
│  ┌──────────────┐      ┌──────────────┐                │
│  │   Service A  │◄────►│   Service B  │                │
│  │  (Basic Svc) │      │ (Virt Object)│                │
│  └──────┬───────┘      └──────┬───────┘                │
│         │                     │                         │
└─────────┼─────────────────────┼──────────────────────────┘
          │                     │
          │    All calls go     │
          │    through          │
          │    Restate          │
          └──────────┬──────────┘
                     ▼
          ┌─────────────────────┐
          │  Restate Server     │
          │                     │
          │  ✅ Journals calls  │
          │  ✅ Manages state   │
          │  ✅ Handles retries │
          │  ✅ Ensures exactly-│
          │     once execution  │
          └─────────────────────┘
```

## 🎓 Key Principles

### 1. **Durability by Default**

Every operation through `ctx` is durable:
```go
// ✅ Durable - survives crashes
result, _ := restate.Service[string](ctx, "Svc", "Handler").Request(data)

// ❌ Not durable - lost on crash
result := callExternalAPI(data) // Wrong!
```

### 2. **Exactly-Once Execution**

Each journaled step executes exactly once:
```go
// This payment is charged exactly once, even if handler retries
payment, _ := restate.Service[Payment](ctx, "Payment", "Charge").
    Request(user)
// ↑ On retry, Restate replays the result from the journal
```

### 3. **Automatic Retry**

Restate retries on transient failures:
```go
// If this fails with a network error, Restate automatically retries
data, err := restate.Service[Data](ctx, "DB", "Fetch").Request(query)
// No manual retry logic needed!
```

## 💡 Mental Model

Think of Restate as:

> **A database for your code execution**
>
> - Your handler is a transaction
> - Each step is logged (journaled)
> - Failures trigger replay from the log
> - State and calls are automatically managed

## ✅ Concept Check

Before moving to hands-on, ensure you understand:

- [ ] What durable execution means
- [ ] How journaling and replay work
- [ ] The three service types and when to use each
- [ ] Difference between Terminal and Retriable errors
- [ ] Why we use `ctx` for operations

## 🎯 Next Step

Now that you understand the theory, let's build something!

👉 **Continue to [Hands-On Tutorial](./02-hands-on.md)**

---

**Questions?** These concepts will become clearer as you code. Let's get hands-on!
