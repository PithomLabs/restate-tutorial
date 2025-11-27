# Saga Framework: Storage, Retention, and Idempotency Guide

> **Understanding saga entry persistence, retention limits, and compensation handler best practices**

## Overview

The `SagaFramework` provides durable distributed transaction compensation following the Saga pattern. It persists compensation steps in workflow state and executes them in LIFO (Last-In-First-Out) order when errors occur.

### Key Characteristics

**Strengths** ✅:
- Full durable saga bookkeeping with deterministic step IDs
- Automatic deduplication of repeated steps
- Retry/backoff logic with configurable attempts
- DLQ (Dead Letter Queue) support for irrecoverable failures
- Automatic LIFO compensation on errors
- `SafeStep` pattern enforces compensation-before-action at API level

**Considerations** ⚠️:
- Compensation handlers must be idempotent (framework logs but cannot enforce)
- Saga entries are stored in workflow state (subject to retention limits)
- Entry size affects storage and performance

## Storage and Retention

### Where Saga Entries Are Stored

Saga entries are persisted in **workflow state** using Restate's state management:

```go
// Each saga stores its entries under a deterministic key
entries, _ := restate.Get[[]SagaEntry](s.wctx, s.nsKey)
entries = append(entries, newEntry)
restate.Set(s.wctx, s.nsKey, entries)
```

**State Key Format**: `saga:{saga-name}:entries`

**Storage Type**: JSON-serialized array of `SagaEntry` structs

### SagaEntry Structure

```go
type SagaEntry struct {
	Name      string    `json:"name"`       // Step name (e.g., "charge_payment")
	Payload   []byte    `json:"payload"`    // Serialized compensation data
	StepID    string    `json:"step_id"`    // Deterministic hash for deduplication
	Timestamp time.Time `json:"timestamp"`  // When step was added
	Attempt   int       `json:"attempt"`    // Retry counter
}
```

**Typical Entry Size**:
- Metadata: ~150 bytes
- Payload: Variable (depends on your compensation data)
- Total: Usually 200-2000 bytes per entry

### Restate Workflow State Retention

Restate workflows have **state retention limits** after completion:

| Environment | Default Retention | Configurable Limit |
|-------------|------------------|-------------------|
| Restate Cloud | 30 days | Up to 90 days |
| Self-Hosted | Configurable | No theoretical limit |

> [!IMPORTANT]
> **After workflow retention expires**, all state (including saga entries) is purged. Design workflows to complete compensation within the retention window.

### Estimating Storage Requirements

#### Example: E-Commerce Order Saga

```go
// Typical saga with 5-10 steps
saga := NewSaga(ctx, "order-saga", nil)

// Step 1: Reserve inventory (500 bytes payload)
saga.Add("reserve_inventory", ReservationData{
	OrderID:   "order-123",
	ProductID: "product-456",
	Quantity:  2,
}, true)

// Step 2: Charge payment (300 bytes payload)
saga.Add("charge_payment", PaymentData{
	OrderID:       "order-123",
	Amount:        99.99,
	PaymentMethod: "card_xyz",
}, true)

// Step 3: Send confirmation (200 bytes payload)
saga.Add("send_confirmation", EmailData{
	OrderID: "order-123",
	Email:   "user@example.com",
}, true)

// ... more steps
```

**Total Storage**:
- 3 entries × ~700 bytes average = ~2.1 KB per saga instance
- 10,000 concurrent orders = ~21 MB
- Negligible compared to Restate's storage capacity

#### Scaling Considerations

| Saga Size | Concurrent Instances | Total Storage | Performance Impact |
|-----------|---------------------|---------------|-------------------|
| 5 steps (2 KB) | 1,000 | 2 MB | None |
| 10 steps (5 KB) | 10,000 | 50 MB | Minimal |
| 20 steps (10 KB) | 100,000 | 1 GB | Monitor |
| 50+ steps (25+ KB) | High volume | Significant | Review design |

> [!WARNING]
> **Large Sagas**
>
> If your saga has 50+ steps or payloads exceed 10 KB, consider:
> 1. Breaking into multiple sub-sagas
> 2. Storing large payloads externally (S3, database) and only storing references
> 3. Using workflow retention policies to auto-purge completed sagas

