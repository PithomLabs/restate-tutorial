# Exercise Solutions

> **Reference implementations for idempotency exercises**

## 📁 Solutions Included

| Exercise | File | Difficulty |
|----------|------|------------|
| 1. Idempotent Order Service | `exercise1_order_service.go` | ⭐ |
| 2. Email Service | `exercise2_email_service.go` | ⭐⭐ |
| 5. Webhook Handler | `exercise5_webhook_handler.go` | ⭐⭐ |

## 🎯 How to Use

### Option 1: Study the Code

Review the solution files to understand idempotency patterns:
- State-based deduplication
- Journaled side effects
- Status tracking
- Error handling

### Option 2: Test the Solutions

1. Copy solution file to main `code/` directory
2. Update `main.go` to register the service
3. Run and test

Example:
```bash
# Copy solution
cp solutions/exercise1_order_service.go ../code/

# Update main.go to add:
# restateServer.Bind(restate.Reflect(OrderService{}))

# Run
cd ../code && go run .
```

### Option 3: Compare with Your Implementation

After solving exercises yourself:
1. Compare your solution with the reference
2. Identify differences in approach
3. Test both implementations
4. Learn from alternative patterns

## 📚 Key Patterns Demonstrated

### Pattern 1: State-Based Deduplication

**Before creating resources, check if they exist:**

```go
existing, _ := restate.Get[*Resource](ctx, "resource")
if existing != nil {
    return *existing  // Idempotent!
}
```

### Pattern 2: Journaled Side Effects

**Wrap external calls in `restate.Run()`:**

```go
result, err := restate.Run(ctx, func(ctx restate.RunContext) (Response, error) {
    return externalAPI.Call(params)
})
// On retry: returns journaled result, API not called again
```

### Pattern 3: Status Tracking

**Use explicit status fields:**

```go
type Order struct {
    Status string  // "pending" → "confirmed" → "shipped"
}

// Validate state transitions
if order.Status != "pending" {
    return error("cannot confirm non-pending order")
}
```

### Pattern 4: Idempotency Keys

**Generate deterministic sub-operation keys:**

```go
// Compose parent + child operation
idempotencyKey := parentID + "-" + operation

restate.Service[T](ctx, "Service", "Handler").
    Request(req, restate.WithIdempotencyKey(idempotencyKey))
```

## 🎓 Learning Tips

### 1. Understand the "Why"

Don't just copy code - understand:
- Why is this operation idempotent?
- What happens on retry?
- Where are side effects journaled?
- How is state used for deduplication?

### 2. Test Extensively

For each solution:
- Send duplicate requests
- Test concurrent calls
- Simulate failures and retries
- Verify no duplicate operations

### 3. Adapt to Your Use Case

These solutions are examples. Adapt them:
- Change data structures
- Add validation
- Extend functionality
- Apply to your domain

## 🔍 Solution Highlights

### Exercise 1: Order Service

**Key Features:**
- State-based order existence check
- Deterministic order number generation
- Idempotent cancellation
- Status transition validation

**Idempotency Techniques:**
```go
// Check before create
existingOrder, _ := restate.Get[*Order](ctx, "order")
if existingOrder != nil {
    return existingOrder
}

// Deterministic order number
orderNumber := fmt.Sprintf("ORD-%d", restate.Rand(ctx).Int())
```

### Exercise 2: Email Service

**Key Features:**
- Journaled SMTP calls
- Send status tracking
- Safe resend logic
- Failure handling

**Idempotency Techniques:**
```go
// Check if already sent
existing, _ := restate.Get[*EmailRecord](ctx, "email")
if existing != nil && existing.Status == "sent" {
    return existing
}

// Journal email send
msgID, err := restate.Run(ctx, func(ctx restate.RunContext) (string, error) {
    return sendViaProvider(recipient, subject, body)
})
```

### Exercise 5: Webhook Handler

**Key Features:**
- Webhook ID-based deduplication
- Immediate return for duplicates
- Type-specific processing
- Result storage

**Idempotency Techniques:**
```go
// Check if webhook processed
result, _ := restate.Get[*WebhookResult](ctx, "result")
if result != nil {
    ctx.Log().Info("Webhook already processed")
    return result  // Idempotent!
}

// Process and store
processWebhook(webhook)
restate.Set(ctx, "result", result)
```

## ⚠️ Common Mistakes to Avoid

### Mistake 1: Not Checking State First

```go
// ❌ BAD
func Create(ctx restate.ObjectContext, req Request) error {
    resource := createResource(req)
    restate.Set(ctx, "resource", resource)
    // Creates duplicate on retry!
}

// ✅ GOOD
func Create(ctx restate.ObjectContext, req Request) error {
    if existing, _ := restate.Get[*Resource](ctx, "resource"); existing != nil {
        return existing
    }
    resource := createResource(req)
    restate.Set(ctx, "resource", resource)
}
```

### Mistake 2: Side Effects Outside `restate.Run()`

```go
// ❌ BAD
func Send(ctx restate.ObjectContext, email Email) error {
    sendEmail(email)  // Sends on every retry!
    restate.Set(ctx, "sent", true)
}

// ✅ GOOD
func Send(ctx restate.ObjectContext, email Email) error {
    _, err := restate.Run(ctx, func(ctx restate.RunContext) (bool, error) {
        return sendEmail(email), nil  // Journaled, executes once
    })
    restate.Set(ctx, "sent", true)
}
```

### Mistake 3: Non-Deterministic Values

```go
// ❌ BAD
func Create(ctx restate.ObjectContext) error {
    id := uuid.New()  // Different on retry!
    timestamp := time.Now()  // Different on retry!
}

// ✅ GOOD
func Create(ctx restate.ObjectContext) error {
    id := restate.UUID(ctx)  // Deterministic
    timestamp, _ := restate.Run(ctx, func(ctx restate.RunContext) (time.Time, error) {
        return time.Now(), nil  // Journaled
    })
}
```

## 🚀 Next Steps

After reviewing solutions:

1. ✅ Compare with your implementations
2. ✅ Test the reference solutions
3. ✅ Understand the patterns used
4. ✅ Apply learnings to your code
5. ✅ Try the bonus challenges

## 🎓 Additional Resources

- [Module README](../README.md) - Module overview
- [Concepts](../01-concepts.md) - Idempotency theory
- [Hands-On](../02-hands-on.md) - Payment service tutorial
- [Validation](../03-validation.md) - Testing guide
- [Exercises](../04-exercises.md) - Practice problems

---

**Happy Learning!** 🎉

Questions? Review the module documentation or experiment with the code!
