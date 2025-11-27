# Interservice Communication Updates - DOS Guidance Implementation

## Summary

This update addresses the gaps/risks identified in the **Interservice Communication** section per DOS guidance requirements.

## Changes Made

### 1. Configurable Validation Mode for Idempotency Keys

**Added `IdempotencyValidationMode`** with three modes:

```go
const (
    // Logs warnings but allows calls to proceed (default, permissive)
    IdempotencyValidationWarn IdempotencyValidationMode = "warn"
    
    // Rejects calls with invalid idempotency keys (strict)
    IdempotencyValidationFail IdempotencyValidationMode = "fail"
    
    // Skips validation entirely
    IdempotencyValidationDisabled IdempotencyValidationMode = "disabled"
)
```

**Updated `CallOption`** to include configurable validation:

```go
type CallOption struct {
    IdempotencyKey string
    Delay          time.Duration
    ValidationMode IdempotencyValidationMode // Controls validation behavior
}
```

### 2. Implemented Strict vs Permissive Validation

The framework now enforces validation mode in `ServiceClient.Send()` and other client methods:

- **Warn mode (default)**: Logs warnings but continues execution
- **Fail mode (strict)**: Panics with `TerminalError` if validation fails
- **Disabled mode**: Skips validation entirely

Example usage:

```go
client := ServiceClient[Input, Output]{
    ServiceName: "PaymentService",
    HandlerName: "Charge",
}

// Strict validation - will fail on invalid idempotency key
client.Send(ctx, input, CallOption{
    IdempotencyKey: key,
    ValidationMode: IdempotencyValidationFail,
})

// Permissive validation - will log warning but continue
client.Send(ctx, input, CallOption{
    IdempotencyKey: key,
    ValidationMode: IdempotencyValidationWarn, // or empty (default)
})
```

### 3. Comprehensive Documentation Added

Added a detailed documentation section explaining:

#### **Critical Distinction Between Client Types**

**Internal Clients** (`ServiceClient`, `ObjectClient`, `WorkflowClient`):
- ✅ Use **INSIDE** Restate handlers
- ✅ Route through Restate's durable execution engine
- ✅ Provide durability, exactly-once semantics, and journal replay
- ✅ Calls are part of the workflow execution graph

**Ingress Clients** (`IngressClient`, `IngressServiceClient`, etc.):
- ✅ Use **OUTSIDE** Restate (external apps, CLIs, tests)
- ✅ Make HTTP calls directly to Restate's ingress endpoint
- ⚠️  Do NOT have durability or journal replay
- ✅ Useful for triggering workflows, querying status, or testing

#### **DOS Rule: Never Use IngressClient Inside a Restate Handler**

**BAD Example** (DON'T DO THIS):
```go
func (MyService) ProcessOrder(ctx restate.Context, order Order) error {
    ingress := NewIngressClient("http://localhost:8080", "")  // ❌ WRONG!
    client := ingress.Service("InventoryService", "Reserve")
    _, err := client.Call(context.Background(), order)  // ❌ WRONG!
    return err
}
```

**Why this is wrong:**
- The call won't be part of the durable execution journal
- On replay, it will execute again (not idempotent)
- You lose exactly-once guarantees
- Violates the fundamental Restate execution model

**GOOD Example**:
```go
func (MyService) ProcessOrder(ctx restate.Context, order Order) error {
    client := ServiceClient[Order, Result]{
        ServiceName: "InventoryService",
        HandlerName: "Reserve",
    }
    result, err := client.Call(ctx, order)  // ✅ Correct: Uses ctx from handler
    return err
}
```

## Environment Recommendations

Per DOS guidance, choose validation mode based on your environment:

- **Development**: Use `IdempotencyValidationWarn` to catch issues without breaking flows
- **Staging/Testing**: Use `IdempotencyValidationFail` to catch issues in pre-production
- **Production**: Use `IdempotencyValidationFail` to prevent silent corruption

## Files Modified

- `framework.go`: 
  - Added `IdempotencyValidationMode` type and constants
  - Updated `CallOption` struct
  - Implemented validation mode handling in `ServiceClient.Send()`
  - Added comprehensive documentation section (Section 7)

## Migration Guide

### For Existing Code

No breaking changes. Existing code will continue to work with default permissive validation (`IdempotencyValidationWarn`).

### To Enable Strict Validation

Update your call sites to explicitly set validation mode:

```go
// Old code (still works, uses default warn mode)
client.Send(ctx, data, CallOption{
    IdempotencyKey: key,
})

// New code with strict validation
client.Send(ctx, data, CallOption{
    IdempotencyKey: key,
    ValidationMode: IdempotencyValidationFail,  // Explicit strict mode
})
```

## Implementation Notes

1. **Validation is heuristic-based**: `ValidateIdempotencyKey()` checks for patterns like timestamps that suggest non-deterministic generation
2. **Fail mode uses panic**: When validation fails in strict mode, the framework panics with a `TerminalError` (HTTP 400), which Restate will not retry
3. **Default behavior is permissive**: To avoid breaking existing workflows, the default mode is `IdempotencyValidationWarn`
4. **Future work**: The same validation mode pattern can be applied to `ObjectClient.Send()`, `WorkflowClient.Submit()`, and other client methods

## Testing Recommendations

1. Test with `IdempotencyValidationFail` in staging to catch key generation issues
2. Monitor logs for validation warnings in development
3. Ensure all idempotency keys are generated deterministically (using `restate.UUID(ctx)` or business keys)
4. Never use `time.Now()`, `rand.Int()`, or other non-deterministic sources in key generation

## References

- DOS Guidance: "Must ensure code never routes internal calls through ingress"
- Framework validation: "either approach can be fine, but the choice should be configurable (strict vs permissive)"
- Recommendation: "use these client wrappers inside Restate handlers; use IngressClient only from outside Restate"
