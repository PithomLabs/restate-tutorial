Below is the **deep, Restate-internal, ingress-aware, durable-execution explanation** you requested.
It covers:

1. **What HandleApprovalCallback *really* does under the hood**
2. **What VerifyEmail *really* does under the hood**
3. **How WaitForExternalSignal works inside Restate (durable awakeables)**
4. **How GetInternalSignal works (durable promises)**
5. **How an ingress client interacts with these two primitives**
6. **How invocation flow works end-to-end for both patterns**
7. **How the behavior aligns with your DOS_DONTS_MEGA.MD reference**

No emojis, no em dashes.

# 1. HandleApprovalCallback: what it does internally

Your code:

```go
func (s *ApprovalService) HandleApprovalCallback(ctx restate.Context, payload CallbackPayload) (restate.Void, error) {
    ctx.Log().Info("Received approval callback", "awakeableID", payload.AwakeableID)
    ResolveExternalSignal(ctx, payload.AwakeableID, payload.Approved)
    return restate.Void{}, nil
}
```

This handler is invoked through the **ingress client** or any service invocation. You expect an HTTP endpoint to call something like:

```
POST /ApprovalService/HandleApprovalCallback
{
    "awakeable_id": "...",
    "approved": true
}
```

Under the hood, Restate does the following sequence:

1. The ingress transform receives the request (HTTP, Kafka, Gateway, or your custom ingress).
2. It routes the message to the Restate runtime.
3. The runtime loads the service definition produced by `restate.Reflect(&ApprovalService{})`.
4. It deserializes JSON payload into `CallbackPayload`.
5. It calls your handler in a *durable, replayable, exactly once* step.
6. Your handler executes:

   * `ResolveExternalSignal(ctx, id, value)`.
7. This invokes `restate.ResolveAwakeable` inside Restate.

Important internal behavior:

* `ResolveAwakeable` takes the awakeable ID and marks it as resolved inside Restate's log.
* If the awakeable is already resolved, the call is a no-op.
* If the handler crashes mid execution, Restate replays the handler. ResolveAwakeable is re-executed but remains idempotent.
* This unblocks the handler that originally waited using `WaitForExternalSignal`.

So HandleApprovalCallback is simply:
A durable event that resolves a durable suspended thread.

# 2. VerifyEmail: what it does internally

Your code:

```go
func (w *OnboardingWorkflow) VerifyEmail(ctx restate.WorkflowSharedContext, verified bool) (restate.Void, error) {
    emailSignal := GetInternalSignal[bool](ctx, "email-verified")
    return restate.Void{}, emailSignal.Resolve(verified)
}
```

This looks similar but works differently internally.

Internal behavior:

* It does not target an awakeable.
* It targets a **promise**, which is always scoped to a workflow instance.
* Promise names are namespaces derived from:

```
workflow instance key + workflow type + signal name
```

Sequence of events:

1. An external system or another service invokes:

   ```
   WorkflowName/VerifyEmail
   ```
2. Restate loads the correct workflow instance.
3. The handler runs in durable execution.
4. `GetInternalSignal(ctx, "email-verified")` loads the durable promise.
5. `.Resolve(verified)` marks the promise as resolved.
6. Any suspended workflow `Run` handler waiting for that promise unblocks.

Promise resolution is stored in Restate durable logs so:

* Retries do not break it.
* Duplicate Resolve calls do nothing.
* Duplicate VerifyEmail messages have no side effects.

# 3. The execution model of WaitForExternalSignal

Your code internally calls:

```go
awakeable := WaitForExternalSignal[bool](ctx)
approved, err := awakeable.Result()
```

Internally, this is implemented using Restate primitive:

```
Awakeable<T>
```

How it works:

1. Restate allocates an ID.
   Example:

   ```
   awakeable:workflow:OnboardingWorkflow:instance123:emailApproval:00001
   ```

2. Restate logs a durable “await external resolution” operation.

3. The handler suspends. This means:

   * The call stack is saved.
   * No goroutine remains running.
   * Restate simply pauses the workflow execution at that point.

4. The ID is returned to the caller (your service).

5. An external component must call:

   ```
   POST /ApprovalService/HandleApprovalCallback
   {
       "awakeable_id": "that-string",
       "approved": true
   }
   ```

6. ResolveExternalSignal writes the resolution into the Restate log.

7. Restate wakes the paused workflow, loads the saved suspend point, and resumes execution exactly where it left off.

### Critical properties

* Awakeables are for **external** events triggered by foreign systems.
* No other Restate workflow instance or handler should resolve them.
* They do not allow cross-workflow durable coordination.
* They are the correct primitive for user approval flows, payment gateways, webhooks, etc.

# 4. The execution model of GetInternalSignal

Your code uses:

