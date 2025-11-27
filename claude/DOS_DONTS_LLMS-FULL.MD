Based on the comprehensive review of the provided sources, here is an itemized list of do's and don'ts for developing, deploying, and operating services within the Restate environment.

***

### Do's (Recommendations and Requirements)

#### I. General Development & Handler Logic

1.  **Use Context for Operations:** Always invoke handlers via the Restate context (`ctx`) if you have access to it.
2.  **Ensure Durability:** Use the `Context` to interact with Restate, as the SDK stores these actions in the Restate journal to make them durable.
3.  **Wrap Non-Deterministic Operations:** Use `ctx.run` (or `restate.Run`/equivalent SDK function) to safely wrap any non-deterministic operation (like HTTP calls, database responses, network calls, API calls, or database interaction), ensuring Restate persists its result in the execution log and retries transient errors until they succeed.
4.  **Use Context Logger for Logging:** Use the **Restate context logger** (`ctx.Log()` in Go, `ctx.console` in TypeScript) to avoid duplicate log statements during handler replays.
5.  **Manage Handler Runtime:** Break long-running handlers (days or months) into smaller chunks and **chain them via delayed messages or service-to-service calls**.
6.  **Handle Long Operations:** Adjust the service's **abort timeout and inactivity timeout** settings to accommodate long-running operations, such as LLM calls that take up to 3 minutes.

#### II. Concurrency and Parallelism

1.  **Use Concurrent Tasks:** Use concurrent tasks when you need to call multiple external services simultaneously, race operations (for timeouts/first result), or perform batch operations.
2.  **Safely Wait on Futures (Go):** Use `restate.Wait` or `restate.WaitFirst` to safely wait on futures concurrently.
3.  **Safely Wait on Futures (Java/Kotlin):** Use `Select.select()` or `select { ... }` to wait for the first completion, and `DurableFuture.all` or `list.awaitAll()` to wait for all tasks to complete.
4.  **Safely Wait on Futures (Python):** Use `restate.select()` to race operations, `restate.gather()` to wait for all tasks, or `restate.wait_completed()` to handle completed results and potentially cancel pending ones.
5.  **Parallelize Agent Steps:** To parallelize AI agent tool steps, implement an orchestrator tool that uses durable execution combinators (like `restate.gather` or `RestatePromise.all`) to run multiple steps in parallel.

#### III. Service Types and Communication

1.  **Choose Appropriate Service Types:**
    *   Start with **Basic Services** for general business logic, data processing, API calls, sagas, background jobs, and ETL.
    *   Use **Virtual Objects** to model stateful entities (user accounts, agents, state machines).
    *   Use **Workflows** for multi-step processes that execute exactly-once and require external interaction or orchestration.
2.  **Handle Workflow Execution:** Ensure the workflow's `run` handler executes **exactly once per workflow ID**.
3.  **Use Idempotency Keys:** Use an **idempotency key** when invoking a handler (via request header, SDK client, or service message options) to prevent duplicate execution, especially if multiple callers might call the same service independently.
4.  **Attach to Invocations:** Use the `attach` mechanism (via SDK or HTTP) to retrieve the result of an ongoing invocation or workflow if an idempotency key was used.

#### IV. State Management and Data

1.  **Use State for Correctness:** Use Restate state (Virtual Objects/Workflows) for state requiring **tight integration with function logic, resilience, and correctness**, such as transactional state machines, session state, or AI agent context.
2.  **Use Database for Complex Access:** Use a separate database when you need complex access patterns, full SQL, text search, time-series analysis, or for core business data accessed by external services.
3.  **Manage State Access:** Use **exclusive handlers** (`ObjectContext`, `WorkflowContext`) for **read and write state access**.
4.  **Query State Safely:** Use **shared handlers** (`ObjectSharedContext`, `WorkflowSharedContext`) for **read-only state access**, as they run concurrently.
5.  **Clear State:** Clear long-lived Virtual Object state explicitly using `ctx.clear()` or `ctx.clear_all()`.
6.  **Handle Serialization:** Use **Pydantic models** (Python) or similar structures/custom serializers to define the structure of data for handler input/output, state, and serialization if default JSON is insufficient.
7.  **Choose State Loading Strategy:** Consider enabling **lazy state** loading for large state objects that are rarely accessed. Use the default **eager state** for small, frequently accessed state, especially when running on platforms like AWS Lambda where replay costs are a concern.
8.  **Database Semantic Locks (Advanced):** To ensure exactly-once semantics for database updates, implement the semantic lock pattern by separating reads (via `ctx.run`) and conditional updates based on versions.