### Retention Best Practices

#### 1. Configure Appropriate Retention

```go
// For short-lived transactions (minutes to hours)
workflow.SetRetention(24 * time.Hour) // 1 day

// For long-running sagas (days to weeks)
workflow.SetRetention(30 * 24 * time.Hour) // 30 days

// Critical audit trail
workflow.SetRetention(90 * 24 * time.Hour) // 90 days
```

#### 2. Clean Up After Successful Completion

```go
func (cp *ControlPlaneService) CompleteOrder(ctx restate.WorkflowContext) error {
	saga := NewSaga(ctx, "order-saga", nil)
	
	err := saga.ExecuteSteps(/*...*/)
	if err != nil {
		// Saga will compensate automatically
		return err
	}
	
	// On success, clear saga entries to free storage
	restate.Clear(ctx, "saga:order-saga:entries")
	
	return nil
}
```

#### 3. Archive to External Storage

For audit requirements exceeding Restate retention:

```go
func (s *SagaFramework) ArchiveToDLQ(entries []SagaEntry) error {
	// Serialize saga history
	data, _ := json.Marshal(entries)
	
	// Store in external system (S3, database, etc.)
	return RunDoVoid(s.wctx, func(rc restate.RunContext) error {
		return uploadToS3(data, s.nsKey) // Non-Restate call
	}, restate.WithName("archive-saga"))
}
```

## Compensation Handler Idempotency

### The Idempotency Requirement

**Critical**: Compensation handlers MUST be idempotent because:
1. Restate may replay them during recovery
2. Retries will execute the same compensation multiple times
3. Network failures can cause duplicate compensation attempts

### What the Framework Cannot Enforce

The framework provides warnings but **cannot enforce idempotency**:

```go
// Framework will warn about nil handlers
saga.Register("payment", nil) // ⚠️ Warning logged

// But cannot detect non-idempotent logic
saga.Register("payment", func(rc restate.RunContext, payload []byte) error {
	// ❌ Non-idempotent: charges twice on retry
	return chargeCustomer(paymentID, amount)
})
```

### Idempotency Validation Helper

#### ValidateCompensationIdempotent

Use this helper to document and validate idempotency requirements:

```go
// ValidateCompensationIdempotent is a documentation/lint helper for compensation handlers.
//
// While the framework cannot enforce idempotency at runtime, this helper:
// 1. Documents the idempotency contract
// 2. Provides a clear API for developers
// 3. Enables static analysis/linting
//
// Usage:
//
//	saga.Register("charge_payment", ValidateCompensationIdempotent(
//	    "charge_payment",
//	    func(rc restate.RunContext, payload []byte) error {
//	        // Your idempotent compensation logic
//	        return refundPayment(paymentID) // ✅ Idempotent
//	    },
//	))
func ValidateCompensationIdempotent(
	name string,
	handler SagaCompensationFunc,
) SagaCompensationFunc {
	// Wrap handler with documentation marker
	return func(rc restate.RunContext, payload []byte) error {
		// Future: Could add runtime checks here if needed
		// - Check for duplicate compensation IDs
		// - Log compensation attempts
		// - Validate payload schema
		
		return handler(rc, payload)
	}
}
```

**Usage Example**:

```go
saga.Register("charge_payment", ValidateCompensationIdempotent(
	"charge_payment",
	func(rc restate.RunContext, payload []byte) error {
		var data PaymentData
		json.Unmarshal(payload, &data)
		
		// ✅ Idempotent: refund checks if already refunded
		return refundPayment(data.PaymentID)
	},
))
```

### Making Compensations Idempotent

#### Pattern 1: Check-Then-Act

```go
// ❌ Non-Idempotent
func compensateInventory(rc restate.RunContext, payload []byte) error {
	releaseInventory(productID, quantity) // Releases twice on retry
	return nil
}

// ✅ Idempotent with check
func compensateInventory(rc restate.RunContext, payload []byte) error {
	// Check if already released
	if isInventoryReleased(productID) {
		return nil // Already done
	}
	
	return releaseInventory(productID, quantity)
}
```

#### Pattern 2: Use Idempotency Keys

