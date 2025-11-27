# Validation: Testing Saga Compensation

> **Verify distributed transactions and compensation work correctly**

## 🎯 Objectives

Verify that:
- ✅ Saga completes when all steps succeed
- ✅ Compensation runs when steps fail
- ✅ Compensations are idempotent
- ✅ Saga state is durable across failures
- ✅ All services are properly cancelled on rollback

## 📋 Pre-Validation Checklist

- [ ] Restate server running (ports 8080/9080)
- [ ] Travel saga service running (port 9090)
- [ ] Service registered with Restate
- [ ] `curl` and `jq` available

## 🧪 Test Suite

### Test 1: Successful Saga Completion

**Purpose:** Verify all steps complete successfully

```bash
# Run saga
curl -X POST http://localhost:9080/TravelSaga/test-success-001/Run \
  -H 'Content-Type: application/json' \
  -d '{
    "bookingId": "test-success-001",
    "customerId": "customer-123",
    "flightInfo": {
      "from": "NYC",
      "to": "LAX",
      "departDate": "2024-06-01T10:00:00Z",
      "returnDate": "2024-06-07T18:00:00Z",
      "passengers": 2
    },
    "hotelInfo": {
      "location": "LAX",
      "checkIn": "2024-06-01T15:00:00Z",
      "checkOut": "2024-06-07T11:00:00Z",
      "guests": 2
    },
    "carInfo": {
      "location": "LAX",
      "pickupDate": "2024-06-01T10:00:00Z",
      "returnDate": "2024-06-07T18:00:00Z",
      "carType": "SUV"
    }
  }' | jq .
```

**Expected (90% probability):**
```json
{
  "status": "confirmed",
  "flightConfirmation": "FL-xxxxxxxx",
  "hotelConfirmation": "HT-xxxxxxxx",
  "carConfirmation": "CR-xxxxxxxx"
}
```

**Validation:**
- ✅ Status is "confirmed"
- ✅ All three confirmation codes present
- ✅ No failure reason

**Note:** Due to random failures, you may need to run this a few times.

---

### Test 2: Compensation on Failure

**Purpose:** Verify compensation runs when saga fails

```bash
# Run multiple times to trigger failures
for i in {1..20}; do
  echo "Attempt $i:"
  curl -s -X POST http://localhost:9080/TravelSaga/test-compensate-$i/Run \
    -H 'Content-Type: application/json' \
    -d '{
      "bookingId": "test-compensate-'$i'",
      "customerId": "customer-123",
      "flightInfo": {"from": "NYC", "to": "LAX", "departDate": "2024-06-01T10:00:00Z", "returnDate": "2024-06-07T18:00:00Z", "passengers": 2},
      "hotelInfo": {"location": "LAX", "checkIn": "2024-06-01T15:00:00Z", "checkOut": "2024-06-07T11:00:00Z", "guests": 2},
      "carInfo": {"location": "LAX", "pickupDate": "2024-06-01T10:00:00Z", "returnDate": "2024-06-07T18:00:00Z", "carType": "SUV"}
    }' | jq '{status, failureReason}'
  sleep 0.5
done
```

**Expected:** Mix of "confirmed" and "failed" results

**Failed Examples:**
```json
{"status": "failed", "failureReason": "flight: flight unavailable"}
{"status": "failed", "failureReason": "hotel: hotel unavailable"}
{"status": "failed", "failureReason": "car: car unavailable"}
```

**Validation:**
- ✅ Some attempts succeed
- ✅ Some attempts fail with reason
- ✅ Check logs for compensation messages

**Check service logs:**
```
Compensating: cancelling flight
Flight cancelled successfully
```

---

### Test 3: Journal Inspection

**Purpose:** Examine saga execution in detail

