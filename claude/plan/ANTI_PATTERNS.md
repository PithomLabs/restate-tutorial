# Anti-Pattern Protection Guide

> **Preventing common mistakes in Restate applications through compile-time and runtime guards**

## Overview

The framework provides **three layers of protection** against common anti-patterns in Restate development:

1. **🛡️ Type-Safe Wrappers** - Use Go's type system to prevent dangerous patterns at compile time
2. **⚠️ Runtime Guards** - Detect anti-patterns during execution with clear error messages  
3. **📊 Static Analysis** - Use custom linters to catch issues in CI/CD (see `tools/restatelint/`)

## Protection Coverage

| Anti-Pattern | Type-Safe | Runtime | Static Lint | Status |
|--------------|-----------|---------|-------------|--------|
| Context in `restate.Run` | ✅ `SafeRun` | ✅ `DetectContextMisuse` | ⚠️ Planned | **Protected** |
| Non-deterministic maps | ✅ `DeterministicMap` | ✅ `ValidateMapIteration` | ⚠️ Planned | **Protected** |
| Blocking handlers | ❌ | ✅ `WarnOnBlockingCall` | ⚠️ Planned | **Partial** |
| Self-referencing objects | ❌ | ✅ `DetectSelfReferencingCall` | ⚠️ Planned | **Protected** |
| Futures in goroutines | ❌ | ⚠️ `GuardAgainstGoroutines` | ⚠️ Planned | **Partial** |
| Global state usage | ❌ | ❌ | ⚠️ Planned | **Docs only** |

---

## Anti-Pattern #1: Context Misuse in `restate.Run`

### ❌ The Problem

Using the parent Restate context inside a `restate.Run` closure breaks durable execution:

```go
func BadHandler(ctx restate.Context, input string) (string, error) {
    result, err := restate.Run(ctx, func(rc restate.RunContext) (string, error) {
        // ❌ WRONG: Using 'ctx' instead of 'rc'
        // This will crash during replay
        return restate.Service[string](ctx, "Service", "handler").Request(input)
    })
    return result, err
}
```

**Why it's dangerous:**
- Restate journaling only works with operations called on the `RunContext`
- Using parent `ctx` causes non-deterministic behavior during replay
- Leads to runtime panics that are hard to debug

###✅ Solution 1: SafeRun (Type-Safe)

The `SafeRun` wrapper makes it **impossible** to access the parent context:

```go
func GoodHandler(ctx restate.Context, input string) (string, error) {
    result, err := SafeRun(ctx, func(rc SafeRunContext) (string, error) {
        // ✅ CORRECT: Only SafeRunContext available
        // Attempting to use 'ctx' here would be a compile error
        return callExternalAPI(), nil
    })
    return result, err
}
```

### ✅ Solution 2: Runtime Detection

Add a guard at the start of your `Run` closure:

```go
func HandlerWithGuard(ctx restate.Context, input string) (string, error) {
    return restate.Run(ctx, func(rc restate.RunContext) (string, error) {
        // Runtime check - will error if you accidentally captured 'ctx'
        if err := DetectContextMisuse(ctx); err != nil {
            return "", err
        }
        return callExternalAPI(), nil
    })
}
```

---

## Anti-Pattern #2: Non-Deterministic Map Iteration

### ❌ The Problem

Standard Go maps have **random iteration order**, which breaks Restate replay:

```go
func BadProcessor(ctx restate.Context, items map[string]string) error {
    // ❌ WRONG: Order changes between runs
    for key, value := range items {
        _, err := restate.Service[Void](ctx, "Processor", "process").
            Request(key)
        if err != nil {
            return err
        }
    }
    return nil
}
```

**What goes wrong:**
1. First execution: processes keys in order `[a, b, c]`
2. Replay: iteration order is `[c, a, b]`
3. Journaled operations don't match → **replay failure**

### ✅ Solution: DeterministicMap

Use the framework's deterministic map that maintains insertion order:

```go
func GoodProcessor(ctx restate.Context, items *DeterministicMap[string, string]) error {
    // ✅ CORRECT: Iteration order is deterministic
    items.Range(func(key string, value string) bool {
        _, err := restate.Service[Void](ctx, "Processor", "process").
            Request(key)
        if err != nil {
            ctx.Log().Error("processing failed", "key", key, "error", err)
            return false // stop iteration
        }
        return true // continue
    })
    return nil
}
```

**DeterministicMap API:**

