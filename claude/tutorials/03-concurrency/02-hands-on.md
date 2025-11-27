# Hands-On: Building a Concurrent Order Processing Pipeline

> **Build a high-performance order processor using parallel execution**

## 🎯 What We're Building

An **Order Processing Service** that:
- Validates inventory across multiple warehouses (parallel)
- Checks payment and fraud detection simultaneously (parallel)
- Calculates shipping from multiple carriers (parallel)
- Aggregates results and creates order

**Performance Goal:** Reduce order processing time from ~1 second to ~200ms using parallelism

## 📋 Prerequisites

- ✅ Completed [Module 02](../02-side-effects/README.md)
- ✅ Understanding of futures from [concepts](./01-concepts.md)
- ✅ Restate server running

## 🚀 Step-by-Step Tutorial

### Step 1: Project Setup

```bash
# Create project directory
mkdir -p ~/restate-tutorials/module03
cd ~/restate-tutorials/module03

# Initialize Go module
go mod init module03

# Install dependencies
go get github.com/restatedev/sdk-go
```

### Step 2: Create Supporting Services

First, let's create the services our order processor will call.

Create `supporting_services.go`:

```go
package main

import (
	"fmt"
	"math/rand"
	"time"

	restate "github.com/restatedev/sdk-go"
)

// ============================================
// Inventory Service
// ============================================

type InventoryService struct{}

type InventoryRequest struct {
	ProductID  string `json:"productId"`
	Quantity   int    `json:"quantity"`
	Warehouse  string `json:"warehouse"`
}

type InventoryResponse struct {
	Available bool   `json:"available"`
	Warehouse string `json:"warehouse"`
	Quantity  int    `json:"quantity"`
}

func (InventoryService) CheckInventory(
	ctx restate.Context,
	req InventoryRequest,
) (InventoryResponse, error) {
	// Simulate database query delay
	err := restate.Sleep(ctx, 80*time.Millisecond)
	if err != nil {
		return InventoryResponse{}, err
	}

	// Simulate occasional failures
	if rand.Float64() < 0.05 {
		return InventoryResponse{}, fmt.Errorf("warehouse %s temporarily unavailable", req.Warehouse)
	}

	// Simulate inventory availability (80% available)
	available := rand.Float64() < 0.8

	return InventoryResponse{
		Available: available,
		Warehouse: req.Warehouse,
		Quantity:  req.Quantity,
	}, nil
}

// ============================================
// Payment Service
// ============================================

type PaymentService struct{}

type PaymentRequest struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	CardLast4 string `json:"cardLast4"`
}

type PaymentResponse struct {
	Authorized    bool   `json:"authorized"`
	TransactionID string `json:"transactionId"`
}

func (PaymentService) AuthorizePayment(
	ctx restate.Context,
	req PaymentRequest,
) (PaymentResponse, error) {
	// Simulate payment gateway delay
	err := restate.Sleep(ctx, 120*time.Millisecond)
	if err != nil {
		return PaymentResponse{}, err
	}

	// Simulate failures
	if rand.Float64() < 0.02 {
		return PaymentResponse{}, fmt.Errorf("payment gateway timeout")
	}

	// Simulate authorization (95% success)
	authorized := rand.Float64() < 0.95

	txID := ""
	if authorized {
		txID = fmt.Sprintf("tx_%s", restate.UUID(ctx).String()[:8])
	}

	return PaymentResponse{
		Authorized:    authorized,
		TransactionID: txID,
	}, nil
}

// ============================================
// Fraud Detection Service
// ============================================

type FraudService struct{}

type FraudRequest struct {
	UserID    string  `json:"userId"`
	Amount    float64 `json:"amount"`
	IPAddress string  `json:"ipAddress"`
}

type FraudResponse struct {
	RiskScore float64 `json:"riskScore"` // 0-100
	Flagged   bool    `json:"flagged"`
}

func (FraudService) CheckFraud(
	ctx restate.Context,
	req FraudRequest,
) (FraudResponse, error) {
	// Simulate ML model inference delay
	err := restate.Sleep(ctx, 150*time.Millisecond)
	if err != nil {
		return FraudResponse{}, err
	}

	// Simulate risk score (mostly low risk)
	riskScore := rand.Float64() * 50 // 0-50 (low risk)
	
	// Occasionally flag high risk
	if rand.Float64() < 0.1 {
		riskScore = 50 + rand.Float64()*50 // 50-100 (high risk)
	}

	return FraudResponse{
		RiskScore: riskScore,
		Flagged:   riskScore > 70,
	}, nil
}

// ============================================
// Shipping Service
// ============================================

type ShippingService struct{}

type ShippingRequest struct {
	Weight      float64 `json:"weight"`
	Destination string  `json:"destination"`
	Carrier     string  `json:"carrier"`
}

type ShippingResponse struct {
	Carrier        string  `json:"carrier"`
	Cost           float64 `json:"cost"`
	EstimatedDays  int     `json:"estimatedDays"`
}

func (ShippingService) CalculateShipping(
	ctx restate.Context,
	req ShippingRequest,
) (ShippingResponse, error) {
	// Simulate API call delay
	err := restate.Sleep(ctx, 100*time.Millisecond)
	if err != nil {
		return ShippingResponse{}, err
	}

	// Simulate carrier-specific costs
	baseCost := req.Weight * 2.5
	switch req.Carrier {
	case "FastShip":
		baseCost *= 1.5
	case "Standard":
		baseCost *= 1.0
	case "Economy":
		baseCost *= 0.7
	}

	days := 3
	if req.Carrier == "FastShip" {
		days = 1
	} else if req.Carrier == "Economy" {
		days = 7
	}

	return ShippingResponse{
		Carrier:       req.Carrier,
		Cost:          baseCost,
		EstimatedDays: days,
	}, nil
}
```

