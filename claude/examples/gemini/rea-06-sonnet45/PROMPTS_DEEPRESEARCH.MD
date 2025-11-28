PROMPTS.MD: Advanced Prompt and Context Engineering for Resilient Distributed Applications (Restate Go SDK & REA Framework)

1. Executive Summary: The Context-Engineered Agent Paradigm

The successful implementation of durable, resilient distributed applications, particularly those leveraging AI agents, requires moving beyond traditional prompt engineering (PE) toward comprehensive **Context Engineering (CE)**. While PE focuses narrowly on the linguistic clarity of instructions, CE addresses the optimization of the entire informational state available to the Large Language Model (LLM) during inference.[1, 2] For critical systems built on frameworks like the Restate Go SDK and the PithomLabs/rea framework [3], the LLM cannot be treated as a general code assistant; it must function as an expert constrained by the specific architectural mandates of durable execution.

1.1 Why Traditional Prompting Fails in Distributed Systems

Traditional prompting often yields code that is syntactically correct but fundamentally unsuitable for a durable execution environment. The core challenge lies in the distributed system constraints, particularly the strict requirement for **determinism**.[4] Restate guarantees reliable execution through history replay.[5] If code executions are not pure and deterministic between durable operations, the replay process will lead to state divergence, compromising fault tolerance.[4]

Vague prompts, such as "Write a Go service to handle a database operation," frequently result in generic code that fails to implement essential resilience patterns like idempotency key handling or the avoidance of non-deterministic calls (e.g., using wall-clock time or global state for workflow logic).[6] The LLM must be explicitly configured, via CE, to prioritize architectural resilience—specifically determinism and idempotency—as non-negotiable mandates for all generated code.[7]

1.2 Restate/REA as the Source of Truth for Agent State

Restate’s Durable Execution platform fundamentally changes the resilience paradigm by guaranteeing that code runs to completion, handling infrastructure failures, service restarts, and communication issues transparently.[8] The platform provides reliable execution, consistent Key/Value state per entity, and durable primitives such as Promises and Timers.[9] This eliminates the need for developers to manually write state recovery code.[10]

Restate internally leverages an event sourcing approach to track state changes over time.[5] Consequently, any service or agent built on Restate—especially through the high-level `pithomlabs/rea` abstraction layer [3]—is effectively generating code that functions as a robust state machine processing an event stream. To prevent recovery failures, the context provided to the LLM must emphasize that **all persistent state must be managed via Restate’s** **ObjectContext** or the corresponding REA abstraction, strictly forbidding volatile memory or global variables for workflow state.[9] This ensures that state recovery is automatic and transparent, as the platform recovers by replaying the history of durable operations.[5]

1.3 Audience & Usage Guide: Maximizing the Utility of This Document

This document serves as a prescriptive guide for Lead Software Engineers and Engineering Managers experienced in Go and durable architectures. It provides standardized **Context Blueprints**—defining how input policies and architectural constraints must be structured—and high-utility **Prompt Templates**—detailing the exact phrasing for requesting compliant code. The objective is to shift the development process from reactive, iterative debugging to proactive, compliance-driven generation, leveraging the LLM as a disciplined architectural assistant.

2. Foundational Context Engineering (CE) Principles for Durable Execution

Effective Context Engineering for Restate and REA applications involves establishing a strong, restrictive system persona for the LLM and optimizing the informational context for architectural compliance.

2.1 System Prompt Blueprint: Establishing the Durable Architect Persona

The System Prompt is the foundation of Context Engineering. It must define the LLM’s identity as a gatekeeper of resilience, preventing it from generating generic patterns and forcing adherence to distributed systems best practices. Complex prompts benefit from structured formats (e.g., Markdown sections) to ensure clarity and scannability.[11] By explicitly listing the core architectural risks (determinism, idempotency, state isolation) as non-negotiable mandates, every line of generated code is resilient by default.[6, 12]

The following table outlines the mandatory directives that must preface any request for Restate or REA code generation.

Table: Mandatory System Prompt Directives for Restate/REA Code Generation

