# Module 07: Testing Restate Applications

> **Build confidence with comprehensive testing strategies**

## 🎯 Module Overview

Testing distributed, durable applications requires different strategies than traditional applications. This module teaches you how to test Restate services effectively using unit tests, integration tests, and mocking.

### What You'll Learn

- ✅ Unit testing Restate handlers
- ✅ Mocking external dependencies
- ✅ Testing with the Restate mock context
- ✅ Integration testing with real Restate server
- ✅ Testing sagas and compensation logic
- ✅ Test-driven development (TDD) with Restate

### Real-World Use Cases

- 🧪 **Unit Tests** - Test business logic in isolation
- 🔗 **Integration Tests** - Verify end-to-end workflows
- 🎭 **Mocking** - Simulate external services
- 📊 **Test Coverage** - Ensure reliability
- 🐛 **Regression Tests** - Prevent bugs from returning

## 🧪 Testing Challenges

### Traditional Testing

```go
func TestAdd(t *testing.T) {
    result := Add(2, 3)
    assert.Equal(t, 5, result)
}
```

**Simple:** Pure functions, no side effects

### Restate Testing Challenges

1. **Durable Context** - Tests need Restate context
2. **Side Effects** - External calls wrapped in `restate.Run`
3. **State** - Virtual Objects with persistent state
4. **Workflows** - Long-running, asynchronous
5. **Compensation** - Saga rollback logic

## 💡 The Solution: Test Strategies

### 1. Unit Tests with Mock Context

```go
func TestGreeting(t *testing.T) {
    // Create mock context
    ctx := restate.NewMockContext()
    
    // Test handler
    result, err := Greeter{}.Greet(ctx, "Alice")
    
    assert.NoError(t, err)
    assert.Equal(t, "Hello, Alice!", result)
}
```

### 2. Integration Tests with Real Server

```go
func TestWorkflowE2E(t *testing.T) {
    // Start Restate server
    // Deploy service
    // Invoke workflow
    // Verify result
}
```

### 3. Mocking External Dependencies

```go
func TestWithMockedService(t *testing.T) {
    ctx := restate.NewMockContext()
    
    // Mock external service response
    ctx.MockService("ExternalAPI", "Call").
        Returns("mocked response")
    
    // Test
    result, err := MyService{}.Process(ctx, input)
}
```

## 🏗️ Module Structure

### 1. [Concepts](./01-concepts.md) (~30 min)
Learn about:
- Testing strategies for durable execution
- Mock context API
- Integration test patterns
- Best practices

### 2. [Hands-On](./02-hands-on.md) (~45 min)
Build tests for:
- Simple service handlers
- Virtual Objects with state
- Workflows with promises
- Sagas with compensation

### 3. [Validation](./03-validation.md) (~20 min)
Verify:
- Test coverage
- Mock behavior
- Integration tests pass
- Edge cases handled

### 4. [Exercises](./04-exercises.md) (~60 min)
Practice testing:
- Complex business logic
- Error scenarios
- Concurrent execution
- Long-running workflows

## 🎓 Prerequisites

- ✅ Completed [Module 06](../06-sagas/README.md) - Sagas
- ✅ Basic Go testing knowledge
- ✅ Understanding of test-driven development (TDD)

## 🚀 Quick Start

```bash
# Navigate to module directory
cd ~/restate-tutorials/module07

# Follow hands-on tutorial
cat 02-hands-on.md
```

## 🎯 Learning Objectives

By the end of this module, you will:

1. **Write Unit Tests**
   - Test handlers in isolation
   - Mock external dependencies
   - Verify business logic

2. **Create Integration Tests**
   - Test with real Restate server
   - Verify end-to-end workflows
   - Test state persistence

3. **Test Edge Cases**
   - Error handling
   - Compensation logic
   - Race conditions

4. **Apply TDD**
   - Write tests first
   - Red-green-refactor cycle
   - Maintain test coverage

## 📖 Module Flow

