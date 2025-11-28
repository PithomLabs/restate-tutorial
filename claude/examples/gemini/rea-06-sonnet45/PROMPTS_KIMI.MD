# PROMPTS.MD

## Comprehensive Prompt Engineering Guide for Restate Go SDK & PithomLabs/REA Framework

This document provides actionable prompts for developers and LLMs to build resilient distributed applications using Restate's Durable Execution model. Each prompt is designed to be copy-pasted into your AI assistant with clear context and expected outcomes.

---

## 1. CODE GENERATION PROMPTS

### 1.1 Generate a Production-Ready Basic Service Handler

**Prompt:**
```
You are a senior distributed systems engineer expert in Restate Go SDK. Generate a complete Basic Service handler that processes webhook events from a payment gateway with:
1. Durable execution wrapping for the external API call
2. Proper idempotency key handling using restate.UUID(ctx)
3. TerminalError for permanent failures (invalid payment)
4. Context logging to avoid duplicate logs
5. Fan-out pattern for parallel notification sending
6. Proper error handling with compensation logic

The handler should accept a PaymentWebhookRequest struct and return a PaymentProcessingResult.

Use the following signature:
func (p *PaymentService) ProcessWebhook(ctx restate.Context, req PaymentWebhookRequest) (PaymentProcessingResult, error)

Include all necessary imports and type definitions.
```

**Expected Output:** Complete Go code with `restate.Run` blocks, `restate.UUID(ctx)`, `ctx.Log()`, `restate.Wait` for fan-out, `restate.TerminalError`, and proper struct definitions.

---

### 1.2 Generate a Virtual Object for Stateful User Cart

**Prompt:**
```
Generate a Virtual Object implementation for a user shopping cart using Restate Go SDK with these requirements:
1. Exclusive handler for adding items with state mutation
2. Shared handler for reading cart contents (concurrent safe)
3. State management using restate.Get/Set with proper error handling
4. State cleanup when cart is empty
5. Prevent stock race conditions using single-writer consistency
6. Use DeterministicHelpers for generating cart item IDs

Provide both AddItem(ctx restate.ObjectContext, item CartItem) error and GetCart(ctx restate.ObjectSharedContext) (Cart, error) handlers.

Include the struct definition and reflection setup.
```

**Expected Output:** Complete Virtual Object with `restate.ObjectContext` for writes, `restate.ObjectSharedContext` for reads, state accessors, and deterministic ID generation.

---

### 1.3 Generate a Human-in-the-Loop Workflow with Awakeable

**Prompt:**
```
Create a Workflow for an approval process that:
1. Waits for manager approval via Awakeable (external system)
2. Sends reminder emails every 24 hours using durable timers
3. Times out after 7 days if no approval
4. Uses Promise for internal status queries
5. Implements idempotent compensation for any external actions
6. Contains shared handlers for querying status and canceling

Use restate.WorkflowContext for Run handler and restate.WorkflowSharedContext for signal handlers.
Show how to generate the awakeable ID and pass it to the external system.
```

**Expected Output:** Complete workflow with `restate.Awakeable`, `restate.After`, `restate.WaitFirst`, shared context handlers, and proper timeout handling.

---

### 1.4 Generate a REA-Compliant Service with Data Plane Isolation

**Prompt:**
```
Using the PithomLabs/REA framework, generate a service that:
1. Uses framework.RunDo[T] for all external API calls
2. Implements framework.Saga with compensation registration before actions
3. Uses framework.DeterministicHelpers for all ID generation
4. Follows Control Plane vs Data Plane separation strictly
5. Includes framework.SetFrameworkPolicy(framework.PolicyStrict)
6. Uses framework.NewIngressClient for external calls with proper idempotency keys

Show the complete service structure with proper imports and framework initialization.
```

**Expected Output:** REA-specific code with `framework.RunDo`, `framework.Saga`, `framework.PolicyStrict`, and clear separation of concerns.

---

## 2. CODE REVIEW & AUDIT PROMPTS

### 2.1 Audit for Determinism Violations

**Prompt:**
```
Review this Restate Go handler code for determinism violations. Identify any:
1. Non-deterministic operations outside restate.Run blocks
2. Context operations (ctx.Sleep, ctx.Call, ctx.Set) inside Run blocks
3. Use of time.Now(), rand, or crypto/rand outside Run blocks
4. Iteration over maps/dicts with non-deterministic order
5. Improper use of goroutines with blocking Restate operations

Provide specific line numbers and explain why each is a violation. Suggest fixes with code examples.

[CODE TO REVIEW]
```

