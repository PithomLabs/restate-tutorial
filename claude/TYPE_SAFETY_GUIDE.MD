# Enhanced Guardrails with Type Safety - Complete Guide

## Overview

The framework now provides compile-time type safety through Go generics and type constraints, preventing common mistakes before code even runs. This dramatically reduces runtime errors and improves developer experience.

## Type Safety Enhancements

### 1. Compile-Time Context Constraints
### 2. Safe State Operations
### 3. Type-Safe Context Wrappers
### 4. Runtime Validation (Fallback)

## Type Constraints

### MutableContext
Contexts that allow state mutation:

```go
type MutableContext interface {
    restate.ObjectContext | restate.WorkflowContext
}
```

**Usage:** Guarantees state can be modified.

### ReadOnlyContext
Contexts that are read-only:

```go
type ReadOnlyContext interface {
    restate.ObjectSharedContext | restate.WorkflowSharedContext
}
```

**Usage:** Prevents accidental state mutation.

### AnyRestateContext
Any valid Restate context:

```go
type AnyRestateContext interface {
    restate.Context | restate.ObjectContext | restate.WorkflowContext |
    restate.ObjectSharedContext | restate.WorkflowSharedContext | restate.RunContext
}
```

**Usage:** Generic operations that work with any context.

## SafeState - Compile-Time Guaranteed Mutation

### Definition

```go
type SafeState[T any, C MutableContext] struct {
    key     string
    ctx     C
}

func NewSafeState[T any, C MutableContext](
    ctx C,
    key string,
) *SafeState[T, C]
```

### Methods

```go
func (s *SafeState[T, C]) Get() (T, error)
func (s *SafeState[T, C]) Set(value T) error
func (s *SafeState[T, C]) Clear()
```

### Why It's Safe

The `MutableContext` constraint **prevents** you from creating a `SafeState` with read-only contexts at **compile time**.

## Usage Examples

### Example 1: Compile-Time Safe State Mutation

```go
type UserAccount struct{}

// Run handler (exclusive access - can mutate)
func (UserAccount) UpdateBalance(
    ctx restate.ObjectContext,  // Mutable context
    amount float64,
) error {
    // ✓ COMPILES: ObjectContext satisfies MutableContext
    balance := NewSafeState[float64, restate.ObjectContext](ctx, "balance")
    
    current, err := balance.Get()
    if err != nil {
        current = 0.0
    }
    
    // Type-safe mutation
    return balance.Set(current + amount)
}

// Shared handler (concurrent access - read-only)
func (UserAccount) GetBalance(
    ctx restate.ObjectSharedContext,  // Read-only context
) (float64, error) {
    // ✗ COMPILE ERROR: ObjectSharedContext doesn't satisfy MutableContext
    // balance := NewSafeState[float64, restate.ObjectSharedContext](ctx, "balance")
    
    // ✓ COMPILES: Use ReadOnlyState instead
    balance := NewReadOnlyState[float64, restate.ObjectSharedContext](ctx, "balance")
    
    return balance.Get()  // Only Get() is available - no Set()!
}
```

**Key Benefit:** Impossible to accidentally mutate state from shared handler - **caught at compile time!**

### Example 2: Before vs After

#### Before (Runtime Errors Possible)

```go
func (MyService) SharedHandler(ctx restate.ObjectSharedContext) error {
    // Runtime error - will panic or return error at runtime
    state := State[int]{ctx: ctx, key: "counter"}
    state.Set(100)  // ERROR at runtime!
    
    return nil
}
```

#### After (Compile-Time Safety)

```go
func (MyService) SharedHandler(ctx restate.ObjectSharedContext) error {
    // ✗ COMPILE ERROR: Cannot create SafeState with ObjectSharedContext
    // state := NewSafeState[int, restate.ObjectSharedContext](ctx, "counter")
    
    // ✓ COMPILES: Use read-only state
    state := NewReadOnlyState[int, restate.ObjectSharedContext](ctx, "counter")
    value, _ := state.Get()  // Only Get() exists!
    
    // state.Set(100) // ✗ Method doesn't exist!
    
    return nil
}
```

