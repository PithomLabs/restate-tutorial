# PROMPTS.MD

## Effective Prompt Engineering for Restate Go SDK & pithomlabs/rea Framework Development

This guide provides optimized prompts for developers and AI assistants when building resilient distributed applications with Restate. These prompts incorporate context engineering principles to ensure accurate, deterministic, and durable code generation.

## I. Context Engineering Templates

When working with LLMs, always include this foundational context in your prompts:

```
I am developing a distributed application using the Restate Go SDK and pithomlabs/rea framework. I need code that follows these core principles:
1. DETERMINISM: All non-deterministic operations must be wrapped in restate.Run or framework.RunDo
2. JOURNALING: Only use Restate context (ctx) operations for durable execution flow
3. STATE ISOLATION: Virtual Objects use ObjectContext for writes, ObjectSharedContext for reads
4. ERROR HANDLING: Use TerminalError for non-recoverable failures, standard errors for transient issues
5. CONCURRENCY: Use Restate's durable combinators (Wait, WaitFirst) instead of goroutines/channels
6. EXTERNAL COORDINATION: Use Awakeables for external systems, Durable Promises for workflow internal signaling

I'm targeting Restate Server v1.5+ and using the pithomlabs/rea framework with PolicyStrict enforcement.
```

## II. Service Type Selection Prompts

### For System Design Questions

```
I need to design a [describe your business process] in Restate. Based on the requirements for state management, concurrency needs, and process longevity:

1. Which service type is most appropriate: Basic Service, Virtual Object, or Workflow?
2. What are the specific handler signatures needed?
3. How should state be structured and accessed?
4. What coordination primitives would be needed (Awakeables, Durable Promises, timers)?
5. What compensation logic (Sagas) would be required for failure scenarios?

Focus on Restate-specific concerns rather than general architecture. Include code snippets showing the service definition structure.
```

### Virtual Object Prompt Template

```
Generate a Virtual Object for [entity type] with these requirements:
- Key: [key type, e.g., userID, cartID]
- State model: [describe state schema]
- Exclusive handlers needed: [list of write operations]
- Shared handlers needed: [list of read-only operations]
- Concurrency constraints: [describe any limitations]
- Error handling requirements: [terminal vs transient]

Ensure code follows these rules:
- Use framework.NewMutableObjectState for write operations
- Use framework.NewReadOnlyObjectState for read operations
- Never use Sleep or Awakeables in exclusive handlers
- Separate Control Plane (orchestration) from Data Plane (side effects)
- Include TypeSafe accessors for state

Generate the complete Go implementation using pithomlabs/rea framework patterns.
```

## III. Handler Implementation Prompts

### Basic Service Handler Prompt

```
Create a Basic Service handler for [operation name] that:
1. Accepts input of type [input type]
2. Returns [output type] or error
3. Performs these side effects: [list external operations]
4. Has these coordination steps: [list service calls/timers/etc]

Enforce these constraints:
- Wrap ALL external operations in framework.RunDo
- Never use time.Now() directly, use framework.NewTime(ctx).Now()
- Use framework.DeterministicHelpers for UUID generation
- Use ctx.Log() for logging, not fmt.Printf or global loggers
- Handle TerminalError appropriately for business validation failures
- Use framework.ServiceClient for internal service calls, NOT ingress clients

Generate the complete handler implementation with proper error handling and context usage.
```

## IV. Error Handling & Sagas Prompts

### Saga Implementation Prompt

```
Implement a Saga pattern for [business process name] with these steps:
1. Step 1: [action] with compensation [compensation action]
2. Step 2: [action] with compensation [compensation action]
3. ...

Requirements:
- Use framework.SafeStep to register compensations BEFORE executing actions
- Ensure all compensations are idempotent
- Use framework.NewTerminalError for non-recoverable failures
- Implement proper deferred compensation handling
- Use the appropriate context type (WorkflowContext for long-running processes)

Provide the complete implementation using pithomlabs/rea framework's saga utilities, showing both the forward actions and compensation logic. Include comments explaining why the compensation order matters.
```

