# Concepts: Virtual Objects and Stateful Services

> **Understanding key-addressable, stateful services with durable state**

## 🎯 What are Virtual Objects?

### Definition

A **Virtual Object** is a stateful service where:
- Each instance has a **unique key** (e.g., user ID, cart ID)
- State is **durable** and survives failures
- Handlers are **exclusive** by default (one at a time per key)
- Multiple keys can process **in parallel**

Think of Virtual Objects as **actors** or **entities** identified by a key.

### Real-World Analogy

Imagine a bank with many teller windows:

**Basic Service** = Information desk
- Stateless
- Any clerk can help any customer
- No memory between requests

**Virtual Object** = Personal banker
- Stateful (knows your account)
- Exclusive access (one transaction at a time per account)
- Different accounts can transact simultaneously

## 🆚 Services vs Virtual Objects

### Basic Service (Stateless)

```go
type WeatherService struct{}

// Every request is independent
func (WeatherService) GetWeather(
    ctx restate.Context,
    city string,
) (Weather, error) {
    // No state
    return fetchWeather(city), nil
}
```

**Characteristics:**
- No state between calls
- Fully parallel execution
- Context: `restate.Context`

### Virtual Object (Stateful)

```go
type ShoppingCart struct{}

// State is per cart (key)
func (ShoppingCart) AddItem(
    ctx restate.ObjectContext,  // ← ObjectContext!
    item CartItem,
) error {
    // Get state for THIS key
    items, _ := restate.Get[[]CartItem](ctx, "items")
    
    // Update state
    items = append(items, item)
    restate.Set(ctx, "items", items)
    
    return nil
}
```

**Characteristics:**
- Durable state per key
- Exclusive execution per key
- Context: `restate.ObjectContext`

## 🔑 Key-Based Addressing

### How Keys Work

```
Service: ShoppingCart
Keys:    user123, user456, user789
State:   Separate for each key

┌─────────────────┐
│ ShoppingCart    │
├─────────────────┤
│ Key: user123    │ ← [Item1, Item2]
│ Key: user456    │ ← [Item3]
│ Key: user789    │ ← []
└─────────────────┘
```

### Calling Virtual Objects

```go
// Call AddItem for user123's cart
err := restate.Object[error](
    ctx,
    "ShoppingCart",  // Service name
    "user123",       // Key (which cart)
    "AddItem",       // Handler name
).Request(CartItem{...})

// Call for user456's cart (completely separate)
err := restate.Object[error](
    ctx,
    "ShoppingCart",
    "user456",       // Different key = different state
    "AddItem",
).Request(CartItem{...})
```

### Getting the Current Key

Inside a Virtual Object handler:

```go
func (ShoppingCart) AddItem(
    ctx restate.ObjectContext,
    item CartItem,
) error {
    // Get the key for this invocation
    cartID := restate.Key(ctx)
    ctx.Log().Info("Adding item", "cartId", cartID)
    
    // State operations automatically scoped to this key
    items, _ := restate.Get[[]CartItem](ctx, "items")
    // ...
}
```

## 💾 State Management

### State Operations

Restate provides three core state operations:

```go
// 1. Get - Retrieve state
items, err := restate.Get[[]CartItem](ctx, "items")

// 2. Set - Store state
restate.Set(ctx, "items", items)

// 3. Clear - Delete state
restate.Clear(ctx, "items")
```

### State is Durable

```
First Call:
┌──────────────────────┐
│ AddItem("Item1")     │
│ Set("items", [...])  │ ← State saved
└──────────────────────┘

Second Call (same key):
┌──────────────────────┐
│ AddItem("Item2")     │
│ Get("items")         │ ← Returns ["Item1"]
│ Append Item2         │
│ Set("items", [...])  │ ← State updated
└──────────────────────┘

After Failure & Recovery:
┌──────────────────────┐
│ GetItems()           │
│ Get("items")         │ ← Still returns ["Item1", "Item2"]!
└──────────────────────┘
```

### Complete State API

```go
// Get with default value
items, err := restate.Get[[]CartItem](ctx, "items")
if err != nil {
    // Handle error
}
// If key doesn't exist, returns zero value

// Get all state keys
keys, err := restate.Keys(ctx)
// Returns: ["items", "total", "coupon"]

// Clear specific key
restate.Clear(ctx, "coupon")

// Clear all state for this object key
restate.ClearAll(ctx)
```

## 🔒 Exclusive vs Concurrent Handlers

### Exclusive Handlers (Default)

**Exclusive handlers** run **one at a time per key**:

```go
// Exclusive - modifies state
func (ShoppingCart) AddItem(
    ctx restate.ObjectContext,  // ← ObjectContext
    item CartItem,
) error {
    // Only ONE AddItem per key at a time
    items, _ := restate.Get[[]CartItem](ctx, "items")
    items = append(items, item)
    restate.Set(ctx, "items", items)
    return nil
}
```

**Execution Timeline:**

