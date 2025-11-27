# Concepts: Workflows and Durable Promises

> **Understanding long-running orchestrations with external event handling**

## 🎯 What are Workflows?

### Definition

A **Workflow** is a special type of Virtual Object designed for **long-running orchestrations** that:
- Can wait for external events (hours/days/weeks)
- Use **durable promises** to pause and resume
- Support human-in-the-loop patterns
- Maintain state across very long timeframes

Think of Workflows as state machines that can sleep indefinitely while waiting for external input.

### Real-World Analogy

**Virtual Object** = Coffee shop order
- Quick process (minutes)
- All steps happen immediately
- No external waiting

**Workflow** = Home construction project
- Long process (months)
- Waiting for inspections, permits, materials
- Many external dependencies
- Can pause and resume

## 🆚 Virtual Objects vs Workflows

### Virtual Object (Short-Lived)

```go
type ShoppingCart struct{}

// Quick operations
func (ShoppingCart) Checkout(
    ctx restate.ObjectContext,
    _ restate.Void,
) (CheckoutResult, error) {
    // Execute immediately
    // Complete in seconds
    return ProcessPayment(), nil
}
```

**Duration:** Seconds to minutes

### Workflow (Long-Running)

```go
type OrderFulfillment struct{}

// Long-running orchestration
func (OrderFulfillment) Run(
    ctx restate.WorkflowContext,
    order Order,
) (FulfillmentResult, error) {
    // Wait for payment (may take hours)
    payment := restate.Promise[Payment](ctx, "payment")
    result, _ := payment.Result()
    
    // Wait for warehouse (may take days)
    label := restate.Promise[Label](ctx, "label")
    shipping, _ := label.Result()
    
    // May take weeks total
    return FulfillmentResult{...}, nil
}
```

**Duration:** Hours to weeks

## 🔮 Durable Promises

### What is a Durable Promise?

A **durable promise** is a placeholder for a future value that:
- Can be resolved from **outside** the workflow
- Survives failures and restarts
- Can wait indefinitely
- Is journaled by Restate

### Promise Lifecycle

```
1. Create Promise
   promise := restate.Promise[T](ctx, "myPromise")

2. Wait for Value (blocks workflow)
   value, err := promise.Result()
   
3. (External) Resolve Promise
   restate.Promise[T](ctx, "myPromise").Resolve(value)
   
4. Workflow Resumes
   // Continue with value
```

### Visual Flow

```
Workflow:                    External System:
┌─────────────────────┐
│ Start workflow      │
│ Create promise      │
│ Wait...             │────┐
│    (suspended)      │    │
│                     │    │  Time passes
│                     │    │  (hours/days)
│                     │    │
│                     │◄───┘ Resolve promise
│ Resume!             │
│ Continue with value │
│ Complete            │
└─────────────────────┘
```

## 🏗️ Workflow Structure

### Basic Workflow

```go
type MyWorkflow struct{}

// Main workflow logic
func (MyWorkflow) Run(
    ctx restate.WorkflowContext,  // ← WorkflowContext
    input InputType,
) (OutputType, error) {
    // Create promise for external event
    promise := restate.Promise[EventData](ctx, "event")
    
    // Wait for event (may be hours/days)
    data, err := promise.Result()
    if err != nil {
        return OutputType{}, err
    }
    
    // Continue processing
    return OutputType{Data: data}, nil
}

// Shared handler to resolve promise
func (MyWorkflow) SubmitEvent(
    ctx restate.WorkflowSharedContext,  // ← WorkflowSharedContext
    event EventData,
) error {
    // Resolve the promise
    return restate.Promise[EventData](ctx, "event").Resolve(event)
}

// Shared handler to get status
func (MyWorkflow) GetStatus(
    ctx restate.WorkflowSharedContext,
    _ restate.Void,
) (Status, error) {
    // Read workflow state
    state, _ := restate.Get[WorkflowState](ctx, "state")
    return state.Status, nil
}
```

### Context Types

