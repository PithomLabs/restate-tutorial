# Validation: Testing Observability

> **Verify your observable order processing system**

## 🎯 Validation Goals

- ✅ Verify structured logging works
- ✅ Confirm metrics are collected
- ✅ Test status queries
- ✅ Validate metric accuracy
- ✅ Check log context

## 🧪 Test Scenarios

### Scenario 1: Successful Order with Logs

**Test:** Process order and verify logs

```bash
curl -X POST http://localhost:8080/OrderService/order-logs-001/ProcessOrder \
  -H 'Content-Type: application/json' \
  -d '{
    "id": "order-logs-001",
    "customerId": "cust-123",
    "items": ["laptop", "mouse"],
    "total": 1299.99
  }'
```

**Expected Logs:**
```
INFO Processing order started orderId=order-logs-001 customerId=cust-123 itemCount=2 total=1299.99
INFO Processing payment orderId=order-logs-001 amount=1299.99
INFO Payment successful orderId=order-logs-001
INFO Fulfilling order orderId=order-logs-001
INFO Order processing completed orderId=order-logs-001 duration=X.XXX
```

**Validation:**
- [ ] All logs have structured fields
- [ ] OrderID appears in every log
- [ ] Duration is logged
- [ ] No errors logged

### Scenario 2: Metrics Collection

**Test:** Process multiple orders and check metrics

```bash
# Process 5 orders
for i in {1..5}; do
  curl -X POST http://localhost:8080/OrderService/order-metric-$i/ProcessOrder \
    -H 'Content-Type: application/json' \
    -d "{\"id\":\"order-metric-$i\",\"customerId\":\"cust-$i\",\"items\":[\"item1\"],\"total\":50.00}"
done

# Check metrics
curl http://localhost:2112/metrics | grep orders
```

**Expected Metrics:**
```
orders_total{status="success"} 5
orders_active 0
order_processing_seconds_count{status="success"} 5
order_value_dollars_count 5
payment_attempts_total{result="success"} 5
```

**Validation:**
- [ ] `orders_total` shows 5 successful orders
- [ ] `orders_active` is 0 (all completed)
- [ ] `payment_attempts_total` matches orders
- [ ] Histogram buckets populated

### Scenario 3: Status Query

**Test:** Query order status without blocking

```bash
# Process order
curl -X POST http://localhost:8080/OrderService/order-status-001/ProcessOrder \
  -d '{"id":"order-status-001","customerId":"cust-123","items":["item"],"total":25.00}'

# Query status (concurrent handler, non-blocking)
curl -X POST http://localhost:8080/OrderService/order-status-001/GetStatus \
  -d '{}'
```

**Expected Response:**
```json
{
  "orderId": "order-status-001",
  "status": "completed",
  "message": "Order fulfilled: fulfillment-XXXXX",
  "timestamp": "2024-11-22T..."
}
```

**Validation:**
- [ ] Status returned immediately
- [ ] Status is "completed"
- [ ] Fulfillment ID present
- [ ] Timestamp included

### Scenario 4: Failed Order Metrics

**Test:** Trigger validation failure

```bash
# Order with no items (validation fails)
curl -X POST http://localhost:8080/OrderService/order-fail-001/ProcessOrder \
  -d '{"id":"order-fail-001","customerId":"cust-123","items":[],"total":0}'

# Check metrics
curl http://localhost:2112/metrics | grep -E "orders_total|payment_attempts"
```

**Expected:**
- Error log with context
- `orders_total{status="failed"}` incremented
- No payment attempt recorded

**Validation:**
- [ ] Error logged with details
- [ ] Failed order counted
- [ ] No payment attempted
- [ ] Terminal error returned

### Scenario 5: Concurrent Processing

**Test:** Multiple simultaneous orders

```bash
# Start 10 orders in parallel
for i in {1..10}; do
  curl -X POST http://localhost:8080/OrderService/concurrent-$i/ProcessOrder \
    -d "{\"id\":\"concurrent-$i\",\"customerId\":\"cust-$i\",\"items\":[\"item\"],\"total\":100}" &
done
wait

# Check active gauge was accurate
curl http://localhost:2112/metrics | grep orders_active
```

**Expected:**
- `orders_active` returns to 0
- All orders completed
- Metrics accurate

**Validation:**
- [ ] orders completed
- [ ] Active gauge back to 0
- [ ] No metric inconsistencies
- [ ] All logs present

## ✅ Validation Checklist

### Logging
- [ ] Structured logs with key-value pairs
- [ ] Rich context in every log
- [ ] Appropriate log levels used
- [ ] No sensitive data logged
- [ ] Consistent field naming

### Metrics
- [ ] All metric types working (counter, gauge, histogram)
- [ ] Metrics endpoint accessible
- [ ] Metric values accurate
- [ ] Labels applied correctly
- [ ] No metric cardinality explosion

### Observability
- [ ] Status queries work
- [ ] Metrics track business KPIs
- [ ] Logs correlate with metrics
- [ ] Performance data captured
- [ ] Error rates tracked

## 🎓 Success Criteria

Pass when:
- ✅ All test scenarios pass
- ✅ Logs are structured and informative
- ✅ Metrics accurately reflect system state
- ✅ Status queries provide visibility
- ✅ System is debuggable

## 🚀 Next Steps

👉 **Continue to [Exercises](./04-exercises.md)**
