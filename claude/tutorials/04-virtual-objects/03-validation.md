# Validation: Testing Stateful Virtual Objects

> **Verify state persistence, isolation, and exclusive execution**

## 🎯 Objectives

Verify that:
- ✅ State persists across multiple calls
- ✅ Each key has isolated state
- ✅ Exclusive handlers run one at a time per key
- ✅ Concurrent handlers allow parallel reads
- ✅ State survives service restarts
- ✅ Checkout clears state properly

## 📋 Pre-Validation Checklist

- [ ] Restate server running (ports 8080/9080)
- [ ] Cart service running (port 9090)
- [ ] Service registered with Restate
- [ ] `curl` and `jq` available

## 🧪 Test Suite

### Test 1: State Persistence

**Purpose:** Verify state persists across multiple calls

```bash
# Start fresh
curl -X POST http://localhost:9080/ShoppingCart/test-user-1/ClearCart \
  -H 'Content-Type: application/json' \
  -d 'null'

# Add first item
curl -X POST http://localhost:9080/ShoppingCart/test-user-1/AddItem \
  -H 'Content-Type: application/json' \
  -d '{
    "productId": "prod1",
    "productName": "Product 1",
    "quantity": 2,
    "price": 29.99
  }'

# Add second item
curl -X POST http://localhost:9080/ShoppingCart/test-user-1/AddItem \
  -H 'Content-Type: application/json' \
  -d '{
    "productId": "prod2",
    "productName": "Product 2",
    "quantity": 1,
    "price": 49.99
  }'

# Get cart
curl -s -X POST http://localhost:9080/ShoppingCart/test-user-1/GetCart \
  -H 'Content-Type: application/json' \
  -d 'null' | jq '{itemCount, subtotal, total}'
```

**Expected Output:**
```json
{
  "itemCount": 3,
  "subtotal": 109.97,
  "total": 118.77
}
```

**Validation:**
- ✅ Second AddItem saw the first item
- ✅ State accumulated across calls
- ✅ Math is correct (2×$29.99 + 1×$49.99 = $109.97)

---

### Test 2: State Isolation Per Key

**Purpose:** Verify different keys have separate state

```bash
# User A adds items
curl -X POST http://localhost:9080/ShoppingCart/userA/AddItem \
  -H 'Content-Type: application/json' \
  -d '{"productId": "laptop", "productName": "Laptop", "quantity": 1, "price": 999.99}'

# User B adds different items
curl -X POST http://localhost:9080/ShoppingCart/userB/AddItem \
  -H 'Content-Type: application/json' \
  -d '{"productId": "mouse", "productName": "Mouse", "quantity": 3, "price": 25.00}'

# Check User A's cart
echo "User A:"
curl -s -X POST http://localhost:9080/ShoppingCart/userA/GetCart \
  -H 'Content-Type: application/json' \
  -d 'null' | jq '{items: [.items[].productName], total}'

# Check User B's cart  
echo "User B:"
curl -s -X POST http://localhost:9080/ShoppingCart/userB/GetCart \
  -H 'Content-Type: application/json' \
  -d 'null' | jq '{items: [.items[].productName], total}'
```

**Expected:**
```
User A:
{
  "items": ["Laptop"],
  "total": 1079.99
}
User B:
{
  "items": ["Mouse"],
  "total": 81.00
}
```

**Validation:**
- ✅ User A sees only laptop
- ✅ User B sees only mouse
- ✅ Complete state isolation by key

---

### Test 3: Idempotent Operations

**Purpose:** Verify same request with idempotency key is idempotent

```bash
# First call
curl -s -X POST http://localhost:9080/ShoppingCart/idem-test/AddItem \
  -H 'Content-Type: application/json' \
  -H 'idempotency-key: add-item-001' \
  -d '{
    "productId": "prod123",
    "productName": "Product",
    "quantity": 5,
    "price": 10.00
  }'

# Second call (same idempotency key)
curl -s -X POST http://localhost:9080/ShoppingCart/idem-test/AddItem \
  -H 'Content-Type: application/json' \
  -H 'idempotency-key: add-item-001' \
  -d '{
    "productId": "prod123",
    "productName": "Product",
    "quantity": 5,
    "price": 10.00
  }'

# Check cart - should have 5 items, not 10!
curl -s -X POST http://localhost:9080/ShoppingCart/idem-test/GetCart \
  -H 'Content-Type: application/json' \
  -d 'null' | jq '.itemCount'
```

