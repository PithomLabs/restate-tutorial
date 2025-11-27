# E-Commerce Integration Service

> **Complete e-commerce order processing with external API integrations**

## 🎯 Overview

This service demonstrates how to safely integrate with multiple external APIs using Restate's journaling capabilities. It orchestrates payment processing (Stripe), email notifications (SendGrid), and shipping label creation (Shippo) in a resilient, idempotent manner.

## 🏗️ Architecture

```
Client Request
   ↓
OrderOrchestrator (Virtual Object)
   ├─→ Stripe Client (payment)
   ├─→ Shippo Client (shipping label)
   └─→ SendGrid Client (email)
   ↓
Order Status in State
```

## 📁 File Structure

```
code/
├── main.go                 # Server setup and registration
├── types.go                # Data structures
├── order_orchestrator.go   # Main orchestration logic
├── webhook_processor.go    # Webhook handling
├── stripe_client.go        # Stripe integration
├── sendgrid_client.go      # SendGrid integration
├── shippo_client.go        # Shippo integration
└── go.mod                  # Dependencies
```

## 🚀 Quick Start

### 1. Install Dependencies

```bash
go mod tidy
```

###2. Run in Mock Mode

```bash
export MOCK_MODE=true
go run .
```

### 3. Register with Restate

```bash
curl -X POST http://localhost:9070/deployments \
  -H 'Content-Type: application/json' \
  -d '{"uri": "http://host.docker.internal:9090"}'
```

### 4. Process an Order

```bash
curl -X POST http://localhost:8080/OrderOrchestrator/order-001/ProcessOrder \
  -H 'Content-Type: application/json' \
  -d '{
    "items": [{"productId": "prod-123", "quantity": 2, "price": 2500}],
    "customer": {"email": "alice@example.com", "name": "Alice Smith"},
    "shipping": {"street": "123 Main St", "city": "SF", "state": "CA", "zip": "94105", "country": "US"}
  }'
```

## 🎯 Key Integration Patterns

### 1. Journaled API Calls

All external API calls are wrapped in `restate.Run()`:

```go
chargeResp, err := restate.Run(ctx, func(ctx restate.RunContext) (StripeChargeResponse, error) {
    return stripeClient.CreateCharge(ctx, chargeRequest)
})
```

**Benefits:**
- API called exactly once
- Result journaled for replay
- Safe retries on failures

### 2. Service Orchestration

Multiple services coordinated in order:

1. **Charge customer** (Stripe) - Must succeed
2. **Create shipping label** (Shippo) - Optional
3. **Send confirmation email** (SendGrid) - Optional

Each step is independent and journaled.

### 3. State-Based Idempotency

Order checked before processing:

```go
existingOrder, _ := restate.Get[*Order](ctx, "order")
if existingOrder != nil {
    return existing result  // Already processed!
}
```

### 4. Mock Mode for Development

Environment variable controls real vs mock:

```go
if c.mockMode {
    return c.mockCharge(req)  // No real API call
}
return c.realCharge(req)  // Real Stripe API
```

## 📊 Order States

```
pending → paid → confirmed
   ↓
payment_failed
```

- **pending**: Order created, payment in progress
- **paid**: Payment successful
- **confirmed**: Shipping label created, email sent
- **payment_failed**: Payment declined or failed

## 🔌 External Services

### Stripe (Payment Processing)

- **API**: `CreateCharge` - Process payment
- **API**: `RefundCharge` - Refund payment
- **Mock Mode**: 95% success rate, simulated delays

### SendGrid (Email Notifications)

- **API**: `SendEmail` - Send confirmation email
- **Mock Mode**: Always succeeds

### Shippo (Shipping Labels)

- **API**: `CreateLabel` - Generate shipping label
- **Mock Mode**: Returns mock tracking number

## 🎓 What This Demonstrates

1. **Journaling External Calls** - All API calls wrapped in `restate.Run()`
2. **Multi-Service Orchestration** - Coordinating 3 external APIs
3. **Failure Handling** - Payment failure stops workflow, label failure continues
4. **Idempotent Processing** - Safe retries at every step
5. **State Management** - Order state tracks progress
6. **Mock Mode** - Development without real APIs

## ⚙️ Configuration

### Environment Variables

- `MOCK_MODE=true` - Use mock implementations (recommended for learning)
- `STRIPE_API_KEY` - Stripe API key (for real mode)
- `SENDGRID_API_KEY` - SendGrid API key (for real mode)
- `SHIPPO_API_KEY` - Shippo API key (for real mode)

## 🧪 Testing

See [validation guide](../03-validation.md) for comprehensive testing scenarios including:
- Basic order processing
- Idempotency verification
- Concurrent requests
- Webhook processing
- Failure recovery

## 📚 Related Documentation

- [Module README](../README.md)
- [Concepts](../01-concepts.md)
- [Hands-On Tutorial](../02-hands-on.md)
- [Validation Guide](../03-validation.md)

---

**Ready to integrate!** 🚀
