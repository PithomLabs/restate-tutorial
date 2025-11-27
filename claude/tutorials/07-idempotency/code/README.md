# Idempotency Module - Code

> **Complete working implementation of idempotent payment service**

## 📁 Files

- **`types.go`** - Payment and refund data structures
- **`payment_service.go`** - Virtual object with idempotent handlers
- **`gateway.go`** - Mock payment gateway (simulates Stripe, etc.)
- **`main.go`** - Server setup and registration
- **`go.mod`** - Go module dependencies

## 🚀 Quick Start

### 1. Install Dependencies

```bash
go mod tidy
```

### 2. Start Restate Server

```bash
docker run --name restate_dev --rm \
  -p 8080:8080 -p 9070:9070 -p 9091:9091 \
  --add-host=host.docker.internal:host-gateway \
  docker.io/restatedev/restate:latest
```

### 3. Start Payment Service

```bash
go run .
```

### 4. Register with Restate

```bash
curl -X POST http://localhost:9070/deployments \
  -H 'Content-Type: application/json' \
  -d '{"uri": "http://host.docker.internal:9090"}'
```

## 💳 Usage Examples

### Create Payment

```bash
curl -X POST http://localhost:8080/PaymentService/payment-001/CreatePayment \
  -H 'Content-Type: application/json' \
  -d '{
    "amount": 10000,
    "currency": "USD",
    "description": "Premium subscription",
    "customerId": "cust-alice"
  }'
```

### Test Idempotency (Duplicate Request)

```bash
# Send same request again - should return same result, no duplicate charge
curl -X POST http://localhost:8080/PaymentService/payment-001/CreatePayment \
  -H 'Content-Type: application/json' \
  -d '{
    "amount": 10000,
    "currency": "USD",
    "description": "Premium subscription",
    "customerId": "cust-alice"
  }'
```

### Get Payment Status

```bash
curl -X POST http://localhost:8080/PaymentService/payment-001/GetPayment \
  -H 'Content-Type: application/json' \
  -d '{}'
```

### Refund Payment

```bash
curl -X POST http://localhost:8080/PaymentService/payment-001/RefundPayment \
  -H 'Content-Type: application/json' \
  -d '{
    "reason": "Customer request",
    "amount": 10000
  }'
```

## 🎓 Key Patterns

### 1. State-Based Deduplication

The service checks if a payment already exists before processing:

```go
existingPayment, _ := restate.Get[*Payment](ctx, stateKeyPayment)
if existingPayment != nil {
    return existingPayment  // Idempotent!
}
```

### 2. Journaled Side Effects

External gateway calls are wrapped in `restate.Run()`:

```go
chargeResp, err := restate.Run(ctx, func(ctx restate.RunContext) (ChargeResponse, error) {
    return gateway.Charge(amount, currency, customerID), nil
})
// On retry: returns journaled result, NO duplicate gateway call!
```

### 3. Status Tracking

Payments track their lifecycle:

```go
"pending" → "completed" (or "failed")
```

### 4. Refund Protection

Refunds check for existing refund state:

```go
existingRefund, _ := restate.Get[*RefundResult](ctx, stateKeyRefund)
if existingRefund != nil {
    return existingRefund  // No duplicate refunds!
}
```

## ✅ Idempotency Guarantees

- ✅ **Same payment ID** = Same result
- ✅ **No duplicate charges** - Gateway called exactly once
- ✅ **No duplicate refunds** - Refund processed exactly once
- ✅ **Safe retries** - Network failures handled gracefully
- ✅ **Deterministic execution** - Same input = Same output

## 🔑 Virtual Object Key Strategy

Payment ID is used as the virtual object key:

```
PaymentService/payment-001  → Isolated state
PaymentService/payment-002  → Different isolated state
```

This provides:
- State isolation per payment
- Concurrent processing of different payments
- Sequential processing of operations on same payment

## 📚 Related Documentation

- [Hands-On Tutorial](../02-hands-on.md) - Step-by-step guide
- [Concepts](../01-concepts.md) - Idempotency theory
- [Validation](../03-validation.md) - Testing idempotency
- [Exercises](../04-exercises.md) - Practice problems

---

**Module: Idempotency** | [Main README](../README.md)
