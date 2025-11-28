# How Idempotency Works with External APIs and Side Effects in Restate

**Based on:** DOS_DONTS_REA.MD and Restate SDK principles  
**Key Concept:** Journaling makes side effects exactly-once, even during replays

---

## The Core Problem

When your Restate handler calls an external API:
```go
func (MyService) ProcessPayment(ctx restate.Context, amount int) error {
    // ❌ WRONG: This will execute MULTIPLE times if the handler retries!
    response := callStripeAPI(amount)  // Charges customer!
    
    // If handler crashes here and retries, customer gets charged again! 💸💸
    return processResponse(response)
}
```

**The Issue:** Without journaling, the external API call executes **every time** the handler runs, including during replay after failures.

---

## The Solution: `restate.Run()` - Journaling Side Effects

### How It Works

```go
func (MyService) ProcessPayment(ctx restate.Context, amount int) error {
    // ✅ CORRECT: Wrap in restate.Run for exactly-once execution
    response, err := restate.Run(ctx, func(ctx restate.RunContext) (StripeResponse, error) {
        // This block executes ONLY ONCE
        // The result is recorded in Restate's journal
        return callStripeAPI(amount), nil  // Charges customer exactly once ✅
    })
    
    if err != nil {
        return err
    }
    
    // If handler crashes HERE and retries:
    // 1. Restate sees "callStripeAPI" in the journal
    // 2. It returns the RECORDED response (from journal)
    // 3. The Run block is SKIPPED entirely
    // 4. Customer is NOT charged again!
    
    return processResponse(response)
}
```

### The Magic of Journaling

**First Execution:**
1. Handler reaches `restate.Run`
2. Journal says: "No entry for this Run block yet"
3. Executes `callStripeAPI(amount)`  → Customer charged $50
4. Records in journal: `Run[callStripeAPI] = {success: true, charge_id: "ch_123"}`

**After Crash (Replay):**
1. Handler restarts from beginning (deterministic replay)
2. Reaches `restate.Run` again
3. Journal says: "Already have entry: `{success: true, charge_id: "ch_123"}`"
4. **SKIPS** the `callStripeAPI` function entirely
5. Returns the recorded value from journal
6. ✅ Customer NOT charged again!

---

## From DOS_DONTS_REA.MD

### Section I: Data Plane Isolation

> **Do** wrap all non-deterministic side effects (external HTTP calls, database updates, file I/O) within `restate.Run`.

```go
// ❌ DON'T: Direct external call
user := fetchUserFromDB(userID)  // Executes multiple times on retry!

// ✅ DO: Wrapped in restate.Run
user, err := restate.Run(ctx, func(rc restate.RunContext) (User, error) {
    return fetchUserFromDB(userID), nil  // Executes ONCE, result journaled
})
```

### Section I: Context Misuse

> **Don't** use the outer Restate context (`ctx.Sleep`, `ctx.Client().Call()`, nested `ctx.Run`, `ctx.SetState()`) inside the `restate.Run` closure.

```go
// ❌ WRONG: Using outer ctx inside Run
restate.Run(ctx, func(rc restate.RunContext) (bool, error) {
    // NEVER do this - breaks determinism!
    ctx.Sleep(1 * time.Second)  // ❌
    return doSideEffect(), nil
})

// ✅ CORRECT: Only use RunContext inside
restate.Run(ctx, func(rc restate.RunContext) (bool, error) {
    // Use rc.Log() for logging
    rc.Log().Info("Calling external API")
    
    // Pure side effect code
    return doSideEffect(), nil
})
```

---

## Real-World Example: Shipping Service

### Problem: Non-Idempotent Shipping API

Many shipping APIs (FedEx, UPS) are **NOT idempotent**:
- Calling `createShipment(orderID, address)` multiple times creates **multiple shipments**
- Each call generates a new tracking number
- Customer gets charged multiple times for shipping

### Solution: Journal the Side Effect

