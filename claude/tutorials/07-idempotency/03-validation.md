# Validation: Testing Idempotency

> **Verify your payment service handles duplicates correctly**

## 🎯 Validation Goals

Test that your payment service:
- ✅ Processes payments exactly once
- ✅ Returns same result for duplicate requests
- ✅ Handles network retries safely
- ✅ Prevents duplicate refunds
- ✅ Maintains consistent state

## 🧪 Test Scenarios

### Scenario 1: Basic Payment Idempotency

**Goal:** Verify duplicate payment requests return same result

#### Step 1: Create Payment

```bash
curl -X POST http://localhost:8080/PaymentService/test-001/CreatePayment \
  -H 'Content-Type: application/json' \
  -d '{
    "amount": 5000,
    "currency": "USD",
    "description": "Test payment",
    "customerId": "test-customer"
  }'
```

**Expected Response:**
```json
{
  "paymentId": "test-001",
  "status": "completed",
  "chargeId": "ch_test-customer_1700000001",
  "message": "Payment processed successfully"
}
```

**Note the `chargeId`** - we'll verify it doesn't change!

#### Step 2: Send Duplicate Request

```bash
# Exact same request
curl -X POST http://localhost:8080/PaymentService/test-001/CreatePayment \
  -H 'Content-Type: application/json' \
  -d '{
    "amount": 5000,
    "currency": "USD",
    "description": "Test payment",
    "customerId": "test-customer"
  }'
```

**Expected Response:**
```json
{
  "paymentId": "test-001",
  "status": "completed",
  "chargeId": "ch_test-customer_1700000001",
  "message": "Payment already processed"
}
```

**✅ Validation Checklist:**
- [x] Same `paymentId` returned
- [x] Same `chargeId` (proves no duplicate gateway call)
- [x] Message indicates "already processed"
- [x] Response time much faster (no gateway call)

---

### Scenario 2: Concurrent Duplicate Requests

**Goal:** Test handling of simultaneous duplicate requests

#### Step 1: Send Multiple Concurrent Requests

Save this script as `test_concurrent.sh`:

```bash
#!/bin/bash

PAYMENT_ID="concurrent-test-001"

# Send 5 concurrent requests
for i in {1..5}; do
  curl -X POST http://localhost:8080/PaymentService/$PAYMENT_ID/CreatePayment \
    -H 'Content-Type: application/json' \
    -d '{
      "amount": 10000,
      "currency": "USD",
      "description": "Concurrent test",
      "customerId": "concurrent-customer"
    }' &
done

# Wait for all requests to complete
wait

echo "All requests completed"
```

Run it:
```bash
chmod +x test_concurrent.sh
./test_concurrent.sh
```

#### Step 2: Verify Single Charge

```bash
curl -X POST http://localhost:8080/PaymentService/concurrent-test-001/GetPayment \
  -H 'Content-Type: application/json' \
  -d '{}'
```

**✅ Validation Checklist:**
- [x] All 5 requests return same `chargeId`
- [x] Only one charge ID in payment state
- [x] Payment status is "completed"
- [x] No errors or inconsistencies

---

### Scenario 3: Retry After Failure

**Goal:** Verify safe retry after temporary failures

The mock gateway has a 10% failure rate. Let's test retry behavior:

#### Step 1: Create Payment (May Fail)

```bash
curl -X POST http://localhost:8080/PaymentService/retry-test-001/CreatePayment \
  -H 'Content-Type: application/json' \
  -d '{
    "amount": 15000,
    "currency": "USD",
    "description": "Retry test",
    "customerId": "retry-customer"
  }'
```

**Possible Responses:**

**Success:**
```json
{
  "paymentId": "retry-test-001",
  "status": "completed",
  "chargeId": "ch_retry-customer_1700000002",
  "message": "Payment processed successfully"
}
```

**Failure (10% chance):**
```json
{
  "paymentId": "retry-test-001",
  "status": "failed",
  "message": "insufficient funds"
}
```

#### Step 2: Check Payment State

```bash
curl -X POST http://localhost:8080/PaymentService/retry-test-001/GetPayment \
  -H 'Content-Type: application/json' \
  -d '{}'
```

#### Step 3: Retry If Failed

If status is "failed", create a new payment with different ID:

```bash
curl -X POST http://localhost:8080/PaymentService/retry-test-002/CreatePayment \
  -H 'Content-Type: application/json' \
  -d '{
    "amount": 15000,
    "currency": "USD",
    "description": "Retry test - second attempt",
    "customerId": "retry-customer"
  }'
```

**✅ Validation Checklist:**
- [x] Failed payments stay "failed" (idempotent failure)
- [x] Retrying failed payment ID returns "failed" status
- [x] New payment ID creates new payment
- [x] State accurately reflects payment status

---

### Scenario 4: Refund Idempotency

**Goal:** Ensure refunds are processed exactly once

#### Step 1: Create Payment

```bash
curl -X POST http://localhost:8080/PaymentService/refund-test-001/CreatePayment \
  -H 'Content-Type: application/json' \
  -d '{
    "amount": 20000,
    "currency": "USD",
    "description": "Refund test payment",
    "customerId": "refund-customer"
  }'
```

#### Step 2: Process Refund

```bash
curl -X POST http://localhost:8080/PaymentService/refund-test-001/RefundPayment \
  -H 'Content-Type: application/json' \
  -d '{
    "reason": "Customer request",
    "amount": 20000
  }'
```

**Expected Response:**
```json
{
  "refundId": "re_ch_refund-customer_..._1700000003",
  "status": "completed",
  "amount": 20000,
  "message": "Refund processed successfully"
}
```

**Note the `refundId`!**

#### Step 3: Try Refunding Again