```
Concepts → Hands-On → Validation → Exercises
   ↓          ↓          ↓            ↓
 Theory → Write Tests → Run Tests → Practice
```

## 🔑 Key Concept Preview

### Unit Test Example

```go
func TestShoppingCart_AddItem(t *testing.T) {
    ctx := restate.NewMockContext()
    cart := ShoppingCart{}
    
    // Add item
    err := cart.AddItem(ctx, AddItemRequest{
        SKU:      "ITEM-001",
        Quantity: 2,
    })
    
    assert.NoError(t, err)
    
    // Verify state
    state, _ := restate.Get[CartState](ctx, "cart")
    assert.Len(t, state.Items, 1)
    assert.Equal(t, 2, state.Items[0].Quantity)
}
```

### Integration Test Example

```go
func TestOrderSaga_E2E(t *testing.T) {
    // Start service
    server := startTestServer(t)
    defer server.Stop()
    
    // Invoke saga
    result := invokeSaga(t, "order-123", orderRequest)
    
    // Verify
    assert.Equal(t, "confirmed", result.Status)
    assert.NotEmpty(t, result.PaymentID)
}
```

## 💡 Why Test Restate Applications?

**Testing Benefits:**
- ✅ **Confidence** - Know your code works
- ✅ **Documentation** - Tests show how to use your code
- ✅ **Refactoring** - Change code safely
- ✅ **Regression Prevention** - Catch bugs early
- ✅ **Design Feedback** - Better code structure

**Restate-Specific Benefits:**
- ✅ **Verify Durability** - State persists correctly
- ✅ **Test Compensation** - Saga rollback works
- ✅ **Validate Journaling** - Operations are idempotent
- ✅ **Check Timeouts** - Workflows handle delays

## 🆚 Unit vs Integration Tests

| Aspect | Unit Tests | Integration Tests |
|--------|-----------|------------------|
| **Speed** | Fast (milliseconds) | Slower (seconds) |
| **Scope** | Single handler | Full workflow |
| **Dependencies** | Mocked | Real services |
| **Setup** | Minimal | Complex |
| **Isolation** | Complete | Partial |
| **Coverage** | Business logic | Integration points |

**Best Practice:** Use both!
- Many unit tests (fast feedback)
- Fewer integration tests (confidence)

## ⚠️ Testing Gotchas

### 1. Time-Based Tests

```go
// ❌ WRONG - Non-deterministic
func TestTimeout(t *testing.T) {
    start := time.Now()
    // ... test ...
    duration := time.Since(start)
    assert.Less(t, duration, 1*time.Second) // Flaky!
}

// ✅ CORRECT - Mock time
func TestTimeout(t *testing.T) {
    ctx := restate.NewMockContext()
    ctx.MockSleep(30 * time.Second) // Instant
    // ... test ...
}
```

### 2. Random Values

```go
// ❌ WRONG - Non-deterministic
func TestWithRandom(t *testing.T) {
    ctx := restate.NewMockContext()
    // restate.Rand(ctx) returns different values each run
}

// ✅ CORRECT - Seed mock context
func TestWithRandom(t *testing.T) {
    ctx := restate.NewMockContext(restate.WithSeed(42))
    // Deterministic random values
}
```

### 3. External Services

```go
// ❌ WRONG - Calls real service
func TestPayment(t *testing.T) {
    ctx := restate.NewMockContext()
    // This calls real payment API!
    result, _ := processPayment(ctx, payment)
}

// ✅ CORRECT - Mock external call
func TestPayment(t *testing.T) {
    ctx := restate.NewMockContext()
    ctx.MockRun("processPayment").Returns("payment-123", nil)
    result, _ := processPayment(ctx, payment)
}
```

## 🎯 Ready to Start?

Let's learn how to test Restate applications effectively!

👉 **Start with [Concepts](./01-concepts.md)**

---

**Questions?** Check the main [tutorials README](../README.md) or review [Module 06](../06-sagas/README.md).
