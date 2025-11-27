package main

import (
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
)

type OrderService struct{}

type OrderRequest struct {
	Items      []OrderItem `json:"items"`
	CustomerID string      `json:"customerId"`
	Total      int         `json:"total"`
}

type OrderItem struct {
	ProductID string `json:"productId"`
	Quantity  int    `json:"quantity"`
	Price     int    `json:"price"`
}

type Order struct {
	OrderID     string      `json:"orderId"`
	OrderNumber string      `json:"orderNumber"`
	Items       []OrderItem `json:"items"`
	CustomerID  string      `json:"customerId"`
	Total       int         `json:"total"`
	Status      string      `json:"status"` // "pending", "confirmed", "cancelled"
	CreatedAt   time.Time   `json:"createdAt"`
}

// CreateOrder creates a new order (IDEMPOTENT)
func (OrderService) CreateOrder(
	ctx restate.ObjectContext,
	req OrderRequest,
) (Order, error) {
	orderID := restate.Key(ctx)

	ctx.Log().Info("Creating order",
		"orderId", orderID,
		"customerId", req.CustomerID,
		"total", req.Total)

	// Check if order already exists (state-based deduplication)
	existingOrder, err := restate.Get[*Order](ctx, "order")
	if err != nil {
		return Order{}, err
	}

	if existingOrder != nil {
		ctx.Log().Info("Order already exists",
			"orderId", orderID,
			"status", existingOrder.Status)
		return *existingOrder, nil
	}

	// Validate request
	if len(req.Items) == 0 {
		return Order{}, restate.TerminalError(
			fmt.Errorf("order must have at least one item"), 400)
	}

	if req.Total <= 0 {
		return Order{}, restate.TerminalError(
			fmt.Errorf("order total must be positive"), 400)
	}

	// Generate deterministic order number
	// Uses Restate's deterministic random for consistent results on retries
	orderNumber := fmt.Sprintf("ORD-%010d", restate.Rand(ctx).Uint64())

	// Get current time for creation timestamp
	createdAt, err := restate.Run(ctx, func(ctx restate.RunContext) (time.Time, error) {
		return time.Now(), nil
	})
	if err != nil {
		return Order{}, err
	}

	// Create order
	order := Order{
		OrderID:     orderID,
		OrderNumber: orderNumber,
		Items:       req.Items,
		CustomerID:  req.CustomerID,
		Total:       req.Total,
		Status:      "pending",
		CreatedAt:   createdAt,
	}

	restate.Set(ctx, "order", order)

	ctx.Log().Info("Order created successfully",
		"orderId", orderID,
		"orderNumber", orderNumber)

	return order, nil
}

// GetOrder retrieves order details (read-only, naturally idempotent)
func (OrderService) GetOrder(
	ctx restate.ObjectSharedContext,
	_ restate.Void,
) (Order, error) {
	orderID := restate.Key(ctx)

	order, err := restate.Get[Order](ctx, "order")
	if err != nil {
		return Order{}, err
	}

	ctx.Log().Info("Retrieved order", "orderId", orderID)

	return order, nil
}

// ConfirmOrder confirms a pending order (IDEMPOTENT)
func (OrderService) ConfirmOrder(
	ctx restate.ObjectContext,
	_ restate.Void,
) (Order, error) {
	orderID := restate.Key(ctx)

	ctx.Log().Info("Confirming order", "orderId", orderID)

	// Get order
	order, err := restate.Get[Order](ctx, "order")
	if err != nil {
		return Order{}, err
	}

	// Idempotent: if already confirmed, return it
	if order.Status == "confirmed" {
		ctx.Log().Info("Order already confirmed", "orderId", orderID)
		return order, nil
	}

	// Validate state transition
	if order.Status != "pending" {
		return Order{}, restate.TerminalError(
			fmt.Errorf("cannot confirm order with status: %s", order.Status), 400)
	}

	// Update status
	order.Status = "confirmed"
	restate.Set(ctx, "order", order)

	// Send confirmation notification (idempotent side effect)
	_, err = restate.Run(ctx, func(ctx restate.RunContext) (bool, error) {
		ctx.Log().Info("Sending order confirmation",
			"orderId", orderID,
			"customerId", order.CustomerID)
		// In production: send email/SMS
		return true, nil
	})

	ctx.Log().Info("Order confirmed", "orderId", orderID)

	return order, nil
}

// CancelOrder cancels an order (IDEMPOTENT)
func (OrderService) CancelOrder(
	ctx restate.ObjectContext,
	_ restate.Void,
) (Order, error) {
	orderID := restate.Key(ctx)

	ctx.Log().Info("Cancelling order", "orderId", orderID)

	// Get order
	order, err := restate.Get[Order](ctx, "order")
	if err != nil {
		return Order{}, err
	}

	// Idempotent: if already cancelled, return it
	if order.Status == "cancelled" {
		ctx.Log().Info("Order already cancelled", "orderId", orderID)
		return order, nil
	}

	// Validate can cancel
	if order.Status != "pending" && order.Status != "confirmed" {
		return Order{}, restate.TerminalError(
			fmt.Errorf("cannot cancel order with status: %s", order.Status), 400)
	}

	// Update status
	order.Status = "cancelled"
	restate.Set(ctx, "order", order)

	// Release inventory (idempotent call to external service)
	_, err = restate.Run(ctx, func(ctx restate.RunContext) (bool, error) {
		ctx.Log().Info("Releasing inventory for cancelled order",
			"orderId", orderID)
		// In production: call InventoryService.ReleaseReservation
		return true, nil
	})

	ctx.Log().Info("Order cancelled", "orderId", orderID)

	return order, nil
}
