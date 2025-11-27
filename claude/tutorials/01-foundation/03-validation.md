# Validation: Testing Your Durable Service

> **Comprehensive testing to verify your service works correctly**

## 🎯 Objectives

Verify that:
- ✅ Service handles successful requests
- ✅ Retry behavior works on transient errors
- ✅ Terminal errors don't retry
- ✅ Journaling prevents duplicate execution
- ✅ Deterministic operations work correctly

## 📋 Pre-Validation Checklist

Before running tests:

- [ ] Restate server is running on ports 8080/9080
- [ ] Your greeting service is running on port 9090
- [ ] Service is registered with Restate
- [ ] You have `curl` and `jq` installed

## 🧪 Test Suite

### Test 1: Basic Success Case

**Purpose:** Verify the service works for valid input

```bash
curl -X POST http://localhost:9080/Greeting Service/Greet \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "Alice",
    "shouldFail": false
  }'
```

**Expected Result:**
```json
{
  "message": "Hello, Alice! You're awesome!",
  "timestamp": "<some-uuid>"
}
```

**Validation:**
- ✅ Returns 200 OK
- ✅ Message contains "Alice"
- ✅ Timestamp is a valid UUID
- ✅ Service logs show "Processing greeting" once

---

### Test 2: Deterministic UUID

**Purpose:** Verify that UUIDs are deterministic (same invocation = same UUID)

```bash
# Make the same call twice with an idempotency key
curl -X POST http://localhost:9080/GreetingService/Greet \
  -H 'Content-Type: application/json' \
  -H 'idempotency-key: test-123' \
  -d '{"name": "Bob", "shouldFail": false}'

# Same call again
curl -X POST http://localhost:9080/GreetingService/Greet \
  -H 'Content-Type: application/json' \
  -H 'idempotency-key: test-123' \
  -d '{"name": "Bob", "shouldFail": false}'
```

**Expected Result:**
- Both responses should be **identical** (same timestamp UUID)
- Service logs show "Processing greeting" only **once** (second call is from journal)

**Why This Matters:**
This proves that:
1. Restate deduplicates calls with the same idempotency key
2. `restate.UUID(ctx)` is deterministic
3. On the second call, Restate replays from the journal

---

### Test 3: Terminal Error (No Retry)

**Purpose:** Verify terminal errors don't trigger retries

```bash
# Call with empty name
curl -X POST http://localhost:9080/GreetingService/Greet \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "",
    "shouldFail": false
  }'
```

**Expected Result:**
```json
{
  "error": "name cannot be empty",
  "code": 400
}
```

**Validation:**
- ✅ Returns immediately (no delay)
- ✅ Error code is 400
- ✅ Service logs show only ONE attempt (no retries)
- ✅ Response is instant (not retrying)

**Check Logs:**
```bash
# In your service terminal, you should see:
# INFO Processing greeting name= shouldFail=false
# (Only once, no retries)
```

---

### Test 4: Retry Behavior (Transient Error)

**Purpose:** Observe automatic retry on transient failures

**⚠️ Warning:** This test will cause infinite retries. Be ready to cancel!

```bash
# Start the failing request (in background)
curl -X POST http://localhost:9080/GreetingService/Greet \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "Charlie",
    "shouldFail": true
  }' &

# Note the invocation ID from the logs or query it:
curl http://localhost:8080/invocations | jq '.invocations[] | select(.target_service_name == "GreetingService")'
```

**Expected Behavior:**
1. Request fails with "simulated transient failure"
2. Restate waits (exponential backoff)
3. Retries automatically
4. Repeats indefinitely until success or manual cancellation

**Check Service Logs:**
You should see repeated entries:
```
INFO Processing greeting name=Charlie shouldFail=true
ERROR Handler returned error: simulated transient failure
(wait 1 second)
INFO Processing greeting name=Charlie shouldFail=true
ERROR Handler returned error: simulated transient failure
(wait 2 seconds)
INFO Processing greeting name=Charlie shouldFail=true
...
```

**Cancel the Invocation:**
```bash
# Get invocation ID
INV_ID=$(curl -s http://localhost:8080/invocations | \
  jq -r '.invocations[] | select(.target_service_name == "GreetingService") | .id' | \
  head -1)

# Kill it
curl -X DELETE "http://localhost:8080/invocations/$INV_ID"
```

**Validation:**
- ✅ Service logs show multiple retry attempts
- ✅ Delay between retries increases (exponential backoff)
- ✅ Invocation stops after DELETE

---

### Test 5: Context Logging (No Duplication)

**Purpose:** Verify that `ctx.Log()` doesn't duplicate on replay

**Setup:**
Modify your service temporarily to simulate a mid-execution failure:

