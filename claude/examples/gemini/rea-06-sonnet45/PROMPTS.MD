# PROMPTS.MD - Effective Prompt Engineering for Restate & Rea Framework Development

> **Purpose**: Comprehensive prompt templates for LLMs and developers to build resilient distributed applications using Restate Go SDK and the `pithomlabs/rea` framework.

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
## Restate Core Principles
1. ALL non-deterministic operations MUST be wrapped in `restate.Run()`
2. NEVER use Go native concurrency (goroutines/channels) with Restate futures
3. ALWAYS use `restate.Wait()` or `restate.WaitFirst()` for concurrent operations
4. Use `ctx.Log()` not standard logging packages
5. Use `restate.UUID(ctx)` for deterministic IDs, NEVER for cryptography
6. Throw `restate.TerminalError()` for business logic failures
7. Use `restate.Sleep()` for durable timers, NEVER `time.Sleep()`

## Service Type Selection
- **Basic Service**: Stateless logic, API calls, orchestration
- **Virtual Object**: Stateful entities with single-writer consistency
- **Workflow**: Multi-step processes with exactly-once execution

## Communication Rules
- **Awakeables**: External systems (payment gateways, webhooks)
- **Durable Promises**: Internal workflow signaling ONLY
- **Never** call External HTTP directly - wrap in `restate.Run()`
```

---

## Code Review Prompts

### Prompt #1: Determinism Violation Check

```
Review the following Restate handler code for determinism violations:

[CODE HERE]

Check for:
1. Any external API calls not wrapped in `restate.Run()`
2. Use of `time.Now()`, `rand.Float64()`, or non-deterministic UUID generation
3. Iteration over maps with Context operations inside loops
4. Use of goroutines or channels with Restate futures
5. Context usage inside `restate.Run()` blocks

For each violation, provide:
- Line number and code snippet
- Why it violates determinism
- Correct implementation example
```

### Prompt #2: Concurrency Safety Audit

```
Analyze this Restate code for concurrency anti-patterns:

[CODE HERE]

Identify:
1. Virtual Object handlers using `ctx.Sleep()` or `Awakeable.Result()`
2. Request-response calls between exclusive handlers (deadlock risk)
3. Mutually calling agents in same Virtual Object key
4. Use of native Go concurrency instead of `restate.Wait()`
5. Blocking operations in exclusive handlers

Provide fixes using:
- `restate.ObjectSend()` with delays instead of sleep
- `restate.WaitFirst()` for timeouts
- Proper separation of exclusive vs shared handlers
```

### Prompt #3: State Management Review

```
Evaluate state management patterns in this code:

[CODE HERE]

Check:
1. Are state operations only in exclusive contexts (ObjectContext/WorkflowContext)?
2. Is long-lived Virtual Object state explicitly cleared when done?
3. Are shared handlers (ObjectSharedContext) only reading state?
4. Is Workflow state retention configured properly?
5. Are state access patterns deterministic?

Suggest improvements for:
- Lazy vs eager state loading strategy
- State key naming conventions
- Proper use of `restate.Clear()` and `restate.ClearAll()`
```

### Prompt #4: Error Handling Pattern Check

```
Review error handling in this Restate service:

[CODE HERE]

Verify:
1. Terminal errors used for permanent failures (invalid input, business logic)
2. Retryable errors propagated for transient failures
3. Saga compensation logic registered BEFORE actions execute
4. All compensations are idempotent
5. Proper use of `restate.TerminalError()` with appropriate status codes

Check for Python-specific: bare `except:` or `except Exception:` usage
```

---

## Implementation Prompts

### Prompt #5: Implement Basic Service

```
Create a Restate Basic Service in Go using the rea framework for:

[REQUIREMENT: e.g., "process payment transactions from multiple providers"]

Requirements:
1. Service name: [NAME]
2. Handler: [HANDLER_NAME]
3. Input type: [INPUT_STRUCT]
4. Output type: [OUTPUT_STRUCT]
5. External integrations: [LIST]

Implementation checklist:
- [ ] Wrap all external API calls in `restate.Run()`
- [ ] Use `ctx.Log()` for logging
- [ ] Implement timeout handling with `restate.WaitFirst()`
- [ ] Add Terminal errors for business failures
- [ ] Use deterministic IDs via `restate.UUID(ctx)`
- [ ] Include error context in all error returns
```

### Prompt #6: Implement Virtual Object

```
Implement a Virtual Object for:

[REQUIREMENT: e.g., "shopping cart with inventory reservation"]

