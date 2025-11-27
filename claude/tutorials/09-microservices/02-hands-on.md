# Hands-On: Travel Booking Orchestrator

> **Build a microservices orchestration system**

## 🎯 What We're Building

A **Travel Booking System** with distributed microservices:
- ✈️ **FlightService** - Flight reservations
- 🏨 **HotelService** - Hotel bookings
- 💳 **PaymentService** - Payment processing
- 📧 **NotificationService** - Email confirmations
- 🎫 **TripOrchestrator** - Coordinates all services

**Workflow:**
1. Reserve flight (tentative)
2. Reserve hotel (tentative)
3. Charge customer
4. Confirm flight + hotel (permanent)
5. Send confirmation email

**Compensation:** If any step fails, release all reservations.

## 📋 Prerequisites

- ✅ Go 1.23+ installed
- ✅ Completed previous modules
- ✅ Docker for Restate server

## 🏗️ Project Setup

### Step 1: Create Project

```bash
mkdir -p ~/restate-tutorials/microservices/code
cd ~/restate-tutorials/microservices/code
go mod init microservices
go get github.com/restatedev/sdk-go@latest
```

## 📝 Implementation

### Step 1: Define Types (`types.go`)

```go
package main

import "time"

// Trip booking request
type TripRequest struct {
	CustomerID    string        `json:"customerId"`
	CustomerEmail string        `json:"customerEmail"`
	Flight        FlightRequest `json:"flight"`
	Hotel         HotelRequest  `json:"hotel"`
}

type FlightRequest struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Date       string `json:"date"`
	Passengers int    `json:"passengers"`
}

type HotelRequest struct {
	City     string `json:"city"`
	CheckIn  string `json:"checkIn"`
	CheckOut string `json:"checkOut"`
	Guests   int    `json:"guests"`
}

// Service responses
type FlightReservation struct {
	ID     string  `json:"id"`
	From   string  `json:"from"`
	To     string  `json:"to"`
	Price  float64 `json:"price"`
	Status string  `json:"status"` // "reserved", "confirmed", "cancelled"
}

type HotelReservation struct {
	ID       string  `json:"id"`
	Hotel    string  `json:"hotel"`
	City     string  `json:"city"`
	Price    float64 `json:"price"`
	Status   string  `json:"status"`
}

type PaymentResult struct {
	ID     string  `json:"id"`
	Amount float64 `json:"amount"`
	Status string  `json:"status"` // "succeeded", "failed"
}

// Trip result
type TripBooking struct {
	TripID            string             `json:"tripId"`
	Flight            FlightReservation  `json:"flight"`
	Hotel             HotelReservation   `json:"hotel"`
	Payment           PaymentResult      `json:"payment"`
	Status            string             `json:"status"`
	ConfirmationEmail bool               `json:"confirmationEmail"`
	CreatedAt         time.Time          `json:"createdAt"`
}
```

### Step 2: Flight Service (`flight_service.go`)