|   |   |   |
|---|---|---|
|**Directive Category**|**Mandatory Instruction Component**|**Architectural Rationale**|
|**Persona & Role**|"You are a Senior Go Developer specializing in Restate Durable Execution. Prioritize architectural resilience over simple functionality."|Establishes an authoritative, technical voice focused on resilience constraints.[6, 12]|
|**Language & SDK**|"All output must be type-safe Go code using the `github.com/restatedev/sdk-go` and PithomLabs/rea abstractions."|Enforces adherence to the specific required technology stack.[3]|
|**Resilience Constraint**|"Strictly adhere to Durable Execution Constraints (Code must be deterministic and pure between Restate state/call operations)."|Prevents replay inconsistencies, which is the primary failure mode in durable execution frameworks.[4]|
|**Idempotency Mandate**|"Identify all external calls (HTTP, DB writes) and implement idempotency key mechanisms unless strictly read-only."|Ensures exactly-once business effects, critical for reliable retries in distributed transactions.[7]|
|**State Management**|"Use Restate `ObjectContext` (or the REA state abstraction) for all persistent state. Do not use volatile memory, global variables, or standard DB writers for workflow state."|Guarantees automatic recovery and state consistency through event-sourcing.[9]|
|**Output Structure**|"Provide code blocks using Go markdown, followed by a separate Resilience Analysis section explaining how the code maintains determinism and idempotency."|Enforces scannable, auditable output with required meta-analysis.[11]|

2.2 Context Structuring for Resilience Policy Enforcement

LLM-based agentic systems frequently rely on injecting extensive policy documents that encode specific business rules, such as `DOS_DONTS_MEGA.MD` and `DOS_DONTS_REA.MD`.[13] Simply including these documents (a basic Retrieval-Augmented Generation or RAG approach) is insufficient; the LLM must understand which rules carry the highest architectural weight.[14]

To optimize policy adherence, the injected context should categorize policies. Research suggests grouping rules into factual, behavioral, and conditional types, focusing on the latter category, as complex conditional specifications often drive the most intensive reasoning challenges in workflow execution.[13] By injecting policies prefixed with category tags (e.g., ``), the developer provides the LLM with a prioritized reasoning framework. The instruction, "Prioritize Conditional Policy rules when performing code generation and auditing," ensures the LLM focuses on generating robust failure handling and state transition logic (like Saga compensation steps) over simple fact adherence.

2.3 Token Budget Optimization for Codebases

LLM context limits are an inherent constraint. When generating code that interfaces with an existing microservice architecture, injecting the full source code for every dependency is wasteful and often exceeds the available token budget.[14]

The optimal Context Engineering strategy for Restate Go applications is to utilize a **Contract-First Context** approach. Instead of providing Go implementation files, developers must inject the canonical **Protocol Buffer (****.proto****) definitions** for all interacting services.[15] Protobuf files define the exact structure of data messages and the RPC contracts (methods and their signatures) necessary for generating type-safe Restate clients. This data format represents the most efficient, unambiguous, and compressed form of architectural context available, saving significant tokens while ensuring the LLM has all the necessary type information for accurate code generation.[15] The prompt strategy must explicitly constrain the LLM: "When referencing internal dependencies, strictly rely on the provided Protobuf schema for type information."

3. Mandatory Constraints: Engineering for Restate Determinism and Consistency

3.1 The Determinism Mandate: Prompts to Identify and Eliminate Non-Deterministic Operations

Durable execution platforms require that workflow code must execute deterministically so that repeating the execution with the same input yields the same side effects and state transitions.[4] Failures in determinism lead directly to execution divergence during recovery, potentially resulting in data inconsistencies and history log size bloat.[4]

Non-deterministic operations—such as reading wall-clock time (`time.Now()`), generating standard random numbers, or performing raw external I/O—must be replaced by Restate’s durable equivalents. The LLM must be taught to identify and replace these vulnerabilities with Restate primitives that capture the result in the execution history, ensuring consistent replay.[9]

Deterministic Code Generation Prompt:

```
TASK: Generate a Restate Virtual Object handler function named `InitiateDelayedVerification` for a keyed user ID object. This handler must execute the following:
1. Schedule a durable delay of exactly 10 minutes using the Restate context timer primitive.
2. After the delay, generate a random 8-digit verification code. Use **only** Restate Go SDK primitives for time manipulation and randomness to ensure determinism.
3. Store the generated code in the object's state.
```

Audit Prompt for Determinism (Pre-CI Check):

```
TASK: Review the provided Go function `CalculateSLA()` for non-deterministic calls. Specifically check for direct calls to `time.Now()`, `time.Since()`, unmanaged goroutines, and any package outside the Restate context that generates random numbers. If found, generate a list of violations and the exact lines of code to replace with Restate deterministic calls.
```

