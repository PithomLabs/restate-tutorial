# Hands-On: Building a Travel Booking Saga

> **Build a distributed transaction with automatic compensation**

## 🎯 What We're Building

A **travel booking saga** that coordinates:
- ✈️ Flight reservation
- 🏨 Hotel reservation  
- 🚗 Car rental reservation

If any step fails, we automatically compensate (cancel) all previous reservations.

## 📋 Prerequisites

- ✅ Restate server running (localhost:8080, 9080)
- ✅ Go 1.23+ installed
- ✅ Completed Module 05 (Workflows)

## 🏗️ Project Setup

### Step 1: Create Project Directory

```bash
mkdir -p ~/restate-tutorials/module06/code
cd ~/restate-tutorials/module06/code
```

### Step 2: Initialize Go Module

```bash
go mod init module06
go get github.com/restatedev/sdk-go@latest
```

### Step 3: Create File Structure

```bash
touch main.go types.go travel_saga.go supporting_services.go
```

## 📝 Implementation

### Step 1: Define Types (`types.go`)

```go
package main

import "time"

// Travel booking request
type TravelBooking struct {
    BookingID    string      `json:"bookingId"`
    CustomerID   string      `json:"customerId"`
    FlightInfo   FlightInfo  `json:"flightInfo"`
    HotelInfo    HotelInfo   `json:"hotelInfo"`
    CarInfo      CarInfo     `json:"carInfo"`
}

type FlightInfo struct {
    From         string    `json:"from"`
    To           string    `json:"to"`
    DepartDate   time.Time `json:"departDate"`
    ReturnDate   time.Time `json:"returnDate"`
    Passengers   int       `json:"passengers"`
}

type HotelInfo struct {
    Location     string    `json:"location"`
    CheckIn      time.Time `json:"checkIn"`
    CheckOut     time.Time `json:"checkOut"`
    Guests       int       `json:"guests"`
}

type CarInfo struct {
    Location     string    `json:"location"`
    PickupDate   time.Time `json:"pickupDate"`
    ReturnDate   time.Time `json:"returnDate"`
    CarType      string    `json:"carType"`
}

// Confirmation responses
type FlightConfirmation struct {
    ConfirmationCode string    `json:"confirmationCode"`
    FlightNumber     string    `json:"flightNumber"`
    BookedAt         time.Time `json:"bookedAt"`
}

type HotelConfirmation struct {
    ConfirmationCode string    `json:"confirmationCode"`
    HotelName        string    `json:"hotelName"`
    BookedAt         time.Time `json:"bookedAt"`
}

type CarConfirmation struct {
    ConfirmationCode string    `json:"confirmationCode"`
    CarModel         string    `json:"carModel"`
    BookedAt         time.Time `json:"bookedAt"`
}

// Final saga result
type BookingResult struct {
    Status               string `json:"status"` // "confirmed", "failed"
    FlightConfirmation   string `json:"flightConfirmation,omitempty"`
    HotelConfirmation    string `json:"hotelConfirmation,omitempty"`
    CarConfirmation      string `json:"carConfirmation,omitempty"`
    FailureReason        string `json:"failureReason,omitempty"`
}
```

### Step 2: Implement Supporting Services (`supporting_services.go`)

These simulate external services (flight, hotel, car rental).

