# Chapter 6: Run (Side Effects)

In [Chapter 5: Client (Inter-Service Communication)](05_client__inter_service_communication__.md), we learned how your Restate services can durably and reliably communicate with *each other*. This is great for building internal logic within your Restate application.

But what happens when your service needs to talk to the *outside world*? Imagine calling an external payment gateway, updating a third-party CRM, sending an email, or interacting with a traditional database. These are operations that Restate doesn't directly manage, and they often involve:
1.  **Non-determinism**: The result might vary each time (e.g., getting a random number, fetching the current time).
2.  **Side Effects**: They change state *outside* Restate or have observable effects on external systems (e.g., sending an email, writing to a database).
3.  **External Failures**: The external system might be slow, unavailable, or return errors.

If your Restate service just made these calls directly, it would break Restate's core guarantees of durability and fault tolerance. This is where the `Run` operation comes in!

## The Tightly Controlled Experiment

Think of your Restate service as a meticulous scientist. While it performs all its internal experiments with perfect, repeatable steps, sometimes it needs to interact with a chaotic, unpredictable outside world.

The `restate.Run` operation is like a **tightly controlled experiment** where the scientist says:
"I'm going to perform this external action (e.g., call the weather API). I know it might be unpredictable. So, I will **record its exact outcome** (success or failure, and the result data). If I ever need to repeat this experiment, I'll just check my records and use the *original outcome* I already recorded, ensuring consistency."

Restate ensures that the function you provide to `restate.Run`:
*   Is executed only **once successfully**, even if the surrounding handler retries.
*   Its result (or error) is **durably stored** in Restate's journal.
*   If the service retries (due to a crash or temporary network issue), Restate will skip re-executing the `Run` block and instead return the **durably stored result**, guaranteeing a consistent outcome.

This makes it safe to interact with external, non-deterministic, or side-effecting systems from within your durable Restate services.

## A Central Use Case: Post-Order External Interactions

Let's expand our `OrderProcessor` service. After successfully checking inventory and processing payment ([Chapter 5: Client (Inter-Service Communication)](05_client__inter_service_communication__.md)), it now needs to:
1.  **Log the final order status** to an external analytics system (e.g., DataDog, Segment).
2.  **Integrate with a shipping provider API** to schedule the physical shipment.

These are perfect candidates for `restate.Run`.

## Key Concepts Explained

The `restate.Context` provides the `Run` methods. These methods execute a standard Go function (often called a "closure") that contains your external interaction logic.

### 1. `restate.Run()`: Synchronous Side Effects

The most common way to use `Run` is synchronously. Your handler calls `restate.Run()`, provides a function, and then **durably pauses** until that function completes its execution (potentially interacting with an external system) and its result is recorded by Restate.

### 2. `restate.RunVoid()`: Side Effects Without a Return Value

If your side effect function doesn't need to return a value (only an error), you can use `restate.RunVoid()` for convenience.

### 3. `restate.RunAsync()`: Asynchronous Side Effects

Just like `restate.Client` has `Request()` and `RequestFuture()`, `restate.Run` has `restate.Run()` and `restate.RunAsync()`. `RunAsync()` returns a `restate.RunAsyncFuture` immediately, allowing your handler to perform other tasks concurrently and then wait for the `Run` block's result later using `restate.WaitFirst` or `restate.Wait` ([Chapter 4: Futures & Selectables (Asynchronous Operations)](04_futures___selectables__asynchronous_operations__.md)).

### Options for `restate.Run`

You can pass options to `restate.Run`, `restate.RunAsync`, and `restate.RunVoid` to configure their behavior:

| Option                           | Description                                                                                                                                                                                                                                                               |
| :------------------------------- | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `restate.WithName(name string)`  | Assigns a logical name to the `Run` block. This is highly recommended for observability and debugging, as it appears in logs and metrics.                                                                                                                                |
| `restate.WithMaxRetryAttempts(uint)` | Sets a maximum number of retries for the inner function if it fails with a *retryable* error. If set to 0, it will not retry (only execute once).                                                                                                                     |
| `restate.WithInitialRetryInterval(time.Duration)` | The initial delay before the first retry. Defaults to 50ms.                                                                                                                                                                                           |
| `restate.WithRetryIntervalFactor(float64)` | Multiplier for the retry interval. For example, a factor of 2.0 means delays double with each retry. Defaults to 2.0.                                                                                                                                              |
| `restate.WithMaxRetryInterval(time.Duration)` | The maximum duration between retries. Defaults to 2s.                                                                                                                                                                                                     |
| `restate.WithMaxRetryDuration(time.Duration)` | The maximum total duration for all retries. If this limit is reached, further retries are stopped.                                                                                                                                                          |
| `restate.WithCodec(encoding.Codec)` | Specifies the codec (e.g., JSON, ProtoBuf) to use for marshaling/unmarshaling the `Run` block's input/output. Defaults to JSON. Useful if your `Run` block returns complex data structures. 