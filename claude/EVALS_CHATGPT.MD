# Quick evaluation — **framework.go** vs standard Restate Go SDK (short, actionable)

I read **DOS_DONTS_MEGA.MD**, **AGENTS.MD**, and the uploaded **framework.go** and compared how the framework reduces boilerplate and how well it follows the Restate “dos & don’ts”. Below is a category-by-category assessment with strengths, gaps, and concrete recommendations. I reference the uploaded files as I go.  

---

## Summary (one-liner)

`framework.go` provides a solid, opinionated layer that **wraps many common Restate patterns** (typed clients, state helpers, saga primitives, idempotency checks, timers, awakeable/promise helpers) and enforces several dos from DOS_DONTS_MEGA — but it leaves a couple of important gaps (notably a stubbed ingress-client implementation and some runtime-context-type checks that need careful testing) and must be used carefully to avoid the SDK “don'ts” it tries to protect against.  

---

## Evaluation by category

### 1) State invocation

**What framework does:** exposes a `State[T]` wrapper (`NewState`, `Get`, `Set`, `Clear`) that dispatches to the appropriate Restate `Get`/`Set` based on the concrete context and emits runtime errors when used from read-only contexts. This removes repetitive type assertions and Get/Set boilerplate. 

**Strengths**

* Type-safe generic wrapper reduces duplicated `restate.Get[T]` type signatures.
* Runtime guard prevents accidental writes from shared/read-only contexts (helpful safety net given Restate rules). 

**Gaps / Risks**

* Runtime type switch still means a developer can construct `State` with wrong `ctx` at compile time; static guarantees would be nicer.
* Ensure error messages map precisely to Restate context semantics (exclusive vs shared). If the framework panics or returns generic TerminalErrors, it should map to clear messages per DOS_DONTS guidance. 

**Recommendation**

* Add constructor helpers `NewStateForObject(ctx restate.ObjectContext, key string)` and `NewStateForWorkflow(ctx restate.WorkflowContext, key string)` to get compile-time safety and remove runtime branching.

---

### 2) State management

**What framework does:** supplies `ClearAll`, `WorkflowStatus` utilities, `WorkflowTimer`, `WorkflowLoop` and `ForEach` helpers, plus guidance + defaults. These are higher-level reusable primitives over `restate.Set/Get/Clear` and `restate.Sleep/After`. 

**Strengths**

* Centralizes patterns (status tracking, loops, retries) and enforces safe defaults (max iterations, backoff).
* Encourages using `WorkflowContext`/`ObjectContext` for writes and `*SharedContext` for reads in docs and runtime checks. Aligns with DOS_DONTS recommendations. 

**Gaps**

* Lazy vs eager state loading strategy is not surfaced — framework should expose a way to declare/opt-in lazy state for heavy payloads (DOS doc recommends it). 

**Recommendation**

* Add optional flags to state helpers to opt into lazy loading and document cost/tradeoffs.

---

### 3) Interservice communication

**What framework does:** provides typed `ServiceClient`, `ObjectClient`, `WorkflowClient` wrappers and a `Call/Send/RequestFuture` API that builds `restate.WithIdempotencyKey` and `WithDelay` automatically from `CallOption`. This massively reduces repetitive option plumbing. 

**Strengths**

* Removes boilerplate in building send/request option slices, idempotency injection, delay handling — matches recommended usage patterns. 
* Encourages `RequestFuture` for concurrency by offering wrapper methods. 

**Gaps / Risks**

* Framework validates idempotency keys heuristically and logs errors but does not fail — either approach can be fine, but the choice should be configurable (strict vs permissive) per DOS guidance to avoid silent problems. 
* Must ensure code never routes internal calls through ingress (DOS says don’t use ingress client for internal calls). The framework keeps both APIs but must document the distinction. 

**Recommendation**

* Add a `ValidationMode` to the client wrappers (fail vs warn) and explicitly document: *use these client wrappers inside Restate handlers; use IngressClient only from outside Restate*.

---

### 4) Ingress client (external callers)

**What framework does:** defines an `IngressClient` and `IngressServiceClient` types but the `Call` implementation is a **stub** that returns an error telling the developer to import `restateingress`. The wrapper includes idempotency validation and logging. 

**Strengths**

* The wrapper design and idempotency checks follow DOS recommendations for ingress usage (idempotency keys on external writes). 

**Gaps (important)**

* The actual integration is *not implemented* — calling `Call` returns an error; this is a functional gap for external apps. 

**Recommendation**

