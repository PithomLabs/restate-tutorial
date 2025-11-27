# Hands-On: Building a Shopping Cart Service

> **Build a stateful shopping cart using Virtual Objects**

## 🎯 What We're Building

A **Shopping Cart Service** with:
- Add/remove items to cart per user
- Update item quantities
- Apply discount coupons
- Calculate totals with tax
- Checkout and clear cart
- View cart history

**Key Feature:** Each user's cart is completely isolated with durable state.

## 📋 Prerequisites

- ✅ Completed [Module 03](../03-concurrency/README.md)
- ✅ Understanding of Virtual Objects from [concepts](./01-concepts.md)
- ✅ Restate server running

## 🚀 Step-by-Step Tutorial

### Step 1: Project Setup

```bash
# Create project directory
mkdir -p ~/restate-tutorials/module04
cd ~/restate-tutorials/module04

# Initialize Go module
go mod init module04

# Install dependencies
go get github.com/restatedev/sdk-go
```

### Step 2: Define Data Structures

Create `types.go`:

```go
package main

import "time"

// CartItem represents a product in the cart
type CartItem struct {
	ProductID   string  `json:"productId"`
	ProductName string  `json:"productName"`
	Quantity    int     `json:"quantity"`
	Price       float64 `json:"price"`
}

// Cart represents the complete cart state
type Cart struct {
	Items       []CartItem `json:"items"`
	CouponCode  string     `json:"couponCode"`
	Discount    float64    `json:"discount"` // Percentage (0-100)
	LastUpdated time.Time  `json:"lastUpdated"`
}

// AddItemRequest for adding items
type AddItemRequest struct {
	ProductID   string  `json:"productId"`
	ProductName string  `json:"productName"`
	Quantity    int     `json:"quantity"`
	Price       float64 `json:"price"`
}

// UpdateQuantityRequest for updating item quantities
type UpdateQuantityRequest struct {
	ProductID string `json:"productId"`
	Quantity  int    `json:"quantity"`
}

// CartSummary for viewing cart
type CartSummary struct {
	Items        []CartItem `json:"items"`
	Subtotal     float64    `json:"subtotal"`
	Discount     float64    `json:"discount"`
	Tax          float64    `json:"tax"`
	Total        float64    `json:"total"`
	ItemCount    int        `json:"itemCount"`
	CouponCode   string     `json:"couponCode,omitempty"`
	LastUpdated  time.Time  `json:"lastUpdated"`
}

// CheckoutResult returned after checkout
type CheckoutResult struct {
	OrderID     string      `json:"orderId"`
	Total       float64     `json:"total"`
	ItemCount   int         `json:"itemCount"`
	CouponCode  string      `json:"couponCode,omitempty"`
	CheckedOutAt time.Time  `json:"checkedOutAt"`
}
```

### Step 3: Implement Shopping Cart Virtual Object

Create `shopping_cart.go`:

```go
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
```

### Step 4: Create Main Entry Point

Create `main.go`:

```go
package main

import (
	"context"
	"fmt"
	"log"

	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/server"
)

func main() {
	restateServer := server.NewRestate()

	// Register Shopping Cart Virtual Object
	if err := restateServer.Bind(restate.Reflect(ShoppingCart{})); err != nil {
		log.Fatal("Failed to bind ShoppingCart:", err)
	}

	fmt.Println("🛒 Starting Shopping Cart Service on :9090...")
	fmt.Println("📝 Virtual Object: ShoppingCart")
	fmt.Println("")
	fmt.Println("Handlers:")
	fmt.Println("  Exclusive (modify state):")
	fmt.Println("    - AddItem")
	fmt.Println("    - RemoveItem")
	fmt.Println("    - UpdateQuantity")
	fmt.Println("    - ApplyCoupon")
	fmt.Println("    - Checkout")
	fmt.Println("    - ClearCart")
	fmt.Println("")
	fmt.Println("  Concurrent (read-only):")
	fmt.Println("    - GetCart")
	fmt.Println("    - GetItemCount")
	fmt.Println("")
	fmt.Println("✓ Ready to accept requests")

	if err := restateServer.Start(context.Background(), ":9090"); err != nil {
		log.Fatal("Server error:", err)
	}
}
```

### Step 5: Build and Run

```bash
# Build
go mod tidy
go build -o cart-service

# Run
./cart-service
```

### Step 6: Register with Restate

```bash
curl -X POST http://localhost:8080/deployments \
  -H 'Content-Type: application/json' \
  -d '{"uri": "http://localhost:9090"}'
```

### Step 7: Test the Shopping Cart

#### Add Items to Cart

```bash
# User 123 adds a laptop
curl -X POST http://localhost:9080/ShoppingCart/user123/AddItem \
  -H 'Content-Type: application/json' \
  -d '{
    "productId": "laptop-001",
    "productName": "Dell XPS 13",
    "quantity": 1,
    "price": 999.99
  }'

# User 123 adds a mouse
curl -X POST http://localhost:9080/ShoppingCart/user123/AddItem \
  -H 'Content-Type: application/json' \
  -d '{
    "productId": "mouse-001",
    "productName": "Logitech MX Master",
    "quantity": 2,
    "price": 79.99
  }'
```

**Notice the URL pattern:**
```
http://localhost:9080/ShoppingCart/{key}/{handler}
                        ^              ^     ^
                        |              |     |
                   Service Name      Key  Handler
```

#### View Cart