**Expected Output:** Detailed violation report with specific issues, explanations referencing the journal/replay mechanism, and corrected code snippets.

---

### 2.2 Audit for Virtual Object Deadlock Risks

**Prompt:**
```
Analyze this Virtual Object code for potential deadlock scenarios:
1. Check for invocation chains A→B→A with same keys
2. Identify uses of ctx.Sleep or Awakeable.Result in exclusive handlers
3. Check for complex request-response patterns between objects
4. Verify delayed messages are used instead of blocking sleeps

Classify each finding as: CRITICAL (will deadlock), WARNING (may deadlock under load), or INFO (best practice violation).

[CODE TO REVIEW]
```

**Expected Output:** Deadlock risk analysis with classification, explanation of execution order, and recommendations for refactoring.

---

### 2.3 Audit for State Management Issues

**Prompt:**
```
Review this code for state management anti-patterns:
1. State mutation in shared handlers (ObjectSharedContext/WorkflowSharedContext)
2. Missing error handling for state operations
3. State not cleared for long-lived Virtual Objects
4. Use of non-pointer types with Get when zero-value checking is needed
5. Concurrent state access patterns that could violate consistency

For each issue, show the correct pattern using restate.Get/Set or framework.NewMutableObjectState.
```

**Expected Output:** State management audit report with specific anti-patterns, correct patterns, and code examples.

---

### 2.4 Audit for Sagas and Compensation Issues

**Prompt:**
```
This code implements a Saga pattern. Audit it for:
1. Compensation registration order (must be before action)
2. Idempotency of compensation functions
3. Proper use of TerminalError vs transient errors
4. Missing compensation for "one-shot" APIs
5. Use of defer for compensation execution

Provide a corrected version with framework.Saga helpers if beneficial.
```

**Expected Output:** Saga pattern analysis with compensation flow diagrams, idempotency checks, and improved code using defer or framework.Saga.

---

### 2.5 Audit for REA Framework Compliance

**Prompt:**
```
Review this REA framework code against PolicyStrict settings:
1. Verify all external calls use framework.RunDo or framework.SafeRun
2. Check DeterministicHelpers usage for UUIDs and time
3. Verify framework.Policy is set
4. Check for proper Data Plane isolation
5. Validate idempotency keys don't use timestamps
6. Check identity propagation via headers

Provide a compliance score and specific violations with fixes.
```

**Expected Output:** REA compliance report with scoring, specific framework violation detection, and PolicyStrict enforcement examples.

---

## 3. DEBUGGING & TROUBLESHOOTING PROMPTS

### 3.1 Debug Non-Determinism Error (RT0016)

**Prompt:**
```
I'm getting error code RT0016 (Journal Mismatch) in Restate. My handler:
1. Makes external API calls
2. Uses time.Sleep for delays
3. Generates IDs with uuid.New()
4. Iterates over a map to process items

Explain the root cause and provide a step-by-step fix using restate.Run, restate.After, and restate.UUID.
```

**Expected Output:** Explanation of journal mismatch, identification of non-deterministic operations, and complete refactored handler.

---

### 3.2 Debug Virtual Object Queueing Behavior

**Prompt:**
```
My Virtual Object handler processes one request at a time, even though I expected concurrent reads. The handler:
- Uses ObjectContext for all operations
- Contains a 30-second sleep for rate limiting
- Makes external API calls

Why is it blocking? Show me how to:
1. Convert to ObjectSharedContext for reads
2. Use delayed messages instead of sleep
3. Wrap external calls in RunAsync
```

**Expected Output:** Explanation of single-writer semantics, refactoring to shared context, delayed message pattern, and async execution.

---

### 3.3 Debug Workflow Resume Issues

**Prompt:**
```
My Workflow doesn't resume after waiting for a Promise. The.Run handler calls:
promise := restate.Promise[string](ctx, "approval")
result, err := promise.Result()

I have a separate handler:
func (w *MyWorkflow) Approve(ctx restate.WorkflowSharedContext, req Approval)

But the workflow never continues. What's wrong?

Show the correct Promise resolution pattern and verify handler registration.
```

**Expected Output:** Explanation of shared context requirement for signal handlers, correct promise resolution pattern, and registration verification steps.