```
Time →
Key: user123
  |---- AddItem("Item1") ----|
                               |---- AddItem("Item2") ----|
                                                            |---- Checkout ----|

Key: user456
  |---- AddItem("Item3") ----|  ← Runs in parallel with user123
```

### Concurrent Handlers (Read-Only)

**Concurrent handlers** can run **simultaneously per key**:

```go
// Concurrent - read-only
func (ShoppingCart) GetItems(
    ctx restate.ObjectSharedContext,  // ← ObjectSharedContext!
    _ restate.Void,
) ([]CartItem, error) {
    // Multiple GetItems can run at once
    items, _ := restate.Get[[]CartItem](ctx, "items")
    return items, nil
}
```

**Why Concurrent?**
- Read-only operations don't conflict
- Better performance for reads
- Multiple readers, one writer pattern

**Execution Timeline:**

```
Time →
Key: user123
  |---- AddItem("Item1") ----|
                           |---- GetItems() ----|  ← Waits for AddItem
                           |---- GetItems() ----|  ← Can run in parallel with other GetItems
```

## 🎓 Declaring Virtual Objects

### Basic Structure

```go
// 1. Define the struct
type MyObject struct{}

// 2. Exclusive handler (writes)
func (MyObject) ExclusiveHandler(
    ctx restate.ObjectContext,  // ← ObjectContext
    input InputType,
) (OutputType, error) {
    // Get/Set state
    // Only one per key at a time
}

// 3. Concurrent handler (reads)
func (MyObject) ConcurrentHandler(
    ctx restate.ObjectSharedContext,  // ← ObjectSharedContext
    input InputType,
) (OutputType, error) {
    // Get state (read-only)
    // Multiple per key allowed
}

// 4. Register with Restate
restate.Reflect(MyObject{})
```

### Context Types

| Context Type | Use Case | State Access | Concurrency |
|--------------|----------|--------------|-------------|
| `ObjectContext` | Exclusive handlers | Get, Set, Clear, ClearAll | One at a time per key |
| `ObjectSharedContext` | Concurrent handlers | Get, Keys (read-only) | Multiple per key |

## 🏗️ Shopping Cart Example

### Complete Implementation

```go
type ShoppingCart struct{}

type CartItem struct {
    ProductID string  `json:"productId"`
    Quantity  int     `json:"quantity"`
    Price     float64 `json:"price"`
}

type Cart struct {
    Items []CartItem `json:"items"`
    Total float64    `json:"total"`
}

// Exclusive: Add item
func (ShoppingCart) AddItem(
    ctx restate.ObjectContext,
    item CartItem,
) error {
    cartID := restate.Key(ctx)
    ctx.Log().Info("Adding item", "cartId", cartID)
    
    // Get current cart
    cart, _ := restate.Get[Cart](ctx, "cart")
    
    // Update
    cart.Items = append(cart.Items, item)
    cart.Total += item.Price * float64(item.Quantity)
    
    // Save
    restate.Set(ctx, "cart", cart)
    
    return nil
}

// Concurrent: Get cart (read-only)
func (ShoppingCart) GetCart(
    ctx restate.ObjectSharedContext,
    _ restate.Void,
) (Cart, error) {
    cart, _ := restate.Get[Cart](ctx, "cart")
    return cart, nil
}

// Exclusive: Checkout (clears cart)
func (ShoppingCart) Checkout(
    ctx restate.ObjectContext,
    _ restate.Void,
) (float64, error) {
    // Get cart
    cart, _ := restate.Get[Cart](ctx, "cart")
    total := cart.Total
    
    // Clear cart after checkout
    restate.ClearAll(ctx)
    
    return total, nil
}
```

### Using the Cart

```go
// Add items
restate.Object[error](ctx, "ShoppingCart", "user123", "AddItem").
    Request(CartItem{ProductID: "prod1", Quantity: 2, Price: 29.99})

restate.Object[error](ctx, "ShoppingCart", "user123", "AddItem").
    Request(CartItem{ProductID: "prod2", Quantity: 1, Price: 49.99})

// Get cart (concurrent - fast!)
cart, _ := restate.Object[Cart](ctx, "ShoppingCart", "user123", "GetCart").
    Request(restate.Void{})

// Checkout (exclusive - clears state)
total, _ := restate.Object[float64](ctx, "ShoppingCart", "user123", "Checkout").
    Request(restate.Void{})
```

## ⚠️ Common Patterns and Anti-Patterns

### ✅ Correct: State Per Key

```go
func (UserProfile) UpdateEmail(
    ctx restate.ObjectContext,
    newEmail string,
) error {
    // State is automatically scoped to Key(ctx)
    profile, _ := restate.Get[Profile](ctx, "profile")
    profile.Email = newEmail
    restate.Set(ctx, "profile", profile)
    return nil
}

// user123's state
Object(ctx, "UserProfile", "user123", "UpdateEmail").Request("a@b.com")

// user456's state (completely separate)
Object(ctx, "UserProfile", "user456", "UpdateEmail").Request("c@d.com")
```

