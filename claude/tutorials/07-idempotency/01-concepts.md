# Concepts: Idempotency in Distributed Systems

> **Understanding idempotency: The foundation of reliable distributed systems**

## 🎯 What You'll Learn

- What idempotency means and why it's critical
- How network failures create duplicate requests
- Exactly-once vs at-least-once semantics
- Idempotency keys and their role
- How Restate makes idempotency automatic
- Common patterns and anti-patterns

---

## 📖 What is Idempotency?

### Definition

**Idempotency** means that performing an operation multiple times has the same effect as performing it once.

In mathematical terms:
```
f(x) = f(f(x)) = f(f(f(x))) = ...
```

### In Distributed Systems

An **idempotent operation** produces the same result when called multiple times with the same input, regardless of how many times it's executed.

```go
// ✅ IDEMPOTENT
SET user_status = "active"  
// Calling 100 times = status is "active"

// ❌ NOT IDEMPOTENT  
INCREMENT page_views
// Calling 100 times = views incremented by 100
```

---

## 🤔 Why Idempotency Matters

### The Distributed Systems Problem

In distributed systems, operations can fail or be retried for many reasons:

#### 1. Network Failures
```
Client → Request → [Network Timeout] → Server
Client: "Did it work? Let me retry..."
Server: "I got both requests! 😱"
```

#### 2. Client Timeouts
```
Client sends request
↓
Server processes (slow)
↓
Client timeout (30s)
↓
Client retries
↓
Server finishes original request
Server receives retry
→ Operation happens TWICE!
```

#### 3. Duplicate Messages
```
Load Balancer
↓         ↓
Request duplicated by network
↓         ↓
Server A  Server B
Both process the same request!
```

### Real-World Consequences

Without idempotency:

| Operation | Non-Idempotent Result | Impact |
|-----------|----------------------|--------|
| **Payment** | Customer charged twice | 💰 Money loss, customer anger |
| **Order Creation** | Duplicate orders | 📦 Wasted inventory, shipping costs |
| **Email Sending** | 100 duplicate emails | 📧 Spam, poor UX |
| **Account Creation** | Conflicting accounts | 🔐 Data corruption |
| **Inventory Update** | Wrong stock levels | 📊 Overselling |

### The Cost of Non-Idempotent Systems

1. **Financial Loss** - Duplicate charges, refund processing
2. **Poor User Experience** - Confusion, support tickets
3. **Data Corruption** - Inconsistent state
4. **Engineering Overhead** - Manual fixes, reconciliation
5. **Lost Trust** - Customers leave

---

## 🎭 Idempotency Patterns

### Pattern 1: Naturally Idempotent Operations

Some operations are inherently idempotent:

```go
// SET operations (idempotent)
user.status = "active"
account.balance = 1000
settings.theme = "dark"

// DELETE operations (idempotent if delete of non-existent is OK)
DELETE users WHERE id = 123

// GET operations (idempotent - reading doesn't change state)
GET /api/users/123
```

### Pattern 2: Idempotency Keys

For non-idempotent operations, use unique keys to deduplicate:

```go
// Client generates unique key
idempotencyKey := uuid.New()

// Send with request
POST /api/payments
Headers:
  Idempotency-Key: "unique-key-abc-123"
Body:
  {
    "amount": 10000,
    "currency": "USD"
  }

// Server stores key and result
if seen(idempotencyKey) {
    return cachedResult(idempotencyKey)
}

result := processPayment(amount)
cache(idempotencyKey, result)
return result
```

### Pattern 3: State-Based Deduplication

Use current state to determine if operation is needed:

```go
// Check state first
if order.status == "completed" {
    return order  // Already processed
}

// Process only if needed
order.status = "completed"
processOrder(order)
```

### Pattern 4: Token-Based Deduplication

One-time tokens that can only be used once:

```go
// Generate token
token := generateOneTimeToken()

// Use token (can only work once)
if !consumeToken(token) {
    return error("Token already used")
}

performOperation()
```

---

## 🔄 Exactly-Once vs At-Least-Once

### At-Least-Once Delivery

**Guarantee:** Message will be delivered one or more times

```
Client sends payment request
↓
Network retry on failure
↓
Server might receive:
- 1 time (success)
- 2 times (retry after timeout)
- 3 times (network duplicate + retry)
```