**🎓 Learning Points:**
- These are simple Basic Services
- Each has realistic latency (80-150ms)
- Occasional failures for testing resilience

### Step 3: Create Order Service (Sequential - Slow!)

Let's first build a sequential version to see the problem.

Create `order_service_slow.go`:

```go
package main

import (
	"fmt"
	restate "github.com/restatedev/sdk-go"
)

type OrderServiceSlow struct{}

type OrderRequest struct {
	UserID      string  `json:"userId"`
	ProductID   string  `json:"productId"`
	Quantity    int     `json:"quantity"`
	Amount      float64 `json:"amount"`
	Weight      float64 `json:"weight"`
	Destination string  `json:"destination"`
}

type OrderResult struct {
	OrderID         string  `json:"orderId"`
	Status          string  `json:"status"`
	ProcessingTimeMs int64  `json:"processingTimeMs"`
	Details         string  `json:"details"`
}

// ❌ SLOW: Sequential execution
func (OrderServiceSlow) ProcessOrder(
	ctx restate.Context,
	req OrderRequest,
) (OrderResult, error) {
	startTime := time.Now()
	
	ctx.Log().Info("Processing order sequentially", "userId", req.UserID)

	// Step 1: Check inventory - 80ms
	inv, err := restate.Service[InventoryResponse](
		ctx, "InventoryService", "CheckInventory",
	).Request(InventoryRequest{
		ProductID: req.ProductID,
		Quantity:  req.Quantity,
		Warehouse: "Main",
	})
	if err != nil || !inv.Available {
		return OrderResult{
			Status:  "failed",
			Details: "Inventory not available",
		}, nil
	}

	// Step 2: Authorize payment - 120ms
	payment, err := restate.Service[PaymentResponse](
		ctx, "PaymentService", "AuthorizePayment",
	).Request(PaymentRequest{
		Amount:   req.Amount,
		Currency: "USD",
	})
	if err != nil || !payment.Authorized {
		return OrderResult{
			Status:  "failed",
			Details: "Payment authorization failed",
		}, nil
	}

	// Step 3: Check fraud - 150ms
	fraud, err := restate.Service[FraudResponse](
		ctx, "FraudService", "CheckFraud",
	).Request(FraudRequest{
		UserID: req.UserID,
		Amount: req.Amount,
	})
	if err != nil {
		return OrderResult{}, err
	}
	if fraud.Flagged {
		return OrderResult{
			Status:  "failed",
			Details: "High fraud risk",
		}, nil
	}

	// Step 4: Calculate shipping - 100ms
	shipping, err := restate.Service[ShippingResponse](
		ctx, "ShippingService", "CalculateShipping",
	).Request(ShippingRequest{
		Weight:      req.Weight,
		Destination: req.Destination,
		Carrier:     "Standard",
	})
	if err != nil {
		return OrderResult{}, err
	}

	// Create order
	orderID := fmt.Sprintf("ORD_%s", restate.UUID(ctx).String()[:8])
	
	duration := time.Since(startTime)
	ctx.Log().Info("Order processed sequentially",
		"orderId", orderID,
		"durationMs", duration.Milliseconds())

	return OrderResult{
		OrderID:         orderID,
		Status:          "confirmed",
		ProcessingTimeMs: duration.Milliseconds(),
		Details:         fmt.Sprintf("Shipping: $%.2f via %s", shipping.Cost, shipping.Carrier),
	}, nil
}
```

