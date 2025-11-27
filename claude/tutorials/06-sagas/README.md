# Module 06: Sagas - Distributed Transactions

> **Build reliable distributed transactions with compensation patterns**

## 🎯 Module Overview

**Sagas** are a pattern for managing distributed transactions across multiple services where traditional ACID transactions aren't possible. In Restate, Sagas are implemented as Workflows with built-in compensation logic to roll back partial failures.

### What You'll Learn

- ✅ What Sagas are and when to use them
- ✅ Forward recovery vs backward recovery
- ✅ Implementing compensation logic
- ✅ Building reliable multi-service transactions
- ✅ Handling partial failures gracefully
- ✅ Saga coordination patterns

### Real-World Use Cases

- 🛒 **E-commerce Order** - Payment → Inventory → Shipping
- ✈️ **Travel Booking** - Flight + Hotel + Car rental
- 💰 **Money Transfer** - Debit account A → Credit account B
- 📦 **Supply Chain** - Order → Reserve → Manufacture → Ship
- 🎫 **Event Ticketing** - Reserve seats → Process payment → Issue tickets

## 📊 The Problem: Distributed Transactions

### Traditional Transaction (Single Database)

```sql
BEGIN TRANSACTION;
  UPDATE accounts SET balance = balance - 100 WHERE id = 'A';
  UPDATE accounts SET balance = balance + 100 WHERE id = 'B';
COMMIT;
```

**Properties:**
- ✅ Atomic (all or nothing)
- ✅ Consistent
- ✅ Isolated
- ✅ Durable

### Distributed Transaction (Multiple Services)

```
Service A: Debit $100 from account A
Service B: Credit $100 to account B
Service C: Send notification
```

**Problem:** What if Service B fails after Service A succeeds?

❌ **Can't use database transactions** - services are independent
❌ **No atomic commit** across services
❌ **Partial failures** leave system in inconsistent state

## 💡 The Solution: Sagas

A **Saga** is a sequence of local transactions where each transaction updates its own service and publishes an event or message. If a step fails, compensating transactions run to undo the completed steps.

### Saga Pattern

```
Try Steps:          Compensating Steps:
┌─────────────┐     ┌──────────────────┐
│ Step 1      │ ───→│ Compensate 1     │
│ Step 2      │ ───→│ Compensate 2     │
│ Step 3 ❌   │     │ (not needed)     │
└─────────────┘     └──────────────────┘

If Step 3 fails:
1. Run Compensate 2
2. Run Compensate 1
3. Return to consistent state
```

## 🏗️ Module Structure

### 1. [Concepts](./01-concepts.md) (~30 min)
Learn about:
- Saga fundamentals
- Compensation patterns
- Forward vs backward recovery
- Saga coordination

### 2. [Hands-On](./02-hands-on.md) (~45 min)
Build a travel booking saga:
- Reserve flight
- Reserve hotel  
- Reserve car rental
- Automatic compensation on failure

### 3. [Validation](./03-validation.md) (~20 min)
Test:
- Successful saga completion
- Compensation on partial failure
- Idempotent compensations
- Saga state tracking

### 4. [Exercises](./04-exercises.md) (~60 min)
Practice building:
- E-commerce order saga
- Money transfer saga
- Event booking saga
- Multi-step workflow with rollback

## 🎓 Prerequisites

- ✅ Completed [Module 05](../05-workflows/README.md) - Workflows
- ✅ Understanding of distributed systems
- ✅ Familiarity with failure scenarios

## 🚀 Quick Start

```bash
# Navigate to module directory
cd ~/restate-tutorials/module06

# Follow hands-on tutorial
cat 02-hands-on.md
```

## 🎯 Learning Objectives

By the end of this module, you will:

1. **Understand Sagas**
   - What they are and why they're needed
   - When to use vs avoid Sagas
   - Common saga patterns

2. **Implement Compensation**
   - Write compensating transactions
   - Handle idempotency
   - Track saga progress

