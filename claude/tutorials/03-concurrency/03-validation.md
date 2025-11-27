# Validation: Testing Concurrent Execution

> **Verify parallel execution, measure performance improvements, and test resilience**

## 🎯 Objectives

Verify that:
- ✅ Parallel execution is faster than sequential
- ✅ Futures are properly journaled
- ✅ Partial failures are handled gracefully
- ✅ Service calls don't duplicate on replay
- ✅ Results are deterministic with idempotency keys

## 📋 Pre-Validation Checklist

- [ ] Restate server running (ports 8080/9080)
- [ ] Order service running (port 9090)
- [ ] All services registered with Restate
- [ ] `curl` and `time` command available

## 🧪 Test Suite

### Test 1: Performance Comparison - Sequential vs Parallel

**Purpose:** Measure the speedup from parallelization

```bash
echo "=== Testing Sequential Version ==="
time curl -s -X POST http://localhost:9080/OrderServiceSlow/ProcessOrder \
  -H 'Content-Type: application/json' \
  -d '{
    "userId": "user123",
    "productId": "prod456",
    "quantity": 2,
    "amount": 99.99,
    "weight": 5.5,
    "destination": "New York"
  }' | jq '.processingTimeMs'

echo ""
echo "=== Testing Parallel Version ==="
time curl -s -X POST http://localhost:9080/OrderService/ProcessOrder \
  -H 'Content-Type: application/json' \
  -d '{
    "userId": "user123",
    "productId": "prod456",
    "quantity": 2,
    "amount": 99.99,
    "weight": 5.5,
    "destination": "New York"
  }' | jq '.processingTimeMs'
```

**Expected Results:**
- Sequential: ~450-500ms
- Parallel: ~150-200ms
- **Speedup: 2.5-3x**

**Validation:**
- ✅ Parallel version is significantly faster
- ✅ Both return `"status": "confirmed"`
- ✅ `processingTimeMs` reflects actual timing

---

### Test 2: Journaling - Futures Don't Re-execute

**Purpose:** Verify futures are journaled and don't re-execute on replay

```bash
# First call with idempotency key
curl -s -X POST http://localhost:9080/OrderService/ProcessOrder \
  -H 'Content-Type: application/json' \
  -H 'idempotency-key: test-replay-001' \
  -d '{
    "userId": "user123",
    "productId": "prod456",
    "quantity": 2,
    "amount": 99.99,
    "weight": 5.5,
    "destination": "Boston"
  }' | jq > response1.json

# Second call with same key
curl -s -X POST http://localhost:9080/OrderService/ProcessOrder \
  -H 'Content-Type: application/json' \
  -H 'idempotency-key: test-replay-001' \
  -d '{
    "userId": "user123",
    "productId": "prod456",
    "quantity": 2,
    "amount": 99.99,
    "weight": 5.5,
    "destination": "Boston"
  }' | jq > response2.json

# Compare
diff response1.json response2.json
```

**Expected Result:**
- No differences! Identical `orderId`, shipping cost, etc.

**Check Service Logs:**
First call:
```
INFO Processing order in parallel userId=user123
INFO Inventory available warehouse=Main
INFO Payment check complete authorized=true
INFO Fraud check complete riskScore=25.3
INFO Order processed in parallel orderId=ORD_abc123 durationMs=150
```

Second call:
```
(No logs - served from journal!)
```

**Validation:**
- ✅ Responses are identical
- ✅ Second call returns instantly (from journal)
- ✅ No service logs on second call

---

### Test 3: Partial Failure Handling

**Purpose:** Verify service handles when some futures fail

Since our mocks have a 5-10% failure rate, call multiple times:

```bash
for i in {1..10}; do
  echo "=== Call $i ==="
  curl -s -X POST http://localhost:9080/OrderService/ProcessOrder \
    -H 'Content-Type: application/json' \
    -d '{
      "userId": "user'$i'",
      "productId": "prod456",
      "quantity": 2,
      "amount": 99.99,
      "weight": 5.5,
      "destination": "Seattle"
    }' | jq '{status: .status, details: .details}'
  sleep 1
done
```

**Expected Results:**
- Most calls succeed with `"status": "confirmed"`
- Some may fail if all warehouses are down
- Service continues despite partial failures

**Check Logs for Partial Failures:**
```
WARN Future failed error="warehouse Main temporarily unavailable"
INFO Inventory available warehouse=Backup  ← Used backup!
```

**Validation:**
- ✅ Service succeeds even if one warehouse fails
- ✅ Falls back to backup warehouse
- ✅ Continues processing despite individual failures

---

### Test 4: Inventory Unavailable Scenario

**Purpose:** Test graceful handling when no inventory is available

Run multiple times until both warehouses fail:

```bash
# Keep trying until we hit the failure case
for i in {1..20}; do
  result=$(curl -s -X POST http://localhost:9080/OrderService/ProcessOrder \
    -H 'Content-Type: application/json' \
    -d '{
      "userId": "user'$i'",
      "productId": "prod789",
      "quantity": 2,
      "amount": 99.99,
      "weight": 5.5,
      "destination": "Portland"
    }')
  
  status=$(echo $result | jq -r '.status')
  if [ "$status" == "failed" ]; then
    echo "Found failure case:"
    echo $result | jq
    break
  fi
done
```

**Expected Response:**
```json
{
  "orderId": "",
  "status": "failed",
  "details": "Product not available in any warehouse",
  "processingTimeMs": 150
}
```

**Validation:**
- ✅ Returns failure status
- ✅ Clear error message
- ✅ Still fast (checked both warehouses in parallel)

---

### Test 5: Payment/Fraud Rejection

**Purpose:** Test validation failures

Eventually you'll hit payment or fraud rejection:

```bash
# Run multiple times
for i in {1..30}; do
  result=$(curl -s -X POST http://localhost:9080/OrderService/ProcessOrder \
    -H 'Content-Type: application/json' \
    -d '{
      "userId": "user'$i'",
      "productId": "prod456",
      "quantity": 2,
      "amount": 999.99,
      "weight": 5.5,
      "destination": "Miami"
    }')
  
  status=$(echo $result | jq -r '.status')
  details=$(echo $result | jq -r '.details')
  
  if [[ "$details" == *"fraud"* ]] || [[ "$details" == *"Payment"* ]]; then
    echo "Found rejection:"
    echo $result | jq
    break
  fi
done
```

**Possible Failures:**
```json
{"status": "failed", "details": "Payment authorization failed"}
```
or
```json
{"status": "failed", "details": "High fraud risk: 85.3"}
```

**Validation:**
- ✅ Handles payment failures
- ✅ Handles fraud detection
- ✅ Clear reason in details

---

### Test 6: Shipping Options Selection

**Purpose:** Verify service chooses best shipping option

```bash
curl -s -X POST http://localhost:9080/OrderService/ProcessOrder \
  -H 'Content-Type: application/json' \
  -d '{
    "userId": "user123",
    "productId": "prod456",
    "quantity": 2,
    "amount": 99.99,
    "weight": 10.0,
    "destination": "Denver"
  }' | jq '{status, details}'
```

**Expected Response:**
```json
{
  "status": "confirmed",
  "details": "Shipping: $17.50 via Economy (7 days)"
}
```

**Why Economy?**
- Standard: $25.00
- Economy: $17.50 ← Cheaper!
- Service automatically chose the best option

**Validation:**
- ✅ Calculates shipping from multiple carriers
- ✅ Chooses lowest cost option
- ✅ Includes shipping details in response

---

### Test 7: Concurrent Requests

**Purpose:** Verify service handles multiple concurrent orders

```bash
# Launch 5 concurrent orders
for i in {1..5}; do
  curl -s -X POST http://localhost:9080/OrderService/ProcessOrder \
    -H 'Content-Type: application/json' \
    -H "idempotency-key: concurrent-$i" \
    -d '{
      "userId": "user'$i'",
      "productId": "prod456",
      "quantity": 2,
      "amount": 99.99,
      "weight": 5.5,
      "destination": "Austin"
    }' > /tmp/order$i.json &
done

wait
echo "All orders completed"

# Check results
for i in {1..5}; do
  echo "Order $i:"
  cat /tmp/order$i.json | jq '{orderId, status}'
done
```

**Expected:**
- All 5 orders process successfully
- Each has unique `orderId`
- All complete in ~150ms range

**Validation:**
- ✅ Handles concurrent load
- ✅ Each order is independent
- ✅ No interference between requests

---

### Test 8: Inspect Journal Entries

**Purpose:** See the parallel futures in the journal

```bash
# Make a call
curl -s -X POST http://localhost:9080/OrderService/ProcessOrder \
  -H 'Content-Type: application/json' \
  -H 'idempotency-key: journal-inspect-123' \
  -d '{
    "userId": "user123",
    "productId": "prod456",
    "quantity": 2,
    "amount": 99.99,
    "weight": 5.5,
    "destination": "Phoenix"
  }' > /dev/null

# Get invocation ID
INV_ID=$(curl -s http://localhost:8080/invocations | \
  jq -r '.invocations[] | select(.target_service_name == "OrderService") | .id' | \
  head -1)

echo "Invocation ID: $INV_ID"

# View journal with entry count
curl -s "http://localhost:8080/invocations/$INV_ID/journal" | \
  jq '{totalEntries: .entries | length, entries: .entries[] | {index, type, name}}'
```

