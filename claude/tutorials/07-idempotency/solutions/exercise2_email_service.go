package main

import (
	"fmt"
	"math/rand"
	"time"

	restate "github.com/restatedev/sdk-go"
)

type EmailService struct{}

type EmailRequest struct {
	Recipient string `json:"recipient"`
	Subject   string `json:"subject"`
	Body      string `json:"body"`
}

type EmailRecord struct {
	EmailID   string    `json:"emailId"`
	Recipient string    `json:"recipient"`
	Subject   string    `json:"subject"`
	Body      string    `json:"body"`
	Status    string    `json:"status"` // "pending", "sent", "failed"
	SentAt    time.Time `json:"sentAt,omitempty"`
	MessageID string    `json:"messageId,omitempty"`
	Error     string    `json:"error,omitempty"`
}

type EmailResult struct {
	EmailID   string `json:"emailId"`
	Status    string `json:"status"`
	MessageID string `json:"messageId,omitempty"`
	Message   string `json:"message"`
}

// SendEmailViaSMTP simulates external email provider (like SendGrid, SES)
func SendEmailViaSMTP(recipient, subject, body string) (messageID string, err error) {
	// Simulate network delay
	time.Sleep(50 * time.Millisecond)

	// Simulate 5% failure rate
	if rand.Float64() < 0.05 {
		return "", fmt.Errorf("SMTP timeout")
	}

	return fmt.Sprintf("msg_%s_%d", recipient, time.Now().Unix()), nil
}

// SendEmail sends an email (IDEMPOTENT)
func (EmailService) SendEmail(
	ctx restate.ObjectContext,
	req EmailRequest,
) (EmailResult, error) {
	emailID := restate.Key(ctx)

	ctx.Log().Info("Sending email",
		"emailId", emailID,
		"recipient", req.Recipient,
		"subject", req.Subject)

	// Check if email already sent (state-based deduplication)
	existingRecord, err := restate.Get[*EmailRecord](ctx, "email")
	if err != nil {
		return EmailResult{}, err
	}

	if existingRecord != nil {
		ctx.Log().Info("Email already processed",
			"emailId", emailID,
			"status", existingRecord.Status)

		// Return existing result (idempotent!)
		return EmailResult{
			EmailID:   existingRecord.EmailID,
			Status:    existingRecord.Status,
			MessageID: existingRecord.MessageID,
			Message:   fmt.Sprintf("Email already %s", existingRecord.Status),
		}, nil
	}

	// Validate request
	if req.Recipient == "" {
		return EmailResult{}, restate.TerminalError(
			fmt.Errorf("recipient is required"), 400)
	}

	if req.Subject == "" {
		return EmailResult{}, restate.TerminalError(
			fmt.Errorf("subject is required"), 400)
	}

	// Create email record in pending state
	record := EmailRecord{
		EmailID:   emailID,
		Recipient: req.Recipient,
		Subject:   req.Subject,
		Body:      req.Body,
		Status:    "pending",
	}

	restate.Set(ctx, "email", record)

	// Send email via SMTP (IDEMPOTENT side effect)
	messageID, err := restate.Run(ctx, func(ctx restate.RunContext) (string, error) {
		ctx.Log().Info("Calling email provider", "emailId", emailID)
		return SendEmailViaSMTP(req.Recipient, req.Subject, req.Body)
	})

	if err != nil {
		ctx.Log().Warn("Email send failed",
			"emailId", emailID,
			"error", err.Error())

		// Update record to failed state
		record.Status = "failed"
		record.Error = err.Error()
		restate.Set(ctx, "email", record)

		return EmailResult{
			EmailID: emailID,
			Status:  "failed",
			Message: fmt.Sprintf("Failed to send email: %s", err.Error()),
		}, nil
	}

	// Get send timestamp
	sentAt, err := restate.Run(ctx, func(ctx restate.RunContext) (time.Time, error) {
		return time.Now(), nil
	})
	if err != nil {
		return EmailResult{}, err
	}

	// Email sent successfully!
	ctx.Log().Info("Email sent successfully",
		"emailId", emailID,
		"messageId", messageID)

	record.Status = "sent"
	record.MessageID = messageID
	record.SentAt = sentAt
	restate.Set(ctx, "email", record)

	return EmailResult{
		EmailID:   emailID,
		Status:    "sent",
		MessageID: messageID,
		Message:   "Email sent successfully",
	}, nil
}

// GetEmailStatus retrieves email status (read-only, naturally idempotent)
func (EmailService) GetEmailStatus(
	ctx restate.ObjectSharedContext,
	_ restate.Void,
) (EmailRecord, error) {
	emailID := restate.Key(ctx)

	record, err := restate.Get[EmailRecord](ctx, "email")
	if err != nil {
		return EmailRecord{}, err
	}

	ctx.Log().Info("Retrieved email status",
		"emailId", emailID,
		"status", record.Status)

	return record, nil
}

// ResendEmail safely resends email (IDEMPOTENT)
func (EmailService) ResendEmail(
	ctx restate.ObjectContext,
	_ restate.Void,
) (EmailResult, error) {
	emailID := restate.Key(ctx)

	ctx.Log().Info("Resending email", "emailId", emailID)

	// Get existing email record
	record, err := restate.Get[EmailRecord](ctx, "email")
	if err != nil {
		return EmailResult{}, err
	}

	// If already sent successfully, don't resend (idempotent)
	if record.Status == "sent" {
		ctx.Log().Info("Email already sent, not resending",
			"emailId", emailID,
			"messageId", record.MessageID)

		return EmailResult{
			EmailID:   record.EmailID,
			Status:    record.Status,
			MessageID: record.MessageID,
			Message:   "Email already sent successfully",
		}, nil
	}

	// If failed or pending, try sending again
	ctx.Log().Info("Retrying email send",
		"emailId", emailID,
		"previousStatus", record.Status)

	// Update status back to pending
	record.Status = "pending"
	record.Error = ""
	restate.Set(ctx, "email", record)

	// Attempt to send (journaled operation)
	messageID, err := restate.Run(ctx, func(ctx restate.RunContext) (string, error) {
		return SendEmailViaSMTP(record.Recipient, record.Subject, record.Body)
	})

	if err != nil {
		record.Status = "failed"
		record.Error = err.Error()
		restate.Set(ctx, "email", record)

		return EmailResult{
			EmailID: emailID,
			Status:  "failed",
			Message: fmt.Sprintf("Resend failed: %s", err.Error()),
		}, nil
	}

	// Update to sent
	sentAt, _ := restate.Run(ctx, func(ctx restate.RunContext) (time.Time, error) {
		return time.Now(), nil
	})

	record.Status = "sent"
	record.MessageID = messageID
	record.SentAt = sentAt
	restate.Set(ctx, "email", record)

	ctx.Log().Info("Email resent successfully",
		"emailId", emailID,
		"messageId", messageID)

	return EmailResult{
		EmailID:   emailID,
		Status:    "sent",
		MessageID: messageID,
		Message:   "Email resent successfully",
	}, nil
}