3. **Build Reliable Sagas**
   - Coordinate multiple services
   - Handle partial failures gracefully
   - Ensure eventual consistency

4. **Handle Edge Cases**
   - Compensation failures
   - Retries and idempotency
   - Saga recovery

## 📖 Module Flow

```
Concepts → Hands-On → Validation → Exercises
   ↓          ↓          ↓            ↓
 Theory → Build Saga → Test Rollback → Practice
```

## 🔑 Key Concept Preview

### Basic Saga Structure

```go
type TravelBookingSaga struct{}

func (TravelBookingSaga) Run(
    ctx restate.WorkflowContext,
    booking BookingRequest,
) (BookingResult, error) {
    var completed []string // Track completed steps
    
    // Step 1: Reserve flight
    flightID, err := reserveFlight(ctx, booking.FlightInfo)
    if err != nil {
        return BookingResult{Status: "failed"}, nil
    }
    completed = append(completed, "flight:"+flightID)
    
    // Step 2: Reserve hotel
    hotelID, err := reserveHotel(ctx, booking.HotelInfo)
    if err != nil {
        // Compensate: Cancel flight
        cancelFlight(ctx, flightID)
        return BookingResult{Status: "failed"}, nil
    }
    completed = append(completed, "hotel:"+hotelID)
    
    // Step 3: Reserve car
    carID, err := reserveCar(ctx, booking.CarInfo)
    if err != nil {
        // Compensate: Cancel hotel and flight
        cancelCar(ctx, carID)
        cancelHotel(ctx, hotelID)
        cancelFlight(ctx, flightID)
        return BookingResult{Status: "failed"}, nil
    }
    
    // All succeeded!
    return BookingResult{
        Status:   "confirmed",
        FlightID: flightID,
        HotelID:  hotelID,
        CarID:    carID,
    }, nil
}
```

### With Restate Workflows

```go
func (TravelBookingSaga) Run(
    ctx restate.WorkflowContext,
    booking BookingRequest,
) (BookingResult, error) {
    // Steps are durable - survive failures!
    // Each external call is journaled
    // Compensation is automatic on workflow failure
}
```

## 💡 Why Sagas with Restate?

**Traditional Saga Challenges:**
- 😰 Manual compensation tracking
- 😰 Complex retry logic
- 😰 State management across failures
- 😰 Ensuring idempotency

**Restate Sagas:**
- ✅ Automatic journaling of completed steps
- ✅ Built-in retry mechanism
- ✅ Durable state across failures
- ✅ Idempotency built-in

## 🆚 Saga vs Workflow

| Feature | Workflow | Saga |
|---------|----------|------|
| **Purpose** | Long-running orchestration | Distributed transaction |
| **Focus** | Wait for events | Coordinate services |
| **Failure** | Retry or timeout | Compensate completed steps |
| **Pattern** | Human-in-the-loop | Multi-service coordination |
| **Example** | Approval process | Order processing |

**Both are Workflows** - Saga is a specific pattern using Workflows!

## ⚠️ Important Concepts

### Compensation Must Be Idempotent

```go
// ✅ Idempotent compensation
func cancelFlight(ctx restate.WorkflowContext, flightID string) error {
    // Calling multiple times has same effect
    return flightService.Cancel(flightID)
    // Flight service: if already cancelled, returns success
}

// ❌ Non-idempotent compensation
func refundPayment(ctx restate.WorkflowContext, amount float64) error {
    // Calling multiple times refunds multiple times!
    return paymentService.Refund(amount)
}
```

### Forward vs Backward Recovery

**Backward Recovery (Compensation):**
- Undo completed steps
- Return to initial state
- Most common pattern

**Forward Recovery (Retry):**
- Continue despite failures
- Retry failed steps
- Use when compensation is hard/impossible

## 🎯 Ready to Start?

Let's dive into saga patterns and compensation!

👉 **Start with [Concepts](./01-concepts.md)**

---

**Questions?** Check the main [tutorials README](../README.md) or review [Module 05](../05-workflows/README.md).