```bash
# Run saga
curl -X POST http://localhost:9080/TravelSaga/test-journal-001/Run \
  -H 'Content-Type: application/json' \
  -H 'idempotency-key: journal-test' \
  -d '{
    "bookingId": "test-journal-001",
    "customerId": "customer-123",
    "flightInfo": {"from": "NYC", "to": "LAX", "departDate": "2024-06-01T10:00:00Z", "returnDate": "2024-06-07T18:00:00Z", "passengers": 2},
    "hotelInfo": {"location": "LAX", "checkIn": "2024-06-01T15:00:00Z", "checkOut": "2024-06-07T11:00:00Z", "guests": 2},
    "carInfo": {"location": "LAX", "pickupDate": "2024-06-01T10:00:00Z", "returnDate": "2024-06-07T18:00:00Z", "carType": "SUV"}
  }'

sleep 2

# Get invocation ID
INV_ID=$(curl -s 'http://localhost:8080/invocations?target_service=TravelSaga&target_key=test-journal-001&target_handler=Run' | jq -r '.invocations[0].id')

echo "Invocation ID: $INV_ID"

# View journal
curl -s "http://localhost:8080/invocations/$INV_ID/journal" | \
  jq '.entries[] | {index, type, name}'
```

**Expected Journal Entries:**
```json
{"index": 0, "type": "Run", "name": "FlightService/Reserve"}
{"index": 1, "type": "Run", "name": "HotelService/Reserve"}
{"index": 2, "type": "Run", "name": "CarService/Reserve"}
{"index": 3, "type": "Output", "name": null}
```

**If Failed at Hotel:**
```json
{"index": 0, "type": "Run", "name": "FlightService/Reserve"}
{"index": 1, "type": "Run", "name": "HotelService/Reserve"}
{"index": 2, "type": "Run", "name": "FlightService/Cancel"}
{"index": 3, "type": "Output", "name": null}
```

**Validation:**
- ✅ Each reservation attempt journaled
- ✅ Compensations visible if failure occurred
- ✅ Output recorded

---

### Test 4: Service Restart Resilience

**Purpose:** Verify saga survives service restart

```bash
# Start saga
curl -X POST http://localhost:9080/TravelSaga/test-restart-001/Run \
  -H 'Content-Type: application/json' \
  -d '{
    "bookingId": "test-restart-001",
    "customerId": "customer-123",
    "flightInfo": {"from": "NYC", "to": "LAX", "departDate": "2024-06-01T10:00:00Z", "returnDate": "2024-06-07T18:00:00Z", "passengers": 2},
    "hotelInfo": {"location": "LAX", "checkIn": "2024-06-01T15:00:00Z", "checkOut": "2024-06-07T11:00:00Z", "guests": 2},
    "carInfo": {"location": "LAX", "pickupDate": "2024-06-01T10:00:00Z", "returnDate": "2024-06-07T18:00:00Z", "carType": "SUV"}
  }' &

# Wait briefly
sleep 0.5

# Restart service
echo "Kill and restart your travel-saga service now!"
echo "Press Enter when restarted..."
read

# Check result
curl -s 'http://localhost:8080/invocations?target_service=TravelSaga&target_key=test-restart-001&target_handler=Run' | \
  jq '.invocations[0].status'
```

**Expected:** "completed" (saga resumed and finished)

**Validation:**
- ✅ Saga doesn't start over
- ✅ Completed steps not re-executed
- ✅ Resumes from interruption point

---

### Test 5: Compensation Statistics

**Purpose:** Measure compensation effectiveness

```bash
# Run 50 attempts
echo "Running 50 saga attempts..."
for i in {1..50}; do
  curl -s -X POST http://localhost:9080/TravelSaga/batch-$i/Run \
    -H 'Content-Type: application/json' \
    -d '{
      "bookingId": "batch-'$i'",
      "customerId": "customer-123",
      "flightInfo": {"from": "NYC", "to": "LAX", "departDate": "2024-06-01T10:00:00Z", "returnDate": "2024-06-07T18:00:00Z", "passengers": 2},
      "hotelInfo": {"location": "LAX", "checkIn": "2024-06-01T15:00:00Z", "checkOut": "2024-06-07T11:00:00Z", "guests": 2},
      "carInfo": {"location": "LAX", "pickupDate": "2024-06-01T10:00:00Z", "returnDate": "2024-06-07T18:00:00Z", "carType": "SUV"}
    }' > /dev/null &
  
  if [ $((i % 10)) -eq 0 ]; then
    wait
  fi
done

wait
echo "All attempts completed!"

# Count results
echo ""
echo "Results:"
curl -s 'http://localhost:8080/invocations?target_service=TravelSaga&target_handler=Run' | \
  jq -r '.invocations[] | select(.target_key | startswith("batch-")) | .status' | \
  sort | uniq -c
```

