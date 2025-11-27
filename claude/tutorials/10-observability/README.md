# Module 10: Observability

> **Monitor, debug, and understand your Restate applications**

## 🎯 Learning Objectives

By completing this module, you will:
- ✅ Implement structured logging
- ✅ Add distributed tracing
- ✅ Collect and export metrics
- ✅ Build observability dashboards
- ✅ Debug distributed workflows
- ✅ Monitor system health

## 📚 Module Content

### 1. Concepts (~20 min)
- Observability pillars (logs, metrics, traces)
- Restate's built-in observability
- OpenTelemetry integration
- Logging best practices
- Monitoring strategies

### 2. Hands-On (~30 min)
- Add structured logging
- Configure OpenTelemetry
- Export metrics to Prometheus
- Visualize with Grafana
- Trace distributed workflows

### 3. Best Practices (~15 min)
- Log levels and context
- Metric naming conventions
- Trace sampling strategies
- Dashboard design
- Alerting rules

## 🎯 Key Concepts

### The Three Pillars of Observability

**1. Logs** - What happened
```
2024-11-22T01:00:00Z INFO [OrderService] Order created orderId=123
2024-11-22T01:00:01Z INFO [PaymentService] Payment processing orderId=123 amount=99.99
2024-11-22T01:00:02Z ERROR [PaymentService] Payment failed orderId=123 reason=declined
```

**2. Metrics** - How much/how many
```
order_total{status="completed"}  1547
order_total{status="failed"}      23
payment_latency_ms{p99}           245
```

**3. Traces** - Where time was spent
```
Trace: order-123
├─ OrderService.Create (10ms)
├─ InventoryService.Reserve (50ms)
├─ PaymentService.Charge (200ms)
└─ NotificationService.Send (30ms)
Total: 290ms
```

## 🏗️ Restate's Built-In Observability

### Structured Logging

```go
func (OrderService) CreateOrder(
    ctx restate.ObjectContext,
    order Order,
) (OrderResult, error) {
    ctx.Log().Info("Creating order",
        "orderId", order.ID,
        "customer", order.CustomerID,
        "total", order.Total)
    
    // ... processing ...
    
    ctx.Log().Error("Payment failed",
        "orderId", order.ID,
        "error", err,
        "retryCount", retries)
}
```

### OpenTelemetry Integration

Restate automatically:
- ✅ Creates spans for each handler invocation
- ✅ Propagates trace context
- ✅ Records handler duration
- ✅ Captures errors and status

### Metrics Endpoints

```bash
# Restate metrics
curl http://localhost:9091/metrics

# Your service metrics
curl http://localhost:9090/metrics
```

## 🚀 Quick Example

### Add Logging

```go
type OrderService struct{}

func (OrderService) ProcessOrder(
    ctx restate.ObjectContext,
    order Order,
) (OrderResult, error) {
    orderID := restate.Key(ctx)
    
    // Structured logging with context
    ctx.Log().Info("Processing order started",
        "orderId", orderID,
        "items", len(order.Items),
        "total", order.Total)
    
    // Log important steps
    ctx.Log().Debug("Reserving inventory",
        "orderId", orderID,
        "itemCount", len(order.Items))
    
    inventory, err := inventoryService.Reserve(order.Items)
    if err != nil {
        ctx.Log().Error("Inventory reservation failed",
            "orderId", orderID,
            "error", err,
            "items", order.Items)
        return OrderResult{}, err
    }
    
    ctx.Log().Info("Processing order completed",
        "orderId", orderID,
        "duration", time.Since(start))
    
    return OrderResult{Success: true}, nil
}
```

### Export Metrics

```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    ordersTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "orders_total",
            Help: "Total number of orders",
        },
        []string{"status"},
    )
    
    orderDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "order_duration_seconds",
            Help: "Order processing duration",
        },
        []string{"status"},
    )
)

func (OrderService) ProcessOrder(ctx restate.ObjectContext, order Order) {
    start := time.Now()
    
    result, err := processOrder(ctx, order)
    
    duration := time.Since(start).Seconds()
    status := "success"
    if err != nil {
        status = "failed"
    }
    
    ordersTotal.WithLabelValues(status).Inc()
    orderDuration.WithLabelValues(status).Observe(duration)
}
```

## ⚠️ Common Pitfalls

### Anti-Pattern 1: Over-Logging

```go
// ❌ BAD - Too verbose, adds noise
ctx.Log().Debug("Starting function")
ctx.Log().Debug("Variable x =", x)
ctx.Log().Debug("Calling service")
ctx.Log().Debug("Service returned")
ctx.Log().Debug("Ending function")
```

### Anti-Pattern 2: Missing Context

```go
// ❌ BAD - No context for debugging
ctx.Log().Error("Payment failed")
// What order? What amount? Why?
```

### Anti-Pattern 3: No Metrics

```go
// ❌ BAD - Can't monitor performance
func ProcessOrder() {
    // No metrics, no visibility
}
```

## ✅ Best Practices

### 1. Log with Context

```go
// ✅ GOOD - Rich context
ctx.Log().Error("Payment processing failed",
    "orderId", orderID,
    "customerId", customerID,
    "amount", amount,
    "paymentMethod", method,
    "error", err,
    "attemptNumber", attempt)
```

### 2. Use Appropriate Log Levels

```go
ctx.Log().Debug("Cache hit")              // Development debugging
ctx.Log().Info("Order created")            // Normal operations
ctx.Log().Warn("Retry attempt 3/5")       // Potential issues
ctx.Log().Error("Payment gateway timeout") // Errors requiring attention
```

### 3. Track Key Metrics

```go
// Business metrics
ordersTotal.Inc()
revenueTotal.Add(order.Total)

// Performance metrics
processingDuration.Observe(duration)
queueDepth.Set(float64(len(queue)))

// Error rates
errorsTotal.WithLabelValues("payment_failed").Inc()
```

## 📊 Monitoring Dashboard Example

**Key Metrics to Track:**
- Order success rate
- Average processing time
- Error rate by type
- Active workflow count
- Service availability

## 🔗 Tools Integration

- **Prometheus** - Metrics collection
- **Grafana** - Visualization
- **Jaeger/Tempo** - Distributed tracing
- **Loki** - Log aggregation
- **OpenTelemetry** - Unified observability

## 📈 Success Criteria

You've mastered observability when:
- [x] All services have structured logging
- [x] Key metrics are tracked
- [x] Distributed traces work end-to-end
- [x] Dashboards provide visibility
- [x] Alerts catch issues proactively

## 🎓 Learning Path

**Current Module:** Observability  
**Previous:** [Microservices](../09-microservices/README.md)  
**Next:** [Module 11 - Security](../11-security/README.md)

---

**Ready to monitor your applications!** 🔍
