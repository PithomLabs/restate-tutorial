# Concepts: Side Effects and Durable Execution

> **Understanding the critical distinction between deterministic and non-deterministic operations**

## 🎯 What Are Side Effects?

### Definition

A **side effect** is any operation that:
1. Interacts with the external world
2. Produces non-deterministic results
3. Cannot be safely replayed without consequences

### Examples of Side Effects

#### 🌐 Network Calls
```go
// Side effect: HTTP API call
response := http.Get("https://api.weather.com/data")
// Result may differ each time!
```

#### 💾 Database Operations
```go
// Side effect: Database query
result := db.Query("SELECT * FROM users WHERE active = true")
// Data may change between calls
```

#### 📧 External Service Interactions
```go
// Side effect: Send email
emailService.Send(to, subject, body)
// Should only happen once!
```

#### 🎲 Non-Deterministic Operations
```go
// Side effect: Random number
num := rand.Float64()
// Different value each time!

// Side effect: Current time
now := time.Now()
// Changes every moment!
```

## ⚠️ The Problem with Side Effects

### Without Durable Execution

```go
func ProcessOrder(ctx restate.Context, orderID string) error {
    // Step 1: Check inventory (side effect)
    available := checkInventoryAPI(orderID) // ✅ Executes
    
    // Step 2: Charge payment (side effect)
    charged := chargePaymentAPI(orderID)    // ✅ Executes
    
    // Step 3: Send confirmation (side effect)
    sendEmailAPI(orderID)                    // ❌ CRASH!
    
    // On retry (service restarts):
    // - checkInventoryAPI() executes AGAIN
    // - chargePaymentAPI() executes AGAIN (double charge!)
    // - sendEmailAPI() finally executes
    
    return nil
}
```

**Problems:**
- 😱 Duplicate operations
- 💸 Potential double charges
- 📊 Inconsistent state
- 🔄 Wasted API calls

### With Restate Journaling (Still Wrong!)

```go
func ProcessOrder(ctx restate.Context, orderID string) error {
    // Even with Restate, these are NOT journaled:
    available := checkInventoryAPI(orderID) // ❌ Lost on crash
    charged := chargePaymentAPI(orderID)    // ❌ Lost on crash
    sendEmailAPI(orderID)                   // ❌ Lost on crash
    
    return nil
}
```

Restate journals your **handler execution**, but not external calls unless you use `restate.Run`!

## ✅ The Solution: `restate.Run`

### Basic Pattern

```go
func ProcessOrder(ctx restate.Context, orderID string) error {
    // Wrap side effect in restate.Run
    available, err := restate.Run(ctx, func(rc restate.RunContext) (bool, error) {
        // This executes exactly once
        return checkInventoryAPI(orderID), nil
    })
    if err != nil {
        return err
    }
    
    // Result is journaled! On retry, Restate replays from journal
    if !available {
        return restate.TerminalError(fmt.Errorf("out of stock"), 400)
    }
    
    // More side effects...
    _, err = restate.Run(ctx, func(rc restate.RunContext) (string, error) {
        return chargePaymentAPI(orderID), nil
    })
    
    return err
}
```

### How It Works

```
First Execution:
┌─────────────────────────────────────┐
│ Journal Entry 1: Run checkInventory │
│   Input: orderID=123                │
│   Executes: checkInventoryAPI()     │
│   Result: true                      │ ← Stored in journal
├─────────────────────────────────────┤
│ Journal Entry 2: Run chargePayment  │
│   Input: orderID=123                │
│   Executes: chargePaymentAPI()      │
│   Result: "payment_abc"             │ ← Stored
│   ... CRASH ...                     │
└─────────────────────────────────────┘

On Retry (Automatic):
┌─────────────────────────────────────┐
│ Journal Entry 1: REPLAY from journal│
│   Result: true (no API call!)       │ ← From journal
├─────────────────────────────────────┤
│ Journal Entry 2: REPLAY from journal│
│   Result: "payment_abc"             │ ← From journal
├─────────────────────────────────────┤
│ Journal Entry 3: NEW - send email   │
│   Executes: sendEmailAPI()          │ ← Runs only once
│   Result: success                   │
└─────────────────────────────────────┘
```

**Key Points:**
- ✅ First execution: API calls happen
- ✅ Results are journaled
- ✅ On retry: Results replay from journal
- ✅ No duplicate API calls
- ✅ Exactly-once guarantee