**Expected Output:**
```
5
```

**Validation:**
- ✅ Second call was deduplicated
- ✅ Only 5 items in cart (not 10)
- ✅ Idempotency key worked

---

### Test 4: Update Operations

**Purpose:** Test updating existing items

```bash
# Add item
curl -X POST http://localhost:9080/ShoppingCart/update-test/AddItem \
  -H 'Content-Type: application/json' \
  -d '{"productId": "widget", "productName": "Widget", "quantity": 3, "price": 15.00}'

# Update quantity
curl -X POST http://localhost:9080/ShoppingCart/update-test/UpdateQuantity \
  -H 'Content-Type: application/json' \
  -d '{"productId": "widget", "quantity": 10}'

# Verify
curl -s -X POST http://localhost:9080/ShoppingCart/update-test/GetCart \
  -H 'Content-Type: application/json' \
  -d 'null' | jq '.items[0].quantity'
```

**Expected:**
```
10
```

**Validation:**
- ✅ Quantity updated from 3 to 10
- ✅ Same product, new quantity

---

### Test 5: Remove Operations

**Purpose:** Test removing items

```bash
# Add two items
curl -X POST http://localhost:9080/ShoppingCart/remove-test/AddItem \
  -H 'Content-Type: application/json' \
  -d '{"productId": "item1", "productName": "Item 1", "quantity": 1, "price": 10.00}'

curl -X POST http://localhost:9080/ShoppingCart/remove-test/AddItem \
  -H 'Content-Type: application/json' \
  -d '{"productId": "item2", "productName": "Item 2", "quantity": 1, "price": 20.00}'

# Remove first item
curl -X POST http://localhost:9080/ShoppingCart/remove-test/RemoveItem \
  -H 'Content-Type: application/json' \
  -d '"item1"'

# Verify only item2 remains
curl -s -X POST http://localhost:9080/ShoppingCart/remove-test/GetCart \
  -H 'Content-Type: application/json' \
  -d 'null' | jq '{items: [.items[].productId], count: .itemCount}'
```

**Expected:**
```json
{
  "items": ["item2"],
  "count": 1
}
```

**Validation:**
- ✅ Item1 removed
- ✅ Item2 still present
- ✅ Count correct

---

### Test 6: Coupon Application

**Purpose:** Test discount logic

```bash
# Add items
curl -X POST http://localhost:9080/ShoppingCart/coupon-test/AddItem \
  -H 'Content-Type: application/json' \
  -d '{"productId": "prod", "productName": "Product", "quantity": 1, "price": 100.00}'

# Get initial total
echo "Before coupon:"
curl -s -X POST http://localhost:9080/ShoppingCart/coupon-test/GetCart \
  -H 'Content-Type: application/json' \
  -d 'null' | jq '{subtotal, discount, total}'

# Apply 20% off coupon
curl -X POST http://localhost:9080/ShoppingCart/coupon-test/ApplyCoupon \
  -H 'Content-Type: application/json' \
  -d '"SAVE20"'

# Get new total
echo "After coupon:"
curl -s -X POST http://localhost:9080/ShoppingCart/coupon-test/GetCart \
  -H 'Content-Type: application/json' \
  -d 'null' | jq '{subtotal, discount, total, couponCode}'
```

**Expected:**
```
Before coupon:
{
  "subtotal": 100,
  "discount": 0,
  "total": 108
}
After coupon:
{
  "subtotal": 100,
  "discount": 20,
  "total": 86.4,
  "couponCode": "SAVE20"
}
```

**Calculation:**
- Subtotal: $100
- Discount (20%): $20
- After discount: $80
- Tax (8%): $6.40
- **Total: $86.40** ✅

---

### Test 7: Checkout Flow

**Purpose:** Test complete checkout process

