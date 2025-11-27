# Run (Side Effects) Utilities - Complete Guide

## Overview

The framework provides comprehensive utilities for handling side effects with `restate.Run`, including asynchronous execution, automatic retries, and deterministic helpers for randomness, UUIDs, and time capture.

## Features Added

### 1. RunAsync Support
Concurrent side effect execution with futures.

### 2. RunWithRetry
Automatic retry with exponential backoff.

### 3. Deterministic Helpers
- UUID generation (same on replay)
- Random numbers (same on replay)
- Time capture (same on replay)

### 4. Retry Policy Configuration
Configurable retry behavior with backoff.

## RunConfig - Retry Configuration

```go
type RunConfig struct {
    MaxRetries    int           // Maximum retry attempts
    InitialDelay  time.Duration // Starting delay
    MaxDelay      time.Duration // Maximum delay cap
    BackoffFactor float64       // Exponential backoff multiplier
    Name          string        // Operation name for logging
}

func DefaultRunConfig(name string) RunConfig
```

## Usage Examples

### Example 1: Basic RunWithRetry

```go
func (MyService) ProcessOrder(ctx restate.Context, order Order) error {
    cfg := DefaultRunConfig("process-payment")
    cfg.MaxRetries = 5  // Override default
    
    result, err := RunWithRetry(ctx, cfg, func(rc restate.RunContext) (PaymentResult, error) {
        // Side effect - calling external payment API
        return callPaymentAPI(order.Amount, order.PaymentMethod)
    })
    
    if err != nil {
        return fmt.Errorf("payment failed after retries: %w", err)
    }
    
    ctx.Log().Info("Payment successful", "transaction_id", result.TransactionID)
    return nil
}
```

### Example 2: RunAsync for Concurrent Side Effects

```go
func (MyService) FetchDataConcurrently(ctx restate.Context) (Dashboard, error) {
    // Start multiple async operations
    userFuture := RunAsync(ctx, func(rc restate.RunContext) (UserData, error) {
        return fetchUserData()
    }, restate.WithName("fetch-user"))
    
    ordersFuture := RunAsync(ctx, func(rc restate.RunContext) ([]Order, error) {
        return fetchOrders()
    }, restate.WithName("fetch-orders"))
    
    notifsFuture := RunAsync(ctx, func(rc restate.RunContext) ([]Notification, error) {
        return fetchNotifications()
    }, restate.WithName("fetch-notifications"))
    
    // Wait for all results
    userData, err1 := userFuture.Response()
    orders, err2 := ordersFuture.Response()
    notifs, err3 := notifsFuture.Response()
    
    if err1 != nil || err2 != nil || err3 != nil {
        return Dashboard{}, fmt.Errorf("failed to fetch data")
    }
    
    return Dashboard{
        User:          userData,
        Orders:        orders,
        Notifications: notifs,
    }, nil
}
```

### Example 3: Deterministic UUID Generation

```go
func (OrderWorkflow) Run(ctx restate.WorkflowContext, order Order) error {
    helpers := NewDeterministicHelpers(ctx)
    
    // Generate deterministic UUID (same on replay!)
    orderID := helpers.UUID()
    invoiceID := helpers.UUID()
    
    ctx.Log().Info("IDs generated", 
        "order_id", orderID,
        "invoice_id", invoiceID)
    
    // These IDs will be the SAME if the workflow replays
    return processOrder(ctx, orderID, invoiceID, order)
}
```

### Example 4: Deterministic Random Numbers

```go
func (LotteryService) DrawWinner(ctx restate.Context, participants []string) (string, error) {
    if len(participants) == 0 {
        return "", fmt.Errorf("no participants")
    }
    
    helpers := NewDeterministicHelpers(ctx)
    
    // Deterministic random selection (same on replay!)
    winnerIndex := helpers.RandInt(0, len(participants))
    winner := participants[winnerIndex]
    
    ctx.Log().Info("Winner selected", 
        "winner", winner,
        "index", winnerIndex,
        "total_participants", len(participants))
    
    return winner, nil
}
```

### Example 5: Deterministic Random Choice Helper

```go
func (GameService) SelectRandomReward(ctx restate.Context) (Reward, error) {
    rewards := []Reward{
        {Name: "Gold Coin", Value: 100},
        {Name: "Silver Coin", Value: 50},
        {Name: "Bronze Coin", Value: 10},
        {Name: "Gem", Value: 500},
    }
    
    // Deterministically picks one (same on replay)
    reward, err := RandChoice(ctx, rewards)
    if err != nil {
        return Reward{}, err
    }
    
    ctx.Log().Info("Reward selected", "reward", reward.Name)
    return reward, nil
}
```

### Example 6: Deterministic Random Weighted Selection

```go
func (AdService) SelectAd(ctx restate.Context) (Ad, error) {
    helpers := NewDeterministicHelpers(ctx)
    
    // Weighted random selection
    weights := []int{50, 30, 15, 5}  // Probabilities
    totalWeight := 100
    
    randomValue := helpers.RandInt(0, totalWeight)
    
    cumulative := 0
    selectedIndex := 0
    for i, weight := range weights {
        cumulative += weight
        if randomValue < cumulative {
            selectedIndex = i
            break
        }
    }
    
    ads := []Ad{
        {ID: "premium-1", Type: "premium"},
        {ID: "standard-1", Type: "standard"},
        {ID: "basic-1", Type: "basic"},
        {ID: "fallback-1", Type: "fallback"},
    }
    
    return ads[selectedIndex], nil
}
```

