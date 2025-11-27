# Concepts: Observability in Restate

> **Understand monitoring, logging, and tracing for distributed systems**

## 🎯 What You'll Learn

- The three pillars of observability
- Restate's built-in observability features
- Structured logging best practices
- Metrics collection and visualization
- Distributed tracing
- Debugging workflows

---

## 📖 The Three Pillars of Observability

### 1. Logs - What Happened

**Purpose:** Detailed event records

```
2024-11-22T01:00:00Z INFO [OrderService/order-123] Order created customer=cust-456 total=99.99
2024-11-22T01:00:01Z INFO [PaymentService/pay-789] Processing payment amount=99.99
2024-11-22T01:00:02Z ERROR [PaymentService/pay-789] Payment declined reason="insufficient_funds"
2024-11-22T01:00:03Z INFO [OrderService/order-123] Order cancelled reason="payment_failed"
```

**Best for:**
- Debugging specific issues
- Understanding event sequences
- Auditing user actions
- Compliance requirements

### 2. Metrics - How Much/How Many

**Purpose:** Aggregate measurements over time

```
# Counter - Total orders
orders_total{status="completed"} 1547
orders_total{status="failed"} 23

# Gauge - Current queue depth
order_queue_depth 42

# Histogram - Processing time distribution
order_processing_seconds{quantile="0.5"} 0.123
order_processing_seconds{quantile="0.99"} 0.856
```

**Best for:**
- System health monitoring
- Performance tracking
- Capacity planning
- SLA compliance

### 3. Traces - Where Time Was Spent

**Purpose:** Request flow through distributed system

```
Trace ID: abc123-def456
Span: ProcessOrder [290ms total]
  ├─ OrderService.CreateOrder [10ms]
  ├─ InventoryService.Reserve [50ms]
  ├─ PaymentService.Charge [200ms]  ← Bottleneck!
  └─ NotificationService.Send [30ms]
```

**Best for:**
- Finding performance bottlenecks
- Understanding system dependencies
- Debugging distributed workflows
- Optimizing response times

---

## 🏗️ Restate's Built-In Observability

### Automatic Instrumentation

Restate automatically provides:

**1. Structured Logs**
- Every handler invocation logged
- Request/response payloads
- Execution duration
- Error details

**2. OpenTelemetry Traces**
- Spans for each handler
- Trace context propagation
- Parent-child relationships
- Automatic error recording

**3. Prometheus Metrics**
- Invocation counts
- Duration histograms
- Queue depths
- Replay metrics

### Context-Aware Logging

```go
func (OrderService) ProcessOrder(
    ctx restate.ObjectContext,
    order Order,
) (OrderResult, error) {
    // Automatic context: service name, handler name, invocation ID
    ctx.Log().Info("Processing order",
        "orderId", restate.Key(ctx),
        "items", len(order.Items),
        "total", order.Total)
    
    // Logs include:
    // - Timestamp
    // - Log level
    // - Service: OrderService
    // - Key: order-123
    // - Invocation ID: inv_abc123
    // - Custom fields: orderId, items, total
}
```

---

## 🪵 Structured Logging Best Practices

### Use Appropriate Log Levels

```go
// DEBUG - Development details (disabled in production)
ctx.Log().Debug("Cache lookup", "key", cacheKey, "hit", true)

// INFO - Normal operations
ctx.Log().Info("Order created", "orderId", orderID, "total", total)

// WARN - Potential issues, but system continues
ctx.Log().Warn("Retrying payment", 
    "attempt", 3, 
    "maxAttempts", 5,
    "error", err)

// ERROR - Errors requiring attention
ctx.Log().Error("Payment gateway timeout",
    "orderId", orderID,
    "gateway", "stripe",
    "error", err,
    "duration", duration)
```

### Rich Context in Every Log

```go
// ❌ BAD - No context
ctx.Log().Error("Failed")

// ✅ GOOD - Rich context for debugging
ctx.Log().Error("Payment processing failed",
    "orderId", orderID,
    "customerId", customerID,
    "amount", amount,
    "currency", currency,
    "paymentMethod", method,
    "gatewayResponse", response,
    "attemptNumber", attempt,
    "error", err)
```

### Consistent Field Naming

```go
// Use consistent field names across services
ctx.Log().Info("Order event",
    "orderId", orderID,        // Always "orderId" not "order_id" or "id"
    "customerId", customerID,  // Always "customerId"
    "timestamp", timestamp,    // Always "timestamp"
    "amount", amount)          // Always "amount"
```

---

