# Concepts: Testing Restate Applications

> **Understanding test strategies for durable execution**

## 🎯 Why Testing is Different

### Traditional Application Testing

```go
func Add(a, b int) int {
    return a + b
}

func TestAdd(t *testing.T) {
    result := Add(2, 3)
    assert.Equal(t, 5, result)
}
```

**Characteristics:**
- ✅ Pure functions
- ✅ No side effects
- ✅ Deterministic
- ✅ Fast

### Restate Application Testing

```go
func (OrderService) ProcessOrder(
    ctx restate.Context,
    order Order,
) (OrderResult, error) {
    // Side effect: charge payment
    paymentID, _ := restate.Run(ctx, func(ctx restate.RunContext) (string, error) {
        return paymentAPI.Charge(order.Amount)
    })
    
    // State: save order
    restate.Set(ctx, "order", order)
    
    // Service call: reserve inventory
    restate.Service[bool](ctx, "Inventory", "Reserve").Request(order.Items)
    
    return OrderResult{PaymentID: paymentID}, nil
}
```

**Challenges:**
- ⚠️ Requires Restate context
- ⚠️ Contains side effects
- ⚠️ Calls external services
- ⚠️ Manages state
- ⚠️ May be long-running

## 📊 Testing Strategy Pyramid

```
        /\
       /  \
      / E2E\      ← Few, slow, high confidence
     /------\
    /  Integ \    ← Some, moderate speed
   /----------\
  /  Unit Tests\  ← Many, fast, focused
 /--------------\
```

### Unit Tests (70%)
- Test individual handlers
- Mock all dependencies
- Fast execution
- Focused on business logic

### Integration Tests (20%)
- Test multiple components together
- Use real Restate server
- Moderate speed
- Verify integration points

### End-to-End Tests (10%)
- Test complete user flows
- All real services
- Slow but comprehensive
- Production-like environment

## 🧪 Mock Context API

Restate provides a mock context for unit testing without a running server.

### Basic Usage

```go
import (
    "testing"
    restate "github.com/restatedev/sdk-go"
    "github.com/stretchr/testify/assert"
)

func TestGreeting(t *testing.T) {
    // Create mock context
    ctx := restate.NewMockContext()
    
    // Test handler
    greeter := Greeter{}
    result, err := greeter.Greet(ctx, "Alice")
    
    // Verify
    assert.NoError(t, err)
    assert.Equal(t, "Hello, Alice!", result)
}
```

### Mock Context Features

```go
// 1. State management
ctx := restate.NewMockContext()
restate.Set(ctx, "key", "value")
value, _ := restate.Get[string](ctx, "key")

// 2. Deterministic random
ctx := restate.NewMockContext(restate.WithSeed(42))
random := restate.Rand(ctx).Float64() // Always same value

// 3. Deterministic UUIDs
uuid := restate.UUID(ctx) // Predictable in tests

// 4. Instant sleep
ctx.MockSleep(1 * time.Hour) // Returns immediately

// 5. Mock service calls
ctx.MockService("PaymentService", "Charge").
    Returns("payment-123", nil)
```

## 🎭 Mocking Patterns

### Pattern 1: Mock External API Calls

```go
func TestPaymentProcessing(t *testing.T) {
    ctx := restate.NewMockContext()
    
    // Mock the payment API call
    ctx.MockRun("chargePayment").
        Returns("payment-123", nil)
    
    // Test
    service := OrderService{}
    result, err := service.ProcessOrder(ctx, order)
    
    assert.NoError(t, err)
    assert.Equal(t, "payment-123", result.PaymentID)
}
```

### Pattern 2: Mock Service-to-Service Calls

```go
func TestOrderWithInventory(t *testing.T) {
    ctx := restate.NewMockContext()
    
    // Mock inventory service response
    ctx.MockService("InventoryService", "Reserve").
        Request(items).
        Returns(true, nil)
    
    // Test
    result, err := OrderService{}.ProcessOrder(ctx, order)
    
    assert.NoError(t, err)
}
```

### Pattern 3: Mock Workflow Promises

```go
func TestApprovalWorkflow(t *testing.T) {
    ctx := restate.NewMockContext()
    
    // Mock promise resolution
    ctx.MockPromise("approval").
        ResolveWith(ApprovalDecision{
            Approved: true,
            Approver: "manager",
        })
    
    // Test
    result, err := ApprovalWorkflow{}.Run(ctx, doc)
    
    assert.Equal(t, "approved", result.Status)
}
```

## ✅ Unit Testing Best Practices

