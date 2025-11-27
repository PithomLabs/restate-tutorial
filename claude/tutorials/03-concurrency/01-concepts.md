# Concepts: Concurrent Execution and Fan-Out/Fan-In

> **Understanding parallel execution patterns for high-performance distributed systems**

## 🎯 What is Concurrent Execution?

### Definition

**Concurrent execution** means running multiple operations **simultaneously** instead of sequentially, reducing overall latency.

### The Latency Problem

Imagine processing an order that requires:
- Inventory check: 100ms
- Payment validation: 150ms
- Fraud detection: 200ms
- Shipping calculation: 120ms

**Sequential approach:**
```
[Inventory] → [Payment] → [Fraud] → [Shipping]
   100ms       150ms       200ms      120ms
Total: 570ms ❌ Slow!
```

**Parallel approach:**
```
┌─ [Inventory: 100ms] ─┐
├─ [Payment: 150ms] ───┤
├─ [Fraud: 200ms] ─────┤── Wait for all
└─ [Shipping: 120ms] ──┘
Total: 200ms ✅ 2.85x faster!
```

## 🌟 Fan-Out/Fan-In Pattern

### Pattern Overview

```
        Single Request (Fan-Out)
               ↓
    ┌──────────┼──────────┐
    ↓          ↓          ↓
  Task 1    Task 2    Task 3    ← Execute in parallel
    ↓          ↓          ↓
    └──────────┼──────────┘
               ↓
        Aggregate (Fan-In)
               ↓
        Single Response
```

### Real-World Examples

1. **E-Commerce Order**
   - Fan-out: Check inventory, validate payment, calculate shipping
   - Fan-in: Aggregate results, create order

2. **Data Aggregation**
   - Fan-out: Query multiple databases/APIs
   - Fan-in: Merge and deduplicate results

3. **Content Personalization**
   - Fan-out: Fetch user profile, recommendations, ads
   - Fan-in: Render personalized page

## 🔮 Futures in Restate

### What is a Future?

A **future** is a placeholder for a result that will be available... in the future!

```go
// Start async operation - returns immediately
future := restate.Service[Result](ctx, "Service", "Handler").
    RequestFuture(input)

// Do other work...
doSomethingElse()

// Wait for result when you need it
result, err := future.Response()
```

### Sequential vs Async

**Sequential (blocking):**
```go
// Each call waits for the previous one
result1, _ := restate.Service[T](ctx, "Svc1", "Handler").Request(input1) // Wait
result2, _ := restate.Service[T](ctx, "Svc2", "Handler").Request(input2) // Wait
result3, _ := restate.Service[T](ctx, "Svc3", "Handler").Request(input3) // Wait
// Total: sum of all latencies
```

**Parallel (async):**
```go
// Start all calls immediately
fut1 := restate.Service[T](ctx, "Svc1", "Handler").RequestFuture(input1)
fut2 := restate.Service[T](ctx, "Svc2", "Handler").RequestFuture(input2)
fut3 := restate.Service[T](ctx, "Svc3", "Handler").RequestFuture(input3)

// Wait for all
for fut, err := range restate.Wait(ctx, fut1, fut2, fut3) {
    // Process as they complete
}
// Total: max latency among all calls
```

## 🔧 Restate Concurrency APIs

### 1. RequestFuture - Async Service Calls

```go
// Start async call to another service
future := restate.Service[OutputType](ctx, "ServiceName", "HandlerName").
    RequestFuture(input)

// Future is journaled immediately
// Do other work...

// Get result when needed
result, err := future.Response()
if err != nil {
    return err
}
```

**Key Points:**
- ✅ Returns immediately (non-blocking)
- ✅ Journaled by Restate
- ✅ Result replays from journal on retry
- ✅ Can create many futures before waiting

### 2. RunAsync - Async Side Effects

```go
// Start async side effect
future := restate.RunAsync(ctx, func(rc restate.RunContext) (Result, error) {
    return callExternalAPI(), nil
})

// Do other work...

// Get result
result, err := future.Result()
```

**Use For:**
- Parallel external API calls
- Concurrent database queries
- Independent side effects

### 3. restate.Wait - Collect Results

```go
// Create multiple futures
fut1 := restate.Service[T](ctx, "Svc1", "H1").RequestFuture(input1)
fut2 := restate.Service[T](ctx, "Svc2", "H2").RequestFuture(input2)
fut3 := restate.Service[T](ctx, "Svc3", "H3").RequestFuture(input3)

// Wait for ALL to complete
for fut, err := range restate.Wait(ctx, fut1, fut2, fut3) {
    if err != nil {
        // Handle error
        continue
    }
    
    // Process each result
    switch fut {
    case fut1:
        result1, _ := fut1.Response()
        // Use result1
    case fut2:
        result2, _ := fut2.Response()
        // Use result2
    case fut3:
        result3, _ := fut3.Response()
        // Use result3
    }
}
```