## 📊 Metrics Collection

### Prometheus Integration

Restate exposes metrics at `/metrics` endpoint:

```bash
# Restate internal metrics
curl http://localhost:9091/metrics

# Your service metrics
curl http://localhost:9090/metrics
```

### Custom Application Metrics

```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    // Counter - Monotonically increasing
    ordersTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "orders_total",
            Help: "Total number of orders",
        },
        []string{"status"}, // Labels
    )
    
    // Gauge - Can go up or down
    activeOrders = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "orders_active",
            Help: "Number of orders being processed",
        },
    )
    
    // Histogram - Distribution of values
    orderValue = promauto.NewHistogram(
        prometheus.HistogramOpts{
            Name: "order_value_dollars",
            Help: "Order value distribution",
            Buckets: []float64{10, 50, 100, 500, 1000, 5000},
        },
    )
    
    // Summary - Similar to histogram
    processingDuration = promauto.NewSummaryVec(
        prometheus.SummaryOpts{
            Name: "order_processing_seconds",
            Help: "Order processing duration",
            Objectives: map[float64]float64{
                0.5: 0.05,   // 50th percentile ±5%
                0.9: 0.01,   // 90th percentile ±1%
                0.99: 0.001, // 99th percentile ±0.1%
            },
        },
        []string{"status"},
    )
)

func (OrderService) ProcessOrder(
    ctx restate.ObjectContext,
    order Order,
) (OrderResult, error) {
    start := time.Now()
    activeOrders.Inc() // Increment active orders
    defer activeOrders.Dec() // Decrement when done
    
    result, err := processOrder(ctx, order)
    
    duration := time.Since(start).Seconds()
    status := "success"
    if err != nil {
        status = "failed"
    }
    
    // Record metrics
    ordersTotal.WithLabelValues(status).Inc()
    orderValue.Observe(order.Total)
    processingDuration.WithLabelValues(status).Observe(duration)
    
    return result, err
}
```

### Metric Naming Conventions

```
# Format: <namespace>_<subsystem>_<name>_<unit>
restate_orders_total           # Counter
restate_orders_active          # Gauge
restate_order_processing_seconds  # Duration
restate_order_value_dollars    # Histogram

# With labels
restate_orders_total{status="completed"}
restate_orders_total{status="failed"}
```

---

## 🔍 Distributed Tracing

### OpenTelemetry Integration

Restate automatically creates spans:

```
Trace: Process Order (trace_id: abc123)
├─ Span: OrderService/ProcessOrder [duration: 290ms]
│  ├─ Span: restate.journal.Run [duration: 50ms]
│  │  └─ Operation: Reserve inventory
│  ├─ Span: restate.call.Object [duration: 200ms]
│  │  └─ Target: PaymentService/Charge
│  │     └─ Span: PaymentService/Charge [duration: 195ms]
│  │        └─ Span: restate.journal.Run [duration: 180ms]
│  │           └─ Operation: Call Stripe API
│  └─ Span: restate.call.Service [duration: 30ms]
│     └─ Target: NotificationService/Send
│        └─ Span: NotificationService/Send [duration: 25ms]
```

### Configure Tracing

```bash
# Enable OpenTelemetry export
export RESTATE_OBSERVABILITY__TRACING__ENDPOINT=http://jaeger:4317
export RESTATE_OBSERVABILITY__TRACING__JSON=false

# Start Restate with tracing
docker run -e RESTATE_OBSERVABILITY__TRACING__ENDPOINT=http://jaeger:4317 \
  restatedev/restate:latest
```

### Custom Spans (Optional)

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
)

func complexOperation(ctx restate.RunContext) error {
    tracer := otel.Tracer("myservice")
    _, span := tracer.Start(ctx, "ComplexOperation")
    defer span.End()
    
    // Your complex logic here
    span.SetAttributes(
        attribute.String("operation", "data_processing"),
        attribute.Int("records", 1000),
    )
    
    return nil
}
```

---

## 🐛 Debugging Workflows

### Restate Invocation Journal

Every invocation has a journal showing:

```bash
# Get invocation details
curl http://localhost:9070/invocations/{invocation_id}

