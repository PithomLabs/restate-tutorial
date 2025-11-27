package main

import (
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
)

type OrderService struct{}
type OrderServiceSlow struct{}

type OrderRequest struct {
	UserID      string  `json:"userId"`
	ProductID   string  `json:"productId"`
	Quantity    int     `json:"quantity"`
	Amount      float64 `json:"amount"`
	Weight      float64 `json:"weight"`
	Destination string  `json:"destination"`
}

type OrderResult struct {
	OrderID          string `json:"orderId"`
	Status           string `json:"status"`
	ProcessingTimeMs int64  `json:"processingTimeMs"`
	Details          string `json:"details"`
}

// ============================================
// SLOW VERSION - Sequential Execution
// ============================================

func (OrderServiceSlow) ProcessOrder(
	ctx restate.Context,
	req OrderRequest,
) (OrderResult, error) {
	startTime := time.Now()

	ctx.Log().Info("Processing order sequentially", "userId", req.UserID)

	// Step 1: Check inventory - 80ms
	inv, err := restate.Service[InventoryResponse](
		ctx, "InventoryService", "CheckInventory",
	).Request(InventoryRequest{
		ProductID: req.ProductID,
		Quantity:  req.Quantity,
		Warehouse: "Main",
	})
	if err != nil || !inv.Available {
		return OrderResult{
			Status:  "failed",
			Details: "Inventory not available",
		}, nil
	}

	// Step 2: Authorize payment - 120ms
	payment, err := restate.Service[PaymentResponse](
		ctx, "PaymentService", "AuthorizePayment",
	).Request(PaymentRequest{
		Amount:   req.Amount,
		Currency: "USD",
	})
	if err != nil || !payment.Authorized {
		return OrderResult{
			Status:  "failed",
			Details: "Payment authorization failed",
		}, nil
	}

	// Step 3: Check fraud - 150ms
	fraud, err := restate.Service[FraudResponse](
		ctx, "FraudService", "CheckFraud",
	).Request(FraudRequest{
		UserID: req.UserID,
		Amount: req.Amount,
	})
	if err != nil {
		return OrderResult{}, err
	}
	if fraud.Flagged {
		return OrderResult{
			Status:  "failed",
			Details: "High fraud risk",
		}, nil
	}

	// Step 4: Calculate shipping - 100ms
	shipping, err := restate.Service[ShippingResponse](
		ctx, "ShippingService", "CalculateShipping",
	).Request(ShippingRequest{
		Weight:      req.Weight,
		Destination: req.Destination,
		Carrier:     "Standard",
	})
	if err != nil {
		return OrderResult{}, err
	}

	// Create order
	orderID := fmt.Sprintf("ORD_%s", restate.UUID(ctx).String()[:8])

	duration := time.Since(startTime)
	ctx.Log().Info("Order processed sequentially",
		"orderId", orderID,
		"durationMs", duration.Milliseconds())

	return OrderResult{
		OrderID:          orderID,
		Status:           "confirmed",
		ProcessingTimeMs: duration.Milliseconds(),
		Details:          fmt.Sprintf("Shipping: $%.2f via %s", shipping.Cost, shipping.Carrier),
	}, nil
}

// ============================================
// FAST VERSION - Parallel Execution
// ============================================