```go
package main

import (
    "fmt"
    "time"
    
    restate "github.com/restatedev/sdk-go"
)

// FlightService simulates airline booking API
type FlightService struct{}

func (FlightService) Reserve(
    ctx restate.Context,
    info FlightInfo,
) (FlightConfirmation, error) {
    // Simulate API call
    _, err := restate.Run(ctx, func(ctx restate.RunContext) (bool, error) {
        ctx.Log().Info("Calling flight booking API",
            "from", info.From,
            "to", info.To,
            "passengers", info.Passengers)
        
        time.Sleep(100 * time.Millisecond) // Simulate latency
        
        // 10% failure rate
        if restate.Rand(ctx).Float64() < 0.1 {
            return false, fmt.Errorf("flight unavailable")
        }
        
        return true, nil
    })
    
    if err != nil {
        return FlightConfirmation{}, err
    }
    
    return FlightConfirmation{
        ConfirmationCode: fmt.Sprintf("FL-%s", restate.UUID(ctx).String()[:8]),
        FlightNumber:     "AA123",
        BookedAt:         time.Now(),
    }, nil
}

func (FlightService) Cancel(
    ctx restate.Context,
    confirmationCode string,
) error {
    ctx.Log().Info("Cancelling flight", "code", confirmationCode)
    
    _, err := restate.Run(ctx, func(ctx restate.RunContext) (bool, error) {
        // Idempotent cancellation
        ctx.Log().Info("Flight cancelled successfully")
        return true, nil
    })
    
    return err
}

// HotelService simulates hotel booking API
type HotelService struct{}

func (HotelService) Reserve(
    ctx restate.Context,
    info HotelInfo,
) (HotelConfirmation, error) {
    _, err := restate.Run(ctx, func(ctx restate.RunContext) (bool, error) {
        ctx.Log().Info("Calling hotel booking API",
            "location", info.Location,
            "guests", info.Guests)
        
        time.Sleep(100 * time.Millisecond)
        
        // 10% failure rate
        if restate.Rand(ctx).Float64() < 0.1 {
            return false, fmt.Errorf("hotel unavailable")
        }
        
        return true, nil
    })
    
    if err != nil {
        return HotelConfirmation{}, err
    }
    
    return HotelConfirmation{
        ConfirmationCode: fmt.Sprintf("HT-%s", restate.UUID(ctx).String()[:8]),
        HotelName:        "Grand Hotel",
        BookedAt:         time.Now(),
    }, nil
}

func (HotelService) Cancel(
    ctx restate.Context,
    confirmationCode string,
) error {
    ctx.Log().Info("Cancelling hotel", "code", confirmationCode)
    
    _, err := restate.Run(ctx, func(ctx restate.RunContext) (bool, error) {
        ctx.Log().Info("Hotel cancelled successfully")
        return true, nil
    })
    
    return err
}

// CarService simulates car rental API
type CarService struct{}

func (CarService) Reserve(
    ctx restate.Context,
    info CarInfo,
) (CarConfirmation, error) {
    _, err := restate.Run(ctx, func(ctx restate.RunContext) (bool, error) {
        ctx.Log().Info("Calling car rental API",
            "location", info.Location,
            "type", info.CarType)
        
        time.Sleep(100 * time.Millisecond)
        
        // 10% failure rate
        if restate.Rand(ctx).Float64() < 0.1 {
            return false, fmt.Errorf("car unavailable")
        }
        
        return true, nil
    })
    
    if err != nil {
        return CarConfirmation{}, err
    }
    
    return CarConfirmation{
        ConfirmationCode: fmt.Sprintf("CR-%s", restate.UUID(ctx).String()[:8]),
        CarModel:         info.CarType,
        BookedAt:         time.Now(),
    }, nil
}

func (CarService) Cancel(
    ctx restate.Context,
    confirmationCode string,
) error {
    ctx.Log().Info("Cancelling car rental", "code", confirmationCode)
    
    _, err := restate.Run(ctx, func(ctx restate.RunContext) (bool, error) {
        ctx.Log().Info("Car rental cancelled successfully")
        return true, nil
    })
    
    return err
}
```

### Step 3: Implement Travel Saga (`travel_saga.go`)

The main saga workflow with compensation logic.

