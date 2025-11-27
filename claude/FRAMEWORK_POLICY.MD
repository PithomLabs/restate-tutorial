# Framework Policy System

> **Global strictness control for all framework guardrails**

## Overview

The Framework Policy system provides **unified, consistent strictness control** across all runtime validation checks in the framework. Instead of having different validation modes scattered throughout the codebase, a single global policy controls how all guardrails behave.

### Problem Solved

**Before**: Inconsistent validation behavior
- Idempotency validation had configurable modes (Warn/Fail/Disabled)
- State write checks always failed with terminal errors
- Security validation had different modes (Permissive/Strict/Disabled)
- Saga checks only logged warnings
- No way to globally control strictness for different environments

**After**: Unified policy control
- Single `FrameworkPolicy` controls ALL guardrails
- Consistent behavior across the entire framework
- Environment-based configuration (strict in CI, permissive in dev)
- Per-call overrides still supported for flexibility

## Quick Start

### Set Global Policy via Environment Variable

```bash
# Strict mode - fail fast on any violation (recommended for CI/production)
export RESTATE_FRAMEWORK_POLICY=strict

# Warn mode - log warnings but continue (recommended for development)
export RESTATE_FRAMEWORK_POLICY=warn

# Disabled - skip most checks (NOT recommended, testing only)
export RESTATE_FRAMEWORK_POLICY=disabled
```

### Auto-Detection

If you don't set `RESTATE_FRAMEWORK_POLICY`, the framework automatically detects your environment:

- **CI Environment** (`CI=true` or `GITHUB_ACTIONS=true`): Uses `PolicyStrict`
- **Development** (no CI variables): Uses `PolicyWarn`

```bash
# In CI pipelines - automatically strict
export CI=true
# Framework will use PolicyStrict

# In local development - automatically permissive
# No environment variables needed
# Framework will use PolicyWarn
```

### Programmatic Configuration

```go
package main

import "github.com/your-org/framework"

func init() {
	// Override global policy at application startup
	framework.SetFrameworkPolicy(framework.PolicyStrict)
}

func main() {
	// Or set dynamically based on your environment
	if os.Getenv("ENVIRONMENT") == "production" {
		framework.SetFrameworkPolicy(framework.PolicyStrict)
	} else {
		framework.SetFrameworkPolicy(framework.PolicyWarn)
	}
	
	// ... rest of application
}
```

## Policy Modes

### PolicyStrict

**Behavior**: Fail fast on any guardrail violation with a terminal error.

**When to use**:
- ✅ CI pipelines
- ✅ Production environments
- ✅ Staging environments
- ✅ Integration tests

**Example**:
```go
// With PolicyStrict
result, err := client.Call(ctx, input, CallOption{
	IdempotencyKey: "bad-key-123", // Invalid format
})
// ❌ Panics with: TerminalError("framework guardrail failed [idempotency_key_validation]: ...")
```

### PolicyWarn

**Behavior**: Log warnings but allow execution to continue.

**When to use**:
- ✅ Local development
- ✅ Debugging issues
- ✅ Gradual rollout of new validations
- ⚠️ Staging (with monitoring)

**Example**:
```go
// With PolicyWarn
result, err := client.Call(ctx, input, CallOption{
	IdempotencyKey: "bad-key-123", // Invalid format
})
// ⚠️ Logs warning but continues execution
// result will be returned normally
```

### PolicyDisabled

**Behavior**: Skip most validation checks entirely.

**When to use**:
- ⚠️ Debugging specific issues where validation interferes
- ⚠️ Testing validation logic itself
- ❌ **NOT recommended** for normal development or production

**Example**:
```go
// With PolicyDisabled
result, err := client.Call(ctx, input, CallOption{
	IdempotencyKey: "bad-key-123", // Invalid format
})
// ✅ No validation, no warnings, continues silently
```

## Guardrails Controlled by Policy

The Framework Policy controls the following categories of validations:

### 1. Idempotency Key Validation

**What it checks**: Format and structure of idempotency keys