3.2 Idempotency by Design: Prompt Templates for Idempotency Key Generation and Validation Logic

Idempotency is the foundation of resilience, guaranteeing that an operation can be retried safely without unintended side effects, thus enabling reliable retries and exactly-once business effects.[7] While Restate ensures that internal communication (RPC) is reliable and avoids duplicates [8], developers must implement idempotency logic for interactions with external, mutable systems (e.g., legacy APIs or external payment gateways).[7]

The robust implementation of idempotency involves: 1) generating a unique key, 2) checking if the key has been processed, and 3) atomically processing the request and storing the result keyed by the unique identifier.[7] For Restate, the idempotency key should often be derived from the Virtual Object's key or a unique identifier within the durable execution logic.

Idempotency Implementation Prompt Template:

```
TASK: Generate a Restate Virtual Object handler `ProcessInvoice(ctx restate.ObjectContext, req InvoiceRequest)`. This function is required to call an external, non-idempotent legacy billing system. Generate the Go code that enforces idempotency using the following logic:
1. Extract `req.InvoiceId` from the request as the unique idempotency key.
2. Use `ctx.Get("invoice_status_{key}")` to check for prior completion. If found, return the stored result immediately.
3. If incomplete, call the external API (`ExternalBillingService.Charge()`).
4. Atomically store the final result (or error) and update the status using `ctx.Set` keyed by the transaction ID.
LANGUAGE: Go (Restate SDK)
```

3.3 Reliable Communication Context: Auditing Against Retry Amplification

Restate inherently handles resilient RPC, including automatic retries until success, with no duplication of calls.[8] This baked-in resilience simplifies development significantly. However, developers sometimes fall into the pitfall of manually wrapping Restate client calls in their own retry loops (e.g., using a `for` loop with exponential backoff). This practice is a severe anti-pattern known as retry amplification, where a single transient failure in a deep call chain can trigger an exponential surge of retries across multiple layers, overwhelming the entire distributed system.[7, 16]

The LLM must be explicitly directed to audit against this manual retry logic, relying instead on Restate’s guarantees, which can also be configured with specific retry policies where necessary.[17]

Audit Prompt: Retry Amplification Check:

```
CONTEXT: Provided below is a Restate service function where a developer mistakenly wrapped a Restate client call in a 5-attempt retry loop.
CODE:
TASK: Analyze the provided Go function `ExecutePaymentChain`. Identify the redundant manual retry logic. Explain concisely why this violates the resilience philosophy of Restate (citing potential retry amplification). Refactor the code to remove the manual retry loop, relying solely on Restate’s guaranteed RPC resilience.
```

4. Prompt Engineering for Restate Go Service and Workflow Generation

4.1 Durable Service Scaffolding Prompts

Standardized prompts accelerate the creation of compliant Restate services by enforcing the correct structural boilerplate, including service registration and the definition of durable state structures.

Prompt: Generating a new Restate Virtual Object (`KeyedRouter`)

```
TASK: Generate the complete boilerplate, including `main()` function and registration, for a Restate Virtual Object service named `SessionManager`. The service key is `UserID` (string). Define a state struct `SessionState` containing `LastActive time.Time` (ensure deterministic time usage) and `IsAuthenticated bool`. Include two object-scoped methods: `StartSession(ctx ObjectContext, req StartRequest)` and `TerminateSession(ctx ObjectContext, req TerminateRequest)`.
LANGUAGE: Go (Restate SDK)
```

Prompt: Generating a Restate Workflow (`Workflow` service) structure.

```
TASK: Generate a fully commented Restate Workflow scaffolding (`ShipmentWorkflow`) that takes a `ShippingAddress` input. The workflow must utilize three asynchronous steps (Invoking `WarehouseService.Allocate`, `CarrierService.Book`, and `NotificationService.SendAlert`). Structure the workflow using sequential, durable calls to facilitate implementation of the Saga pattern failure logic later.
LANGUAGE: Go (Restate SDK)
```

4.2 State and Timer Prompts

Restate manages durable promises and futures, allowing long-running code to suspend while awaiting an asynchronous result (e.g., a webhook or timer completion) and resume reliably exactly where it left off, regardless of server crashes.[9, 10]

Prompt: Durable Promises and Futures.