---

### 3.4 Debugging Compensation Not Executing

**Prompt:**
```
My Saga compensation functions aren't running when a step fails. I have:
func (s *BookingService) Book(ctx restate.Context) error {
    compensations := []func() error{}
    defer func() {
        if err != nil {
            for _, c := range compensations {
                c() // Not executing
            }
        }
    }()
    // ... register compensations ...
    return restate.Run(ctx, func(rc restate.RunContext) error {
        return bookFlight() // fails here
    })
}

Why aren't compensations running? Show the correct defer pattern with TerminalError handling.
```

**Expected Output:** Explanation of error shadowing, correct defer placement, TerminalError propagation, and compensation execution flow.

---

## 4. ARCHITECTURE & DESIGN DECISION PROMPTS

### 4.1 Service Type Selection Decision Tree

**Prompt:**
```
I need to implement a user notification system that:
- Stores per-user notification preferences (persistent)
- Sends notifications via external SMS/email APIs
- Must handle 1000+ concurrent users
- Needs rate limiting per user
- Requires human escalation for failed deliveries

Should I use Basic Service, Virtual Object, or Workflow? Provide a detailed decision matrix with pros/cons for each option and recommend the optimal architecture with code sketch.
```

**Expected Output:** Decision matrix comparing service types, concurrency implications, state management tradeoffs, and recommended hybrid architecture (Virtual Object for state + Basic Service for sending + Workflow for escalation).

---

### 4.2 Awakeable vs Durable Promise Decision

**Prompt:**
```
My use case: A payment processing system where:
- External payment gateway calls our webhook when payment completes
- Internal fraud check service needs to signal approval
- Workflow must wait for both signals before proceeding

For each signal type (external webhook, internal fraud check), should I use Awakeable or Durable Promise? Explain the resolution mechanism, ID management, and provide code for both patterns.

Also show how to race both signals with a 24-hour timeout.
```

**Expected Output:** Clear decision rationale, Awakeable for external webhook (with ID passing), Durable Promise for internal fraud check (with logical name), and timeout racing pattern.

---

### 4.3 State Storage Strategy Decision

**Prompt:**
```
I need to store:
1. User session state for AI chatbot (conversation history, 10KB per user)
2. Product catalog data (100K products, 1MB each, read-heavy)
3. Real-time rate limit counters (per-user, frequent updates)
4. Workflow orchestration state (temporary, 24hr lifetime)

Which should use Restate state vs external database? Provide a decision framework with criteria (access patterns, size, lifetime, consistency needs) and migration strategy if needed later.
```

**Expected Output:** Decision framework with criteria matrix, clear separation (Restate for session/rate limiting, external DB for catalog), and migration pattern code.

---

### 4.4 REA Framework Architecture Review

**Prompt:**
```
Review my proposed microservices architecture using REA framework:
- 5 services total
- 3 Virtual Objects for stateful entities
- 2 Workflows for long-running processes
- All services call external payment and notification APIs

Create a security and compliance checklist covering:
1. L1/L2/L3 security layers
2. PolicyStrict enforcement points
3. Idempotency key validation
4. Identity propagation patterns
5. Private service markings
6. Request signature validation

Provide a scorecard template I can use for each service.
```

**Expected Output:** Comprehensive security checklist, scoring rubric, and service-by-service evaluation template with specific REA framework configurations.

---

## 5. BEST PRACTICE ENFORCEMENT PROMPTS

### 5.1 Enforce Control Plane/Data Plane Separation

**Prompt:**
```
Analyze this Restate handler and refactor it to strictly separate Control Plane and Data Plane:
[ORIGINAL CODE WITH MIXED LOGIC]

Rules:
1. All external calls (HTTP, DB, Email) MUST be in framework.RunDo blocks
2. All orchestration logic (conditionals, loops, state access) stays in Control Plane
3. No time.Now(), uuid.New(), or random outside Run blocks
4. Use framework.DeterministicHelpers for IDs and timestamps
5. Use framework.NewTime for deterministic time

Provide the refactored code with clear CP/DP comments.
```

**Expected Output:** Clearly separated code with CP and DP sections labeled, all side effects isolated, and deterministic helpers used.

---

### 5.2 Enforce Idempotency Key Best Practices

