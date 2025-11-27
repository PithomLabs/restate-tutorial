# Hands-On: Building an Idempotent Payment Service

> **Build a production-grade payment service with bulletproof idempotency**

## 🎯 What We're Building

A **Payment Processing Service** that demonstrates:
- ✅ Idempotent payment creation
- ✅ Duplicate request detection
- ✅ Safe retry handling
- ✅ Payment status tracking
- ✅ Refund processing

## 📋 Prerequisites

- ✅ Go 1.23+ installed
- ✅ Completed Module 01 (Foundation)
- ✅ Completed Module 04 (Virtual Objects)
- ✅ Docker for Restate server

## 🏗️ Project Setup

### Step 1: Create Project Directory

```bash
mkdir -p ~/restate-tutorials/idempotency/code
cd ~/restate-tutorials/idempotency/code
```

### Step 2: Initialize Go Module

```bash
go mod init idempotency
go get github.com/restatedev/sdk-go@latest
```

### Step 3: Create File Structure

```bash
touch types.go payment_service.go gateway.go main.go
```

## 📝 Implementation

### Step 1: Define Types (`types.go`)

```go
package main

import "time"

// Payment request from client
type PaymentRequest struct {
	Amount      int    `json:"amount"`       // Amount in cents
	Currency    string `json:"currency"`     // USD, EUR, etc.
	Description string `json:"description"`  // Payment description
	CustomerID  string `json:"customerId"`   // Customer identifier
}

// Payment stored in state
type Payment struct {
	PaymentID   string    `json:"paymentId"`
	Amount      int       `json:"amount"`
	Currency    string    `json:"currency"`
	Description string    `json:"description"`
	CustomerID  string    `json:"customerId"`
	Status      string    `json:"status"`      // "pending", "completed", "failed"
	ChargeID    string    `json:"chargeId"`    // External gateway charge ID
	CreatedAt   time.Time `json:"createdAt"`
	CompletedAt time.Time `json:"completedAt,omitempty"`
	ErrorMsg    string    `json:"errorMsg,omitempty"`
}

// Payment result returned to client
type PaymentResult struct {
	PaymentID string `json:"paymentId"`
	Status    string `json:"status"`
	ChargeID  string `json:"chargeId,omitempty"`
	Message   string `json:"message,omitempty"`
}

// Refund request
type RefundRequest struct {
	Reason string `json:"reason"`
	Amount int    `json:"amount,omitempty"` // Optional partial refund
}

// Refund result
type RefundResult struct {
	RefundID  string `json:"refundId"`
	Status    string `json:"status"`
	Amount    int    `json:"amount"`
	Message   string `json:"message,omitempty"`
}
```

### Step 2: Mock Payment Gateway (`gateway.go`)

In production, this would call Stripe, Square, etc. For learning, we'll mock it:

```go
package main

import (
	"fmt"
	"math/rand"
	"time"
)

// MockPaymentGateway simulates an external payment processor
type MockPaymentGateway struct{}

// ChargeResponse from gateway
type ChargeResponse struct {
	ChargeID  string
	Success   bool
	ErrorMsg  string
}

// Charge processes a payment (simulated)
func (g *MockPaymentGateway) Charge(
	amount int,
	currency string,
	customerID string,
) ChargeResponse {
	// Simulate network delay
	time.Sleep(100 * time.Millisecond)
	
	// Simulate occasional failures (10% of the time)
	if rand.Float64() < 0.1 {
		return ChargeResponse{
			Success:  false,
			ErrorMsg: "insufficient funds",
		}
	}
	
	// Generate charge ID
	chargeID := fmt.Sprintf("ch_%s_%d", customerID, time.Now().Unix())
	
	return ChargeResponse{
		ChargeID: chargeID,
		Success:  true,
	}
}

// Refund processes a refund (simulated)
func (g *MockPaymentGateway) Refund(
	chargeID string,
	amount int,
) ChargeResponse {
	// Simulate network delay
	time.Sleep(100 * time.Millisecond)
	
	// Generate refund ID
	refundID := fmt.Sprintf("re_%s_%d", chargeID, time.Now().Unix())
	
	return ChargeResponse{
		ChargeID: refundID,
		Success:  true,
	}
}
```

### Step 3: Payment Service (`payment_service.go`)

The core idempotent payment logic:

