# Validation: Testing Side Effects and Journaling

> **Verify that your weather service properly handles side effects and leverages journaling**

## 🎯 Objectives

Verify that:
- ✅ Side effects are properly journaled
- ✅ API calls don't duplicate on replay
- ✅ Partial failures are handled gracefully
- ✅ Idempotent calls return identical results
- ✅ Service handles complete API failures

## 📋 Pre-Validation Checklist

- [ ] Restate server running (ports 8080/9080)
- [ ] Weather service running (port 9090)
- [ ] Service registered with Restate
- [ ] `curl` and `jq` available

## 🧪 Test Suite

### Test 1: Basic Success - All APIs Work

**Purpose:** Verify the service aggregates data from all three APIs

```bash
curl -X POST http://localhost:9080/WeatherService/GetWeather \
  -H 'Content-Type: application/json' \
  -d '{"city": "London"}' | jq
```

**Expected Result:**
```json
{
  "city": "London",
  "sources": [
    {"source": "OpenWeather", "temperature": 24.5, ...},
    {"source": "WeatherBit", "temperature": 23.8, ...},
    {"source": "Weatherstack", "temperature": 24.2, ...}
  ],
  "averageTemp": 24.17,
  "successfulAPIs": 3
}
```

**Validation:**
- ✅ Returns 200 OK
- ✅ Has 3 sources (or 2-3, some may randomly fail)
- ✅ Average temperature is calculated correctly
- ✅ Service logs show three "Fetching from API" messages

---

### Test 2: Journaling - Identical Results on Replay

**Purpose:** Verify that `restate.Run` journals API results

```bash
# First call with idempotency key
curl -X POST http://localhost:9080/WeatherService/GetWeather \
  -H 'Content-Type: application/json' \
  -H 'idempotency-key: replay-test-001' \
  -d '{"city": "Paris"}' | jq > response1.json

# Wait a moment
sleep 2

# Second call with same key
curl -X POST http://localhost:9080/WeatherService/GetWeather \
  -H 'Content-Type: application/json' \
  -H 'idempotency-key: replay-test-001' \
  -d '{"city": "Paris"}' | jq > response2.json

# Compare responses
diff response1.json response2.json
```

**Expected Result:**
- No differences! Both responses are **identical**
- Temperatures are exactly the same
- Even random mock data is identical

**Check Service Logs:**
First call:
```
INFO Starting weather aggregation city=Paris
INFO Fetching from API 1
INFO API 1 succeeded temp=24.5
INFO Fetching from API 2
INFO API 2 succeeded temp=23.8
INFO Fetching from API 3
INFO API 3 succeeded temp=24.2
```

Second call:
```
(No logs! Served from journal)
```

**Why This Works:**
1. First call executes API calls via `restate.Run`
2. Results are journaled
3. Second call (same idempotency key) replays from journal
4. No API calls executed, instant response

---

### Test 3: Partial Failure Handling

**Purpose:** Verify service handles when some APIs fail

The mock APIs have a 10% failure rate. Let's test multiple times:

```bash
# Call multiple times to trigger failures
for i in {1..10}; do
  echo "=== Call $i ==="
  curl -s -X POST http://localhost:9080/WeatherService/GetWeather \
    -H 'Content-Type: application/json' \
    -d '{"city": "Berlin"}' | jq '.successfulAPIs'
  sleep 1
done
```

**Expected Results:**
- Most calls return `3` (all APIs succeeded)
- Some calls return `2` (one API failed)
- Rarely returns `1` (two APIs failed)
- Very rarely retries due to all failures

**Check Logs for Failed API:**
```
INFO Fetching from API 1
WARN API 1 failed error="API1 temporarily unavailable"
INFO Fetching from API 2
INFO API 2 succeeded temp=23.8
INFO Fetching from API 3
INFO API 3 succeeded temp=24.2
INFO Weather aggregation complete successfulAPIs=2
```

**Validation:**
- ✅ Service continues even if one API fails
- ✅ Returns result with successful APIs
- ✅ Logs show which API failed

---

### Test 4: Terminal Error - Invalid Input

**Purpose:** Verify terminal errors for bad input

```bash
# Empty city name
curl -X POST http://localhost:9080/WeatherService/GetWeather \
  -H 'Content-Type: application/json' \
  -d '{"city": ""}'
```

**Expected Result:**
```json
{
  "error": "city cannot be empty",
  "code": 400
}
```

**Validation:**
- ✅ Returns immediately (no retry)
- ✅ Status code 400
- ✅ Error message is clear
- ✅ No API calls made (check logs)

---

### Test 5: No Duplicate API Calls on Retry

**Purpose:** Verify API calls aren't duplicated when handler retries

This is harder to test with our current setup, but we can simulate it:

**Modify service temporarily** - Add a crash after first API:

```go
// In service.go, after first API call
data1, err := restate.Run(ctx, func(rc restate.RunContext) (WeatherData, error) {
    return MockWeatherAPI1(req.City)
})
if err == nil {
    sources = append(sources, data1)
    
    // TEMPORARY: Simulate a crash after first API
    if req.City == "CrashTest" {
        return AggregatedWeather{}, fmt.Errorf("simulated crash")
    }
}
```

Rebuild and restart service.

**Test:**
```bash
curl -X POST http://localhost:9080/WeatherService/GetWeather \
  -H 'Content-Type: application/json' \
  -d '{"city": "CrashTest"}'
```

**Check Logs:**
First attempt:
```
INFO Fetching from API 1
INFO API 1 succeeded temp=24.5
(crash)
```