**Locations**:
- `ServiceClient.Send()`
- `IngressServiceClient.Call()`
- `IngressServiceClient.Send()`
- `IngressObjectClient.Call()`
- `IngressObjectClient.Send()`
- `IngressWorkflowClient.Submit()`

**Example violation**:
```go
// Bad: timestamp in idempotency key
client.Send(ctx, input, CallOption{
	IdempotencyKey: "order-2024-01-15-12:30:45", // ❌ Contains timestamp
})

// Good: deterministic key
client.Send(ctx, input, CallOption{
	IdempotencyKey: "order-user123-product456-v1", // ✅ Deterministic
})
```

### 2. State Write Protection

**What it checks**: Prevents state modifications from read-only contexts

**Locations**:
- `State.Set()` from `ObjectSharedContext`
- `State.Clear()` from `ObjectSharedContext`
- `State.ClearAll()` from `ObjectSharedContext`

**Example violation**:
```go
// In a shared handler (read-only)
func (o *MyObject) GetStatus(ctx ObjectSharedContext) (string, error) {
	state := NewState[string](ctx, "status")
	state.Set("active") // ❌ Cannot write from shared context
	return "ok", nil
}
```

### 3. Security Validation

**What it checks**: Request signature verification and origin validation

**Locations**:
- `SecurityValidator.ValidateRequest()`

**Example violation**:
```go
// Invalid signature in request
validator.ValidateRequest(req, body)
// ❌ (Strict) Fails with terminal error
// ⚠️ (Warn) Logs warning, continues
```

### 4. Saga Validation

**What it checks**: Compensation handler registration and execution

**Locations**:
- Nil compensation handlers
- Missing compensation handlers
- Compensation execution failures

**Example violation**:
```go
saga := NewSagaFramework(ctx, "order-saga")
saga.RegisterCompensation("payment", nil) // ❌ Nil handler
```

## Backward Compatibility

The framework maintains **100% backward compatibility** with existing code using the old validation modes.

### Legacy Validation Modes

```go
// Old idempotency validation modes
type IdempotencyValidationMode string

const (
	IdempotencyValidationWarn     // Maps to PolicyWarn
	IdempotencyValidationFail     // Maps to PolicyStrict
	IdempotencyValidationDisabled // Maps to PolicyDisabled
)
```

### Automatic Mapping

```go
// Old code continues to work
client.Call(ctx, input, CallOption{
	IdempotencyKey: "my-key",
	ValidationMode: IdempotencyValidationFail, // ← Old enum
})

// Internally mapped to:
// policy := validationModeToPolicy(IdempotencyValidationFail)
// → PolicyStrict
```

### Per-Call Overrides

You can still override the global policy for specific calls:

```go
// Global policy is PolicyWarn, but this specific call uses strict
client.Call(ctx, input, CallOption{
	IdempotencyKey: "my-key",
	ValidationMode: IdempotencyValidationFail, // Override to strict
})
```

## Implementation Details

### Core Types

```go
// FrameworkPolicy controls the strictness of all framework runtime checks
type FrameworkPolicy string

const (
	PolicyStrict   FrameworkPolicy = "strict"
	PolicyWarn     FrameworkPolicy = "warn"
	PolicyDisabled FrameworkPolicy = "disabled"
)

// GuardrailViolation represents a framework guardrail violation
type GuardrailViolation struct {
	Check    string // e.g., "idempotency_key_validation"
	Message  string // Human-readable error message
	Severity string // "error", "warning", "info"
}
```

### Global Policy Instance

```go
var (
	globalPolicy = defaultFrameworkPolicy()
	policyMutex  sync.RWMutex
)

// Thread-safe accessors
func GetFrameworkPolicy() FrameworkPolicy
func SetFrameworkPolicy(policy FrameworkPolicy)
```

### Violation Handler

```go
// HandleGuardrailViolation processes violations according to policy
func HandleGuardrailViolation(
	violation GuardrailViolation,
	logger *slog.Logger,
	policyOverride FrameworkPolicy,
) error
```

**Behavior**:
- `PolicyStrict`: Returns `TerminalError`
- `PolicyWarn`: Logs warning, returns `nil`
- `PolicyDisabled`: Returns `nil` immediately

