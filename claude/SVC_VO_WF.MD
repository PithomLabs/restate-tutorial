The Restate platform offers three distinct service types—Basic Service, Virtual Object, and Workflow—each optimized for different concerns regarding state, concurrency, and longevity.

Here is a detailed breakdown of when to use each service type, including specific use cases and the rationale for that choice, drawing comprehensively from the sources.

***

## 1. Basic Service (Stateless Service)

The Basic Service is the most straightforward service type, acting as a collection of **independent stateless handlers**.

| Characteristic | Description |
| :--- | :--- |
| **State & Identity** | **None**. They do not share state between requests. |
| **Concurrency** | **Unlimited parallel execution**. |
| **Context Type** | Uses the basic `Context` type. |

### Key Use Cases and Rationale

| Use Case | Why Use Basic Service |
| :--- | :--- |
| **API Calls & Background Jobs** | **Rationale:** They are ideal for business logic, data processing, and API integrations that do not require state persistence or concurrency management within Restate. |
| **Task Parallelization / Fan-Out Fan-In** | **Rationale:** Used to schedule multiple tasks asynchronously and gather the results, ensuring **high concurrency and horizontal scaling**. The ability to scale horizontally with high concurrency is a key benefit. |
| **Sagas / Distributed Transactions** | **Rationale:** Basic Services can be used as the orchestration point for Sagas, which implement resilient compensation (rollback) logic for multi-service transactions in case of terminal failures. |
| **Durable Webhooks / ETL** | **Rationale:** They function as durable webhook processors, persisting incoming events and ensuring they are **processed exactly once** using idempotency keys. |
| **Microservice Orchestration** | **Rationale:** Serves as the main orchestrator for services where the orchestration steps themselves are atomic and don't need persistent state *between* different key invocations. |

***

## 2. Virtual Object (Durable Object)

A Virtual Object models a **stateful entity** identified by a unique key. It provides actor-like semantics with built-in consistency guarantees.

| Characteristic | Description |
| :--- | :--- |
| **State & Identity** | **Isolated persistent K/V state per object key**. State is retained indefinitely (until explicitly cleared). |
| **Concurrency** | **Single writer per key** (exclusive handlers use `ObjectContext`) plus **concurrent readers** (shared handlers use `ObjectSharedContext`). Calls execute in order of arrival, serially. |
| **Context Type** | `ObjectContext` (exclusive, read/write state) and `ObjectSharedContext` (shared, read-only state). |

### Key Use Cases and Rationale

| Use Case | Why Use Virtual Object |
| :--- | :--- |
| **Stateful Entities (Actors/Digital Twins)** | **Rationale:** Ideal for modeling real-world entities like **user accounts, shopping carts, or chat sessions**. The built-in K/V state is retained permanently and automatically journaled alongside execution steps. |
| **Consistent State Machines** | **Rationale:** Guarantees **single-writer consistency per object key**. This prevents race conditions, concurrent/lost writes, and ensures that state updates are always consistent with the durable execution logic. |
| **Stateful Event Processing** | **Rationale:** When invoked via Kafka, the Kafka key maps directly to the Virtual Object key, enabling **queue-per-key ordering guarantees**. This is necessary for event-driven state machines and processing pipelines where aggregates or joins are needed. |
| **AI Agents and Session Management** | **Rationale:** Used to manage **persistent memory and context** (conversation history) across multi-turn interactions for AI agents. The exclusive handler ensures only one message is processed at a time per session ID, ensuring consistency. |
| **Distributed Coordination** | **Rationale:** Can model distributed locks, semaphores, or leases, as they offer state that is guaranteed to be consistent and is scoped per entity. |

***

## 3. Workflow

A Workflow is a long-lived process designed for **orchestrating multi-step operations** with a guaranteed exactly-once execution per ID.

| Characteristic | Description |
| :--- | :--- |
| **State & Identity** | **Isolated persistent state per workflow instance** (workflow ID). State and metadata are retained only for the configured retention period (default 24 hours). |
| **Concurrency** | A **single `run` handler** executes the main logic exactly once per ID. Additional **shared handlers** (for signals/queries) run concurrently with `run`. |
| **Context Type** | `WorkflowContext` (exclusive, for `run` handler) and `WorkflowSharedContext` (shared, for signals/queries). |