| Context Type | Use Case | Handlers | Features |
|--------------|----------|----------|----------|
| `WorkflowContext` | Main workflow logic | `Run` handler only | Promises, state, calls |
| `WorkflowSharedContext` | Interact with workflow | All other handlers | Resolve promises, read state |

## 📝 Creating and Using Promises

### Create a Promise

```go
// In Run handler
promise := restate.Promise[ApprovalData](ctx, "approval")
```

**Parameters:**
- `ctx` - Workflow context
- `"approval"` - Promise name (unique within workflow)

### Wait for Promise

```go
// Blocks until promise is resolved or rejected
approval, err := promise.Result()
if err != nil {
    // Promise was rejected
    return handleRejection(err)
}

// Use the value
processApproval(approval)
```

### Resolve Promise (External)

```go
// In a shared handler
func (MyWorkflow) Approve(
    ctx restate.WorkflowSharedContext,
    approval ApprovalData,
) error {
    // Resolve the promise - workflow will resume
    return restate.Promise[ApprovalData](ctx, "approval").Resolve(approval)
}
```

### Reject Promise (External)

```go
// In a shared handler
func (MyWorkflow) Reject(
    ctx restate.WorkflowSharedContext,
    reason string,
) error {
    // Reject the promise
    return restate.Promise[ApprovalData](ctx, "approval").
        Reject(fmt.Errorf("rejected: %s", reason))
}
```

## ⏰ Promises with Timeouts

### Using Select Pattern

```go
func (MyWorkflow) Run(
    ctx restate.WorkflowContext,
    input InputType,
) (OutputType, error) {
    // Create promise
    promise := restate.Promise[Approval](ctx, "approval")
    
    // Create timeout
    timeout := restate.After(ctx, 24*time.Hour)
    
    // Wait for first to complete
    winner, err := restate.WaitFirst(ctx, promise, timeout)
    if err != nil {
        return OutputType{}, err
    }
    
    switch winner {
    case promise:
        // Got approval in time
        approval, _ := promise.Result()
        return OutputType{Approved: true, Data: approval}, nil
        
    case timeout:
        // Timed out
        return OutputType{Approved: false, Reason: "timeout"}, nil
    }
    
    return OutputType{}, nil
}
```

**Key Points:**
- Use `restate.After()` for durable sleep
- `WaitFirst()` returns when first completes
- Handle both success and timeout cases

## 🎭 Human-in-the-Loop Pattern

### Complete Example

```go
type DocumentApproval struct{}

type Document struct {
    ID      string `json:"id"`
    Content string `json:"content"`
    Author  string `json:"author"`
}

type Approval struct {
    Approved  bool   `json:"approved"`
    Approver  string `json:"approver"`
    Comments  string `json:"comments"`
}

type ApprovalResult struct {
    Status    string `json:"status"` // "approved", "rejected", "timeout"
    Approver  string `json:"approver,omitempty"`
    Comments  string `json:"comments,omitempty"`
}

// Main workflow
func (DocumentApproval) Run(
    ctx restate.WorkflowContext,
    doc Document,
) (ApprovalResult, error) {
    docID := restate.Key(ctx)
    
    ctx.Log().Info("Approval workflow started", "docId", docID)
    
    // Send notification to approver (side effect)
    _, err := restate.Run(ctx, func(ctx restate.RunContext) (bool, error) {
        return sendEmailNotification(doc), nil
    })
    if err != nil {
        return ApprovalResult{}, err
    }
    
    // Create promise for approval
    promise := restate.Promise[Approval](ctx, "approval")
    
    // Wait up to 48 hours for approval
    timeout := restate.After(ctx, 48*time.Hour)
    
    winner, err := restate.WaitFirst(ctx, promise, timeout)
    if err != nil {
        return ApprovalResult{}, err
    }
    
    switch winner {
    case promise:
        // Got approval
        approval, _ := promise.Result()
        if approval.Approved {
            return ApprovalResult{
                Status:   "approved",
                Approver: approval.Approver,
                Comments: approval.Comments,
            }, nil
        } else {
            return ApprovalResult{
                Status:   "rejected",
                Approver: approval.Approver,
                Comments: approval.Comments,
            }, nil
        }
        
    case timeout:
        // Timed out - auto-reject
        return ApprovalResult{
            Status:   "timeout",
            Comments: "No response within 48 hours",
        }, nil
    }
    
    return ApprovalResult{}, nil
}

// Approve the document (called by manager)
func (DocumentApproval) Approve(
    ctx restate.WorkflowSharedContext,
    approval Approval,
) error {
    docID := restate.Key(ctx)
    ctx.Log().Info("Document approved", "docId", docID, "approver", approval.Approver)
    
    // Resolve promise - workflow will resume
    return restate.Promise[Approval](ctx, "approval").Resolve(approval)
}

// Get current status
func (DocumentApproval) GetStatus(
    ctx restate.WorkflowSharedContext,
    _ restate.Void,
) (string, error) {
    // Check if workflow has completed
    // (In real implementation, track state)
    return "pending", nil
}
```