```go
package main

import (
	"fmt"
	"time"
	
	restate "github.com/restatedev/sdk-go"
)

type PaymentService struct{}

const (
	stateKeyPayment = "payment"
	stateKeyRefund  = "refund"
)

// CreatePayment creates a new payment (IDEMPOTENT)
func (PaymentService) CreatePayment(
	ctx restate.ObjectContext,
	req PaymentRequest,
) (PaymentResult, error) {
	paymentID := restate.Key(ctx)
	
	ctx.Log().Info("Creating payment",
		"paymentId", paymentID,
		"amount", req.Amount,
		"customerId", req.CustomerID)
	
	// Check if payment already exists (state-based deduplication)
	existingPayment, err := restate.Get[*Payment](ctx, stateKeyPayment)
	if err != nil {
		return PaymentResult{}, err
	}
	
	if existingPayment != nil {
		ctx.Log().Info("Payment already exists, returning cached result",
			"paymentId", paymentID,
			"status", existingPayment.Status)
		
		// Return existing payment (idempotent!)
		return PaymentResult{
			PaymentID: existingPayment.PaymentID,
			Status:    existingPayment.Status,
			ChargeID:  existingPayment.ChargeID,
			Message:   "Payment already processed",
		}, nil
	}
	
	// Validate request
	if req.Amount <= 0 {
		return PaymentResult{}, restate.TerminalError(
			fmt.Errorf("invalid amount: must be positive"), 400)
	}
	
	if req.Currency == "" {
		req.Currency = "USD"
	}
	
	// Create payment record in pending state
	payment := Payment{
		PaymentID:   paymentID,
		Amount:      req.Amount,
		Currency:    req.Currency,
		Description: req.Description,
		CustomerID:  req.CustomerID,
		Status:      "pending",
		CreatedAt:   time.Now(),
	}
	
	restate.Set(ctx, stateKeyPayment, payment)
	
	// Charge customer via payment gateway (IDEMPOTENT side effect)
	chargeResp, err := restate.Run(ctx, func(ctx restate.RunContext) (ChargeResponse, error) {
		ctx.Log().Info("Calling payment gateway", "paymentId", paymentID)
		
		gateway := &MockPaymentGateway{}
		resp := gateway.Charge(req.Amount, req.Currency, req.CustomerID)
		
		return resp, nil
	})
	
	if err != nil {
		// Update payment to failed state
		payment.Status = "failed"
		payment.ErrorMsg = err.Error()
		restate.Set(ctx, stateKeyPayment, payment)
		
		return PaymentResult{}, fmt.Errorf("gateway error: %w", err)
	}
	
	// Check charge result
	if !chargeResp.Success {
		ctx.Log().Warn("Payment failed",
			"paymentId", paymentID,
			"error", chargeResp.ErrorMsg)
		
		payment.Status = "failed"
		payment.ErrorMsg = chargeResp.ErrorMsg
		restate.Set(ctx, stateKeyPayment, payment)
		
		return PaymentResult{
			PaymentID: paymentID,
			Status:    "failed",
			Message:   chargeResp.ErrorMsg,
		}, nil
	}
	
	// Payment succeeded!
	ctx.Log().Info("Payment completed successfully",
		"paymentId", paymentID,
		"chargeId", chargeResp.ChargeID)
	
	payment.Status = "completed"
	payment.ChargeID = chargeResp.ChargeID
	payment.CompletedAt = time.Now()
	restate.Set(ctx, stateKeyPayment, payment)
	
	// Send receipt (IDEMPOTENT side effect)
	_, err = restate.Run(ctx, func(ctx restate.RunContext) (bool, error) {
		ctx.Log().Info("Sending payment receipt",
			"paymentId", paymentID,
			"customerId", req.CustomerID)
		
		// In production: send email via SendGrid, SES, etc.
		return true, nil
	})
	
	return PaymentResult{
		PaymentID: paymentID,
		Status:    "completed",
		ChargeID:  chargeResp.ChargeID,
		Message:   "Payment processed successfully",
	}, nil
}

// GetPayment retrieves payment status (read-only, naturally idempotent)
func (PaymentService) GetPayment(
	ctx restate.ObjectSharedContext,
	_ restate.Void,
) (Payment, error) {
	paymentID := restate.Key(ctx)
	
	payment, err := restate.Get[Payment](ctx, stateKeyPayment)
	if err != nil {
		return Payment{}, err
	}
	
	ctx.Log().Info("Retrieved payment", "paymentId", paymentID, "status", payment.Status)
	
	return payment, nil
}

// RefundPayment refunds a completed payment (IDEMPOTENT)
func (PaymentService) RefundPayment(
	ctx restate.ObjectContext,
	req RefundRequest,
) (RefundResult, error) {
	paymentID := restate.Key(ctx)
	
	ctx.Log().Info("Processing refund",
		"paymentId", paymentID,
		"reason", req.Reason)
	
	// Check if refund already exists
	existingRefund, err := restate.Get[*RefundResult](ctx, stateKeyRefund)
	if err != nil {
		return RefundResult{}, err
	}
	
	if existingRefund != nil {
		ctx.Log().Info("Refund already processed", "refundId", existingRefund.RefundID)
		return *existingRefund, nil
	}
	
	// Get payment
	payment, err := restate.Get[Payment](ctx, stateKeyPayment)
	if err != nil {
		return RefundResult{}, err
	}
	
	// Validate payment can be refunded
	if payment.Status != "completed" {
		return RefundResult{}, restate.TerminalError(
			fmt.Errorf("cannot refund payment with status: %s", payment.Status), 400)
	}
	
	// Determine refund amount
	refundAmount := req.Amount
	if refundAmount == 0 || refundAmount > payment.Amount {
		refundAmount = payment.Amount // Full refund
	}
	
	// Process refund via gateway (IDEMPOTENT side effect)
	refundResp, err := restate.Run(ctx, func(ctx restate.RunContext) (ChargeResponse, error) {
		ctx.Log().Info("Calling gateway for refund", "paymentId", paymentID)
		
		gateway := &MockPaymentGateway{}
		resp := gateway.Refund(payment.ChargeID, refundAmount)
		
		return resp, nil
	})
	
	if err != nil {
		return RefundResult{}, fmt.Errorf("refund failed: %w", err)
	}
	
	// Create refund result
	result := RefundResult{
		RefundID: refundResp.ChargeID,
		Status:   "completed",
		Amount:   refundAmount,
		Message:  "Refund processed successfully",
	}
	
	// Store refund (prevents duplicate refunds)
	restate.Set(ctx, stateKeyRefund, result)
	
	// Update payment status
	payment.Status = "refunded"
	restate.Set(ctx, stateKeyPayment, payment)
	
	ctx.Log().Info("Refund completed", "refundId", result.RefundID)
	
	return result, nil
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
	
	// Register Payment Service Virtual Object
	if err := restateServer.Bind(restate.Reflect(PaymentService{})); err != nil {
		log.Fatal("Failed to bind PaymentService:", err)
	}
	
	fmt.Println("💳 Starting Payment Service on :9090...")
	fmt.Println("")
	fmt.Println("📝 Virtual Object: PaymentService")
	fmt.Println("Handlers:")
	fmt.Println("  Exclusive (state-modifying, idempotent):")
	fmt.Println("    - CreatePayment")
	fmt.Println("    - RefundPayment")
	fmt.Println("")
	fmt.Println("  Concurrent (read-only):")
	fmt.Println("    - GetPayment")
	fmt.Println("")
	fmt.Println("✓ Ready to accept requests")
	fmt.Println("")
	fmt.Println("💡 Idempotency Features:")
	fmt.Println("  - Automatic request deduplication")
	fmt.Println("  - Journaled side effects (gateway calls)")
	fmt.Println("  - State-based duplicate detection")
	fmt.Println("  - Safe retries")
	
	if err := restateServer.Start(context.Background(), ":9090"); err != nil {
		log.Fatal("Server error:", err)
	}
}
```