### Key Use Cases and Rationale

| Use Case | Why Use Workflow |
| :--- | :--- |
| **Long-Running Orchestration** | **Rationale:** Designed to coordinate complex, multi-step processes like user onboarding or signup flows. The **`run` handler executes exactly once** for a given ID, making it ideal for processes that should never be duplicated. |
| **Human-in-the-Loop Processes** | **Rationale:** Necessary for processes requiring interaction (approvals, reviews, manual steps). Workflows use **Durable Promises** (for internal signaling) or **Awakeables** (for external systems) to pause execution and wait for events. |
| **Time-Based Coordination** | **Rationale:** Critical for implementing **long-running durable timers, sleeps, or timeouts** that survive crashes and service restarts. When running on FaaS, the workflow handler suspends while waiting for timers or events, saving costs. |
| **Event-Driven Workflows** | **Rationale:** Allows for complex control flow (loops, conditions) around durable steps. Unlike Basic Services, workflows manage their own dedicated lifecycle and state retention. |
| **Queryable Process Status** | **Rationale:** Handlers using `WorkflowSharedContext` can expose state and completion status concurrently while the main `run` handler is executing or suspended, allowing external systems to query the current progress. |


## 4. Code

The Restate platform classifies services into three types: **Basic Service**, **Virtual Object**, and **Workflow**. The choice depends on requirements for state management, concurrency control, and process longevity.

Below, I itemize the use cases, rationale, and corresponding GoLang code (where available in the sources) for each service type.

***

## 1. Basic Service (Stateless Service)

The Basic Service is for executing **stateless business logic** and orchestrating tasks with high concurrency.

| Use Case | Rationale | GoLang Code Example |
| :--- | :--- | :--- |
| **Microservice Orchestration** | Used for orchestrating complex business logic by calling multiple services sequentially. Durable Execution ensures the request runs to completion, retrying transient errors at each step. | **Subscription Service `Add` Handler (Sequential durable steps):** [Go] ```func (SubscriptionService) Add(ctx restate.Context, req SubscriptionRequest) error { // ... paymentId := restate.UUID(ctx).String() payRef, err := restate.Run(ctx, func(ctx restate.RunContext) (string, error) { return CreateRecurringPayment(req.CreditCard, paymentId) }, restate.WithName("pay")) if err != nil { return err } for _, subscription := range req.Subscriptions { _, err := restate.Run(ctx, func(ctx restate.RunContext) (string, error) { return CreateSubscription(req.UserId, subscription, payRef) }, restate.WithName(fmt.Sprintf("add-%s", subscription))) if err != nil { return err } } return nil }``` |
| **Task Parallelization / Fan-Out Fan-In** | Used to execute multiple durable operations (`ctx.run` or RPCs) simultaneously to improve performance. Restate logs the order of completion for deterministic replay. | **Fan-Out Worker `Run` Handler (Parallel tasks using `restate.Wait`):** [Go] `// Fan out the subtasks - run them in parallel subtaskFutures := make([]restate.Future, 0, len(subtasks)) for _, subtask := range subtasks { subtaskFutures = append(subtaskFutures, restate.Service[SubTaskResult](ctx, "FanOutWorker", "RunSubtask").RequestFuture(subtask)) } // Fan in - Aggregate the results subResults := make([]SubTaskResult, 0, len(subtasks)) for fut, err := range restate.Wait(ctx, subtaskFutures...) { // ... error handling and result aggregation ... }` |
| **Sagas / Distributed Compensation** | Used to implement resilient compensation (rollback) logic for multi-service transactions when a non-transient error (`TerminalError`) occurs. | **Booking Workflow `Run` Handler (Compensation using `defer`):** [Go] `func (BookingWorkflow) Run(ctx restate.Context, req BookingRequest) (err error) { var compensations []func() (restate.Void, error) defer func() { if err != nil { for _, compensation := range slices.Backward(compensations) { if _, compErr := compensation(); compErr != nil { err = compErr } } } }() compensations = append(compensations, func() (restate.Void, error) { // Compensation logic (CancelFlight) ... }) if _, err = restate.Run(ctx, // ... BookFlight logic ... ); err != nil { return err } // ... append more compensations and run subsequent steps ... return nil }` |
| **Durable Webhooks / ETL** | Acts as a durable processor for incoming events (like webhooks or Kafka events), ensuring they are processed exactly once. | *No specific Go code example is provided for a generic webhook handler, but the implementation relies on the Basic Service structure combined with Durable Execution.* |