**Prompt:**
```
Review all restate.ServiceSend and ingress client calls in this codebase:
1. Verify idempotency keys are deterministic (not timestamps)
2. Add keys where missing for external calls
3. Remove redundant keys for intra-handler calls
4. Ensure keys are unique per business operation
5. Validate key format matches framework policy

Provide a report with line references and corrected code.
```

**Expected Output:** Idempotency audit report with key generation best practices, corrected calls, and policy validation.

---

### 5.3 Enforce Proper Error Handling Patterns

**Prompt:**
```
Convert all error returns in this handler to proper Restate error types:
1. Identify business logic errors (invalid input, validation fails) → TerminalError
2. Identify transient errors (network, DB) → let Restate retry
3. Add context to errors using fmt.Errorf with %w
4. Ensure TerminalError includes proper status codes
5. Add error logging using ctx.Log()

Show before/after for each error return.
```

**Expected Output:** Systematic error classification, TerminalError conversion table, and improved error messages with logging.

---

### 5.4 Enforce REA Framework PolicyStrict

**Prompt:**
```
Apply PolicyStrict to this REA codebase by:
1. Adding framework.SetFrameworkPolicy(framework.PolicyStrict) to main()
2. Converting all ctx.Run to framework.RunDo
3. Replacing direct restate.UUID with framework.DeterministicHelpers
4. Adding security middleware to all public endpoints
5. Validating all idempotency keys comply with policy

Provide the complete diff and explain each change's security benefit.
```

**Expected Output:** PolicyStrict implementation diff, security justification for each change, and compliance verification.

---

## 6. COMMON PATTERN IMPLEMENTATION PROMPTS

### 6.1 Implement Durable Webhook Processor

**Prompt:**
```
Generate a complete Basic Service that acts as a durable webhook processor:
1. Accepts events via HTTP handler
2. Uses idempotency key from X-Idempotency-Key header
3. Wraps business logic in restate.Run
4. Returns 200 OK immediately, processes asynchronously
5. Handles duplicate events safely
6. Includes retry logic with exponential backoff

Show the HTTP handler setup and Restate handler implementation.
```

**Expected Output:** Production-ready webhook processor with idempotency, durability, and async processing.

---

### 6.2 Implement Distributed Saga with Compensation

**Prompt:**
```
Implement a booking saga across 3 services (Flight, Hotel, Payment) using REA framework:
1. Use framework.Saga with SafeStep for each booking
2. Register compensation BEFORE each action
3. Use Send with delay for async coordination
4. Handle TerminalError to trigger compensations
5. Ensure all compensations are idempotent
6. Add query handler for saga status

Provide the complete workflow code with proper error propagation.
```

**Expected Output:** Complete Saga implementation with SafeStep, compensation registration, and status tracking.

---

### 6.3 Implement AI Agent Session Management

**Prompt:**
```
Generate a Virtual Object for AI Agent session management:
1. Stores conversation history in Restate state (max 50 messages)
2. Uses ObjectContext for exclusive access to prevent race conditions
3. Wraps LLM calls in restate.Run with durableCalls middleware
4. Implements cleanup after 24 hours of inactivity
5. Provides shared handler for read-only session queries
6. Handles session expiration gracefully

Use framework.DeterministicHelpers for session IDs and include state size limits.
```

**Expected Output:** AI Agent session manager with durable LLM calls, state management, and cleanup logic.

---

### 6.4 Implement Rate Limiter using Virtual Object

**Prompt:**
```
Create a Virtual Object rate limiter implementing token bucket algorithm:
1. Stores tokens and last refill time in state
2. Uses restate.Sleep for refill scheduling (with delayed message alternative)
3. Handles concurrent requests with single-writer consistency
4. Returns rate limit headers (X-RateLimit-Remaining, X-RateLimit-Reset)
5. Implements proper state cleanup for unused limiters

Show both blocking and async (delayed message) implementations with tradeoffs.
```

**Expected Output:** Rate limiter implementation with token bucket logic, state management, and alternative async pattern.

---

## 7. MIGRATION & UPGRADE PROMPTS

### 7.1 Migrate from SDK < 0.16 to Latest

**Prompt:**
```
My codebase uses deprecated Restate Go SDK features:
1. restate.Select (deprecated)
2. Handler methods on pointer receivers
3. SDK version < 0.16 patterns
4. No context logger usage

Provide a migration guide with:
- Before/after code for each deprecated feature
- Step-by-step upgrade sequence
- Testing strategy for ensuring durability after migration
- Common pitfalls to avoid

My current version: 0.15.2, target: 1.0.0
```