```go
func (ShippingService) InitiateShipment(
    ctx restate.Context,
    shipment models.ShipmentRequest,
) (bool, error) {
    ctx.Log().Info("Initiating shipment", "order_id", shipment.OrderID)
    
    // External API call wrapped in restate.Run
    trackingNumber, err := restate.Run(ctx, func(rc restate.RunContext) (string, error) {
        // This code executes EXACTLY ONCE, even across retries
        rc.Log().Info("Calling FedEx API")
        
        response := fedexAPI.CreateShipment(
            orderID:  shipment.OrderID,
            address:  shipment.Address,
            package:  "standard_box",
        )
        
        if response.Error != nil {
            return "", fmt.Errorf("fedex error: %w", response.Error)
        }
        
        // Result recorded in journal: "FDX123456789"
        return response.TrackingNumber, nil
    })
    
    if err != nil {
        // Transient error - Restate will retry entire handler
        // But Run block won't execute again!
        return false, err
    }
    
    // Store tracking number (journaled operation)
    restate.Set(ctx, "tracking_number", trackingNumber)
    
    ctx.Log().Info("Shipment created",  "tracking_number", trackingNumber)
    return true, nil
}
```

**Timeline:**

| Attempt | What Happens | FedEx API Called? | Tracking # |
|---------|--------------|-------------------|------------|
| 1st execution | Creates shipment successfully | ✅ Yes | FDX123456789 |
| *Crash after Set* | - | - | - |
| 2nd execution (retry) | Run block **skipped**, returns `"FDX123456789"` from journal | ❌ No | FDX123456789 |
| Set executes again | Idempotent set operation | ❌ No | FDX123456789 |

✅ **Result:** Only ONE shipment created, despite handler executing twice!

---

## Using `rea.RunWithRetry` for Enhanced Patterns

The `rea` framework provides `RunWithRetry` which adds automatic retry logic **within** the Run block:

```go
func (ShippingService) InitiateShipment(
    ctx restate.Context,
    shipment models.ShipmentRequest,
) (bool, error) {
    cfg := rea.RunConfig{
        MaxRetries:    3,                     // Retry up to 3 times
        InitialDelay:  100 * time.Millisecond,
        MaxDelay:      2 * time.Second,
        BackoffFactor: 2.0,
        Name:          "InitiateShipment",
    }
    
    // RunWithRetry handles transient failures WITHIN the Run block
    success, err := rea.RunWithRetry(ctx, cfg, func(ctx restate.RunContext) (bool, error) {
        // Try calling FedEx API
        response := fedexAPI.CreateShipment(shipment)
        
        if response.RateLimited {
            // Return error - RunWithRetry will retry with backoff
            return false, fmt.Errorf("rate limited")
        }
        
        if response.NetworkError {
            // Transient error - will retry
            return false, fmt.Errorf("network error")
        }
        
        if response.InvalidAddress {
            // Business logic error - DON'T retry
            return false, restate.TerminalError(
                fmt.Errorf("invalid address"), 400)
        }
        
        // Success!
        return true, nil
    })
    
    // After retries exhausted or success:
    // 1st attempt: Runs up to 4 times (1 + 3 retries), journals final result
    // 2nd attempt (after crash): Skips entirely, returns journaled result
    
    return success, err
}
```

**Key Point:** `RunWithRetry` retries **within the same journal entry**. Once it succeeds (or fails terminally), that result is journaled. Future replays skip the entire block.

---

## Idempotency Keys for External Systems

### When External API is Idempotent

Some modern APIs support idempotency keys:

```go
func (PaymentService) ChargeCustomer(
    ctx restate.Context,
    amount int,
    customerId string,
) (string, error) {
    // Generate deterministic idempotency key
    helpers := rea.NewDeterministicHelpers(ctx)
    idempotencyKey := helpers.UUID()  // Deterministic based on invocation
    
    chargeID, err := restate.Run(ctx, func(rc restate.RunContext) (string, error) {
        // Stripe accepts idempotency keys!
        charge := stripe.Charge.Create(&stripe.ChargeParams{
            Amount:         amount,
            Currency:       "usd",
            Customer:       customerId,
            IdempotencyKey: idempotencyKey,  // ✅ Safe to retry
        })
        
        return charge.ID, nil
    })
    
    return chargeID, err
}
```

