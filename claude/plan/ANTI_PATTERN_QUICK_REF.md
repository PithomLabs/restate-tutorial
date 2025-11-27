# Anti-Pattern Protection - Quick Reference

## 🛡️ Type-Safe Wrappers (Compile-Time)

### SafeRun - Prevent Context Capture
```go
// ✅ Use this instead of restate.Run
result, err := SafeRun(ctx, func(rc SafeRunContext) (string, error) {
    return callExternalAPI()  // Parent ctx not accessible
})
```

### DeterministicMap - Ordered Iteration
```go
// ✅ Use this instead of map[K]V
m := NewDeterministicMap[string, int]()
m.Set("key1", 1)
m.Set("key2", 2)
m.Range(func(k string, v int) bool {
    process(k, v)
    return true  // continue
})
```

---

## ⚠️ Runtime Guards

### Context Misuse Detection
```go
restate.Run(ctx, func(rc restate.RunContext) (string, error) {
    if err := DetectContextMisuse(ctx); err != nil {
        return "", err
    }
    return callAPI()
})
```

### Blocking Operation Warning
```go
func Handler(ctx restate.ObjectContext) error {
    defer WarnOnBlockingCall(ctx, 5*time.Second)()
    // Warns if this takes >5s
    return process()
}
```

### Duration Enforcement
```go
return ValidateHandlerDuration(ctx, 10*time.Second, func() error {
    return longRunningOperation()
})
```

### Self-Call Detection
```go
targetKey := getTargetKey()
if err := DetectSelfReferencingCall(ctx, "MyObject", targetKey); err != nil {
    return err
}
```

### Map Iteration Validation
```go
if err := ValidateMapIteration(myMap); err != nil {
    log.Warn("use DeterministicMap instead")
}
```

---

## 📋 Anti-Pattern Catalog

| Category | Description | Fix |
|----------|-------------|-----|
| `AntiPatternContextMisuse` | Using parent ctx in Run | Use `SafeRun` |
| `AntiPatternNonDeterministic` | Standard map iteration | Use `DeterministicMap` |
| `AntiPatternBlockingOperation` | time.Sleep in handler | Use `restate.Sleep` |
| `AntiPatternDeadlock` | Self-referencing object call | Use shared handlers |
| `AntiPatternConcurrencyMisuse` | Futures in goroutines | Use `restate.Wait` |
| `AntiPatternStateInconsistency` | Global variables | Use `restate.Get/Set` |

---

## 🔍 Logging

```go
// Get anti-pattern info
ap := GetAntiPatternByCategory(AntiPatternDeadlock)
fmt.Printf("Fix: %s\n", ap.Fix)

// Log structured warning
LogAntiPatternWarning(ctx, AntiPatternContextMisuse, "additional details")
```

---

## ✅ Best Practices Checklist

Before committing code, verify:

- [ ] Using `SafeRun` or `DetectContextMisuse` in Run closures
- [ ] Using `DeterministicMap` instead of `map[K]V` for iteration  
- [ ] Using `restate.Sleep` instead of `time.Sleep`
- [ ] No self-calls on exclusive object handlers
- [ ] Handling futures with `restate.Wait`, not goroutines
- [ ] Using `restate.Get/Set` for state, not global vars
- [ ] Added `WarnOnBlockingCall` to long handlers

---

## 📦 Import

```go
import (
    restate "github.com/restatedev/sdk-go"
    // "github.com/yourorg/framework"  // Your framework package
)
```

---

## 🚀 Common Patterns

### External API Call
```go
SafeRun(ctx, func(rc SafeRunContext) (string, error) {
    return http.Get(url)
})
```

### Batch Processing
```go
items := NewDeterministicMap[string, Data]()
items.Range(func(key string, data Data) bool {
    process(ctx, key, data)
    return true
})
```

### Object Handler
```go
func (o *MyObject) Handler(ctx restate.ObjectContext) error {
    defer WarnOnBlockingCall(ctx, 5*time.Second)()
    
    targetKey := computeTarget()
    if err := DetectSelfReferencingCall(ctx, "MyObject", targetKey); err != nil {
        return err
    }
    
    return process()
}
```

### Concurrent Calls
```go
fut1 := restate.Service[string](ctx, "Svc1", "handler").RequestFuture("a")
fut2 := restate.Service[string](ctx, "Svc2", "handler").RequestFuture("b")

for fut, err := range restate.Wait(ctx, fut1, fut2) {
    if err != nil { return err }
    result, _ := fut.(restate.ResponseFuture[string]).Response()
    use(result)
}
```

---

## 📚 Full Documentation

- **Detailed Guide:** `ANTI_PATTERNS.md`
- **Examples:** `examples/anti_pattern_examples.go`
- **Implementation Plan:** `anti_pattern_protection_plan.md`
- **Summary:** `ANTI_PATTERN_SUMMARY.md`
