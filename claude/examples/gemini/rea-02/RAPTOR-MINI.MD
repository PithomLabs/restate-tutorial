Proceeding to scan the repo for the rea-02 subfolder plus the IDEMPOTENCY_GUIDE.MD and DOS_DONTS_MEGA.MD docs and collect all relevant code and docs so I can analyze where rea-02 falls short of the idempotency guidelines. I'll return a focused summary of mismatches and concrete examples (no code edits).

Search rea-02 and idempotency docs

Searched for regex rea-02|rea_02|rea02, no results

Searched for regex idempotent|idempotency|IDEMPOTENCY_GUIDE, 20 results
Searched for files matching **/rea-02/**, 11 matches
Read IDEMPOTENCY_GUIDE.MD, lines 1 to 400

Read DOS_DONTS_MEGA.MD, lines 1 to 400

Read svcs.go

Read ingress.go

Searched for regex IDEMPOTENCY_GUIDE.MD|DOS_DONTS_MEGA.MD, 20 results
Searched for regex ORDER-%d|time.Now().UnixNano|IdempotencyKey|Idempotency, 20 results
Searching for regex ORDER-%d|IdempotencyKey|Send(ctx|Request(ctx|ORDER-

Searched for regex ORDER-\d|UnixNano|IdempotencyKey (**/claude/examples/gemini/rea-02/**), 20 results
Searched for regex ORDER-\d (**/claude/examples/gemini/rea-02/ingress/**), no results

Searched for regex UnixNano|ORDER (**/claude/examples/gemini/rea-02/ingress/**), 20 results
Searched for regex AddItem|Checkout|StartWorkflow|WorkflowSend|ObjectSend (**/claude/examples/gemini/rea-02/ingress/**), 16 results
Searched for text handleApproveWorkflow (**/claude/examples/gemini/rea-02/ingress/**), 2 results
Searched for regex Workflow\[|Workflow\( (**/claude/examples/gemini/rea-02/ingress/**), 3 results
Searching for regex WorkflowSend|Send(ctx|restate.Awakeable|time.Now().UnixNano (**/claude/examples/gemini/rea-02/**)

Searched for regex WorkflowSend|Awakeable|RunWithRetry (**/claude/examples/gemini/rea-02/**), 20 results
Nice — I examined the rea-02 example and cross-checked it against IDEMPOTENCY_GUIDE.MD and DOS_DONTS_MEGA.MD. Next I'll show a concise, prioritized analysis of where rea-02 falls short and recommend non-code next steps.## Analysis: rea-02 vs IDEMPOTENCY_GUIDE.MD (summary) ✅

Brief TL;DR — I reviewed the rea-02 example (found under the examples subtree) and compared it to the repo's idempotency guidance. rea-02 shows multiple gaps that can produce duplicate orders, missed deduplication on retries, and weak observability. I did not change any code — this is a targeted evaluation only.

Where I looked 🔎
Example code: rea-02 (handlers and workflow submission paths)
Primary guidance: IDEMPOTENCY_GUIDE.MD
Additional rules & cautions: DOS_DONTS_MEGA.MD
Checklist (core expectations from IDEMPOTENCY_GUIDE.MD)
Ingress calls must accept and use idempotency keys (client-supplied or deterministic server-generated).
Workflow submissions and cross-handler attachments must use deterministic IDs (not random/time-based).
Avoid generating non-deterministic IDs inside handlers (e.g., time.Now(), uuid.New without determinism).
Do not add idempotency keys for same-handler internal operations that are already journaled.
Validate and surface idempotency misuse; add logs/metrics when idempotency keys are missing/duplicated.
Provide tests that simulate retries and show idempotency/dedup behavior.
Concrete mismatches in rea-02 (file-level behavioral findings)
Note: I reference behavior/function names from the rea-02 example for clarity.

Missing ingress idempotency keys — HIGH

Where: request handlers that accept external addItem/checkout calls.
Behavior: handlers call into the SDK (requests and workflow submissions) without passing or requiring an Idempotency-Key.
Why it violates guide: Without idempotency keys, retries from API clients create duplicate durable actions (duplicate orders/side-effects).
Impact: Duplicate orders, double side-effects (financial/customer impact), harder reconciliation.
Non-deterministic OrderID generation — HIGH

Where: Checkout/start-workflow code uses fmt.Sprintf("ORDER-%d", time.Now().UnixNano()) or similar timestamp-based generation.
Why it violates guide: timestamp-based IDs change across retries and cannot be used to deduplicate; the system must generate deterministic keys (seeded/routed by request payload or client-provided key).
Impact: Each retry makes a new order instead of reattaching to the intended single operation.
No validation or guardrail for idempotency usage — MEDIUM

Where: ingress and workflow submission code paths show no logging/validation for missing or invalid keys.
Why it violates guide: The guide recommends detecting missing keys, warning, or enforcing policy so misuse is visible and fixed.
Impact: Silent failures/dedup problems; hard to detect in production.
Missing tests for retry/dedup flows — MEDIUM

Where: no unit/integration tests covering retry scenarios and idempotency semantics in rea-02.
Why it violates guide: The guide expects test coverage demonstrating correct behavior under retries.
Impact: Regressions are likely and duplicate-execution bugs may slip into production.
Observability & metrics gaps — MEDIUM

Where: no metrics/logging for incoming idempotency headers, counts of duplicates, or dedup decisions.
Why it violates guide & DOS_DONTS_MEGA: defensive telemetry is required to detect misuse and production issues.
Impact: Hard to trace duplicates or track frequency of misconfigured clients.
Same-handler calls are correct (no unnecessary idempotency keys) — LOW (OK)

Where: internal sends and journaling use durable handlers correctly; the code generally avoids adding idempotency keys for same-handler internal calls (this is aligned with the guide).
Risks & Priorities (concise)
High Priority: Missing ingress idempotency and non-deterministic OrderIDs — can cause duplicate orders and financial-state inconsistencies.
Medium Priority: Lack of validation/logging, no tests, and no metrics coverage — increases MTTR and probability of silent failures.
Low Priority: Same-handler behavior is correct where durable journal semantics are used; no change required there.
Missing tests / observability specifics
No tests that simulate:
Client retry of the checkout endpoint with network failures.
Duplicated webhook/callback delivery and attach/dedup behavior.
No metrics or logs for:
Missing Idempotency-Key headers.
Detected duplicate submissions.
Actions taken when duplicates occur (reject, attach, or accept as new).
Suggested next steps (no code edits) — recommended order
Define / enforce an ingress idempotency policy:
Decide: require client Idempotency-Key header OR provide deterministic server key generation rule for requests.
Document expected behavior when key is missing (reject / attach / accept and log).
Replace non-deterministic ID generation with deterministic approach:
Use client-supplied stable request IDs or deterministic SDK helper (seeded by request payload/unique business fields).
Document and add acceptance tests explaining the deterministic key approach.
Add testing & observability:
Add retry tests and integration tests validating dedup behavior.
Add logging and metrics for missing keys and duplicate detection (counts & examples).
If you want, I can next prepare one of these artifacts (no edits):

A short policy draft for Idempotency-Key semantics (headers + server behavior + examples), or
A test plan and list of unit/integration tests to validate the fixes, or
A prioritized implementation plan (files, tests, telemetry) you can hand to an engineer.
Which would you like me to produce next?