**Characteristics:**
- Waits for **all** futures to complete
- Iterates as results become available
- Handles errors per future
- Order of iteration is non-deterministic

### 4. restate.WaitFirst - Race Pattern

```go
// Start multiple alternatives
fut1 := restate.Service[T](ctx, "FastAPI", "Get").RequestFuture(input)
fut2 := restate.Service[T](ctx, "SlowAPI", "Get").RequestFuture(input)

// Wait for the FIRST to complete
winner, err := restate.WaitFirst(ctx, fut1, fut2)
if err != nil {
    return err
}

// Use the fastest result
switch winner {
case fut1:
    result, _ := fut1.Response()
case fut2:
    result, _ := fut2.Response()
}
```

**Use Cases:**
- Redundant requests for reliability
- Fastest response wins
- Timeout with fallback

## 📊 Comparing Patterns

### When to Use Each Pattern

| Pattern | Use Case | Example |
|---------|----------|---------|
| **Sequential** | Dependencies between calls | Get user → Get user's orders → Get order details |
| **Parallel (Wait)** | Independent calls | Check inventory + payment + shipping |
| **Race (WaitFirst)** | Redundancy/fallback | Try primary API, if slow try backup |
| **RunAsync** | Parallel side effects | Call 3 external APIs simultaneously |

### Performance Comparison

```go
// Scenario: 3 calls, each takes 100ms

// Sequential: 300ms
result1, _ := call1() // 100ms
result2, _ := call2() // 100ms
result3, _ := call3() // 100ms

// Parallel: ~100ms
fut1 := call1Async()
fut2 := call2Async()
fut3 := call3Async()
for fut, _ := range restate.Wait(ctx, fut1, fut2, fut3) {
    // All complete in ~100ms
}
```

**Speedup:** 3x faster! 🚀

## 🎓 Understanding Journaling with Futures

### How Futures are Journaled

```
First Execution:
┌──────────────────────────────────┐
│ Journal Entry 1: Start fut1      │
│ Journal Entry 2: Start fut2      │
│ Journal Entry 3: Start fut3      │
│ ... futures execute in parallel  │
│ Journal Entry 4: fut2 completed  │ ← Results as they arrive
│ Journal Entry 5: fut1 completed  │
│ Journal Entry 6: fut3 completed  │
└──────────────────────────────────┘

On Replay:
┌──────────────────────────────────┐
│ Entry 1-3: Futures created       │
│ Entry 4-6: Results from journal  │ ← No re-execution!
│ Total time: Near instant         │
└──────────────────────────────────┘
```

**Key Insight:** Futures and their results are journaled, so on replay:
- No service calls are re-executed
- Results come from journal
- Order of completion is preserved

## ⚠️ Common Patterns and Anti-Patterns

### ✅ Correct: Using restate.Wait

```go
// Create futures
futures := []restate.ResponseFuture[Result]{}
for i := 0; i < 5; i++ {
    fut := restate.Service[Result](ctx, "Svc", "Handler").
        RequestFuture(input)
    futures = append(futures, fut)
}

// Wait for all with restate.Wait
for fut, err := range restate.Wait(ctx, futures...) {
    if err != nil {
        continue
    }
    result, _ := fut.(restate.ResponseFuture[Result]).Response()
    // Process result
}
```

### ❌ Anti-Pattern: Futures in Goroutines

```go
// ❌ WRONG - Don't use goroutines with futures!
fut := restate.Service[T](ctx, "Svc", "Handler").RequestFuture(input)

go func() {
    result, _ := fut.Response() // Anti-pattern!
    // This breaks determinism and journaling
}()
```

**Why Wrong?**
- Goroutines aren't journaled by Restate
- Non-deterministic execution order
- Breaks replay guarantee

**Use Instead:** `restate.Wait` which is journaled and deterministic

### ✅ Correct: Parallel Side Effects

```go
// Create async side effects
fut1 := restate.RunAsync(ctx, func(rc restate.RunContext) (string, error) {
    return callAPI1(), nil
})

fut2 := restate.RunAsync(ctx, func(rc restate.RunContext) (string, error) {
    return callAPI2(), nil
})

// Wait for results
for fut, err := range restate.Wait(ctx, fut1, fut2) {
    // Process results
}
```