Retry attempts:
```
INFO Starting weather aggregation city=CrashTest
(skips API 1 - replays from journal!)
INFO Fetching from API 2
INFO API 2 succeeded temp=23.8
(crash again)
INFO Starting weather aggregation city=CrashTest
(skips API 1 and 2 - replays from journal!)
INFO Fetching from API 3
...
```

**Key Observation:**
- API 1 is called **only once**
- On retries, result is replayed from journal
- This is the power of `restate.Run`!

**Clean up:** Remove the crash simulation code and rebuild.

---

### Test 6: Inspect Journal Entries

**Purpose:** See the actual journal entries

```bash
# Make a call
curl -X POST http://localhost:9080/WeatherService/GetWeather \
  -H 'Content-Type: application/json' \
  -H 'idempotency-key: journal-inspect-123' \
  -d '{"city": "Tokyo"}'

# Get invocation ID
INV_ID=$(curl -s http://localhost:8080/invocations | \
  jq -r '.invocations[] | select(.target_service_name == "WeatherService") | .id' | \
  head -1)

echo "Invocation ID: $INV_ID"

# View journal
curl -s "http://localhost:8080/invocations/$INV_ID/journal" | jq
```

**Expected Output:**
```json
{
  "entries": [
    {
      "index": 0,
      "type": "Start"
    },
    {
      "index": 1,
      "type": "Run",
      "name": "Run",
      "result": {...}  // API 1 result
    },
    {
      "index": 2,
      "type": "Run",
      "name": "Run",
      "result": {...}  // API 2 result
    },
    {
      "index": 3,
      "type": "Run",
      "name": "Run",
      "result": {...}  // API 3 result
    },
    {
      "index": 4,
      "type": "Output",
      "result": {...}  // Final aggregated result
    }
  ]
}
```

**What This Shows:**
- Each `restate.Run` creates a journal entry
- Results are stored in the journal
- On replay, Restate uses these stored results

---

## 📊 Test Results Summary

| Test | Purpose | Expected | Pass/Fail |
|------|---------|----------|-----------|
| Basic Success | All APIs work | 3 sources, avg temp | |
| Journaling | Identical results on replay | Exact same response | |
| Partial Failure | Handle some API failures | 1-2 sources, success | |
| Terminal Error | Invalid input | 400 error, no retry | |
| No Duplicate Calls | API called once per journal entry | Check logs | |
| Journal Inspection | View journal entries | See Run entries | |

## 🔍 Advanced Validation

### Performance Test

Measure the difference between first call and replay:

```bash
# First call (executes APIs)
time curl -X POST http://localhost:9080/WeatherService/GetWeather \
  -H 'Content-Type: application/json' \
  -H 'idempotency-key: perf-test-001' \
  -d '{"city": "NYC"}'

# Second call (from journal)
time curl -X POST http://localhost:9080/WeatherService/GetWeather \
  -H 'Content-Type: application/json' \
  -H 'idempotency-key: perf-test-001' \
  -d '{"city": "NYC"}'
```

**Expected:**
- First call: ~300-400ms (3 APIs × 100-150ms each)
- Second call: ~10-50ms (instant from journal!)

### Concurrent Calls

Test that concurrent calls with different keys work independently:

```bash
# Launch 5 concurrent requests
for i in {1..5}; do
  curl -X POST http://localhost:9080/WeatherService/GetWeather \
    -H 'Content-Type: application/json' \
    -H "idempotency-key: concurrent-$i" \
    -d "{\"city\": \"City$i\"}" &
done

wait
echo "All requests completed"
```

**Validation:**
- All return successfully
- Each has unique results (different keys)
- Service logs show interleaved execution

## ✅ Validation Checklist

- [ ] ✅ Basic aggregation works
- [ ] ✅ Idempotent calls return identical results
- [ ] ✅ Partial failures handled gracefully
- [ ] ✅ Terminal errors don't retry
- [ ] ✅ API calls journaled (not duplicated on retry)
- [ ] ✅ Journal entries visible via admin API
- [ ] ✅ Replay is significantly faster than first execution

## 🎓 What You Learned

1. **Journaling Works** - `restate.Run` stores results
2. **No Duplicates** - API calls execute exactly once per journal entry
3. **Fast Replay** - Replaying from journal is nearly instant
4. **Resilience** - Partial failures don't break the service
5. **Idempotency** - Same key = same result, always

## 🐛 Troubleshooting

### All Three APIs Always Succeed

Mock APIs have only 10% failure rate. This is expected - most calls succeed!

To test failures more reliably, increase failure rate in `weather_apis.go`:
```go
if rand.Float64() < 0.5 {  // 50% failure rate
    return WeatherData{}, fmt.Errorf("API temporarily unavailable")
}
```

### Different Results with Same Idempotency Key

Check:
1. Idempotency key is actually the same (case-sensitive!)
2. You're using `restate.Run` (not calling APIs directly)
3. Service code hasn't changed between calls

### Can't Find Invocation ID

```bash
# List all invocations
curl http://localhost:8080/invocations | jq

# Filter by service
curl http://localhost:8080/invocations | \
  jq '.invocations[] | select(.target_service_name == "WeatherService")'
```

## 🎯 Next Steps

Excellent! Your service properly handles side effects.

Now let's practice with exercises:

👉 **Continue to [Exercises](./04-exercises.md)**

---

**Questions?** Review [concepts](./01-concepts.md) or check the [complete code](./code/)!