Specifications:
- Object name: [NAME]
- Key: [USER_ID/SESSION_ID/etc]
- State: [DESCRIBE STATE STRUCTURE]
- Exclusive handlers: [LIST]
- Shared handlers: [LIST]

Ensure:
1. Exclusive handlers use `restate.ObjectContext`
2. Shared handlers use `restate.ObjectSharedContext` (read-only)
3. NO `ctx.Sleep()` in exclusive handlers (use delayed sends)
4. State updates journaled with execution
5. Explicit state clearing when session ends
6. Deadlock prevention (no self-calls on same key)
```

### Prompt #7: Implement Workflow with Saga

```
Create a Workflow for:

[REQUIREMENT: e.g., "order fulfillment with payment and shipping"]

Steps:
1. [STEP 1 description]
2. [STEP 2 description]
3. [STEP 3 description]

Requirements:
- Workflow ID: [HOW TO GENERATE]
- Human approval required: [YES/NO]
- Durable timers needed: [DESCRIBE]
- Saga compensation for: [LIST STEPS]

Implementation pattern:
```go
type MyWorkflow struct{}

func (MyWorkflow) Run(ctx restate.WorkflowContext, req MyRequest) error {
    // Pattern: Register compensations BEFORE actions
    var compensations []func() error
    defer func() {
        if err != nil {
            for _, comp := range slices.Backward(compensations) {
                comp() // Execute in reverse
            }
        }
    }()
    
    // Step 1: Action + Compensation registration
    // Step 2: Action + Compensation registration
    // etc.
}
```

### Prompt #8: Implement Awakeable Pattern

```
Implement awakeable pattern for external integration:

[SCENARIO: e.g., "wait for payment gateway callback"]

Setup:
- External system: [PAYMENT_GATEWAY/WEBHOOK/etc]
- Callback endpoint: [DESCRIBE]
- Timeout: [DURATION]

Implementation requirements:
1. Create awakeable in handler
2. Send awakeable ID to external system via `restate.Run()`
3. Wait for result with timeout using `restate.WaitFirst()`
4. Handle both success and failure cases
5. Create ingress endpoint to resolve awakeable
6. Validate awakeable ID in callback

IMPORTANT: 
- Use Awakeable for external systems
- Use Promise for workflow internal signaling
- Always implement timeout handling
```

### Prompt #9: Implement Fan-Out/Fan-In Pattern

```
Implement parallel task execution using rea framework:

[SCENARIO: e.g., "fetch user data from multiple microservices"]

Tasks to parallelize:
1. [TASK 1]
2. [TASK 2]
3. [TASK 3]

Requirements:
- Use `restate.RequestFuture()` for non-blocking calls
- Collect results with `restate.Wait()` for all tasks
- OR use `restate.WaitFirst()` for first completion
- Error handling strategy: [FAIL_FAST / COLLECT_ALL]

Example using rea framework:
```go
// Fan-out
future1 := restate.Service[Result1](ctx, "Svc1", "Handler1").RequestFuture(req1)
future2 := restate.Service[Result2](ctx, "Svc2", "Handler2").RequestFuture(req2)

// Fan-in
for fut, err := range restate.Wait(ctx, future1, future2) {
    if err != nil {
        return err // Handle error
    }
    // Process result
}
```

---

## Debugging Prompts

### Prompt #10: Debug Non-Determinism Error

```
I'm getting a non-determinism error (RT0004) in my Restate handler:

[ERROR MESSAGE]
[HANDLER CODE]

Debug checklist:
1. Identify deployment version changes without re-registration
2. Find non-wrapped external calls affecting control flow
3. Look for unrecorded side effects (API calls, DB queries)
4. Check for map iteration with Context operations
5. Verify no code changes in container without new deployment

Provide:
- Root cause analysis
- Exact line causing non-determinism
- Fixed code following Restate patterns
```

### Prompt #11: Debug Journal Mismatch (RT0016)

```
Encountering Journal Mismatch error:

[ERROR DETAILS]
[RELEVANT CODE]

Possible causes:
1. Code updated without new deployment version
2. Handler signature changed
3. Service type changed (Service → Object → Workflow)
4. Non-deterministic branching logic

Investigation steps:
- Check deployment versions in Restate admin
- Review recent code changes
- Verify immutable deployment pattern
- Identify control flow changes

Solution: Deploy as NEW immutable version
```

### Prompt #12: Debug Deadlock in Virtual Objects

```
Virtual Object appears deadlocked:

Object: [NAME]
Key: [KEY]
Last known state: [DESCRIBE]

Analyze for:
1. Cross deadlock (A calls B, B calls A with same key)
2. Cycle deadlock (A → B → C → A)
3. Self-call in exclusive handler
4. Blocking wait in exclusive handler (Sleep/Awakeable)