## V. Concurrency Pattern Prompts

### Parallel Task Execution Prompt

```
Create a handler that executes these [N] tasks in parallel:
1. Task 1: [description and service call]
2. Task 2: [description and service call]
...

Requirements:
- Use framework.FanOutFail for execution (fail fast on first error)
- OR use framework.Gather if all results are needed regardless of errors
- Include timeout handling using framework.RacePromiseWithTimeout
- Handle partial failures appropriately
- Process results in deterministic order

Show the complete implementation with proper error handling and result aggregation. Explain why native Go concurrency primitives (goroutines/channels) must NOT be used for this pattern in Restate.
```

## VI. State Management Prompts

### State Migration Prompt

```
I need to migrate state from this old structure:
[type definition of old state]

To this new structure:
[type definition of new state]

Requirements:
- Maintain backward compatibility during deployment
- Handle cases where state might be in either format
- Implement a dedicated migration handler that can be triggered
- Ensure the migration is idempotent and safe to retry
- Clear obsolete state fields after migration

Using the pithomlabs/rea framework, create:
1. A versioned state accessor that handles both formats
2. A migration handler that transforms old state to new state
3. Updated business logic handlers that work with the new state format
4. Proper error handling for migration failures

Include comments explaining how this approach maintains determinism during the migration process.
```

## VII. External Coordination Prompts

### External System Integration Prompt

```
Create a handler that integrates with [external system] using the callback pattern:

Workflow:
1. Initiate operation with external system
2. Return an operation ID to the caller immediately
3. Wait for external system to call back with results
4. Process the results and update state accordingly

Requirements:
- Use framework.WaitForExternalSignal for waiting (NOT Durable Promises)
- Generate a deterministic operation ID using framework.DeterministicHelpers
- Include timeout handling with framework.RacePromiseWithTimeout
- Validate callback authenticity using [authentication method]
- Handle duplicate callbacks safely (idempotency)
- Log appropriate context for debugging but avoid PII

Generate both:
1. The handler that initiates the external operation
2. The callback handler that processes the external system's response

Show proper error handling for timeouts, invalid callbacks, and system errors.
```

## VIII. Advanced Pattern Prompts

### Workflow with Human Approval Prompt

```
Design a Workflow for [business process] requiring human approval:

Steps:
1. Initial processing (automated)
2. Send approval request to [approval system]
3. Wait for approval/rejection with timeout
4. Handle approval path
5. Handle rejection path
6. Handle timeout path

Requirements:
- Use framework.GetInternalSignal for internal workflow signaling
- Implement proper timeout handling with framework.WorkflowTimer
- Design shared handlers for status queries and external signals
- Implement compensation logic for the entire process
- Allow manual cancellation via a dedicated signal handler
- Ensure state is properly cleared after completion

Generate:
1. The main workflow Run handler
2. Signal handlers for approval/rejection
3. Query handler for status checking
4. Cleanup handler for state management

Include comments explaining how this design ensures exactly-once execution and survives workflow suspension/resumption.
```

## IX. Testing & Validation Prompts

### Determinism Validation Prompt

```
Create test cases to validate the determinism of this Restate handler:
[include handler code to be tested]

Tests should verify:
1. Context operations are not used inside restate.Run closures
2. All non-deterministic operations are properly wrapped
3. Time and UUID generation uses deterministic helpers
4. No global state is accessed or modified
5. Map iteration doesn't affect execution flow
6. Logging uses ctx.Log() not global loggers
7. Error paths are correctly handling TerminalError vs regular errors

Generate:
1. Unit tests using the restate/test package
2. Framework policy enforcement tests
3. Replay simulation tests
4. Negative test cases that should fail determinism checks

Include setup code for the test environment and explain how each test validates a specific determinism constraint.
```

## X. Framework-Specific Prompts for pithomlabs/rea

### Framework Policy Enforcement Prompt