***

## 2. Virtual Object (Durable Object)

A Virtual Object is used to model a **stateful entity** identified by a key, offering **single-writer consistency** and permanent state storage.

| Use Case | Rationale | GoLang Code Example |
| :--- | :--- | :--- |
| **Stateful Entities (User Accounts, Shopping Carts)** | Ideal for modeling entities with isolated, persistent K/V state that must survive crashes and restarts. Uses exclusive handlers (`ObjectContext`) for read/write access and serial execution to ensure data consistency per key. | **User Account `UpdateBalance` Handler (Exclusive, Read/Write State):** [Go] `func (UserAccount) UpdateBalance(ctx restate.ObjectContext, amount float64) (float64, error) { balance, err := restate.Get[float64](ctx, "balance") // ... error handling ... newBalance := balance + amount if newBalance < 0.0 { return 0.0, restate.TerminalError(errors.New("insufficient funds")) } restate.Set(ctx, "balance", newBalance) return newBalance, nil }` |
| **AI Agents and Session Management** | Used to implement stateful AI agents by persisting memory, conversation history, and context using the K/V store. The single-writer guarantee maintains chat integrity across turns. | *The provided sources do not include a Go example for the AI Session Provider, but the underlying state logic is the same as the `UserAccount` example above, using `restate.ObjectContext` to manage state.* |
| **Rate Limiting** | Provides isolated rate limit counters per key (e.g., user ID or endpoint), combining state (`tokens`) with durable timers (`restate.Sleep`) to implement algorithms like the Token Bucket, ensuring resilience and correctness. | **Limiter Service `ReserveN` Handler (State management with durable steps):** [Go] `func (Limiter) ReserveN(ctx restate.ObjectContext, req types.ReserveRequest) (types.Reservation, error) { // ... get state ... // ... calculations wrapped in restate.Run to get deterministic time // ... if reservation is successful and update is needed restate.Set(ctx, "state", result.LimiterState) return result.Reservation, nil }` |

***

## 3. Workflow

A Workflow is a long-lived process optimized for **orchestrating multi-step operations** that require waiting for external interaction or durable timers, with an exactly-once guarantee for the main execution thread (`Run` handler).

| Use Case | Rationale | GoLang Code Example |
| :--- | :--- | :--- |
| **Human-in-the-Loop Workflows & Approvals** | Workflows can pause execution indefinitely using **Durable Promises** (for internal signaling) or **Awakeables** (for external systems), and suspend the handler to save FaaS costs. | **Signup Workflow `Run` Handler (Waiting for Promise resolution):** [Go] `func (SignupWithSignalsWorkflow) Run(ctx restate.WorkflowContext, user User) (bool, error) { // ... send verification email step ... // Wait for user to click verification link clickedSecret, err := restate.Promise[string](ctx, "email-verified").Result() if err != nil { return false, err } return clickedSecret == secret, nil } func (SignupWithSignalsWorkflow) VerifyEmail(ctx restate.WorkflowSharedContext, req VerifyEmailRequest) error { // Resolve the promise to continue the main workflow return restate.Promise[string](ctx, "email-verified").Resolve(req.Secret) }` |
| **Durable Timers and Scheduling (Cron Jobs)** | Used for long-running pauses, scheduling repeating tasks, and implementing complex logic (e.g., loops with delays). Timers survive crashes and suspend execution, resuming at the exact remaining time. | **Signup Workflow `Run` Handler (Racing Promise against Timers):** [Go] `clickedPromise := restate.Promise[string](ctx, "email-verified") verificationTimeoutFuture := restate.After(ctx, 24*time.Hour) for { reminderTimerFuture := restate.After(ctx, 15*time.Second) resFut, err := restate.WaitFirst(ctx, clickedPromise, reminderTimerFuture, verificationTimeoutFuture) // ... switch on resFut to handle promise result, reminder email, or timeout ... }` |
| **Complex Multi-Step Flows** | The combination of a single exactly-once `Run` handler, durable state persistence, and native looping/conditional constructs (`if/else`, `while`) enables complex orchestration flows that are highly observable and resilient. | **Workflow Structure:** [Go] `type MyWorkflow struct{} func (MyWorkflow) Run(ctx restate.WorkflowContext, req string) (string, error) { // implement the workflow logic here return "success", nil } func (MyWorkflow) InteractWithWorkflow(ctx restate.WorkflowSharedContext) error { // implement interaction logic here return nil }` |