### Using the Workflow

```go
// Start workflow (non-blocking)
doc := Document{
    ID:      "doc123",
    Content: "Proposal for new feature",
    Author:  "alice",
}

restate.WorkflowSend(ctx, "DocumentApproval", "doc123", "Run").
    Send(doc)

// Later, manager approves (from separate request)
approval := Approval{
    Approved: true,
    Approver: "bob",
    Comments: "Looks good!",
}

err := restate.Workflow[error](ctx, "DocumentApproval", "doc123", "Approve").
    Request(approval)

// Workflow resumes and completes!
```

## 🔄 Workflow Lifecycle

### States

```
Created → Running → Waiting → Completed
    ↓         ↓        ↓
  Initial   Active   Suspended
```

### Timeline Example

```
Time 0:00   - Workflow starts (Run handler called)
Time 0:01   - Create promise, start waiting
Time 0:01   - Workflow suspended (waiting for promise)
            - (Service can restart, workflow stays suspended)
Time 12:30  - External approval received
Time 12:30  - Promise resolved, workflow resumes
Time 12:31  - Workflow completes
```

**Key Observation:** Workflow was suspended for 12+ hours!

## 📦 Workflow State Management

### Using State

Workflows can use state just like Virtual Objects:

```go
func (MyWorkflow) Run(
    ctx restate.WorkflowContext,
    input InputType,
) (OutputType, error) {
    // Save state
    state := WorkflowState{
        Status:    "waiting_approval",
        CreatedAt: time.Now(),
    }
    restate.Set(ctx, "state", state)
    
    // Wait for promise
    promise := restate.Promise[Data](ctx, "event")
    data, _ := promise.Result()
    
    // Update state
    state.Status = "processing"
    state.UpdatedAt = time.Now()
    restate.Set(ctx, "state", state)
    
    // Continue...
}
```

### Reading State from Shared Handlers

```go
func (MyWorkflow) GetStatus(
    ctx restate.WorkflowSharedContext,
    _ restate.Void,
) (WorkflowState, error) {
    state, _ := restate.Get[WorkflowState](ctx, "state")
    return state, nil
}
```

## ⚠️ Common Patterns and Anti-Patterns

### ✅ Correct: Promise Per Event

```go
func (MyWorkflow) Run(ctx restate.WorkflowContext, input Input) (Output, error) {
    // One promise for payment
    payment := restate.Promise[Payment](ctx, "payment")
    paymentData, _ := payment.Result()
    
    // Another promise for shipping
    shipping := restate.Promise[Shipping](ctx, "shipping")
    shippingData, _ := shipping.Result()
    
    return combineResults(paymentData, shippingData), nil
}
```

### ❌ Anti-Pattern: Reusing Promise Names

```go
// ❌ WRONG - Don't reuse promise names!
func (MyWorkflow) Run(ctx restate.WorkflowContext, input Input) (Output, error) {
    // First event
    event1 := restate.Promise[Data](ctx, "event")
    data1, _ := event1.Result()
    
    // ❌ Can't reuse "event" name!
    event2 := restate.Promise[Data](ctx, "event")  // ERROR
    data2, _ := event2.Result()
}

// ✅ CORRECT - Use unique names
func (MyWorkflow) Run(ctx restate.WorkflowContext, input Input) (Output, error) {
    event1 := restate.Promise[Data](ctx, "event1")
    data1, _ := event1.Result()
    
    event2 := restate.Promise[Data](ctx, "event2")
    data2, _ := event2.Result()
}
```