```
Configure the pithomlabs/rea framework with these policies:
- Policy level: PolicyStrict
- Context violation handling: [describe behavior]
- Determinism checks: [describe requirements]
- State access validation: [describe requirements]

Generate:
1. Application bootstrap code that sets global framework policy
2. Service registration with appropriate handlers
3. Middleware configuration for security and validation
4. Error handling middleware that respects TerminalError semantics
5. Configuration for local development vs production environments

Include comments explaining how these policies prevent common Restate development errors and ensure durability guarantees are maintained.
```

## XI. Security & Compliance Prompts

### Secure Service Communication Prompt

```
Implement secure communication between these Restate services:
- Service A: [service name and purpose]
- Service B: [service name and purpose]

Requirements:
- All service-to-service calls must use internal clients (framework.ServiceClient)
- No ingress clients inside handlers
- Request identity validation for all endpoints
- Headers propagation for tracing and user context
- Proper service isolation (mark internal services as private)
- L3 security enforcement with framework.SecurityMiddleware

Generate:
1. Service definitions with proper ingress configuration
2. Client creation patterns for inter-service communication
3. Middleware setup for request validation
4. Handler implementations that respect security boundaries
5. Deployment configuration snippets

Include comments explaining how this implementation satisfies the zero-trust security model required by Restate applications.
```

## XII. Context Engineering for LLMs

### Best Practices for Querying LLMs about Restate

When asking an LLM about Restate development:

1. **ALWAYS** include the framework version and Restate server version
   ```
   I'm using pithomlabs/rea v1.2.0 with Restate Server v1.5.0
   ```

2. **NEVER** ask for simplified examples that skip framework abstractions
   ```
   Don't say: "Show me a simple example without all the framework overhead"
   Instead say: "Show me a production-ready implementation using framework.RunDo and framework.ServiceClient"
   ```

3. **ALWAYS** specify the service type context
   ```
   This is for a Virtual Object handler using ObjectContext, not a Basic Service
   ```

4. **Include failure scenarios** in your prompts
   ```
   Also show how this would handle network failures during the external API call and what compensation would be needed
   ```

5. **Request validation of determinism** explicitly
   ```
   Verify that this implementation maintains determinism by checking for any non-deterministic operations outside of Run blocks
   ```

## XIII. Troubleshooting Prompts

### Debugging Replay Failures Prompt

```
I'm experiencing a journal mismatch error during replay with this handler:
[include handler code]

Error message:
[include exact error message]

Context:
- Framework version: [version]
- Restate server version: [version]
- Operation occurs at: [step in workflow]

Troubleshoot by:
1. Identifying non-deterministic operations that aren't properly wrapped
2. Checking for context misuse inside Run closures
3. Verifying time/UUID generation methods
4. Examining error handling paths
5. Reviewing state access patterns

Provide:
1. Specific fixes for the identified issues
2. A corrected version of the handler
3. Tests to prevent recurrence
4. Logging improvements to detect similar issues earlier

Explain how each change maintains the durability guarantees of Restate while fixing the determinism issue.
```

## XIV. Performance Optimization Prompts

### Handler Optimization Prompt

```
Optimize this Restate handler for performance while maintaining durability guarantees:
[include handler code]

Focus on:
1. Minimizing handler execution time (especially in Virtual Object exclusive handlers)
2. Efficient state access patterns (lazy vs eager loading)
3. Appropriate use of parallelism with framework.FanOut
4. Proper timeout configuration to avoid zombie invocations
5. Memory usage during replay operations
6. Efficient serialization of state objects

Do NOT sacrifice:
1. Determinism guarantees
2. Exactly-once semantics for side effects
3. Proper error handling and compensation
4. State consistency

Provide:
1. Annotated optimized code
2. Performance comparison metrics
3. Configuration recommendations for timeouts and retries
4. Monitoring suggestions to detect future performance regressions

Explain the trade-offs made and why certain optimizations were rejected to maintain Restate's core guarantees.
```