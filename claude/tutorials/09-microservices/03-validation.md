# Validation: Testing Microservices Orchestration

> **Verify your travel booking orchestrator**

## 🎯 Validation Goals

- ✅ Test end-to-end booking flow
- ✅ Verify compensation logic
- ✅ Test failure scenarios
- ✅ Confirm idempotency
- ✅ Validate service coordination

## 🧪 Test Scenarios

### Scenario 1: Successful Booking

**Test:** Complete trip booking

```bash
curl -X POST http://localhost:8080/TripOrchestrator/trip-success/BookTrip \
  -H 'Content-Type: application/json' \
  -d '{
    "customerId": "cust-001",
    "customerEmail": "test@example.com",
    "flight": {"from": "SFO", "to": "NYC", "date": "2024-12-15", "passengers": 2},
    "hotel": {"city": "NYC", "checkIn": "2024-12-15", "checkOut": "2024-12-18", "guests": 2}
  }'
```

**Expected:**
- ✅ Flight reserved and confirmed
- ✅ Hotel reserved and confirmed
- ✅ Payment processed
- ✅ Confirmation email sent
- ✅ Status = "confirmed"

### Scenario 2: Idempotent Booking

**Test:** Duplicate booking request

```bash
# First request
curl -X POST http://localhost:8080/TripOrchestrator/trip-idem/BookTrip -d '{...}'

# Second request (exact same)
curl -X POST http://localhost:8080/TripOrchestrator/trip-idem/BookTrip -d '{...}'
```

**Expected:**
- ✅ Same result from both requests
- ✅ Only one flight reservation
- ✅ Only one hotel reservation
- ✅ Only one payment

### Scenario 3: Get Booking Status

**Test:** Query trip status

```bash
curl -X POST http://localhost:8080/TripOrchestrator/trip-success/GetBooking \
  -H 'Content-Type: application/json' \
  -d '{}'
```

**Expected:** Full booking details with all reservations

### Scenario 4: Service Verification

Check individual services were called:

```bash
# Check flight reservation
curl -X POST http://localhost:8080/FlightService/trip-success-flight/Reserve \
  -H 'Content-Type: application/json' \
  -d '{"from": "SFO", "to": "NYC", "date": "2024-12-15", "passengers": 2}'

# Check hotel reservation  
curl -X POST http://localhost:8080/HotelService/trip-success-hotel/Reserve \
  -H 'Content-Type: application/json' \
  -d '{"city": "NYC", "checkIn": "2024-12-15", "checkOut": "2024-12-18", "guests": 2}'
```

**Expected:** Both show "confirmed" status

## ✅ Validation Checklist

### Orchestration
- [ ] Trip orchestrator coordinates all services
- [ ] Services called in correct order
- [ ] State tracked through workflow
- [ ] Logs show clear workflow progression

### Compensation
- [ ] Reservations cancelled on failure
- [ ] No orphaned resources
- [ ] Clean rollback

### Idempotency
- [ ] Duplicate requests handled
- [ ] No double bookings
- [ ] Same result on retry

### Service Integration
- [ ] All services registered
- [ ] Inter-service communication works
- [ ] State persisted correctly

## 🎓 Success Criteria

- ✅ All scenarios pass
- ✅ Compensation works
- ✅ No resource leaks
- ✅ Idempotent operations
- ✅ Clear workflow visibility

## 🚀 Next Steps

👉 **Continue to [Exercises](./04-exercises.md)**
