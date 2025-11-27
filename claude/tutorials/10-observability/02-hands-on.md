# Hands-On: Observable Order Processing System

> **Add comprehensive observability to a Restate application**

## 🎯 What We're Building

An **order processing system** with full observability:
- 📊 Structured logging
- 📈 Prometheus metrics
- 🔍 Distributed tracing
- 📱 Status dashboards

## 🏗️ Project Setup

```bash
mkdir -p ~/restate-tutorials/10-observability/code
cd ~/restate-tutorials/10-observability/code
go mod init observability
go get github.com/restatedev/sdk-go@latest
go get github.com/prometheus/client_golang/prometheus
go get github.com/prometheus/client_golang/prometheus/promauto
go get github.com/prometheus/client_golang/prometheus/promhttp
```

## 📝 Implementation

### Step 1: Types and Metrics (`types.go`)

```go
package main

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Order types
type Order struct {
	ID         string    `json:"id"`
	CustomerID string    `json:"customerId"`
	Items      []string  `json:"items"`
	Total      float64   `json:"total"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"createdAt"`
}

type OrderResult struct {
	OrderID   string `json:"orderId"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// Prometheus metrics
var (
	// Counter - Total orders
	ordersTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "orders_total",
			Help: "Total number of orders processed",
		},
		[]string{"status"}, // success, failed
	)

	// Gauge - Active orders
	ordersActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "orders_active",
			Help: "Number of orders currently being processed",
		},
	)

	// Histogram - Processing duration
	orderProcessingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "order_processing_seconds",
			Help:    "Order processing duration in seconds",
			Buckets: []float64{0.1, 0.5, 1.0, 2.0, 5.0, 10.0},
		},
		[]string{"status"},
	)

	// Histogram - Order value distribution
	orderValue = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "order_value_dollars",
			Help:    "Distribution of order values",
			Buckets: []float64{10, 50, 100, 200, 500, 1000, 5000},
		},
	)

	// Counter - Payment attempts
	paymentAttempts = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "payment_attempts_total",
			Help: "Total payment processing attempts",
		},
		[]string{"result"}, // success, declined, error
	)
)
```

### Step 2: Order Service (`order_service.go`)

```go
package main

import (
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
)

type OrderService struct{}

func (OrderService) ProcessOrder(
	ctx restate.ObjectContext,
	order Order,
) (OrderResult, error) {
	start := time.Now()
	orderID := restate.Key(ctx)

	// Track active orders
	ordersActive.Inc()
	defer ordersActive.Dec()

	// Structured logging with rich context
	ctx.Log().Info("Processing order started",
		"orderId", orderID,
		"customerId", order.CustomerID,
		"itemCount", len(order.Items),
		"total", order.Total)

	// Check if already processed (idempotent)
	existing, _ := restate.Get[*OrderResult](ctx, "result")
	if existing != nil {
		ctx.Log().Info("Order already processed",
			"orderId", orderID,
			"status", existing.Status)
		return *existing, nil
	}

	// Record order value metric
	orderValue.Observe(order.Total)

	// Step 1: Validate order
	ctx.Log().Debug("Validating order",
		"orderId", orderID,
		"items", order.Items)

	if len(order.Items) == 0 {
		err := fmt.Errorf("order has no items")
		ctx.Log().Error("Order validation failed",
			"orderId", orderID,
			"error", err)

		recordFailure(ctx, orderID, start, "validation_failed")
		return OrderResult{}, restate.TerminalError(err, 400)
	}

	// Step 2: Process payment
	ctx.Log().Info("Processing payment",
		"orderId", orderID,
		"amount", order.Total)

	paymentSuccess, err := processPayment(ctx, order)
	if err != nil {
		ctx.Log().Error("Payment processing failed",
			"orderId", orderID,
			"amount", order.Total,
			"error", err)

		paymentAttempts.WithLabelValues("error").Inc()
		recordFailure(ctx, orderID, start, "payment_failed")
		return OrderResult{}, err
	}

	if !paymentSuccess {
		ctx.Log().Warn("Payment declined",
			"orderId", orderID,
			"amount", order.Total)

		paymentAttempts.WithLabelValues("declined").Inc()
		recordFailure(ctx, orderID, start, "payment_declined")
		return OrderResult{
			OrderID:   orderID,
			Status:    "payment_declined",
			Message:   "Payment was declined",
			Timestamp: time.Now(),
		}, nil
	}

	paymentAttempts.WithLabelValues("success").Inc()
	ctx.Log().Info("Payment successful",
		"orderId", orderID,
		"amount", order.Total)

	// Step 3: Fulfill order
	ctx.Log().Info("Fulfilling order",
		"orderId", orderID)

	fulfillmentID, err := fulfillOrder(ctx, order)
	if err != nil {
		ctx.Log().Error("Order fulfillment failed",
			"orderId", orderID,
			"error", err)

		recordFailure(ctx, orderID, start, "fulfillment_failed")
		return OrderResult{}, err
	}

	// Success!
	result := OrderResult{
		OrderID:   orderID,
		Status:    "completed",
		Message:   fmt.Sprintf("Order fulfilled: %s", fulfillmentID),
		Timestamp: time.Now(),
	}

	restate.Set(ctx, "result", result)

	// Record metrics
	duration := time.Since(start).Seconds()
	ordersTotal.WithLabelValues("success").Inc()
	orderProcessingDuration.WithLabelValues("success").Observe(duration)

	ctx.Log().Info("Order processing completed",
		"orderId", orderID,
		"fulfillmentId", fulfillmentID,
		"duration", duration)

	return result, nil
}

