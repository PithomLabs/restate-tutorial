# PROMPTS_REA.MD - Effective Prompt Engineering for Rea Framework

> **Purpose**: Comprehensive prompt templates for LLMs and developers to build resilient distributed applications using the `pithomlabs/rea` framework (package `framework`).

**Target Audience**: Developers, AI Coding Assistants (Cursor, Claude, GitHub Copilot), Code Review Tools

---

## Table of Contents

1. [Quick Reference](#quick-reference)
2. [Code Review Prompts](#code-review-prompts)
3. [Implementation Prompts](#implementation-prompts)
4. [Debugging Prompts](#debugging-prompts)
5. [Architecture & Design Prompts](#architecture--design-prompts)
6. [Testing & Verification Prompts](#testing--verification-prompts)
7. [Context Engineering for LLMs](#context-engineering-for-llms)

---

## Quick Reference

### Critical Rules (Copy-Paste to AI Agent Context)

```markdown
## Rea Framework Core Principles
1. ALL non-deterministic operations MUST be wrapped in `framework.RunDo()` or `framework.SafeRun()`
2. NEVER use `restate.Run()` directly - use framework wrappers for safety
3. Use `framework.NewTime(ctx).Now()` for deterministic time
4. Use `framework.DeterministicHelpers(ctx).UUID()` for IDs
5. Use `framework.ServiceClient` for internal calls, NOT ingress client
6. Use `framework.NewMutableObjectState` for state writes (Exclusive Context only)
7. Use `framework.NewReadOnlyObjectState` for state reads
8. Use `framework.WaitForExternalSignal` for Awakeables
9. Use `framework.GetInternalSignal` for Workflow Promises

## Service Type Selection
- **Basic Service**: Stateless logic, API calls, orchestration
- **Virtual Object**: Stateful entities (Single Writer)
- **Workflow**: Multi-step processes (Exactly-Once)

## Security & Policy
- Configure `framework.SetFrameworkPolicy(framework.PolicyStrict)`
- Use `framework.SecurityMiddleware` for ingress
- Validate idempotency keys with `framework.CallOption`
```

---

## Code Review Prompts

### Prompt #1: Framework Compliance Check

```
Review the following Restate handler code for Rea framework compliance:

[CODE HERE]

Check for:
1. Direct use of `restate.Run` (should be `framework.RunDo` or `framework.SafeRun`)
2. Direct use of `time.Now()` (should be `framework.NewTime(ctx).Now()`)
3. Direct use of `restate.UUID` (should be via `framework.DeterministicHelpers`)
4. State access without framework wrappers (`NewMutableObjectState` etc.)
5. Ingress client usage inside handlers (should be `framework.ServiceClient`)

For each violation, provide:
- Line number and code snippet
- Why it violates framework policy
- Correct `framework.*` replacement
```

### Prompt #2: Concurrency Safety Audit

```
Analyze this Rea framework code for concurrency anti-patterns:

[CODE HERE]

Identify:
1. Virtual Object handlers using `restate.Sleep` (should use delayed sends)
2. Request-response calls between exclusive handlers (deadlock risk)
3. Use of native Go concurrency instead of `framework.FanOutFail` or `framework.Gather`
4. Blocking operations in exclusive handlers

Provide fixes using:
- `framework.FanOutFail` for parallel tasks
- `framework.RacePromiseWithTimeout` for timeouts
- Delayed messages via `framework.CallOption{Delay: ...}`
```

### Prompt #3: State Management Review

```
Evaluate state management patterns in this code:

[CODE HERE]

Check:
1. Is `framework.NewMutableObjectState` used only in Exclusive Contexts?
2. Is `framework.NewReadOnlyObjectState` used for reads?
3. Is long-lived state cleared with `framework.ClearAll(ctx)`?
4. Are state keys defined as constants?

Suggest improvements for:
- Lazy vs eager state loading (via framework options)
- Proper use of `state.Clear()` vs `framework.ClearAll()`
```

### Prompt #4: Error Handling Pattern Check

```
Review error handling in this service:

[CODE HERE]

Verify:
1. `framework.NewTerminalError` used for business failures
2. Retryable errors propagated correctly
3. Saga compensation using `framework.SafeStep` or manual defer
4. All compensations are idempotent

Check for:
- Bare `return err` without wrapping
- Missing compensation registration
```

---

## Implementation Prompts

### Prompt #5: Implement Basic Service with Rea

```
Create a Restate Basic Service in Go using the rea framework for:

[REQUIREMENT: e.g., "process payment transactions"]

Requirements:
1. Service name: [NAME]
2. Handler: [HANDLER_NAME]
3. Input/Output types

Implementation checklist:
- [ ] Wrap external calls in `framework.RunDo`
- [ ] Use `ctx.Log()` for logging
- [ ] Implement timeout handling with `framework.RacePromiseWithTimeout`
- [ ] Use `framework.DeterministicHelpers` for IDs
- [ ] Use `framework.ServiceClient` for downstream calls

Example pattern:
```go
func (s *Service) Handler(ctx restate.Context, req Input) (Output, error) {
    // Deterministic ID
    id := framework.NewDeterministicHelpers(ctx).UUID()
    
    // Side effect
    result, err := framework.RunDo(ctx, func(rc restate.RunContext) (string, error) {
        return externalCall()
    })
    
    // Internal call
    client := framework.ServiceClient[Req, Resp]{ServiceName: "OtherSvc"}
    resp, err := client.Request(ctx, Req{...})
}
```

### Prompt #6: Implement Virtual Object with Rea

```
Implement a Virtual Object using rea framework for:

[REQUIREMENT: e.g., "shopping cart"]

Specifications:
- Object name: [NAME]
- Key: [KEY_TYPE]
- State: [STATE_STRUCT]

Ensure:
1. Use `framework.NewMutableObjectState` for writes
2. Use `framework.NewReadOnlyObjectState` for reads
3. NO `restate.Sleep` (use delayed sends)
4. Explicit state clearing

Example pattern:
```go
func (o *Object) AddItem(ctx restate.ObjectContext, item Item) error {
    state := framework.NewMutableObjectState[Cart](ctx, "cart")
    cart, _ := state.Get()
    cart.Items = append(cart.Items, item)
    return state.Set(cart)
}
```

### Prompt #7: Implement Workflow with Saga (Rea)

```
Create a Workflow using rea framework for:

[REQUIREMENT]

Requirements:
- Workflow ID generation
- Saga compensation steps

Implementation pattern:
```go
func (w *Workflow) Run(ctx restate.WorkflowContext, req Request) error {
    // Register compensation
    defer func() {
        // Manual compensation logic or use framework helper if available
    }()
    
    // Step 1
    err := framework.RunDo(ctx, func(rc restate.RunContext) (any, error) {
        // Action
    })
}
```

### Prompt #8: Implement Awakeable Pattern (Rea)

```
Implement awakeable pattern using rea framework:

[SCENARIO]

Implementation requirements:
1. Use `framework.WaitForExternalSignal[T](ctx)`
2. Send ID to external system via `framework.RunDo`
3. Wait for result
4. Handle timeout

Example:
```go
awakeable := framework.WaitForExternalSignal[string](ctx)
id := awakeable.Id()

// Send to external system
framework.RunDo(ctx, func(rc restate.RunContext) (any, error) {
    return sendToExternal(id)
})

// Wait
result, err := awakeable.Result()
```

### Prompt #9: Implement Fan-Out/Fan-In (Rea)

```
Implement parallel task execution using rea framework:

[SCENARIO]

Requirements:
- Use `framework.FanOutFail` or `framework.Gather`
- Handle errors

Example:
```go
ops := []func() (Result, error){
    func() (Result, error) { return call1() },
    func() (Result, error) { return call2() },
}
results, err := framework.FanOutFail(ctx, ops)
```

---

## Debugging Prompts

### Prompt #10: Debug Non-Determinism (Rea)

```
I'm getting a non-determinism error (RT0004).

[CODE]

Debug checklist:
1. Is `restate.Run` used directly? (Switch to `framework.RunDo`)
2. Is `time.Now()` used? (Switch to `framework.NewTime(ctx).Now()`)
3. Is `restate.UUID` used directly? (Switch to `framework.DeterministicHelpers`)
4. Are map iterations deterministic?

Provide fix using framework helpers.
```

### Prompt #11: Debug Deadlock (Rea)

```
Virtual Object deadlock detected.

[CODE]

Check for:
1. Cross-call between exclusive handlers
2. Blocking wait (`restate.Sleep`) in exclusive handler

Fix:
- Use `framework.CallOption{Delay: ...}` for delayed self-calls
- Use `framework.NewReadOnlyObjectState` and shared handlers for reads
```

---

## Architecture & Design Prompts

### Prompt #12: Service Type Selection (Rea)

```
I need to implement: [REQUIREMENT]

Help me choose the service type and rea framework patterns.

Provide:
- Service Type (Basic/Object/Workflow)
- State strategy (`NewMutableObjectState` vs `NewReadOnlyObjectState`)
- Concurrency pattern (`FanOutFail`, `Gather`)
- External integration (`WaitForExternalSignal`)
```

### Prompt #13: Design External Integration (Rea)

```
Design external integration for: [SYSTEM]

Requirements:
- Ingress security
- Idempotency

Use:
- `framework.NewIngressClient`
- `framework.SecurityMiddleware`
- `framework.CallOption{IdempotencyKey: ...}`

Provide architecture and code snippets.
```

---

## Testing & Verification Prompts

### Prompt #14: Generate Unit Tests (Rea)

```
Generate unit tests for this rea framework handler:

[CODE]

Use `github.com/restatedev/sdk-go/testing` but mock framework helpers where necessary.
Test for:
- Happy path
- Terminal errors (`framework.NewTerminalError`)
- State transitions
```

---

## Context Engineering for LLMs

### Effective Context Patterns

#### Pattern #1: System Prompt for Rea Framework

```markdown
You are an expert Restate developer using the `pithomlabs/rea` framework (package `framework`).

CRITICAL RULES:
1. ALWAYS use `framework.RunDo` or `framework.SafeRun` for side effects.
2. ALWAYS use `framework.NewTime(ctx)` for time.
3. ALWAYS use `framework.DeterministicHelpers(ctx)` for UUIDs.
4. ALWAYS use `framework.ServiceClient` for internal calls.
5. NEVER use `restate.Run` directly.
6. NEVER use `time.Now()` directly.

When implementing:
- Import `github.com/pithomlabs/rea` (usually aliased as `framework` or used as `rea` depending on package name).
- Use framework wrappers for all Restate primitives.
```

#### Pattern #2: File-Specific Instructions

```markdown
For SERVICE files:
- Import `github.com/pithomlabs/rea`
- Use `framework.RunDo` for I/O
- Use `framework.NewMutableObjectState` for state

For INGRESS files:
- Use `framework.NewIngressClient`
- Use `framework.SecurityMiddleware`
```

---

## Quick Command Reference

### Essential Rea Framework Snippets

```go
// Side Effect
res, err := framework.RunDo(ctx, func(rc restate.RunContext) (string, error) {
    return externalCall()
})

// State Write
state := framework.NewMutableObjectState[MyType](ctx, "key")
state.Set(newValue)

// Internal Call
client := framework.ServiceClient[Req, Resp]{ServiceName: "Svc"}
resp, err := client.Request(ctx, req)

// External Signal (Awakeable)
sig := framework.WaitForExternalSignal[string](ctx)
```

---

## Best Practices Summary

### Golden Rules for LLM Responses

1. **Cite DOS_DONTS_REA.MD patterns**
2. **Use framework helpers** exclusively
3. **Avoid raw Restate SDK calls** where framework wrappers exist
4. **Implement security** (middleware, policies)

### Response Format Template

```markdown
## Analysis
[Problem description]

## Framework Compliance Issues
1. ❌ **Issue**: Direct `restate.Run` usage
   - Fix: Use `framework.RunDo`
   
## Recommended Fix
✅ **Solution**:

```go
// Correct implementation using rea framework
[CODE]
```
```