## 🧠 Deterministic vs Non-Deterministic

### Deterministic Operations (Safe Outside Run)

Operations that **always produce the same result** for the same input:

```go
func MyHandler(ctx restate.Context, input int) (int, error) {
    // ✅ Safe - pure computation
    result := input * 2
    
    // ✅ Safe - deterministic string operation
    name := strings.ToUpper("alice")
    
    // ✅ Safe - deterministic from input
    if input > 100 {
        return result + 10, nil
    }
    
    return result, nil
}
```

### Non-Deterministic Operations (Need Run)

Operations that **may produce different results**:

```go
func MyHandler(ctx restate.Context, input string) (string, error) {
    // ❌ Non-deterministic - wrap in Run
    currentTime, err := restate.Run(ctx, func(rc restate.RunContext) (time.Time, error) {
        return time.Now(), nil
    })
    
    // ❌ Non-deterministic - use restate.Rand instead
    randomVal := restate.Rand(ctx).Float64() // ✅ Deterministic alternative
    
    // ❌ Non-deterministic - wrap in Run
    apiData, err := restate.Run(ctx, func(rc restate.RunContext) (string, error) {
        return fetchFromAPI(input), nil
    })
    
    return apiData, err
}
```

### Quick Reference

| Operation | Deterministic? | Approach |
|-----------|---------------|----------|
| Math operations | ✅ Yes | Use directly |
| String manipulation | ✅ Yes | Use directly |
| Input processing | ✅ Yes | Use directly |
| HTTP calls | ❌ No | Wrap in `restate.Run` |
| Database queries | ❌ No | Wrap in `restate.Run` |
| File I/O | ❌ No | Wrap in `restate.Run` |
| `time.Now()` | ❌ No | Wrap in `restate.Run` |
| `rand.Float64()` | ❌ No | Use `restate.Rand(ctx)` |
| UUID generation | ❌ No | Use `restate.UUID(ctx)` |

## 🎓 Understanding RunContext

### The Two Contexts

```go
restate.Run(ctx, func(rc restate.RunContext) (T, error) {
    // ctx  - Main handler context (DO NOT USE HERE!)
    // rc   - Run context (USE THIS!)
})
```

**Critical Rule:** Inside `restate.Run`, **only use `rc`**, never `ctx`!

### Why Two Contexts?

```go
func BadExample(ctx restate.Context, input string) error {
    return restate.Run(ctx, func(rc restate.RunContext) (error) {
        // ❌ WRONG - using ctx inside Run
        ctx.Log().Info("calling API")
        
        // This breaks journaling!
        return nil
    })
}

func GoodExample(ctx restate.Context, input string) error {
    // ✅ Log outside Run
    ctx.Log().Info("about to call API")
    
    _, err := restate.Run(ctx, func(rc restate.RunContext) (string, error) {
        // ✅ Just do the side effect
        return callAPI(input), nil
    })
    
    return err
}
```

### What Can Go Inside Run?

**✅ Allowed:**
```go
restate.Run(ctx, func(rc restate.RunContext) (T, error) {
    // ✅ HTTP calls
    resp, err := http.Get(url)
    
    // ✅ Database queries
    result := db.Query(query)
    
    // ✅ File operations
    data, err := os.ReadFile(path)
    
    // ✅ External service calls
    result := externalService.Call(params)
    
    // ✅ Pure Go code
    processed := strings.ToUpper(data)
    
    return result, nil
})
```

**❌ NOT Allowed:**
```go
restate.Run(ctx, func(rc restate.RunContext) (T, error) {
    // ❌ Calling other Restate services
    restate.Service[T](ctx, "Svc", "Handler").Request(data)
    
    // ❌ State operations
    restate.Get[T](ctx, "key")
    restate.Set(ctx, "key", value)
    
    // ❌ Sleep
    restate.Sleep(ctx, duration)
    
    // ❌ Logging
    ctx.Log().Info("message")
    
    // These should be OUTSIDE the Run block!
    return result, nil
})
```

## 🔄 Retry Strategies

### Automatic Retry (Default)

```go
// Transient error - Restate retries automatically
_, err := restate.Run(ctx, func(rc restate.RunContext) (string, error) {
    resp, err := http.Get(url)
    if err != nil {
        return "", err // Restate will retry
    }
    return resp.Body, nil
})
```

### Terminal Error (Stop Retry)