### ❌ Anti-Pattern: Blocking in Futures

```go
// ❌ WRONG - Don't block immediately after creating future
fut := restate.Service[T](ctx, "Svc", "Handler").RequestFuture(input)
result, _ := fut.Response() // Blocks immediately - no parallelism!

// ✅ CORRECT - Create multiple futures first
fut1 := restate.Service[T](ctx, "Svc1", "H1").RequestFuture(input1)
fut2 := restate.Service[T](ctx, "Svc2", "H2").RequestFuture(input2)
// Now wait for all
for fut, _ := range restate.Wait(ctx, fut1, fut2) { ... }
```

## 🔄 Error Handling in Parallel Execution

### Partial Failures

```go
futures := []restate.ResponseFuture[Result]{}
// Create 5 parallel calls
for i := 0; i < 5; i++ {
    fut := restate.Service[Result](ctx, "Svc", "Handler").
        RequestFuture(data[i])
    futures = append(futures, fut)
}

var results []Result
var errors []error

// Collect results, handle partial failures
for fut, err := range restate.Wait(ctx, futures...) {
    if err != nil {
        errors = append(errors, err)
        continue
    }
    
    result, _ := fut.(restate.ResponseFuture[Result]).Response()
    results = append(results, result)
}

// Decide: Do we need all results or can we proceed with partial success?
if len(results) == 0 {
    return fmt.Errorf("all parallel calls failed")
}

if len(errors) > 0 {
    ctx.Log().Warn("Some calls failed", "errorCount", len(errors))
}

// Proceed with available results
return processResults(results)
```

### All-or-Nothing Pattern

```go
// If ANY call fails, fail the entire operation
for fut, err := range restate.Wait(ctx, fut1, fut2, fut3) {
    if err != nil {
        return fmt.Errorf("parallel execution failed: %w", err)
    }
}

// All succeeded - proceed
```

### Best-Effort Pattern

```go
// Accept whatever results we get
var successCount int
for fut, err := range restate.Wait(ctx, futures...) {
    if err == nil {
        successCount++
    }
}

ctx.Log().Info("Parallel execution complete",
    "total", len(futures),
    "successful", successCount)
```

## 💡 Optimization Strategies

### 1. Batch Similar Operations

```go
// Instead of sequential
for _, item := range items {
    process(item) // Slow
}

// Use parallel futures
futures := []restate.RunAsyncFuture[Result]{}
for _, item := range items {
    fut := restate.RunAsync(ctx, func(rc restate.RunContext) (Result, error) {
        return processItem(item), nil
    })
    futures = append(futures, fut)
}

// Collect results
for fut, _ := range restate.Wait(ctx, futures...) {
    result, _ := fut.(restate.RunAsyncFuture[Result]).Result()
    // Use result
}
```

### 2. Pipeline Pattern

```go
// Stage 1: Fetch data in parallel
fetchFutures := []restate.RunAsyncFuture[Data]{}
for _, source := range sources {
    fut := restate.RunAsync(ctx, func(rc restate.RunContext) (Data, error) {
        return fetchFrom(source), nil
    })
    fetchFutures = append(fetchFutures, fut)
}

// Stage 2: Process results in parallel
processFutures := []restate.RunAsyncFuture[Processed]{}
for fut, err := range restate.Wait(ctx, fetchFutures...) {
    if err != nil {
        continue
    }
    data, _ := fut.(restate.RunAsyncFuture[Data]).Result()
    
    pFut := restate.RunAsync(ctx, func(rc restate.RunContext) (Processed, error) {
        return process(data), nil
    })
    processFutures = append(processFutures, pFut)
}

// Stage 3: Aggregate
for fut, _ := range restate.Wait(ctx, processFutures...) {
    // Collect processed results
}
```

## ✅ Concept Check

Before moving to hands-on, ensure you understand:

- [ ] Fan-out/fan-in pattern and when to use it
- [ ] Difference between `Request` and `RequestFuture`
- [ ] How to use `restate.Wait` to collect results
- [ ] When to use `RunAsync` for side effects
- [ ] How futures are journaled and replayed
- [ ] Why goroutines + futures is an anti-pattern
- [ ] Handling partial failures in parallel execution

## 🎯 Next Step

Ready to build a real parallel execution pipeline!

👉 **Continue to [Hands-On Tutorial](./02-hands-on.md)**

---

**Key Takeaway:** Parallelism dramatically reduces latency. Use futures for async operations and `restate.Wait` to collect results safely!