```go
package main

import (
    "fmt"
    
    restate "github.com/restatedev/sdk-go"
)

type TravelSaga struct{}

// Run executes the travel booking saga
func (TravelSaga) Run(
    ctx restate.WorkflowContext,
    booking TravelBooking,
) (BookingResult, error) {
    bookingID := restate.Key(ctx)
    
    ctx.Log().Info("Starting travel booking saga",
        "bookingId", bookingID,
        "customer", booking.CustomerID)
    
    // Initialize services
    flightSvc := FlightService{}
    hotelSvc := HotelService{}
    carSvc := CarService{}
    
    // Step 1: Reserve Flight
    ctx.Log().Info("Step 1: Reserving flight")
    flightConf, err := flightSvc.Reserve(ctx, booking.FlightInfo)
    if err != nil {
        ctx.Log().Error("Flight reservation failed", "error", err)
        return BookingResult{
            Status:        "failed",
            FailureReason: fmt.Sprintf("flight: %v", err),
        }, nil
    }
    ctx.Log().Info("Flight reserved", "code", flightConf.ConfirmationCode)
    
    // Step 2: Reserve Hotel
    ctx.Log().Info("Step 2: Reserving hotel")
    hotelConf, err := hotelSvc.Reserve(ctx, booking.HotelInfo)
    if err != nil {
        ctx.Log().Error("Hotel reservation failed", "error", err)
        
        // COMPENSATE: Cancel flight
        ctx.Log().Warn("Compensating: cancelling flight")
        flightSvc.Cancel(ctx, flightConf.ConfirmationCode)
        
        return BookingResult{
            Status:        "failed",
            FailureReason: fmt.Sprintf("hotel: %v", err),
        }, nil
    }
    ctx.Log().Info("Hotel reserved", "code", hotelConf.ConfirmationCode)
    
    // Step 3: Reserve Car
    ctx.Log().Info("Step 3: Reserving car")
    carConf, err := carSvc.Reserve(ctx, booking.CarInfo)
    if err != nil {
        ctx.Log().Error("Car reservation failed", "error", err)
        
        // COMPENSATE: Cancel hotel and flight
        ctx.Log().Warn("Compensating: cancelling hotel and flight")
        hotelSvc.Cancel(ctx, hotelConf.ConfirmationCode)
        flightSvc.Cancel(ctx, flightConf.ConfirmationCode)
        
        return BookingResult{
            Status:        "failed",
            FailureReason: fmt.Sprintf("car: %v", err),
        }, nil
    }
    ctx.Log().Info("Car reserved", "code", carConf.ConfirmationCode)
    
    // All steps succeeded!
    ctx.Log().Info("Travel booking completed successfully",
        "flight", flightConf.ConfirmationCode,
        "hotel", hotelConf.ConfirmationCode,
        "car", carConf.ConfirmationCode)
    
    return BookingResult{
        Status:             "confirmed",
        FlightConfirmation: flightConf.ConfirmationCode,
        HotelConfirmation:  hotelConf.ConfirmationCode,
        CarConfirmation:    carConf.ConfirmationCode,
    }, nil
}
```

### Step 4: Main Server (`main.go`)

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    restate "github.com/restatedev/sdk-go"
    "github.com/restatedev/sdk-go/server"
)

func main() {
    restateServer := server.NewRestate()
    
    // Register saga
    if err := restateServer.Bind(restate.Reflect(TravelSaga{})); err != nil {
        log.Fatal("Failed to bind TravelSaga:", err)
    }
    
    // Register services (for direct testing)
    if err := restateServer.Bind(restate.Reflect(FlightService{})); err != nil {
        log.Fatal("Failed to bind FlightService:", err)
    }
    
    if err := restateServer.Bind(restate.Reflect(HotelService{})); err != nil {
        log.Fatal("Failed to bind HotelService:", err)
    }
    
    if err := restateServer.Bind(restate.Reflect(CarService{})); err != nil {
        log.Fatal("Failed to bind CarService:", err)
    }
    
    fmt.Println("🌍 Starting Travel Booking Saga Service on :9090...")
    fmt.Println("📋 Registered:")
    fmt.Println("  - TravelSaga (workflow)")
    fmt.Println("  - FlightService")
    fmt.Println("  - HotelService")
    fmt.Println("  - CarService")
    fmt.Println("")
    fmt.Println("✓ Ready to accept bookings")
    
    if err := restateServer.Start(context.Background(), ":9090"); err != nil {
        log.Fatal("Server error:", err)
    }
}
```

## 🚀 Running the Saga

### Step 1: Build and Run

```bash
go mod tidy
go build -o travel-saga
./travel-saga
```

### Step 2: Register with Restate

```bash
curl -X POST http://localhost:8080/deployments \
  -H 'Content-Type: application/json' \
  -d '{"uri": "http://localhost:9090"}'