```go
// Create
m := NewDeterministicMap[string, int]()

// Insert (maintains order)
m.Set("first", 1)
m.Set("second", 2)
m.Set("third", 3)

// Iterate (always in insertion order)
m.Range(func(k string, v int) bool {
    fmt.Printf("%s: %d\n", k, v)
    return true
})

// Get/Delete
value, exists := m.Get("first")
m.Delete("second")

// Utility
keys := m.Keys()     // []string in order
values := m.Values() // []int in order
len := m.Len()
```

---

## Anti-Pattern #3: Blocking Operations in Handlers

### ❌ The Problem

Using `time.Sleep` or long-running operations in object handlers blocks the entire object key:

```go
func (o *OrderObject) ProcessOrder(ctx restate.ObjectContext, orderID string) error {
    // ❌ WRONG: Blocks all requests for this order key
    time.Sleep(30 * time.Second)
    return processOrder(orderID)
}
```

**Impact:**
- Other requests to the same object key must wait
- No concurrency for that key
- Poor system throughput

### ✅ Solution: Use Durable Sleep

```go
func (o *OrderObject) ProcessOrder(ctx restate.ObjectContext, orderID string) error {
    // ✅ CORRECT: Durable sleep releases the handler
    if err := restate.Sleep(ctx, 30*time.Second); err != nil {
        return err
    }
    return processOrder(orderID)
}
```

### ✅ Alternative: Early Warning System

Add runtime monitoring to detect slow handlers:

```go
func (o *OrderObject) ProcessOrder(ctx restate.ObjectContext, orderID string) error {
    // Warn if handler takes >5 seconds
    defer WarnOnBlockingCall(ctx, 5*time.Second)()
    
    return processOrder(orderID)
}
```

### ✅ Validation Wrapper

Wrap your handler logic to enforce duration limits:

```go
func (o *OrderObject) ProcessOrder(ctx restate.ObjectContext, orderID string) error {
    return ValidateHandlerDuration(ctx, 10*time.Second, func() error {
        // Handler logic here
        return processOrder(orderID)
    })
}
```

---

## Anti-Pattern #4: Self-Referencing Object Calls

### ❌ The Problem

An object calling itself on the same key with an **exclusive handler** causes deadlock:

```go
func (o *UserObject) UpdateProfile(ctx restate.ObjectContext, data ProfileData) error {
    myKey := restate.Key(ctx)
    
    // ❌ WRONG: Calling self = deadlock
    // This handler is exclusive, so it locks the key
    _, err := restate.Object[Void](ctx, "UserObject", myKey, "ValidateProfile").
        Request(data)
    return err
}
```

**Why it deadlocks:**
1. `UpdateProfile` acquires exclusive lock on key "user-123"
2. It tries to call `ValidateProfile` on the same key
3. `ValidateProfile` waits for the lock to be released
4. `UpdateProfile` waits for `ValidateProfile` to complete
5. **Deadlock!** 🔒

### ✅ Solution 1: Use Shared Handlers

Make the target handler `shared` instead of exclusive:

```go
// Define as shared handler (allows concurrent access)
func (o *UserObject) ValidateProfile(ctx restate.ObjectSharedContext, data ProfileData) (bool, error) {
    // Read-only validation logic
    return isValid(data), nil
}

func (o *UserObject) UpdateProfile(ctx restate.ObjectContext, data ProfileData) error {
    myKey := restate.Key(ctx)
    
    // ✅ OK: Calling shared handler doesn't block
    valid, err := restate.Object[bool](ctx, "UserObject", myKey, "ValidateProfile").
        Request(data)
    if !valid {
        return fmt.Errorf("invalid profile data")
    }
    
    // Continue with update...
    return nil
}
```

### ✅ Solution 2: Runtime Detection

Add a guard before object calls:

```go
func (o *UserObject) UpdateProfile(ctx restate.ObjectContext, data ProfileData) error {
    targetKey := getTargetKey()
    
    // Guard against self-call
    if err := DetectSelfReferencingCall(ctx, "UserObject", targetKey); err != nil {
        return err
    }
    
    _, err := restate.Object[Void](ctx, "UserObject", targetKey, "SomeHandler").
        Request(data)
    return err
}
```

---

## Anti-Pattern #5: Futures in Goroutines

### ❌ The Problem

Handling Restate futures in goroutines breaks journaling:

```go
func BadConcurrentCalls(ctx restate.Context) error {
    fut1 := restate.Service[string](ctx, "Svc1", "handler").RequestFuture("req1")
    fut2 := restate.Service[string](ctx, "Svc2", "handler").RequestFuture("req2")
    
    // ❌ WRONG: Futures in goroutines
    go func() {
        result, _ := fut1.Response()
        fmt.Println(result)
    }()
    
    go func() {
        result, _ := fut2.Response()
        fmt.Println(result)
    }()
    
    return nil
}
```