```bash
curl -X POST http://localhost:8080/PaymentService/refund-test-001/RefundPayment \
  -H 'Content-Type: application/json' \
  -d '{
    "reason": "Customer request",
    "amount": 20000
  }'
```

**Expected Response:**
```json
{
  "refundId": "re_ch_refund-customer_..._1700000003",
  "status": "completed",
  "amount": 20000,
  "message": "Refund processed successfully"
}
```

**✅ Validation Checklist:**
- [x] Same `refundId` returned on duplicate request
- [x] Customer refunded exactly once
- [x] Payment status is "refunded"
- [x] No error on duplicate refund request

---

### Scenario 5: Invalid Refund Attempts

**Goal:** Verify refund validation works correctly

#### Test 5a: Refund Non-Existent Payment

```bash
curl -X POST http://localhost:8080/PaymentService/nonexistent-payment/RefundPayment \
  -H 'Content-Type: application/json' \
  -d '{
    "reason": "Test",
    "amount": 1000
  }'
```

**Expected:** Error (payment not found)

#### Test 5b: Refund Failed Payment

```bash
# First create a payment (keep retrying until it fails)
# Then try to refund it

curl -X POST http://localhost:8080/PaymentService/failed-payment/RefundPayment \
  -H 'Content-Type: application/json' \
  -d '{
    "reason": "Test",
    "amount": 1000
  }'
```

**Expected:** Error (cannot refund failed payment)

**✅ Validation Checklist:**
- [x] Cannot refund non-existent payment
- [x] Cannot refund failed payment
- [x] Appropriate error messages returned

---

### Scenario 6: Partial Refunds

**Goal:** Test partial refund amounts

#### Step 1: Create $100 Payment

```bash
curl -X POST http://localhost:8080/PaymentService/partial-refund-001/CreatePayment \
  -H 'Content-Type: application/json' \
  -d '{
    "amount": 10000,
    "currency": "USD",
    "description": "Partial refund test",
    "customerId": "partial-customer"
  }'
```

#### Step 2: Partial Refund ($30)

```bash
curl -X POST http://localhost:8080/PaymentService/partial-refund-001/RefundPayment \
  -H 'Content-Type: application/json' \
  -d '{
    "reason": "Partial refund",
    "amount": 3000
  }'
```

**Expected:** Refund of $30

#### Step 3: Try Full Refund

```bash
curl -X POST http://localhost:8080/PaymentService/partial-refund-001/RefundPayment \
  -H 'Content-Type: application/json' \
  -d '{
    "reason": "Full refund attempt",
    "amount": 10000
  }'
```

**Expected:** Same $30 refund (idempotent - refund already exists)

**✅ Validation Checklist:**
- [x] Partial refund amount respected
- [x] Duplicate refund request returns same result
- [x] Cannot process second refund (already refunded)

---

## 🔍 State Inspection

### View Restate Invocation Journal

Restate provides APIs to inspect the invocation journal:

```bash
# List invocations for payment
curl http://localhost:9070/invocations?service=PaymentService&key=test-001
```

### Check Service State

```bash
# Get payment state
curl -X POST http://localhost:8080/PaymentService/test-001/GetPayment \
  -H 'Content-Type: application/json' \
  -d '{}'
```

---

## 📊 Performance Testing

### Idempotent Request Performance

Compare performance of first vs duplicate requests:

```bash
# First request (processes payment)
time curl -X POST http://localhost:8080/PaymentService/perf-001/CreatePayment \
  -H 'Content-Type: application/json' \
  -d '{"amount": 1000, "currency": "USD", "customerId": "perf-test"}'

# Duplicate request (returns cached result)
time curl -X POST http://localhost:8080/PaymentService/perf-001/CreatePayment \
  -H 'Content-Type: application/json' \
  -d '{"amount": 1000, "currency": "USD", "customerId": "perf-test"}'
```

**Expected:**
- First request: ~100ms (gateway call)
- Duplicate request: <10ms (cached result, no gateway call)

---

## ✅ Validation Summary

### Checklist

Complete these validations:

- [x] **Basic Idempotency**
  - Duplicate payment returns same result
  - Same charge ID on retries
  
- [x] **Concurrent Requests**
  - Multiple simultaneous requests deduplicated
  - Only one charge created
  
- [x] **Retry After Failure**
  - Failed payments stay failed
  - Retries are safe
  
- [x] **Refund Idempotency**
  - Refund processed exactly once
  - Duplicate refund requests safe
  
- [x] **Validation**
  - Cannot refund invalid payments
  - Appropriate error handling
  
- [x] **Partial Refunds**
  - Partial amount respected
  - No duplicate partial refunds

### Success Criteria

Your implementation passes if:

1. ✅ No duplicate charges (check charge IDs)
2. ✅ Same result for same request
3. ✅ Concurrent requests handled correctly
4. ✅ Failed payments don't create charges
5. ✅ Refunds are idempotent
6. ✅ State is always consistent

---

## 🎓 What You Validated

### Idempotency Guarantees

- **Exactly-once processing** - Payment gateway called once
- **Deterministic results** - Same input = Same output
- **Safe retries** - Network failures don't create duplicates
- **State consistency** - Status accurately reflects reality
- **Concurrent safety** - Multiple requests deduplicated

### Restate Features Verified

- Virtual object state isolation
- Journaled side effects (`restate.Run`)
- State-based deduplication
- Request deduplication
- Durable execution

---

## 🚀 Next Steps

Excellent! You've validated your idempotent payment service!

👉 **Continue to [Exercises](./04-exercises.md)**

Practice building more idempotent services!

---

**Questions?** Review [hands-on tutorial](./02-hands-on.md) or [concepts](./01-concepts.md).