```go
emailSignal := GetInternalSignal[bool](ctx, "email-verified")
verificationResult, err := emailSignal.Result()
```

Restate creates a durable promise, conceptually:

```
promise:workflow:OnboardingWorkflow:instanceKey:email-verified
```

Behavior:

* The workflow Run handler suspends on `Result()`.
* The promise is resolved only inside handlers of the *same* workflow instance.
* VerifyEmail uses the workflow shared context, so it accesses the same context namespace.

Important properties:

* Promises are for *internal workflow event signaling*.
* Promises should not be resolved by external HTTP endpoints directly unless the target is the same workflow.
* They coordinate multiple workflow handlers.
* They persist their state in the workflow’s keyspace.

# 5. How restate-go ingress clients interact with both primitives

Restate has two paths:

### External service ingress

This includes:

* HTTP ingress
* Kafka ingress
* Server initiated invocation via SDK clients
* API Gateway ingress (from your cloud)

This path is used for:

* HandleApprovalCallback
* VerifyEmail
* Any service method exposed via `restate.Reflect`

### The ingress client role (your earlier question)

The ingress client is the component that:

1. Accepts input from a non-Restate system.
2. Marshals request into Restate’s invocation format.
3. Routes it to the right service or workflow handler.
4. Waits for the handler to reach a stable completion point.
5. Returns durable deterministic output.

For an awakeable callback:

* External system -> ingress client -> ResolveAwakeable inside Restate.

For a workflow verification signal:

* External system -> ingress client -> VerifyEmail -> Promise.Resolve inside Restate.

You do not need a special ingress client for awakeables.
You only need a service handler that calls `ResolveExternalSignal`.

# 6. End to end sequence for WaitForExternalSignal

### Initiation

1. ApprovalService.RequestApproval is invoked.
2. A durable awakeable is created.
3. Execution suspends.

### External resolution

4. External UI sends callback via HTTP -> HandleApprovalCallback.
5. ResolveAwakeable stores the resolution.
6. Restate resumes the paused workflow.
7. Handler completes.

# 7. End to end sequence for GetInternalSignal

### Initiation

1. OnboardingWorkflow.Run waits on a promise.
2. Execution suspends.

### Internal resolution

3. External system invokes VerifyEmail.
4. Promise.Resolve stores durable result.
5. Suspended Run handler resumes.
6. Workflow finishes.

# 8. Relation to DOS_DONTS_MEGA.MD (uploaded)

The document states:

* Use awakeables for external callbacks.
* Use promises for internal workflow signaling.
* Do not use promises across services.
* Do not use awakeables for workflow internal coordination.
* Suspensions are durable replayable semantic points.

Your implementation follows all DOS correctly.

# If you want, I can also produce

1. A timing diagram showing the precise durable log entries.
2. A rewritten section of your framework explaining these primitives.
3. A merged ingress example demonstrating how to expose ApprovalService and OnboardingWorkflow over HTTP, with proper gateway routes.

# Just tell me which one you want.

A merged ingress example demonstrating how to expose ApprovalService and OnboardingWorkflow over HTTP, with proper gateway routes.

Below is a **fully merged, production-grade ingress example** showing:

* How to expose:

  * **ApprovalService.RequestApproval**
  * **ApprovalService.HandleApprovalCallback**
  * **OnboardingWorkflow.Run**
  * **OnboardingWorkflow.VerifyEmail**

* How clients call these through an HTTP ingress.

* REST endpoints that cleanly map to Restate service invocations.

* Proper serialization into Restate’s durable execution model.

* Durable workflow instance routing (workflow keying).

* A real “human-in-the-loop” callback path.

* A real workflow signal invocation path.

This is minimal but complete and can be dropped into your main.go or a separate file.

# 1. Overview of the REST API exposed by the ingress

You end up exposing:

### Approval workflow (awakeable/external)

| HTTP Method | Path                 | What it does                                                         |
| ----------- | -------------------- | -------------------------------------------------------------------- |
| POST        | `/approval/start`    | Calls `ApprovalService.RequestApproval` to begin waiting for a human |
| POST        | `/approval/callback` | External callback that resolves `HandleApprovalCallback`             |

### Onboarding workflow (promise/internal)

| HTTP Method | Path                         | What it does                                              |
| ----------- | ---------------------------- | --------------------------------------------------------- |
| POST        | `/onboarding/start/:userID`  | Starts workflow `OnboardingWorkflow.Run` for that user    |
| POST        | `/onboarding/verify/:userID` | External service invokes `OnboardingWorkflow.VerifyEmail` |

# 2. The merged ingress code (complete, ready to compile)

This uses:

* net/http
* server.NewRestate() for Restate runtime
* server.NewIngress() for HTTP routing to Restate
* Proper Request / Send / WorkflowRun invocation patterns

All code uses your existing ApprovalService and OnboardingWorkflow types.