#### V. Error Handling, Timers, and External Events

1.  **Signal Permanent Failure:** Use `TerminalError` (or equivalent exception) to signal permanent failures (e.g., invalid input, business logic failures) to **stop infinite retries**.
2.  **Implement Sagas:** Implement Sagas by building a list of compensating actions for business logic steps and executing them in **reverse order** within a `catch` block that handles `TerminalError`.
3.  **Ensure Compensations are Idempotent:** Make sure that compensating actions are idempotent.
4.  **Order Compensation Registration:** When using one-shot APIs (e.g., payment), register the compensation (e.g., refund) **before** executing the main action (e.g., charge).
5.  **Use Awakeables for External Coordination:** Use **Awakeables** for Services and Virtual Objects when coordinating with external systems (human-in-the-loop, webhooks, async APIs).
6.  **Use Durable Promises for Internal Signaling:** Use **Durable Promises** exclusively within Workflows for signaling between different workflow handlers.
7.  **Handle Await Failures:** **Always handle rejections** from Durable Promises or Awakeables to gracefully manage failures.
8.  **Set Timeouts for External Waits:** Include timeouts for long-running external processes and context actions (calls, awakeables).
9.  **Use Suspensions on FaaS:** Leverage Restate's suspension feature (when waiting on durable operations like sleep or awakeables) to **save costs** on Function-as-a-Service platforms.

#### VI. Deterministic Randomness and Time

1.  **Generate Deterministic UUIDs:** Use the SDK's deterministic helpers (`restate.UUID(ctx)`, `ctx.random().nextUUID()`, `ctx.random().uuid4()`) to generate stable UUIDs for purposes like idempotency keys.
2.  **Generate Deterministic Time:** Use `ctx.date.now()` (TS) or similar methods to get a deterministic time consistent across retries.

#### VII. Deployment, Infrastructure, and Security

1.  **Use Immutable Versions:** Restate requires the use of **immutable versions** for deployments to ensure deterministic durable execution upon replay.
2.  **Keep Old Deployments:** If using long sleeps or long-running invocations, you need to **keep the old deployment version** around until the invocation completes.
3.  **Register HTTP/1.1 Deployments:** If deploying services using a server that only supports HTTP/1.1 (e.g., Uvicorn, Vercel/Next.js), specify the `--use-http1.1` flag during registration.
4.  **Use HTTPS for Public Endpoints:** If your services are deployed as public endpoints (e.g., serverless functions), **HTTPS must be used** between Restate and your services.
5.  **Use HTTP/2 Load Balancer:** For optimum performance, the load balancer in front of your services **must support HTTP/2**.
6.  **Secure Public Endpoints:** **Secure access** to publicly routable services so that only the designated Restate environment can call them.
7.  **Validate Request Identity:** Use Restate's native request identity feature/SDK methods to cryptographically verify that incoming requests originate from the correct Restate instance.
8.  **Configure Kubernetes Security:** In a production Kubernetes environment, configure appropriate **network policies for security**.
9.  **RocksDB Memory Sizing:** When configuring Restate Server settings for RocksDB memory, set the limit (e.g., `rocksdb-total-memory-size`) to align with pod memory limits, typically **75% of pod requests**.
10. **Use Snapshots in Clusters:** **Snapshots are essential** and required when operating a multi-node cluster, adding/removing nodes, or enabling log trimming.
11. **Metadata Storage:** **S3 is the only supported object store for metadata storage**.
12. **Upgrades:** Perform **data backups** and **verify the upgrade path in an isolated test environment** before performing software upgrades.

#### VIII. AI Agents and Assisted Development

1.  **Use Durable Execution for Agents:** Use Durable Execution to make AI agents resilient to LLM calls and tool execution failures.
2.  **Persist LLM Responses:** Wrap the model with the `durableCalls` middleware to ensure every LLM response is saved and can be replayed.
3.  **Set Tool Error Handling (OpenAI SDK):** When using the OpenAI Agent SDK, always set the tool's `failure_error_function` to **raise Restate errors** (like suspensions or terminal errors) to ensure the agent doesn't consume them internally.
4.  **Use AGENTS.md:** For AI coding agents (Cursor, Claude Code), **add the AGENTS.md rules** to the agent's context (in the `.cursor/rules` or `.claude` folders) to improve performance.

***

### Don'ts (Actions to Avoid)

#### I. Core Durability and Determinism Violations