### Example 3: Type-Safe Workflow State

```go
type OrderWorkflow struct{}

func (OrderWorkflow) Run(
    ctx restate.WorkflowContext,  // Mutable
    order Order,
) error {
    // ✓ Safe mutation in run handler
    status := NewSafeState[string, restate.WorkflowContext](ctx, "status")
    status.Set("processing")
    
    // Process order...
    
    status.Set("completed")
    return nil
}

func (OrderWorkflow) GetStatus(
    ctx restate.WorkflowSharedContext,  // Read-only
) (string, error) {
    // ✓ Read-only access in shared handler
    status := NewReadOnlyState[string, restate.WorkflowSharedContext](ctx, "status")
    
    return status.Get()  // Cannot Set() - method doesn't exist!
}
```

### Example 4: Compile-Time Context Assertions

```go
func requireMutableContext[C MutableContext](ctx C) {
    // Function signature guarantees ctx is mutable
    state := NewSafeState[int, C](ctx, "value")
    state.Set(42)  // Always safe!
}

func requireReadOnlyContext[C ReadOnlyContext](ctx C) {
    // Function signature guarantees ctx is read-only
    state := NewReadOnlyState[int, C](ctx, "value")
    value, _ := state.Get()  // Only read operations available
    
    // state.Set(42) // ✗ Compile error - Set doesn't exist!
}

// Usage
func (MyObject) Handler(ctx restate.ObjectContext) error {
    requireMutableContext(ctx)  // ✓ Compiles
    // requireReadOnlyContext(ctx)  // ✗ Compile error!
    return nil
}

func (MyObject) SharedHandler(ctx restate.ObjectSharedContext) error {
    // requireMutableContext(ctx)  // ✗ Compile error!
    requireReadOnlyContext(ctx)  // ✓ Compiles
    return nil
}
```

### Example 5: Type-Safe Generic Helpers

```go
// Generic state updater - only works with mutable contexts
func incrementCounter[C MutableContext](ctx C, key string) error {
    counter := NewSafeState[int, C](ctx, key)
    
    current, err := counter.Get()
    if err != nil {
        current = 0
    }
    
    return counter.Set(current + 1)  // Guaranteed to work!
}

// Usage in run handler
func (Counter) Increment(ctx restate.ObjectContext) error {
    return incrementCounter(ctx, "count")  // ✓ Compiles
}

// Usage in shared handler
func (Counter) TryIncrement(ctx restate.ObjectSharedContext) error {
    // return incrementCounter(ctx, "count")  // ✗ Compile error!
    // ObjectSharedContext doesn't satisfy MutableContext
    return nil
}
```

### Example 6: Type-Safe Run Wrapper

```go
func processWithSideEffect[C AnyRestateContext](
    ctx C,
    data string,
) (string, error) {
    // TypeSafeRun works with any Restate context
    return TypeSafeRun(ctx, func(rc restate.RunContext) (string, error) {
        // Side effect here
        return callExternalAPI(data)
    })
}

// Works with any context type
func (MyService) Handler1(ctx restate.Context) error {
    result, _ := processWithSideEffect(ctx, "data")
    return nil
}

func (MyObject) Handler2(ctx restate.ObjectContext) error {
    result, _ := processWithSideEffect(ctx, "data")
    return nil
}

func (MyWorkflow) Handler3(ctx restate.WorkflowContext) error {
    result, _ := processWithSideEffect(ctx, "data")
    return nil
}
```

### Example 7: Preventing State Leakage