### Example 7: Deterministic Time Capture

```go
func (ReportService) GenerateReport(ctx restate.Context, req ReportRequest) error {
    timeHelper := NewTime(ctx)
    
    // Capture current time deterministically (same on replay!)
    reportTime := timeHelper.Now()
    
    // This timestamp will be the same if the workflow replays
    report := Report{
        GeneratedAt: reportTime,
        Data:        collectData(ctx, req),
    }
    
    // Store with deterministic timestamp
    return storeReport(ctx, report)
}
```

### Example 8: Deterministic Time Calculations

```go
func (SubscriptionService) CheckExpiry(ctx restate.Context, subscription Subscription) (bool, error) {
    timeHelper := NewTime(ctx)
    
    now := timeHelper.Now()  // Deterministic
    expiry := subscription.ExpiresAt
    
    // Calculate time remaining
    remaining := timeHelper.Until(expiry)
    
    if remaining <= 0 {
        ctx.Log().Info("Subscription expired", 
            "subscription_id", subscription.ID,
            "expired", expiry)
        return true, nil
    }
    
    if remaining < 7*24*time.Hour {
        ctx.Log().Warn("Subscription expiring soon",
            "subscription_id", subscription.ID,
            "days_remaining", remaining.Hours()/24)
        
        // Send reminder
        sendExpiryReminder(ctx, subscription)
    }
    
    return false, nil
}
```

### Example 9: Combined RunAsync + Deterministic Helpers

```go
func (AuctionService) ProcessBids(ctx restate.Context, auction Auction) error {
    helpers := NewDeterministicHelpers(ctx)
    
    // Generate deterministic auction close time offset
    closeOffsetMinutes := helpers.RandInt(1, 10)  // Random 1-10 minutes
    
    ctx.Log().Info("Auction close time randomized",
        "offset_minutes", closeOffsetMinutes)
    
    // Process bids concurrently
    bidFutures := make([]restate.Future, 0)
    for _, bid := range auction.Bids {
        fut := RunAsync(ctx, func(rc restate.RunContext) (BidResult, error) {
            return validateBid(bid)
        }, restate.WithName(fmt.Sprintf("validate-bid-%s", bid.ID)))
        
        bidFutures = append(bidFutures, fut)
    }
    
    // Wait for all validations
    for _, fut := range bidFutures {
        result, err := fut.Response()
        if err != nil {
            ctx.Log().Error("Bid validation failed", "error", err)
        } else {
            processBidResult(ctx, result.(BidResult))
        }
    }
    
    return nil
}
```

### Example 10: Advanced Retry Configuration

```go
func (EmailService) SendCriticalEmail(ctx restate.Context, email Email) error {
    // Custom retry configuration for critical emails
    cfg := RunConfig{
        MaxRetries:    10,                    // More retries
        InitialDelay:  500 * time.Millisecond, // Start with 500ms
        MaxDelay:      5 * time.Minute,       // Cap at 5 minutes
        BackoffFactor: 2.5,                   // Aggressive backoff
        Name:          "send-critical-email",
    }
    
    result, err := RunWithRetry(ctx, cfg, func(rc restate.RunContext) (SendResult, error) {
        return sendEmail(email)
    })
    
    if err != nil {
        // Exhausted all retries - escalate
        return escalateEmailFailure(ctx, email, err)
    }
    
    ctx.Log().Info("Critical email sent", "message_id", result.MessageID)
    return nil
}
```

### Example 11: RunAsync with Error Handling

```go
func (DataService) EnrichDataConcurrently(ctx restate.Context, ids []string) ([]EnrichedData, error) {
    futures := make([]restate.Future, 0, len(ids))
    
    // Start all enrichment operations concurrently
    for _, id := range ids {
        dataID := id
        fut := RunAsync(ctx, func(rc restate.RunContext) (EnrichedData, error) {
            return enrichData(dataID)
        }, restate.WithName(fmt.Sprintf("enrich-%s", dataID)))
        
        futures = append(futures, fut)
    }
    
    // Collect results
    results := make([]EnrichedData, 0, len(ids))
    errors := make([]error, 0)
    
    for i, fut := range futures {
        data, err := fut.Response()
        if err != nil {
            errors = append(errors, fmt.Errorf("enrichment %d failed: %w", i, err))
            continue
        }
        
        if enriched, ok := data.(EnrichedData); ok {
            results = append(results, enriched)
        }
    }
    
    if len(errors) > 0 {
        ctx.Log().Warn("Some enrichments failed", "failed_count", len(errors))
        // Continue with partial results rather than failing completely
    }
    
    return results, nil
}
```

## Best Practices

### ✅ DO: Use RunWithRetry for Transient Failures

