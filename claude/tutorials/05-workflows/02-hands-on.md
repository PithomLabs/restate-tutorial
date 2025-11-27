# Hands-On: Building a Document Approval Workflow

> **Create a real-world approval workflow with human-in-the-loop**

## 🎯 What We're Building

A **Document Approval Workflow** featuring:
- Submit documents for review
- Wait for manager approval (may take hours/days)
- Handle approval/rejection/timeout scenarios
- Track workflow status
- Send notifications

**Real-World Scenario:** Employee submits expense report → Manager approves → Process payment

## 📋 Prerequisites

- ✅ Completed [Module 04](../04-virtual-objects/README.md)
- ✅ Understanding of workflows from [concepts](./01-concepts.md)
- ✅ Restate server running

## 🚀 Step-by-Step Tutorial

### Step 1: Project Setup

```bash
# Create project directory
mkdir -p ~/restate-tutorials/module05
cd ~/restate-tutorials/module05

# Initialize Go module
go mod init module05

# Install dependencies
go get github.com/restatedev/sdk-go
```

### Step 2: Define Data Structures

Create `types.go`:

```go
package main

import "time"

// Document submitted for approval
type Document struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Author      string    `json:"author"`
	SubmittedAt time.Time `json:"submittedAt"`
}

// Approval decision from manager
type ApprovalDecision struct {
	Approved bool   `json:"approved"`
	Approver string `json:"approver"`
	Comments string `json:"comments"`
	DecidedAt time.Time `json:"decidedAt"`
}

// Final result of the workflow
type ApprovalResult struct {
	Status    string    `json:"status"` // "approved", "rejected", "timeout"
	Approver  string    `json:"approver,omitempty"`
	Comments  string    `json:"comments,omitempty"`
	CompletedAt time.Time `json:"completedAt"`
}

// Workflow status for querying
type WorkflowStatus struct {
	DocumentID  string    `json:"documentId"`
	Status      string    `json:"status"` // "pending", "approved", "rejected", "timeout"
	Approver    string    `json:"approver,omitempty"`
	SubmittedAt time.Time `json:"submittedAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}
```

### Step 3: Implement the Workflow

Create `approval_workflow.go`:

```go
package main

import (
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
)

type ApprovalWorkflow struct{}

const (
	stateKeyStatus = "status"
	promiseNameApproval = "approval"
	approvalTimeout = 48 * time.Hour
)

// ============================================
// Main Workflow Handler
// ============================================

// Run executes the approval workflow
func (ApprovalWorkflow) Run(
	ctx restate.WorkflowContext,
	doc Document,
) (ApprovalResult, error) {
	docID := restate.Key(ctx)
	
	ctx.Log().Info("Starting approval workflow",
		"docId", docID,
		"author", doc.Author,
		"title", doc.Title)

	// Save initial status
	status := WorkflowStatus{
		DocumentID:  docID,
		Status:      "pending",
		SubmittedAt: time.Now(),
	}
	restate.Set(ctx, stateKeyStatus, status)

	// Send notification to approver (side effect)
	err := sendNotification(ctx, doc)
	if err != nil {
		ctx.Log().Error("Failed to send notification", "error", err)
		// Don't fail workflow if notification fails
	}

	// Create promise for approval decision
	approvalPromise := restate.Promise[ApprovalDecision](ctx, promiseNameApproval)
	
	// Create timeout (48 hours)
	timeoutFuture := restate.After(ctx, approvalTimeout)

	ctx.Log().Info("Waiting for approval", "timeout", approvalTimeout)

	// Wait for either approval or timeout
	winner, err := restate.WaitFirst(ctx, approvalPromise, timeoutFuture)
	if err != nil {
		return ApprovalResult{}, fmt.Errorf("error waiting: %w", err)
	}

	// Handle result based on which completed first
	var result ApprovalResult
	
	switch winner {
	case approvalPromise:
		// Got approval/rejection before timeout
		decision, err := approvalPromise.Result()
		if err != nil {
			// Promise was rejected
			ctx.Log().Error("Approval promise rejected", "error", err)
			result = ApprovalResult{
				Status:      "error",
				Comments:    err.Error(),
				CompletedAt: time.Now(),
			}
		} else if decision.Approved {
			// Approved!
			ctx.Log().Info("Document approved",
				"approver", decision.Approver,
				"comments", decision.Comments)
			
			result = ApprovalResult{
				Status:      "approved",
				Approver:    decision.Approver,
				Comments:    decision.Comments,
				CompletedAt: time.Now(),
			}
		} else {
			// Rejected
			ctx.Log().Info("Document rejected",
				"approver", decision.Approver,
				"comments", decision.Comments)
			
			result = ApprovalResult{
				Status:      "rejected",
				Approver:    decision.Approver,
				Comments:    decision.Comments,
				CompletedAt: time.Now(),
			}
		}

	case timeoutFuture:
		// Timed out - no response in 48 hours
		ctx.Log().Warn("Approval timed out", "timeout", approvalTimeout)
		
		result = ApprovalResult{
			Status:      "timeout",
			Comments:    fmt.Sprintf("No response within %v", approvalTimeout),
			CompletedAt: time.Now(),
		}
	}

	// Update final status
	status.Status = result.Status
	status.Approver = result.Approver
	completedAt := result.CompletedAt
	status.CompletedAt = &completedAt
	restate.Set(ctx, stateKeyStatus, status)

	ctx.Log().Info("Workflow completed", "status", result.Status)

	return result, nil
}