### Summary of Service Usage

To summarize, you should **Start with Basic Services** for general business logic and orchestration, **Use Virtual Objects** to model data entities that need consistent state per key, and **Use Workflows** for processes that require pauses, waiting for events, or have a defined, long lifecycle.


## 5. CODE REDUX

This list comprehensively details the required and recommended actions ("Do's") and the forbidden or cautioned actions ("Don'ts") within the Restate development and operational environments, categorized for clarity.

***

## Do's (Recommendations and Requirements)

### I. Durable Execution & Determinism

1.  **Wrap Non-Deterministic Operations:**
    Any non-deterministic operation, such as HTTP calls, database responses, network calls, or UUID generation, must be wrapped using the Restate context durable step feature (`ctx.run` or equivalent) to ensure deterministic replay and persistence in the execution log.

    ```go
    // Go Example: Wrapping a database request in restate.Run
    result, err := restate.Run(ctx, func(ctx restate.RunContext) (string, error) {
        return doDbRequest()
    })
    if err != nil {
        return err
    } //
    
    // TypeScript Example: Wrapping an external API call
    const apiResult = await ctx.run("fetch-data", async () => {
        const response = await fetch("https://api.example.com/data");
        return response.json();
    }); //
    ```

2.  **Use Context Logger:**
    To avoid duplicate log statements during replays, use the Restate context logger (`ctx.Log()` in Go, `ctx.console` in TypeScript) instead of the default logger.

    ```typescript
    ctx.console.info("This will not be printed again during replays"); //
    ```

3.  **Generate Deterministic IDs and Time:**
    Generate stable UUIDs for idempotency keys and similar needs using the SDK's deterministic helpers, as they are seeded by the invocation ID and return the same result on retries. Similarly, use `ctx.date.now()` or equivalent methods for deterministic time.

    ```go
    uuid := restate.UUID(ctx) //
    ```

4.  **Handle Tool Execution Resiliency (AI Agents):**
    For AI agents, use Restate Context actions (e.g., `ctx.run`) to make tool executions resilient. Persist LLM responses using the `durableCalls` middleware.

    ```typescript
    // Vercel AI SDK example using ctx.run for resilience
    execute: async ({ city }) => {
        return await ctx.run("get weather", () => fetchWeather(city));
    }, //
    ```

### II. Concurrency and Flow Control

1.  **Parallelize Tasks Safely (Fan-Out/Fan-In):**
    Start multiple durable operations concurrently by calling them without immediate awaiting, and then use the SDK's durable combinators (`restate.Wait`, `restate.WaitFirst`, `restate.gather`, `RestatePromise.all`, etc.) to collect results.

    ```go
    // Go Example: Waiting for all tasks to complete
    callFuture1 := restate.Service[string](ctx, "MyService", "MyHandler").RequestFuture("hi")
    callFuture2 := restate.Service[string](ctx, "MyService", "MyHandler").RequestFuture("hi again") //
    
    for fut, err := range restate.Wait(ctx, callFuture1, callFuture2) {
        // ... process results ...
    } //
    ```