**Expected Output:**
```json
{
  "totalEntries": 10,
  "entries": [
    {"index": 0, "type": "Start"},
    {"index": 1, "type": "Invoke", "name": "CheckInventory"},    ← Future 1
    {"index": 2, "type": "Invoke", "name": "CheckInventory"},    ← Future 2
    {"index": 3, "type": "Invoke", "name": "AuthorizePayment"},  ← Future 3
    {"index": 4, "type": "Invoke", "name": "CheckFraud"},        ← Future 4
    {"index": 5, "type": "Invoke", "name": "CalculateShipping"}, ← Future 5
    {"index": 6, "type": "Invoke", "name": "CalculateShipping"}, ← Future 6
    {"index": 7, "type": "Output"}                               ← Result
  ]
}
```

**Key Observation:**
- All 6 futures created before any complete
- This proves they execute in parallel
- Journal shows the fan-out pattern

---

## 📊 Test Results Summary

| Test | Purpose | Expected | Pass/Fail |
|------|---------|----------|-----------|
| Performance | Parallel is faster | 2.5-3x speedup | |
| Journaling | Futures don't re-execute | Identical results | |
| Partial Failure | Handles some failures | Backup warehouse | |
| Inventory Check | All warehouses down | Failed status | |
| Payment/Fraud | Validation failures | Clear error | |
| Shipping | Chooses best option | Lowest cost | |
| Concurrency | Multiple requests | All succeed | |
| Journal Inspection | View futures | 6 parallel invokes | |

## 🔍 Advanced Validation

### Measure Actual Speedup

Create a script to measure average times:

```bash
#!/bin/bash

echo "Measuring Sequential..."
total_seq=0
for i in {1..5}; do
  time_ms=$(curl -s -X POST http://localhost:9080/OrderServiceSlow/ProcessOrder \
    -H 'Content-Type: application/json' \
    -d '{
      "userId": "test",
      "productId": "prod456",
      "quantity": 2,
      "amount": 99.99,
      "weight": 5.5,
      "destination": "Test"
    }' | jq '.processingTimeMs')
  total_seq=$((total_seq + time_ms))
done
avg_seq=$((total_seq / 5))

echo "Measuring Parallel..."
total_par=0
for i in {1..5}; do
  time_ms=$(curl -s -X POST http://localhost:9080/OrderService/ProcessOrder \
    -H 'Content-Type: application/json' \
    -d '{
      "userId": "test",
      "productId": "prod456",
      "quantity": 2,
      "amount": 99.99,
      "weight": 5.5,
      "destination": "Test"
    }' | jq '.processingTimeMs')
  total_par=$((total_par + time_ms))
done
avg_par=$((total_par / 5))

echo ""
echo "=== Results ==="
echo "Sequential avg: ${avg_seq}ms"
echo "Parallel avg: ${avg_par}ms"
speedup=$(echo "scale=2; $avg_seq / $avg_par" | bc)
echo "Speedup: ${speedup}x"
```

**Expected:**
- Sequential: 450-500ms
- Parallel: 150-200ms
- Speedup: 2.5-3.0x

## ✅ Validation Checklist

- [ ] ✅ Parallel version 2-3x faster than sequential
- [ ] ✅ Idempotent calls return identical results
- [ ] ✅ Futures journaled (no re-execution)
- [ ] ✅ Handles partial failures gracefully
- [ ] ✅ Falls back to backup warehouse
- [ ] ✅ Chooses best shipping option
- [ ] ✅ Handles concurrent requests
- [ ] ✅ Journal shows parallel invocations

## 🎓 What You Learned

1. **Parallelism Works** - Dramatic performance improvement
2. **Futures are Journaled** - No duplicate execution on replay
3. **Resilience** - Graceful handling of partial failures
4. **Redundancy** - Backup options improve reliability
5. **Smart Aggregation** - Choosing best from multiple options

## 🐛 Troubleshooting

### Parallel Not Faster

**Check:**
1. Services are actually registered
2. Not hitting rate limits
3. Restate server has capacity

### Different Results with Same Key

Ensure:
1. Idempotency key is exactly the same
2. Request body is identical
3. Service hasn't been redeployed

### All Orders Failing

Increase mock success rates in `supporting_services.go`:
```go
// Change from 0.05 to 0.01 (1% failure instead of 5%)
if rand.Float64() < 0.01 {
```

## 🎯 Next Steps

Excellent! Your concurrent service is validated and optimized.

Now let's practice with exercises:

👉 **Continue to [Exercises](./04-exercises.md)**

---

**Questions?** Review [concepts](./01-concepts.md) or check the [complete code](./code/)!
