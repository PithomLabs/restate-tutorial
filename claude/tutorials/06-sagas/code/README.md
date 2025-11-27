# Module 06 - Travel Booking Saga with Compensation

This directory contains a complete travel booking saga demonstrating distributed transactions with automatic compensation.

## 📂 Files

- `main.go` - Server initialization
- `types.go` - Data structures
- `travel_saga.go` - Saga workflow with compensation
- `supporting_services.go` - Flight, hotel, car services
- `go.mod` - Dependencies

## 🚀 Quick Start

```bash
# Build
go mod tidy
go build -o travel-saga

# Run
./travel-saga
```

Service starts on port 9090.

## 📋 Register with Restate

```bash
curl -X POST http://localhost:8080/deployments \
  -H 'Content-Type: application/json' \
  -d '{"uri": "http://localhost:9090"}'
```

## 🧪 Test

### Successful Booking

```bash
curl -X POST http://localhost:9080/TravelSaga/booking-001/Run \
  -H 'Content-Type: application/json' \
  -d '{
    "bookingId": "booking-001",
    "customerId": "customer-123",
    "flightInfo": {
      "from": "NYC",
      "to": "LAX",
      "departDate": "2024-06-01T10:00:00Z",
      "returnDate": "2024-06-07T18:00:00Z",
      "passengers": 2
    },
    "hotelInfo": {
      "location": "LAX",
      "checkIn": "2024-06-01T15:00:00Z",
      "checkOut": "2024-06-07T11:00:00Z",
      "guests": 2
    },
    "carInfo": {
      "location": "LAX",
      "pickupDate": "2024-06-01T10:00:00Z",
      "returnDate": "2024-06-07T18:00:00Z",
      "carType": "SUV"
    }
  }'
```

### Test Compensation

Run multiple times to see compensation in action (10% random failure rate):

```bash
for i in {1..10}; do
  echo "Booking $i:"
  curl -s -X POST http://localhost:9080/TravelSaga/booking-$i/Run \
    -H 'Content-Type: application/json' \
    -d '{"bookingId": "booking-'$i'", "customerId": "cust-123", "flightInfo": {"from": "NYC", "to": "LAX", "departDate": "2024-06-01T10:00:00Z", "returnDate": "2024-06-07T18:00:00Z", "passengers": 2}, "hotelInfo": {"location": "LAX", "checkIn": "2024-06-01T15:00:00Z", "checkOut": "2024-06-07T11:00:00Z", "guests": 2}, "carInfo": {"location": "LAX", "pickupDate": "2024-06-01T10:00:00Z", "returnDate": "2024-06-07T18:00:00Z", "carType": "SUV"}}' | jq '{status, failureReason}'
done
```

## 🎓 Key Concepts Demonstrated

### Saga Structure

```go
func (TravelSaga) Run(ctx restate.WorkflowContext, booking TravelBooking) (BookingResult, error) {
    // Step 1: Reserve flight
    flightConf, err := flightSvc.Reserve(ctx, booking.FlightInfo)
    if err != nil {
        return BookingResult{Status: "failed"}, nil
    }
    
    // Step 2: Reserve hotel
    hotelConf, err := hotelSvc.Reserve(ctx, booking.HotelInfo)
    if err != nil {
        // COMPENSATE: Cancel flight
        flightSvc.Cancel(ctx, flightConf.ConfirmationCode)
        return BookingResult{Status: "failed"}, nil
    }
    
    // Step 3: Reserve car
    carConf, err := carSvc.Reserve(ctx, booking.CarInfo)
    if err != nil {
        // COMPENSATE: Cancel hotel and flight
        hotelSvc.Cancel(ctx, hotelConf.ConfirmationCode)
        flightSvc.Cancel(ctx, flightConf.ConfirmationCode)
        return BookingResult{Status: "failed"}, nil
    }
    
    // All succeeded!
    return BookingResult{Status: "confirmed"}, nil
}
```

### Compensation Pattern

```go
// Each service has idempotent cancellation
func (FlightService) Cancel(ctx restate.Context, confirmationCode string) error {
    _, err := restate.Run(ctx, func(ctx restate.RunContext) (bool, error) {
        // Idempotent - safe to call multiple times
        return true, nil
    })
    return err
}
```

### Durable Execution

Each step is journaled - if the service crashes:
- Completed steps are NOT re-executed
- Saga resumes from the last checkpoint
- Compensations run correctly

## 💡 Features

1. **Multi-Service Coordination** - Flight, hotel, car
2. **Automatic Compensation** - Rollback on failure
3. **Reverse Order Compensation** - Cancel most recent first
4. **Idempotent Operations** - Safe to retry
5. **Durable State** - Survives failures
6. **Random Failures** - 10% failure rate for testing

## 📊 Saga Flow

### Success Flow
```
1. Reserve Flight  ✅ → FL-abc123
2. Reserve Hotel   ✅ → HT-def456
3. Reserve Car     ✅ → CR-ghi789
Result: CONFIRMED
```

### Failure at Hotel
```
1. Reserve Flight  ✅ → FL-abc123
2. Reserve Hotel   ❌ → Error!
   ↓ Compensate
   Cancel FL-abc123
Result: FAILED
```

### Failure at Car
```
1. Reserve Flight  ✅ → FL-abc123
2. Reserve Hotel   ✅ → HT-def456
3. Reserve Car     ❌ → Error!
   ↓ Compensate
   Cancel HT-def456
   Cancel FL-abc123
Result: FAILED
```

## 🎯 Next Steps

- Complete [validation tests](../03-validation.md)
- Try [exercises](../04-exercises.md)
- Build your own sagas

---

**Questions?** See the main [hands-on tutorial](../02-hands-on.md)!
