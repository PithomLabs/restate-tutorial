# Module: Idempotency

> **Master idempotent operations for bulletproof distributed systems**

## 🎯 Learning Objectives

By completing this module, you will:
- ✅ Understand what idempotency means and why it matters
- ✅ Implement idempotent handlers with idempotency keys
- ✅ Handle duplicate requests safely
- ✅ Use Restate's built-in idempotency features
- ✅ Design APIs for safe retries
- ✅ Prevent duplicate payments and side effects

## 📚 Module Structure

### 1. [Concepts](./01-concepts.md) (~25 min)
Learn the theory behind idempotency:
- What is idempotency?
- Why distributed systems need it
- Exactly-once vs at-least-once semantics
- Idempotency keys and deduplication
- Common anti-patterns

### 2. [Hands-On Tutorial](./02-hands-on.md) (~45 min)
Build a **Payment Processing Service**:
- Idempotent payment creation
- Duplicate request handling
- Idempotency key management
- Testing retry scenarios
- Real-world payment workflows

### 3. [Validation](./03-validation.md) (~30 min)
Test your implementation:
- Verify idempotent behavior
- Test duplicate requests
- Simulate network retries
- Validate payment safety
- Integration testing

### 4. [Exercises](./04-exercises.md) (~60 min)
Practice with challenges:
- Idempotent order creation
- Refund processing
- Email sending deduplication
- Custom idempotency strategies
- Advanced scenarios

## 🎓 Prerequisites

Before starting this module:
- ✅ Completed Module 01 (Foundation)
- ✅ Completed Module 04 (Virtual Objects)
- ✅ Basic understanding of REST APIs
- ✅ Familiarity with payment systems (helpful)

## 💡 Why Idempotency Matters

### The Problem

```go
// ❌ NON-IDEMPOTENT - Running twice charges twice!
func CreatePayment(amount int) error {
    chargeCustomer(amount)  // Charges customer
    return nil
}
```

**What happens if:**
- Client timeout causes retry?
- Network duplicates the request?
- User clicks "Pay" twice?

**Result:** Customer charged multiple times! 💸💸

### The Solution

```go
// ✅ IDEMPOTENT - Safe to retry
func CreatePayment(ctx restate.ObjectContext, req PaymentRequest) error {
    // Restate automatically deduplicates based on invocation ID
    // OR you can use explicit idempotency keys
    
    chargeCustomer(req.Amount)  // Executes exactly once
    return nil
}
```

**With idempotency:**
- Same request = Same result
- Safe retries
- No duplicate charges
- Predictable behavior

## 🏗️ What You'll Build

A **Payment Processing Service** with:

### Features
- 💳 Create payments (idempotent)
- 💰 Process refunds (idempotent)
- 📊 Get payment status
- 🔄 Automatic retry handling
- 🛡️ Duplicate prevention

### Architecture
```
Client Request (with idempotency key)
    ↓
Restate Ingress (deduplication)
    ↓
PaymentService (virtual object, key = payment_id)
    ↓
External Payment Gateway (Stripe, etc.)
```

### Idempotency Strategies
1. **Restate Invocation IDs** - Automatic deduplication
2. **Idempotency Keys** - Explicit client-provided keys
3. **Virtual Object Keys** - State-based deduplication

## 📊 Module Outline

```
07-idempotency/
├── README.md                    # This file
├── 01-concepts.md              # Idempotency theory
├── 02-hands-on.md              # Payment service tutorial
├── 03-validation.md            # Testing guide
├── 04-exercises.md             # Practice problems
├── code/                       # Working implementation
│   ├── main.go
│   ├── types.go
│   ├── payment_service.go
│   ├── gateway.go              # Mock payment gateway
│   ├── go.mod
│   └── README.md
└── solutions/                  # Exercise solutions
    ├── exercise1_order_service.go
    ├── exercise2_refund_service.go
    └── README.md
```

## 🎯 Key Concepts Covered

### 1. Idempotency Fundamentals
- Definition and importance
- Exactly-once semantics
- Side effect management
- State immutability

### 2. Restate Idempotency Features
- Automatic invocation deduplication
- Idempotency key support
- Journaling for determinism
- State-based deduplication

### 3. API Design Patterns
- Idempotent HTTP endpoints
- Client retry strategies
- Idempotency headers
- Best practices

### 4. Real-World Applications
- Payment processing
- Order creation
- Email sending
- External API calls

## 🚀 Quick Start

### 1. Read Concepts
```bash
less 01-concepts.md
```

### 2. Build Payment Service
```bash
cd code/
go mod download
go run .
```

### 3. Test Idempotency
```bash
# Send same request twice with same idempotency key
curl -X POST http://localhost:8080/PaymentService/payment-123/Create \
  -H 'Content-Type: application/json' \
  -H 'idempotency-key: unique-key-123' \
  -d '{"amount": 10000, "currency": "USD"}'

# Second call returns same result, no duplicate charge!
curl -X POST http://localhost:8080/PaymentService/payment-123/Create \
  -H 'Content-Type: application/json' \
  -H 'idempotency-key: unique-key-123' \
  -d '{"amount": 10000, "currency": "USD"}'
```

## ⚠️ Common Pitfalls

### Anti-Pattern 1: Non-Idempotent Side Effects
```go
// ❌ BAD - Counter increments on every retry
func ProcessOrder(ctx restate.ObjectContext, order Order) error {
    incrementOrderCount()  // Not journaled!
    return nil
}
```

### Anti-Pattern 2: Relying on External Idempotency
```go
// ❌ BAD - External service might not be idempotent
func ChargeCustomer(ctx restate.ObjectContext, amount int) error {
    // Stripe might be idempotent, but YOUR handler isn't!
    _, err := restate.Run(ctx, func(ctx restate.RunContext) (bool, error) {
        return stripe.Charge(amount), nil
    })
    return err
}
```

### Anti-Pattern 3: Mutable State Without Keys
```go
// ❌ BAD - State mutations aren't protected
count := restate.Get[int](ctx, "count")
restate.Set(ctx, "count", count+1)  // Not idempotent!
```

## ✅ Best Practices

1. **Always use idempotency keys** for critical operations
2. **Leverage Restate's journaling** - side effects in `restate.Run()` are automatic
3. **Design deterministic handlers** - same input = same output
4. **Document idempotent APIs** - make it clear to clients
5. **Test retry scenarios** - verify duplicate requests work correctly

## 🔗 Related Modules

- **Module 01: Foundation** - Durable execution basics
- **Module 02: Side Effects** - `restate.Run()` for idempotent side effects
- **Module 04: Virtual Objects** - State-based deduplication
- **Module 05: Workflows** - Idempotent workflow steps

## 📈 Success Criteria

You've mastered this module when you can:
- [x] Explain idempotency and its importance
- [x] Implement idempotent handlers with Restate
- [x] Use idempotency keys effectively
- [x] Handle duplicate requests safely
- [x] Design APIs for safe retries
- [x] Test idempotent behavior

## 🎓 Learning Path

**Current Module:** Idempotency
**Previous:** [Module 07 - Testing](../07-testing/README.md)
**Next:** [Module 08 - External Integration](../08-external-integration/README.md)

---

## 🚀 Let's Get Started!

Ready to build bulletproof distributed systems?

👉 **Start with [Concepts](./01-concepts.md)** to understand idempotency fundamentals!

---

**Questions?** Review [previous modules](../README.md) or check the [main README](../README.md).