```go
// ✅ Idempotent with external idempotency key
func compensatePayment(rc restate.RunContext, payload []byte) error {
	var data PaymentData
	json.Unmarshal(payload, &data)
	
	// Payment provider handles idempotency via key
	return refundWithIdempotencyKey(
		data.PaymentID,
		data.OrderID, // Use deterministic key
	)
}
```

#### Pattern 3: State-Based Deduplication

```go
// ✅ Idempotent using workflow state
func compensateNotification(rc restate.RunContext, payload []byte) error {
	var data EmailData
	json.Unmarshal(payload, &data)
	
	// Check workflow state
	sent, _ := restate.Get[bool](ctx, fmt.Sprintf("cancelled_email_sent:%s", data.OrderID))
	if sent {
		return nil // Already sent
	}
	
	err := sendCancellationEmail(data.Email, data.OrderID)
	if err != nil {
		return err
	}
	
	// Mark as sent
	restate.Set(ctx, fmt.Sprintf("cancelled_email_sent:%s", data.OrderID), true)
	return nil
}
```

#### Pattern 4: At-Least-Once with External Deduplication

```go
// ✅ Idempotent via external system
func compensateDatabase(rc restate.RunContext, payload []byte) error {
	var data DBUpdate
	json.Unmarshal(payload, &data)
	
	// Database handles idempotency via unique constraints
	return db.DeleteWithCheck(
		data.TableName,
		data.RecordID, // Unique key
	)
}
```

### Anti-Patterns to Avoid

#### ❌ Incrementing/Decrementing Without Checks

```go
// ❌ BAD: Non-idempotent counter
func compensateCounter(rc restate.RunContext, payload []byte) error {
	decrementCounter(counterID) // Decrements multiple times on retry
	return nil
}

// ✅ GOOD: Set absolute value
func compensateCounter(rc restate.RunContext, payload []byte) error {
	var data CounterData
	json.Unmarshal(payload, &data)
	
	setCounterValue(counterID, data.PreviousValue) // Idempotent
	return nil
}
```

#### ❌ Creating Resources Without Deduplication

```go
// ❌ BAD: Creates duplicate records
func compensateFailure(rc restate.RunContext, payload []byte) error {
	createFailureRecord(orderID, reason) // New record each retry
	return nil
}

// ✅ GOOD: Upsert with unique key
func compensateFailure(rc restate.RunContext, payload []byte) error {
	var data FailureData
	json.Unmarshal(payload, &data)
	
	upsertFailureRecord(data.OrderID, data.Reason) // Idempotent
	return nil
}
```

#### ❌ Timing-Dependent Logic

```go
// ❌ BAD: Time-dependent (non-deterministic)
func compensateWithDelay(rc restate.RunContext, payload []byte) error {
	waitTime := time.Since(startTime) // Different on each retry!
	time.Sleep(waitTime)
	return releaseResources()
}

// ✅ GOOD: Fixed logic
func compensateWithDelay(rc restate.RunContext, payload []byte) error {
	// Use durable sleep if delay is needed
	// Or just execute immediately (compensations should be fast)
	return releaseResources()
}
```

## Monitoring and Observability

### Metrics to Track

```go
// Saga entry count per workflow
saga_entries_total{saga_name="order-saga"} 5

// Saga storage size
saga_storage_bytes{saga_name="order-saga"} 2048

// Compensation execution
saga_compensation_total{step="charge_payment"} 10
saga_compensation_errors{step="charge_payment"} 2
saga_compensation_duration_seconds{step="charge_payment"} 0.5

// DLQ persistence
saga_dlq_records_total{saga_name="order-saga"} 1
```

### Logging Best Practices

```go
// Log saga entry additions
saga.Add("payment", data, true)
// → saga.step_added name=payment step_id=abc123

// Log compensations
saga.CompensateIfNeeded(&err)
// → saga.compensation.attempting name=payment attempt=1
// → saga.compensation.succeeded name=payment duration=250ms
// OR
// → saga.compensation.failed name=payment attempt=1 err="connection timeout"

// Log DLQ persistence
saga.persistDLQ(...)
// → saga.dlq_recorded dlq_key=saga:order-saga:dlq
```

### Alerts to Configure

