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
	Approved  bool      `json:"approved"`
	Approver  string    `json:"approver"`
	Comments  string    `json:"comments"`
	DecidedAt time.Time `json:"decidedAt"`
}

// Final result of the workflow
type ApprovalResult struct {
	Status      string    `json:"status"` // "approved", "rejected", "timeout"
	Approver    string    `json:"approver,omitempty"`
	Comments    string    `json:"comments,omitempty"`
	CompletedAt time.Time `json:"completedAt"`
}

// Workflow status for querying
type WorkflowStatus struct {
	DocumentID  string     `json:"documentId"`
	Status      string     `json:"status"` // "pending", "approved", "rejected", "timeout"
	Approver    string     `json:"approver,omitempty"`
	SubmittedAt time.Time  `json:"submittedAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}