```bash
# Setup cart
curl -X POST http://localhost:9080/ShoppingCart/checkout-test/AddItem \
  -H 'Content-Type: application/json' \
  -d '{"productId": "prod1", "productName": "Product 1", "quantity": 2, "price": 50.00}'

# Apply coupon
curl -X POST http://localhost:9080/ShoppingCart/checkout-test/ApplyCoupon \
  -H 'Content-Type: application/json' \
  -d '"SAVE10"'

# Checkout
echo "Checkout result:"
curl -s -X POST http://localhost:9080/ShoppingCart/checkout-test/Checkout \
  -H 'Content-Type: application/json' \
  -d 'null' | jq '{orderId, total, itemCount, couponCode}'

# Verify cart is empty
echo "Cart after checkout:"
curl -s -X POST http://localhost:9080/ShoppingCart/checkout-test/GetCart \
  -H 'Content-Type: application/json' \
  -d 'null' | jq '{itemCount, total}'
```

**Expected:**
```
Checkout result:
{
  "orderId": "ORD_abc12345",
  "total": 97.20,
  "itemCount": 2,
  "couponCode": "SAVE10"
}
Cart after checkout:
{
  "itemCount": 0,
  "total": 0
}
```

**Validation:**
- ✅ Order created with ID
- ✅ Total calculated correctly
- ✅ Cart cleared after checkout

---

### Test 8: Concurrent Reads Don't Block

**Purpose:** Verify concurrent handlers allow parallel execution

```bash
# Setup cart
curl -X POST http://localhost:9080/ShoppingCart/concurrent-test/AddItem \
  -H 'Content-Type: application/json' \
  -d '{"productId": "prod1", "productName": "Product", "quantity": 1, "price": 50.00}'

# Launch 5 concurrent GetCart requests
for i in {1..5}; do
  (time curl -s -X POST http://localhost:9080/ShoppingCart/concurrent-test/GetCart \
    -H 'Content-Type: application/json' \
    -d 'null' > /tmp/cart_$i.json) 2>&1 | grep real &
done

wait
echo "All reads completed"

# Verify all got same result
md5sum /tmp/cart_*.json
```

**Expected:**
- All 5 requests complete quickly (~simultaneously)
- All have identical content (same md5sum)

**Validation:**
- ✅ Concurrent reads don't block each other
- ✅ All returned consistent state

---

### Test 9: Exclusive Writes Execute Sequentially

**Purpose:** Verify exclusive handlers are sequential per key

```bash
# Launch 3 concurrent AddItem requests
for i in {1..3}; do
  curl -s -X POST http://localhost:9080/ShoppingCart/sequential-test/AddItem \
    -H 'Content-Type: application/json' \
    -d '{
      "productId": "prod'$i'",
      "productName": "Product '$i'",
      "quantity": 1,
      "price": 10.00
    }' &
done

wait
echo "All additions completed"

# Verify all 3 items added
curl -s -X POST http://localhost:9080/ShoppingCart/sequential-test/GetCart \
  -H 'Content-Type: application/json' \
  -d 'null' | jq '.itemCount'
```

**Expected:**
```
3
```

**Validation:**
- ✅ All 3 additions completed
- ✅ No race conditions (all items present)
- ✅ Exclusive execution prevented conflicts

---

### Test 10: Error Handling

**Purpose:** Test terminal errors

```bash
# Try to checkout empty cart
curl -s -X POST http://localhost:9080/ShoppingCart/error-test/Checkout \
  -H 'Content-Type: application/json' \
  -d 'null' 2>&1

# Try invalid coupon
curl -X POST http://localhost:9080/ShoppingCart/error-test/AddItem \
  -H 'Content-Type: application/json' \
  -d '{"productId": "prod", "productName": "Product", "quantity": 1, "price": 10.00}'

curl -s -X POST http://localhost:9080/ShoppingCart/error-test/ApplyCoupon \
  -H 'Content-Type: application/json' \
  -d '"INVALID_CODE"' 2>&1

# Try negative quantity
curl -s -X POST http://localhost:9080/ShoppingCart/error-test/AddItem \
  -H 'Content-Type: application/json' \
  -d '{"productId": "prod", "productName": "Product", "quantity": -5, "price": 10.00}' 2>&1
```

**Expected:** All return error status codes (400/404)