# Response shows:
{
  "id": "inv_abc123",
  "target": "OrderService/order-123/ProcessOrder",
  "status": "completed",
  "journal": [
    {"index": 0, "type": "GetState", "key": "order"},
    {"index": 1, "type": "Run", "name": "reserveInventory"},
    {"index": 2, "type": "Call", "target": "PaymentService/Charge"},
    {"index": 3, "type": "SetState", "key": "order", "value": "..."},
  ]
}
```

### Workflow Status Queries

```go
// Add status query handler
func (OrderOrchestrator) GetStatus(
    ctx restate.WorkflowSharedContext,
    _ restate.Void,
) (WorkflowStatus, error) {
    state, err := restate.Get[WorkflowState](ctx, "state")
    if err != nil {
        return WorkflowStatus{}, err
    }
    
    return WorkflowStatus{
        CurrentStep: state.CurrentStep,
        Progress: state.Progress,
        StartedAt: state.StartedAt,
        UpdatedAt: time.Now(),
    }, nil
}

// Query from external system
status := GET /OrderOrchestrator/order-123/GetStatus
// Returns current workflow state without blocking
```

---

## 📈 Dashboard Design

### Key Metrics to Monitor

**1. Business Metrics**
- Order success rate
- Revenue per hour
- Customer signups
- Feature usage

**2. Performance Metrics**
- Request latency (p50, p95, p99)
- Throughput (requests/second)
- Error rate
- Queue depth

**3. System Metrics**
- CPU usage
- Memory usage
- Disk I/O
- Network traffic

**4. Restate Metrics**
- Active invocations
- Journal replay count
- State size
- Timer backlog

### Sample Grafana Dashboard

```
┌─────────────────────────────────────────────┐
│ Order Processing Dashboard                  │
├─────────────────────────────────────────────┤
│ [📊 Orders/sec] [✅ Success Rate] [⏱️ P99]  │
│     125            98.5%           245ms    │
├─────────────────────────────────────────────┤
│ Order Status Distribution                   │
│ [████████████████████] Completed: 1,547     │
│ [██] Failed: 23                             │
│ [████] Processing: 42                       │
├─────────────────────────────────────────────┤
│ Processing Time (P50, P95, P99)             │
│ [Line graph showing latency over time]      │
├─────────────────────────────────────────────┤
│ Error Rate by Type                          │
│ payment_declined: 15                        │
│ inventory_unavailable: 5                    │
│ timeout: 3                                  │
└─────────────────────────────────────────────┘
```

---

## ⚠️ Common Pitfalls

### Anti-Pattern 1: Over-Logging

```go
// ❌ BAD - Too noisy
ctx.Log().Debug("Entering function")
ctx.Log().Debug("Setting variable x")
ctx.Log().Debug("Variable x =", x)
ctx.Log().Debug("Calling service")
ctx.Log().Debug("Service returned")
ctx.Log().Debug("Exiting function")
```

### Anti-Pattern 2: Logging Sensitive Data

```go
// ❌ BAD - Logs PII and secrets!
ctx.Log().Info("User login",
    "email", email,
    "password", password,      // NEVER!
    "ssn", ssn,               // NEVER!
    "creditCard", cardNumber) // NEVER!

// ✅ GOOD - Hash or omit sensitive data
ctx.Log().Info("User login",
    "email", email,
    "userIdHash", hash(userID))
```

### Anti-Pattern 3: No Error Context

```go
// ❌ BAD
if err != nil {
    ctx.Log().Error("Error occurred")
    return err
}

// ✅ GOOD
if err != nil {
    ctx.Log().Error("Failed to process payment",
        "orderId", orderID,
        "paymentMethod", method,
        "amount", amount,
        "error", err)
    return fmt.Errorf("payment processing failed: %w", err)
}
```

---

## ✅ Best Practices Summary

### Logging
1. **Use structured logging** with key-value pairs
2. **Add rich context** to every log
3. **Use appropriate log levels**
4. **Never log sensitive data**
5. **Use consistent field names**

### Metrics
1. **Track business and technical metrics**
2. **Use appropriate metric types** (counter, gauge, histogram)
3. **Follow naming conventions**
4. **Add relevant labels**
5. **Don't create too many time series**

### Tracing
1. **Enable OpenTelemetry** in Restate
2. **Let Restate auto-instrument** your code
3. **Add custom spans** only when needed
4. **Sample traces** in high-volume systems
5. **Use trace IDs** for correlation

### Dashboards
1. **Show metrics that matter** to your business
2. **Use SLI/SLO-based alerts**
3. **Design for quick troubleshooting**
4. **Include both real-time and historical views**
5. **Test alerts before production**

---

## 🚀 Next Steps

You now understand observability patterns!

👉 **Continue to [Hands-On Tutorial](./02-hands-on.md)**

Build an observable order processing system!

---

**Questions?** Review this document or check the [module README](./README.md).
