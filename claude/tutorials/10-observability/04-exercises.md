# Exercises: Observability Practice

> **Practice building observable systems**

## 🎯 Learning Objectives

Practice adding comprehensive observability to Restate applications.

---

## Exercise 1: Add Metrics to Existing Service ⭐

**Goal:** Add Prometheus metrics to a service

**Scenario:** You have a simple user registration service. Add metrics to track:
- Total registrations (counter)
- Email verification rate (counter with status label)
- Registration processing time (histogram)

**Requirements:**
1. Add counters for success/failure
2. Track verification email sends
3. Record processing duration
4. Expose metrics on `/metrics`

**Success Criteria:**
- [ ] Metrics endpoint working
- [ ] All events tracked
- [ ] Appropriate metric types used
- [ ] Labels applied correctly

---

## Exercise 2: Structured Logging ⭐

**Goal:** Improve logging with structured fields

**Scenario:** Add comprehensive structured logging to an order fulfillment service.

**Requirements:**
1. Log with rich context (orderID, customerID, etc.)
2. Use appropriate log levels
3. Add correlation IDs
4. Never log sensitive data (credit cards, passwords)

**Success Criteria:**
- [ ] All logs structured
- [ ] Consistent field naming
- [ ] Proper log levels
- [ ] Easy to query/filter

---

## Exercise 3: Custom Dashboard ⭐⭐

**Goal:** Create a Grafana dashboard

**Scenario:** Build a dashboard for the order processing system showing:
- Orders per minute
- Success rate
- P50, P95, P99 latency
- Error rate by type

**Requirements:**
1. Set up Prometheus
2. Create Grafana dashboard
3. Add 4-6 panels
4. Include alerts for SLA violations

**Success Criteria:**
- [ ] Dashboard shows real-time data
- [ ] Multiple visualization types
- [ ] Alerts configured
- [ ] Useful for debugging

---

## Exercise 4: Distributed Tracing ⭐⭐⭐

**Goal:** Implement OpenTelemetry traces

**Scenario:** Add tracing to a multi-service workflow (order → inventory → payment → shipping).

**Requirements:**
1. Configure OpenTelemetry exporter
2. Set up Jaeger for trace visualization
3. Add custom spans for key operations
4. Trace across service boundaries

**Success Criteria:**
- [ ] End-to-end traces visible
- [ ] Service dependencies clear
- [ ] Bottlenecks identifiable
- [ ] Error traces captured

---

## Exercise 5: SLI/SLO Monitoring ⭐⭐

**Goal:** Define and monitor SLIs/SLOs

**Scenario:** Set up SLI/SLO monitoring for an API service.

**SLOs:**
- 99.9% availability
- P95 latency < 500ms
- Error rate < 0.1%

**Requirements:**
1. Define SLIs (metrics that measure SLOs)
2. Create recording rules
3. Set up alerts for SLO violations
4. Build error budget dashboard

**Success Criteria:**
- [ ] SLIs accurately measured
- [ ] Alerts fire when SLOs violated
- [ ] Error budget tracked
- [ ] Dashboard shows compliance

---

## Exercise 6: Log Aggregation ⭐⭐

**Goal:** Set up centralized logging

**Scenario:** Aggregate logs from multiple services using Loki or Elasticsearch.

**Requirements:**
1. Configure log shipping
2. Set up log retention policy
3. Create useful log queries
4. Build log-based alerts

**Success Criteria:**
- [ ] All service logs aggregated
- [ ] Logs searchable
- [ ] Retention policy working
- [ ] Alerts on log patterns

---

## 💡 Implementation Tips

### Adding Metrics

```go
var requestDuration = promauto.NewHistogramVec(
    prometheus.HistogramOpts{
        Name: "service_request_duration_seconds",
        Help: "Request duration distribution",
        Buckets: prometheus.DefBuckets,
    },
    []string{"method", "status"},
)

func trackMetrics(method string, fn func() error) error {
    start := time.Now()
    err := fn()
    
    status := "success"
    if err != nil {
        status = "error"
    }
    
    requestDuration.WithLabelValues(method, status).
        Observe(time.Since(start).Seconds())
    
    return err
}
```

### Structured Logging

```go
ctx.Log().Info("Operation completed",
    "operation", "createUser",
    "userId", userID,
    "duration", duration,
    "success", true)
```

### OpenTelemetry Setup

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func initTracer() {
    exporter, _ := otlptrace.New(context.Background(),
        otlptrace.WithEndpoint("localhost:4317"))
    
    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter))
    
    otel.SetTracerProvider(tp)
}
```

## 📚 Resources

- [Concepts](./01-concepts.md)
- [Hands-On](./02-hands-on.md)
- [Validation](./03-validation.md)
- [Prometheus Documentation](https://prometheus.io/docs/)
- [OpenTelemetry Go](https://opentelemetry.io/docs/languages/go/)

---

**Happy monitoring!** 📊
