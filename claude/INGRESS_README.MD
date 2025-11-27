# Deep Dive: External Signals (Awakeables) vs Internal Signals (Promises)

Let me elaborate on how these two patterns work in the Restate Go SDK, covering both the handler-side mechanics and the invocation patterns.

---

## External Signals: `WaitForExternalSignal` (Awakeables)

### The `HandleApprovalCallback` Pattern

```go
// 1. INITIAL REQUEST: Handler creates an awakeable and suspends
func (s *ApprovalService) RequestApproval(ctx restate.Context, req ApprovalRequest) (string, error) {
    // Create awakeable - generates unique opaque ID
    awakeable := WaitForExternalSignal[bool](ctx)
    
    // Log or send this ID to external system
    // In production: send email, post to Slack, store in DB for UI
    ctx.Log().Info("Awaiting approval", "awakeableID", awakeable.Id())
    
    // Handler SUSPENDS here (on FaaS, saves costs!)
    // Restate records journal entry: "waiting for awakeable: abc123..."
    approved, err := awakeable.Result()
    if err != nil {
        return "", err
    }
    
    return fmt.Sprintf("Decision: %v", approved), nil
}

// 2. CALLBACK: External system resolves the awakeable
func (s *ApprovalService) HandleApprovalCallback(ctx restate.Context, payload CallbackPayload) (restate.Void, error) {
    ctx.Log().Info("Processing callback", "awakeableID", payload.AwakeableID)
    
    // This RESUMES the suspended RequestApproval handler
    // Restate finds the invocation waiting on this awakeable ID
    ResolveExternalSignal(ctx, payload.AwakeableID, payload.Approved)
    
    return restate.Void{}, nil
}
```

### How to Invoke (External System Perspective)

**Option 1: Via Ingress HTTP API** (Most Common)
```bash
# External webhook/UI calls this directly
curl -X POST http://restate:8080/awakeables/prom_1McZj3fJ8RTju7BAhyBygXdhwwd/resolve \
  -H "Content-Type: application/json" \
  -d '{"approved": true}'
```

**Option 2: Via Another Restate Handler** (As shown in example)
```go
// From external Go application using ingress client
client := ingress.NewClient("http://restate:8080")

// Call the callback handler, which internally resolves the awakeable
_, err := ingress.Service[CallbackPayload, restate.Void](client, "ApprovalService", "HandleApprovalCallback").
    Request(context.Background(), CallbackPayload{
        AwakeableID: "prom_1McZj3fJ8RTju7BAhyBygXdhwwd",
        Approved:    true,
    })
```

**Option 3: Direct Resolution (No Handler)**
```go
// From within ANY Restate handler (even different service)
func (s *OtherService) ApproveFromElsewhere(ctx restate.Context, awakeableID string) error {
    // Directly resolve without needing a dedicated callback handler
    restate.ResolveAwakeable[bool](ctx, awakeableID, true)
    return nil
}
```

### Key Characteristics of Awakeables:

| Aspect | Details |
|--------|---------|
| **ID Format** | Opaque, globally unique string (e.g., `prom_1McZj3fJ8RTju7BAhyBygXdhwwd`) |
| **Scope** | Global - can be resolved from ANY service or external HTTP call |
| **Use Case** | External coordination: webhooks, human approval, payment callbacks |
| **Resolution** | Via Restate ingress API or `ResolveAwakeable`/`RejectAwakeable` from any handler |
| **Suspension** | Handler suspends at `.Result()`, saving FaaS costs |
| **Service Type** | Works with Basic Services, Virtual Objects (but avoid in exclusive handlers!) |

---

## Internal Signals: `GetInternalSignal` (Durable Promises)

### The `VerifyEmail` Pattern

```go
// 1. WORKFLOW RUN: Main handler waits for internal signal
func (w *OnboardingWorkflow) Run(ctx restate.WorkflowContext, userID string) (string, error) {
    ctx.Log().Info("Starting onboarding", "user", userID)
    
    // Get promise by LOGICAL NAME (not an opaque ID)
    // This is scoped to THIS SPECIFIC workflow instance
    emailSignal := GetInternalSignal[bool](ctx, "email-verified")
    
    // Handler SUSPENDS here
    // Restate records: "waiting for promise 'email-verified' in workflow:user123"
    verificationResult, err := emailSignal.Result()
    if err != nil {
        return "", err
    }
    
    if verificationResult {
        return "Onboarding complete", nil
    }
    return "Onboarding failed", nil
}

// 2. SIGNAL HANDLER: Shared handler resolves the promise
func (w *OnboardingWorkflow) VerifyEmail(ctx restate.WorkflowSharedContext, verified bool) (restate.Void, error) {
    ctx.Log().Info("Received verification signal", "verified", verified)
    
    // Get the SAME promise by name (scoped to this workflow instance)
    emailSignal := GetInternalSignal[bool](ctx, "email-verified")
    
    // Resolve - this RESUMES the Run handler
    if err := emailSignal.Resolve(verified); err != nil {
        return restate.Void{}, err
    }
    
    return restate.Void{}, nil
}
```

### How to Invoke (Service Invocation Perspective)