**Benefits:**
- Even if `restate.Run` somehow executes twice (bug in Restate), Stripe's idempotency protects you
- Defense in depth: **Restate journaling + API idempotency** = bulletproof

### When External API is NOT Idempotent

Use Restate's journaling as your ONLY protection:

```go
func (EmailService) SendConfirmation(
    ctx restate.Context,
    email string,
    orderID string,
) error {
    // Email APIs are usually NOT idempotent
    // User will get duplicate emails if called twice!
    
    _, err := restate.Run(ctx, func(rc restate.RunContext) (bool, error) {
        // SendGrid, Mailgun, etc. - no idempotency support
        err := emailAPI.Send(To: email, Subject: "Order Confirmed", Body: "...")
        return true, err
    })
    
    // ✅ Restate's journaling ensures this sends EXACTLY ONCE
    return err
}
```

---

## State-Based Deduplication (Pattern C)

For **double protection**, combine journaling with state-based deduplication:

```go
func (UserSession) Checkout(ctx restate.ObjectContext, orderID string) (bool, error) {
    // Pattern C: Explicit State-Based Deduplication
    dedupKey := fmt.Sprintf("checkout:exec:%s", orderID)
    
    // Check if already executed
    executed, err := restate.Get[bool](ctx, dedupKey)
    if err == nil && executed {
        ctx.Log().Info("Duplicate checkout detected", "order_id", orderID)
        return true, nil  // Already done, skip execution
    }
    
    // Execute checkout logic (with journaled side effects)
    awakeable := restate.Awakeable[models.PaymentReceipt](ctx)
    
    // Notify payment gateway (journaled)
    _, err = restate.Run(ctx, func(rc restate.RunContext) (bool, error) {
        rc.Log().Info("Notifying payment gateway")
        return notifyPaymentGateway(awakeable.Id()), nil
    })
    
    if err != nil {
        return false, err
    }
    
    // Wait for payment
    receipt, err := awakeable.Result()
    if err != nil {
        return false, restate.TerminalError(fmt.Errorf("payment failed"), 500)
    }
    
    // Mark as executed (journaled state operation)
    restate.Set(ctx, dedupKey, true)
    
    return true, nil
}
```

**Why both?**
1. **Journaling** prevents duplicate external API calls during replay
2. **State deduplication** prevents duplicate execution if called with same `orderID` from different invocations

---

## Summary: The Idempotency Guarantee

| Mechanism | What It Protects Against | How It Works |
|-----------|-------------------------|--------------|
| **`restate.Run()` Journaling** | Handler retries/replays after failures | Records result in journal; skips execution on replay |
| **Deterministic IDs** | Different IDs generated on retry | Uses `restate.UUID()` or `rea.NewDeterministicHelpers()` |
| **State Deduplication** | Same operation called multiple times with same input | Checks state flag before executing |
| **External API Idempotency** | API called twice (belt + suspenders) | Send idempotency key to external API |

### The Golden Rules

1. ✅ **ALWAYS** wrap external API calls in `restate.Run()`
2. ✅ **ALWAYS** use `restate.RunContext` (not outer `ctx`) inside Run blocks
3. ✅ **NEVER** use `time.Now()`, `rand.Float64()`, or non-deterministic functions outside Run
4. ✅ **PREFER** `rea.RunWithRetry()` for automatic backoff on transient failures
5. ✅ **OPTIONAL** Add state-based deduplication for extra protection
6. ✅ **BONUS** Send idempotency keys to external APIs when supported

### The Result

```
🎯 Exactly-Once Side Effects
   └─ No duplicate charges
   └─ No duplicate shipments
   └─ No duplicate emails
   └─ No duplicate database writes
   
✅ Even across:
   - Handler crashes
   - Network failures
   - Service restarts
   - Restate runtime restarts
```

---

**Core Insight:** Restate's journal is a **time machine** for side effects. Once an external call succeeds and is recorded, all future timelines (replays) know the result and skip re-execution. This is how you get exactly-once guarantees in a distributed system.
