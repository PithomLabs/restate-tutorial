# Validation: Testing External Integration

> **Verify your e-commerce integration service**

## 🎯 Validation Goals

- ✅ Verify external API calls are journaled
- ✅ Test idempotent order processing
- ✅ Validate webhook handling
- ✅ Ensure failure recovery
- ✅ Confirm no duplicate operations

## 📋 Prerequisites

- ✅ Restate server running
- ✅ Integration service running (`MOCK_MODE=true`)
- ✅ Service registered with Restate

## 🧪 Test Scenarios

### Scenario 1: Complete Order Flow

**Test:** Process order with all integrations

```bash
curl -X POST http://localhost:8080/OrderOrchestrator/test-order-1/ProcessOrder \
  -H 'Content-Type: application/json' \
  -d '{
    "items": [
      {"productId": "prod-123", "quantity": 2, "price": 2500}
    ],
    "customer": {
      "email": "test@example.com",
      "name": "Test User"
    },
    "shipping": {
      "street": "123 Test St",
      "city": "San Francisco",
      "state": "CA",
      "zip": "94105",
      "country": "US"
    }
  }'
```

**Expected:**
```json
{
  "orderId": "test-order-1",
  "status": "confirmed",
  "chargeId": "ch_mock_...",
  "trackingNumber": "1Z999AA1...",
  "message": "Order processed successfully"
}
```

**Verify:**
- ✅ Payment charged (check logs)
- ✅ Shipping label created
- ✅ Email sent
- ✅ Order status = "confirmed"

### Scenario 2: Idempotent Processing

**Test:** Send same request twice

```bash
# First request
curl -X POST http://localhost:8080/OrderOrchestrator/test-order-2/ProcessOrder \
  -H 'Content-Type: application/json' \
  -d '{...same payload...}'

# Second request (duplicate)
curl -X POST http://localhost:8080/OrderOrchestrator/test-order-2/ProcessOrder \
  -H 'Content-Type: application/json' \
  -d '{...same payload...}'
```

**Expected:**
- ✅ Same result from both requests
- ✅ Payment charged only once
- ✅ Only one email sent
- ✅ Logs show "Order already exists"

### Scenario 3: Payment Failure Handling

The mock Stripe client has a 5% failure rate. To test failure handling, you may need to send multiple requests or modify the mock failure rate.

**Expected on Failure:**
```json
{
  "orderId": "test-order-3",
  "status": "payment_failed",
  "message": "Payment failed: card declined"
}
```

**Verify:**
- ✅ Order status = "payment_failed"
- ✅ No shipping label created
- ✅ No email sent
- ✅ Safe to retry

### Scenario 4: Webhook Processing

**Test:** Process Stripe webhook

```bash
curl -X POST http://localhost:8080/WebhookProcessor/webhook-001/ProcessStripeWebhook \
  -H 'Content-Type: application/json' \
  -d '{
    "id": "webhook-001",
    "type": "charge.succeeded",
    "data": {"payment_id": "pay_123"},
    "created": 1234567890
  }'
```

**Expected:**
```json
{
  "webhookId": "webhook-001",
  "type": "charge.succeeded",
  "status": "processed",
  "message": "Charge succeeded"
}
```

**Test Idempotency:**
```bash
# Send same webhook again
curl -X POST http://localhost:8080/WebhookProcessor/webhook-001/ProcessStripeWebhook \
  [same payload]
```

**Expected:**
- ✅ Same result
- ✅ Logs show "Webhook already processed"

### Scenario 5: Concurrent Requests

**Test:** Send multiple requests simultaneously

```bash
# In terminal 1
curl -X POST http://localhost:8080/OrderOrchestrator/concurrent-1/ProcessOrder -d '{...}'

# In terminal 2 (immediately)
curl -X POST http://localhost:8080/OrderOrchestrator/concurrent-1/ProcessOrder -d '{...}'
```

**Expected:**
- ✅ Both return same result
- ✅ Only one payment charged
- ✅ No race conditions

### Scenario 6: Get Order Status

**Test:** Retrieve order

```bash
curl -X POST http://localhost:8080/OrderOrchestrator/test-order-1/GetOrder \
  -H 'Content-Type: application/json' \
  -d '{}'
```

**Expected:**
```json
{
  "orderId": "test-order-1",
  "items": [...],
  "status": "confirmed",
  "chargeId": "ch_mock_...",
  "trackingNumber": "1Z999AA1..."
}
```

## ✅ Validation Checklist

### External Integration
- [ ] All API calls logged in console
- [ ] Each API call wrapped in `restate.Run()`
- [ ] Mock mode works without real APIs
- [ ] Real mode would use actual API keys

### Idempotency
- [ ] Duplicate requests return same result
- [ ] No duplicate payments
- [ ] No duplicate emails
- [ ] State correctly persisted

### Failure Handling
- [ ] Payment failures handled gracefully
- [ ] Failed orders don't create labels/emails
- [ ] Safe to retry failed orders

### Webhooks
- [ ] Webhooks processed idempotently
- [ ] Duplicate webhooks handled
- [ ] Unknown webhook types logged

### Performance
- [ ] Order processing < 1s in mock mode
- [ ] No unnecessary retries
- [ ] Concurrent requests handled

## 🎓 Success Criteria

Your implementation is valid when:
- ✅ All test scenarios pass
- ✅ No duplicate external operations
- ✅ Idempotent at all levels
- ✅ Failures handled gracefully
- ✅ Webhooks processed safely

## 🚀 Next Steps

Module complete! 

👉 **Continue to [Exercises](./04-exercises.md)**

Practice building your own integrations!

---

**Questions?** Review [concepts](./01-concepts.md) or [hands-on tutorial](./02-hands-on.md).