func (OrderService) ProcessOrder(
	ctx restate.Context,
	req OrderRequest,
) (OrderResult, error) {
	startTime := time.Now()

	ctx.Log().Info("Processing order in parallel", "userId", req.UserID)

	// ===================================
	// Fan-Out: Start all checks in parallel
	// ===================================

	// Check inventory in multiple warehouses (parallel)
	invFutMain := restate.Service[InventoryResponse](
		ctx, "InventoryService", "CheckInventory",
	).RequestFuture(InventoryRequest{
		ProductID: req.ProductID,
		Quantity:  req.Quantity,
		Warehouse: "Main",
	})

	invFutBackup := restate.Service[InventoryResponse](
		ctx, "InventoryService", "CheckInventory",
	).RequestFuture(InventoryRequest{
		ProductID: req.ProductID,
		Quantity:  req.Quantity,
		Warehouse: "Backup",
	})

	// Authorize payment (parallel)
	paymentFut := restate.Service[PaymentResponse](
		ctx, "PaymentService", "AuthorizePayment",
	).RequestFuture(PaymentRequest{
		Amount:   req.Amount,
		Currency: "USD",
	})

	// Check fraud (parallel)
	fraudFut := restate.Service[FraudResponse](
		ctx, "FraudService", "CheckFraud",
	).RequestFuture(FraudRequest{
		UserID: req.UserID,
		Amount: req.Amount,
	})

	// Calculate shipping from multiple carriers (parallel)
	shippingFutStandard := restate.Service[ShippingResponse](
		ctx, "ShippingService", "CalculateShipping",
	).RequestFuture(ShippingRequest{
		Weight:      req.Weight,
		Destination: req.Destination,
		Carrier:     "Standard",
	})

	shippingFutEconomy := restate.Service[ShippingResponse](
		ctx, "ShippingService", "CalculateShipping",
	).RequestFuture(ShippingRequest{
		Weight:      req.Weight,
		Destination: req.Destination,
		Carrier:     "Economy",
	})

	// ===================================
	// Fan-In: Collect and process results
	// ===================================

	var inventoryAvailable bool
	var paymentAuthorized bool
	var fraudRisk float64
	var shippingOptions []ShippingResponse

	// Wait for all futures to complete
	for fut, err := range restate.Wait(ctx,
		invFutMain, invFutBackup,
		paymentFut, fraudFut,
		shippingFutStandard, shippingFutEconomy) {

		if err != nil {
			ctx.Log().Warn("Future failed", "error", err)
			continue // Partial failure handling
		}

		// Process each result
		switch fut {
		case invFutMain:
			inv, _ := invFutMain.Response()
			if inv.Available {
				inventoryAvailable = true
				ctx.Log().Info("Inventory available", "warehouse", "Main")
			}

		case invFutBackup:
			inv, _ := invFutBackup.Response()
			if inv.Available && !inventoryAvailable {
				inventoryAvailable = true
				ctx.Log().Info("Inventory available", "warehouse", "Backup")
			}

		case paymentFut:
			payment, _ := paymentFut.Response()
			paymentAuthorized = payment.Authorized
			ctx.Log().Info("Payment check complete", "authorized", paymentAuthorized)

		case fraudFut:
			fraud, _ := fraudFut.Response()
			fraudRisk = fraud.RiskScore
			ctx.Log().Info("Fraud check complete", "riskScore", fraudRisk)

		case shippingFutStandard:
			shipping, _ := shippingFutStandard.Response()
			shippingOptions = append(shippingOptions, shipping)

		case shippingFutEconomy:
			shipping, _ := shippingFutEconomy.Response()
			shippingOptions = append(shippingOptions, shipping)
		}
	}

	// ===================================
	// Validation: Check all requirements
	// ===================================

	if !inventoryAvailable {
		return OrderResult{
			Status:  "failed",
			Details: "Product not available in any warehouse",
		}, nil
	}

	if !paymentAuthorized {
		return OrderResult{
			Status:  "failed",
			Details: "Payment authorization failed",
		}, nil
	}

	if fraudRisk > 70 {
		return OrderResult{
			Status:  "failed",
			Details: fmt.Sprintf("High fraud risk: %.1f", fraudRisk),
		}, nil
	}

	if len(shippingOptions) == 0 {
		return OrderResult{
			Status:  "failed",
			Details: "No shipping options available",
		}, nil
	}

	// Choose best shipping option (lowest cost)
	bestShipping := shippingOptions[0]
	for _, opt := range shippingOptions {
		if opt.Cost < bestShipping.Cost {
			bestShipping = opt
		}
	}

	// ===================================
	// Success: Create order
	// ===================================

	orderID := fmt.Sprintf("ORD_%s", restate.UUID(ctx).String()[:8])

	duration := time.Since(startTime)
	ctx.Log().Info("Order processed in parallel",
		"orderId", orderID,
		"durationMs", duration.Milliseconds())

	return OrderResult{
		OrderID:          orderID,
		Status:           "confirmed",
		ProcessingTimeMs: duration.Milliseconds(),
		Details: fmt.Sprintf(
			"Shipping: $%.2f via %s (%d days)",
			bestShipping.Cost,
			bestShipping.Carrier,
			bestShipping.EstimatedDays,
		),
	}, nil
}
