package main

import (
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
)

type ShoppingCart struct{}

const (
	stateKeyCart = "cart"
	taxRate      = 0.08 // 8% tax
)

// ============================================
// Exclusive Handlers (Modify State)
// ============================================

// AddItem adds an item to the cart
func (ShoppingCart) AddItem(
	ctx restate.ObjectContext,
	req AddItemRequest,
) error {
	userID := restate.Key(ctx)
	ctx.Log().Info("Adding item to cart", "userId", userID, "productId", req.ProductID)

	// Validate input
	if req.Quantity <= 0 {
		return restate.TerminalError(fmt.Errorf("quantity must be positive"), 400)
	}
	if req.Price < 0 {
		return restate.TerminalError(fmt.Errorf("price cannot be negative"), 400)
	}

	// Get current cart
	cart, err := restate.Get[Cart](ctx, stateKeyCart)
	if err != nil {
		return err
	}

	// Check if item already exists
	found := false
	for i, item := range cart.Items {
		if item.ProductID == req.ProductID {
			// Update quantity
			cart.Items[i].Quantity += req.Quantity
			found = true
			break
		}
	}

	// Add new item if not found
	if !found {
		cart.Items = append(cart.Items, CartItem{
			ProductID:   req.ProductID,
			ProductName: req.ProductName,
			Quantity:    req.Quantity,
			Price:       req.Price,
		})
	}

	// Update timestamp
	cart.LastUpdated = time.Now()

	// Save state
	restate.Set(ctx, stateKeyCart, cart)

	ctx.Log().Info("Item added successfully", "userId", userID, "itemCount", len(cart.Items))
	return nil
}