```go
package main

import (
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
)

type FlightService struct{}

// Reserve creates a tentative flight reservation
func (FlightService) Reserve(
	ctx restate.ObjectContext,
	req FlightRequest,
) (FlightReservation, error) {
	reservationID := restate.Key(ctx)

	ctx.Log().Info("Reserving flight",
		"id", reservationID,
		"from", req.From,
		"to", req.To)

	// Check existing reservation
	existing, _ := restate.Get[*FlightReservation](ctx, "reservation")
	if existing != nil {
		return *existing, nil
	}

	// Create reservation (tentative)
	reservation := FlightReservation{
		ID:     reservationID,
		From:   req.From,
		To:     req.To,
		Price:  calculateFlightPrice(req),
		Status: "reserved",
	}

	restate.Set(ctx, "reservation", reservation)
	return reservation, nil
}

// Confirm makes the reservation permanent
func (FlightService) Confirm(
	ctx restate.ObjectContext,
	_ restate.Void,
) (bool, error) {
	reservationID := restate.Key(ctx)

	reservation, err := restate.Get[FlightReservation](ctx, "reservation")
	if err != nil {
		return false, fmt.Errorf("reservation not found")
	}

	if reservation.Status != "reserved" {
		return false, fmt.Errorf("cannot confirm %s reservation", reservation.Status)
	}

	reservation.Status = "confirmed"
	restate.Set(ctx, "reservation", reservation)

	ctx.Log().Info("Flight confirmed", "id", reservationID)
	return true, nil
}

// Cancel releases the reservation
func (FlightService) Cancel(
	ctx restate.ObjectContext,
	_ restate.Void,
) (bool, error) {
	reservationID := restate.Key(ctx)

	reservation, err := restate.Get[FlightReservation](ctx, "reservation")
	if err != nil {
		return false, nil  // Already cancelled or doesn't exist
	}

	reservation.Status = "cancelled"
	restate.Set(ctx, "reservation", reservation)

	ctx.Log().Info("Flight cancelled", "id", reservationID)
	return true, nil
}

func calculateFlightPrice(req FlightRequest) float64 {
	basePrice := 300.0
	return basePrice * float64(req.Passengers)
}
```

### Step 3: Hotel Service (`hotel_service.go`)

```go
package main

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
)

type HotelService struct{}

// Reserve creates a tentative hotel reservation
func (HotelService) Reserve(
	ctx restate.ObjectContext,
	req HotelRequest,
) (HotelReservation, error) {
	reservationID := restate.Key(ctx)

	ctx.Log().Info("Reserving hotel",
		"id", reservationID,
		"city", req.City)

	// Check existing
	existing, _ := restate.Get[*HotelReservation](ctx, "reservation")
	if existing != nil {
		return *existing, nil
	}

	// Create reservation
	reservation := HotelReservation{
		ID:     reservationID,
		Hotel:  fmt.Sprintf("Hotel %s", req.City),
		City:   req.City,
		Price:  calculateHotelPrice(req),
		Status: "reserved",
	}

	restate.Set(ctx, "reservation", reservation)
	return reservation, nil
}

// Confirm makes the reservation permanent
func (HotelService) Confirm(
	ctx restate.ObjectContext,
	_ restate.Void,
) (bool, error) {
	reservationID := restate.Key(ctx)

	reservation, err := restate.Get[HotelReservation](ctx, "reservation")
	if err != nil {
		return false, fmt.Errorf("reservation not found")
	}

	if reservation.Status != "reserved" {
		return false, fmt.Errorf("cannot confirm %s reservation", reservation.Status)
	}

	reservation.Status = "confirmed"
	restate.Set(ctx, "reservation", reservation)

	ctx.Log().Info("Hotel confirmed", "id", reservationID)
	return true, nil
}

// Cancel releases the reservation
func (HotelService) Cancel(
	ctx restate.ObjectContext,
	_ restate.Void,
) (bool, error) {
	reservationID := restate.Key(ctx)

	reservation, err := restate.Get[HotelReservation](ctx, "reservation")
	if err != nil {
		return false, nil
	}

	reservation.Status = "cancelled"
	restate.Set(ctx, "reservation", reservation)

	ctx.Log().Info("Hotel cancelled", "id", reservationID)
	return true, nil
}

func calculateHotelPrice(req HotelRequest) float64 {
	nightlyRate := 200.0
	// Simple calculation: 3 nights
	return nightlyRate * 3.0
}
```

### Step 4: Payment Service (`payment_service.go`)