```go
type BankAccount struct{}

// WRONG: Using old State[T] - runtime error possible
func (BankAccount) GetBalanceWrong(
    ctx restate.ObjectSharedContext,
) (float64, error) {
    // This compiles but will fail at runtime!
    balance := State[float64]{ctx: ctx, key: "balance"}
    
    // Mutation attempt from shared handler - RUNTIME ERROR
    balance.Set(1000.0)  
    return balance.Get()
}

// RIGHT: Using SafeState/ReadOnlyState - compile-time safety
func (BankAccount) GetBalanceRight(
    ctx restate.ObjectSharedContext,
) (float64, error) {
    // ✓ ReadOnlyState for shared context
    balance := NewReadOnlyState[float64, restate.ObjectSharedContext](
        ctx,
        "balance",
    )
    
    // Only Get() available - Set() doesn't exist!
    return balance.Get()
}
```

### Example 8: Builder Pattern with Type Safety

```go
// Type-safe workflow builder
type WorkflowStateBuilder[C MutableContext] struct {
    ctx   C
    state map[string]interface{}
}

func NewWorkflowStateBuilder[C MutableContext](ctx C) *WorkflowStateBuilder[C] {
    return &WorkflowStateBuilder[C]{
        ctx:   ctx,
        state: make(map[string]interface{}),
    }
}

func (b *WorkflowStateBuilder[C]) SetString(key, value string) *WorkflowStateBuilder[C] {
    state := NewSafeState[string, C](b.ctx, key)
    state.Set(value)
    return b
}

func (b *WorkflowStateBuilder[C]) SetInt(key string, value int) *WorkflowStateBuilder[C] {
    state := NewSafeState[int, C](b.ctx, key)
    state.Set(value)
    return b
}

// Usage - only works with mutable contexts
func (MyWorkflow) Run(ctx restate.WorkflowContext, req Request) error {
    // ✓ Compiles - WorkflowContext is mutable
    NewWorkflowStateBuilder(ctx).
        SetString("status", "running").
        SetInt("progress", 0)
    
    return nil
}

func (MyWorkflow) GetData(ctx restate.WorkflowSharedContext) error {
    // ✗ Compile error - WorkflowSharedContext is not mutable
    // NewWorkflowStateBuilder(ctx).SetString("status", "done")
    
    return nil
}
```

## Comparison: Old vs New Guardrails

### Old (Runtime Validation)

```go
type State[T any] struct {
    ctx interface{}
    key string
}

func (s *State[T]) Set(value T) error {
    switch c := s.ctx.(type) {
    case restate.ObjectContext:
        restate.Set(c, s.key, value)
        return nil
    case restate.WorkflowContext:
        restate.Set(c, s.key, value)
        return nil
    default:
        // ✗ Runtime error!
        return restate.TerminalError(
            fmt.Errorf("Set called from read-only context: %T", s.ctx),
            400,
        )
    }
}
```

**Problem:** Error only caught at runtime when Set() is called.

### New (Compile-Time Validation)

```go
type SafeState[T any, C MutableContext] struct {
    ctx C
    key string
}

func NewSafeState[T any, C MutableContext](ctx C, key string) *SafeState[T, C] {
    return &SafeState[T, C]{ctx: ctx, key: key}
}
```

**Benefit:** Cannot even create `SafeState` with wrong context type - **compile error!**

## Best Practices

### ✅ DO: Use Type Constraints

```go
// ✓ Function signature prevents misuse
func updateState[C MutableContext](ctx C, key string, value int) {
    state := NewSafeState[int, C](ctx, key)
    state.Set(value)  // Guaranteed safe
}
```

### ✅ DO: Use SafeState for Exclusive Handlers

```go
// ✓ Run handler = mutable context = SafeState
func (MyObject) Run(ctx restate.ObjectContext) error {
    state := NewSafeState[Data, restate.ObjectContext](ctx, "data")
    return state.Set(newData)
}
```

### ✅ DO: Use ReadOnlyState for Shared Handlers

```go
// ✓ Shared handler = read-only context = ReadOnlyState
func (MyObject) Get(ctx restate.ObjectSharedContext) (Data, error) {
    state := NewReadOnlyState[Data, restate.ObjectSharedContext](ctx, "data")
    return state.Get()
}
```

### ❌ DON'T: Mix Context Types