**Challenge:** Your handler MUST be idempotent!

### Exactly-Once Processing

**Guarantee:** Message will be processed exactly one time

**How it works:**
1. Assign unique ID to each request
2. Track which IDs have been processed
3. Skip processing if ID seen before
4. Return cached result for duplicates

```go
// Exactly-once processing
func ProcessPayment(requestID string, amount int) Result {
    // Check if already processed
    if result := getProcessedResult(requestID); result != nil {
        return result  // Return cached result
    }
    
    // Process and store result
    result := chargeCustomer(amount)
    storeResult(requestID, result)
    return result
}
```

### With Restate

Restate provides **exactly-once processing** automatically:

```go
// Every invocation has a unique ID
// Restate deduplicates retries automatically
// You get exactly-once semantics for free!

func (PaymentService) Create(
    ctx restate.ObjectContext,
    req PaymentRequest,
) (PaymentResult, error) {
    // This ENTIRE function is idempotent
    // Retries return the same result
    // No duplicate charges!
    
    result, err := restate.Run(ctx, func(ctx restate.RunContext) (string, error) {
        return chargeCustomer(req.Amount)
    })
    
    return PaymentResult{ChargeID: result}, err
}
```

---

## 🛠️ Restate's Idempotency Features

### 1. Automatic Invocation Deduplication

Every invocation has a unique ID. Restate automatically deduplicates retries:

```go
// Client calls with same invocation ID
invocationID := "payment-abc-123"

// First call
curl -X POST http://localhost:8080/PaymentService/payment-1/Create \
  -H 'idempotency-key: payment-abc-123'
// → Processes payment, stores result

// Retry (same idempotency key)
curl -X POST http://localhost:8080/PaymentService/payment-1/Create \
  -H 'idempotency-key: payment-abc-123'
// → Returns cached result, NO duplicate charge!
```

**How it works:**
- Restate generates invocation ID from: service name + handler name + object key + idempotency key
- First invocation executes and journal is stored
- Subsequent invocations with same ID replay from journal
- Result is deterministic and consistent

### 2. Journaled Side Effects

Side effects wrapped in `restate.Run()` are journaled:

```go
// First invocation
result, err := restate.Run(ctx, func(ctx restate.RunContext) (string, error) {
    return callStripeAPI(amount)  // Executes, result journaled
})

// Retry of same invocation
result, err := restate.Run(ctx, func(ctx restate.RunContext) (string, error) {
    return callStripeAPI(amount)  // NOT executed! Returns journaled result
})
```

**Benefits:**
- External APIs called exactly once
- Deterministic replay
- No duplicate side effects
- Automatic retry safety

### 3. State-Based Deduplication

Virtual Objects provide state-based idempotency:

```go
func (OrderService) Create(
    ctx restate.ObjectContext,
    req OrderRequest,
) (Order, error) {
    orderID := restate.Key(ctx)
    
    // Check if order already exists
    existingOrder, err := restate.Get[*Order](ctx, "order")
    if err != nil {
        return Order{}, err
    }
    
    if existingOrder != nil {
        // Order already created, return it
        return *existingOrder, nil
    }
    
    // Create new order
    order := Order{
        ID:     orderID,
        Items:  req.Items,
        Status: "pending",
    }
    
    restate.Set(ctx, "order", order)
    return order, nil
}
```

### 4. Idempotency Key Support

Clients can provide explicit idempotency keys:

```go
// Client-provided key
restate.ServiceSend(ctx, "PaymentService", "Create").
    Send(payment, restate.WithIdempotencyKey("client-key-123"))

// Or via HTTP header
curl -H 'idempotency-key: my-unique-key' ...
```

---

## 📋 Idempotency Key Best Practices

### 1. Key Generation

```go
// ✅ GOOD - Client generates UUID
idempotencyKey := uuid.New().String()

// ✅ GOOD - Hash of request content
idempotencyKey := sha256(userId + timestamp + amount)

// ❌ BAD - Sequential numbers (predictable)
idempotencyKey := "payment-" + strconv.Itoa(counter++)

// ❌ BAD - Same key for different requests
idempotencyKey := "my-payment"  // Don't reuse!
```

### 2. Key Scope

Keys should be unique per logical operation:

```go
// ✅ GOOD - Unique per payment
key = "payment-" + userId + "-" + uuid()

// ❌ BAD - Same key for all payments
key = "payment-for-" + userId  // Can only pay once ever!
```

### 3. Key Storage

```go
// Client stores key with request state
type PendingPayment struct {
    IdempotencyKey string
    Amount         int
    Status         string  // "pending", "completed", "failed"
}

// Before retry, use same key
if payment.Status == "pending" {
    retryWithKey(payment.IdempotencyKey)
}
```

### 4. Key Expiration

How long should keys be remembered?

```go
// Short-lived operations (payments)
keyTTL := 24 * time.Hour

// Long-lived operations (subscriptions)
keyTTL := 30 * 24 * time.Hour

// Restate: retention configurable per service
restate.WithIdempotencyRetention(24 * time.Hour)
```

---

## ⚠️ Common Anti-Patterns

### Anti-Pattern 1: Assuming Network Reliability

```go
// ❌ WRONG ASSUMPTION
// "Network never duplicates requests"
// "Clients never retry"
// "I don't need idempotency"

// ✅ REALITY
// Networks are unreliable
// Clients WILL retry
// You MUST handle duplicates
```

### Anti-Pattern 2: Relying on External Idempotency

```go
// ❌ BAD - Trusting external service alone
func ChargeCustomer(amount int) error {
    // Stripe has idempotency, but handler doesn't!
    stripe.Charge(amount)
    incrementAnalyticsCounter()  // Not idempotent!
    sendEmail("Payment processed")  // Duplicate emails!
}

// ✅ GOOD - Wrap everything
func ChargeCustomer(ctx restate.ObjectContext, amount int) error {
    _, err := restate.Run(ctx, func(ctx restate.RunContext) (bool, error) {
        stripe.Charge(amount)  // Journaled
        return true, nil
    })
    
    _, err = restate.Run(ctx, func(ctx restate.RunContext) (bool, error) {
        incrementAnalyticsCounter()  // Journaled
        return true, nil
    })
    
    _, err = restate.Run(ctx, func(ctx restate.RunContext) (bool, error) {
        sendEmail("Payment processed")  // Journaled
        return true, nil
    })
    
    return err
}
```

### Anti-Pattern 3: Non-Deterministic Operations

```go
// ❌ BAD - Non-deterministic
func ProcessOrder(ctx restate.ObjectContext, order Order) error {
    // Random values differ on retries!
    trackingNumber := generateRandomTrackingNumber()
    
    // Timestamps differ on retries!
    processedAt := time.Now()
    
    // This breaks idempotency!
}

// ✅ GOOD - Deterministic
func ProcessOrder(ctx restate.ObjectContext, order Order) error {
    // Use Restate's deterministic random
    trackingNumber := restate.Rand(ctx).Int()
    
    // Use Restate's deterministic time
    processedAt, _ := restate.Run(ctx, func(ctx restate.RunContext) (time.Time, error) {
        return time.Now(), nil
    })
    
    // Same results on retries!
}
```

### Anti-Pattern 4: Mutable Counters

```go
// ❌ BAD - Counter increments on every retry
func RecordPageView(ctx restate.ObjectContext) error {
    views, _ := restate.Get[int](ctx, "views")
    restate.Set(ctx, "views", views+1)  // Increments on retry!
}

// ✅ GOOD - Idempotent recording
func RecordPageView(ctx restate.ObjectContext, viewID string) error {
    // Check if already recorded
    views, _ := restate.Get[map[string]bool](ctx, "recorded_views")
    if views[viewID] {
        return nil  // Already counted
    }
    
    views[viewID] = true
    restate.Set(ctx, "recorded_views", views)
    
    count, _ := restate.Get[int](ctx, "total_views")
    restate.Set(ctx, "total_views", count+1)
}
```

---

## 🎨 Design Patterns for Idempotency

### Pattern 1: Check-Then-Set

```go
func UpdateStatus(ctx restate.ObjectContext, newStatus string) error {
    currentStatus, _ := restate.Get[string](ctx, "status")
    
    // Only update if different
    if currentStatus == newStatus {
        return nil  // Already set, idempotent
    }
    
    restate.Set(ctx, "status", newStatus)
    return nil
}
```

### Pattern 2: Tombstone Pattern