### 1. Test Business Logic, Not Framework

```go
// ✅ GOOD - Tests business logic
func TestDiscountCalculation(t *testing.T) {
    ctx := restate.NewMockContext()
    
    order := Order{Total: 100, CouponCode: "SAVE10"}
    result, _ := OrderService{}.ApplyDiscount(ctx, order)
    
    assert.Equal(t, 90.0, result.Total)
}

// ❌ BAD - Tests framework behavior
func TestStateIsPersisted(t *testing.T) {
    ctx := restate.NewMockContext()
    restate.Set(ctx, "key", "value")
    value, _ := restate.Get[string](ctx, "key")
    assert.Equal(t, "value", value) // Testing Restate, not your code!
}
```

### 2. Use Table-Driven Tests

```go
func TestOrderValidation(t *testing.T) {
    tests := []struct {
        name    string
        order   Order
        wantErr bool
    }{
        {"valid order", Order{Items: []Item{{SKU: "A", Qty: 1}}}, false},
        {"empty items", Order{Items: []Item{}}, true},
        {"negative quantity", Order{Items: []Item{{SKU: "A", Qty: -1}}}, true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctx := restate.NewMockContext()
            _, err := OrderService{}.Validate(ctx, tt.order)
            
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### 3. Test Error Handling

```go
func TestPaymentFailure(t *testing.T) {
    ctx := restate.NewMockContext()
    
    // Mock payment failure
    ctx.MockRun("chargePayment").
        Returns("", errors.New("insufficient funds"))
    
    // Test
    result, err := OrderService{}.ProcessOrder(ctx, order)
    
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "insufficient funds")
}
```

### 4. Verify State Changes

```go
func TestCartAddItem(t *testing.T) {
    ctx := restate.NewMockContext()
    
    // Initial state
    restate.Set(ctx, "cart", Cart{Items: []Item{}})
    
    // Add item
    cart := ShoppingCart{}
    err := cart.AddItem(ctx, AddItemRequest{SKU: "ITEM-001", Qty: 2})
    
    assert.NoError(t, err)
    
    // Verify state
    updatedCart, _ := restate.Get[Cart](ctx, "cart")
    assert.Len(t, updatedCart.Items, 1)
    assert.Equal(t, "ITEM-001", updatedCart.Items[0].SKU)
    assert.Equal(t, 2, updatedCart.Items[0].Quantity)
}
```

## 🔗 Integration Testing

Integration tests use a real Restate server to test end-to-end workflows.

### Setup Test Server

```go
func setupTestServer(t *testing.T) (*server.Restate, string) {
    // Create server
    srv := server.NewRestate()
    
    // Register services
    srv.Bind(restate.Reflect(OrderService{}))
    srv.Bind(restate.Reflect(PaymentService{}))
    
    // Start on random port
    port := findFreePort()
    go srv.Start(context.Background(), fmt.Sprintf(":%d", port))
    
    // Wait for server to be ready
    time.Sleep(100 * time.Millisecond)
    
    return srv, fmt.Sprintf("http://localhost:%d", port)
}
```

### Integration Test Example

```go
func TestOrderWorkflow_Integration(t *testing.T) {
    // Setup
    srv, url := setupTestServer(t)
    defer srv.Stop()
    
    // Register with Restate
    registerService(t, url)
    
    // Invoke workflow
    client := restateingress.NewClient("http://localhost:8080")
    result, err := restateingress.Service[Order, OrderResult](
        client, "OrderService", "ProcessOrder",
    ).Request(context.Background(), testOrder)
    
    // Verify
    assert.NoError(t, err)
    assert.NotEmpty(t, result.OrderID)
    assert.Equal(t, "confirmed", result.Status)
}
```

### Testing State Persistence

```go
func TestStateSurvivesRestart(t *testing.T) {
    srv, url := setupTestServer(t)
    
    // Set state
    invokeHandler(t, url, "Cart", "cart-123", "AddItem", item)
    
    // Restart server
    srv.Stop()
    srv, url = setupTestServer(t)
    
    // Verify state persisted
    cart := invokeHandler(t, url, "Cart", "cart-123", "GetCart", nil)
    assert.Len(t, cart.Items, 1)
}
```

## 🎯 Testing Sagas

Sagas require special attention to test compensation logic.

### Test Successful Saga

```go
func TestTravelSaga_Success(t *testing.T) {
    ctx := restate.NewMockContext()
    
    // Mock all services to succeed
    ctx.MockService("FlightService", "Reserve").Returns(flightConf, nil)
    ctx.MockService("HotelService", "Reserve").Returns(hotelConf, nil)
    ctx.MockService("CarService", "Reserve").Returns(carConf, nil)
    
    // Test
    result, err := TravelSaga{}.Run(ctx, booking)
    
    assert.NoError(t, err)
    assert.Equal(t, "confirmed", result.Status)
}
```

### Test Compensation

```go
func TestTravelSaga_CompensationOnCarFailure(t *testing.T) {
    ctx := restate.NewMockContext()
    
    // Flight and hotel succeed
    ctx.MockService("FlightService", "Reserve").Returns(flightConf, nil)
    ctx.MockService("HotelService", "Reserve").Returns(hotelConf, nil)
    
    // Car fails
    ctx.MockService("CarService", "Reserve").
        Returns(CarConfirmation{}, errors.New("unavailable"))
    
    // Expect compensations
    hotelCancel := ctx.MockService("HotelService", "Cancel").Returns(nil)
    flightCancel := ctx.MockService("FlightService", "Cancel").Returns(nil)
    
    // Test
    result, err := TravelSaga{}.Run(ctx, booking)
    
    assert.NoError(t, err)
    assert.Equal(t, "failed", result.Status)
    
    // Verify compensations were called
    assert.True(t, hotelCancel.WasCalled())
    assert.True(t, flightCancel.WasCalled())
}
```

## 🕐 Testing Time-Dependent Logic

### Mock Sleep for Instant Tests

```go
func TestWorkflowTimeout(t *testing.T) {
    ctx := restate.NewMockContext()
    
    // Mock sleep to return instantly
    ctx.MockSleep(24 * time.Hour)
    
    // Test workflow with timeout
    result, err := ApprovalWorkflow{}.Run(ctx, doc)
    
    // Verify timeout occurred
    assert.Equal(t, "timeout", result.Status)
}
```

### Test Promise vs Timeout Race

```go
func TestPromiseBeforeTimeout(t *testing.T) {
    ctx := restate.NewMockContext()
    
    // Promise resolves
    ctx.MockPromise("approval").ResolveWith(decision)
    
    // Timeout doesn't trigger
    ctx.MockSleep(48 * time.Hour)
    
    result, _ := ApprovalWorkflow{}.Run(ctx, doc)
    assert.Equal(t, "approved", result.Status)
}