```go
func (GreetingService) Greet(ctx restate.Context, req GreetRequest) (GreetResponse, error) {
    ctx.Log().Info("Step 1: Received request", "name", req.Name)
    
    // Simulate work
    requestID := restate.UUID(ctx).String()
    ctx.Log().Info("Step 2: Generated UUID", "uuid", requestID)
    
    // Simulate a crash after logging but before completion
    if req.Name == "CrashTest" {
        return GreetResponse{}, fmt.Errorf("simulated crash")
    }
    
    ctx.Log().Info("Step 3: Completing successfully")
    
    return GreetResponse{
        Message:   fmt.Sprintf("Hello, %s!", req.Name),
        Timestamp: requestID,
    }, nil
}
```

Rebuild and restart your service.

**Test:**
```bash
# This will fail and retry
curl -X POST http://localhost:9080/GreetingService/Greet \
  -H 'Content-Type: application/json' \
  -d '{"name": "CrashTest", "shouldFail": false}'
```

**Check Logs:**
You should see:
```
INFO Step 1: Received request name=CrashTest
INFO Step 2: Generated UUID uuid=...
ERROR Handler returned error: simulated crash
(retry)
INFO Step 1: Received request name=CrashTest  ← NOT LOGGED (replay!)
INFO Step 2: Generated UUID uuid=...          ← NOT LOGGED (replay!)
ERROR Handler returned error: simulated crash
```

**Key Observation:** 
- Log entries from completed journal steps don't print again
- This is because Restate replays from the journal, skipping already-logged steps

Cancel the invocation after observing.

---

### Test 6: Invocation Inspection

**Purpose:** Learn to inspect running invocations

```bash
# List all invocations
curl http://localhost:8080/invocations | jq

# Get specific invocation details
curl http://localhost:8080/invocations/<invocation-id> | jq

# View the journal entries
curl http://localhost:8080/invocations/<invocation-id>/journal | jq
```

**Explore:**
- Invocation status (pending, running, completed)
- Target service and handler
- Number of journal entries
- Retry count

---

## 📊 Test Results Summary

Create a table to track your test results:

| Test | Expected | Actual | Pass/Fail |
|------|----------|--------|-----------|
| Basic Success | Returns greeting | | |
| Deterministic UUID | Same UUID on same invocation | | |
| Terminal Error | No retry, instant fail | | |
| Retry Behavior | Auto-retry with backoff | | |
| Logging | No log duplication on replay | | |
| Invocation Inspection | Can view journal | | |

## 🔍 Advanced Validation

### Validate Journaling

**Test:** Restart your service mid-invocation

1. Start a long-running call (with a sleep if needed)
2. Restart your service (Ctrl+C and restart)
3. Observe that the invocation completes from where it left off

### Validate Idempotency

**Test:** Multiple identical calls

```bash
# Send the same request multiple times with the same idempotency key
for i in {1..5}; do
  curl -X POST http://localhost:9080/GreetingService/Greet \
    -H 'Content-Type: application/json' \
    -H 'idempotency-key: duplicate-test' \
    -d '{"name": "Test", "shouldFail": false}'
  echo ""
done
```

**Expected:** All 5 responses are identical (served from journal after first execution)

## ✅ Validation Checklist

Mark each as complete:

- [ ] ✅ Basic success case works
- [ ] ✅ UUIDs are deterministic with idempotency key
- [ ] ✅ Terminal errors return immediately
- [ ] ✅ Transient errors trigger retry with backoff
- [ ] ✅ Context logging doesn't duplicate on replay
- [ ] ✅ Can inspect invocations and journals
- [ ] ✅ Idempotency keys deduplicate requests

## 🎓 What You Learned

By completing this validation, you've verified:

1. **Durable Execution** - Your code survives failures
2. **Journaling** - Restate records every step
3. **Smart Retry** - Automatic retry on transient errors
4. **Proper Error Handling** - Terminal vs retriable errors
5. **Determinism** - Same inputs = same outputs
6. **Deduplication** - Idempotency keys prevent duplicates

## 🐛 Troubleshooting

### Service Not Responding

```bash
# Check if service is running
lsof -i :9090

# Check Restate server logs
# Look for connection errors
```

### Invocations Stuck

```bash
# List all invocations
curl http://localhost:8080/invocations | jq

# Kill stuck invocations
curl -X DELETE http://localhost:8080/invocations/<id>
```

### Unexpected Behavior

```bash
# Check service logs carefully
# Look for:
# - Multiple log entries (indicates retry)
# - Error messages
# - Journal entry counts
```

## 🎯 Next Steps

Great! Your service is validated and working correctly.

Now let's practice with some exercises:

👉 **Continue to [Exercises](./04-exercises.md)**

---

**Need help?** Compare with the [complete code](./code/) or check the [troubleshooting guide](../appendix/troubleshooting.md)!