**Expected Output:** Comprehensive migration guide with deprecation mappings, code transformations, and testing checklist.

---

### 7.2 Migrate to REA Framework

**Prompt:**
```
I want to migrate my existing Restate services to PithomLabs/REA framework. Current state:
- 10 services using raw restate.Run
- No Data Plane isolation
- Mix of Basic Services and Virtual Objects
- Custom error handling

Provide a migration strategy:
1. Which services to migrate first (priority order)
2. How to introduce framework.RunDo incrementally
3. How to enable PolicyStrict gradually
4. Testing approach to ensure zero-downtime migration
5. Rollback plan if issues occur

Show a migration example for one service.
```

**Expected Output:** Phased migration plan, priority criteria, incremental adoption strategy, and example migration with testing.

---

## 8. TESTING & VALIDATION PROMPTS

### 8.1 Generate Determinism Test Cases

**Prompt:**
```
Generate comprehensive test cases to verify my Restate handler is deterministic:
1. Test that replay produces identical journal entries
2. Test external calls are wrapped and results are stable
3. Test concurrent operations complete in deterministic order
4. Test state operations replay correctly
5. Test error handling doesn't affect determinism

Provide table-driven tests using Go's testing package and restate test harness.
```

**Expected Output:** Test suite with journal verification, replay scenarios, and determinism assertions.

---

### 8.2 Generate Load Test for Virtual Object Concurrency

**Prompt:**
```
Create a load test for my Virtual Object that verifies:
1. Exclusive handlers process sequentially per key
2. Shared handlers process concurrently per key
3. No deadlocks under high concurrency (1000 goroutines)
4. State consistency is maintained
5. Performance metrics (p50, p95, p99 latency)

Use Go's testing/benchmark and provide metrics interpretation guide.
```

**Expected Output:** Load test harness with concurrency verification, deadlock detection, and performance analysis.

---

### 8.3 Generate Saga Compensation Test

**Prompt:**
```
Generate tests for my Saga implementation that verify:
1. Compensations execute in reverse order on failure
2. Compensations are idempotent (can be called multiple times)
3. Partial failure scenarios are handled correctly
4. Successful completion doesn't trigger compensations
5. TerminalError vs transient error distinction

Provide test fixtures that simulate failures at each step.
```

**Expected Output:** Comprehensive Saga test suite with failure injection, compensation verification, and idempotency checks.

---

## 9. LLM-SPECIFIC PROMPTS (for AI Assistants)

### 9.1 Context Engineering Prompt

**Prompt:**
```
You are REA-CodeGen, a specialized AI for PithomLabs/REA framework. Your context:
- Framework: github.com/pithomlabs/rea v0.5.0
- Restate SDK: github.com/restatedev/sdk-go v1.0.0
- Policy: PolicyStrict enabled
- Architecture: Control Plane / Data Plane separation mandatory
- Security: L1/L2/L3 layers must be validated

When generating code:
1. Always start with framework.SetFrameworkPolicy(framework.PolicyStrict)
2. Use framework.RunDo for all external calls
3. Use framework.DeterministicHelpers for IDs/time
4. Validate idempotency keys against policy
5. Add security middleware to public endpoints

Current task: [USER TASK]
```

**Expected Output:** AI assistant response that follows REA best practices strictly, includes necessary imports, and validates security.

---

### 9.2 Anti-Pattern Detection System Prompt

**Prompt:**
```
Analyze this Restate codebase as an expert linter. For each file, detect:
1. Non-deterministic operations outside Run blocks (CRITICAL)
2. Context misuse in Run closures (CRITICAL)
3. Missing idempotency keys on external calls (WARNING)
4. State mutation in shared handlers (CRITICAL)
5. Blocking operations in Virtual Object exclusive handlers (CRITICAL)
6. Use of deprecated APIs like restate.Select (INFO)
7. Missing TerminalError for business failures (WARNING)

Output in SARIF format with file:line:column references and severity levels.
```

**Expected Output:** Machine-readable linting report with severity classification and precise location information.

---

### 9.3 Code Explanation Prompt for Distributed Systems Novices

