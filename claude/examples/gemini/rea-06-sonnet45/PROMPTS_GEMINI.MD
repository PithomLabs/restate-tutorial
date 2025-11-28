Based on the deep analysis of `DOS_DONTS_MEGA.MD` (Core Restate Go SDK) and `DOS_DONTS_REA.MD` (pithomlabs/rea framework), I have compiled a comprehensive `PROMPTS.MD`.

This file is designed for two audiences:

1. **Developers:** To copy-paste into LLMs (ChatGPT, Claude, Cursor) to get high-quality, framework-compliant code.
    
2. **LLMs:** To be used as "System Instructions" or "Context Files" to ground the model in the specific constraints of Durable Execution.
    

---

### PROMPTS.MD

Markdown

````
# Prompts for Restate Go SDK & pithomlabs/rea Framework

This document contains a curated list of prompts designed to generate resilient, deterministic, and framework-compliant code using the Restate Go SDK and the `pithomlabs/rea` wrapper. 

**Usage:** Copy these prompts into your LLM chat (ChatGPT, Claude, or IDE agents like Cursor).

---

## 1. System Context / Meta-Prompts
*Use these prompts to initialize an LLM session or define custom instructions in `.cursor/rules`.*

### The "Restate Expert" Persona
> You are an expert backend engineer specializing in **Durable Execution** using the **Restate Go SDK** and the **`pithomlabs/rea` framework**. 
> 
> Your goal is to write code that is:
> 1. **Deterministic:** Strictly separating Control Plane (Orchestration) from Data Plane (Side Effects).
> 2. **Resilient:** Handling failures gracefully using Sagas and Retries.
> 3. **Compliant:** Adhering to the `rea` framework patterns.
> 
> **Core Constraints:**
> - **NEVER** use native Go concurrency (`go func`, `chan`, `select`) outside of `restate.Run`. Use `restate.Wait` or framework combinators.
> - **NEVER** perform side effects (HTTP calls, DB writes, UUID generation, `time.Now`) directly in a handler. ALWAYS wrap them in `framework.RunDo` or `framework.SafeRun`.
> - **ALWAYS** use `framework.ServiceClient` for internal calls and `framework.NewIngressClient` for external entry.
> - **ALWAYS** use `framework.NewMutableObjectState` or `framework.NewReadOnlyObjectState` for state access.
> - **NEVER** use `restate.Sleep` inside a Virtual Object Exclusive handler (blocking). Use Delayed Sends instead.

### The "Refactoring Guard" (For Code Reviews)
> Analyze the provided Go code for violations of Restate's Durable Execution principles. Specifically look for:
> 1. Non-deterministic operations (randomness, time, I/O) not wrapped in `framework.RunDo`.
> 2. Native concurrency usage (`go routines`) mixed with Restate Context.
> 3. Context misuse (e.g., using `ctx.Sleep` inside a `Run` closure).
> 4. Deadlock risks (Virtual Object cross-calls).
> 5. Infinite retry loops (missing `TerminalError`).
> 
> If you find issues, rewrite the code using `pithomlabs/rea` primitives.

---

## 2. Service Scaffolding Prompts

### Basic Service (Stateless Orchestration)
> Generate a **Restate Basic Service** using the `rea` framework named `OrderProcessor`.
> - It should have a handler `ProcessOrder` that takes an `OrderID`.
> - The handler should orchestrate three steps: Validate Inventory, Charge Payment, and Email User.
> - Use `framework.RunDo` for the external side effects (mock the logic).
> - Ensure the service struct is properly reflected.

### Virtual Object (Stateful Entity)
> Create a **Virtual Object** named `DigitalWallet` using the `rea` framework.
> - **State:** Store a `Balance` (float64) using `framework.NewMutableObjectState`.
> - **Handlers:**
>   1. `Deposit(amount)`: Updates state.
>   2. `Withdraw(amount)`: Checks balance, throws `TerminalError` if insufficient, updates state.
>   3. `GetBalance()`: Uses `framework.NewReadOnlyObjectState` (Shared Handler).
> - **Constraint:** Ensure `Deposit` and `Withdraw` use the Exclusive Context (`ObjectContext`).

### Workflow (Long-Running Process)
> Scaffold a **Restate Workflow** named `OnboardingWorkflow` using `rea`.
> - **Goal:** Manage a 3-day user onboarding flow.
> - **Steps:**
>   1. Send Welcome Email (Side effect).
>   2. Wait for User to Verify Email (Signal).
>   3. Sleep for 24 hours (Durable Timer).
>   4. Check if Profile is Complete.
> - Use `framework.GetInternalSignal` (Promise) for the email verification step.
> - Use `restate.After` for the sleep.

---

## 3. Business Logic & Side Effects (Data Plane)

### Wrapping External API Calls
> I need to call an external API (Stripe) to charge a user inside my `ProcessPayment` handler. 
> Write the code using `framework.RunDo`. 
> - **Requirement:** The API call must be isolated in the Data Plane.
> - **Context:** Do NOT use the Restate context inside the closure. Use `restate.RunContext`.
> - **Output:** Return a `ChargeResult` struct.