### ✅ Correct: Timeout Handling

```go
func (MyWorkflow) Run(ctx restate.WorkflowContext, input Input) (Output, error) {
    promise := restate.Promise[Data](ctx, "data")
    timeout := restate.After(ctx, 1*time.Hour)
    
    winner, _ := restate.WaitFirst(ctx, promise, timeout)
    
    switch winner {
    case promise:
        data, _ := promise.Result()
        return processData(data), nil
    case timeout:
        return handleTimeout(), nil
    }
}
```

### ❌ Anti-Pattern: No Timeout

```go
// ❌ Risky - could wait forever!
func (MyWorkflow) Run(ctx restate.WorkflowContext, input Input) (Output, error) {
    promise := restate.Promise[Data](ctx, "data")
    data, _ := promise.Result()  // What if promise is never resolved?
    return processData(data), nil
}
```

## 🎯 When to Use Workflows

### Use Workflows When:

✅ **Long duration** (hours/days/weeks)
✅ **External events** (human approvals, webhooks)
✅ **Complex orchestration** (multiple async steps)
✅ **Timeout handling** needed

**Examples:**
- Document approval (wait for manager)
- Order fulfillment (wait for warehouse)
- User onboarding (wait for email verification)
- Loan application (wait for credit check)

### Use Virtual Objects When:

✅ **Short duration** (seconds/minutes)
✅ **Immediate processing**
✅ **CRUD operations**
✅ **No external waiting**

**Examples:**
- Shopping cart
- User profile
- Inventory tracking
- Counters

### Use Basic Services When:

✅ **Stateless**
✅ **Quick computation**
✅ **No state needed**

**Examples:**
- API aggregation
- Format conversion
- Calculations

## 🔍 Advanced: Multiple Promises

### Sequential Promises

```go
func (MyWorkflow) Run(ctx restate.WorkflowContext, order Order) (Result, error) {
    // Wait for payment first
    payment := restate.Promise[Payment](ctx, "payment")
    paymentData, _ := payment.Result()
    
    // Then wait for inventory
    inventory := restate.Promise[Inventory](ctx, "inventory")
    inventoryData, _ := inventory.Result()
    
    // Then wait for shipping
    shipping := restate.Promise[Shipping](ctx, "shipping")
    shippingData, _ := shipping.Result()
    
    return Result{...}, nil
}
```

### Parallel Promises

```go
func (MyWorkflow) Run(ctx restate.WorkflowContext, order Order) (Result, error) {
    // Create all promises
    payment := restate.Promise[Payment](ctx, "payment")
    inventory := restate.Promise[Inventory](ctx, "inventory")
    shipping := restate.Promise[Shipping](ctx, "shipping")
    
    // Wait for all
    for fut, err := range restate.Wait(ctx, payment, inventory, shipping) {
        if err != nil {
            return Result{}, err
        }
    }
    
    // All resolved - continue
    paymentData, _ := payment.Result()
    inventoryData, _ := inventory.Result()
    shippingData, _ := shipping.Result()
    
    return Result{...}, nil
}
```

## ✅ Concept Check

Before moving to hands-on, ensure you understand:

- [ ] Difference between Workflows and Virtual Objects
- [ ] What durable promises are and how they work
- [ ] How to create and wait for promises
- [ ] How to resolve/reject promises from external handlers
- [ ] Implementing timeouts with `WaitFirst`
- [ ] When to use Workflows vs Virtual Objects
- [ ] Promise naming and uniqueness

## 🎯 Next Step

Ready to build a real approval workflow!

👉 **Continue to [Hands-On Tutorial](./02-hands-on.md)**

---

**Key Takeaway:** Workflows enable durable, long-running orchestrations with natural async/await patterns using durable promises!