```go
func DeleteOrder(ctx restate.ObjectContext, orderID string) error {
    order, _ := restate.Get[*Order](ctx, "order")
    
    if order == nil {
        return nil  // Already deleted, idempotent
    }
    
    // Don't actually delete, mark as deleted
    order.Deleted = true
    order.DeletedAt = time.Now()
    restate.Set(ctx, "order", order)
    
    return nil
}
```

### Pattern 3: Event Sourcing

```go
func ApplyEvent(ctx restate.ObjectContext, event Event) error {
    events, _ := restate.Get[[]Event](ctx, "events")
    
    // Check if event already applied
    for _, e := range events {
        if e.ID == event.ID {
            return nil  // Already applied
        }
    }
    
    // Apply new event
    events = append(events, event)
    restate.Set(ctx, "events", events)
    
    // Update derived state
    state := computeState(events)
    restate.Set(ctx, "state", state)
    
    return nil
}
```

---

## 📊 Idempotency in Practice

### E-Commerce Order

```go
type OrderService struct{}

func (OrderService) CreateOrder(
    ctx restate.ObjectContext,
    req CreateOrderRequest,
) (Order, error) {
    orderID := restate.Key(ctx)
    
    // Check if order exists (state-based deduplication)
    existingOrder, err := restate.Get[*Order](ctx, "order")
    if err != nil {
        return Order{}, err
    }
    
    if existingOrder != nil {
        // Order already created
        return *existingOrder, nil
    }
    
    // Reserve inventory (idempotent call)
    _, err = restate.Service[bool](
        ctx, "InventoryService", "Reserve",
    ).Request(req.Items, restate.WithIdempotencyKey(orderID+"-inventory"))
    if err != nil {
        return Order{}, fmt.Errorf("inventory unavailable: %w", err)
    }
    
    // Charge customer (idempotent call)
    chargeID, err := restate.Service[string](
        ctx, "PaymentService", "Charge",
    ).Request(req.Payment, restate.WithIdempotencyKey(orderID+"-payment"))
    if err != nil {
        // Release inventory on payment failure
        restate.ServiceSend(ctx, "InventoryService", "Release").
            Send(req.Items)
        return Order{}, fmt.Errorf("payment failed: %w", err)
    }
    
    // Create order
    order := Order{
        ID:        orderID,
        Items:     req.Items,
        ChargeID:  chargeID,
        Status:    "confirmed",
        CreatedAt: time.Now(),
    }
    
    restate.Set(ctx, "order", order)
    
    // Send confirmation (idempotent)
    _, err = restate.Run(ctx, func(ctx restate.RunContext) (bool, error) {
        sendOrderConfirmation(order)
        return true, nil
    })
    
    return order, nil
}
```

**Idempotency guarantees:**
- ✅ Inventory reserved exactly once
- ✅ Customer charged exactly once
- ✅ Order created exactly once
- ✅ Confirmation email sent exactly once
- ✅ Safe to retry entire operation

---

## ✅ Summary

### Key Takeaways

1. **Idempotency is Critical** - Not optional in distributed systems
2. **Retries are Inevitable** - Networks fail, clients timeout
3. **Use Restate Features** - Automatic deduplication, journaling
4. **Wrap Side Effects** - Use `restate.Run()` for external calls
5. **Leverage State** - Check before executing
6. **Idempotency Keys** - Client-provided or auto-generated
7. **Test Retry Scenarios** - Verify duplicate handling

### Mental Model

```
Non-Idempotent System:
Request → Process → Side Effect 1
Request → Process → Side Effect 2  ❌
Request → Process → Side Effect 3  ❌

Idempotent System:
Request → Process → Side Effect 1
Request → Return Cached Result  ✅
Request → Return Cached Result  ✅
```

### Restate Advantages

| Challenge | Restate Solution |
|-----------|-----------------|
| Duplicate requests | Invocation ID deduplication |
| Multiple side effects | `restate.Run()` journaling |
| State inconsistency | Durable state with transactions |
| Non-determinism | Deterministic random/time |
| Complex retry logic | Automatic retry with replay |

---

## 🚀 Next Steps

You now understand idempotency theory!

👉 **Continue to [Hands-On Tutorial](./02-hands-on.md)**

Build a real payment service with idempotent operations!

---

**Questions?** Review this document or check the [module README](./README.md).
