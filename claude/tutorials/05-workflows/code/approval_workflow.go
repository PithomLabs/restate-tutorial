package main

import (
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
)

type ApprovalWorkflow struct{}

const (
	stateKeyStatus      = "status"
	promiseNameApproval = "approval"
	approvalTimeout     = 48 * time.Hour
)

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