```

### Step 3: Test Successful Booking

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

**Expected Response:**
```json
{
  "status": "confirmed",
  "flightConfirmation": "FL-abc12345",
  "hotelConfirmation": "HT-def67890",
  "carConfirmation": "CR-ghi13579"
}
```

### Step 4: Run Multiple Times to Trigger Failures

Due to the 10% failure rate in each service, run the saga multiple times:

```bash
# Run 10 bookings - some will fail and compensate
for i in {1..10}; do
  echo "Booking attempt $i"
  curl -s -X POST http://localhost:9080/TravelSaga/booking-$i/Run \
    -H 'Content-Type: application/json' \
    -d '{
      "bookingId": "booking-'$i'",
      "customerId": "customer-123",
      "flightInfo": {"from": "NYC", "to": "LAX", "departDate": "2024-06-01T10:00:00Z", "returnDate": "2024-06-07T18:00:00Z", "passengers": 2},
      "hotelInfo": {"location": "LAX", "checkIn": "2024-06-01T15:00:00Z", "checkOut": "2024-06-07T11:00:00Z", "guests": 2},
      "carInfo": {"location": "LAX", "pickupDate": "2024-06-01T10:00:00Z", "returnDate": "2024-06-07T18:00:00Z", "carType": "SUV"}
    }' | jq '{status, failureReason}'
  sleep 1
done
```

Watch the service logs to see compensation in action!

## 📊 Understanding the Flow

### Successful Saga

```
1. Reserve Flight  ✅ → FL-abc123
2. Reserve Hotel   ✅ → HT-def456
3. Reserve Car     ✅ → CR-ghi789
Result: CONFIRMED
```

### Failed at Hotel (Compensation)

```
1. Reserve Flight  ✅ → FL-abc123
2. Reserve Hotel   ❌ → Error!
   ↓
   Compensate: Cancel FL-abc123
Result: FAILED
```

### Failed at Car (Compensation)

```
1. Reserve Flight  ✅ → FL-abc123
2. Reserve Hotel   ✅ → HT-def456
3. Reserve Car     ❌ → Error!
   ↓
   Compensate: Cancel HT-def456
   Compensate: Cancel FL-abc123
Result: FAILED
```

## 🎓 Key Concepts Demonstrated

### 1. Sequential Transaction Steps

Each step depends on previous success:
```go
flight, err := reserveFlight(ctx)
if err != nil { return fail }

hotel, err := reserveHotel(ctx)
if err != nil {
    cancelFlight(ctx, flight) // Compensate
    return fail
}
```

### 2. Compensation in Reverse Order

```go
// Failed at car - compensate hotel, then flight
carSvc.Reserve() // ❌ FAILS
hotelSvc.Cancel()  // ← Compensate most recent
flightSvc.Cancel() // ← Compensate first
```

### 3. Durable Execution

Each step is journaled - if service crashes mid-saga, Restate automatically resumes!

### 4. Idempotent Operations

Cancellations are idempotent - safe to call multiple times.

## ✅ Success Criteria

Your saga should:
- ✅ Complete successfully when all services respond
- ✅ Cancel flight if hotel fails
- ✅ Cancel flight+hotel if car fails
- ✅ Log all operations clearly
- ✅ Survive service restarts

## 🎯 Next Steps

Excellent! Your distributed transaction is working!

👉 **Continue to [Validation](./03-validation.md)**

Test compensation thoroughly!

---

**Questions?** Review [concepts](./01-concepts.md) or check the [module README](./README.md).
