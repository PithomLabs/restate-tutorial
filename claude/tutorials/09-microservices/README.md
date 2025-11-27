# Module 09: Microservices Orchestration

> **Master coordinating distributed microservices with Restate**

## 🎯 Learning Objectives

By completing this module, you will:
- ✅ Coordinate multiple microservices in workflows
- ✅ Implement service-to-service communication patterns
- ✅ Build complex distributed transactions
- ✅ Handle cross-service failures and compensation
- ✅ Design resilient orchestration patterns
- ✅ Manage long-running distributed processes

## 📚 Module Structure

### 1. [Concepts](./01-concepts.md) (~35 min)
Learn orchestration patterns and architecture:
- Orchestration vs choreography
- Service communication patterns
- Distributed transaction coordination
- Saga patterns (orchestration-based)
- Failure handling across services
- State management in workflows

### 2. [Hands-On Tutorial](./02-hands-on.md) (~60 min)
Build a **Travel Booking System**:
- Flight reservation service
- Hotel booking service
- Payment processing service
- Notification service
- Trip orchestrator coordinating all services
- Compensation logic for failures

### 3. [Validation](./03-validation.md) (~40 min)
Test your orchestration:
- End-to-end booking flow
- Partial failure scenarios
- Compensation logic verification
- Concurrent booking handling
- Performance testing

### 4. [Exercises](./04-exercises.md) (~70 min)
Practice with real-world scenarios:
- E-commerce order fulfillment
- Bank transfer orchestration
- Food delivery coordination
- CI/CD pipeline orchestration
- Multi-tenant workflow management

## 🎓 Prerequisites

Before starting this module:
- ✅ Completed Module 01 (Foundation)
- ✅ Completed Module 06 (Sagas)
- ✅ Completed Idempotency module
- ✅ Completed External Integration module
- ✅ Understanding of microservices architecture

## 💡 Why Microservices Orchestration Matters

### The Challenge

Distributed microservices need coordination:

```
OrderService
    ↓
InventoryService → Manual coordination
    ↓
PaymentService → Scattered logic
    ↓
ShippingService → No central view
    ↓
NotificationService → Hard to debug

Problems:
- No single point of control
- Distributed state
- Unclear failure handling
- Difficult testing
- No workflow visibility
```

### The Restate Solution

```
OrderOrchestrator (Restate Workflow)
    ├→ InventoryService.Reserve()
    ├→ PaymentService.Charge()
    ├→ ShippingService.CreateLabel()
    └→ NotificationService.Send()

Benefits:
- Central coordination
- Durable state
- Automatic retries
- Clear compensation
- Full visibility
- Easy testing
```

## 🏗️ What You'll Build

A **Travel Booking Orchestrator** coordinating:

### Microservices
- ✈️ **FlightService** - Search and reserve flights
- 🏨 **HotelService** - Search and book hotels
- 💳 **PaymentService** - Process payments
- 📧 **NotificationService** - Email confirmations
- 🎫 **TripOrchestrator** - Coordinate everything

### Features
- Search flights and hotels
- Reserve both atomically
- Process payment
- Send confirmations
- Automatic compensation on failure
- Idempotent operations

### Workflow
```
TripOrchestrator
    1. Reserve flight (FlightService)
    2. Reserve hotel (HotelService)
    3. Charge customer (PaymentService)
    4. Confirm reservations (Flight + Hotel)
    5. Send confirmation (NotificationService)
    
If any step fails → Compensate:
    - Release flight reservation
    - Release hotel reservation
    - Refund payment (if charged)
```

## 📊 Module Outline

```
09-microservices/
├── README.md                    # This file
├── 01-concepts.md              # Orchestration patterns
├── 02-hands-on.md              # Travel booking system
├── 03-validation.md            # Testing guide
├── 04-exercises.md             # Practice problems
├── code/                       # Working implementation
│   ├── main.go
│   ├── types.go
│   ├── flight_service.go       # Flight reservations
│   ├── hotel_service.go        # Hotel bookings
│   ├── payment_service.go      # Payment processing
│   ├── notification_service.go # Email notifications
│   ├── trip_orchestrator.go    # Main orchestration
│   ├── go.mod
│   └── README.md
└── solutions/                  # Exercise solutions
    ├── order_fulfillment.go
    ├── bank_transfer.go
    └── README.md
```

## 🎯 Key Concepts Covered

### 1. Orchestration Patterns
- Central coordinator (orchestrator)
- Service-to-service calls
- Workflow state management
- Long-running processes