```go
package main

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
)

type PaymentService struct{}

// Charge processes a payment
func (PaymentService) Charge(
	ctx restate.ObjectContext,
	amount float64,
) (PaymentResult, error) {
	paymentID := restate.Key(ctx)

	ctx.Log().Info("Processing payment",
		"id", paymentID,
		"amount", amount)

	// Check existing
	existing, _ := restate.Get[*PaymentResult](ctx, "payment")
	if existing != nil {
		return *existing, nil
	}

	// Simulate payment processing (journaled)
	success, err := restate.Run(ctx, func(ctx restate.RunContext) (bool, error) {
		// In production: call payment gateway
		// Here: simulate 95% success rate
		return true, nil  // Success
	})

	if err != nil || !success {
		return PaymentResult{}, fmt.Errorf("payment failed")
	}

	payment := PaymentResult{
		ID:     paymentID,
		Amount: amount,
		Status: "succeeded",
	}

	restate.Set(ctx, "payment", payment)
	return payment, nil
}

// Refund reverses a payment
func (PaymentService) Refund(
	ctx restate.ObjectContext,
	_ restate.Void,
) (bool, error) {
	paymentID := restate.Key(ctx)

	payment, err := restate.Get[PaymentResult](ctx, "payment")
	if err != nil {
		return false, fmt.Errorf("payment not found")
	}

	// Simulate refund (journaled)
	_, err = restate.Run(ctx, func(ctx restate.RunContext) (bool, error) {
		// In production: call payment gateway refund API
		ctx.Log().Info("Refunding payment", "id", paymentID)
		return true, nil
	})

	if err != nil {
		return false, err
	}

	payment.Status = "refunded"
	restate.Set(ctx, "payment", payment)
	return true, nil
}
```

### Step 5: Notification Service (`notification_service.go`)

```go
package main

import (
	restate "github.com/restatedev/sdk-go"
)

type NotificationService struct{}

// SendConfirmation sends trip confirmation email
func (NotificationService) SendConfirmation(
	ctx restate.ServiceContext,
	booking TripBooking,
) (bool, error) {
	ctx.Log().Info("Sending confirmation email",
		"tripId", booking.TripID,
		"email", booking.Flight.From) // using From as placeholder

	// Send email (journaled)
	sent, err := restate.Run(ctx, func(ctx restate.RunContext) (bool, error) {
		// In production: call SendGrid/SES
		ctx.Log().Info("Email sent successfully")
		return true, nil
	})

	return sent, err
}
```

### Step 6: Trip Orchestrator (`trip_orchestrator.go`)

The main coordinator:

```go
package main

import (
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
)

type TripOrchestrator struct{}

// BookTrip orchestrates the entire booking workflow
func (TripOrchestrator) BookTrip(
	ctx restate.WorkflowContext,
	req TripRequest,
) (TripBooking, error) {
	tripID := restate.Key(ctx)

	ctx.Log().Info("Booking trip",
		"tripId", tripID,
		"customer", req.CustomerID)

	// Check if already booked (idempotent)
	existing, _ := restate.Get[*TripBooking](ctx, "booking")
	if existing != nil {
		ctx.Log().Info("Trip already booked")
		return *existing, nil
	}

	// PHASE 1: Reserve resources (tentative)

	// Reserve flight
	flightResID := fmt.Sprintf("%s-flight", tripID)
	flightRes, err := restate.Object[FlightReservation](
		ctx, "FlightService", flightResID, "Reserve",
	).Request(req.Flight)

	if err != nil {
		return TripBooking{}, fmt.Errorf("flight reservation failed: %w", err)
	}

	// Reserve hotel
	hotelResID := fmt.Sprintf("%s-hotel", tripID)
	hotelRes, err := restate.Object[HotelReservation](
		ctx, "HotelService", hotelResID, "Reserve",
	).Request(req.Hotel)

	if err != nil {
		// Compensate: cancel flight
		restate.ObjectSend(ctx, "FlightService", flightResID, "Cancel").Send(restate.Void{})
		return TripBooking{}, fmt.Errorf("hotel reservation failed: %w", err)
	}

	// PHASE 2: Process payment (commit point)

	total := flightRes.Price + hotelRes.Price
	paymentID := fmt.Sprintf("%s-payment", tripID)

	payment, err := restate.Object[PaymentResult](
		ctx, "PaymentService", paymentID, "Charge",
	).Request(total)

	if err != nil {
		// Compensate: cancel both reservations
		restate.ObjectSend(ctx, "FlightService", flightResID, "Cancel").Send(restate.Void{})
		restate.ObjectSend(ctx, "HotelService", hotelResID, "Cancel").Send(restate.Void{})
		return TripBooking{}, fmt.Errorf("payment failed: %w", err)
	}

	// PHASE 3: Confirm reservations (permanent)

	_, err1 := restate.Object[bool](
		ctx, "FlightService", flightResID, "Confirm",
	).Request(restate.Void{})

	_, err2 := restate.Object[bool](
		ctx, "HotelService", hotelResID, "Confirm",
	).Request(restate.Void{})

	if err1 != nil || err2 != nil {
		ctx.Log().Error("Failed to confirm reservations",
			"flightErr", err1,
			"hotelErr", err2)
		// Payment succeeded but confirmations failed
		// In production: alert operations team
	}

	// PHASE 4: Send confirmation email (non-critical)

	booking := TripBooking{
		TripID:  tripID,
		Flight:  flightRes,
		Hotel:   hotelRes,
		Payment: payment,
		Status:  "confirmed",
		CreatedAt: time.Now(),
	}

	emailSent, err := restate.Service[bool](
		ctx, "NotificationService", "SendConfirmation",
	).Request(booking)

	if err != nil {
		ctx.Log().Warn("Failed to send confirmation email", "error", err)
	} else {
		booking.ConfirmationEmail = emailSent
	}

	// Save final booking
	restate.Set(ctx, "booking", booking)

	ctx.Log().Info("Trip booked successfully", "tripId", tripID)
	return booking, nil
}

// GetBooking retrieves trip status
func (TripOrchestrator) GetBooking(
	ctx restate.WorkflowSharedContext,
	_ restate.Void,
) (TripBooking, error) {
	booking, err := restate.Get[TripBooking](ctx, "booking")
	return booking, err
}
```

