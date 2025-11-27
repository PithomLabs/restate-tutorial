# Module 08: External Integration

> **Master integrating external APIs and services with Restate**

## 🎯 Learning Objectives

By completing this module, you will:
- ✅ Safely integrate with external REST APIs
- ✅ Handle webhooks from third-party services
- ✅ Implement resilient HTTP clients
- ✅ Manage API rate limits and retries
- ✅ Process asynchronous callbacks
- ✅ Build adapter patterns for external services

## 📚 Module Structure

### 1. [Concepts](./01-concepts.md) (~30 min)
Learn integration patterns and best practices:
- External API integration challenges
- Journaling external calls with `restate.Run()`
- Webhook handling patterns
- Rate limiting and backoff strategies
- Error handling and circuit breakers
- API versioning and compatibility

### 2. [Hands-On Tutorial](./02-hands-on.md) (~50 min)
Build an **E-Commerce Integration Service**:
- Stripe payment integration
- SendGrid email notifications
- Shippo shipping label creation
- Webhook processors for all services
- Unified order orchestration

### 3. [Validation](./03-validation.md) (~35 min)
Test your integration:
- API call verification
- Webhook processing tests
- Retry and failure scenarios
- Rate limit handling
- End-to-end order flow

### 4. [Exercises](./04-exercises.md) (~60 min)
Practice with real-world scenarios:
- SMS notifications via Twilio
- Slack integration for alerts
- GitHub API for deployments
- Custom webhook processors
- Multi-service orchestration

## 🎓 Prerequisites

Before starting this module:
- ✅ Completed Module 01 (Foundation)
- ✅ Completed Module 02 (Side Effects)
- ✅ Completed Idempotency module
- ✅ Basic understanding of REST APIs
- ✅ Familiarity with HTTP requests

## 💡 Why External Integration Matters

### The Challenge

Distributed systems must integrate with external services:

```
Your Service
    ↓
  ❌ Direct API Call (risky!)
    ↓
External Service (Stripe, SendGrid, etc.)

Problems:
- Network failures
- Timeouts
- Rate limits
- Duplicate calls
- Non-idempotent APIs
- Callback coordination
```

### The Restate Solution

```
Your Service
    ↓
  restate.Run()  ✅ Journaled!
    ↓
External Service

Benefits:
- Automatic retries
- Exactly-once execution
- Failure recovery
- State coordination
- Webhook handling
```

## 🏗️ What You'll Build

An **E-Commerce Integration Hub** that coordinates:

### External Services
- 💳 **Stripe** - Payment processing
- 📧 **SendGrid** - Email notifications
- 📦 **Shippo** - Shipping labels
- 💬 **Slack** - Team notifications

### Features
- Create order with payment
- Send confirmation emails
- Generate shipping labels
- Process webhooks from all services
- Handle failures and retries
- Coordinate multi-service workflows

### Architecture
```
Client Request
    ↓
OrderOrchestrator (Restate Service)
    ├─→ Stripe (payment)
    ├─→ SendGrid (email)
    ├─→ Shippo (shipping)
    └─→ Slack (notification)
    ↓
Webhooks (async updates)
    ↓
WebhookProcessor (Restate Service)
    ↓
Order Status Updates
```

## 📊 Module Outline

```
08-external-integration/
├── README.md                    # This file
├── 01-concepts.md              # Integration patterns
├── 02-hands-on.md              # E-commerce integration
├── 03-validation.md            # Testing guide
├── 04-exercises.md             # Practice problems
├── code/                       # Working implementation
│   ├── main.go
│   ├── types.go
│   ├── stripe_client.go        # Stripe integration
│   ├── sendgrid_client.go      # Email integration
│   ├── shippo_client.go        # Shipping integration
│   ├── order_orchestrator.go  # Multi-service orchestration
│   ├── webhook_processor.go   # Webhook handling
│   ├── go.mod
│   └── README.md
└── solutions/                  # Exercise solutions
    ├── twilio_integration.go
    ├── slack_integration.go
    └── README.md
```

## 🎯 Key Concepts Covered

### 1. API Integration Patterns
- HTTP client best practices
- Authentication handling
- Request/response mapping
- Error handling strategies
- Retry logic

### 2. Journaling External Calls
- Using `restate.Run()` for API calls
- Deterministic execution
- Idempotent wrappers
- State management

### 3. Webhook Processing
- Webhook verification
- Signature validation
- Idempotent processing
- Callback coordination

