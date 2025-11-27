package main

import (
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
)

type PaymentService struct{}

const (
	stateKeyPayment = "payment"
	stateKeyRefund  = "refund"
)

// CreatePayment creates a new payment (IDEMPOTENT)
func (PaymentService) CreatePayment(
	ctx restate.ObjectContext,
	req PaymentRequest,
) (PaymentResult, error) {
	paymentID := restate.Key(ctx)

	ctx.Log().Info("Creating payment",
		"paymentId", paymentID,
		"amount", req.Amount,
		"customerId", req.CustomerID)

	// Check if payment already exists (state-based deduplication)
	existingPayment, err := restate.Get[*Payment](ctx, stateKeyPayment)
	if err != nil {
		return PaymentResult{}, err
	}

	if existingPayment != nil {
		ctx.Log().Info("Payment already exists, returning cached result",
			"paymentId", paymentID,
			"status", existingPayment.Status)

		// Return existing payment (idempotent!)
		return PaymentResult{
			PaymentID: existingPayment.PaymentID,
			Status:    existingPayment.Status,
			ChargeID:  existingPayment.ChargeID,
			Message:   "Payment already processed",
		}, nil
	}

	// Validate request
	if req.Amount <= 0 {
		return PaymentResult{}, restate.TerminalError(
			fmt.Errorf("invalid amount: must be positive"), 400)
	}

	if req.Currency == "" {
		req.Currency = "USD"
	}

	// Create payment record in pending state
	payment := Payment{
		PaymentID:   paymentID,
		Amount:      req.Amount,
		Currency:    req.Currency,
		Description: req.Description,
		CustomerID:  req.CustomerID,
		Status:      "pending",
		CreatedAt:   time.Now(),
	}

	restate.Set(ctx, stateKeyPayment, payment)

	// Charge customer via payment gateway (IDEMPOTENT side effect)
	chargeResp, err := restate.Run(ctx, func(ctx restate.RunContext) (ChargeResponse, error) {
		ctx.Log().Info("Calling payment gateway", "paymentId", paymentID)

		gateway := &MockPaymentGateway{}
		resp := gateway.Charge(req.Amount, req.Currency, req.CustomerID)

		return resp, nil
	})

	if err != nil {
		// Update payment to failed state
		payment.Status = "failed"
		payment.ErrorMsg = err.Error()
		restate.Set(ctx, stateKeyPayment, payment)

		return PaymentResult{}, fmt.Errorf("gateway error: %w", err)
	}

	// Check charge result
	if !chargeResp.Success {
		ctx.Log().Warn("Payment failed",
			"paymentId", paymentID,
			"error", chargeResp.ErrorMsg)

		payment.Status = "failed"
		payment.ErrorMsg = chargeResp.ErrorMsg
		restate.Set(ctx, stateKeyPayment, payment)

		return PaymentResult{
			PaymentID: paymentID,
			Status:    "failed",
			Message:   chargeResp.ErrorMsg,
		}, nil
	}

	// Payment succeeded!
	ctx.Log().Info("Payment completed successfully",
		"paymentId", paymentID,
		"chargeId", chargeResp.ChargeID)

	payment.Status = "completed"
	payment.ChargeID = chargeResp.ChargeID
	payment.CompletedAt = time.Now()
	restate.Set(ctx, stateKeyPayment, payment)

	// Send receipt (IDEMPOTENT side effect)
	_, err = restate.Run(ctx, func(ctx restate.RunContext) (bool, error) {
		ctx.Log().Info("Sending payment receipt",
			"paymentId", paymentID,
			"customerId", req.CustomerID)

		// In production: send email via SendGrid, SES, etc.
		return true, nil
	})

	return PaymentResult{
		PaymentID: paymentID,
		Status:    "completed",
		ChargeID:  chargeResp.ChargeID,
		Message:   "Payment processed successfully",
	}, nil
}

// GetPayment retrieves payment status (read-only, naturally idempotent)
func (PaymentService) GetPayment(
	ctx restate.ObjectSharedContext,
	_ restate.Void,
) (Payment, error) {
	paymentID := restate.Key(ctx)

	payment, err := restate.Get[Payment](ctx, stateKeyPayment)
	if err != nil {
		return Payment{}, err
	}

	ctx.Log().Info("Retrieved payment", "paymentId", paymentID, "status", payment.Status)

	return payment, nil
}

// RefundPayment refunds a completed payment (IDEMPOTENT)
func (PaymentService) RefundPayment(
	ctx restate.ObjectContext,
	req RefundRequest,
) (RefundResult, error) {
	paymentID := restate.Key(ctx)

	ctx.Log().Info("Processing refund",
		"paymentId", paymentID,
		"reason", req.Reason)

	// Check if refund already exists
	existingRefund, err := restate.Get[*RefundResult](ctx, stateKeyRefund)
	if err != nil {
		return RefundResult{}, err
	}

	if existingRefund != nil {
		ctx.Log().Info("Refund already processed", "refundId", existingRefund.RefundID)
		return *existingRefund, nil
	}

	// Get payment
	payment, err := restate.Get[Payment](ctx, stateKeyPayment)
	if err != nil {
		return RefundResult{}, err
	}

	// Validate payment can be refunded
	if payment.Status != "completed" {
		return RefundResult{}, restate.TerminalError(
			fmt.Errorf("cannot refund payment with status: %s", payment.Status), 400)
	}

	// Determine refund amount
	refundAmount := req.Amount
	if refundAmount == 0 || refundAmount > payment.Amount {
		refundAmount = payment.Amount // Full refund
	}

	// Process refund via gateway (IDEMPOTENT side effect)
	refundResp, err := restate.Run(ctx, func(ctx restate.RunContext) (ChargeResponse, error) {
		ctx.Log().Info("Calling gateway for refund", "paymentId", paymentID)

		gateway := &MockPaymentGateway{}
		resp := gateway.Refund(payment.ChargeID, refundAmount)

		return resp, nil
	})

	if err != nil {
		return RefundResult{}, fmt.Errorf("refund failed: %w", err)
	}

	// Create refund result
	result := RefundResult{
		RefundID: refundResp.ChargeID,
		Status:   "completed",
		Amount:   refundAmount,
		Message:  "Refund processed successfully",
	}

	// Store refund (prevents duplicate refunds)
	restate.Set(ctx, stateKeyRefund, result)

	// Update payment status
	payment.Status = "refunded"
	restate.Set(ctx, stateKeyPayment, payment)

	ctx.Log().Info("Refund completed", "refundId", result.RefundID)

	return result, nil
}
