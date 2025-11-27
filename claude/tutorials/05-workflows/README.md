# Module 05: Workflows - Long-Running Orchestrations

> **Build durable workflows with human-in-the-loop and async await patterns**

## 🎯 Module Overview

Workflows extend Virtual Objects with special capabilities for **long-running orchestrations** that may take hours, days, or even weeks. They're perfect for processes involving human approvals, waiting for external events, or complex multi-step operations.

### What You'll Learn

- ✅ What Workflows are and when to use them
- ✅ Durable promises for external events
- ✅ Human-in-the-loop patterns
- ✅ Workflow state and lifecycle
- ✅ Building approval workflows
- ✅ Handling timeouts and deadlines

### Real-World Use Cases

- 📝 **Approval Workflows** - Document reviews, expense approvals
- 🏭 **Order Fulfillment** - Multi-day order processing
- 👤 **User Onboarding** - Multi-step signup with email verification
- 🔄 **Data Migration** - Long-running ETL jobs
- 📧 **Campaign Management** - Drip email sequences over weeks

## 📊 Conceptual Comparison

| Feature | Virtual Object | Workflow |
|---------|----------------|----------|
| **State** | Durable per key | Durable per key |
| **Duration** | Short (seconds/minutes) | Long (hours/days/weeks) |
| **Context** | `ObjectContext` | `WorkflowContext` |
| **Special Feature** | - | Durable promises |
| **Pattern** | CRUD operations | Long-running orchestrations |
| **Use Case** | Shopping carts | Approval processes |

## 🏗️ Module Structure

### 1. [Concepts](./01-concepts.md) (~30 min)
Learn about:
- Workflow fundamentals
- Durable promises
- Human-in-the-loop patterns
- Workflow lifecycle

### 2. [Hands-On](./02-hands-on.md) (~45 min)
Build a multi-step approval workflow:
- Submit document for review
- Wait for human approval
- Handle timeout scenarios
- Track workflow status

### 3. [Validation](./03-validation.md) (~20 min)
Test:
- Promise resolution from external handlers
- Timeout handling
- Workflow state persistence
- Long-running execution

### 4. [Exercises](./04-exercises.md) (~60 min)
Practice building:
- Order fulfillment workflow
- User onboarding with email verification
- Multi-approver workflow
- Scheduled task execution

## 🎓 Prerequisites

- ✅ Completed [Module 04](../04-virtual-objects/README.md) - Virtual Objects
- ✅ Understanding of async patterns
- ✅ Familiarity with promises/futures

## 🚀 Quick Start

```bash
# Navigate to module directory
cd ~/restate-tutorials/module05

# Follow hands-on tutorial
cat 02-hands-on.md
```

## 🎯 Learning Objectives

By the end of this module, you will:

1. **Understand Workflows**
   - What they are and when to use them
   - How they differ from Virtual Objects
   - Workflow lifecycle management

2. **Master Durable Promises**
   - Create promises that survive failures
   - Resolve/reject from external code
   - Wait with timeouts

3. **Implement Human-in-the-Loop**
   - Pause workflows for human input
   - Handle async approvals
   - Manage workflow timeouts

4. **Build Real Workflows**
   - Approval processes
   - Order fulfillment
   - User onboarding
   - Multi-step orchestrations

## 📖 Module Flow

```
Concepts → Hands-On → Validation → Exercises
   ↓          ↓          ↓            ↓
Theory → Build Approval → Test Promises → Practice
```

## 🔑 Key Concept Preview

### Workflow Declaration

```go
type ApprovalWorkflow struct{}

// Main workflow handler - runs the orchestration
func (ApprovalWorkflow) Run(
    ctx restate.WorkflowContext,
    document Document,
) (ApprovalResult, error) {
    // Create a promise for human approval
    promise := restate.Promise[Approval](ctx, "approval")
    
    // Wait for approval (with timeout)
    approval, err := promise.Result()
    if err != nil {
        return ApprovalResult{Status: "timeout"}, nil
    }
    
    return ApprovalResult{
        Status:   "approved",
        Approver: approval.Approver,
    }, nil
}

// External handler to resolve the promise
func (ApprovalWorkflow) Approve(
    ctx restate.WorkflowSharedContext,
    approval Approval,
) error {
    // Resolve the promise from external request
    return restate.Promise[Approval](ctx, "approval").Resolve(approval)
}
```

### Calling a Workflow

```go
// Start workflow (non-blocking)
restate.WorkflowSend(ctx, "ApprovalWorkflow", "doc123", "Run").
    Send(document)

// Later, approve it (from external system)
restate.Workflow[error](ctx, "ApprovalWorkflow", "doc123", "Approve").
    Request(Approval{Approver: "manager", Approved: true})
```

## 💡 Why Workflows?

**Before (Virtual Object):**
```go
// Can't wait for external events naturally
func ProcessOrder(ctx restate.ObjectContext, order Order) {
    // How do we wait for payment confirmation?
    // How do we wait for shipping label?
    // Requires complex polling or callback systems
}
```

**After (Workflow):**
```go
// Naturally wait for external events
func Run(ctx restate.WorkflowContext, order Order) {
    // Wait for payment (may take hours)
    payment := restate.Promise[Payment](ctx, "payment")
    result, _ := payment.Result()
    
    // Wait for shipping label (may take days)
    label := restate.Promise[Label](ctx, "label")
    shippingLabel, _ := label.Result()
    
    // Complete order
}
```

**Benefits:**
- ⏱️ **Long-Running** - Can wait days/weeks
- 🎯 **Natural** - Write sequential code for async processes
- 💾 **Durable** - Survives failures while waiting
- 🔔 **Event-Driven** - React to external events

## ⚠️ Important Notes

### Workflow Run Handler

The `Run` handler executes the main workflow logic:

```go
func (MyWorkflow) Run(
    ctx restate.WorkflowContext,  // ← WorkflowContext
    input InputType,
) (OutputType, error) {
    // Workflow orchestration logic
}
```

**Key Points:**
- Exclusive execution (one run per workflow ID)
- Can use promises
- Can call other services
- Can sleep for long periods

### Shared Handlers

Additional handlers to interact with the workflow:

```go
func (MyWorkflow) GetStatus(
    ctx restate.WorkflowSharedContext,  // ← WorkflowSharedContext  
    _ restate.Void,
) (Status, error) {
    // Read workflow state (concurrent)
}

func (MyWorkflow) Approve(
    ctx restate.WorkflowSharedContext,
    approval Approval,
) error {
    // Resolve promise (concurrent)
    return restate.Promise[Approval](ctx, "approval").Resolve(approval)
}
```

## 🎯 Ready to Start?

Let's dive into workflow concepts!

👉 **Start with [Concepts](./01-concepts.md)**

---

**Questions?** Check the main [tutorials README](../README.md) or review previous modules.
