# Module 05 - Approval Workflow with Durable Promises

This directory contains a complete document approval workflow demonstrating long-running orchestrations with Restate Workflows.

## 📂 Files

- `main.go` - Server initialization
- `types.go` - Data structures
- `approval_workflow.go` - Workflow implementation with promises
- `go.mod` - Dependencies

## 🚀 Quick Start

```bash
# Build
go mod tidy
go build -o approval-service

# Run
./approval-service
```

Service starts on port 9090.

## 📋 Register with Restate

```bash
curl -X POST http://localhost:8080/deployments \
  -H 'Content-Type: application/json' \
  -d '{"uri": "http://localhost:9090"}'
```

## 🧪 Test

### Start Workflow

```bash
curl -X POST http://localhost:9080/ApprovalWorkflow/doc-001/Run \
  -H 'Content-Type: application/json' \
  -d '{
    "id": "doc-001",
    "title": "Budget Proposal",
    "content": "Requesting funds",
    "author": "alice",
    "submittedAt": "2024-01-15T10:00:00Z"
  }'
```

### Check Status

```bash
curl -X POST http://localhost:9080/ApprovalWorkflow/doc-001/GetStatus \
  -H 'Content-Type: application/json' \
  -d 'null'
```

### Approve Document

```bash
curl -X POST http://localhost:9080/ApprovalWorkflow/doc-001/Approve \
  -H 'Content-Type: application/json' \
  -d '{
    "approved": true,
    "approver": "bob-manager",
    "comments": "Approved!"
  }'
```

### Reject Document

```bash
curl -X POST http://localhost:9080/ApprovalWorkflow/doc-002/Reject \
  -H 'Content-Type: application/json' \
  -d '{
    "approved": false,
    "approver": "bob-manager",
    "comments": "Needs revision"
  }'
```

## 🎓 Key Concepts Demonstrated

### Workflow Structure

```go
func (ApprovalWorkflow) Run(
    ctx restate.WorkflowContext,  // ← WorkflowContext
    doc Document,
) (ApprovalResult, error) {
    // Create durable promise
    promise := restate.Promise[ApprovalDecision](ctx, "approval")
    
    // Wait with timeout
    timeout := restate.After(ctx, 48*time.Hour)
    winner, _ := restate.WaitFirst(ctx, promise, timeout)
    
    // Handle result
}
```

### Durable Promises

```go
// Create promise for external event
promise := restate.Promise[ApprovalDecision](ctx, "approval")

// Wait for resolution
decision, err := promise.Result()
```

### External Resolution

```go
// External handler resolves promise
func (ApprovalWorkflow) Approve(
    ctx restate.WorkflowSharedContext,
    decision ApprovalDecision,
) error {
    return restate.Promise[ApprovalDecision](ctx, "approval").
        Resolve(decision)
}
```

### Timeout Pattern

```go
// Race promise against timeout
promise := restate.Promise[T](ctx, "event")
timeout := restate.After(ctx, 48*time.Hour)

winner, _ := restate.WaitFirst(ctx, promise, timeout)

switch winner {
case promise:
    // Got result in time
case timeout:
    // Timed out
}
```

## 💡 Features

1. **Long-Running** - Can wait days for approval
2. **Durable Promises** - Survive failures while waiting
3. **External Events** - Manager approves from separate request
4. **Timeout Handling** - Auto-reject after 48 hours
5. **Status Queries** - Check workflow state anytime
6. **State Management** - Track workflow progress

## 📊 Workflow Handlers

### Main Handler
- `Run` - Execute workflow orchestration (WorkflowContext)

### Shared Handlers  
- `Approve` - Resolve promise with approval
- `Reject` - Resolve promise with rejection
- `GetStatus` - Query current status

## 🔍 Workflow Timeline

```
Time 0:00    - Workflow starts
Time 0:01    - Promise created, workflow suspended
             - (Can wait hours/days...)
Time 12:30   - Manager approves (external call)
Time 12:30   - Promise resolved, workflow resumes
Time 12:31   - Workflow completes
```

## 🎯 Next Steps

- Complete [validation tests](../03-validation.md)
- Try [exercises](../04-exercises.md)
- Build your own workflows

---

**Questions?** See the main [hands-on tutorial](../02-hands-on.md)!
