# Module 03: Concurrent Execution - Fan-Out/Fan-In Patterns

> **Master concurrent service calls and parallel execution for high-performance distributed systems**

## 🎯 Learning Objectives

By the end of this module, you will:

- ✅ Understand fan-out/fan-in patterns in distributed systems
- ✅ Use `RequestFuture` for asynchronous service calls
- ✅ Implement parallel execution with `restate.Wait`
- ✅ Handle partial failures in concurrent operations
- ✅ Optimize service latency with parallelism
- ✅ Use `RunAsync` for parallel side effects

## 📚 Module Contents

| File | Description | Time |
|------|-------------|------|
| **[01-concepts.md](./01-concepts.md)** | Concurrency patterns and futures | 20 min |
| **[02-hands-on.md](./02-hands-on.md)** | Build order processing pipeline | 40 min |
| **[03-validation.md](./03-validation.md)** | Test concurrent execution | 15 min |
| **[04-exercises.md](./04-exercises.md)** | Practice exercises | 20 min |

## 🎓 Prerequisites

- Completed [Module 02: Side Effects](../02-side-effects/README.md)
- Understanding of `restate.Run` and journaling
- Familiarity with async/await patterns (helpful but not required)

## 🏗️ What You'll Build

**Project: Multi-Service Order Processing Pipeline**

A realistic e-commerce order processor that:
- Validates inventory across multiple warehouses (parallel)
- Checks payment and fraud detection (parallel)
- Calculates shipping costs from multiple carriers (parallel)
- Aggregates results and creates order

```
Order Request
     ↓
┌────────────────────────────────────┐
│ Fan-Out (Parallel Execution)       │
├────────────────────────────────────┤
│  ├─→ Check Inventory (Warehouse A) │
│  ├─→ Check Inventory (Warehouse B) │
│  ├─→ Validate Payment              │
│  ├─→ Fraud Detection               │
│  └─→ Calculate Shipping            │
└────────────────────────────────────┘
     ↓
┌────────────────────────────────────┐
│ Fan-In (Aggregate Results)         │
└────────────────────────────────────┘
     ↓
Create Order (if all checks pass)
```

## 💡 Key Concepts Preview

### Sequential vs Parallel

**Sequential (Slow):**
```go
// Takes 3 seconds total if each call is 1 second
result1 := callService1() // 1s
result2 := callService2() // 1s  
result3 := callService3() // 1s
```

**Parallel (Fast):**
```go
// Takes ~1 second total (all execute simultaneously)
future1 := callService1Async()
future2 := callService2Async()
future3 := callService3Async()

// Wait for all
result1 := await future1
result2 := await future2
result3 := await future3
```

### Restate Futures

```go
// Start async operations
fut1 := restate.Service[T](ctx, "Svc1", "Handler").RequestFuture(input)
fut2 := restate.Service[T](ctx, "Svc2", "Handler").RequestFuture(input)

// Do other work...

// Wait for results
for fut, err := range restate.Wait(ctx, fut1, fut2) {
    if err != nil {
        // Handle error
    }
    // Process result
}
```

## 📁 Module Structure

```
03-concurrency/
├── README.md           ← You are here
├── 01-concepts.md      ← Concurrency patterns
├── 02-hands-on.md      ← Build order processor
├── 03-validation.md    ← Test parallel execution
├── 04-exercises.md     ← Practice
├── code/
│   ├── main.go
│   ├── order_service.go
│   ├── supporting_services.go
│   └── go.mod
└── solutions/
    └── *.go
```

## 🎯 Success Criteria

You've mastered this module when you can:

- [ ] Explain fan-out/fan-in patterns
- [ ] Use `RequestFuture` for async calls
- [ ] Implement parallel execution with `restate.Wait`
- [ ] Handle partial failures in concurrent operations
- [ ] Measure and optimize service latency
- [ ] Choose between sequential and parallel execution

## ⏱️ Time Commitment

- **Minimum:** 45 minutes (concepts + hands-on)
- **Recommended:** 1.5 hours (all materials)
- **Mastery:** 2 hours (with exercises)

## 🚀 Performance Benefits

Typical improvements with parallelization:

| Pattern | Sequential Time | Parallel Time | Speedup |
|---------|----------------|---------------|---------|
| 3 API calls (100ms each) | 300ms | ~100ms | **3x faster** |
| 5 validations (50ms each) | 250ms | ~50ms | **5x faster** |
| 10 inventory checks (80ms each) | 800ms | ~80ms | **10x faster** |

## ⚠️ Important Concepts

### Futures are Journaled

```go
// These futures are journaled by Restate
fut1 := restate.Service[T](ctx, "Svc", "Handler").RequestFuture(input)
fut2 := restate.Service[T](ctx, "Svc", "Handler").RequestFuture(input)

// On replay, Restate replays the results from the journal
// No duplicate service calls!
```

### Safe Concurrency

```go
// ✅ CORRECT - Use restate.Wait
for fut, err := range restate.Wait(ctx, future1, future2) {
    // Process results
}

// ❌ WRONG - Don't use Go goroutines with futures
go func() {
    result := future1.Result() // Anti-pattern!
}()
```

## 🔗 Next Module

After completing this module:

👉 **[Module 4: Virtual Objects - Stateful Services](../04-virtual-objects/README.md)**

Learn to build stateful, key-addressable services!

---

**Ready to speed things up?** Start with [Concepts](./01-concepts.md)!