**Problems:**
- Restate can't track future resolutions across goroutines
- Non-deterministic execution order
- Race conditions during replay

### ✅ Solution: Use `restate.Wait`

```go
func GoodConcurrentCalls(ctx restate.Context) ([]string, error) {
    fut1 := restate.Service[string](ctx, "Svc1", "handler").RequestFuture("req1")
    fut2 := restate.Service[string](ctx, "Svc2", "handler").RequestFuture("req2")
    
    // ✅ CORRECT: Wait for all futures deterministically
    results := []string{}
    for fut, err := range restate.Wait(ctx, fut1, fut2) {
        if err != nil {
            return nil, err
        }
        response, err := fut.(restate.ResponseFuture[string]).Response()
        if err != nil {
            return nil, err
        }
        results = append(results, response)
    }
    
    return results, nil
}
```

---

## Anti-Pattern #6: Global State

### ❌ The Problem

Using global variables or in-memory state that doesn't survive restarts:

```go
// ❌ WRONG: Global state
var requestCount int

func BadHandler(ctx restate.Context, input string) error {
    requestCount++  // Lost on restart
    return process(input)
}
```

### ✅ Solution: Use Restate State

```go
type MyObject struct{}

func (o *MyObject) GoodHandler(ctx restate.ObjectContext, input string) error {
    // ✅ CORRECT: Durable state
    count, err := restate.Get[int](ctx, "requestCount")
    if err != nil {
        return err
    }
    
    count++
    restate.Set(ctx, "requestCount", count)
    
    return process(input)
}
```

---

## Best Practices Checklist

When writing Restate handlers, verify:

- [ ] ✅ Using `SafeRun` or checking contexts in `restate.Run` closures
- [ ] ✅ Using `DeterministicMap` instead of `map[K]V` for iteration
- [ ] ✅ Using `restate.Sleep` instead of `time.Sleep`
- [ ] ✅ No self-referencing calls on exclusive object handlers
- [ ] ✅ Handling futures with `restate.Wait`, not goroutines
- [ ] ✅ Using `restate.Get/Set` for state, not global variables
- [ ] ✅ Adding `WarnOnBlockingCall` to long-running handlers
- [ ] ✅ Running static analysis with `restatelint` in CI/CD

---

## Quick Reference

### Type-Safe Wrappers

```go
// Safe Run context
SafeRun(ctx, func(rc SafeRunContext) (T, error) { ... })
SafeRunAsync(ctx, func(rc SafeRunContext) (T, error) { ... })

// Deterministic collections
m := NewDeterministicMap[K, V]()
m.Set(key, value)
m.Range(func(k K, v V) bool { return true })
```

### Runtime Guards

```go
// Context misuse detection
DetectContextMisuse(suspectCtx)

// Blocking operation warning
defer WarnOnBlockingCall(ctx, threshold)()

// Duration validation
ValidateHandlerDuration(ctx, maxDuration, handlerFunc)

// Self-call detection
DetectSelfReferencingCall(ctx, serviceName, targetKey)

// Map iteration validation
ValidateMapIteration(myMap)
```

### Anti-Pattern Logging

```go
// Log structured anti-pattern warnings
LogAntiPatternWarning(ctx, AntiPatternContextMisuse, "details...")

// Get anti-pattern information
ap := GetAntiPatternByCategory(AntiPatternDeadlock)
fmt.Printf("Fix: %s\n", ap.Fix)
```

---

## Static Analysis (Future)

For CI/CD integration, use the custom linter:

```bash
# Run anti-pattern checks
go run ./tools/restatelint/cmd/restatelint ./...

# Integrate with golangci-lint
golangci-lint run --enable=restatelint
```

Configuration in `.golangci.yml`:

```yaml
linters-settings:
  custom:
    restatelint:
      path: ./tools/restatelint
      description: Detects Restate anti-patterns
      original-url: github.com/yourorg/restatelint
```

---

## Summary

The framework provides **defense in depth** against anti-patterns:

1. **Prevention** - Type-safe wrappers make mistakes impossible
2. **Detection** - Runtime guards catch issues early with clear errors
3. **Education** - Documentation and examples show the right way
4. **Automation** - Static analysis catches issues in CI/CD

**Remember:** The best protection is understanding *why* these patterns are problematic. Read the Restate documentation on [durable execution](https://docs.restate.dev/) to learn the underlying concepts.