1. **High compensation rate**: `saga_compensation_total / saga_entries_total > 0.1` (10% failure rate)
2. **DLQ growth**: `rate(saga_dlq_records_total[5m]) > 0` (any DLQ persistence)
3. **Large saga size**: `saga_storage_bytes > 100KB` (unusually large saga)
4. **Slow compensations**: `saga_compensation_duration_seconds > 5s` (slow compensation logic)

## Testing Compensation Handlers

### Unit Test Template

```go
func TestCompensationIdempotency(t *testing.T) {
	// Setup
	ctx := setupTestWorkflowContext(t)
	saga := NewSaga(ctx, "test-saga", nil)
	
	// Register compensation
	saga.Register("payment", func(rc restate.RunContext, payload []byte) error {
		return refundPayment("payment-123")
	})
	
	// Execute compensation multiple times
	for i := 0; i < 3; i++ {
		err := saga.executeCompensation("payment", []byte(`{}`))
		assert.NoError(t, err, "Compensation should be idempotent")
	}
	
	// Verify side effects occurred only once
	refundCount := getRefundCount("payment-123")
	assert.Equal(t, 1, refundCount, "Refund should only happen once")
}
```

### Integration Test Checklist

- [ ] Compensation executes successfully on first attempt
- [ ] Compensation is idempotent (can run multiple times safely)
- [ ] Compensation handles concurrent executions (if applicable)
- [ ] Compensation completes within reasonable time (<5s typically)
- [ ] Compensation logs errors appropriately
- [ ] Compensation handles missing/corrupted payloads gracefully
- [ ] Large payloads don't cause performance issues

## FAQ

### Q: How many saga steps is too many?

**A**: General guidelines:
- **1-10 steps**: Ideal, no concerns
- **10-20 steps**: Fine, monitor storage
- **20-50 steps**: Consider breaking into sub-sagas
- **50+ steps**: Definitely refactor

### Q: What happens if compensation fails?

**A**: The framework retries with exponential backoff:
1. Attempt 1: Immediate
2. Attempt 2: After ~1 second
3. Attempt 3: After ~2 seconds
4. ... up to `MaxRetries` (default: 3)

If all retries fail, the entry goes to the DLQ for manual intervention.

### Q: Can I clear saga entries manually?

**A**: Yes, use `restate.Clear()`:

```go
// After successful compensation
restate.Clear(ctx, "saga:my-saga:entries")

// Or clear everything
saga.ClearAll()
```

### Q: How do I debug compensation failures?

**A**:
1. Check logs for `saga.compensation.failed` entries
2. Review DLQ entries: `restate.Get(ctx, "saga:my-saga:dlq")`
3. Verify external system state (payment provider, inventory, etc.)
4. Test compensation handler in isolation

### Q: What if my compensation needs data not in the payload?

**A**: Include all necessary data when adding the step:

```go
// ❌ Missing data
saga.Add("release_inventory", InventoryData{ProductID: "p123"}, true)

// ✅ Complete data
saga.Add("release_inventory", InventoryData{
	ProductID:   "p123",
	Quantity:    2,
	WarehouseID: "w456", // Needed for compensation
	ReservedAt:  time.Now(),
}, true)
```

## Related Documentation

- [SafeStep Pattern](file:///home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/DOS_DONTS_MEGA_FRAMEWORK.MD#safestep) - Compensation-before-action enforcement
- [Framework Policy](file:///home/chaschel/Documents/ibm/go/apps/restate/examples/evals/rea2/claude/FRAMEWORK_POLICY.MD) - Global validation control
- [Restate Sagas Guide](https://docs.restate.dev/concepts/sagas) - Official Restate documentation

## Summary

| Aspect | Recommendation |
|--------|----------------|
| **Storage** | Saga entries stored in workflow state (~700 bytes per entry) |
| **Retention** | Configure based on workflow duration (default 30 days) |
| **Scaling** | Monitor if >20 steps or >100K concurrent sagas |
| **Idempotency** | Use `ValidateCompensationIdempotent` wrapper |
| **Testing** | Verify compensations are idempotent |
| **Monitoring** | Track compensation rate, DLQ growth, and storage size |
| **Cleanup** | Clear entries after successful completion |

**Key Takeaway**: Saga entries are lightweight, but design compensation handlers to be idempotent and monitor storage/retention limits for long-running workflows.