1.  **Avoid Using Bare Exceptions (Python):** **DO NOT use bare `except:` or `except Exception:`** in Restate handlers, as this catches internal SDK exceptions that must not be handled, leading to unexpected behavior.
2.  **Avoid Combining Blocking Operations with Native Concurrency (Go):** **Do not use Go routines, channels, or select statements** to combine blocking Restate operations outside of a `restate.Run` function, as this causes **non-determinism errors** upon replay.
3.  **Avoid Context Misuse in Run Blocks:** Inside a `ctx.run` or `restate.Run` block, **you cannot use the Restate context** (e.g., `ctx.get`, `ctx.sleep`, or nested `ctx.run`).
4.  **Avoid Non-Deterministic Execution Flow:** **Do not branch the execution flow based on non-deterministic information** (e.g., elapsed time, unrecorded HTTP results, random values, current date-time).
5.  **Avoid Non-Deterministic Iteration:** **Do not execute Context operations** while iterating over data structures with **non-deterministic iteration order** (like sets/maps/dictionaries).
6.  **Avoid Non-Cryptographic Deterministic IDs:** **Do not use Restate-generated UUIDs** (e.g., `restate.UUID(ctx)`) in cryptographic contexts.

#### II. Service Communication and State Risks

1.  **Avoid Deadlocks in Virtual Objects:** Be aware that **Request-response calls between exclusive handlers of Virtual Objects may lead to deadlocks** (Cross deadlock: A $\to$ B and B $\to$ A; Cycle deadlock: A $\to$ B $\to$ C $\to$ A).
2.  **Avoid Sleeping in Exclusive Handlers:** **Do not sleep or wait for an awakeable** in an **exclusive handler** of a Virtual Object, as this will queue all other calls to that object key.
3.  **Avoid Idempotency Key Redundancy:** **Do not provide an idempotency key** for calls made *during the same handler execution*, as Restate handles deduplication automatically in that case.
4.  **Avoid Virtual Object Mutual Calls (Agents):** **Do not put mutually calling agents within the same Virtual Object keyed instance**, as this leads to deadlocks.

#### III. Deployment and Infrastructure

1.  **Avoid Dangerous Deployment Updates:** **Do not use PUT `/deployments/{deployment}`** (in-place update) unless absolutely necessary for failing invocations; it is a dangerous operation.
2.  **Avoid Incompatible Deployment Updates:** When updating a deployment, **do not change the service type**, and ensure the new deployment contains **at least all the previous services and handlers**, with the same supported protocol versions.
3.  **Avoid Non-Versioned Code Changes:** **Do not change the underlying service code** (e.g., update the container image) without registering a new immutable deployment version, as this causes **non-determinism errors** upon replay.
4.  **Avoid Unnecessary In-Place Updates:** Do not use **in-place updates** for deployment except for bug fixes in special situations; stick to immutable versions.
5.  **Avoid Dangerous Tunnel Access:** If the local remote proxy ports (8080 and 9080) are enabled via the tunnel client, be careful to **restrict access** to them.
6.  **Avoid Non-Snapshot Clusters:** **Running a distributed cluster without snapshots is not recommended for production use**.
7.  **Avoid Non-S3 Metadata Storage:** **Do not use S3-compatible stores other than S3 itself** (like MinIO) for cluster metadata storage.
8.  **Avoid Location String Change:** It is **strongly recommended to not change the node's location string** after its initial registration if the `log-server` role is enabled, as this may result in data loss or inconsistency.
9.  **Avoid Unauthorized Public Access:** **Do not allow unrestricted access** to publicly routable services.

#### IV. Cluster Management and Rollback

1.  **Avoid Using Deprecated Kill API:** **Do not use DELETE `/invocations/{invocation_id}`**; use `kill_invocation`/`cancel_invocation`/`purge_invocation` instead.
2.  **Avoid Killing Invocations Recklessly:** Be aware that **killing an invocation** does not guarantee consistency for virtual object instance state or in-flight invocations.
3.  **Avoid Corrupt Backups:** **Do not attempt to coordinate simultaneous backups across multiple nodes** due to the high risk of creating inconsistent data sets.
4.  **Avoid Deep Rollbacks:** **Do not roll back more than one minor version** behind the most recent version used with the data store.
5.  **Avoid Rollback after New Feature Use:** **Do not roll back** if you have used **new features exclusive to the newer version**.

#### V. Encryption

1.  **Avoid Assuming Full Encryption:** Be aware that **it is not possible to encrypt any journal entries** other than input/output parameters, `ctx.run` results, RPC parameters/return values, state values, Awakeables, and Durable Promise results.