// GetStatus allows querying order status without blocking
func (OrderService) GetStatus(
	ctx restate.ObjectSharedContext,
	_ restate.Void,
) (OrderResult, error) {
	orderID := restate.Key(ctx)

	result, err := restate.Get[OrderResult](ctx, "result")
	if err != nil {
		return OrderResult{
			OrderID: orderID,
			Status:  "not_found",
			Message: "Order not found",
		}, nil
	}

	ctx.Log().Debug("Status query",
		"orderId", orderID,
		"status", result.Status)

	return result, nil
}

// Helper functions

func processPayment(ctx restate.ObjectContext, order Order) (bool, error) {
	// Simulate payment processing (journaled)
	success, err := restate.Run(ctx, func(ctx restate.RunContext) (bool, error) {
		// In production: call payment gateway
		// Here: simulate 90% success rate
		return restate.Rand(ctx).Float64() > 0.1, nil
	})

	return success, err
}

func fulfillOrder(ctx restate.ObjectContext, order Order) (string, error) {
	// Simulate fulfillment (journaled)
	fulfillmentID, err := restate.Run(ctx, func(ctx restate.RunContext) (string, error) {
		// In production: call warehouse system
		return fmt.Sprintf("fulfillment-%d", restate.Rand(ctx).Uint64()), nil
	})

	return fulfillmentID, err
}

func recordFailure(ctx restate.ObjectContext, orderID string, start time.Time, status string) {
	duration := time.Since(start).Seconds()
	ordersTotal.WithLabelValues("failed").Inc()
	orderProcessingDuration.WithLabelValues("failed").Observe(duration)

	result := OrderResult{
		OrderID:   orderID,
		Status:    status,
		Message:   "Order processing failed",
		Timestamp: time.Now(),
	}
	restate.Set(ctx, "result", result)
}
```

### Step 3: Main Server with Metrics Endpoint (`main.go`)

```go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/server"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Start metrics server on separate port
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		fmt.Println("📊 Metrics server starting on :2112/metrics")
		if err := http.ListenAndServe(":2112", nil); err != nil {
			log.Fatal("Metrics server error:", err)
		}
	}()

	// Start Restate server
	restateServer := server.NewRestate()

	if err := restateServer.Bind(restate.Reflect(OrderService{})); err != nil {
		log.Fatal("Failed to bind service:", err)
	}

	fmt.Println("🚀 Observable Order Service starting on :9090")
	fmt.Println("")
	fmt.Println("📝 Services:")
	fmt.Println("  🛒 OrderService - ProcessOrder, GetStatus")
	fmt.Println("")
	fmt.Println("📊 Observability:")
	fmt.Println("  Logs: Structured JSON logs")
	fmt.Println("  Metrics: http://localhost:2112/metrics")
	fmt.Println("  Traces: OpenTelemetry (if configured)")
	fmt.Println("")

	if err := restateServer.Start(context.Background(), ":9090"); err != nil {
		log.Fatal("Server error:", err)
	}
}
```

### Step 4: Go Module (`go.mod`)

```go
module observability

