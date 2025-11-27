# Observability Integration Guide for rea-05/services

**Target File:** `rea-05/services/svcs.go`  
**Framework:** `github.com/pithomlabs/rea`  
**Focus:** Metrics collection, tracing, and observability hooks

---

## 1. Overview of rea Framework Observability

The `rea` framework provides three main observability components:

### 1.1 MetricsCollector
Prometheus-compatible metrics for monitoring service health:
- **Counters:** `InvocationTotal`, `InvocationErrors`, `CompensationTotal`
- **Gauges:** `ActiveInvocations`, `StateSize`
- **Histograms:** `InvocationDuration`, `CompensationDuration`

### 1.2 TracingContext
OpenTelemetry-compatible distributed tracing:
- Span creation with parent-child relationships
- Deterministic span/trace ID generation
- Error tracking and status reporting

### 1.3 ObservabilityHooks
Custom event callbacks for logging and monitoring:
- `OnInvocationStart/End` - Track handler execut execution lifecycle
- `OnStateGet/Set/Clear` - Track state operations
- `OnSagaStart`, `OnCompensationStart/End` - Track saga operations
- `OnError` - Centralized error handling

---

## 2. Current svcs.go State

**Existing Code Structure:**
```go
package main

import (
    "context"
    "fmt"
    "os"
    "svcs/models"  // ❌ Should be rea-05/services/models
    "time"

    "github.com/pithomlabs/rea"
    restate "github.com/restatedev/sdk-go"
    "github.com/restatedev/sdk-go/server"
)

type ShippingService struct{}
type UserSession struct{}
type OrderFulfillmentWorkflow struct{}

func main() {
    server.NewRestate().
        Bind(restate.Reflect(ShippingService{})).
        Bind(restate.Reflect(UserSession{})).
        Bind(restate.Reflect(OrderFulfillmentWorkflow{})).
        Start(context.Background(), "0.0.0.0:9080")
}
```

**Issues to Fix:**
1. ❌ Import path `svcs/models` should be `rea-05/services/models`
2. ❌ No observability components initialized
3. ❌ Services not instrumented for metrics/tracing

---

## 3. Integration Strategy

### Approach: Wrapper Pattern with Instrumented Handlers

Instead of modifying every handler, we'll:
1. Initialize observability components in `main()`
2. Wrap service structs with observability context
3. Use `InstrumentedServiceClient` for inter-service calls
4. Export metrics endpoint

---

## 4. Step-by-Step Integration

### Step 1: Fix Import Path

```go
import (
    "context"
    "fmt"
    "log/slog"
    "net/http"
    "os"
    "rea-05/services/models"  // ✅ Correct path
    "time"

    "github.com/pithomlabs/rea"
    restate "github.com/restatedev/sdk-go"
    "github.com/restatedev/sdk-go/server"
)
```

### Step 2: Add Observable Service Wrapper

```go
// ObservableServices wraps all services with observability
type ObservableServices struct {
    metrics *rea.MetricsCollector
    tracing *rea.TracingContext
    hooks   *rea.ObservabilityHooks
    logger  *slog.Logger
}

func NewObservableServices() *ObservableServices {
    logger := slog.Default()
    
    return &ObservableServices{
        metrics: rea.NewMetricsCollector(),
        hooks:   rea.DefaultObservabilityHooks(logger),
        logger:  logger,
    }
}
```

### Step 3: Instrument ShippingService

**Option A: Manual Instrumentation (Lightweight)**

```go
func (ShippingService) InitiateShipment(
    ctx restate.Context,
    shipment models.ShipmentRequest,
) (bool, error) {
    // Track invocation start
    start := time.Now()
    
    ctx.Log().Info("ShippingService received request", "order_id", shipment.OrderID)
    
    cfg := rea.RunConfig{
        MaxRetries:    3,
        InitialDelay:  100 * time.Millisecond,
        MaxDelay:      2 * time.Second,
        BackoffFactor: 2.0,
        Name:          "InitiateShipment",
    }
    
    success, err := rea.RunWithRetry(ctx, cfg, func(ctx restate.RunContext) (bool, error) {
        time.Sleep(100 * time.Millisecond)
        if shipment.OrderID == "FAIL_SHIP" {
            ctx.Log().Error("Shipping failed", "order_id", shipment.OrderID)
            return false, fmt.Errorf("external shipping API error")
        }
        ctx.Log().Info("Shipment registered", "order_id", shipment.OrderID)
        return true, nil
    })
    
    // Record metrics (accessed via global or context)
    duration := time.Since(start)
    // metrics.RecordInvocation("ShippingService", "InitiateShipment", duration, err)
    
    if err != nil {
        return false, err
    }
    if !success {
        return false, restate.TerminalError(
            fmt.Errorf("shipping company rejected shipment"), 400)
    }
    
    return true, nil
}
```

