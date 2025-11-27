package main

import (
	restate "github.com/restatedev/sdk-go"
)

type WebhookProcessor struct{}

// ProcessStripeWebhook handles Stripe webhooks (IDEMPOTENT)
func (WebhookProcessor) ProcessStripeWebhook(
	ctx restate.ObjectContext,
	webhook StripeWebhook,
) (WebhookResult, error) {
	webhookID := restate.Key(ctx)

	ctx.Log().Info("Processing Stripe webhook",
		"webhookId", webhookID,
		"type", webhook.Type)

	// Check if already processed
	existing, err := restate.Get[*WebhookResult](ctx, "result")
	if err != nil {
		return WebhookResult{}, err
	}

	if existing != nil {
		ctx.Log().Info("Webhook already processed")
		return *existing, nil
	}

	// Process based on type
	var result WebhookResult

	switch webhook.Type {
	case "charge.succeeded":
		result = WebhookResult{
			WebhookID: webhookID,
			Type:      webhook.Type,
			Status:    "processed",
			Message:   "Charge succeeded",
		}

	case "charge.failed":
		result = WebhookResult{
			WebhookID: webhookID,
			Type:      webhook.Type,
			Status:    "processed",
			Message:   "Charge failed",
		}

	default:
		ctx.Log().Warn("Unknown webhook type", "type", webhook.Type)
		result = WebhookResult{
			WebhookID: webhookID,
			Type:      webhook.Type,
			Status:    "skipped",
			Message:   "Unknown webhook type",
		}
	}

	// Store result
	restate.Set(ctx, "result", result)

	return result, nil
}