**Prompt:**
```
Explain this Restate handler to a developer new to distributed systems:
1. Break down the Control Plane vs Data Plane
2. Explain the journal/replay mechanism in simple terms
3. Describe how durability is achieved
4. Explain the concurrency model (Virtual Object single-writer)
5. Describe error handling (retry vs TerminalError)
6. Use the "Resilient Chef" analogy from DOS_DONTS_REA.MD

Make it accessible with diagrams in ASCII art if helpful.
```

**Expected Output:** Beginner-friendly explanation with analogies, ASCII diagrams, and clear separation of concepts.

---

### 9.4 Optimization Prompt for Performance

**Prompt:**
```
Optimize this Restate handler for maximum concurrency and minimal latency:
1. Identify sequential calls that can be parallelized with restate.Wait
2. Convert blocking calls to async with RequestFuture
3. Replace ctx.Sleep in Virtual Objects with delayed messages
4. Add lazy state loading for large state objects
5. Optimize RocksDB configuration for this workload

Provide performance metrics before/after and explain each optimization's impact.
```

**Expected Output:** Optimized code with parallelization, async conversions, and configuration tuning recommendations with performance analysis.

---

## 10. QUICK REFERENCE PROMPTS

### 10.1 Generate Idempotency Key

**Prompt:**
```
Generate a deterministic idempotency key for a payment operation with:
- User ID: "user-123"
- Order ID: "order-456"
- Timestamp: NOT allowed
- Must be reproducible on replay

Show both raw restate.UUID and framework.DeterministicHelpers approaches.
```

**Expected Output:** `restate.UUID(ctx)` or `helpers.UUID()` implementation with user/order context.

---

### 10.2 Generate Timeout Pattern

**Prompt:**
```
Generate the canonical timeout pattern in Restate Go SDK:
- Race an external API call against a 30-second timer
- Handle timeout case explicitly
- Handle success case explicitly
- Ensure deterministic replay of the race result
- Clean up resources properly

Use restate.WaitFirst and restate.After.
```

**Expected Output:** Complete timeout pattern with `restate.WaitFirst`, proper error handling, and cleanup.

---

### 10.3 Generate State Access Boilerplate

**Prompt:**
```
Generate boilerplate for Virtual Object state management:
- Get with default value
- Set with error handling
- Clear single key
- Clear all state
- Check if key exists (distinguish zero value from missing)

Use generics and handle all error cases.
```

**Expected Output:** Reusable state access functions with proper error handling and zero-value distinction.

---

### 10.4 Generate Saga Registration Pattern

**Prompt:**
```
Generate the correct Saga compensation registration pattern:
- Register compensation BEFORE action
- Use defer for execution on error
- Handle TerminalError properly
- Ensure compensations are idempotent
- Support one-shot APIs (payment, booking)

Show both raw restate and framework.SafeStep approaches.
```

**Expected Output:** Correct Saga pattern with early compensation registration, idempotent functions, and error handling.

---

## USAGE INSTRUCTIONS

### For Developers:
1. Copy the relevant prompt based on your task (generation, review, debugging)
2. Replace `[CODE TO REVIEW]` or `[USER TASK]` with your specific content
3. Paste into your preferred AI assistant (Claude, GPT-4, etc.)
4. Review the output against the official Restate documentation
5. Test generated code in a non-production environment first

### For LLM Integration:
1. Use the "Context Engineering Prompt" (9.1) to prime the assistant
2. Follow up with specific prompts from this guide
3. Combine multiple prompts for complex tasks (e.g., generate + audit)
4. Use the anti-pattern detection (9.2) as an automated CI/CD linting step

### For Customization:
- Adjust prompts to match your specific types and service names
- Add business-specific validation rules to audit prompts
- Extend testing prompts with your failure scenarios
- Modify security prompts based on your threat model

---

## PROMPT COMPOSITION TEMPLATE

When creating custom prompts, use this structure:
```
Role: You are [expert role with specific expertise]

Context: 
- Framework: [specific versions]
- Policy: [PolicyStrict/PolicyWarn/PolicyDisabled]
- Architecture: [Control/Data Plane separation required?]
- Current Task: [specific goal]

Constraints:
1. [Technical constraint from DOS_DONTS]
2. [Another constraint]
3. [Security requirement]

Requirements:
- [Specific code pattern required]
- [Error handling approach]
- [Testing expectation]

[Optional: Code to analyze/augment]
```

This ensures consistent, high-quality responses from LLMs when working with Restate and REA framework.