**Option B: Full Instrumentation with TracingContext**

```go
type InstrumentedShippingService struct {
    obs *ObservableServices
}

func (s *InstrumentedShippingService) InitiateShipment(
    ctx restate.Context,
    shipment models.ShipmentRequest,
) (bool, error) {
    // Create tracing context
    tracing := rea.NewTracingContext(ctx)
    
    // Start span
    span := tracing.StartSpan("InitiateShipment", map[string]string{
        "service":  "ShippingService",
        "order_id": shipment.OrderID,
    })
    defer tracing.EndSpan(span, nil)
    
    // Track active invocations
    s.obs.metrics.IncrementActiveInvocations("ShippingService")
    defer s.obs.metrics.DecrementActiveInvocations("ShippingService")
    
    start := time.Now()
    
    // Fire hook
    if s.obs.hooks != nil && s.obs.hooks.OnInvocationStart != nil {
        s.obs.hooks.OnInvocationStart("ShippingService", "InitiateShipment", shipment)
    }
    
    cfg := rea.RunConfig{
        MaxRetries:    3,
        InitialDelay:  100 * time.Millisecond,
        MaxDelay:      2 * time.Second,
        BackoffFactor: 2.0,
        Name:          "InitiateShipment",
    }
    
    success, err := rea.RunWithRetry(ctx, cfg, func(ctx restate.RunContext) (bool, error) {
        time.Sleep(100 * time.Millisecond)
        if shipment.OrderID == "FAIL_SHIP" {
            return false, fmt.Errorf("external shipping API error")
        }
        return true, nil
    })
    
    // Record metrics and fire hooks
    duration := time.Since(start)
    s.obs.metrics.RecordInvocation("ShippingService", "InitiateShipment", duration, err)
    
    if s.obs.hooks != nil && s.obs.hooks.OnInvocationEnd != nil {
        var output interface{} = success
        s.obs.hooks.OnInvocationEnd("ShippingService", "InitiateShipment", output, err, duration)
    }
    
    if err != nil {
        tracing.EndSpan(span, err)
        return false, err
    }
    
    if !success {
        finalErr := restate.TerminalError(fmt.Errorf("shipping rejected"), 400)
        tracing.EndSpan(span, finalErr)
        return false, finalErr
    }
    
    return true, nil
}
```

### Step 4: Instrument Inter-Service Calls (UserSession)

Use `InstrumentedServiceClient` for automatic observability:

```go
type InstrumentedUserSession struct {
    obs *ObservableServices
}

func (s *InstrumentedUserSession) Checkout(
    ctx restate.ObjectContext,
    orderID string,
) (bool, error) {
    userID := restate.Key(ctx)
    
    // Deduplication check
    dedupKey := fmt.Sprintf("checkout:exec:%s", orderID)
    executed, err := restate.Get[bool](ctx, dedupKey)
    if err == nil && executed {
        return true, nil
    }
    
    // Create Awakeable for payment
    awakeable := restate.Awakeable[models.PaymentReceipt](ctx)
    id := awakeable.Id()
    
    // Notify payment gateway
    _, err = restate.Run(ctx, func(ctx restate.RunContext) (bool, error) {
        ctx.Log().Info("Notifying payment gateway", "awakeable_id", id)
        return true, nil
    })
    if err != nil {
        return false, err
    }
    
    // Wait for payment
    receipt, err := awakeable.Result()
    if err != nil {
        return false, restate.TerminalError(fmt.Errorf("payment failed: %v", err), 500)
    }
    
    // Mark as executed
    restate.Set(ctx, dedupKey, true)
    
    // Clear session state
    if err := rea.ClearAll(ctx); err != nil {
        return false, err
    }
    
    // Create instrumented client for workflow invocation
    workflowClient := rea.NewInstrumentedClient(
        rea.WorkflowSend[models.Order](ctx, "OrderFulfillmentWorkflow", orderID, "Run"),
        s.obs.metrics,
        rea.NewTracingContext(ctx),
        s.obs.hooks,
    )
    
    // Launch workflow with full observability
    orderPayload := models.Order{
        OrderID:     orderID,
        UserID:      userID,
        Items:       "item-1,item-2",
        AmountCents: 10000,
    }
    
    // This automatically records metrics and creates trace spans
    workflowClient.Send(orderPayload)
    
    return true, nil
}
```

