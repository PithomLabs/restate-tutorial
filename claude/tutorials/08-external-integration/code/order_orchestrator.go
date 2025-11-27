package main

import (
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
)

type OrderOrchestrator struct{}

var (
	stripeClient   = NewStripeClient()
	sendgridClient = NewSendGridClient()
	shippoClient   = NewShippoClient()
)

// ProcessOrder orchestrates the entire order workflow
func (OrderOrchestrator) ProcessOrder(
	ctx restate.ObjectContext,
	req OrderRequest,
) (OrderResult, error) {
	orderID := restate.Key(ctx)

	ctx.Log().Info("Processing order",
		"orderId", orderID,
		"customer", req.Customer.Email)

	// Check if order already exists (idempotent)
	existingOrder, err := restate.Get[*Order](ctx, "order")
	if err != nil {
		return OrderResult{}, err
	}

	if existingOrder != nil {
		ctx.Log().Info("Order already exists", "status", existingOrder.Status)
		return OrderResult{
			OrderID:        existingOrder.OrderID,
			Status:         existingOrder.Status,
			ChargeID:       existingOrder.ChargeID,
			TrackingNumber: existingOrder.TrackingNumber,
			Message:        "Order already processed",
		}, nil
	}

	// Calculate total
	total := 0
	for _, item := range req.Items {
		total += item.Price * item.Quantity
	}

	// Create order in pending state
	order := Order{
		OrderID:   orderID,
		Items:     req.Items,
		Customer:  req.Customer,
		Shipping:  req.Shipping,
		Status:    "pending",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	restate.Set(ctx, "order", order)

	// Step 1: Charge customer (JOURNALED)
	ctx.Log().Info("Charging customer", "amount", total)

	chargeResp, err := restate.Run(ctx, func(ctx restate.RunContext) (StripeChargeResponse, error) {
		return stripeClient.CreateCharge(ctx, StripeChargeRequest{
			Amount:   total,
			Currency: "usd",
			Email:    req.Customer.Email,
		})
	})

	if err != nil {
		ctx.Log().Error("Payment failed", "error", err)
		order.Status = "payment_failed"
		restate.Set(ctx, "order", order)

		return OrderResult{
			OrderID: orderID,
			Status:  "payment_failed",
			Message: fmt.Sprintf("Payment failed: %s", err.Error()),
		}, nil
	}

	order.ChargeID = chargeResp.ID
	order.Status = "paid"
	order.UpdatedAt = time.Now()
	restate.Set(ctx, "order", order)

	ctx.Log().Info("Payment successful", "chargeId", chargeResp.ID)

	// Step 2: Create shipping label (JOURNALED)
	ctx.Log().Info("Creating shipping label")

	labelResp, err := restate.Run(ctx, func(ctx restate.RunContext) (ShippingLabelResponse, error) {
		return shippoClient.CreateLabel(ctx, ShippingLabelRequest{
			Address: req.Shipping,
			Weight:  1000, // 1kg default
		})
	})

	if err != nil {
		// Label creation failed, but order still valid
		ctx.Log().Warn("Failed to create label", "error", err)
	} else {
		order.LabelID = labelResp.LabelID
		order.TrackingNumber = labelResp.TrackingNumber
		order.UpdatedAt = time.Now()
		restate.Set(ctx, "order", order)

		ctx.Log().Info("Shipping label created",
			"labelId", labelResp.LabelID,
			"tracking", labelResp.TrackingNumber)
	}

	// Step 3: Send confirmation email (JOURNALED)
	ctx.Log().Info("Sending confirmation email")

	emailBody := fmt.Sprintf(`
		Hi %s,
		
		Thank you for your order!
		
		Order ID: %s
		Total: $%.2f
		Tracking: %s
		
		Your order will ship soon!
	`, req.Customer.Name, orderID, float64(total)/100, order.TrackingNumber)

	_, err = restate.Run(ctx, func(ctx restate.RunContext) (EmailResponse, error) {
		return sendgridClient.SendEmail(ctx, EmailRequest{
			To:      req.Customer.Email,
			Subject: fmt.Sprintf("Order Confirmation - %s", orderID),
			Body:    emailBody,
		})
	})

	if err != nil {
		ctx.Log().Warn("Failed to send email", "error", err)
	} else {
		ctx.Log().Info("Confirmation email sent")
	}

	// Mark order as confirmed
	order.Status = "confirmed"
	order.UpdatedAt = time.Now()
	restate.Set(ctx, "order", order)

	return OrderResult{
		OrderID:        orderID,
		Status:         "confirmed",
		ChargeID:       order.ChargeID,
		TrackingNumber: order.TrackingNumber,
		Message:        "Order processed successfully",
	}, nil
}

// GetOrder retrieves order status
func (OrderOrchestrator) GetOrder(
	ctx restate.ObjectSharedContext,
	_ restate.Void,
) (Order, error) {
	orderID := restate.Key(ctx)

	order, err := restate.Get[Order](ctx, "order")
	if err != nil {
		return Order{}, err
	}

	ctx.Log().Info("Retrieved order", "orderId", orderID, "status", order.Status)

	return order, nil
}
