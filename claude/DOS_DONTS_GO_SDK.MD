The sources provide several operational recommendations, technical constraints, and warnings regarding deprecated usage for writing services using the Restate Go SDK.

Here are the itemized dos and don'ts derived from the provided material:

---

## Dos (Recommended Actions, Technical Requirements, and Positive Constraints)

### Prerequisites and Setup
*   **Do** ensure you are using Go version `>= 1.24.0`.
*   **Do** join the Restate online community for help, sharing feedback, and talking to the community.
*   **Do** check out the documentation to get quickly started.
*   **Do** follow Restate on Twitter for staying up to date.
*   **Do** create a GitHub issue for requesting a new feature or reporting a problem.
*   **Do** visit the GitHub organization for exploring other repositories.
*   **Do** join the Restate community and start contributing (whether through feature requests, bug reports, ideas & feedback, or Pull Requests).

### Concurrency and Asynchronous Operations
*   **Do** use `Context.Select` if you want to wait on multiple asynchronous results at once.
*   **Do** use the newer functions `WaitFirst`, `Wait`, or `WaitIter` when waiting for asynchronous results instead of the deprecated `Select`.
*   **Do** ensure the `Done()` method on an `AfterFuture` is always called, even after using `Context.Select`, as it can return a terminal error if the invocation was cancelled mid-sleep.

### Handler Definition and Reflection (`Reflect` API)
*   **Do** ensure methods intended as handlers are exported.
*   **Do** ensure handler methods are defined on the *exact type T* provided to the `.Reflect` function, and not on `*T`.
*   **Do** structure handler methods to have one of the accepted context types as the first parameter following the receiver (`WorkflowContext`, `WorkflowSharedContext`, `ObjectContext`, `ObjectSharedContext`, or `Context`).
*   **Do** ensure service handler methods have 2 or 3 input parameters (receiver, context, optionally input `I`).
*   **Do** ensure service handler methods return 0 to 2 output parameters (the acceptable signatures are `()`, `(O)`, `(error)`, or `(O, error)`).
*   **Do** ensure a workflow service definition has exactly one "Run" handler using `WorkflowContext`.

### Codec and Serialization
*   **Do** ensure that Codecs *must* marshal deterministically, such that a round trip of `[]byte` -> `any` -> `[]byte` leaves the bytes unchanged.
*   **Do** ensure that implementations of `RestateMarshaler` *must* be deterministic.
*   **Do** take care to ensure that only valid types are passed to a codec (e.g., `proto.Message` for `ProtoCodec`).

### Generated Code and Mocking
*   **Do** generate mockery mocks by running `mockery` in the root of the repo.
*   **Do** maintain the structure in `helpers.go` when regenerating mocks, ensuring certain mock structs/functions are redefined there and commented out in their respective files, which otherwise would be a build error.
*   **Do** embed `Unimplemented[Service]Server` by **value** (instead of pointer) to ensure forward compatible implementations and prevent nil pointer dereference when unimplemented methods are called.

### Runtime and Identity
*   **Do** use `restate.UUID` directly, instead of `restate.Rand().UUID()`.
*   **Do** use `.Bidirectional(false)` when serving over a channel that does not support full-duplex request and response (e.g., Cloudflare Workers) to prevent handlers from getting stuck.
*   **Do** restrict handler access if requests are accepted without validating request signatures (`r.keyIDs == nil`).

---

## Don'ts (Forbidden Actions, Warnings, and Deprecated Usage)

### Concurrency and Asynchronous Operations
*   **Don't** call `Response()` on `ResponseFuture` from inside a goroutine.
*   **Don't** call `Response()` on `AttachFuture` from inside a goroutine.
*   **Don't** call `Result()` on an awakeable (`AwakeableFuture`) from inside a goroutine.
*   **Don't** call `Result()` on `RunAsyncFuture` from inside a goroutine.
*   **Don't** call `Result()` on `DurablePromise` from inside a goroutine.
*   **Don't** use the `Selector` API when cancellation is possible, as it is deprecated and cancellation is not supported by this API.
*   **Don't** call `Done()` on `AfterFuture` from inside a goroutine.

### Deprecated Features
*   **Don't** use SDK versions `< 0.16`, as they are deprecated and cannot be registered anymore.
*   **Don't** use the `Select` function.
*   **Don't** use the type alias `Selectable`.
*   **Don't** use the type alias `Selector`.

### Handler Definition and Reflection
*   **Don't** define handler methods that are not exported.
*   **Don't** define handler methods with fewer than two input parameters.
*   **Don't** use a context parameter that doesn't match the service type when defining handlers (e.g., mixing `ObjectSharedContext` and `Context` in methods of the same reflected struct).
*   **Don't** define multiple methods using `WorkflowContext`; a workflow must have exactly one 'Run' method.
*   **Don't** define handler methods with more than one optional input parameter (after the context).
*   **Don't** return more than two parameters from a Restate handler.
*   **Don't** return two parameters from a handler where the second parameter is not an error.
*   **Don't** use streaming methods, as they are "not currently supported in Restate".
*   **Don't** set a handler type for services of type `SERVICE`.

### Service Configuration and Versioning
*   **Don't** set the `WorkflowRetention` option when creating a non-workflow service (e.g., Service or Object), as this will cause a panic.
*   **Don't** register multiple service definitions with the same name, as this is a programming error and will panic.

### Restate Server Version Constraints (Discovery Protocol V3/V4)
If registering a service against **Restate Server < 1.4** (Discovery Protocol Version V3 is required for these options):
*   **Don't** set `WithAbortTimeout` (for service or handler).
*   **Don't** set `WithEnableLazyState` (for service or handler).
*   **Don't** set `WithIdempotencyRetention` (for service or handler).
*   **Don't** set `WithInactivityTimeout` (for service or handler).
*   **Don't** set `WithIngressPrivate` (for service or handler).
*   **Don't** set `WithJournalRetention` (for service or handler).
*   **Don't** set `WithWorkflowRetention` (for service or handler).

If registering a service against **Restate Server < 1.5** (Discovery Protocol Version V4 is required for this option):
*   **Don't** set `WithInvocationRetryPolicy` (for service or handler).