```go
// ✗ Don't try to force wrong context type
func wrongUsage(ctx restate.ObjectSharedContext) {
    // This won't compile!
    // state := NewSafeState[int, restate.ObjectSharedContext](ctx, "key")
}
```

### ❌ DON'T: Bypass Type Safety

```go
// ✗ Don't use interface{} to bypass constraints
func badPattern(ctx interface{}, key string) {
    // Loses type safety - don't do this!
}

// ✓ Use proper constraints
func goodPattern[C MutableContext](ctx C, key string) {
    // Type-safe!
}
```

## Migration Guide

### Step 1: Identify Mutable Operations

Review your code for state mutations:

```go
// Old
state := State[int]{ctx: ctx, key: "counter"}
state.Set(42)
```

### Step 2: Replace with SafeState

In exclusive/run handlers:

```go
// New
state := NewSafeState[int, restate.ObjectContext](ctx, "counter")
state.Set(42)  // Type-safe!
```

### Step 3: Use ReadOnlyState in Shared Handlers

```go
// New
state := NewReadOnlyState[int, restate.ObjectSharedContext](ctx, "counter")
value, _ := state.Get()
```

### Step 4: Add Type Constraints to Functions

```go
// Old
func helper(ctx interface{}, key string) {
    // Runtime checks needed
}

// New
func helper[C MutableContext](ctx C, key string) {
    // Compile-time guarantees!
}
```

## Compile-Time Error Messages

### Error: Context doesn't implement constraint

```go
state := NewSafeState[int, restate.ObjectSharedContext](ctx, "key")

// Error: restate.ObjectSharedContext does not implement MutableContext
//        (missing in restate.ObjectContext | restate.WorkflowContext)
```

**Solution:** Use `ReadOnlyState` instead.

### Error: Type mismatch

```go
func updateCounter[C ReadOnlyContext](ctx C) {
    state := NewSafeState[int, C](ctx, "counter")  // Error!
    //                          ^
    // Error: C (which implements ReadOnlyContext) does not implement MutableContext
}
```

**Solution:** Change constraint to `MutableContext` or use `ReadOnlyState`.

## Testing

```go
func TestSafeState_Set(t *testing.T) {
    ctx := mockObjectContext()
    
    state := NewSafeState[string, restate.ObjectContext](ctx, "test")
    err := state.Set("value")
    
    assert.NoError(t, err)
    
    result, err := state.Get()
    assert.NoError(t, err)
    assert.Equal(t, "value", result)
}

func TestReadOnlyState_CannotMutate(t *testing.T) {
    ctx := mockObjectSharedContext()
    
    state := NewReadOnlyState[string, restate.ObjectSharedContext](ctx, "test")
    
    // This line won't even compile!
    // state.Set("value")  // Method doesn't exist
    
    // Only Get() is available
    value, err := state.Get()
    assert.NoError(t, err)
}
```

## Benefits Summary

| Feature | Old Guardrails | New Type-Safe Guardrails |
|---------|---------------|--------------------------|
| **State Mutation Safety** | Runtime error | Compile-time error |
| **Context Validation** | Runtime check | Type constraint |
| **IDE Support** | Limited | Full autocomplete |
| **Error Detection** | Testing | During coding |
| **Performance** | Runtime overhead | Zero overhead |
| **Developer Experience** | Trial and error | Immediate feedback |

## Conclusion

Enhanced guardrails with compile-time type safety provide:

- ✅ **Compile-time prevention** of state mutation in read-only contexts
- ✅ **Zero runtime overhead** - errors caught before code runs
- ✅ **Better IDE support** - autocomplete knows valid operations
- ✅ **Clearer intent** - function signatures document requirements
- ✅ **Fewer bugs** - impossible to make certain mistakes
- ✅ **Better refactoring** - compiler catches all affected code

**Recommendation:** Use `SafeState` for exclusive handlers, `ReadOnlyState` for shared handlers, and type constraints for generic functions. The old `State[T]` remains for backward compatibility but prefer the new type-safe variants.