**Validation:**
- ✅ Empty cart checkout fails
- ✅ Invalid coupon rejected
- ✅ Negative quantity rejected
- ✅ Clear error messages

---

### Test 11: State Survives Restart

**Purpose:** Verify state is durable

```bash
# Add item
curl -X POST http://localhost:9080/ShoppingCart/restart-test/AddItem \
  -H 'Content-Type: application/json' \
  -d '{"productId": "durable", "productName": "Durable Item", "quantity": 1, "price": 100.00}'

# Save current state
curl -s -X POST http://localhost:9080/ShoppingCart/restart-test/GetCart \
  -H 'Content-Type: application/json' \
  -d 'null' > /tmp/cart_before.json

# Restart service
echo "Restart your cart service now, then press Enter..."
read

# Check state after restart
curl -s -X POST http://localhost:9080/ShoppingCart/restart-test/GetCart \
  -H 'Content-Type: application/json' \
  -d 'null' > /tmp/cart_after.json

# Compare
diff /tmp/cart_before.json /tmp/cart_after.json && echo "✅ State survived restart!"
```

**Validation:**
- ✅ State identical before and after restart
- ✅ No data loss

---

## 📊 Test Results Summary

| Test | Purpose | Expected | Pass/Fail |
|------|---------|----------|-----------|
| State Persistence | State accumulates | 3 items, correct total | |
| State Isolation | Keys are separate | Different carts | |
| Idempotency | Deduplication works | 5 items, not 10 | |
| Updates | Quantity changes | Updated quantity | |
| Removals | Item deleted | Only item2 remains | |
| Coupons | Discount applied | Correct calculations | |
| Checkout | Clears cart | Empty after checkout | |
| Concurrent Reads | Parallel execution | All succeed quickly | |
| Exclusive Writes | Sequential | All 3 items added | |
| Errors | Validation works | Appropriate errors | |
| Restart | Durable state | State survived | |

## 🔍 Advanced Validation

### View Journal Entries

```bash
# Make some calls
curl -X POST http://localhost:9080/ShoppingCart/journal-test/AddItem \
  -H 'Content-Type: application/json' \
  -H 'idempotency-key: journal-inspect' \
  -d '{"productId": "prod", "productName": "Product", "quantity": 1, "price": 50.00}'

# Get invocation ID
INV_ID=$(curl -s 'http://localhost:8080/invocations?target_service=ShoppingCart&target_key=journal-test' | \
  jq -r '.invocations[0].id')

echo "Invocation ID: $INV_ID"

# View journal
curl -s "http://localhost:8080/invocations/$INV_ID/journal" | \
  jq '.entries[] | {index, type, name}'
```

**Look For:**
- `GetState` entries (reading cart)
- `SetState` entries (saving cart)
- `Output` entry (response)

## ✅ Validation Checklist

- [ ] ✅ State persists across calls
- [ ] ✅ Keys have isolated state
- [ ] ✅ Idempotency works
- [ ] ✅ Updates and removals work
- [ ] ✅ Coupons apply correctly
- [ ] ✅ Checkout clears cart
- [ ] ✅ Concurrent reads don't block
- [ ] ✅ Exclusive writes are sequential
- [ ] ✅ Errors handled properly
- [ ] ✅ State survives restarts

## 🎓 What You Learned

1. **State is Durable** - Survives failures and restarts
2. **State is Isolated** - Each key has separate state
3. **Exclusive = Safe** - Prevents race conditions
4. **Concurrent = Fast** - Multiple reads don't block
5. **Idempotency Works** - Duplicate requests deduplicated

## 🐛 Troubleshooting

### State Not Persisting

Check:
1. Using `restate.Set()` to save state
2. Not returning errors before Set
3. Service registered correctly

### Keys Interfering

Ensure:
1. Using correct key in URL
2. Not hardcoding keys in handler

### Concurrent Handlers Blocking

Verify:
1. Using `ObjectSharedContext` (not `ObjectContext`)
2. Handler is truly read-only

## 🎯 Next Steps

Excellent! Your stateful service is working perfectly.

Now practice with more exercises:

👉 **Continue to [Exercises](./04-exercises.md)**

---

**Questions?** Review [concepts](./01-concepts.md) or [hands-on](./02-hands-on.md)!