// ============================================
// Shared Handlers (External Interactions)
// ============================================

// Approve resolves the approval promise with approval
func (ApprovalWorkflow) Approve(
	ctx restate.WorkflowSharedContext,
	decision ApprovalDecision,
) error {
	docID := restate.Key(ctx)
	
	ctx.Log().Info("Approving document",
		"docId", docID,
		"approver", decision.Approver,
		"approved", decision.Approved)

	// Set timestamp
	decision.DecidedAt = time.Now()

	// Resolve the promise - this resumes the workflow!
	err := restate.Promise[ApprovalDecision](ctx, promiseNameApproval).
		Resolve(decision)
	if err != nil {
		return fmt.Errorf("failed to resolve approval: %w", err)
	}

	return nil
}

// Reject resolves the approval promise with rejection
func (ApprovalWorkflow) Reject(
	ctx restate.WorkflowSharedContext,
	decision ApprovalDecision,
) error {
	docID := restate.Key(ctx)
	
	ctx.Log().Info("Rejecting document",
		"docId", docID,
		"approver", decision.Approver)

	// Set timestamp and ensure approved is false
	decision.DecidedAt = time.Now()
	decision.Approved = false

	// Resolve the promise with rejection
	err := restate.Promise[ApprovalDecision](ctx, promiseNameApproval).
		Resolve(decision)
	if err != nil {
		return fmt.Errorf("failed to resolve rejection: %w", err)
	}

	return nil
}

// GetStatus returns the current workflow status (read-only)
func (ApprovalWorkflow) GetStatus(
	ctx restate.WorkflowSharedContext,
	_ restate.Void,
) (WorkflowStatus, error) {
	docID := restate.Key(ctx)
	
	// Get current status
	status, err := restate.Get[WorkflowStatus](ctx, stateKeyStatus)
	if err != nil {
		return WorkflowStatus{}, err
	}

	ctx.Log().Info("Status retrieved", "docId", docID, "status", status.Status)

	return status, nil
}

// ============================================
// Helper Functions
// ============================================

func sendNotification(ctx restate.WorkflowContext, doc Document) error {
	// Send email notification using restate.Run
	_, err := restate.Run(ctx, func(ctx restate.RunContext) (bool, error) {
		// In real app: send email via SendGrid, SES, etc.
		ctx.Log().Info("Sending approval notification",
			"docId", doc.ID,
			"author", doc.Author,
			"title", doc.Title)
		
		// Simulate email sending
		// emailService.Send(...)
		
		return true, nil
	})

	return err
}
```

### Step 4: Create Main Entry Point

Create `main.go`:

```go
package main

