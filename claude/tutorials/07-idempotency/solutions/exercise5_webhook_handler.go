package main

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
)

type WebhookService struct{}

type StripeWebhook struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"` // "payment.succeeded", "payment.failed", etc.
	Data      map[string]interface{} `json:"data"`
	Timestamp int64                  `json:"timestamp"`
}

type WebhookResult struct {
	WebhookID   string `json:"webhookId"`
	Type        string `json:"type"`
	ProcessedAt int64  `json:"processedAt"`
	Status      string `json:"status"` // "processed", "skipped"
	Message     string `json:"message"`
}

// ProcessWebhook processes a webhook (IDEMPOTENT)
func (WebhookService) ProcessWebhook(
	ctx restate.ObjectContext,
	webhook StripeWebhook,
) (WebhookResult, error) {
	webhookID := restate.Key(ctx)

	ctx.Log().Info("Processing webhook",
		"webhookId", webhookID,
		"type", webhook.Type,
		"timestamp", webhook.Timestamp)

	// Check if webhook already processed (state-based deduplication)
	existingResult, err := restate.Get[*WebhookResult](ctx, "result")
	if err != nil {
		return WebhookResult{}, err
	}

	if existingResult != nil {
		ctx.Log().Info("Webhook already processed",
			"webhookId", webhookID,
			"processedAt", existingResult.ProcessedAt)

		// Return immediately (idempotent!)
		return *existingResult, nil
	}

	// Validate webhook
	if webhook.ID == "" {
		return WebhookResult{}, restate.TerminalError(
			fmt.Errorf("webhook ID is required"), 400)
	}

	// Process based on webhook type
	var message string

	switch webhook.Type {
	case "payment.succeeded":
		message = processPaymentSucceeded(ctx, webhook)
	case "payment.failed":
		message = processPaymentFailed(ctx, webhook)
	case "payment.refunded":
		message = processPaymentRefunded(ctx, webhook)
	default:
		ctx.Log().Warn("Unknown webhook type", "type", webhook.Type)
		message = fmt.Sprintf("Skipped unknown webhook type: %s", webhook.Type)
	}

	// Get processing timestamp
	processedAt, err := restate.Run(ctx, func(ctx restate.RunContext) (int64, error) {
		return webhook.Timestamp, nil
	})
	if err != nil {
		return WebhookResult{}, err
	}

	// Create and store result
	result := WebhookResult{
		WebhookID:   webhookID,
		Type:        webhook.Type,
		ProcessedAt: processedAt,
		Status:      "processed",
		Message:     message,
	}

	restate.Set(ctx, "result", result)

	ctx.Log().Info("Webhook processed successfully",
		"webhookId", webhookID,
		"type", webhook.Type)

	return result, nil
}

// GetWebhookStatus retrieves webhook processing status (read-only)
func (WebhookService) GetWebhookStatus(
	ctx restate.ObjectSharedContext,
	_ restate.Void,
) (WebhookResult, error) {
	webhookID := restate.Key(ctx)

	result, err := restate.Get[WebhookResult](ctx, "result")
	if err != nil {
		return WebhookResult{}, err
	}

	ctx.Log().Info("Retrieved webhook status",
		"webhookId", webhookID,
		"status", result.Status)

	return result, nil
}

// processPaymentSucceeded handles successful payment webhooks
func processPaymentSucceeded(ctx restate.ObjectContext, webhook StripeWebhook) string {
	// Extract payment details from webhook data
	paymentID, _ := webhook.Data["paymentId"].(string)
	amount, _ := webhook.Data["amount"].(float64)

	ctx.Log().Info("Processing payment success",
		"paymentId", paymentID,
		"amount", amount)

	// Update order status (idempotent call)
	_, err := restate.Run(ctx, func(ctx restate.RunContext) (bool, error) {
		ctx.Log().Info("Updating order status", "paymentId", paymentID)
		// In production: call OrderService.MarkAsPaid
		return true, nil
	})

	if err != nil {
		return fmt.Sprintf("Failed to update order: %s", err.Error())
	}

	// Send confirmation email (idempotent call)
	_, err = restate.Run(ctx, func(ctx restate.RunContext) (bool, error) {
		ctx.Log().Info("Sending confirmation email", "paymentId", paymentID)
		// In production: call EmailService.SendEmail
		return true, nil
	})

	if err != nil {
		return fmt.Sprintf("Failed to send email: %s", err.Error())
	}

	return fmt.Sprintf("Payment %s processed successfully", paymentID)
}

// processPaymentFailed handles failed payment webhooks
func processPaymentFailed(ctx restate.ObjectContext, webhook StripeWebhook) string {
	paymentID, _ := webhook.Data["paymentId"].(string)
	reason, _ := webhook.Data["reason"].(string)

	ctx.Log().Info("Processing payment failure",
		"paymentId", paymentID,
		"reason", reason)

	// Update order status (idempotent call)
	_, err := restate.Run(ctx, func(ctx restate.RunContext) (bool, error) {
		ctx.Log().Info("Marking order as failed", "paymentId", paymentID)
		// In production: call OrderService.MarkAsFailed
		return true, nil
	})

	if err != nil {
		return fmt.Sprintf("Failed to update order: %s", err.Error())
	}

	// Send failure notification (idempotent call)
	_, err = restate.Run(ctx, func(ctx restate.RunContext) (bool, error) {
		ctx.Log().Info("Sending failure notification", "paymentId", paymentID)
		// In production: call NotificationService.SendFailureEmail
		return true, nil
	})

	return fmt.Sprintf("Payment %s failure processed: %s", paymentID, reason)
}

// processPaymentRefunded handles refund webhooks
func processPaymentRefunded(ctx restate.ObjectContext, webhook StripeWebhook) string {
	paymentID, _ := webhook.Data["paymentId"].(string)
	refundAmount, _ := webhook.Data["amount"].(float64)

	ctx.Log().Info("Processing refund",
		"paymentId", paymentID,
		"amount", refundAmount)

	// Update order status (idempotent call)
	_, err := restate.Run(ctx, func(ctx restate.RunContext) (bool, error) {
		ctx.Log().Info("Marking order as refunded", "paymentId", paymentID)
		// In production: call OrderService.MarkAsRefunded
		return true, nil
	})

	if err != nil {
		return fmt.Sprintf("Failed to update order: %s", err.Error())
	}

	return fmt.Sprintf("Refund %s processed successfully", paymentID)
}
