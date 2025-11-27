# Module 04: Virtual Objects - Stateful Services

> **Build key-addressable, stateful services with durable state**

## 🎯 Module Overview

In previous modules, we built **stateless** services where each request is independent. Now we'll learn to build **stateful** services using **Virtual Objects** - services that maintain durable state keyed by an identifier.

### What You'll Learn

- ✅ What Virtual Objects are and when to use them
- ✅ Key-addressable state management
- ✅ Exclusive vs concurrent handlers
- ✅ State operations (`Get`, `Set`, `Clear`)
- ✅ Building a shopping cart service
- ✅ Avoiding deadlocks with object calls

### Real-World Use Cases

- 🛒 **Shopping Carts** - Per-user shopping state
- 👤 **User Profiles** - Per-user preferences and data
- 📦 **Order Management** - Track per-order state
- 🎮 **Game Sessions** - Per-game state
- 💳 **Wallets** - Per-user balance tracking

## 📊 Conceptual Comparison

| Feature | Basic Service | Virtual Object |
|---------|---------------|----------------|
| **State** | None (stateless) | Durable state per key |
| **Context** | `restate.Context` | `restate.ObjectContext` |
| **Addressing** | Service + handler | Service + key + handler |
| **Concurrency** | Full parallelism | Exclusive by default |
| **Use Case** | API aggregation | User accounts, carts |

## 🏗️ Module Structure

### 1. [Concepts](./01-concepts.md) (~30 min)
Learn about:
- Virtual Object fundamentals
- State management
- Exclusive vs concurrent handlers
- Addressing and keying

### 2. [Hands-On](./02-hands-on.md) (~45 min)
Build a complete shopping cart system:
- Add/remove items
- Update quantities
- Calculate totals
- Checkout with state transitions

### 3. [Validation](./03-validation.md) (~20 min)
Test:
- State persistence across calls
- Exclusive execution guarantees
- Concurrent handler behavior
- State operations (get, set, clear)

### 4. [Exercises](./04-exercises.md) (~60 min)
Practice building:
- User wallet with transactions
- Inventory management
- Order state machine
- Multi-tenant counters

## 🎓 Prerequisites

- ✅ Completed [Module 01](../01-foundation/README.md) - Durable execution
- ✅ Completed [Module 02](../02-side-effects/README.md) - Side effects
- ✅ Completed [Module 03](../03-concurrency/README.md) - Concurrency
- ✅ Understanding of state vs stateless services

## 🚀 Quick Start

```bash
# Navigate to module directory
cd ~/restate-tutorials/module04

# Follow hands-on tutorial
cat 02-hands-on.md
```

## 🎯 Learning Objectives

By the end of this module, you will:

1. **Understand Virtual Objects**
   - What they are and when to use them
   - How they differ from Basic Services
   - Key-based addressing

2. **Master State Management**
   - Durably store and retrieve state
   - Update state transactionally
   - Clear state when needed

3. **Handle Concurrency**
   - Understand exclusive handlers (one at a time per key)
   - Use concurrent handlers for reads
   - Avoid deadlocks

4. **Build Real Applications**
   - Shopping carts
   - User accounts
   - Order tracking
   - Game state

## 📖 Module Flow

```
Concepts → Hands-On → Validation → Exercises
   ↓          ↓          ↓            ↓
Theory → Build Cart → Test State → Practice
```

## 🔑 Key Concept Preview

### Virtual Object Declaration

```go
type ShoppingCart struct{}

// Exclusive handler - one at a time per cart
func (ShoppingCart) AddItem(
    ctx restate.ObjectContext,
    item CartItem,
) error {
    // Get current state for THIS cart
    cart, _ := restate.Get[[]CartItem](ctx, "items")
    
    // Update state
    cart = append(cart, item)
    restate.Set(ctx, "items", cart)
    
    return nil
}

// Concurrent handler - multiple reads allowed
func (ShoppingCart) GetItems(
    ctx restate.ObjectSharedContext,
) ([]CartItem, error) {
    // Read-only access - can run concurrently
    items, _ := restate.Get[[]CartItem](ctx, "items")
    return items, nil
}
```

### Calling a Virtual Object

```go
// Call for specific cart (key = "user123")
err := restate.Object[error](
    ctx,
    "ShoppingCart",  // Service name
    "user123",       // Key (which cart)
    "AddItem",       // Handler
).Request(CartItem{...})
```

## 💡 Why Virtual Objects?

**Before (stateless):**
```go
// Must load from DB every time
func ProcessOrder(ctx restate.Context, orderID string) {
    order := loadFromDB(orderID)  // External call
    // Process...
    saveToD(order)  // External call
}
```

**After (stateful):**
```go
// State is durable and local
func ProcessOrder(ctx restate.ObjectContext, update OrderUpdate) {
    order, _ := restate.Get[Order](ctx, "order")
    // Process...
    restate.Set(ctx, "order", order)
    // No external DB needed!
}
```

**Benefits:**
- 🚀 **Faster** - No DB roundtrips
- 🔒 **Safer** - Exclusive execution prevents race conditions
- 💾 **Durable** - State survives failures
- 🎯 **Simpler** - No external state management

## ⚠️ Important Notes

### Exclusive Execution

Virtual Object handlers are **exclusive by default** per key:

```
Thread 1: cart.AddItem(key="user123", "item1")
Thread 2: cart.AddItem(key="user123", "item2")  ← Waits for Thread 1
Thread 3: cart.AddItem(key="user456", "item3")  ← Different key, runs in parallel
```

### State Scope

State is scoped to the object key:

```go
// user123's cart
cart123 := restate.Object[Cart](ctx, "ShoppingCart", "user123", "Get").Request()

// user456's cart (completely separate state)
cart456 := restate.Object[Cart](ctx, "ShoppingCart", "user456", "Get").Request()
```

## 🎯 Ready to Start?

Let's dive into the concepts!

👉 **Start with [Concepts](./01-concepts.md)**

---

**Questions?** Check the main [tutorials README](../README.md) or review previous modules.