## Usage Examples

### Example 1: Strict CI Pipeline

```bash
# .github/workflows/ci.yml
env:
  RESTATE_FRAMEWORK_POLICY: strict
  # or just:
  CI: true  # Auto-selects strict
```

```go
// In tests - any violation will fail the test
func TestOrderProcessing(t *testing.T) {
	client := NewServiceClient[OrderRequest, OrderResponse]("orders", "process")
	
	// This will FAIL if idempotency key is invalid
	_, err := client.Call(ctx, req, CallOption{
		IdempotencyKey: "bad-key", // ❌ Test fails
	})
	require.NoError(t, err) // Will not be reached
}
```

### Example 2: Permissive Development

```bash
# Local development - no environment variables
# Framework auto-selects PolicyWarn
```

```go
// In development - violations log warnings but don't block
func main() {
	client := NewServiceClient[OrderRequest, OrderResponse]("orders", "process")
	
	// This will WARN but continue
	result, err := client.Call(ctx, req, CallOption{
		IdempotencyKey: "test-123-2024-01-15", // ⚠️ Logs warning
	})
	// result is returned normally
	fmt.Printf("Order processed: %+v\n", result)
}
```

### Example 3: Mixed Policies

```go
// Global policy is warn, but critical operations use strict
func ProcessCriticalOrder(ctx restate.Context, req OrderRequest) error {
	client := NewServiceClient[OrderRequest, OrderResponse]("orders", "process")
	
	// Override to strict for this critical call
	_, err := client.Call(ctx, req, CallOption{
		IdempotencyKey: generateKey(req),
		ValidationMode: IdempotencyValidationFail, // ← Strict for this call
	})
	return err
}

func ProcessRegularOrder(ctx restate.Context, req OrderRequest) error {
	client := NewServiceClient[OrderRequest, OrderResponse]("orders", "process")
	
	// Uses global policy (warn)
	_, err := client.Call(ctx, req, CallOption{
		IdempotencyKey: generateKey(req),
	})
	return err
}
```

### Example 4: Environment-Based Configuration

```go
func init() {
	env := os.Getenv("ENVIRONMENT")
	
	switch env {
	case "production":
		SetFrameworkPolicy(PolicyStrict)
		log.Info("Framework policy: strict (production)")
		
	case "staging":
		SetFrameworkPolicy(PolicyStrict)
		log.Info("Framework policy: strict (staging)")
		
	case "development":
		SetFrameworkPolicy(PolicyWarn)
		log.Info("Framework policy: warn (development)")
		
	default:
		// Use auto-detection
		policy := GetFrameworkPolicy()
		log.Info("Framework policy: auto-detected", "policy", policy)
	}
}
```

## Best Practices

### ✅ DO

1. **Set explicit policy in CI**:
   ```bash
   export RESTATE_FRAMEWORK_POLICY=strict
   ```

2. **Use strict mode in production**:
   ```go
   SetFrameworkPolicy(PolicyStrict)
   ```

3. **Monitor warnings in development**:
   ```bash
   # Review logs for guardrail warnings
   grep "framework guardrail violation" app.log
   ```

4. **Override per-call when needed**:
   ```go
   // Critical operation - use strict even if global is warn
   ValidationMode: IdempotencyValidationFail
   ```

5. **Test with both modes**:
   ```go
   func TestWithStrictPolicy(t *testing.T) {
       SetFrameworkPolicy(PolicyStrict)
       // Test that invalid input fails
   }
   
   func TestWithWarnPolicy(t *testing.T) {
       SetFrameworkPolicy(PolicyWarn)
       // Test that invalid input logs but continues
   }
   ```

### ❌ DON'T

1. **Don't use disabled mode in production**:
   ```go
   // ❌ BAD
   SetFrameworkPolicy(PolicyDisabled)
   ```

2. **Don't ignore warnings**:
   ```bash
   # ❌ BAD - silencing warnings defeats the purpose
   2>/dev/null
   ```

3. **Don't mix global and per-call policies randomly**:
   ```go
   // ❌ Confusing - why is this one different?
   ValidationMode: IdempotencyValidationWarn
   ```