### Step 7: Main Server (`main.go`)

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

	// Register all services
	services := []interface{}{
		FlightService{},
		HotelService{},
		PaymentService{},
		NotificationService{},
		TripOrchestrator{},
	}

	for _, svc := range services {
		if err := restateServer.Bind(restate.Reflect(svc)); err != nil {
			log.Fatalf("Failed to bind service: %v", err)
		}
	}

	fmt.Println("🎫 Starting Travel Booking System on :9090...")
	fmt.Println("")
	fmt.Println("📝 Services:")
	fmt.Println("  ✈️  FlightService - Reserve, Confirm, Cancel")
	fmt.Println("  🏨 HotelService - Reserve, Confirm, Cancel")
	fmt.Println("  💳 PaymentService - Charge, Refund")
	fmt.Println("  📧 NotificationService - SendConfirmation")
	fmt.Println("  🎫 TripOrchestrator - BookTrip, GetBooking")
	fmt.Println("")

	if err := restateServer.Start(context.Background(), ":9090"); err != nil {
		log.Fatal("Server error:", err)
	}
}
```

### Step 8: Go Module (`go.mod`)

```go
module microservices

go 1.23

require github.com/restatedev/sdk-go v0.13.1
```

## 🚀 Running the System

### 1. Start Restate

```bash
docker run --name restate_dev --rm \
  -p 8080:8080 -p 9070:9070 -p 9091:9091 \
  --add-host=host.docker.internal:host-gateway \
  docker.io/restatedev/restate:latest
```

### 2. Start Services

```bash
go mod tidy
go run .
```

### 3. Register Services

```bash
curl -X POST http://localhost:9070/deployments \
  -H 'Content-Type: application/json' \
  -d '{"uri": "http://host.docker.internal:9090"}'
```

## 🧪 Testing

### Book a Trip

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

**Expected:** Complete booking with flight, hotel, and payment confirmed!

## 🎓 What You Learned

1. **Orchestration Pattern** - Central coordinator managing workflow
2. **Two-Phase Operations** - Reserve → Confirm pattern
3. **Compensation Logic** - Automatic cleanup on failures
4. **Service Communication** - Object-to-object calls
5. **State Management** - Workflow state tracking

## 🚀 Next Steps

👉 **Continue to [Validation](./03-validation.md)**

Test failure scenarios and compensation!
