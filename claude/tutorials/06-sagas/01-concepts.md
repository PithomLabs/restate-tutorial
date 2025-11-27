# Concepts: Sagas and Distributed Transactions

> **Understanding reliable distributed transactions with compensation**

## 🎯 What is a Saga?

### Definition

A **Saga** is a pattern for managing **long-running transactions** that span multiple services. Instead of a single ACID transaction, a saga is a sequence of local transactions, where each step:
1. Performs a local transaction
2. Publishes an event or returns
3. Can be **compensated** (undone) if later steps fail

Think of a Saga as a "try-catch" block for distributed systems.

### Real-World Analogy

**Booking a Vacation:**
1. Reserve flight ✅
2. Reserve hotel ✅  
3. Reserve car ❌ (fails - sold out)
4. **Compensation:** Cancel hotel, cancel flight

Without compensation, you'd be stuck with a hotel reservation and flight with no car!

## 🆚 ACID vs Saga

### ACID Transaction (Single Database)

```go
// All or nothing
db.Transaction(func(tx *sql.Tx) error {
    tx.Exec("UPDATE account SET balance = balance - 100 WHERE id = 1")
    tx.Exec("UPDATE account SET balance = balance + 100 WHERE id = 2")
    return nil // Commits or rolls back atomically
})
```

**Properties:**
- ✅ **Atomic** - All or nothing
- ✅ **Consistent** - Valid state always
- ✅ **Isolated** - No interference
- ✅ **Durable** - Persisted

### Saga (Multiple Services)

```go
// Try each step, compensate on failure
func ProcessOrder(ctx restate.WorkflowContext, order Order) error {
    // Step 1
    paymentID, err := chargePayment(ctx, order.Amount)
    if err != nil {
        return err // No compensation needed
    }
    
    // Step 2
    err = reserveInventory(ctx, order.Items)
    if err != nil {
        refundPayment(ctx, paymentID) // Compensate step 1
        return err
    }
    
    // Step 3
    err = shipOrder(ctx, order)
    if err != nil {
        releaseInventory(ctx, order.Items) // Compensate step 2
        refundPayment(ctx, paymentID)      // Compensate step 1
        return err
    }
    
    return nil
}
```

**Properties:**
- ⚠️ **Eventually Consistent** - Temporary inconsistency allowed
- ⚠️ **No Isolation** - Other transactions see intermediate state
- ✅ **Durable** - Each step is persisted
- ✅ **Compensatable** - Can undo completed steps

## 🎯 Next Step

Ready to build a real distributed transaction!

👉 **Continue to [Hands-On Tutorial](./02-hands-on.md)**

---

**Key Takeaway:** Sagas enable reliable distributed transactions through compensation - if a step fails, we systematically undo completed work!