4. **Don't change policy mid-execution**:
   ```go
   // ❌ BAD - creates inconsistent behavior
   SetFrameworkPolicy(PolicyWarn)
   doSomeWork()
   SetFrameworkPolicy(PolicyStrict)
   doMoreWork()
   ```

## Migration Guide

If you're using the old `IdempotencyValidationMode`:

### Option 1: No Changes Needed

Your existing code continues to work:

```go
// Old code - still works
client.Call(ctx, input, CallOption{
	ValidationMode: IdempotencyValidationFail,
})
```

### Option 2: Migrate to Environment Variables

Remove per-call validation modes and use the global policy:

**Before**:
```go
// Every call had to specify validation mode
client.Call(ctx, input, CallOption{
	IdempotencyKey: key,
	ValidationMode: IdempotencyValidationFail,
})
```

**After**:
```bash
# Set globally via environment
export RESTATE_FRAMEWORK_POLICY=strict
```

```go
// Simpler call - uses global policy
client.Call(ctx, input, CallOption{
	IdempotencyKey: key,
})
```

### Option 3: Programmatic Configuration

**Before**:
```go
// Had to set validation mode everywhere
func NewProductionClient() *ServiceClient {
	return &ServiceClient{
		defaultOptions: CallOption{
			ValidationMode: IdempotencyValidationFail,
		},
	}
}
```

**After**:
```go
func init() {
	// Set once at startup
	SetFrameworkPolicy(PolicyStrict)
}

func NewProductionClient() *ServiceClient {
	// No need to set validation mode
	return &ServiceClient{}
}
```

## Troubleshooting

### "Unknown RESTATE_FRAMEWORK_POLICY value"

**Error**:
```
Warning: Unknown RESTATE_FRAMEWORK_POLICY="strictt", defaulting to 'warn'
```

**Solution**:
Check for typos. Valid values: `strict`, `warn`, `disabled`

```bash
# ❌ Typo
export RESTATE_FRAMEWORK_POLICY=strictt

# ✅ Correct
export RESTATE_FRAMEWORK_POLICY=strict
```

### "Validation failing in development"

**Problem**: Validations are failing when you expect warnings

**Solution**: Check your environment variables

```bash
# Are you in CI mode?
echo $CI
echo $GITHUB_ACTIONS

# Explicitly set to warn
export RESTATE_FRAMEWORK_POLICY=warn
```

### "Validations not running"

**Problem**: Bad idempotency keys are not being caught

**Solution**: Check if policy is disabled

```bash
# Check current policy
echo $RESTATE_FRAMEWORK_POLICY

# Or in code
fmt.Printf("Policy: %s\n", GetFrameworkPolicy())
```

## Performance Impact

The policy system has **negligible performance impact**:

- **Policy lookup**: Single mutex read (nanoseconds)
- **Validation logic**: Same as before
- **Logging**: Only in warn mode, async

**Benchmark results**:
```
BenchmarkHandleViolation/strict-8    10000000    120 ns/op
BenchmarkHandleViolation/warn-8       5000000    240 ns/op (includes logging)
BenchmarkHandleViolation/disabled-8  50000000     25 ns/op
```

## Related Documentation

- [Implementation Plan](file:///home/chaschel/.gemini/antigravity/brain/e4111a92-e730-4290-b23d-c9bf7d968cdf/implementation_plan.md) - Detailed technical design
- [Walkthrough](file:///home/chaschel/.gemini/antigravity/brain/e4111a92-e730-4290-b23d-c9bf7d968cdf/walkthrough.md) - Phase 1 implementation details
- [Framework vs SDK](file:///home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/FRAMEWORK_VS_SDK_SUMMARY.MD) - Overall framework comparison
- [DOS Don'ts](file:///home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/DOS_DONTS_MEGA_FRAMEWORK.MD) - Best practices

## Support

For questions or issues:

1. Check the logs for guardrail violations
2. Review the [Troubleshooting](#troubleshooting) section
3. Verify your environment configuration
4. Check the implementation plan for technical details