go 1.23

require (
	github.com/prometheus/client_golang v1.19.0
	github.com/restatedev/sdk-go v0.13.1
)
```

## 🚀 Running the System

### 1. Start Restate

```bash
docker run --name restate_dev --rm \
  -p 8080:8080 -p 9070:9070 -p 9091:9091 \
  --add-host=host.docker.internal:host-gateway \
  docker.io/restatedev/restate:latest
```

### 2. Start Order Service

```bash
go mod tidy
go run .
```

### 3. Register Service

```bash
curl -X POST http://localhost:9070/deployments \
  -H 'Content-Type: application/json' \
  -d '{"uri": "http://host.docker.internal:9090"}'
```

## 🧪 Testing Observability

### Process Orders

```bash
# Successful order
curl -X POST http://localhost:8080/OrderService/order-001/ProcessOrder \
  -H 'Content-Type: application/json' \
  -d '{
    "id": "order-001",
    "customerId": "cust-123",
    "items": ["item1", "item2"],
    "total": 99.99
  }'

# Another order
curl -X POST http://localhost:8080/OrderService/order-002/ProcessOrder \
  -H 'Content-Type: application/json' \
  -d '{
    "id": "order-002",
    "customerId": "cust-456",
    "items": ["item3"],
    "total": 45.00
  }'
```

### View Logs

Check your terminal for structured logs:
```json
{"level":"info","service":"OrderService","key":"order-001","msg":"Processing order started","orderId":"order-001","customerId":"cust-123","itemCount":2,"total":99.99}
{"level":"info","service":"OrderService","key":"order-001","msg":"Processing payment","orderId":"order-001","amount":99.99}
{"level":"info","service":"OrderService","key":"order-001","msg":"Payment successful","orderId":"order-001","amount":99.99}
{"level":"info","service":"OrderService","key":"order-001","msg":"Order processing completed","orderId":"order-001","duration":0.234}
```

### View Metrics

```bash
curl http://localhost:2112/metrics | grep orders

# Output:
# orders_total{status="success"} 2
# orders_total{status="failed"} 0
# orders_active 0
# order_processing_seconds_bucket{status="success",le="0.5"} 2
# order_value_dollars_bucket{le="100"} 2
```

### Query Order Status

```bash
curl -X POST http://localhost:8080/OrderService/order-001/GetStatus \
  -H 'Content-Type: application/json' \
  -d '{}'
```

## 📊 Setting Up Dashboards (Optional)

### Prometheus Configuration

Create `prometheus.yml`:
```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'order-service'
    static_configs:
      - targets: ['host.docker.internal:2112']
  
  - job_name: 'restate'
    static_configs:
      - targets: ['host.docker.internal:9091']
```

### Start Prometheus

```bash
docker run -d -p 9090:9090 \
  -v $(pwd)/prometheus.yml:/etc/prometheus/prometheus.yml \
  prom/prometheus
```

Access at: http://localhost:9090

### Sample Queries

```promql
# Order success rate
rate(orders_total{status="success"}[5m]) / rate(orders_total[5m])

# P99 processing time
histogram_quantile(0.99, rate(order_processing_seconds_bucket[5m]))

# Active orders
orders_active
```

## 🎓 What You Learned

1. **Structured Logging** - Rich context in every log
2. **Prometheus Metrics** - Counters, gauges, histograms
3. **Metrics Export** - Exposing `/metrics` endpoint
4. **Status Queries** - Non-blocking status checks
5. **Observability Patterns** - Best practices in action

## 🚀 Next Steps

👉 **Continue to [Validation](./03-validation.md)**

Test your observability setup!