func TestTimeoutBeforePromise(t *testing.T) {
    ctx := restate.NewMockContext()
    
    // Timeout triggers (promise never resolves)
    ctx.MockSleep(48 * time.Hour)
    
    result, _ := ApprovalWorkflow{}.Run(ctx, doc)
    assert.Equal(t, "timeout", result.Status)
}
```

## ✅ Testing Checklist

Before deploying, ensure:

- [ ] Unit tests for all business logic
- [ ] Error cases tested
- [ ] State transitions verified
- [ ] Saga compensation tested
- [ ] Integration tests for critical flows
- [ ] Edge cases covered
- [ ] Test coverage > 80%
- [ ] All tests pass

## 🎓 Test-Driven Development (TDD)

### Red-Green-Refactor Cycle

```
1. 🔴 RED - Write failing test
2. 🟢 GREEN - Write minimal code to pass
3. 🔵 REFACTOR - Improve code quality
```

### TDD Example

```go
// 1. RED - Write test first (fails)
func TestApplyDiscount(t *testing.T) {
    ctx := restate.NewMockContext()
    order := Order{Total: 100, Coupon: "SAVE10"}
    
    result, _ := OrderService{}.ApplyDiscount(ctx, order)
    
    assert.Equal(t, 90.0, result.Total)
}

// 2. GREEN - Implement minimal code
func (OrderService) ApplyDiscount(ctx restate.Context, order Order) (Order, error) {
    if order.Coupon == "SAVE10" {
        order.Total = order.Total * 0.9
    }
    return order, nil
}

// 3. REFACTOR - Improve implementation
func (OrderService) ApplyDiscount(ctx restate.Context, order Order) (Order, error) {
    discount, err := restate.Run(ctx, func(ctx restate.RunContext) (float64, error) {
        return couponService.GetDiscount(order.Coupon)
    })
    if err != nil {
        return order, err
    }
    
    order.Total = order.Total * (1 - discount)
    return order, nil
}
```

## 🎯 Next Step

Ready to write tests for real Restate services!

👉 **Continue to [Hands-On Tutorial](./02-hands-on.md)**

---

**Key Takeaway:** Testing Restate applications requires mocking the context and external dependencies, but the patterns are straightforward once you understand them!