Provide deadlock diagram and fix:
- Refactor to use delayed sends
- Change to one-way messages
- Split exclusive handlers
- Use object keys strategically
```

### Prompt #13: Debug High Latency/Queuing

```
Experiencing high latency in Virtual Object:

Object: [NAME]
Observed behavior: [DESCRIBE]
Queue depth: [IF KNOWN]

Check for:
1. Long-running exclusive handlers
2. `ctx.Sleep()` in exclusive handlers
3. External API calls in Run blocks without timeouts
4. Excessive state size causing serialization delays

Optimization strategies:
- Move sleeps to delayed sends
- Use shared handlers for reads
- Implement lazy state loading
- Add request-level timeouts
```

---

## Architecture & Design Prompts

### Prompt #14: Service Type Selection Guidance

```
I need to implement:

[FUNCTIONAL REQUIREMENT]

Help me choose the right Restate service type:

Questions to answer:
1. Does it need persistent state? (Yes → Object/Workflow, No → Service)
2. Is state scoped to a unique key? (Yes → Object)
3. Does it require exactly-once execution per ID? (Yes → Workflow)
4. Is it long-running with external waits? (Yes → Workflow)
5. Does it need high concurrency without state? (Yes → Service)

Provide:
- Service type recommendation with rationale
- Context type to use
- State management strategy
- Concurrency model
- Example skeleton code
```

### Prompt #15: Design Saga Compensation

```
Design saga compensation for:

[BUSINESS PROCESS: e.g., "book flight, reserve hotel, charge card"]

Steps involved:
1. [STEP with compensation]
2. [STEP with compensation]
3. [STEP with compensation]

For each step, define:
- Primary action (what to do)
- Compensation action (how to undo)
- Idempotency guarantee
- Failure handling

Output format:
| Step | Primary Action | Compensation | Idempotent? | Terminal Failures |
|------|---------------|--------------|-------------|-------------------|

Implementation pattern with defer and reverse order execution.
```

### Prompt #16: Design Idempotency Strategy

```
Design idempotency strategy for:

[SERVICE/WORKFLOW]

Requirements:
- External callers: [LIST]
- Duplicate risks: [SCENARIOS]
- State deduplication needs: [DESCRIBE]

Choose pattern:
A. Idempotency keys on service calls
B. State-based deduplication (check executed flag)
C. Workflow ID determinism
D. Combination

For chosen pattern:
1. Key generation algorithm (deterministic!)
2. Where to store/check keys
3. Retention period
4. Edge case handling (race conditions)

Provide implementation with rea framework helpers.
```

### Prompt #17: Design External Integration

```
Design external system integration for:

System: [PAYMENT_GATEWAY/CRM/etc]
Integration type: [WEBHOOK/POLLING/API]
Reliability requirements: [DESCRIBE]

Architecture decisions:
1. Use Awakeable or polling? (Awakeable preferred for webhooks)
2. Timeout strategy for external wait
3. Retry policy for external API calls
4. Callback authentication/verification
5. Failure recovery mechanism

Provide:
- Architecture diagram (text/mermaid)
- Handler pseudocode with awakeable pattern
- Ingress callback endpoint design
- Error handling for all scenarios
```

---

## Testing & Verification Prompts

### Prompt #18: Generate Unit Tests

```
Generate unit tests for this Restate handler:

[HANDLER CODE]

Test cases to cover:
1. Happy path execution
2. External API failure (retryable error)
3. Business logic failure (terminal error)
4. State persistence and retrieval
5. Concurrent operations (if applicable)
6. Timeout scenarios
7. Compensation execution (for sagas)

Use Restate test framework:
```go
import "github.com/restatedev/sdk-go/testing"

func TestMyHandler(t *testing.T) {
    ctx := testing.NewTestContext(t)
    // Mock external calls
    // Test deterministic replay
}
```

### Prompt #19: Integration Test Plan

```
Create integration test plan for:

[SERVICE/WORKFLOW]

Test scenarios:
1. End-to-end happy path
2. Failure at each step with recovery
3. Concurrent invocations with idempotency keys
4. Awakeable resolution (if applicable)
5. State consistency after failures
6. Saga compensation triggers

For each scenario:
- Setup steps
- Execution command
- Expected outcome
- Assertion checks
- Cleanup steps

Include curl commands for Restate API interaction.
```

### Prompt #20: Chaos Testing Scenarios