```
TASK: In the `ShipmentWorkflow` (defined previously), after successfully booking the carrier, generate Go code to wait for an external webhook confirmation using a Durable Promise.
1. Define the necessary code to create and wait on an `ctx.NewAwakeable`.
2. Demonstrate how the workflow reliably suspends until the promise is resolved externally (simulating receipt of the `tracking_id`), highlighting that this process ensures the workflow state is maintained and resumes automatically without consuming thread resources while suspended.[9]
```

4.3 Advanced Orchestration Prompts: The Saga Pattern

In microservice architectures, maintaining transactional consistency across multiple independent databases is achieved using design patterns like the Saga.[18] A Saga is a sequence of local transactions where, if any step fails, the saga performs compensating transactions to reliably undo the completed steps.[18] Restate Workflows are the ideal durable orchestration layer because their execution history is never lost, guaranteeing that the compensating logic will always be executed if a failure occurs.[4]

Saga Implementation Prompt (Orchestration):

```
CONTEXT: The ReservationSaga requires three steps: 1. PaymentService.Charge, 2. InventoryService.Allocate, 3. NotificationService.Confirm. The compensating transaction for Step 2 failure is PaymentService.Refund.
TASK: Implement the Restate Workflow `ReservationSaga` in Go. Generate the orchestration logic. If `InventoryService.Allocate` fails (returns a non-transient error), the code must reliably and durably execute the compensating transaction `PaymentService.Refund` before terminating the workflow and logging the failure. Ensure that the durable state tracks which steps have completed before compensation is invoked.
```

5. REA Framework Abstraction and LLM Agent Prompts

The `pithomlabs/rea` framework is described as a high-level abstraction layer intended for building robust, durable distributed applications, particularly durable AI agents, on top of the Restate Go SDK.[3, 10] Since this abstraction simplifies the complexity of explicit Restate calls, prompts targeting REA must focus on the high-level Agent Domain Specific Language (DSL) that manages state transitions and durable steps, ensuring the underlying Restate durability is leveraged without boilerplate.

5.1 Generating High-Level REA Agent Logic

The following prompts enforce the use of the REA abstraction for durability, treating the agent as a resilient state machine.

REA Agent State Definition Prompt:

```
CONTEXT: You are generating code using the REA framework for Durable Agents. All persistent agent state must be defined in the `REA.AgentState` struct. All execution logic must be defined using `REA.DurableStep(context, name, func)`. State transitions must use `REA.Transition(newState)`.
TASK: Define the Go Agent State and the initial durable step, `DraftGenerationStep`, for an agent that generates technical documentation.
1. The state must capture the LLM's full output text (`string DocumentContent`) and the current retry count for the external LLM API call (`int RetryCount`).
2. Implement the `DraftGenerationStep` which makes an external LLM call (simulated by a Restate RPC to `LLMService.GenerateDraft`).
CONSTRAINT: Ensure the retry count is incremented durably within the state before retrying.
```

5.2 Tooling and Data Contract Prompts

The Restate ecosystem heavily promotes a contract-first approach using Protocol Buffers and tooling like Buf for managing type safety and client generation.[15] This process is integral to efficient Context Engineering, as the Protobuf definition serves as the canonical source of truth for all communication contracts.

Prompt: Generating Type-Safe Protobuf Definitions.

```
TASK: Generate a Protobuf message definition `DurableAgentRequest` intended for use with a Restate service. The message must contain:
1. A unique, immutable identifier (`string agent_id`).
2. An optional field for distributed tracing context (`bytes trace_context`).
3. A field for the full business policy document (`string agent_policy`) to be used as context (RAG/CE input) by the agent.
4. Ensure the schema is compatible with the latest Restate Go SDK code generation.
```

Prompt: Generating `buf.gen.yaml` Configuration.

```
TASK: Generate the complete `buf.gen.yaml` configuration file for a new Restate service module.
1. The configuration must ensure that `protoc-gen-go-restate` is correctly invoked to generate both server interfaces and client stubs from the input directory.
2. Ensure the standard Go Protobuf compiler (`buf.build/protocolbuffers/go`) is also included.
3. The output files must target the current directory (`.`) and respect source relativity.
```

6. Validation and Compliance Prompts (Structured Output)

For critical production systems, auditing LLM-generated code must be automated. Free-form text summaries of compliance status are difficult to parse in CI/CD pipelines. The most effective method is utilizing structured output formats, such as JSON Schema (Prompt Specs).[19] This practice transforms the LLM into a highly reliable, machine-readable auditor, providing rapid schema evolution without requiring entire prompt rewrites.[19]

