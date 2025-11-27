# Module 03 - Concurrent Order Processing

This directory contains a complete, working example of concurrent order processing using Restate futures.

## 📂 Files

- `main.go` - Server initialization and service registration
- `order_service.go` - Parallel and sequential order processors
- `supporting_services.go` - Supporting services (inventory, payment, fraud, shipping)
- `go.mod` - Go module dependencies

## 🚀 Quick Start

```bash
# Build
go mod tidy
go build -o order-service

# Run
./order-service
```

The service will start on port 9090.

## 📋 Register with Restate

```bash
curl -X POST http://localhost:8080/deployments \
  -H 'Content-Type: application/json' \
  -d '{"uri": "http://localhost:9090"}'
```

## 🧪 Test

### Test Sequential (Slow) Version

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

Expected time: **~450-500ms**

### Test Parallel (Fast) Version

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

Expected time: **~150-200ms** 🚀

**Speedup: 2.5-3x faster!**

## 🎓 What This Demonstrates

### Fan-Out Pattern
All operations start in parallel:
- Check inventory (2 warehouses)
- Authorize payment
- Check fraud risk
- Calculate shipping (2 carriers)

### Fan-In Pattern
Collect and aggregate results:
- Choose available warehouse
- Validate payment and fraud
- Select best shipping option

### Resilience
- Falls back to backup warehouse if main unavailable
- Continues despite partial failures
- Clear error messages

## 📊 Performance Comparison

| Version | Execution | Time | Operations |
|---------|-----------|------|------------|
| Sequential | One by one | ~450ms | Inventory → Payment → Fraud → Shipping |
| Parallel | Simultaneous | ~150ms | All at once, wait for slowest |

## 🔍 Understanding the Code

### Sequential Flow (OrderServiceSlow)
```go
// Each operation blocks until complete
inv := service.Call()      // Wait 80ms
payment := service.Call()  // Wait 120ms
fraud := service.Call()    // Wait 150ms
shipping := service.Call() // Wait 100ms
// Total: 450ms
```

### Parallel Flow (OrderService)
```go
// Start all operations
fut1 := service.CallAsync()
fut2 := service.CallAsync()
fut3 := service.CallAsync()

// Wait for all to complete
for fut, err := range restate.Wait(ctx, fut1, fut2, fut3) {
    // Process as results arrive
}
// Total: max(all operations) = 150ms
```

## 💡 Key Learning Points

1. **Futures Enable Parallelism**
   - `RequestFuture()` starts async operation
   - Returns immediately
   - `restate.Wait()` collects results

2. **Journaled Execution**
   - All futures journaled
   - No re-execution on replay
   - Deterministic results

3. **Partial Failure Handling**
   - Check multiple warehouses
   - If one fails, use another
   - Graceful degradation

4. **Smart Aggregation**
   - Collect all shipping quotes
   - Choose best (lowest cost)
   - Flexible decision logic

## 🎯 Next Steps

- Complete [validation tests](../03-validation.md)
- Try [exercises](../04-exercises.md)
- Build your own concurrent service

---

**Questions?** See the main [hands-on tutorial](../02-hands-on.md)!