* Implement the wrapper using `restateingress` (or expose a small helper installer) and include examples for `Attach` / `AttachFuture` semantics described in DOS_DONTS (attach only works if idempotency key used). Add tests demonstrating an external caller using `Attach`. 

---

### 5) Communication with the external world (webhooks, awakeables)

**What framework does:** provides typed helpers `WaitForExternalSignal`, `ResolveExternalSignal`, `RejectExternalSignal`, and uses awakeables for external coordination. Also provides `AwaitHumanApproval` that wraps an awakeable and races with a timeout. This follows DOS guidance. 

**Strengths**

* Encapsulates awakeable creation / resolution pattern to avoid manual ID errors.
* The `AwaitHumanApproval` helper abstracts common human-in-loop pattern and timeout racing (good boilerplate reduction). 

**Gaps / Risks**

* Must ensure handlers avoid waiting on awakeables inside **exclusive object handlers** (framework warns in docs but cannot prevent user calling it incorrectly). DOS explicitly warns this is a concurrency anti-pattern. 

**Recommendation**

* Add runtime guard: reject or warn if `WaitForExternalSignal` is used from `ObjectContext` (exclusive) unless caller explicitly opts-in (and warns about queuing). Or provide `WaitForExternalSignalOnWorkflow` enforced helpers.

---

### 6) Run (side effects)

**What framework does:** wraps `restate.Run` in several places (Saga compensation execution, safe step execution) and documents that side effects must be run inside `restate.Run`. Saga compensations execute `restate.Run` to make them durable. 

**Strengths**

* Enforces the “wrap external non-deterministic operations in `Run`” rule in high-level primitives (good conformity with DOS). 

**Gaps / Risks**

* Some `Run` closures may still need careful attention to avoid using the outer Restate context (DOS warns: don’t use ctx inside ctx.run). The framework should document and test that compensation handlers receive `restate.RunContext` only and that devs don't accidentally use`ctx` inside them. 

**Recommendation**

* Add utility `RunDo[T]` that accepts a function `(rc restate.RunContext) (T, error)` and makes it simpler to implement without accidentally capturing outer `ctx`.

---

### 7) Guardrails against "don'ts"

**What framework does:** implements many runtime guards (state write checks, idempotency validation, restricted public access flags, security validation structs) intended to enforce dos/don’ts. 

**Strengths**

* Helpful runtime checks reduce accidental misuse (e.g., writing state from shared handler, bad idempotency keys). 

**Gaps**

* Runtime checks are not full compile-time guarantees and logging-only modes can be silently ignored. The framework intermixes strict and permissive behavior (sometimes logs instead of returning error). Decide a default policy (fail-fast in CI, warn in dev). 

**Recommendation**

* Add a global `FrameworkPolicy` that toggles strictness: `Strict`, `Warn`, `Disabled`.

---

### 8) Saga (compensation)

**What framework does:** implements `SagaFramework` with `Add`, `Register`, `CompensateIfNeeded`, durable step persistence, retry/backoff, DLQ persistence. It enforces “persist compensation BEFORE action” and provides `SafeStep` that requires compensation registered before executing action. This mirrors DOS guidance strongly. 

**Strengths**

* Full durable saga bookkeeping with deterministic step IDs, deduplication, retry/backoff, DLQ support and automatic LIFO compensation — this directly implements the recommended pattern. 
* `SafeStep` pattern enforces compensation-before-action at the API level — excellent boilerplate reduction and safety.

**Gaps / Risks**

* Ensure compensation handlers themselves are idempotent (framework logs but cannot enforce).
* The saga persists entries in workflow state; ensure their size and retention policies are documented (workflows have retention limits). 

**Recommendation**

* Add a `ValidateCompensationIdempotent` lint helper and documentation on storage/retention sizing for saga entries.

---

### 9) Security

**What framework does:** has `SecurityConfig` with request validation flags, signing keys, HTTPS enforcement, AllowedOrigins, and `ValidationMode` (strict/permissive). This directly implements DOS guidance like “validate request identity” and “require HTTPS” for public endpoints. 

**Strengths**

* Good configuration model and sensible defaults (`DefaultSecurityConfig` enforces validation/HTTPS). 

**Gaps**

* Need runtime integration: are incoming HTTP handlers in your server wired to use `SecurityConfig` for signature verification? The code contains `RequestValidationResult` but I didn’t find the ingress HTTP middleware in the snippet. If missing, implement middleware that rejects invalid requests per `SecurityModeStrict`. 