### ❌ Anti-Pattern: Calling Same Key Recursively

```go
// ❌ WRONG - Deadlock!
func (ShoppingCart) AddItemAndCheckout(
    ctx restate.ObjectContext,
    item CartItem,
) error {
    // Add item
    items, _ := restate.Get[[]CartItem](ctx, "items")
    items = append(items, item)
    restate.Set(ctx, "items", items)
    
    // ❌ This deadlocks! Same key calling itself
    total, _ := restate.Object[float64](
        ctx,
        "ShoppingCart",
        restate.Key(ctx),  // ← Same key!
        "Checkout",
    ).Request(restate.Void{})
    
    return nil
}
```

**Why Deadlock?**
1. `AddItemAndCheckout` holds exclusive lock for key "user123"
2. It calls `Checkout` for same key "user123"
3. `Checkout` waits for exclusive lock
4. But lock is held by `AddItemAndCheckout`!
5. Deadlock! 🔒

**Solution:**
```go
// ✅ CORRECT - Do work in same handler
func (ShoppingCart) AddItemAndCheckout(
    ctx restate.ObjectContext,
    item CartItem,
) (float64, error) {
    // Add item
    items, _ := restate.Get[[]CartItem](ctx, "items")
    items = append(items, item)
    
    // Calculate total
    total := 0.0
    for _, i := range items {
        total += i.Price * float64(i.Quantity)
    }
    
    // Clear cart
    restate.ClearAll(ctx)
    
    return total, nil
}
```

### ✅ Correct: Cross-Object Calls

```go
// Different keys - no deadlock
func (Order) CreateFromCart(
    ctx restate.ObjectContext,
    cartID string,
) error {
    orderID := restate.Key(ctx)
    
    // Call different object (different key)
    cart, _ := restate.Object[Cart](
        ctx,
        "ShoppingCart",
        cartID,  // Different key
        "GetCart",
    ).Request(restate.Void{})
    
    // Create order
    order := CreateOrder(cart)
    restate.Set(ctx, "order", order)
    
    return nil
}
```

### ❌ Anti-Pattern: Using ObjectContext in Concurrent Handler

```go
// ❌ WRONG - Type mismatch!
func (ShoppingCart) GetItems(
    ctx restate.ObjectContext,  // ← Wrong! Should be ObjectSharedContext
    _ restate.Void,
) ([]CartItem, error) {
    // Compiler error: concurrent handlers must use ObjectSharedContext
}
```

### ✅ Correct: Use Shared Context for Reads

```go
// ✅ CORRECT
func (ShoppingCart) GetItems(
    ctx restate.ObjectSharedContext,  // ← Correct!
    _ restate.Void,
) ([]CartItem, error) {
    items, _ := restate.Get[[]CartItem](ctx, "items")
    return items, nil
}
```

## 📊 State Storage Considerations

### State is Versioned

Every state change creates a new version:

```
Version 1: Set("items", [Item1])
Version 2: Set("items", [Item1, Item2])
Version 3: Set("items", [Item1, Item2, Item3])

On replay: Restate uses version history
```

### State Size Limits

- Individual state keys: **~1MB recommended**
- Total state per object: **~10MB recommended**
- For larger data: Store references, not full content

```go
// ❌ BAD - Storing large data
restate.Set(ctx, "image", largeImageBytes)  // 50MB

// ✅ GOOD - Store reference
restate.Set(ctx, "imageUrl", "s3://bucket/image123.jpg")
```

### State Retention

State persists until explicitly cleared:

```go
// Clear specific key
restate.Clear(ctx, "coupon")

// Clear all state for this object
restate.ClearAll(ctx)
```

## 🎯 When to Use Virtual Objects

### Use Virtual Objects When:

✅ **Per-entity state** (users, orders, carts)
✅ **Need exclusive execution** (prevent race conditions)
✅ **State fits in memory** (~1-10MB)
✅ **Moderate read/write ratio**

### Use Basic Services When:

✅ **Stateless operations** (calculations, aggregations)
✅ **Full parallelism needed**
✅ **No state between calls**

### Use External Database When:

✅ **Large datasets** (>10MB per entity)
✅ **Complex queries** (SQL, aggregations)
✅ **Shared across services**

## ✅ Concept Check

Before moving to hands-on, ensure you understand:

- [ ] Difference between Basic Service and Virtual Object
- [ ] How key-based addressing works
- [ ] State operations: Get, Set, Clear, ClearAll
- [ ] Exclusive vs concurrent handlers
- [ ] Why recursive same-key calls deadlock
- [ ] When to use Virtual Objects vs Basic Services

## 🎯 Next Step

Ready to build a real shopping cart with stateful handlers!

👉 **Continue to [Hands-On Tutorial](./02-hands-on.md)**

---

**Key Takeaway:** Virtual Objects provide durable, key-scoped state with exclusive execution guarantees!