6.1 Implementing Structured Compliance Checks via Prompt Specs

By mandating that the LLM output conforms to a target JSON schema, the developer gains precise control over the compliance report structure. This is ideal for domain-specific classification, safety checks, and quality assessment.[19]

Table: Prompt Specs for Restate Code Quality and Compliance

|   |   |   |
|---|---|---|
|**Compliance Check**|**Prompt Input Fields**|**Prompt Output Fields (JSON Schema)**|
|**Determinism Audit**|`go_service_code` (string), `restate_service_type` (enum)|`is_deterministic` (boolean), `violations_found` (list of strings - citing line number), `suggested_fix` (string), `confidence_score` (float)|
|**Idempotency Review**|`go_handler_function` (string), `external_api_calls` (list of strings)|`idempotency_level` (enum: Full, Partial, None), `missing_key_context` (boolean), `risk_assessment` (string - max 100 words) [7]|
|**Policy Adherence Check**|`code_snippet` (string), `policy_document` (string)|`policy_violation_count` (integer), `critical_violations` (list of strings - citing rule ID and line number), `compliance_rating` (enum: A, B, C, D) [13, 19]|

6.2 Prompt Example using JSON Schema Output for Auditing

This example demonstrates how to enforce strict JSON output for an audit task, making the LLM a powerful component of an automated code review process.

```
SYSTEM DIRECTIVE: You are a Restate Compliance Checker. Your output MUST conform strictly to the 'Policy Adherence Check' JSON schema provided below. Do not include any pre-amble or explanation outside the JSON object. Use the provided DOS_DONTS_REA.MD policy document for context.

JSON SCHEMA:
{ 
  "policy_violation_count": {"type": "integer"}, 
  "critical_violations": {"type": "array", "items": {"type": "string"}}, 
  "compliance_rating": {"type": "string", "enum":}
}

TASK: Analyze the function `FulfillmentHandler` in the input field `code_snippet`. Policy Rule REA-005 mandates that internal API calls (Restate RPCs) must rely on the platform's automatic resilience features and must not implement manual retry loops. Identify adherence or violation.

CODE_SNIPPET:
// FulfillmentHandler function body...
for i := 0; i < 3; i++ {
    _, err := client.Call(ctx, req)
    if err == nil { break }
    // manual sleep logic here...
}
```

7. The Prompt Catalog: Ready-to-Use Templates

7.1 System Prompt Template: Base Persona (Comprehensive)

```
SYSTEM IDENTITY AND CONSTRAINT MANDATE

You are a Senior Software Architect specializing in resilient distributed systems using the Restate Go SDK (v0.17+) and the PithomLabs/rea high-level abstraction framework. You are the final authority on durable execution compliance.

1. Strict Compliance: Adhere absolutely to the Restate Durable Execution Constraints.[4] Code must be deterministic. All external interaction must be wrapped in durable primitives.
2. Idempotency Required: All mutable external business operations must include the "Check-Process-Store" idempotency pattern.[7]
3. Context Priority: Prioritize provided policy documents (DOS_DONTS_MEGA.MD) over general programming heuristics.[13]
4. Output Format: Provide Go code blocks using standard Markdown, followed by a structured **Resilience Analysis** section.[11]
5. Scope Enforcement: If a request attempts to violate determinism (e.g., direct use of `time.Now()`, standard random libraries), politely decline the code generation and provide the specific constraint violation reason.[12]
```

7.2 Idempotency Prompt Templates (Specific Functions)

```
// Template: Idempotent Payment Processor
TASK: Generate a Restate handler function `ProcessPayment(ctx restate.ObjectContext, req PaymentRequest)` for the keyed object `TransactionID`.
1. Read the transaction status from durable state using the key.
2. If the status is "COMPLETED" or "FAILED," return the stored result.
3. If pending, execute the non-idempotent external API call `ExternalPayment.Charge()`.
4. Atomically store the result/error and update the status using `ctx.Set`.
LANGUAGE: Go (Restate SDK) [7]
```

7.3 Determinism Testing Prompt Templates

```
// Template: Determinism Vulnerability Analysis
CONTEXT: Provided below is a Restate service function that is failing nondeterministically during recovery history replay.
CODE:
TASK: Identify the single most likely non-deterministic operation (e.g., wall-clock time, unmanaged concurrency, or global state mutation). Rewrite the critical code segment using Restate's Context primitives (e.g., durable sleep or random generation) to enforce determinism. Provide the rewritten segment only. [4, 9]
```