**Expected Distribution:**
```
~73% completed  (0.9 × 0.9 × 0.9)
~27% failed     (compensation triggered)
```

**Validation:**
- ✅ Success rate around 73%
- ✅ No "running" states (all finish)
- ✅ Failures cleanly compensated

---

### Test 6: Idempotent Compensation

**Purpose:** Verify calling saga twice is safe

```bash
# Run with idempotency key
curl -X POST http://localhost:9080/TravelSaga/test-idem-001/Run \
  -H 'Content-Type: application/json' \
  -H 'idempotency-key: idem-test-123' \
  -d '{
    "bookingId": "test-idem-001",
    "customerId": "customer-123",
    "flightInfo": {"from": "NYC", "to": "LAX", "departDate": "2024-06-01T10:00:00Z", "returnDate": "2024-06-07T18:00:00Z", "passengers": 2},
    "hotelInfo": {"location": "LAX", "checkIn": "2024-06-01T15:00:00Z", "checkOut": "2024-06-07T11:00:00Z", "guests": 2},
    "carInfo": {"location": "LAX", "pickupDate": "2024-06-01T10:00:00Z", "returnDate": "2024-06-07T18:00:00Z", "carType": "SUV"}
  }' > /tmp/result1.json

# Call again with same key
curl -X POST http://localhost:9080/TravelSaga/test-idem-001/Run \
  -H 'Content-Type: application/json' \
  -H 'idempotency-key: idem-test-123' \
  -d '{
    "bookingId": "test-idem-001",
    "customerId": "customer-123",
    "flightInfo": {"from": "NYC", "to": "LAX", "departDate": "2024-06-01T10:00:00Z", "returnDate": "2024-06-07T18:00:00Z", "passengers": 2},
    "hotelInfo": {"location": "LAX", "checkIn": "2024-06-01T15:00:00Z", "checkOut": "2024-06-07T11:00:00Z", "guests": 2},
    "carInfo": {"location": "LAX", "pickupDate": "2024-06-01T10:00:00Z", "returnDate": "2024-06-07T18:00:00Z", "carType": "SUV"}
  }' > /tmp/result2.json

# Compare
diff /tmp/result1.json /tmp/result2.json && echo "✅ Results identical!"
```

**Validation:**
- ✅ Both calls return same result
- ✅ Saga not executed twice
- ✅ Idempotency preserved

---

## 📊 Test Results Summary

| Test | Purpose | Expected | Pass/Fail |
|------|---------|----------|-----------|
| Successful Completion | All steps succeed | Confirmed booking | |
| Compensation on Failure | Rollback works | Failed + reason | |
| Journal Inspection | Operations logged | Journal entries | |
| Service Restart | Durability | Saga completes | |
| Statistics | Compensation rate | ~27% failures | |
| Idempotency | Duplicate calls | Same result | |

## ✅ Validation Checklist

- [ ] ✅ Saga completes successfully (~73% of time)
- [ ] ✅ Failures trigger compensation
- [ ] ✅ Flight cancelled if hotel fails
- [ ] ✅ Flight+hotel cancelled if car fails
- [ ] ✅ All operations journaled
- [ ] ✅ Survives service restart
- [ ] ✅ Idempotent execution
- [ ] ✅ Clear error messages

## 🎓 What You Learned

1. **Automatic Compensation** - Failed steps trigger rollback
2. **Reverse Order** - Compensations run from most recent to first
3. **Durability** - Saga survives failures and restarts
4. **Idempotency** - Safe to retry saga calls
5. **Observability** - Journal shows exact execution

## 🐛 Troubleshooting

### Saga Always Succeeds

If you're not seeing failures:
1. Check the random failure rate (should be 10%)
2. Run more attempts (try 20-30)
3. Verify `restate.Rand(ctx)` is being used

### Compensation Not Running

Check:
1. Service logs for "Compensating" messages
2. Journal for Cancel operations
3. Error handling in saga code

### Service Crashes

Ensure:
1. Restate server is running
2. Service registered correctly
3. No port conflicts

## 🎯 Next Steps

Excellent! Your saga is working correctly with automatic compensation.

Practice with more complex scenarios:

👉 **Continue to [Exercises](./04-exercises.md)**

---

**Questions?** Review [concepts](./01-concepts.md) or [hands-on](./02-hands-on.md)!