**⏱️ Total Time:** ~450ms (80 + 120 + 150 + 100)

### Step 4: Create Order Service (Parallel - Fast!)

Now let's build the optimized parallel version.

Create `order_service.go`:

```go
package main

import (
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
)

type OrderService struct{}

// ✅ FAST: Parallel execution
func (OrderService) ProcessOrder(
	ctx restate.Context,
	req OrderRequest,
) (OrderResult, error) {
	startTime := time.Now()
	
	ctx.Log().Info("Processing order in parallel", "userId", req.UserID)

	// ===================================
	// Fan-Out: Start all checks in parallel
	// ===================================

	// Check inventory in multiple warehouses (parallel)
	invFutMain := restate.Service[InventoryResponse](
		ctx, "InventoryService", "CheckInventory",
	).RequestFuture(InventoryRequest{
		ProductID: req.ProductID,
		Quantity:  req.Quantity,
		Warehouse: "Main",
	})

	invFutBackup := restate.Service[InventoryResponse](
		ctx, "InventoryService", "CheckInventory",
	).RequestFuture(InventoryRequest{
		ProductID: req.ProductID,
		Quantity:  req.Quantity,
		Warehouse: "Backup",
	})

	// Authorize payment (parallel)
	paymentFut := restate.Service[PaymentResponse](
		ctx, "PaymentService", "AuthorizePayment",
	).RequestFuture(PaymentRequest{
		Amount:   req.Amount,
		Currency: "USD",
	})

	// Check fraud (parallel)
	fraudFut := restate.Service[FraudResponse](
		ctx, "FraudService", "CheckFraud",
	).RequestFuture(FraudRequest{
		UserID: req.UserID,
		Amount: req.Amount,
	})

	// Calculate shipping from multiple carriers (parallel)
	shippingFutStandard := restate.Service[ShippingResponse](
		ctx, "ShippingService", "CalculateShipping",
	).RequestFuture(ShippingRequest{
		Weight:      req.Weight,
		Destination: req.Destination,
		Carrier:     "Standard",
	})

	shippingFutEconomy := restate.Service[ShippingResponse](
		ctx, "ShippingService", "CalculateShipping",
	).RequestFuture(ShippingRequest{
		Weight:      req.Weight,
		Destination: req.Destination,
		Carrier:     "Economy",
	})

	// ===================================
	// Fan-In: Collect and process results
	// ===================================

	var inventoryAvailable bool
	var paymentAuthorized bool
	var fraudRisk float64
	var shippingOptions []ShippingResponse

	// Wait for all futures to complete
	for fut, err := range restate.Wait(ctx,
		invFutMain, invFutBackup,
		paymentFut, fraudFut,
		shippingFutStandard, shippingFutEconomy) {

		if err != nil {
			ctx.Log().Warn("Future failed", "error", err)
			continue // Partial failure handling
		}

		// Process each result
		switch fut {
		case invFutMain:
			inv, _ := invFutMain.Response()
			if inv.Available {
				inventoryAvailable = true
				ctx.Log().Info("Inventory available", "warehouse", "Main")
			}
			
		case invFutBackup:
			inv, _ := invFutBackup.Response()
			if inv.Available && !inventoryAvailable {
				inventoryAvailable = true
				ctx.Log().Info("Inventory available", "warehouse", "Backup")
			}

		case paymentFut:
			payment, _ := paymentFut.Response()
			paymentAuthorized = payment.Authorized
			ctx.Log().Info("Payment check complete", "authorized", paymentAuthorized)

		case fraudFut:
			fraud, _ := fraudFut.Response()
			fraudRisk = fraud.RiskScore
			ctx.Log().Info("Fraud check complete", "riskScore", fraudRisk)

		case shippingFutStandard:
			shipping, _ := shippingFutStandard.Response()
			shippingOptions = append(shippingOptions, shipping)

		case shippingFutEconomy:
			shipping, _ := shippingFutEconomy.Response()
			shippingOptions = append(shippingOptions, shipping)
		}
	}

	// ===================================
	// Validation: Check all requirements
	// ===================================

	if !inventoryAvailable {
		return OrderResult{
			Status:  "failed",
			Details: "Product not available in any warehouse",
		}, nil
	}

	if !paymentAuthorized {
		return OrderResult{
			Status:  "failed",
			Details: "Payment authorization failed",
		}, nil
	}

	if fraudRisk > 70 {
		return OrderResult{
			Status:  "failed",
			Details: fmt.Sprintf("High fraud risk: %.1f", fraudRisk),
		}, nil
	}

	if len(shippingOptions) == 0 {
		return OrderResult{
			Status:  "failed",
			Details: "No shipping options available",
		}, nil
	}

	// Choose best shipping option (lowest cost)
	bestShipping := shippingOptions[0]
	for _, opt := range shippingOptions {
		if opt.Cost < bestShipping.Cost {
			bestShipping = opt
		}
	}

	// ===================================
	// Success: Create order
	// ===================================

	orderID := fmt.Sprintf("ORD_%s", restate.UUID(ctx).String()[:8])
	
	duration := time.Since(startTime)
	ctx.Log().Info("Order processed in parallel",
		"orderId", orderID,
		"durationMs", duration.Milliseconds())

	return OrderResult{
		OrderID:          orderID,
		Status:           "confirmed",
		ProcessingTimeMs: duration.Milliseconds(),
		Details: fmt.Sprintf(
			"Shipping: $%.2f via %s (%d days)",
			bestShipping.Cost,
			bestShipping.Carrier,
			bestShipping.EstimatedDays,
		),
	}, nil
}
```