```
Design chaos testing for resilience validation:

[SYSTEM DESCRIPTION]

Chaos scenarios:
1. Kill service mid-execution (test journal replay)
2. Network partition during external API call
3. Slow external API (timeout handling)
4. Concurrent writes to same Virtual Object key
5. Restate runtime restart

For each scenario:
- Injection method (kill -9, network rules, etc)
- Expected system behavior
- Recovery verification
- Data consistency checks

Provide bash scripts for chaos injection.
```

---

## Context Engineering for LLMs

### Effective Context Patterns

#### Pattern #1: Comprehensive System Prompt

```markdown
You are an expert Restate developer using the Go SDK and `pithomlabs/rea` framework.

CRITICAL RULES:
1. ALL external I/O MUST be in `restate.Run()` blocks
2. NEVER use `time.Sleep()` - use `restate.Sleep()`
3. NEVER use goroutines with futures - use `restate.Wait()`
4. Use `ctx.Log()` for logging, not fmt.Printf
5. Throw `restate.TerminalError()` for business failures
6. Generate IDs with `restate.UUID(ctx)` for determinism

ANTI-PATTERNS TO AVOID:
- ❌ Using native Go concurrency
- ❌ Direct external API calls
- ❌ `time.Now()` or `rand.Float64()` in handlers
- ❌ Map iteration with Context operations
- ❌ Context usage inside `restate.Run()`

When implementing:
1. Always check service type  (Service/Object/Workflow)
2. Use correct context type
3. Wrap side effects properly
4. Handle errors with Terminal vs retryable
5. Implement timeouts for external waits
```

#### Pattern #2: File-Specific Instructions

```markdown
When editing Restate handler files:

For SERVICE files (svcs.go, services.go):
- Import: `restate "github.com/restatedev/sdk-go"`
- Context: `restate.Context` for stateless
- External calls: Wrap in `restate.Run()`
- Logging: Use `ctx.Log().Info/Error()`

For INGRESS files (ingress.go, api.go):
- Import: `restateingress "github.com/restatedev/sdk-go/ingress"`
- Client: `restateingress.NewClient(url)`
- Calls: `restateingress.Service[I, O](...).Request()`
- Idempotency: Add keys to header or via WithIdempotencyKey

For MODEL files (types.go, models.go):
- Use JSON tags for serialization
- Keep structs simple and serializable
- Document struct purposes clearly
```

#### Pattern #3: Review Checklist Template

```markdown
Before finalizing Restate code, verify:

DETERMINISM CHECKLIST:
- [ ] All HTTP/DB calls wrapped in `restate.Run()`
- [ ] No `time.Now()`, `rand.*()` outside Run blocks
- [ ] All UUIDs generated via `restate.UUID(ctx)`
- [ ] No map iteration with Context operations
- [ ] Logging uses `ctx.Log()` only

CONCURRENCY CHECKLIST:
- [ ] No goroutines/channels with Restate futures
- [ ] Using `restate.Wait()` for parallel operations
- [ ] Timeouts implemented with `restate.WaitFirst()`
- [ ] No blocking in Virtual Object exclusive handlers

ERROR HANDLING CHECKLIST:
- [ ] Terminal errors for permanent failures
- [ ] Retryable errors for transient issues
- [ ] Saga compensations registered before actions
- [ ] All compensations are idempotent

STATE MANAGEMENT CHECKLIST:
- [ ] Exclusive contexts for state writes
- [ ] Shared contexts for concurrent reads
- [ ] Long-lived state explicitly cleared
- [ ] No state access from Service context
```

---

## Specialized Prompts

### Prompt #21: Migrate from Standard Restate to Rea Framework

```
Migrate this standard Restate Go code to use rea framework:

[ORIGINAL CODE]

Migration steps:
1. Import: Add `"github.com/pithomlabs/rea"`
2. State: Replace `restate.Get/Set` with `framework.NewMutableObjectState`
3. Clients: Use `framework.ServiceClient` for internal calls
4. Run blocks: Use `framework.RunDo` or `SafeRun`
5. Security: Add `framework.SecurityMiddleware` to ingress
6. Helpers: Use `framework.DeterministicHelpers`, `framework.NewTime`

Provide:
- Migrated code with rea framework patterns
- Improvements from framework usage
- Policy configuration recommendations
```

### Prompt #22: Optimize Restate Handler Performance

```
Optimize this Restate handler for performance:

[HANDLER CODE]

Performance analysis:
1. Identify sequential operations that can be parallelized
2. Find excessive state access (suggest lazy loading)
3. Check for blocking operations in exclusive handlers
4. Look for unnecessary journal entries
5. Evaluate external call timeout strategies

Provide optimized version with:
- `RequestFuture()` for parallel calls
- Shared handlers for read-only operations
- Lazy state loading where applicable
- Proper timeout handling
- Reduced critical path latency
```

