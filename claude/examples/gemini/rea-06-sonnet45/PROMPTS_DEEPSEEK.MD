# PROMPTS.MD - Restate & REA Framework Engineering Guide

## 🤖 LLM Context Engineering Prompts

### Base Context Setup Prompts
```
You are an expert in distributed systems and the Restate framework. You specialize in building resilient, durable applications using the Go SDK and pithomlabs/rea framework. You always follow Restate's determinism principles and separate control plane from data plane.

CRITICAL RULES:
1. ALL external I/O must be wrapped in restate.Run/ctx.run
2. NEVER use native Go concurrency with Restate operations
3. ALWAYS use deterministic UUIDs and time from Restate context
4. Virtual Object exclusive handlers MUST NOT block with sleep/awakeables
5. Workflows for long-running processes, Virtual Objects for stateful entities

Current project uses: Restate Go SDK + pithomlabs/rea framework
```

```
When answering Restate questions, always consider:
- Is this operation deterministic?
- Should this be in control plane or data plane?
- Which service type is appropriate (Basic/Virtual Object/Workflow)?
- Are we maintaining execution replay safety?
```

### Service Type Selection Prompts
```
Based on these requirements, which Restate service type should I use and why?
Requirements: {describe your use case}
Consider: state needs, concurrency requirements, process duration, external coordination needs

Options to evaluate:
- Basic Service: stateless, high concurrency, orchestration
- Virtual Object: stateful entities, single-writer consistency, actor pattern
- Workflow: long-running, exactly-once execution, human-in-the-loop
```

```
Analyze this distributed system scenario and recommend the Restate architecture:
Scenario: {describe your distributed system problem}
Focus on: service boundaries, state management, error handling, compensation logic
```

## 🔧 Development Workflow Prompts

### Code Generation Prompts
```
Generate a Restate Virtual Object handler for {entity type} with these requirements:
- Key: {key pattern}
- State: {state structure}
- Operations: {list of operations}
- Error handling: {specific error cases}
- Use pithomlabs/rea framework helpers
```

```
Create a Workflow for {business process} with:
- Steps: {list of steps}
- Compensation logic for: {specific steps}
- Timeouts: {timeout requirements}
- External signals: {what to wait for}
- Use Durable Promises for internal coordination
```

```
Write a saga pattern implementation for {transaction} that spans:
Services involved: {service list}
Compensation actions: {compensation mapping}
Error conditions: {error scenarios}
Ensure all compensations are idempotent
```

### Code Review & Analysis Prompts
```
Review this Restate handler for determinism violations and best practices:

```go
{code to review}
```

Check for:
1. Unwrapped non-deterministic operations
2. Incorrect context usage
3. Concurrency safety
4. State management correctness
5. Error handling adequacy
```

```
Analyze this Virtual Object for potential deadlocks and blocking issues:

```go
{object code}
```

Focus on:
- Exclusive handler blocking operations
- Cross-service call patterns
- Key contention risks
- Alternative patterns (delayed messages, etc.)
```

## 🎯 Specific Pattern Implementation Prompts

### Determinism & Side Effects
```
How do I safely call {external API/database} from a Restate handler?
Generate the wrapper using framework.RunDo or framework.SafeRun with proper error handling and timeouts.
```

```
Create a deterministic version of this non-deterministic operation:

{original non-deterministic code}

Requirements:
- Wrap in appropriate Run block
- Handle retries and terminal errors
- Maintain execution replay safety
```

### State Management Patterns
```
Implement a rate limiter Virtual Object using:
- Token bucket algorithm
- Durable state for tokens and last update
- Sleep for refill periods
- Concurrent read access via shared handlers
```

```
Create a session management Virtual Object for {use case} with:
- Session state structure
- TTL/expiration handling
- Concurrent read operations
- Exclusive write operations
- State cleanup on session end
```

### Communication Patterns
```
Implement a request-response pattern between {service A} and {service B} with:
- Proper context usage
- Error propagation
- Timeout handling
- Idempotency considerations
```

```
Create a fire-and-forget message pattern for {use case} using:
- Delayed messages for scheduling
- Idempotency keys
- Error handling for delivery failures
```

### Concurrency & Coordination
```
Implement a fan-out/fan-in pattern for processing {items} with:
- Parallel execution of tasks
- Result aggregation
- Error handling for partial failures
- Progress tracking
```

```
Create a race condition between {operation} and timeout using:
- restate.WaitFirst or framework.RacePromiseWithTimeout
- Proper cleanup of unused futures
- Deterministic result handling
```

## 🚀 Advanced Architecture Prompts

### Workflow Design
```
Design a human approval workflow for {process} with:
- Multiple approval stages
- Timeouts at each stage
- Escalation paths
- State persistence between steps
- Queryable status endpoints
```

```
Implement a long-running ETL pipeline as a Workflow with:
- Checkpointing state
- Resume capability
- Progress reporting
- Error recovery and compensation
```