**⏱️ Total Time:** ~150ms (max of all parallel operations)
**Speedup:** 3x faster! 🚀

**🎓 Key Learning Points:**

1. **Fan-Out Pattern**
   ```go
   fut1 := service1().RequestFuture(...)
   fut2 := service2().RequestFuture(...)
   // All start immediately!
   ```

2. **Fan-In with restate.Wait**
   ```go
   for fut, err := range restate.Wait(ctx, fut1, fut2, ...) {
       // Process as results arrive
   }
   ```

3. **Partial Failure Handling**
   - Check inventory in multiple warehouses
   - If one fails, try another
   - Only fail if all options exhausted

4. **Result Aggregation**
   - Collect shipping options
   - Choose best option (lowest cost)
   - Flexible decision logic

### Step 5: Create Main Entry Point

Create `main.go`:

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
		OrderService{},
		OrderServiceSlow{},
		InventoryService{},
		PaymentService{},
		FraudService{},
		ShippingService{},
	}

	for _, svc := range services {
		if err := restateServer.Bind(restate.Reflect(svc)); err != nil {
			log.Fatal("Failed to bind service:", err)
		}
	}

	fmt.Println("🛒 Starting Order Processing Services on :9090...")
	fmt.Println("📝 Services:")
	fmt.Println("  - OrderService (parallel)")
	fmt.Println("  - OrderServiceSlow (sequential)")
	fmt.Println("  - InventoryService")
	fmt.Println("  - PaymentService")
	fmt.Println("  - FraudService")
	fmt.Println("  - ShippingService")
	fmt.Println("")

	if err := restateServer.Start(context.Background(), ":9090"); err != nil {
		log.Fatal("Server error:", err)
	}
}
```

### Step 6: Build and Run

```bash
# Build
go mod tidy
go build -o order-service

