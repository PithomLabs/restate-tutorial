# Concepts: Microservices Orchestration

> **Learn how to coordinate distributed microservices**

## 🎯 What You'll Learn

- Orchestration vs choreography patterns
- Central coordinator design
- Service-to-service communication
- Distributed transaction coordination
- Compensation and failure handling
- Workflow state management

---

## 📖 Orchestration vs Choreography

### What is Orchestration?

**Orchestration** = Central coordinator directing services

```
TripOrchestrator (Central Brain)
    ├→ Calls FlightService
    ├→ Calls HotelService
    ├→ Calls PaymentService
    └→ Manages entire workflow

Benefits:
✅ Single source of truth
✅ Clear workflow logic
✅ Easy to debug
✅ Centralized error handling
✅ Full visibility
```

### What is Choreography?

**Choreography** = Services react to events independently

```
OrderService → emits "OrderCreated" event
    ↓
InventoryService → listens, reserves stock, emits "StockReserved"
    ↓
PaymentService → listens, charges customer, emits "PaymentProcessed"
    ↓
ShippingService → listens, creates label

Benefits:
✅ Loose coupling
✅ Service autonomy
✅ Scales independently

Drawbacks:
❌ No central view
❌ Hard to debug
❌ Scattered business logic
❌ Difficult to change workflow
```

### When to Use Which?

**Use Orchestration when:**
- Complex multi-step workflows
- Need central visibility
- Sequential dependencies
- Compensation required
- Single business process

**Use Choreography when:**
- Simple event reactions
- Independent services
- Eventual consistency OK
- No central coordinator needed

**Restate excels at:** Orchestration! 🎯

---

## 🏗️ Orchestration Architecture

### The Orchestrator Pattern

```
┌─────────────────────────────┐
│   TripOrchestrator          │  Central Coordinator
│   (Restate Workflow)        │  - Manages workflow
│                             │  - Stores state
│   BookTrip(request):        │  - Handles failures
│     1. Reserve flight       │
│     2. Reserve hotel        │
│     3. Charge payment       │
│     4. Confirm bookings     │
│     5. Send notification    │
└──────────┬──────────────────┘
           │
     ┌─────┴─────┐
     │           │           │
┌────▼───┐  ┌───▼────┐  ┌──▼─────┐
│ Flight │  │ Hotel  │  │Payment │  Services
│Service │  │Service │  │Service │  - Focused
└────────┘  └────────┘  └────────┘  - Reusable
```

### Key Components

**1. Orchestrator (Workflow)**
- Coordinates services
- Manages state
- Handles errors
- Implements compensation

**2. Services (Virtual Objects)**
- Domain logic
- State management
- Idempotent operations
- Status reporting

**3. Communication**
- Request-response calls
- Durable invocations
- Futures for parallelism

---

## 🔄 Service Communication Patterns

### Pattern 1: Sequential Calls

When steps depend on previous results:

```go
func (TripOrchestrator) BookTrip(
    ctx restate.WorkflowContext,
    req TripRequest,
) (TripResult, error) {
    // Step 1: Reserve flight
    flightRes, err := restate.Service[FlightReservation](
        ctx, "FlightService", "Reserve",
    ).Request(req.Flight)
    if err != nil {
        return TripResult{}, err
    }
    
    // Step 2: Reserve hotel (uses flight info)
    hotelRes, err := restate.Service[HotelReservation](
        ctx, "HotelService", "Reserve",
    ).Request(req.Hotel)
    if err != nil {
        // Compensate: cancel flight
        restate.Service[bool](ctx, "FlightService", "Cancel").
            Request(flightRes.ID)
        return TripResult{}, err
    }
    
    // Step 3: Charge payment
    total := flightRes.Price + hotelRes.Price
    payment, err := restate.Service[PaymentResult](
        ctx, "PaymentService", "Charge",
    ).Request(PaymentRequest{Amount: total})
    
    return TripResult{
        FlightID: flightRes.ID,
        HotelID: hotelRes.ID,
        PaymentID: payment.ID,
    }, nil
}
```

### Pattern 2: Parallel Calls (Futures)

When steps are independent:

```go
func (TripOrchestrator) SearchOptions(
    ctx restate.WorkflowContext,
    req SearchRequest,
) (SearchResult, error) {
    // Start both searches in parallel
    flightFuture := restate.Service[[]Flight](
        ctx, "FlightService", "Search",
    ).RequestFuture(req.FlightCriteria)
    
    hotelFuture := restate.Service[[]Hotel](
        ctx, "HotelService", "Search",
    ).RequestFuture(req.HotelCriteria)
    
    // Wait for both results
    flights, err1 := flightFuture.Response()
    hotels, err2 := hotelFuture.Response()
    
    if err1 != nil || err2 != nil {
        return SearchResult{}, fmt.Errorf("search failed")
    }
    
    return SearchResult{
        Flights: flights,
        Hotels: hotels,
    }, nil
}
```

### Pattern 3: Fan-Out / Fan-In

When calling multiple instances:

```go
func (FulfillmentOrchestrator) NotifyAllWarehouses(
    ctx restate.WorkflowContext,
    orderID string,
) error {
    warehouseIDs := []string{"wh-1", "wh-2", "wh-3"}
    
    // Fan-out: notify all warehouses in parallel
    var futures []restate.ResponseFuture[bool]
    for _, whID := range warehouseIDs {
        future := restate.Object[bool](
            ctx, "WarehouseService", whID, "CheckInventory",
        ).RequestFuture(orderID)
        futures = append(futures, future)
    }
    
    // Fan-in: collect all responses
    for _, fut := range futures {
        if hasStock, err := fut.Response(); err == nil && hasStock {
            return nil  // Found warehouse with stock!
        }
    }
    
    return fmt.Errorf("no warehouse has stock")
}
```

---

## 💡 Distributed Transaction Patterns

### Two-Phase Pattern (Reserve → Confirm)

Best for distributed transactions:

```go
func (TripOrchestrator) BookTrip(ctx restate.WorkflowContext, req TripRequest) (TripResult, error) {
    // PHASE 1: Reserve (tentative, can be cancelled)
    flightRes, _ := flightService.Reserve(req.Flight)  // Holds inventory
    hotelRes, _ := hotelService.Reserve(req.Hotel)      // Holds room
    
    // Critical point: Charge customer
    payment, err := paymentService.Charge(total)
    if err != nil {
        // Payment failed → Cancel reservations
        flightService.CancelReservation(flightRes.ID)
        hotelService.CancelReservation(hotelRes.ID)
        return TripResult{}, err
    }
    
    // PHASE 2: Confirm (permanent, cannot be cancelled)
    flightService.ConfirmReservation(flightRes.ID)     // Commits
    hotelService.ConfirmReservation(hotelRes.ID)       // Commits
    
    return TripResult{Success: true}, nil
}
```

**Key Points:**
- Reserve = temporary lock (can cancel)
- Confirm = permanent (cannot cancel)
- Payment is the commit point
- Cancel reservations if payment fails

### Saga Pattern (Forward Recovery)

For workflows where compensation is complex:

```go
func (OrderOrchestrator) ProcessOrder(ctx restate.WorkflowContext, order Order) error {
    // Track what succeeded for compensation
    var completed []string
    
    // Step 1: Reserve inventory
    if err := inventoryService.Reserve(order.Items); err != nil {
        return err
    }
    completed = append(completed, "inventory")
    
    // Step 2: Charge customer
    if err := paymentService.Charge(order.Total); err != nil {
        compensate(ctx, completed)  // Release inventory
        return err
    }
    completed = append(completed, "payment")
    
    // Step 3: Create shipment
    if err := shippingService.CreateLabel(order.Address); err != nil {
        compensate(ctx, completed)  // Refund + release inventory
        return err
    }
    
    return nil
}

func compensate(ctx restate.WorkflowContext, completed []string) {
    for i := len(completed) - 1; i >= 0; i-- {
        switch completed[i] {
        case "payment":
            paymentService.Refund()
        case "inventory":
            inventoryService.Release()
        }
    }
}
```

---

## 🛡️ Failure Handling Strategies

### Strategy 1: Immediate Compensation

Fail fast and clean up:

```go
flightRes, err := flightService.Reserve()
if err != nil {
    return err  // Nothing to compensate
}

hotelRes, err := hotelService.Reserve()
if err != nil {
    flightService.Cancel(flightRes.ID)  // Compensate immediately
    return err
}
```