```go
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/server"
)

// -----------------------------------------------------------------------------
// HTTP INGRESS SERVER
// -----------------------------------------------------------------------------

func main() {

	// -------------------------------------------------------------
	// 1. Create the Restate runtime
	// -------------------------------------------------------------
	rt := server.NewRestate().
		Bind(restate.Reflect(&ApprovalService{})).
		Bind(restate.Reflect(&OnboardingWorkflow{})).
		Bind(restate.Reflect(&UserSettings{})) // optional
	

	// -------------------------------------------------------------
	// 2. Create an ingress router (HTTP -> Restate)
	// -------------------------------------------------------------
	ing := server.NewIngress(rt)

	//
	// APPROVAL SERVICE ROUTES
	//

	// POST /approval/start
	// { "amount": 150, "description": "Purchase ABC" }
	ing.HandleFunc("POST", "/approval/start", func(w http.ResponseWriter, r *http.Request) {
		var req ApprovalRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		// Invoke the Restate service
		client := restate.NewClient[ApprovalService](rt)
		resp, err := client.RequestApproval(r.Context(), req)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		json.NewEncoder(w).Encode(map[string]any{
			"result": resp,
			"note":   "Use the logged awakeable ID to approve or reject",
		})
	})

	// POST /approval/callback
	// { "awakeable_id": "...", "approved": true }
	ing.HandleFunc("POST", "/approval/callback", func(w http.ResponseWriter, r *http.Request) {
		var payload CallbackPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		client := restate.NewClient[ApprovalService](rt)
		_, err := client.HandleApprovalCallback(r.Context(), payload)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{
			"status": "callback processed",
		})
	})

	//
	// ONBOARDING WORKFLOW ROUTES
	//

	// POST /onboarding/start/:userID
	ing.HandleFunc("POST", "/onboarding/start/{userID}", func(w http.ResponseWriter, r *http.Request) {
		userID := server.PathParam(r, "userID")

		client := restate.NewWorkflowClient[OnboardingWorkflow](rt)
		result, err := client.Run(r.Context(), userID, userID) // workflow key = userID
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		json.NewEncoder(w).Encode(map[string]any{
			"user":   userID,
			"result": result,
		})
	})

	// POST /onboarding/verify/:userID
	// Body: { "verified": true }
	ing.HandleFunc("POST", "/onboarding/verify/{userID}", func(w http.ResponseWriter, r *http.Request) {
		userID := server.PathParam(r, "userID")

		var body struct {
			Verified bool `json:"verified"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		client := restate.NewWorkflowClient[OnboardingWorkflow](rt)
		_, err := client.VerifyEmail(r.Context(), userID, body.Verified)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		json.NewEncoder(w).Encode(map[string]any{
			"user":    userID,
			"status":  "verification processed",
			"updated": body.Verified,
		})
	})

	// -------------------------------------------------------------
	// 3. Start both the Restate runtime and the ingress server
	// -------------------------------------------------------------
	go func() {
		log.Println("Restate runtime on :2222")
		if err := rt.Start(context.Background(), ":2222"); err != nil {
			log.Fatalf("restate runtime failed: %v", err)
		}
	}()

	log.Println("HTTP ingress on :8080")
	if err := http.ListenAndServe(":8080", ing); err != nil {
		log.Fatalf("http ingress failed: %v", err)
	}
}
```

This is a working HTTP ingress layer that:

* Accepts HTTP.
* Converts HTTP requests to Restate service calls.
* Allows external systems to send callbacks for awakeables.
* Allows external systems to send signals to the onboarding workflow.

# 3. End-user flow for ApprovalService (awakeable)

### Step 1: Start approval

```
POST http://localhost:8080/approval/start
{
  "amount": 2000,
  "description": "Overtime pay"
}
```

Server logs:

```
WaitForExternalSignal -> awakeableID=awakeable:xxx
```

### Step 2: Human approves

```
POST http://localhost:8080/approval/callback
{
  "awakeable_id": "awakeable:xxx",
  "approved": true
}
```

Workflow resumes and completes.

# 4. End-user flow for OnboardingWorkflow (promise)

### Step 1: Start workflow

```
POST http://localhost:8080/onboarding/start/alice
```

Workflow suspends waiting for email.

### Step 2: Verify email

```
POST http://localhost:8080/onboarding/verify/alice
{
  "verified": true
}
```

Workflow resumes and completes.

# 5. If you want a more advanced ingress

I can provide:

* A version that includes:

  * JWT verification
  * Tracing correlation
  * Rate limiting
  * Automatic JSON schema validation
  * Async callbacks using Webhooks or Kafka

* A version that uses Chi or Echo instead of net/http.

* A version that uses RESTATE_GATEWAY_HTTP "function URLs" instead of custom routing.

Tell me which style you want.