```bash
curl -X POST http://localhost:9080/ShoppingCart/user123/GetCart \
  -H 'Content-Type: application/json' \
  -d 'null'
```

**Expected Response:**
```json
{
  "items": [
    {
      "productId": "laptop-001",
      "productName": "Dell XPS 13",
      "quantity": 1,
      "price": 999.99
    },
    {
      "productId": "mouse-001",
      "productName": "Logitech MX Master",
      "quantity": 2,
      "price": 79.99
    }
  ],
  "subtotal": 1159.97,
  "discount": 0,
  "tax": 92.80,
  "total": 1252.77,
  "itemCount": 3
}
```

#### Update Quantity

```bash
# Change mouse quantity to 1
curl -X POST http://localhost:9080/ShoppingCart/user123/UpdateQuantity \
  -H 'Content-Type: application/json' \
  -d '{
    "productId": "mouse-001",
    "quantity": 1
  }'

# View updated cart
curl -X POST http://localhost:9080/ShoppingCart/user123/GetCart \
  -H 'Content-Type: application/json' \
  -d 'null'
```

#### Apply Coupon

```bash
curl -X POST http://localhost:9080/ShoppingCart/user123/ApplyCoupon \
  -H 'Content-Type: application/json' \
  -d '"SAVE20"'

# View cart with discount
curl -X POST http://localhost:9080/ShoppingCart/user123/GetCart \
  -H 'Content-Type: application/json' \
  -d 'null'
```

**Notice:**
- Subtotal: $1079.98
- Discount (20%): $216.00
- After discount: $863.98
- Tax (8%): $69.12
- **Total: $933.10**

#### Checkout

```bash
curl -X POST http://localhost:9080/ShoppingCart/user123/Checkout \
  -H 'Content-Type: application/json' \
  -d 'null'
```

**Expected Response:**
```json
{
  "orderId": "ORD_abc12345",
  "total": 933.10,
  "itemCount": 2,
  "couponCode": "SAVE20",
  "checkedOutAt": "2024-01-15T10:30:00Z"
}
```

#### Verify Cart is Cleared

```bash
curl -X POST http://localhost:9080/ShoppingCart/user123/GetCart \
  -H 'Content-Type: application/json' \
  -d 'null'
```

**Expected Response:**
```json
{
  "items": [],
  "subtotal": 0,
  "discount": 0,
  "tax": 0,
  "total": 0,
  "itemCount": 0
}
```

### Step 8: Test State Isolation

```bash
# User 456 has completely separate cart
curl -X POST http://localhost:9080/ShoppingCart/user456/AddItem \
  -H 'Content-Type: application/json' \
  -d '{
    "productId": "keyboard-001",
    "productName": "Mechanical Keyboard",
    "quantity": 1,
    "price": 149.99
  }'

# View user456's cart
curl -X POST http://localhost:9080/ShoppingCart/user456/GetCart \
  -H 'Content-Type: application/json' \
  -d 'null'

# View user123's cart (still empty from checkout)
curl -X POST http://localhost:9080/ShoppingCart/user123/GetCart \
  -H 'Content-Type: application/json' \
  -d 'null'
```

**Key Observation:** Each user (key) has completely isolated state!

## 🎓 Understanding the Implementation

### State Management

```go
// Get state for THIS key
cart, _ := restate.Get[Cart](ctx, stateKeyCart)

// Modify
cart.Items = append(cart.Items, newItem)

// Save (automatically scoped to current key)
restate.Set(ctx, stateKeyCart, cart)
```

### Exclusive vs Concurrent

```go
// Exclusive - can modify state
func (ShoppingCart) AddItem(
    ctx restate.ObjectContext,  // ← ObjectContext
    req AddItemRequest,
) error { ... }

// Concurrent - read-only
func (ShoppingCart) GetCart(
    ctx restate.ObjectSharedContext,  // ← ObjectSharedContext
    _ restate.Void,
) (CartSummary, error) { ... }
```

### Key-Based Addressing

```
URL: /ShoppingCart/{key}/{handler}

user123/AddItem  → ShoppingCart with key="user123"
user456/AddItem  → ShoppingCart with key="user456" (different state)
```

### Side Effects with restate.Run

```go
// Wrap external calls in restate.Run
discount, err := restate.Run(ctx, func(ctx restate.RunContext) (float64, error) {
    return callCouponService(code), nil
})
```

## ✅ Verification Checklist

- [ ] Service starts successfully
- [ ] Can add items to cart
- [ ] Can view cart
- [ ] Can update quantities
- [ ] Can apply coupons
- [ ] Can checkout (creates order, clears cart)
- [ ] Multiple users have isolated state
- [ ] Cart persists across service restarts

## 💡 Key Takeaways

1. **State is Per-Key**
   - Each user has completely isolated cart
   - No interference between users

2. **Exclusive Execution**
   - Only one AddItem/RemoveItem at a time per user
   - Prevents race conditions

3. **Concurrent Reads**
   - Multiple GetCart calls can run simultaneously
   - Better performance for read-heavy operations

4. **Durable State**
   - Cart survives failures
   - No external database needed for cart state

5. **Clear After Use**
   - Checkout clears cart with `ClearAll()`
   - Frees up memory

## 🎯 Next Steps

Ready to validate your stateful service!

👉 **Continue to [Validation](./03-validation.md)**

---

**Questions?** Review [concepts](./01-concepts.md) or check the [complete code](./code/)!