### Strategy 2: Defer Compensation

Collect compensation actions:

```go
type CompensationLog struct {
    Actions []func() error
}

func (TripOrchestrator) BookTrip(ctx restate.WorkflowContext) error {
    var compensations CompensationLog
    
    flightRes, err := flightService.Reserve()
    compensations.Add(func() error {
        return flightService.Cancel(flightRes.ID)
    })
    
    hotelRes, err := hotelService.Reserve()
    compensations.Add(func() error {
        return hotelService.Cancel(hotelRes.ID)
    })
    
    if paymentFails {
        return compensations.ExecuteAll()  // Run all compensations
    }
}
```

### Strategy 3: Partial Success

Continue workflow despite non-critical failures:

```go
// Critical: must succeed
flightRes, err := flightService.Reserve()
if err != nil {
    return err  // Fail entire workflow
}

// Non-critical: can fail
notification, err := notificationService.SendEmail()
if err != nil {
    ctx.Log().Warn("Email failed", "error", err)
    // Continue anyway, email not critical
}
```

---

## 📊 State Management

### Workflow State

Track progress through workflow:

```go
type TripState struct {
    Status           string    // "reserving", "paying", "confirmed"
    FlightReservation string
    HotelReservation  string
    PaymentID        string
    CreatedAt        time.Time
}

func (TripOrchestrator) BookTrip(ctx restate.WorkflowContext) error {
    state := TripState{Status: "reserving"}
    restate.Set(ctx, "state", state)
    
    // Reserve flight
    flightRes, _ := flightService.Reserve()
    state.FlightReservation = flightRes.ID
    restate.Set(ctx, "state", state)
    
    // Reserve hotel
    hotelRes, _ := hotelService.Reserve()
    state.HotelReservation = hotelRes.ID
    state.Status = "paying"
    restate.Set(ctx, "state", state)
    
    // Process payment
    payment, _ := paymentService.Charge()
    state.PaymentID = payment.ID
    state.Status = "confirmed"
    restate.Set(ctx, "state", state)
}
```

### Query Workflow Status

Allow external queries:

```go
func (TripOrchestrator) GetStatus(
    ctx restate.WorkflowSharedContext,
    _ restate.Void,
) (TripState, error) {
    state, err := restate.Get[TripState](ctx, "state")
    return state, err
}

// Client queries progress
status := tripOrchestrator.GetStatus(tripID)
// Returns: {Status: "paying", FlightReservation: "FR-123", ...}
```

---

## ⚡ Performance Optimization

### Use Futures for Parallelism

```go
// ❌ SLOW: Sequential (100ms each = 300ms total)
flight := flightService.Reserve()  // 100ms
hotel := hotelService.Reserve()    // 100ms
car := carService.Reserve()        // 100ms

// ✅ FAST: Parallel (100ms total)
flightFut := flightService.ReserveFuture()
hotelFut := hotelService.ReserveFuture()
carFut := carService.ReserveFuture()

flight, _ := flightFut.Response()  // All run in parallel
hotel, _ := hotelFut.Response()
car, _ := carFut.Response()
// Total: 100ms (not 300ms!)
```

### Batch Operations

```go
// ❌ SLOW: Many small calls
for _, item := range items {
    inventoryService.Reserve(item)  // N calls
}

// ✅ FAST: Single batch call
inventoryService.ReserveBatch(items)  // 1 call
```

---

## ✅ Best Practices Summary

### DO's ✅

1. **Use central orchestrator for complex workflows**
2. **Implement two-phase operations (reserve → confirm)**
3. **Add compensation logic for failures**
4. **Use futures for independent operations**
5. **Track workflow state clearly**
6. **Make services idempotent**
7. **Handle partial failures gracefully**

### DON'Ts ❌

1. **Don't skip compensation logic**
2. **Don't make blocking sequential calls when parallel is possible**
3. **Don't scatter workflow logic across services**
4. **Don't forget to handle timeouts**
5. **Don't make services tightly coupled**

---

## 🚀 Next Steps

You now understand orchestration patterns!

👉 **Continue to [Hands-On Tutorial](./02-hands-on.md)**

Build a complete travel booking orchestrator!

---

**Questions?** Review this document or check the [module README](./README.md).