2.  **Implement Timeouts and Racing:**
    Use combinators like `restate.WaitFirst` (Go) or `restate.select()` (Python) to race an asynchronous operation against a durable timer (`restate.After` / `ctx.sleep`), handling the resulting winner deterministically.

    ```go
    // Go Example: Racing a sleep future against a service call future
    sleepFuture := restate.After(ctx, 30*time.Second)
    callFuture := restate.Service[string](ctx, "MyService", "MyHandler").RequestFuture("hi")
    fut, err := restate.WaitFirst(ctx, sleepFuture, callFuture) //
    // ... switch on fut to determine winner ...
    ```

3.  **Use Durable Sleep:**
    Use `ctx.sleep` (`restate.After` in Go) to pause a handler for a set duration, leveraging Restate's suspension feature to save FaaS costs.

    ```go
    sleepFuture := restate.After(ctx, 30*time.Second) //
    ```

### III. State, Communication, and Error Handling

1.  **Manage State in Virtual Objects/Workflows:**
    Use `restate.Set` and `restate.Get` (or equivalent SDK functions) to manage persistent key-value state within Virtual Objects and Workflows. This state is durable and consistent with execution.

    ```go
    // Go Example: Setting and getting state
    restate.Set(ctx, "my-key", "my-new-value") //
    myNumber, err := restate.Get[int](ctx, "my-number-key") //
    restate.ClearAll(ctx) // Clear all state
    ```

2.  **Use Exclusive vs. Shared Handlers:**
    In Virtual Objects, use handlers with `ObjectContext` (exclusive/read-write) for state modification and handlers with `ObjectSharedContext` (shared/read-only) for concurrent reading of state.

3.  **Use Idempotency Keys:**
    Provide an idempotency key when invoking a service call or sending a message (especially via external clients or cross-handler calls) to prevent duplicate executions across different handler invocations.

    ```go
    // Go Example: Sending a message with an idempotency key
    restate.ServiceSend(ctx, "MyService", "MyHandler").
        Send("Hi", restate.WithIdempotencyKey("my-idempotency-key")) //
    ```

4.  **Use Awakeables vs. Durable Promises:**
    Use **Awakeables** when services or Virtual Objects need to coordinate with *external systems* (e.g., human approval, webhooks). Use **Durable Promises** for signaling *between handlers within the same Workflow*.

    ```go
    // Go Example: Resolving a Durable Promise from a Workflow handler
    err := restate.Promise[string](ctx, "review").Resolve(review) //
    ```

5.  **Throw Terminal Errors:**
    For non-transient failures (e.g., invalid input, business logic failures) where retrying won't help, throw a `TerminalError` (or `TerminalException`) to stop execution and propagate the error back to the caller.

    ```go
    return restate.TerminalError(fmt.Errorf("Something went wrong."), 500) //
    ```

6.  **Implement Sagas:**
    Implement compensating actions for each durable step in a list, and execute them in reverse order within a `catch` block that handles the propagated `TerminalError`. Compensations should be idempotent.

    ```typescript
    // TypeScript Example: Sagas compensation logic
    } catch (e) {
        if (e instanceof restate.TerminalError) {
            for (const compensation of compensations.reverse()) {
                await compensation();
            }
        }
        throw e;
    } //
    ```

### IV. Deployment and Operations

1.  **Use Immutable Deployment Versions:**
    Always deploy code changes using new, immutable deployment versions. Restate automatically routes new traffic to the latest version while allowing ongoing invocations to complete on the old version.

2.  **Handle HTTP/1.1 Deployment Registration:**
    If deploying services on platforms that only support HTTP/1.1 (like Uvicorn or certain serverless runtimes), register the service deployment using the `--use-http1.1` flag.

3.  **Ensure HTTPS and HTTP/2 for Public Endpoints:**
    If using publicly accessible service endpoints, **HTTPS must be used** between Restate and your services, and the load balancer should **support HTTP/2** for optimum performance.