# Run
./order-service
```

### Step 7: Register with Restate

```bash
curl -X POST http://localhost:8080/deployments \
  -H 'Content-Type: application/json' \
  -d '{"uri": "http://localhost:9090"}'
```

### Step 8: Test Both Versions

**Test sequential (slow) version:**
```bash
time curl -X POST http://localhost:9080/OrderServiceSlow/ProcessOrder \
  -H 'Content-Type: application/json' \
  -d '{
    "userId": "user123",
    "productId": "prod456",
    "quantity": 2,
    "amount": 99.99,
    "weight": 5.5,
    "destination": "New York"
  }'
```

**Test parallel (fast) version:**
```bash
time curl -X POST http://localhost:9080/OrderService/ProcessOrder \
  -H 'Content-Type: application/json' \
  -d '{
    "userId": "user123",
    "productId": "prod456",
    "quantity": 2,
    "amount": 99.99,
    "weight": 5.5,
    "destination": "New York"
  }'
```

**Compare the results:**
- Sequential: ~450-500ms
- Parallel: ~150-200ms
- **Speedup: 2.5-3x faster!**

## 🎓 Understanding the Flow

### Sequential Flow
```
[Inventory: 80ms] → [Payment: 120ms] → [Fraud: 150ms] → [Shipping: 100ms]
Total: ~450ms
```

### Parallel Flow
```
┌─ [Inventory Main: 80ms] ────┐
├─ [Inventory Backup: 80ms] ──┤
├─ [Payment: 120ms] ──────────┤  ← All execute simultaneously
├─ [Fraud: 150ms] ────────────┤     Max time determines total
├─ [Shipping Standard: 100ms]─┤
└─ [Shipping Economy: 100ms]──┘
Total: ~150ms (max of all)
```

## ✅ Verification Checklist

- [ ] Both services start successfully
- [ ] Can call OrderServiceSlow and OrderService
- [ ] Parallel version is 2-3x faster
- [ ] Handles partial failures (some inventory unavailable)
- [ ] Logs show parallel execution
- [ ] Returns best shipping option

## 💡 Key Takeaways

1. **Parallelism Reduces Latency**
   - Sequential: sum of all operations
   - Parallel: maximum of all operations

2. **Futures are Journaled**
   - Created futures are journaled immediately
   - Results replay from journal on retry
   - No duplicate execution

3. **Resilience Through Redundancy**
   - Check multiple warehouses
   - Calculate shipping from multiple carriers
   - Graceful degradation

4. **Smart Aggregation**
   - Wait for all results
   - Handle partial failures
   - Make decisions based on available data

## 🎯 Next Steps

Ready to validate your concurrent service!

👉 **Continue to [Validation](./03-validation.md)**

---

**Questions?** Review [concepts](./01-concepts.md) or check the [complete code](./code/)!