import (
	"context"
	"fmt"
	"log"

	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/server"
)

func main() {
	restateServer := server.NewRestate()

	// Register Approval Workflow
	if err := restateServer.Bind(restate.Reflect(ApprovalWorkflow{})); err != nil {
		log.Fatal("Failed to bind ApprovalWorkflow:", err)
	}

	fmt.Println("📄 Starting Approval Workflow Service on :9090...")
	fmt.Println("🔄 Workflow: ApprovalWorkflow")
	fmt.Println("")
	fmt.Println("Handlers:")
	fmt.Println("  Main:")
	fmt.Println("    - Run (workflow orchestration)")
	fmt.Println("")
	fmt.Println("  Shared (external interactions):")
	fmt.Println("    - Approve")
	fmt.Println("    - Reject")
	fmt.Println("    - GetStatus")
	fmt.Println("")
	fmt.Println("✓ Ready to accept requests")

	if err := restateServer.Start(context.Background(), ":9090"); err != nil {
		log.Fatal("Server error:", err)
	}
}
```

### Step 5: Build and Run

```bash
# Build
go mod tidy
go build -o approval-service

# Run
./approval-service
```

### Step 6: Register with Restate

```bash
curl -X POST http://localhost:8080/deployments \
  -H 'Content-Type: application/json' \
  -d '{"uri": "http://localhost:9090"}'
```

### Step 7: Test the Workflow

#### Start the Workflow

```bash
curl -X POST http://localhost:9080/ApprovalWorkflow/doc-001/Run \
  -H 'Content-Type: application/json' \
  -d '{
    "id": "doc-001",
    "title": "Q4 Budget Proposal",
    "content": "Requesting $50,000 for new equipment",
    "author": "alice",
    "submittedAt": "2024-01-15T10:00:00Z"
  }'
```

**Response:** Request accepted (workflow starts in background)

**Important:** This call returns immediately! The workflow runs asynchronously.

#### Check Workflow Status

```bash
curl -X POST http://localhost:9080/ApprovalWorkflow/doc-001/GetStatus \
  -H 'Content-Type: application/json' \
  -d 'null'
```

**Expected Response:**
```json
{
  "documentId": "doc-001",
  "status": "pending",
  "submittedAt": "2024-01-15T10:00:00Z"
}
```

**Key Observation:** Workflow is waiting for approval!

#### Approve the Document (Manager Action)

```bash
curl -X POST http://localhost:9080/ApprovalWorkflow/doc-001/Approve \
  -H 'Content-Type: application/json' \
  -d '{
    "approved": true,
    "approver": "bob-manager",
    "comments": "Budget looks reasonable, approved!"
  }'
```

**What Happens:**
1. Promise is resolved
2. Workflow resumes from `promise.Result()`
3. Workflow completes with "approved" status

#### Check Final Status

```bash
curl -X POST http://localhost:9080/ApprovalWorkflow/doc-001/GetStatus \
  -H 'Content-Type: application/json' \
  -d 'null'
```

**Expected Response:**
```json
{
  "documentId": "doc-001",
  "status": "approved",
  "approver": "bob-manager",
  "submittedAt": "2024-01-15T10:00:00Z",
  "completedAt": "2024-01-15T14:30:00Z"
}
```

### Step 8: Test Rejection Flow

```bash
# Start new workflow
curl -X POST http://localhost:9080/ApprovalWorkflow/doc-002/Run \
  -H 'Content-Type: application/json' \
  -d '{
    "id": "doc-002",
    "title": "Expensive Office Chairs",
    "content": "Requesting $100,000 for gold-plated chairs",
    "author": "alice",
    "submittedAt": "2024-01-15T11:00:00Z"
  }'

# Reject it
curl -X POST http://localhost:9080/ApprovalWorkflow/doc-002/Reject \
  -H 'Content-Type: application/json' \
  -d '{
    "approved": false,
    "approver": "bob-manager",
    "comments": "Too expensive, please revise"
  }'

