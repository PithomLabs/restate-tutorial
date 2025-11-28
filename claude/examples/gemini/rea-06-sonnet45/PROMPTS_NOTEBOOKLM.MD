This document, **PROMPTS.MD**, is designed to serve as a comprehensive guide for instructing LLMs (Large Language Models) or for developers performing highly efficient prompt engineering when building resilient distributed applications using the Restate Go SDK, potentially leveraging the abstraction patterns found in the `pithomlabs/rea` framework (referenced in the sources as `framework.go.txt`).

The goal is to provide **Actionable Prompt Engineering Templates** covering core durability, consistency, security, and concurrency concerns.

---

# PROMPTS.MD: Effective Context & Prompt Engineering

## I. LLM Context Engineering Preamble (Mandatory Inclusion)

This section contains critical instructions for conditioning the LLM to behave as a Restate Go expert, ensuring adherence to strict deterministic and concurrency rules outlined in the SDK documentation and best practices.

### Prompt Template: Context Setting & Constraint Enforcement

```
You are an expert Go developer specializing in distributed durable systems using the Restate Go SDK (>= v0.22.0). Your code must be production-ready, strictly deterministic, and follow all Restate Core Principles and Guardrails.

1.  **Determinism is paramount:** All code executed within a durable handler must be deterministic. Non-deterministic operations (HTTP calls, DB writes, `time.Now()`, `rand.Float64()`) **must** be wrapped in `restate.Run()` or the framework equivalent `framework.RunDo()`/`framework.SafeRun()`.
2.  **Context Misuse is forbidden:** Never use the parent `restate.Context` inside a `restate.Run` block.
3.  **Concurrency Rules (Go):** Never use standard Go concurrency primitives (`goroutines`, `channels`, `select` statements) to combine blocking Restate operations outside of `restate.Run`. Use Restate's primitives like `restate.Wait` or `restate.WaitFirst` for concurrent durable tasks.
4.  **Logging:** Use `ctx.Log()` for all logging to avoid duplication during replay.
5.  **Error Handling:** Use `restate.TerminalError` to signal non-transient, permanent business failures to stop retries immediately.

**Goal:** Generate a complete Go file (e.g., `services.go` or `ingress.go`) based on the specific request below.
```

## II. Prompts for Core Durability and Determinism

These prompts focus on the fundamental requirements for resilient Restate services.

|Prompt Category|Specific Task & Required Artifacts|Key Restate Concepts|
|:--|:--|:--|
|**P1: External Integration**|Design a Basic Service handler `InitiateShipment` that performs an external HTTP call to a shipping provider. Ensure the call and its result are logged, executed exactly once, and survive crashes.|`restate.Run()` / `framework.RunDoVoid()`, Exactly-Once Side Effects.|
|**P2: Deterministic State Generation**|Implement a function `CreateInvoiceID` that generates a UUID based on the current durable invocation. Ensure the function is deterministic upon replay.|`restate.UUID(ctx)`, Deterministic Randomness.|
|**P3: Durable Timing**|Write a Virtual Object handler `LockResource` that requires a 5-minute durable sleep after acquiring a lock. Show how to implement the sleep correctly without using `time.Sleep()`.|`restate.Sleep(ctx, duration)`, Durable Timers, Anti-Pattern Avoidance.|
|**P4: Input/Output Serialization**|Define a service handler signature and show how to enable `WithProto` codec option for both input and output serialization, illustrating production hardening for data transmission.|`restate.WithPayloadCodec()`, Codecs, Serialization.|
|**P5: Deterministic Loops**|Generate a Workflow handler that must iterate over a map (`map[string]Task`) and execute a durable RPC for each entry. Use the framework-recommended anti-pattern protection to ensure iteration order consistency.|`framework.DeterministicMap`, Non-Deterministic Iteration Guard.|

## III. Prompts for Resilience, Transactions (Sagas), and Error Handling

These prompts target complex, multi-step transaction integrity and fault tolerance.

|Prompt Category|Specific Task & Required Artifacts|Key Restate Concepts|
|:--|:--|:--|
|**R1: Bulletproof Saga Implementation**|Write a Go Workflow function `OrderFulfillmentWorkflow.Run` that executes three durable steps: `ReserveInventory`, `ProcessPayment`, and `NotifyUser`. Use **Go's `defer` statement** to register durable compensation actions (`CancelReservation`, `RefundPayment`) for the Saga pattern, ensuring rollback on `restate.TerminalError`.|Saga Pattern, `defer`, `restate.TerminalError`, Durable Compensations.|
|**R2: Compensation Idempotency**|Provide the boilerplate implementation of a compensation function `RefundPayment` that adheres to the strict requirement of being idempotent, using a pattern like the one described in the framework for external one-shot APIs.|Idempotency of Compensations, One-Shot APIs.|
|**R3: Non-Retryable Failure**|Within a `restate.Run` block, demonstrate how an external API call should signal a permanent failure (e.g., insufficient funds) back to Restate, halting further retries for that step.|Throwing `restate.TerminalError` inside `ctx.Run`.|
|**R4: Timeout Race**|Write a generic function `RaceWithTimeout` that races an asynchronous RPC call (`client.RequestFuture`) against a 10-second durable timer (`restate.After`), returning the result of the winner using `restate.WaitFirst`.|Durable Timers, `restate.WaitFirst`, Timeout Implementation.|