### Step 5: Go Module (`go.mod`)

```go
module idempotency

go 1.23

require github.com/restatedev/sdk-go v0.13.1
```

## 🚀 Running the Service

### Step 1: Start Restate Server

```bash
docker run --name restate_dev --rm \
  -p 8080:8080 -p 9070:9070 -p 9091:9091 \
  --add-host=host.docker.internal:host-gateway \
  docker.io/restatedev/restate:latest
```

### Step 2: Install Dependencies

```bash
go mod tidy
```

### Step 3: Start Payment Service

```bash
go run .
```

Output:
```
💳 Starting Payment Service on :9090...

📝 Virtual Object: PaymentService
Handlers:
  Exclusive (state-modifying, idempotent):
    - CreatePayment
    - RefundPayment

  Concurrent (read-only):
    - GetPayment

✓ Ready to accept requests

💡 Idempotency Features:
  - Automatic request deduplication
  - Journaled side effects (gateway calls)
  - State-based duplicate detection
  - Safe retries
```

### Step 4: Register with Restate

```bash
curl -X POST http://localhost:9070/deployments \
  -H 'Content-Type: application/json' \
  -d '{"uri": "http://host.docker.internal:9090"}'
```

## 🧪 Testing Idempotency

### Test 1: Create Payment (First Time)

```bash
curl -X POST http://localhost:8080/PaymentService/payment-001/CreatePayment \
  -H 'Content-Type: application/json' \
  -d '{
    "amount": 10000,
    "currency": "USD",
    "description": "Premium subscription",
    "customerId": "cust-alice"
  }'
```