### 2. Distributed Transactions
- Multi-service coordination
- Two-phase operations (reserve → confirm)
- Atomic commits across services
- Compensation logic

### 3. Failure Handling
- Partial failure detection
- Automatic compensation
- Retry strategies
- Circuit breakers

### 4. State Management
- Workflow state tracking
- Service call results
- Compensation history
- Audit trails

## 🚀 Quick Start

### 1. Read Concepts
```bash
less 01-concepts.md
```

### 2. Run Services

```bash
cd code/
go mod download

# Start all services
go run .
```

### 3. Book a Trip

```bash
curl -X POST http://localhost:8080/TripOrchestrator/trip-001/BookTrip \
  -H 'Content-Type: application/json' \
  -d '{
    "customerId": "customer-123",
    "customerEmail": "alice@example.com",
    "flight": {
      "from": "SFO",
      "to": "JFK",
      "date": "2024-12-15",
      "passengers": 2
    },
    "hotel": {
      "city": "New York",
      "checkIn": "2024-12-15",
      "checkOut": "2024-12-18",
      "guests": 2
    }
  }'
```

## ⚠️ Common Pitfalls

### Anti-Pattern 1: No Central Coordinator

```go
// ❌ BAD - Each service calls the next
func OrderService(ctx restate.ObjectContext, order Order) {
    inventoryService.Reserve()
    // Who coordinates if this fails?
}

func InventoryService(ctx restate.ObjectContext) {
    paymentService.Charge()
    // No central view of the workflow!
}
```

### Anti-Pattern 2: No Compensation Logic

```go
// ❌ BAD - Resources not released on failure
flight := flightService.Reserve()
hotel := hotelService.Reserve()
paymentService.Charge()  // Fails!
// Flight and hotel reservations remain locked! 🔒
```

### Anti-Pattern 3: Blocking Calls Without Futures

```go
// ❌ BAD - Sequential when could be parallel
flightFuture := flightService.Search()   // Could be parallel
hotelFuture := hotelService.Search()     // Could be parallel
flight := flightFuture.Wait()            // Waited sequentially!
hotel := hotelFuture.Wait()
```

## ✅ Best Practices

### 1. Use Central Orchestrator

```go
// ✅ GOOD - Orchestrator coordinates everything
func (TripOrchestrator) BookTrip(ctx restate.WorkflowContext, req TripRequest) {
    flightRes := flightService.Reserve()
    hotelRes := hotelService.Reserve()
    payment := paymentService.Charge()
    // Clear workflow, easy to debug
}
```

### 2. Implement Compensation

```go
// ✅ GOOD - Clean up on failure
flightRes, _ := flightService.Reserve()
hotelRes, err := hotelService.Reserve()
if err != nil {
    // Compensate: release flight
    flightService.CancelReservation(flightRes.ID)
    return err
}
```

### 3. Use Futures for Parallelism

```go
// ✅ GOOD - Parallel execution
flightFuture := flightService.SearchFuture()
hotelFuture := hotelService.SearchFuture()

// Both run in parallel
flight, _ := flightFuture.Response()
hotel, _ := hotelFuture.Response()
```

### 4. Two-Phase Operations

```go
// ✅ GOOD - Reserve then confirm pattern
// Phase 1: Reserve (can be cancelled)
flightRes := flightService.Reserve()
hotelRes := hotelService.Reserve()

// Phase 2: Confirm (permanent)
if paymentSucceeds {
    flightService.Confirm(flightRes.ID)
    hotelService.Confirm(hotelRes.ID)
}
```

## 🔗 Related Modules

- **Module 06: Sagas** - Distributed transaction patterns
- **Idempotency Module** - Safe retries
- **External Integration** - API integration patterns
- **Module 10: Observability** - Monitoring workflows

## 📈 Success Criteria

You've mastered this module when you can:
- [x] Design orchestration workflows
- [x] Coordinate multiple services
- [x] Implement compensation logic
- [x] Handle distributed failures
- [x] Use futures for parallelism
- [x] Build resilient microservices systems

## 🎓 Learning Path

**Current Module:** Microservices Orchestration  
**Previous:** [External Integration](../08-external-integration/README.md)  
**Next:** [Module 10 - Observability](../10-observability/README.md)

---

## 🚀 Let's Get Started!

Ready to orchestrate microservices?

👉 **Start with [Concepts](./01-concepts.md)** to learn orchestration patterns!

---

**Questions?** Review [previous modules](../README.md) or check the [Sagas module](../06-sagas/README.md).