// RemoveItem removes an item from the cart
func (ShoppingCart) RemoveItem(
	ctx restate.ObjectContext,
	productID string,
) error {
	userID := restate.Key(ctx)
	ctx.Log().Info("Removing item from cart", "userId", userID, "productId", productID)

	// Get current cart
	cart, err := restate.Get[Cart](ctx, stateKeyCart)
	if err != nil {
		return err
	}

	// Find and remove item
	found := false
	for i, item := range cart.Items {
		if item.ProductID == productID {
			// Remove item by slicing
			cart.Items = append(cart.Items[:i], cart.Items[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return restate.TerminalError(fmt.Errorf("item not found in cart"), 404)
	}

	// Update timestamp
	cart.LastUpdated = time.Now()

	// Save state
	restate.Set(ctx, stateKeyCart, cart)

	ctx.Log().Info("Item removed successfully", "userId", userID)
	return nil
}

// UpdateQuantity updates the quantity of an item in the cart
func (ShoppingCart) UpdateQuantity(
	ctx restate.ObjectContext,
	req UpdateQuantityRequest,
) error {
	userID := restate.Key(ctx)
	ctx.Log().Info("Updating item quantity", "userId", userID, "productId", req.ProductID)

	if req.Quantity < 0 {
		return restate.TerminalError(fmt.Errorf("quantity cannot be negative"), 400)
	}

	// Get current cart
	cart, err := restate.Get[Cart](ctx, stateKeyCart)
	if err != nil {
		return err
	}

	// Find and update item
	found := false
	for i, item := range cart.Items {
		if item.ProductID == req.ProductID {
			if req.Quantity == 0 {
				// Remove item if quantity is 0
				cart.Items = append(cart.Items[:i], cart.Items[i+1:]...)
			} else {
				cart.Items[i].Quantity = req.Quantity
			}
			found = true
			break
		}
	}

	if !found {
		return restate.TerminalError(fmt.Errorf("item not found in cart"), 404)
	}

	// Update timestamp
	cart.LastUpdated = time.Now()

	// Save state
	restate.Set(ctx, stateKeyCart, cart)

	ctx.Log().Info("Quantity updated successfully", "userId", userID)
	return nil
}

// ApplyCoupon applies a discount coupon to the cart
func (ShoppingCart) ApplyCoupon(
	ctx restate.ObjectContext,
	couponCode string,
) error {
	userID := restate.Key(ctx)
	ctx.Log().Info("Applying coupon", "userId", userID, "coupon", couponCode)

	// Get current cart
	cart, err := restate.Get[Cart](ctx, stateKeyCart)
	if err != nil {
		return err
	}

	// Validate coupon (in real app, call external service)
	discount, err := validateCoupon(ctx, couponCode)
	if err != nil {
		return restate.TerminalError(err, 400)
	}

	// Apply coupon
	cart.CouponCode = couponCode
	cart.Discount = discount
	cart.LastUpdated = time.Now()

	// Save state
	restate.Set(ctx, stateKeyCart, cart)

	ctx.Log().Info("Coupon applied", "userId", userID, "discount", discount)
	return nil
}

// Checkout completes the purchase and clears the cart
func (ShoppingCart) Checkout(
	ctx restate.ObjectContext,
	_ restate.Void,
) (CheckoutResult, error) {
	userID := restate.Key(ctx)
	ctx.Log().Info("Processing checkout", "userId", userID)

	// Get current cart
	cart, err := restate.Get[Cart](ctx, stateKeyCart)
	if err != nil {
		return CheckoutResult{}, err
	}

	// Validate cart is not empty
	if len(cart.Items) == 0 {
		return CheckoutResult{}, restate.TerminalError(
			fmt.Errorf("cart is empty"),
			400,
		)
	}

	// Calculate totals
	summary := calculateCartSummary(cart)

	// Create order ID (durable UUID)
	orderID := fmt.Sprintf("ORD_%s", restate.UUID(ctx).String()[:8])

	// In real app: Call payment service, inventory service, etc.
	// For now, just simulate
	err = processPayment(ctx, userID, summary.Total)
	if err != nil {
		return CheckoutResult{}, err
	}

	// Create result
	result := CheckoutResult{
		OrderID:      orderID,
		Total:        summary.Total,
		ItemCount:    summary.ItemCount,
		CouponCode:   cart.CouponCode,
		CheckedOutAt: time.Now(),
	}

	// Clear cart after successful checkout
	restate.ClearAll(ctx)

	ctx.Log().Info("Checkout completed", "userId", userID, "orderId", orderID, "total", summary.Total)

	return result, nil
}

// ClearCart clears all items from the cart
func (ShoppingCart) ClearCart(
	ctx restate.ObjectContext,
	_ restate.Void,
) error {
	userID := restate.Key(ctx)
	ctx.Log().Info("Clearing cart", "userId", userID)

	restate.ClearAll(ctx)

	ctx.Log().Info("Cart cleared", "userId", userID)
	return nil
}

// ============================================
// Concurrent Handlers (Read-Only)
// ============================================

// GetCart returns the current cart summary (read-only)
func (ShoppingCart) GetCart(
	ctx restate.ObjectSharedContext,
	_ restate.Void,
) (CartSummary, error) {
	userID := restate.Key(ctx)

	// Get current cart
	cart, err := restate.Get[Cart](ctx, stateKeyCart)
	if err != nil {
		return CartSummary{}, err
	}

	// Calculate and return summary
	summary := calculateCartSummary(cart)

	return summary, nil
}

// GetItemCount returns the number of items (read-only)
func (ShoppingCart) GetItemCount(
	ctx restate.ObjectSharedContext,
	_ restate.Void,
) (int, error) {
	cart, err := restate.Get[Cart](ctx, stateKeyCart)
	if err != nil {
		return 0, err
	}

	return len(cart.Items), nil
}

// ============================================
// Helper Functions
// ============================================

func calculateCartSummary(cart Cart) CartSummary {
	var subtotal float64
	var itemCount int

	for _, item := range cart.Items {
		subtotal += item.Price * float64(item.Quantity)
		itemCount += item.Quantity
	}

	// Apply discount
	discountAmount := subtotal * (cart.Discount / 100.0)
	afterDiscount := subtotal - discountAmount

	// Calculate tax
	tax := afterDiscount * taxRate

	// Total
	total := afterDiscount + tax

	return CartSummary{
		Items:       cart.Items,
		Subtotal:    subtotal,
		Discount:    discountAmount,
		Tax:         tax,
		Total:       total,
		ItemCount:   itemCount,
		CouponCode:  cart.CouponCode,
		LastUpdated: cart.LastUpdated,
	}
}

func validateCoupon(ctx restate.ObjectContext, code string) (float64, error) {
	// Simulate coupon validation using restate.Run
	discount, err := restate.Run(ctx, func(ctx restate.RunContext) (float64, error) {
		// In real app: call external coupon service
		switch code {
		case "SAVE10":
			return 10.0, nil
		case "SAVE20":
			return 20.0, nil
		case "SAVE50":
			return 50.0, nil
		default:
			return 0, fmt.Errorf("invalid coupon code")
		}
	})

	return discount, err
}

func processPayment(ctx restate.ObjectContext, userID string, amount float64) error {
	// Simulate payment processing using restate.Run
	_, err := restate.Run(ctx, func(ctx restate.RunContext) (bool, error) {
		// In real app: call payment gateway
		ctx.Log().Info("Processing payment", "userId", userID, "amount", amount)

		// Simulate payment success
		return true, nil
	})

	return err
}