4.  **Align RocksDB Memory:**
    In a Kubernetes environment, configure the RocksDB memory limit (`rocksdb-total-memory-size`) to align with pod memory requests and limits, typically setting it to 75% of the pod's memory requests.

5.  **Enable JSON Logging in Production:**
    When deploying in production, set the log level to `info` and enable the `json` format (`log-format = "json"`) for compatibility with log collectors.

### V. AI Agent Best Practices

1.  **Raise Restate Errors in Tools (OpenAI SDK):**
    When defining tools for the OpenAI Agent SDK, set the `failure_error_function` to `raise_restate_errors` to ensure that errors like suspensions or terminal errors are handled by Restate, rather than being consumed by the agent framework.

    ```python
    @function_tool(failure_error_function=raise_restate_errors)
    async def get_weather(
        wrapper: RunContextWrapper[restate.Context], req: WeatherRequest
    ) -> WeatherResponse:
    # ... durable tool execution ...
    ```

2.  **Add AGENTS.md Rules:**
    To improve the performance and behavior of AI coding agents (like Cursor or Claude Code) when generating Restate code, add the relevant SDK's `AGENTS.md` rules file to the agent's context (e.g., in the `.cursor/rules` or `.claude` folder).

***

## Don'ts (Actions to Avoid or Be Careful About)

### I. Determinism Violations

1.  **Avoid Native Concurrency for Durable Operations (Go):**
    Do not try to combine blocking Restate operations using **goroutines, channels, or select statements** outside of a `restate.Run` function, as this leads to non-determinism errors upon replay.

2.  **Avoid Context Operations inside `ctx.run`:**
    You cannot use the Restate context (`ctx.get`, `ctx.sleep`, or nested `ctx.run`) inside a `ctx.run` or `restate.Run` block.

3.  **Avoid Unversioned Code Changes:**
    Do not change the underlying service code (e.g., update the container image) without registering a new version of the service deployment, as this causes a **Journal Mismatch** (RT0016) or non-deterministic errors.

4.  **Do Not Use Deterministic IDs for Cryptography:**
    Do not use UUIDs generated by Restate's deterministic helpers (`restate.UUID(ctx)`, `ctx.random().nextUUID()`) in cryptographic contexts.

### II. Concurrency and Deadlock Risks

1.  **Avoid Blocking Exclusive Handlers:**
    Do not use `ctx.sleep` or wait for an `Awakeable` in an **exclusive handler** of a Virtual Object, because this will queue and block all other calls to that specific object key.

2.  **Avoid Object Call Deadlocks:**
    Avoid complex request-response calls between exclusive handlers of Virtual Objects that lead to **cross deadlocks** (A $\to$ B and B $\to$ A with the same key) or **cycle deadlocks** (A $\to$ B $\to$ C $\to$ A).

3.  **Avoid Mutually Calling Agents in the Same Object:**
    Do not place mutually calling agents within the same Virtual Object keyed instance, as this leads to deadlocks because the exclusive handler will block on the sub-agent call.

### III. Error Handling and Logging

1.  **Do Not Catch Generic Exceptions (Python):**
    In Python, **do not** use bare `except:` or `except Exception:` (`except:` catches internal SDK exceptions); only catch specific errors like `TerminalError`.

2.  **Avoid Redundant Idempotency Keys:**
    Do not provide an idempotency key for service calls made during the same handler execution, as Restate automatically handles deduplication in that specific context.

### IV. Deployment and Storage

1.  **Avoid In-Place Updates:**
    Avoid using **in-place updates** (`PUT /deployments/{deployment}`) except in rare cases for critical bug fixes, as this approach breaks determinism if not carefully handled and is discouraged compared to immutable versioning.

2.  **Avoid Changing Log Server Location:**
    It is **strongly recommended not to change the node's location string** after its initial registration if the `log-server` role is enabled, as this may result in data loss or inconsistency.

3.  **Do Not Assume Full Encryption:**
    It is not possible to encrypt all journal entries. Only specific entries are encrypted: handler I/O, `ctx.run` success results, RPC parameters/returns, state values, Awakeables, and Durable Promise results.