**Recommendation**

* Provide an attachable HTTP middleware example that enforces `SecurityConfig` for ingress handlers and include tests.

---

### 10) When to use idempotency

**What framework does:** exposes `GenerateIdempotencyKey` helpers (deterministic and UUID-based), validates keys before using them, and automatically passes idempotency keys into `Send`/`Request` wrappers. This follows the DOS guidance to use idempotency keys for external and cross-handler writes.

**Strengths**

* Good helpers for deterministic keys and automatic wiring into send APIs reduce developer mental load and boilerplate. 

**Gaps**

* Document *when not to provide keys* — DOS notes that idempotency keys within the same handler execution are redundant. The framework logs but should also avoid injecting keys automatically in that case. 

**Recommendation**

* Auto-detect “same handler execution” calls and avoid adding idempotency keys, or make it opt-in.

---

### 11) Microservices orchestration

**What framework does:** `ControlPlaneService` plus typed clients and workflows provide orchestration-first primitives: idempotent calls, saga-driven compensation, durable timers, and helpers for awaiting human approvals. These map well to recommended orchestration patterns in DOS/AGENTS docs.

**Strengths**

* Clear separation: `ServiceTypeControlPlane` vs `DataPlane` and orchestration helpers simplify building orchestrators (fewer repeated `restate.*` invocations). 

**Gaps**

* Need clear examples showing when to use Basic Service vs Workflow vs Virtual Object in the framework's README (DOS has many examples; framework should mirror them). 

**Recommendation**

* Add a cookbook: “Patterns with framework.go” — sample orchestrator, sample virtual object, sample workflow.

---

### 12) Workflow automation

**What framework does:** provides `WorkflowTimer`, `PromiseRacer`, `WorkflowStatus`, `WorkflowLoop`, retry/backoff helpers and typed `WorkflowClient`. These remove repetitive wiring around `restate.After`, `restate.Promise`, `restate.WaitFirst`, etc. 

**Strengths**

* Good high-level building blocks for durable timers, races, and status reporting — aligns with workflow use-cases in DOS. 

**Gaps**

* Must expose retention policies and warn about workflow state lifetime (24h default in DOS) whenever status persistence is used. 

**Recommendation**

* Provide `WorkflowConfig` defaults and example showing how to set retention and cleanup patterns.

---

### 13) Stateless vs stateful services

**What framework does:** codifies `ServiceTypeControlPlane` vs `ServiceTypeDataPlane` and provides `ObjectClient`/`WorkflowClient` vs `ServiceClient`, nudging devs to choose correct type. Good alignment with DOS descriptions for Basic Service vs Virtual Object vs Workflow.

**Strengths**

* Helps reduce confusion and repeated explanations; enforces the right patterned API per service type.

**Gaps**

* A small runbook in the repo would make adoption frictionless: “If you need persistent per-key state → use Virtual Object; If human-in-loop or long-running → use Workflow; If high concurrency stateless → use Basic Service.” (DOS has this; replicate it). 

---

## Final verdict (practical)

* **Boilerplate reduction:** very good. `framework.go` centralizes a lot of repetitive Restate patterns (typed clients, state wrappers, saga scaffolding, timers/loops, idempotency plumbing). You’ll save many lines and reduce user errors.
* **Correctness vs DOS/DON’TS:** mostly aligned. The framework codifies many DOS recommendations (use `restate.Run` for side effects, awakeables vs promises, idempotency), but must add a few stricter runtime/compile-time guards and finish the ingress implementation.

---

## Concrete next steps I suggest (pick 1 or more)

1. Implement `IngressClient` properly via `restateingress` and add an example showing `Attach` semantics (and idempotency requirement). 
2. Add compile-time constructors `NewStateForObject`, `NewStateForWorkflow` to avoid runtime context switching. 
3. Add a global `FrameworkPolicy` toggle (strict / warn / dev) to control whether idempotency/key validation logs or fails. 
4. Add cookbook examples that mirror DOS/AGENTS snippets (orchestration, saga, awakeable-human-approval) and tests demonstrating anti-patterns (e.g., waiting on awakeable in exclusive handler should warn or error). 

---

If you want, I can:

* produce the `IngressClient` implementation using `restateingress` (I can write the concrete Go code and tests),
* or generate the `NewStateForObject/NewStateForWorkflow` change and update the doc examples,
* or add the `FrameworkPolicy` toggle and wire it into existing validations.

Which of those should I do now? (I can start with one and produce code + tests in this session.)