### Step 5: Update main() with Observability

```go
func main() {
    // Initialize observability
    obs := NewObservableServices()
    
    // Create instrumented services
    shippingService := &InstrumentedShippingService{obs: obs}
    userSession := &InstrumentedUserSession{obs: obs}
    workflowService := &InstrumentedOrderFulfillmentWorkflow{obs: obs}
    
    // Start metrics HTTP server (non-blocking)
    go serveMetrics(obs.metrics)
    
    // Start Restate server with instrumented services
    if err := server.NewRestate().
        Bind(restate.Reflect(shippingService)).
        Bind(restate.Reflect(userSession)).
        Bind(restate.Reflect(workflowService)).
        Start(context.Background(), "0.0.0.0:9080"); err != nil {
        fmt.Fprintf(os.Stderr, "Server failed: %v\n", err)
        os.Exit(1)
    }
}

// Serve Prometheus metrics
func serveMetrics(metrics *rea.MetricsCollector) {
    http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
        allMetrics := metrics.GetMetrics()
        
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(allMetrics)
    })
    
    obs.logger.Info("Metrics server listening", "port", 9091)
    if err := http.ListenAndServe(":9091", nil); err != nil {
        obs.logger.Error("Metrics server failed", "error", err)
    }
}
```

---

## 5. Metrics Exposed

**After integration, the following metrics will be available at `http://localhost:9091/metrics`:**

```json
{
  "invocation_total": {
    "ShippingService.InitiateShipment": 150,
    "UserSession.Checkout": 85,
    "OrderFulfillmentWorkflow.Run": 42
  },
  "invocation_errors": {
    "ShippingService.InitiateShipment": 5,
    "UserSession.Checkout": 2
  },
  "active_invocations": {
    "ShippingService": 3,
    "UserSession": 12
  },
  "invocation_duration": {
    "ShippingService.InitiateShipment": [0.125, 0.110, 0.132, ...]
  }
}
```

---

## 6. Recommended Integration Level: **Lightweight**

### Why Lightweight?

1. **Restate already provides durability** - Don't need full instrumentation of every handler
2. **RunWithRetry already tracks retries** - Metrics built-in
3. **Restate SDK logs extensively** - Don't duplicate logging

### Lightweight Integration:

```go
// Global observability (initialized in main)
var globalMetrics *rea.MetricsCollector
var globalLogger *slog.Logger

func init() {
    globalMetrics = rea.NewMetricsCollector()
    globalLogger = slog.Default()
}

// Add simple metrics wrapper
func recordInvocation(service, handler string, duration time.Duration, err error) {
    if globalMetrics != nil {
        globalMetrics.RecordInvocation(service, handler, duration, err)
    }
}

// Use in handlers:
func (ShippingService) InitiateShipment(...) (bool, error) {
    start := time.Now()
    defer func() {
        recordInvocation("ShippingService", "InitiateShipment", time.Since(start), err)
    }()
    
    // Existing handler logic...
}
```

---

## 7. Testing Observability

```bash
# Start services
cd rea-05/services
go run svcs.go

# In another terminal, trigger some invocations
curl -X POST http://localhost:8080/api/v1/user/user123/checkout \\
  -H "X-API-Key: super-secret-ingress-key" \\
  -H "Idempotency-Key: order-abc-123"

# Check metrics
curl http://localhost:9091/metrics | jq .

# Expected output:
{
  "invocation_total": {
    "UserSession.Checkout": 1
  },
  "invocation_duration": {
    "UserSession.Checkout": [0.245]
  }
}
```

---

## 8. Summary & Recommendation

**For rea-05/services/svcs.go, I recommend:**

### ✅ DO (Minimal):
1. Fix import path to `rea-05/services/models`
2. Add global `MetricsCollector` and `ObservabilityHooks`
3. Record metrics at handler entry/exit using defer
4. Export metrics on separate HTTP port (`:9091`)

### ⚠️ OPTIONAL (Advanced):
1. Full instrumentation with `TracingContext` for distributed tracing
2. Custom hooks for alerting/monitoring
3. `InstrumentedServiceClient` for inter-service calls

### ❌ DON'T:
1. Instrument every line of code (Restate already journals)
2. Block on metrics collection (use async/buffered)
3. Duplicate Restate's built-in logging

**Next Steps:**
1. Should I implement the lightweight integration?
2. Do you want full instrumentation with tracing?
3. Focus only on fixing the import path first?