### Saga Pattern Implementation
```
Create a distributed transaction saga for {business transaction}:

Steps: {list of participating services}
Compensations: {compensation for each step}
Error conditions: {possible failures}
Recovery: {saga recovery logic}

Use the REA framework's saga helpers and ensure all compensations are idempotent.
```

### AI Agent Integration
```
Integrate an AI agent tool with Restate durability:

Tool: {tool description}
Requirements:
- Wrap LLM calls in durable execution
- Persist tool responses
- Handle tool execution failures
- Maintain conversation context in state
```

## 🔍 Debugging & Troubleshooting Prompts

### Determinism Violation Detection
```
I'm getting journal mismatch errors. Analyze this code for non-determinism:

```go
{problematic code}
```

Check for:
- Unwrapped I/O operations
- Non-deterministic control flow
- Incorrect use of native Go features
- Context misuse in Run blocks
```

### Deadlock Analysis
```
My Virtual Object is hanging. Analyze this communication pattern:

```go
{communication code}
```

Identify:
- Potential cross deadlocks
- Cycle deadlocks
- Blocking operations in exclusive handlers
- Alternative non-blocking patterns
```

### Performance Optimization
```
Optimize this Restate service for {performance metric}:

```go
{current implementation}
```

Consider:
- State loading strategy (eager vs lazy)
- Concurrent operation patterns
- Appropriate service type selection
- Database integration patterns
```

## 📚 Learning & Knowledge Gaps

### Concept Explanation Prompts
```
Explain {Restate concept} using the chef/recipe analogy from the documentation:
- Control plane vs data plane
- Deterministic execution
- Journal and replay
- Exactly-once semantics
```

```
Compare and contrast these Restate patterns:
- Awakeable vs Durable Promise
- Virtual Object vs Workflow
- Basic Service vs Virtual Object
- Request-response vs one-way messaging
```

### Best Practices Reinforcement
```
What are the top 5 mistakes to avoid when building {type of system} with Restate?
Focus on: determinism, concurrency, state management, error handling, service communication
```

```
Create a checklist for code review of Restate services covering:
- Determinism guarantees
- Context usage correctness
- Service type appropriateness
- Error handling completeness
- Performance considerations
```

## 🛠️ Framework-Specific Prompts

### REA Framework Usage
```
How do I implement {pattern} using the pithomlabs/rea framework instead of raw Restate SDK?
Focus on framework helpers for: deterministic operations, state management, error handling, sagas
```

```
Migrate this raw Restate code to use REA framework helpers:

```go
{raw restate code}
```

Update to use:
- framework.RunDo/framework.SafeRun
- framework state management
- framework deterministic helpers
- framework error handling
```

### Policy & Security
```
Configure framework security policy for {environment} with:
- Appropriate strictness level
- Request validation
- Identity propagation
- Private service marking
```

## 📈 Production & Operations Prompts

### Deployment & Scaling
```
Design a deployment strategy for Restate services with:
- Immutable versions
- Traffic migration
- Long-running invocation support
- Rollback procedures
```

```
Plan scaling for {workload characteristics}:
- Virtual Object key distribution
- Workflow execution patterns
- State storage considerations
- Concurrency limits
```

### Monitoring & Observability
```
Implement observability for {service type} with:
- Context logging
- Progress tracking
- Error reporting
- Performance metrics
```

## 🔄 Migration & Integration Prompts

### Legacy System Integration
```
Integrate Restate with existing {legacy system} for {use case}:
- Awakeable patterns for callbacks
- Idempotency key strategies
- Error handling for external failures
- State synchronization
```

### Database Integration Patterns
```
Implement the semantic lock pattern for {database operation} with:
- Read via restate.Run
- Conditional updates based on versions
- Exactly-once semantics
- Conflict resolution
```

## 🎓 Training & Knowledge Transfer

### Teaching Scenarios
```
Create a hands-on exercise for learning {concept} that:
- Starts with a broken implementation
- Guides through identifying issues
- Provides step-by-step fixes
- Explains the underlying principles
```

### Code Walkthroughs
```
Walk me through this complex Restate implementation:

```go
{complex code}
```

Explain:
- Service type choices
- State management approach
- Communication patterns
- Error handling strategy
- Determinism guarantees
```

## 📋 Template Prompts

### Service Template Generation
```
Generate a template for a {service type} with:
- Proper context usage
- State management (if applicable)
- Error handling
- Logging
- Common patterns for the service type
```

### Handler Template Generation
```
Create a handler template for {operation type} in {service type} with:
- Input/output types
- Context usage
- Error handling
- Common patterns for the operation
```

These prompts are designed to help developers and LLMs effectively work with Restate and the REA framework while maintaining the critical principles of durable execution and determinism. Use them as starting points and adapt based on specific use cases and requirements.