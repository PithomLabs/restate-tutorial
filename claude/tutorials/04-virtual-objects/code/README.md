# Module 04 - Shopping Cart with Virtual Objects

This directory contains a complete shopping cart implementation using Restate Virtual Objects.

## 📂 Files

- `main.go` - Server initialization
- `types.go` - Data structures
- `shopping_cart.go` - Virtual Object implementation
- `go.mod` - Dependencies

## 🚀 Quick Start

```bash
# Build
go mod tidy
go build -o cart-service

# Run
./cart-service
```

Service starts on port 9090.

## 📋 Register with Restate

```bash
curl -X POST http://localhost:8080/deployments \
  -H 'Content-Type: application/json' \
  -d '{"uri": "http://localhost:9090"}'
```

## 🧪 Test

### Add Items

```bash
curl -X POST http://localhost:9080/ShoppingCart/user123/AddItem \
  -H 'Content-Type: application/json' \
  -d '{
    "productId": "laptop-001",
    "productName": "Dell XPS 13",
    "quantity": 1,
    "price": 999.99
  }'
```

### View Cart

```bash
curl -X POST http://localhost:9080/ShoppingCart/user123/GetCart \
  -H 'Content-Type: application/json' \
  -d 'null'
```

### Apply Coupon

```bash
curl -X POST http://localhost:9080/ShoppingCart/user123/ApplyCoupon \
  -H 'Content-Type: application/json' \
  -d '"SAVE20"'
```

### Checkout

```bash
curl -X POST http://localhost:9080/ShoppingCart/user123/Checkout \
  -H 'Content-Type: application/json' \
  -d 'null'
```

## 🎓 Key Concepts Demonstrated

### Virtual Object Pattern

Each user (key) has isolated cart state:

```go
// User 123's cart
/ShoppingCart/user123/AddItem

// User 456's cart (completely separate)
/ShoppingCart/user456/AddItem
```

### Exclusive Handlers

Modify state - one at a time per key:

```go
func (ShoppingCart) AddItem(
    ctx restate.ObjectContext,  // ← Exclusive
    req AddItemRequest,
) error { ... }
```

### Concurrent Handlers

Read-only - multiple allowed per key:

```go
func (ShoppingCart) GetCart(
    ctx restate.ObjectSharedContext,  // ← Concurrent
    _ restate.Void,
) (CartSummary, error) { ... }
```

### State Management

```go
// Get state (scoped to current key)
cart, _ := restate.Get[Cart](ctx, "cart")

// Modify
cart.Items = append(cart.Items, newItem)

// Save (automatically scoped to current key)
restate.Set(ctx, "cart", cart)
```

### State Clearing

```go
// Clear all state for this cart
restate.ClearAll(ctx)
```

## 💡 Features

1. **Add/Remove Items** - Manage cart contents
2. **Update Quantities** - Change item amounts
3. **Coupon Support** - Apply discounts (SAVE10, SAVE20, SAVE50)
4. **Tax Calculation** - Automatic 8% tax
5. **Checkout** - Process order and clear cart
6. **State Isolation** - Each user has separate cart
7. **Durable State** - Survives failures

## 📊 Available Handlers

### Exclusive (Modify State)
- `AddItem` - Add item to cart
- `RemoveItem` - Remove item from cart
- `UpdateQuantity` - Change item quantity
- `ApplyCoupon` - Apply discount code
- `Checkout` - Complete purchase
- `ClearCart` - Empty cart

### Concurrent (Read-Only)
- `GetCart` - View cart summary
- `GetItemCount` - Get number of unique items

## 🎯 Next Steps

- Complete [validation tests](../03-validation.md)
- Try [exercises](../04-exercises.md)
- Build your own Virtual Objects

---

**Questions?** See the main [hands-on tutorial](../02-hands-on.md)!