**Expected Response:**
```json
{
  "paymentId": "payment-001",
  "status": "completed",
  "chargeId": "ch_cust-alice_1234567890",
  "message": "Payment processed successfully"
}
```

### Test 2: Duplicate Request (Same Payment ID)

```bash
# Send exact same request again
curl -X POST http://localhost:8080/PaymentService/payment-001/CreatePayment \
  -H 'Content-Type: application/json' \
  -d '{
    "amount": 10000,
    "currency": "USD",
    "description": "Premium subscription",
    "customerId": "cust-alice"
  }'
```

**Expected Response:**
```json
{
  "paymentId": "payment-001",
  "status": "completed",
  "chargeId": "ch_cust-alice_1234567890",
  "message": "Payment already processed"
}
```

**Key Observation:**
- ✅ Same `chargeId` returned (from state)
- ✅ Customer NOT charged twice
- ✅ No duplicate gateway call
- ✅ Instant response (no processing delay)

### Test 3: Get Payment Status

```bash
curl -X POST http://localhost:8080/PaymentService/payment-001/GetPayment \
  -H 'Content-Type: application/json' \
  -d '{}'
```

**Expected Response:**
```json
{
  "paymentId": "payment-001",
  "amount": 10000,
  "currency": "USD",
  "description": "Premium subscription",
  "customerId": "cust-alice",
  "status": "completed",
  "chargeId": "ch_cust-alice_1234567890",
  "createdAt": "2024-11-22T00:45:00Z",
  "completedAt": "2024-11-22T00:45:01Z"
}
```

### Test 4: Process Refund

```bash
curl -X POST http://localhost:8080/PaymentService/payment-001/RefundPayment \
  -H 'Content-Type: application/json' \
  -d '{
    "reason": "Customer request",
    "amount": 10000
  }'
```

**Expected Response:**
```json
{
  "refundId": "re_ch_cust-alice_1234567890_1234567891",
  "status": "completed",
  "amount": 10000,
  "message": "Refund processed successfully"
}
```

### Test 5: Duplicate Refund

```bash
# Try refunding again
curl -X POST http://localhost:8080/PaymentService/payment-001/RefundPayment \
  -H 'Content-Type: application/json' \
  -d '{
    "reason": "Customer request",
    "amount": 10000
  }'
```

**Expected Response:**
```json
{
  "refundId": "re_ch_cust-alice_1234567890_1234567891",
  "status": "completed",
  "amount": 10000,
  "message": "Refund processed successfully"
}
```

**Key Observation:**
- ✅ Same `refundId` returned
- ✅ Customer NOT refunded twice
- ✅ Idempotent refund processing

## 🎓 Key Idempotency Patterns Demonstrated

### 1. State-Based Deduplication

```go
// Check if payment already exists
existingPayment, err := restate.Get[*Payment](ctx, stateKeyPayment)
if existingPayment != nil {
    return existingPayment, nil  // Idempotent!
}
```

### 2. Journaled Side Effects

```go
// Gateway call executed and journaled
chargeResp, err := restate.Run(ctx, func(ctx restate.RunContext) (ChargeResponse, error) {
    return gateway.Charge(amount, currency, customerID), nil
})

// On retry: returns journaled result, gateway NOT called again!
```

### 3. Explicit Status Tracking

```go
payment := Payment{
    Status: "pending",  // Initial state
}

// ... process payment ...

payment.Status = "completed"  // Final state
```

### 4. Preventing Duplicate Operations

```go
if payment.Status != "completed" {
    return error("cannot refund non-completed payment")
}

existingRefund := restate.Get[*RefundResult](ctx, stateKeyRefund)
if existingRefund != nil {
    return existingRefund  // Already refunded
}
```

## 🎯 What You Learned

### Idempotency Techniques

1. **State Checking** - Always check state before mutating
2. **Journaling** - Wrap external calls in `restate.Run()`
3. **Status Fields** - Track operation lifecycle
4. **Duplicate Detection** - Use state to prevent re-execution
5. **Consistent Responses** - Return same result for same request

### Restate Features

- ✅ Automatic invocation deduplication
- ✅ Journaled side effects
- ✅ Durable state per virtual object key
- ✅ Deterministic execution
- ✅ Safe retries

## 🚀 Next Steps

Great work! You've built an idempotent payment service!

👉 **Continue to [Validation](./03-validation.md)**

Test retry scenarios and verify idempotency guarantees!

---

**Questions?** Review [concepts](./01-concepts.md) or check the [module README](./README.md).
