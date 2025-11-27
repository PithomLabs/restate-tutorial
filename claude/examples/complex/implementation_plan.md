# Rewrite run.go.txt Using the Rea Framework

## Overview

The task is to rewrite `/home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/examples/complex/run.go.txt` as `run.go` using the `github.com/pithomlabs/rea` framework instead of directly using the Restate Go SDK.

The current `run.go.txt` file (728 lines) is a comprehensive example demonstrating:
- Framework block with `WaitForExternalSignal`, `State`, and `SagaFramework`
- Multiple service types (basic services, workflows)
- Saga-based transaction compensation
- Restate operations (`Run`, `RunAsync`, `Sleep`, promises, awakeables, etc.)
- Service communication patterns

## Key Transformations

### 1. Import Changes

**Current:**
```go
import (
    restate "github.com/restatedev/sdk-go"
    "github.com/restatedev/sdk-go/server"
)
```

**New:**
```go
import (
    restate "github.com/restatedev/sdk-go"  // Keep for base types
    "github.com/restatedev/sdk-go/server"
    rea "github.com/pithomlabs/rea"        // Add rea framework
)
```

### 2. Remove Inline Framework Block

The `run.go.txt` file includes an inline framework implementation (lines 27-354) that replicates functionality now available in the rea package. This entire block will be **removed** as all functionality is provided by `github.com/pithomlabs/rea`.

Functions to be removed and replaced:
- `WaitForExternalSignal` → use `rea.WaitForExternalSignal`
- `ResolveExternalSignal` → use `rea.ResolveExternalSignal`
- `RejectExternalSignal` → use `rea.RejectExternalSignal`
- `GetInternalSignal` → use `rea.GetInternalSignal`
- `State` type → use `rea.State` or `rea.NewState`
- `SagaFramework` → use `rea.NewSaga`
- Helper functions (`computeBackoff`, `deterministicStepID`, `canonicalJSON`, `removeIndex`) → provided by rea internally

### 3. State Management Updates

**Current (inline):**
```go
state := NewState[string](ctx, "my-key")
value, err := state.Get()
state.Set("new-value")
```

**New (using rea):**
```go  
state := rea.NewState[string](ctx, "my-key")
value, err := state.Get()
state.Set("new-value")
```

### 4. Saga Framework Updates

**Current (inline):**
```go
saga := NewSaga(ctx, "order-saga", nil)
defer saga.CompensateIfNeeded(&err)
saga.Register("refund-payment", compensationFunc)
saga.Add("refund-payment", payload, true)
```

**New (using rea):**
```go
saga := rea.NewSaga(ctx, "order-saga", nil)
defer saga.CompensateIfNeeded(&err)
saga.Register("refund-payment", compensationFunc)
saga.Add("refund-payment", payload, true)
```

### 5. Run Helpers (Optional Enhancement)

The rea framework provides `RunDo` and `RunDoVoid` helpers to prevent accidental context capture. While optional, using these enhances code safety:

**Current:**
```go
result, err := restate.Run(ctx, func(rc restate.RunContext) (string, error) {
    return callExternalAPI(), nil
})
```

**New (with rea helpers):**
```go
result, err := rea.RunDo(ctx, func(rc restate.RunContext) (string, error) {
    return callExternalAPI(), nil
})
```

### 6. Service Clients (Optional Enhancement)

The rea framework provides type-safe service client wrappers:

**Current:**
```go
client := restate.Service[bool](ctx, "InventoryService", "CheckAvailability")
available, err := client.Request(req.Item)
```

**New (with rea clients - optional):**
```go
client := rea.ServiceClient[string, bool]{
    ServiceName: "InventoryService",
    HandlerName: "CheckAvailability",
}
available, err := client.Call(ctx, req.Item)
```

## Proposed Changes

### Modified File

#### `/home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/examples/complex/run.go`

**Changes:**
1. Add import for `rea "github.com/pithomlabs/rea"`
2. Remove the entire framework block (lines 27-354 in original)
3. Replace all uses of inline framework functions with rea equivalents:
   - `WaitForExternalSignal` → `rea.WaitForExternalSignal`
   - `ResolveExternalSignal` → `rea.ResolveExternalSignal`  
   - `GetInternalSignal` → `rea.GetInternalSignal`
   - `NewSaga` → `rea.NewSaga`
   - `NewState` → `rea.NewState`
4. Update saga compensation calls to use `rea.RunDoVoid` for better safety
5. Keep all service implementations unchanged (they only depend on Restate SDK types)

## Verification Plan

### Manual Verification

Since this is a self-contained example file demonstrating framework patterns, verification will be manual:

1. **Build Check:**
   ```bash
   cd /home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/examples/complex
   go mod tidy
   go build run.go
   ```
   Expected: Clean build with no errors

2. **Import Resolution:**
   Verify that the rea package can be imported:
   ```bash
   go list -m github.com/pithomlabs/rea
   ```
   Expected: Module found and version displayed

3. **Code Review:**
   - Verify all framework block functions have been removed
   - Verify all calls to framework functions now use `rea.` prefix
   - Verify `NewSaga`, `NewState` usage is correct
   - Verify no breaking changes to service handler signatures

4. **Optional Runtime Testing:**
   If the user has a Restate server running, the example can be started:
   ```bash
   # Start the service
   go run run.go
   
   # In another terminal, register with Restate
   curl http://localhost:8080/deployments -H 'Content-Type: application/json' \
     -d '{"uri": "http://localhost:2223"}'
   ```
   Expected: Service starts successfully on port 2223

### Automated Tests

No automated tests exist for this example file. It's a demonstration/tutorial file showing framework usage patterns.