### Deterministic UUIDs and Time
> I need to generate a unique Transaction ID and get the current timestamp for a database record inside a handler.
> - **Refactor:** `id := uuid.NewString()` and `ts := time.Now()`.
> - **Instruction:** Replace these with `framework.DeterministicHelpers` and `restate.Rand` (or equivalent framework time helpers) to ensure replayability.

### Implementing Sagas (Compensations)
> Write a handler `BookTrip` that reserves a Flight and a Hotel.
> - **Pattern:** Use the Saga pattern.
> - **Logic:**
>   1. Reserve Flight.
>   2. Reserve Hotel.
>   3. If Hotel fails, Cancel Flight.
> - **Implementation:** Use Go's `defer` with a compensation function. Ensure the compensation is registered *before* the action is taken. Use `restate.TerminalError` to stop retries on business failures.

---

## 4. Concurrency & Parallelism

### Fan-Out / Fan-In
> I need to fetch data from 3 different services (`UserProfile`, `Orders`, `Wishlist`) in parallel.
> - **Constraint:** Do NOT use `go func` or `sync.WaitGroup`.
> - **Solution:** Use `restate.Service[T]().RequestFuture()` to start the calls.
> - **Aggregation:** Use `restate.Wait` or `framework.Gather` (if available) to collect results.
> - **Error Handling:** If any call fails, the whole operation should fail.

### Racing (Timeouts)
> I am waiting for a service call `FraudCheck`, but I want to timeout if it takes longer than 5 seconds.
> - **Code:** Use `framework.RacePromiseWithTimeout` (or `restate.WaitFirst` with `restate.After`).
> - **Logic:** Handle the two cases: (A) Service returns result, (B) Timeout fires.

---

## 5. Signaling & Coordination

### External Webhooks (Awakeables)
> My service needs to pause and wait for a webhook from a 3rd party logistics provider.
> - **Primitive:** Use `framework.WaitForExternalSignal` (Awakeable).
> - **Output:** Show me how to generate the ID/URL to send to the provider and how to `await` the result in the handler.
> - **Context:** Explain why I should use this instead of a Promise.

### Workflow Internal Signals (Promises)
> Inside my `OrderWorkflow`, I have a parallel handler `ApproveOrder` that needs to unblock the main `Run` loop.
> - **Primitive:** Use `framework.GetInternalSignal` (Durable Promise).
> - **Action:** Show the code for the `Run` loop waiting on the promise, and the `ApproveOrder` handler resolving it.

---

## 6. State Management

### Lazy vs. Eager State
> I have a large state object `UserHistory` (2MB) that is rarely accessed, and a small `UserPreferences` object.
> - **Code:** Show how to define the handler options to enable **Lazy State** for the history and **Eager State** (default) for preferences.

### Avoiding Deadlocks
> I have two Virtual Objects: `UserA` and `UserB`. I need to transfer money from A to B.
> - **Problem:** `UserA.Send(UserB)` calling `UserB.Receive(UserA)` might cause a deadlock.
> - **Solution:** Rewrite this logic using **One-Way Sends** (`Client.Send`) or by handling the logic in a separate `TransferWorkflow`. Explain why the request-response pattern is dangerous here.

---

## 7. Migration & Refactoring Assistant

### converting Standard Go to Restate
> Here is a standard Go function:
> ```go
> func Process(data string) error {
>    db.Save(data)
>    res, _ := http.Get("[http://api.com](http://api.com)")
>    if res.StatusCode != 200 { return fmt.Errorf("error") }
>    return nil
> }
> ```
> **Task:** Convert this to a Restate handler using `pithomlabs/rea`.
> 1. Inject the Context.
> 2. Wrap DB and HTTP calls in `framework.RunDo`.
> 3. Handle errors correctly (Transient vs Terminal).

### Debugging "Journal Mismatch"
> I am getting a "Journal Mismatch" error in my local environment.
> **Diagnosis:** Explain the common causes based on the Restate constraints.
> **Checklist:**
> 1. Did I change code without versioning?
> 2. Did I use `map` iteration order?
> 3. Did I leave a `fmt.Println` or side effect outside a `Run` block?
> **Fix:** Tell me how to clear the local state or register a new version.

---

## 8. Agent-Specific Context (`AGENTS.md` Snippet)

*Copy this block into your `.cursor/rules` or Agent instructions.*

```text
# Restate & REA Framework Rules
1. **Durable Execution:** All code runs in a replayable log.
2. **Determinism:** - NEVER use `time.Now()`, `uuid.New()`, or random inside a handler body. 
   - Use `restate.Rand(ctx)` or `framework.DeterministicHelpers`.
3. **Side Effects:** - ALL I/O (Database, HTTP, Files) MUST be wrapped in `framework.RunDo(ctx, func(rc restate.RunContext) ...)`
   - Never access the outer `ctx` inside `RunDo`.
4. **Concurrency:** - No `go func`. Use `restate.Service().RequestFuture()`.
   - No `channel`. Use `restate.Promise` or `restate.Awakeable`.
5. **State:** - Virtual Objects are Single-Threaded (per key). 
   - Do not `Sleep` in exclusive handlers; use `Send` with Delay.
````