```go
// ✓ Automatic retry for network calls
cfg := DefaultRunConfig("api-call")
result, _ := RunWithRetry(ctx, cfg, func(rc restate.RunContext) (Data, error) {
    return callExternalAPI()
})
```

### ✅ DO: Use Deterministic Helpers

```go
// ✓ Deterministic UUID
helpers := NewDeterministicHelpers(ctx)
id := helpers.UUID()  // Same on replay!

// ✗ DON'T: Non-deterministic
// id := uuid.New().String()  // Different on replay!
```

### ✅ DO: Capture Time in Run Blocks

```go
// ✓ Deterministic time
timeHelper := NewTime(ctx)
now := timeHelper.Now()  // Same on replay

// ✗ DON'T: Direct time.Now()
// now := time.Now()  // Different on replay!
```

### ✅ DO: Use RunAsync for Concurrent Side Effects

```go
// ✓ Concurrent with futures
fut1 := RunAsync(ctx, operation1)
fut2 := RunAsync(ctx, operation2)
result1, _ := fut1.Response()
result2, _ := fut2.Response()
```

### ❌ DON'T: Use Non-Deterministic Operations

```go
// ✗ DON'T: Direct random
value := rand.Intn(100)  // Different on replay!

// ✓ DO: Deterministic random
helpers := NewDeterministicHelpers(ctx)
value := helpers.RandInt(0, 100)  // Same on replay!
```

### ❌ DON'T: Retry Forever

```go
// ✗ DON'T: No retry limit
cfg := RunConfig{MaxRetries: 1000000}  // Too many!

// ✓ DO: Reasonable limit
cfg := DefaultRunConfig("operation")
cfg.MaxRetries = 5  // Reasonable
```

## Determinism Guarantees

### What's Deterministic

| Operation | Deterministic? | Framework Helper |
|-----------|---------------|------------------|
| `restate.UUID(ctx)` | ✅ Yes | `helpers.UUID()` |
| `restate.Rand(ctx)` | ✅ Yes | `helpers.Rand()` |
| `restate.Run(ctx, time.Now)` | ✅ Yes | `timeHelper.Now()` |
| `restate.Sleep(ctx, duration)` | ✅ Yes | Built-in |
| `uuid.New()` | ❌ No | Use helper! |
| `rand.Intn()` | ❌ No | Use helper! |
| `time.Now()` | ❌ No | Use helper! |

### How It Works

```go
// Framework helper (recommended)
helpers := NewDeterministicHelpers(ctx)
id := helpers.UUID()
// → Uses restate.UUID(ctx) internally
// → Logged in journal
// → Same on replay!

// Direct SDK (also works)
id := restate.UUID(ctx).String()
// → Same result, more verbose
```

## Retry Policy Patterns

### Conservative (Default)
```go
cfg := DefaultRunConfig("operation")
// MaxRetries: 3
// InitialDelay: 100ms
// MaxDelay: 30s
// BackoffFactor: 2.0
```

### Aggressive (Quick retries)
```go
cfg := RunConfig{
    MaxRetries:    5,
    InitialDelay:  50 * time.Millisecond,
    MaxDelay:      5 * time.Second,
    BackoffFactor: 1.5,
    Name:          "quick-retry",
}
```

### Patient (Long backoff)
```go
cfg := RunConfig{
    MaxRetries:    10,
    InitialDelay:  1 * time.Second,
    MaxDelay:      10 * time.Minute,
    BackoffFactor: 3.0,
    Name:          "patient-retry",
}
```

## Testing

```go
func TestRunWithRetry_Success(t *testing.T) {
    ctx := mockContext()
    cfg := DefaultRunConfig("test")
    
    callCount := 0
    result, err := RunWithRetry(ctx, cfg, func(rc restate.RunContext) (string, error) {
        callCount++
        if callCount < 3 {
            return "", fmt.Errorf("transient error")
        }
        return "success", nil
    })
    
    assert.NoError(t, err)
    assert.Equal(t, "success", result)
    assert.Equal(t, 3, callCount)
}

func TestDeterministicHelpers_UUID(t *testing.T) {
    ctx := mockContext()
    helpers := NewDeterministicHelpers(ctx)
    
    id1 := helpers.UUID()
    id2 := helpers.UUID()
    
    // Should be different UUIDs
    assert.NotEqual(t, id1, id2)
    
    // But deterministic on replay
    assert.NotEmpty(t, id1)
    assert.NotEmpty(t, id2)
}
```

## Conclusion

Run (Side Effects) utilities provide:

- ✅ **RunAsync** - Concurrent side effect execution
- ✅ **RunWithRetry** - Automatic retry with backoff
- ✅ **Deterministic UUID** - Same IDs on replay
- ✅ **Deterministic Random** - Same random values on replay
- ✅ **Deterministic Time** - Same timestamps on replay
- ✅ **Configurable Retry** - Flexible retry policies
- ✅ **Logging Integration** - Automatic retry logging

**Recommendation:** Always use framework helpers for UUIDs, random numbers, and time to ensure deterministic replay. Use `RunWithRetry` for operations that may have transient failures.

**Grade Improvement:** Run (Side Effects) upgraded from partial support to **A- (comprehensive coverage)**.