## IV. Prompts for Coordination and Concurrency

These prompts address communication boundaries (Internal vs. External) and parallel task execution.

|Prompt Category|Specific Task & Required Artifacts|Key Restate Concepts|
|:--|:--|:--|
|**C1: External Event Wait (Awakeable)**|Implement a Virtual Object handler `Checkout` that suspends its execution while waiting for an external payment gateway callback. Show how to generate the unique ID, pass it to an external system (via `ctx.Run`), and pause using `awakeable.Result()`.|`restate.Awakeable()`, External Event Handling, Suspension/Resumption.|
|**C2: Internal Workflow Signaling (Promise)**|Implement a Workflow handler `UserOnboarding.Run` that pauses waiting for a named, internal signal called `"profile-complete"`, which is resolved by a separate shared handler `SubmitProfile`.|`restate.Promise()`, Durable Promises, Internal Workflow Signaling.|
|**C3: Fan-Out/Fan-In Pattern**|Write a Basic Service handler that initiates three distinct, durable RPC calls to different microservices concurrently (`ServiceA`, `ServiceB`, `ServiceC`). Use `RequestFuture()` and the modern `restate.Wait` iterator pattern to collect results once all are complete.|Concurrent Tasks, `RequestFuture()`, `restate.Wait` / `WaitIter`.|
|**C4: Avoiding Deadlocks**|Explain why an exclusive Virtual Object handler should **never** call `restate.Sleep` or wait on an `Awakeable`, and provide a revised architecture using a separate Basic Service for the external wait.|Exclusive Handler Blocking, Virtual Object Consistency.|

## V. Prompts for Security and Operational Context

These prompts cover deployment constraints, authentication chains (L1-L3), and configuration hardening.

|Prompt Category|Specific Task & Required Artifacts|Key Restate Concepts|
|:--|:--|:--|
|**S1: L1/L2 Security Chain (Ingress)**|Write the Go code snippet for the Ingress HTTP handler (`ingress.go`) that enforces **L1 Security** (API Key validation) using `go-chi` middleware, extracts the UserID, and propagates this identity to the durable core via **L2 Security** (`restate.WithHeaders`).|`go-chi`, `context.WithValue`, `restate.WithHeaders`, Identity Propagation.|
|**S2: L3 Security Enforcement**|Show how to configure the Restate Server binding (`services.go`) to enforce **L3 Security**, preventing direct external calls by requiring cryptographic verification of the caller's identity using a trusted public key.|`options.RequireRequestIdentity(TRUSTED_PUBLIC_KEY)`, Request Identity Verification.|
|**S3: Idempotency Tuning (L0)**|Demonstrate how to configure a critical Workflow handler (`OrderFulfillmentWorkflow`) with an extended idempotency retention period (e.g., 7 days) to ensure exactly-once processing over long windows.|`options.WithIdempotencyRetention(duration)`, L0 Guarantee, Transaction Uniqueness.|
|**S4: Production Timeouts**|Provide the `ServiceDefinitionOption` configuration needed to adjust the `inactivity timeout` and `abort timeout` for an LLM-calling agent service to handle long-running model responses (e.g., up to 3 minutes).|`WithInactivityTimeout()`, `WithAbortTimeout()`, Handling Long Operations.|
|**S5: Deployment Hardening**|List the three critical security/networking requirements for deploying Restate services as publicly routable serverless functions (e.g., on Vercel or Cloudflare Workers).|HTTPS, HTTP/2 Load Balancer, Request Identity Verification.|

## VI. Prompts for Advanced Framework/Go SDK Usage

These prompts target nuanced usage specific to the Go environment and advanced patterns.

|Prompt Category|Specific Task & Required Artifacts|Key Restate Concepts|
|:--|:--|:--|
|**A1: Framework Safety Enforcement**|Show the **incorrect** Go code demonstrating accidental capture of the parent `ctx` inside a Run block, and provide the **safe version** using `framework.RunDo` to enforce the safety guardrail.|`RunContext` Isolation, Anti-Pattern Protection.|
|**A2: Workflow Status Query**|Implement a Workflow that updates its current status (e.g., `Phase: "Processing"`) in its state storage during the main `Run` handler. Then, write a separate shared handler `GetStatus` that safely reads this status concurrently.|`WorkflowContext` (Write), `WorkflowSharedContext` (Read), Shared/Exclusive Handlers.|
|**A3: Correct Ingress Client Usage**|Write a `main()` function snippet demonstrating the **correct** external invocation pattern using `restate.NewIngressClient` to initiate a long-running Workflow execution asynchronously (`client.Send()`).|`restate.NewIngressClient`, External Invocation, Ingress SDK.|
|**A4: Data Plane Abstraction**|Demonstrate how to define a simple, stateless external utility function (`PaymentGateway.Charge`) as a `framework.DataPlaneService` and show how the `ControlPlaneService` calls its `Execute` method.|`framework.DataPlaneService`, Separation of Concerns (Control vs. Data Plane).|
|**A5: Deprecated API Avoidance**|Show a minimal example of coordinating multiple asynchronous tasks (`AfterFuture`, `ResponseFuture`) using the **recommended modern API** (`restate.Wait` or `restate.WaitFirst`), explicitly avoiding the deprecated `restate.Select` API.|`WaitFirst`, `Wait`, Deprecated API Usage.|