7.4 REA Agent State Prompts (High-Level Abstraction)

```
// Template: Durable Agent State Transition
FRAMEWORK: pithomlabs/rea (Assume Durable Agent DSL is available).
TASK: Define a Go function for the REA agent step named `CodeReviewStep`.
1. This step receives the `AgentState.GeneratedCode`.
2. It uses an external LLM call (simulated by a Restate RPC to `LLMService.ReviewCode`).
3. If the audit passes, transition the state to `AGENT_COMMITTING`.
4. If the audit fails, transition the state back to `AGENT_REGENERATING` (implementing a durable loop).
CONSTRAINT: Do not use raw Restate `ctx.Call`. Use the REA abstraction for durability and state transitions. [10]
```

7.5 Debugging Prompts for Durable Execution Failures

```
// Template: NonDeterministicError Debugging Assistant
ERROR: I received a Restate error: `NonDeterministicError: History divergence detected at line 145.`
CODE CONTEXT: [Insert 20 lines of Go code surrounding line 145 here]
TASK: Analyze the provided code snippet at and around line 145. List three specific code patterns that could lead to a NonDeterministicError (e.g., accessing unmanaged concurrency, external API calls, or reading non-durable environment variables). Provide the most probable root cause and the exact required fix using Restate best practices. [4]
```

Conclusions and Recommendations

The successful development of resilient distributed applications using the Restate Go SDK and the REA framework relies heavily on disciplined Context Engineering. The analysis demonstrates that the LLM must be tightly constrained and provided with a highly optimized context that mirrors the architectural requirements of durable execution.

The most critical factor for resilience is the enforcement of **determinism**. By using highly structured System Prompts that mandate the use of Restate's deterministic primitives for time and randomness, developers can mitigate the primary failure mode associated with replay-based durable execution platforms.[4, 9]

Furthermore, efficiency is achieved by prioritizing **contract-first context engineering**, relying on compressed formats like Protocol Buffers instead of full codebases to define service dependencies.[15] Finally, adopting **structured compliance checks** via JSON schema output allows the LLM to function as an automated, machine-readable compliance auditor, ensuring generated code adheres to complex policies like idempotency requirements and the Saga pattern compensation logic.[18, 19] This comprehensive approach elevates the LLM from a simple code generator to an integral component of the resilient development lifecycle.

--------------------------------------------------------------------------------