### Prompt #23: Security Audit for Restate Service

```
Audit security for this Restate service:

[SERVICE CODE]

Check:
1. L1 (Ingress): API key validation at entry point
2. L2 (Identity): Trusted identity propagation (user ID journaled)
3. L3 (Internal): Request signature verification
4. Idempotency key validation (non-temporal patterns)
5. Private services marked with `WithIngressPrivate(true)`

For rea framework:
- Is `framework.SecurityMiddleware` applied?
- Are guardrails configured (PolicyStrict in prod)?
- Are identity headers validated?

Provide security hardening recommendations.
```

### Prompt #24: Document Restate Service API

```
Generate API documentation for:

[SERVICE CODE]

Include for each handler:
- Handler name and purpose
- Input/Output types with examples
- Error scenarios (terminal vs retryable)
- Idempotency guarantees
- State mutations (if any)
- Expected latency ranges
- External dependencies

Output format: OpenAPI 3.0 spec or markdown with curl examples.

For Workflows:
- Document Run handler lifecycle
- Explain signal/promise interactions
- Provide state diagram for complex flows
```

---

## Meta-Prompts (Prompts for Creating Prompts)

### Prompt #25: Generate Custom Restate Prompt

```
Create a specialized prompt for:

[SPECIFIC TASK/PATTERN]

The prompt should:
1. Clearly state the objective
2. Provide Restate-specific context and constraints
3. Include code examples following best practices
4. List verification criteria
5. Highlight common pitfalls to avoid

Use these patterns from DOS_DONTS_MEGA.MD and DOS_DONTS_REA.MD as reference:
[RELEVANT PATTERNS]

Output as ready-to-use prompt template.
```

---

## Quick Command Reference

### Essential Restate CLI Commands

```bash
# Register service deployment
curl -X POST http://localhost:9070/deployments \
  -H "Content-Type: application/json" \
  -d '{"uri": "http://localhost:9080"}'

# Invoke handler with idempotency
curl -X POST http://localhost:8080/MyService/MyHandler \
  -H "idempotency-key: unique-key-123" \
  -d '"input data"'

# Attach to ongoing invocation
curl http://localhost:8080/restate/invocation/{invocation-id}/attach

# Cancel invocation
curl -X DELETE http://localhost:8080/restate/invocation/{invocation-id}

# Resolve awakeable
curl -X POST http://localhost:9080/restate/awakeables/{id}/resolve \
  -H "Content-Type: application/json" \
  -d '{"value": "result"}'
```

---

## Best Practices Summary

### Golden Rules for LLM Responses

1. **Always cite DOS_DONTS patterns** when explaining decisions
2. **Provide complete, runnable code** - no pseudocode
3. **Include imports** at the beginning of code snippets
4. **Add error handling** for every external call
5. **Follow Go conventions** (exported names, error returns)
6. **Use framework helpers** from rea where applicable
7. **Comment anti-patterns** explicitly with ❌ markers
8. **Show both wrong and right** approaches side-by-side

### Response Format Template

```markdown
## Analysis
[What the code does or problem identified]

## Issues Found
1. ❌ **Issue**: [Description]
   - Location: [File:Line]
   - Why problematic: [Reason]
   
## Recommended Fix
✅ **Solution**: [Description]

```go
// Correct implementation
[CODE]
```

## Verification
- [ ] Compiles without errors
- [ ] Follows Restate determinism rules
- [ ] Proper error handling
- [ ] Includes logging
```

---

## Appendix: Common Error Messages

| Error Code | Meaning | Common Cause | Fix |
|------------|---------|--------------|-----|
| RT0004 | Non-determinism | Code changed without re-deploy | Deploy new version |
| RT0016 | Journal mismatch | Incompatible code change | Rollback or fix + redeploy |
| RT0009 | State access error | Wrong context type | Use ObjectContext for state |
| RT0005 | Invocation suspended | Awakeable not resolved | Check callback endpoint |

---

## Additional Resources

- [Restate Documentation](https://docs.restate.dev/)
- [Restate Go SDK Reference](https://pkg.go.dev/github.com/restatedev/sdk-go)
- [Rea Framework Repository](https://github.com/pithomlabs/rea)
- DOS_DONTS_MEGA.MD (this repository)
- DOS_DONTS_REA.MD (this repository)

---

**Last Updated**: 2025-11-28  
**Version**: 1.0  
**Maintained By**: [Your Team/Project]