```go
_, err := restate.Run(ctx, func(rc restate.RunContext) (string, error) {
    resp, err := http.Get(url)
    if err != nil {
        return "", err // Retry on network error
    }
    
    if resp.StatusCode == 404 {
        // Resource not found - no point retrying
        return "", restate.TerminalError(
            fmt.Errorf("resource not found"),
            404,
        )
    }
    
    return resp.Body, nil
})
```

### Custom Retry Logic

```go
// Retry with limits
maxAttempts := 3
var lastErr error

for attempt := 1; attempt <= maxAttempts; attempt++ {
    result, err := restate.Run(ctx, func(rc restate.RunContext) (string, error) {
        return callAPI(), nil
    })
    
    if err == nil {
        return result, nil
    }
    
    lastErr = err
    
    // Wait before retry (durable sleep!)
    if attempt < maxAttempts {
        restate.Sleep(ctx, time.Duration(attempt) * time.Second)
    }
}

return "", lastErr
```

## ⚠️ Common Anti-Patterns

### Anti-Pattern 1: Using `ctx` Inside Run

```go
// ❌ WRONG
restate.Run(ctx, func(rc restate.RunContext) (string, error) {
    ctx.Log().Info("calling API") // Using wrong context!
    return callAPI(), nil
})

// ✅ CORRECT
ctx.Log().Info("calling API") // Log outside
result, err := restate.Run(ctx, func(rc restate.RunContext) (string, error) {
    return callAPI(), nil
})
```

### Anti-Pattern 2: Not Wrapping External Calls

```go
// ❌ WRONG - API call not journaled
func MyHandler(ctx restate.Context, input string) (string, error) {
    data := callExternalAPI(input) // Lost on crash!
    return process(data), nil
}

// ✅ CORRECT
func MyHandler(ctx restate.Context, input string) (string, error) {
    data, err := restate.Run(ctx, func(rc restate.RunContext) (string, error) {
        return callExternalAPI(input), nil
    })
    if err != nil {
        return "", err
    }
    return process(data), nil
}
```

### Anti-Pattern 3: Calling Restate Operations Inside Run

```go
// ❌ WRONG
restate.Run(ctx, func(rc restate.RunContext) (string, error) {
    // Can't call other services inside Run!
    result, _ := restate.Service[string](ctx, "Svc", "Handler").Request(data)
    return result, nil
})

// ✅ CORRECT - Separate the calls
data, err := restate.Run(ctx, func(rc restate.RunContext) (string, error) {
    return callExternalAPI(), nil
})

result, err := restate.Service[string](ctx, "Svc", "Handler").Request(data)
```

## 💡 Best Practices

### 1. Keep Run Blocks Small

```go
// ✅ Good - focused on side effect
apiData, err := restate.Run(ctx, func(rc restate.RunContext) (APIResponse, error) {
    return fetchFromAPI(url), nil
})

processed := processData(apiData) // Pure logic outside
```

### 2. Handle Errors Appropriately

```go
data, err := restate.Run(ctx, func(rc restate.RunContext) (Data, error) {
    resp, err := http.Get(url)
    if err != nil {
        return Data{}, err // Let Restate retry
    }
    
    if resp.StatusCode == 400 {
        return Data{}, restate.TerminalError(
            fmt.Errorf("bad request"),
            400,
        ) // Don't retry client errors
    }
    
    return parseResponse(resp), nil
})
```

### 3. Log Before and After, Not Inside

```go
ctx.Log().Info("fetching user data", "userID", userID)

userData, err := restate.Run(ctx, func(rc restate.RunContext) (User, error) {
    return fetchUser(userID), nil
})

if err != nil {
    ctx.Log().Error("failed to fetch user", "error", err)
    return User{}, err
}

ctx.Log().Info("user data fetched", "user", userData.Name)
```

## ✅ Concept Check

Before moving to hands-on, ensure you understand:

- [ ] What side effects are and why they matter
- [ ] Why external calls need `restate.Run`
- [ ] Difference between `ctx` and `rc` (RunContext)
- [ ] What can/cannot go inside Run blocks
- [ ] Deterministic vs non-deterministic operations
- [ ] When to use Terminal vs regular errors

## 🎯 Next Step

Now let's put this into practice!

👉 **Continue to [Hands-On Tutorial](./02-hands-on.md)**

---

**Key Takeaway:** `restate.Run` is your tool for making external operations durable. Wrap every side effect!