1. Effective context engineering for AI agents - Anthropic, [https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents](https://www.google.com/url?sa=E&q=https%3A%2F%2Fwww.anthropic.com%2Fengineering%2Feffective-context-engineering-for-ai-agents)

2. Context Engineering Guide, [https://www.promptingguide.ai/guides/context-engineering-guide](https://www.google.com/url?sa=E&q=https%3A%2F%2Fwww.promptingguide.ai%2Fguides%2Fcontext-engineering-guide)

3. github.com/PithomLabs/rea v0.0.0-20251120214118-fd198f39a33d on Go - Libraries.io - security & maintenance data for open source software, [https://libraries.io/go/github.com%2FPithomLabs%2Frea](https://www.google.com/url?sa=E&q=https%3A%2F%2Flibraries.io%2Fgo%2Fgithub.com%252FPithomLabs%252Frea)

4. Common Pitfalls with Durable Execution Frameworks, like Durable Functions or Temporal | by Chris Gillum | Medium, [https://blog.cgillum.tech/common-pitfalls-with-durable-execution-frameworks-like-durable-functions-or-temporal-eaf635d4a8bb](https://www.google.com/url?sa=E&q=https%3A%2F%2Fblog.cgillum.tech%2Fcommon-pitfalls-with-durable-execution-frameworks-like-durable-functions-or-temporal-eaf635d4a8bb)

5. Durable Execution: This Changes Everything : r/programming - Reddit, [https://www.reddit.com/r/programming/comments/1j9ncni/durable_execution_this_changes_everything/](https://www.google.com/url?sa=E&q=https%3A%2F%2Fwww.reddit.com%2Fr%2Fprogramming%2Fcomments%2F1j9ncni%2Fdurable_execution_this_changes_everything%2F)

6. How to write better prompts for AI code generation - Graphite, [https://graphite.com/guides/better-prompts-ai-code](https://www.google.com/url?sa=E&q=https%3A%2F%2Fgraphite.com%2Fguides%2Fbetter-prompts-ai-code)

7. Error handling in distributed systems: A guide to resilience patterns - Temporal, [https://temporal.io/blog/error-handling-in-distributed-systems](https://www.google.com/url?sa=E&q=https%3A%2F%2Ftemporal.io%2Fblog%2Ferror-handling-in-distributed-systems)

8. Key Concepts - Restate docs, [https://docs.restate.dev/foundations/key-concepts](https://www.google.com/url?sa=E&q=https%3A%2F%2Fdocs.restate.dev%2Ffoundations%2Fkey-concepts)

9. restatedev/restate: Restate is the platform for building resilient applications that tolerate all infrastructure faults w/o the need for a PhD. - GitHub, [https://github.com/restatedev/restate](https://www.google.com/url?sa=E&q=https%3A%2F%2Fgithub.com%2Frestatedev%2Frestate)

10. Building an Unbreakable AI Agent: Porting Code2Tutorial to Go with ..., [https://medium.com/@pithomlabs/building-an-unbreakable-ai-agent-porting-code2tutorial-to-go-with-restate-cecab8589ed1](https://www.google.com/url?sa=E&q=https%3A%2F%2Fmedium.com%2F%40pithomlabs%2Fbuilding-an-unbreakable-ai-agent-porting-code2tutorial-to-go-with-restate-cecab8589ed1)

11. The Ultimate Guide to Prompt Engineering in 2025 | Lakera – Protecting AI teams that disrupt the world., [https://www.lakera.ai/blog/prompt-engineering-guide](https://www.google.com/url?sa=E&q=https%3A%2F%2Fwww.lakera.ai%2Fblog%2Fprompt-engineering-guide)

12. Using the system role - Amazon Nova - AWS Documentation, [https://docs.aws.amazon.com/nova/latest/userguide/prompting-system-role.html](https://www.google.com/url?sa=E&q=https%3A%2F%2Fdocs.aws.amazon.com%2Fnova%2Flatest%2Fuserguide%2Fprompting-system-role.html)

13. Analyzing and Internalizing Complex Policy Documents for LLM Agents - arXiv, [https://arxiv.org/html/2510.11588v1](https://www.google.com/url?sa=E&q=https%3A%2F%2Farxiv.org%2Fhtml%2F2510.11588v1)

14. Context Engineering: The New Operating System for AI Agents | by Sylvester Ranjith Francis, [https://medium.com/@sylvesterranjithfrancis/context-engineering-the-new-operating-system-for-ai-agents-3bb5d417a158](https://www.google.com/url?sa=E&q=https%3A%2F%2Fmedium.com%2F%40sylvesterranjithfrancis%2Fcontext-engineering-the-new-operating-system-for-ai-agents-3bb5d417a158)

15. Code Generation - Restate docs, [https://docs.restate.dev/develop/go/code-generation](https://www.google.com/url?sa=E&q=https%3A%2F%2Fdocs.restate.dev%2Fdevelop%2Fgo%2Fcode-generation)

16. Designing Resilient Event-Driven Systems at Scale - InfoQ, [https://www.infoq.com/articles/scalable-resilient-event-systems/](https://www.google.com/url?sa=E&q=https%3A%2F%2Fwww.infoq.com%2Farticles%2Fscalable-resilient-event-systems%2F)

17. restate package - github.com/restatedev/sdk-go - Go Packages, [https://pkg.go.dev/github.com/restatedev/sdk-go](https://www.google.com/url?sa=E&q=https%3A%2F%2Fpkg.go.dev%2Fgithub.com%2Frestatedev%2Fsdk-go)

18. Saga Design Pattern - Azure Architecture Center | Microsoft Learn, [https://learn.microsoft.com/en-us/azure/architecture/patterns/saga](https://www.google.com/url?sa=E&q=https%3A%2F%2Flearn.microsoft.com%2Fen-us%2Fazure%2Farchitecture%2Fpatterns%2Fsaga)

19. LLM Evaluation Prompt Specs - Fiddler | Documentation, [https://docs.fiddler.ai/observability/llm/llm-evaluation-prompt-specs](https://www.google.com/url?sa=E&q=https%3A%2F%2Fdocs.fiddler.ai%2Fobservability%2Fllm%2Fllm-evaluation-prompt-specs)