# Check status
curl -X POST http://localhost:9080/ApprovalWorkflow/doc-002/GetStatus \
  -H 'Content-Type: application/json' \
  -d 'null'
```

**Expected:**
```json
{
  "documentId": "doc-002",
  "status": "rejected",
  "approver": "bob-manager",
  "submittedAt": "2024-01-15T11:00:00Z",
  "completedAt": "2024-01-15T11:05:00Z"
}
```

### Step 9: Test Timeout (Advanced)

For demonstration, we can't wait 48 hours, so let's inspect a pending workflow:

```bash
# Start workflow
curl -X POST http://localhost:9080/ApprovalWorkflow/doc-003/Run \
  -H 'Content-Type: application/json' \
  -d '{
    "id": "doc-003",
    "title": "Test Document",
    "content": "Testing timeout behavior",
    "author": "test-user",
    "submittedAt": "2024-01-15T12:00:00Z"
  }'

# Don't approve or reject - just check status
curl -X POST http://localhost:9080/ApprovalWorkflow/doc-003/GetStatus \
  -H 'Content-Type: application/json' \
  -d 'null'
```

**Expected:** Status remains "pending"

**In Production:** After 48 hours, workflow would auto-complete with "timeout" status

## 🎓 Understanding the Implementation

### Workflow Pattern

```go
func (ApprovalWorkflow) Run(
    ctx restate.WorkflowContext,
    doc Document,
) (ApprovalResult, error) {
    // 1. Save initial state
    restate.Set(ctx, "status", initialStatus)
    
    // 2. Create promise for external event
    promise := restate.Promise[ApprovalDecision](ctx, "approval")
    
    // 3. Create timeout
    timeout := restate.After(ctx, 48*time.Hour)
    
    // 4. Wait for first to complete
    winner, _ := restate.WaitFirst(ctx, promise, timeout)
    
    // 5. Handle result
    switch winner {
    case promise:
        // Got approval
    case timeout:
        // Timed out
    }
}
```

### External Interaction

```go
// Manager calls this from external system
func (ApprovalWorkflow) Approve(
    ctx restate.WorkflowSharedContext,
    decision ApprovalDecision,
) error {
    // Resolve promise - workflow resumes!
    return restate.Promise[ApprovalDecision](ctx, "approval").
        Resolve(decision)
}
```

### Key Workflow ID

```
URL: /ApprovalWorkflow/{workflowId}/Run

doc-001/Run → Workflow with ID "doc-001"
doc-002/Run → Workflow with ID "doc-002" (different workflow)
```

### State Isolation

Each workflow ID has completely isolated state and promises:
- doc-001 has its own "approval" promise
- doc-002 has its own "approval" promise
- They don't interfere with each other

## ✅ Verification Checklist

- [ ] Service starts successfully
- [ ] Can submit documents for approval
- [ ] Workflow enters "pending" state
- [ ] Can check status while pending
- [ ] Approving resolves promise and completes workflow
- [ ] Rejecting resolves promise with rejection
- [ ] Status changes to final state
- [ ] Multiple workflows can run independently

## 💡 Key Takeaways

1. **Async Execution**
   - Starting workflow returns immediately
   - Workflow runs in background
   - Can take hours/days to complete

2. **Durable Promises**
   - Created in Run handler
   - Resolved from external handlers
   - Survives failures while waiting

3. **Timeout Handling**
   - Use `WaitFirst` for timeout pattern
   - Always handle timeout case
   - Prevents workflows from waiting forever

4. **External Interaction**
   - Shared handlers can interact with running workflow
   - Resolve promises to resume workflow
   - Query status without affecting execution

5. **State Management**
   - Track workflow progress in state
   - Update state as workflow progresses
   - Query from shared handlers

## 🎯 Next Steps

Ready to validate your workflow implementation!

👉 **Continue to [Validation](./03-validation.md)**

---

**Questions?** Review [concepts](./01-concepts.md) or check the [complete code](./code/)!