**Option 1: Via Ingress Client** (External System)
```go
// From external Go application
client := ingress.NewClient("http://restate:8080")

// Must target the SPECIFIC workflow instance by key
_, err := ingress.Workflow[bool, restate.Void](
    client,
    "OnboardingWorkflow",
    "user-123",  // WORKFLOW KEY - identifies the instance
    "VerifyEmail",
).Request(context.Background(), true)  // verified = true
```

**Option 2: Via Internal Context** (From Another Handler)
```go
// From within any Restate handler
func (s *EmailService) ConfirmEmail(ctx restate.Context, userID string) error {
    // Signal the specific workflow instance
    restate.Workflow[bool, restate.Void](
        ctx,
        "OnboardingWorkflow",
        userID,  // Workflow key
        "VerifyEmail",
    ).Send(true)  // One-way signal
    
    return nil
}
```

**Option 3: Via HTTP API**
```bash
# Start the workflow
curl -X POST http://restate:8080/OnboardingWorkflow/user-123/Run \
  -H "Content-Type: application/json" \
  -d '"user-123"'

# Later: signal the workflow
curl -X POST http://restate:8080/OnboardingWorkflow/user-123/VerifyEmail \
  -H "Content-Type: application/json" \
  -d 'true'
```

### Key Characteristics of Promises:

| Aspect | Details |
|--------|---------|
| **ID Format** | Logical name string (e.g., `"email-verified"`) |
| **Scope** | Local to specific workflow instance - scoped by workflow key |
| **Use Case** | Internal coordination: multi-step workflows, sub-task completion |
| **Resolution** | Only via handlers in the SAME workflow instance using shared/exclusive context |
| **Service Type** | **ONLY Workflows** - not available in Services or Virtual Objects |
| **Invocation** | Must specify workflow key to target correct instance |

---

## Critical Differences Summary

```go
// AWAKEABLE (External Signal)
// ✓ Global scope - can be resolved from ANYWHERE
// ✓ Works in Services, Objects, Workflows
// ✗ Must manage opaque ID
awakeable := restate.Awakeable[T](ctx)
externalSystemReceives(awakeable.Id())  // Pass to webhook, email, etc.

// Later, from ANYWHERE:
restate.ResolveAwakeable[T](anyCtx, awakeableID, result)

// ═══════════════════════════════════════════════

// PROMISE (Internal Signal)  
// ✓ Logical names - easy to reference
// ✓ Scoped to workflow instance
// ✗ ONLY in Workflows
// ✗ Must know workflow key to signal
promise := restate.Promise[T](workflowCtx, "my-signal")

// Later, from SAME workflow instance:
promise.Resolve(result)
```

---

## Real-World Invocation Examples

### Awakeable Flow (Payment Gateway Callback)

```go
// 1. Handler creates awakeable
func (s *PaymentService) ProcessPayment(ctx restate.Context, amount int) (string, error) {
    awakeable := WaitForExternalSignal[PaymentResult](ctx)
    
    // Send payment request to external gateway with callback URL
    restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
        paymentGateway.InitiatePayment(PaymentRequest{
            Amount: amount,
            CallbackURL: fmt.Sprintf("https://my-restate.com/awakeables/%s/resolve", 
                awakeable.Id()),
        })
        return restate.Void{}, nil
    })
    
    // Suspend until gateway calls back
    result, err := awakeable.Result()
    return result.TransactionID, err
}

// 2. Payment gateway hits the callback URL
// POST https://my-restate.com/awakeables/prom_abc123/resolve
// Body: {"status": "success", "transactionID": "txn_456"}
// Restate automatically resumes the handler!
```

### Promise Flow (Multi-Stage Workflow)

```go
// 1. Start workflow
client := ingress.NewClient("http://restate:8080")
ingress.Workflow[OrderRequest, string](
    client,
    "OrderWorkflow",
    "order-789",  // KEY!
    "Run",
).Send(context.Background(), OrderRequest{Items: []string{"book"}})

// 2. External system signals payment completion
ingress.Workflow[PaymentConfirmation, restate.Void](
    client,
    "OrderWorkflow", 
    "order-789",  // SAME KEY!
    "ConfirmPayment",
).Send(context.Background(), PaymentConfirmation{Amount: 29.99})

// 3. Warehouse signals shipment
ingress.Workflow[ShipmentInfo, restate.Void](
    client,
    "OrderWorkflow",
    "order-789",  // SAME KEY!
    "ConfirmShipment",
).Send(context.Background(), ShipmentInfo{TrackingNumber: "1Z999"})
```

---

## Best Practice Decision Tree

```
Need to wait for something?
│
├─ External system (webhook, human, 3rd party)?
│  └─ Use WaitForExternalSignal (Awakeable)
│     • Can be resolved via HTTP ingress
│     • ID must be passed to external system
│     • Works in any service type
│
└─ Internal workflow coordination?
   └─ Use GetInternalSignal (Promise)
      • Only in Workflows
      • Must know workflow key to signal
      • Clean logical names
      • Scoped to instance
```

The key insight: **Awakeables cross boundaries** (external systems → Restate), while **Promises stay internal** (Restate workflow handlers → same workflow instance).