### 4. Resilience Patterns
- Exponential backoff
- Circuit breakers
- Rate limit handling
- Timeout management
- Fallback strategies

## 🚀 Quick Start

### 1. Read Concepts
```bash
less 01-concepts.md
```

### 2. Set Up Environment

```bash
# Clone the code
cd code/

# Set environment variables (optional for mock mode)
export STRIPE_API_KEY="your_test_key"
export SENDGRID_API_KEY="your_api_key"
export SHIPPO_API_KEY="your_api_key"

# Or run in mock mode (no real API calls)
export MOCK_MODE=true
```

### 3. Run the Integration Service

```bash
go mod download
go run .
```

### 4. Test Integration

```bash
# Create order (triggers all integrations)
curl -X POST http://localhost:8080/OrderOrchestrator/order-001/ProcessOrder \
  -H 'Content-Type: application/json' \
  -d '{
    "items": [{"productId": "prod-123", "quantity": 2, "price": 5000}],
    "customer": {
      "email": "customer@example.com",
      "name": "Alice Smith"
    },
    "shipping": {
      "address": "123 Main St",
      "city": "San Francisco",
      "state": "CA",
      "zip": "94105"
    }
  }'
```

## ⚠️ Common Pitfalls

### Anti-Pattern 1: Calling APIs Directly

```go
// ❌ BAD - Not journaled, will retry on failure
func ProcessPayment(ctx restate.ObjectContext, amount int) error {
    chargeID, err := stripe.Charge(amount)  // Direct call!
    if err != nil {
        return err
    }
    // On retry: charges customer again! 💸💸
}
```

### Anti-Pattern 2: Not Handling Webhooks Idempotently

```go
// ❌ BAD - Processes webhook multiple times
func HandleWebhook(ctx restate.ObjectContext, webhook Webhook) error {
    updateDatabase(webhook)  // Not idempotent!
    sendNotification(webhook)  // Duplicate notifications!
}
```

### Anti-Pattern 3: No Error Handling

```go
// ❌ BAD - Doesn't handle failures
func SendEmail(email Email) {
    sendgrid.Send(email)  // What if this fails?
    // No retry, no logging, no fallback
}
```

## ✅ Best Practices

### 1. Always Journal External Calls

```go
// ✅ GOOD
chargeID, err := restate.Run(ctx, func(ctx restate.RunContext) (string, error) {
    return stripe.Charge(amount)  // Journaled!
})
// On retry: returns journaled result, no duplicate charge
```

### 2. Implement Idempotent Webhooks

```go
// ✅ GOOD
existing, _ := restate.Get[*WebhookResult](ctx, "result")
if existing != nil {
    return existing  // Already processed
}
// Process webhook...
```

### 3. Handle Errors Gracefully

```go
// ✅ GOOD
result, err := restate.Run(ctx, func(ctx restate.RunContext) (T, error) {
    resp, err := externalAPI.Call()
    if err != nil {
        if isRetryable(err) {
            return T{}, err  // Restate retries
        }
        return T{}, restate.TerminalError(err)  // Stop retrying
    }
    return resp, nil
})
```

### 4. Use Adapters for Services

```go
// ✅ GOOD - Encapsulate external service logic
type StripeAdapter struct{}

func (a *StripeAdapter) Charge(ctx restate.RunContext, amount int) (string, error) {
    // Handle auth, retries, errors internally
    client := stripe.NewClient(apiKey)
    return client.Charge(amount)
}
```

## 🔗 Related Modules

- **Module 02: Side Effects** - `restate.Run()` for external calls
- **Idempotency Module** - Handling duplicates
- **Module 06: Sagas** - Compensating transactions
- **Module 10: Observability** - Monitoring external calls

## 📈 Success Criteria

You've mastered this module when you can:
- [x] Integrate external APIs safely with journaling
- [x] Process webhooks idempotently
- [x] Handle failures and retries correctly
- [x] Coordinate multiple external services
- [x] Implement resilience patterns
- [x] Design clean integration adapters

## 🎓 Learning Path

**Current Module:** External Integration  
**Previous:** [Idempotency](../07-idempotency/README.md)  
**Next:** [Module 09 - Microservices Orchestration](../09-microservices/README.md)

---

## 🚀 Let's Get Started!

Ready to build resilient integrations?

👉 **Start with [Concepts](./01-concepts.md)** to learn integration patterns!

---

**Questions?** Review [previous modules](../README.md) or check the [main README](../README.md).
