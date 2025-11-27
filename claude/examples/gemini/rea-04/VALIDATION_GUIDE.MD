# Validation Against Non-Deterministic ID Usage - Integration Guide

## Overview

The framework now includes multiple layers of validation to prevent non-deterministic ID usage. Here's how to use and extend these validations.

## 1. Automatic Validation (Built-in)

### ServiceClient.Send() - Automatic Idempotency Key Validation

**Location:** `framework.go` lines 528-536

The `ServiceClient.Send()` method now **automatically validates** all idempotency keys:

```go
func (c ServiceClient[I, O]) Send(
    ctx restate.Context,
    input I,
    opts ...CallOption,
) restate.Invocation {
    // Build options slice
    var sendOpts []restate.SendOption
    for _, opt := range opts {
        if opt.IdempotencyKey != "" {
            // Automatic validation happens here
            if err := ValidateIdempotencyKey(opt.IdempotencyKey); err != nil {
                ctx.Log().Error("framework: invalid idempotency key detected",
                    "key", opt.IdempotencyKey,
                    "error", err.Error())
                // Logs warning but doesn't block (heuristic-based validation)
            }
            sendOpts = append(sendOpts, restate.WithIdempotencyKey(opt.IdempotencyKey))
        }
        // ...
    }
    return send.Send(input, sendOpts...)
}
```

**What it does:**
- ✅ Automatically validates every idempotency key passed via `CallOption`
- ✅ Logs errors when suspicious patterns detected (10+ consecutive digits)
- ⚠️ Currently logs but doesn't block (configurable behavior)

**Example - No action needed, validation is automatic:**

```go
client := ServiceClient[PaymentRequest, PaymentResponse]{
    ServiceName: "PaymentGateway",
    HandlerName: "Charge",
}

// This will be automatically validated
badKey := fmt.Sprintf("order:%d", time.Now().UnixNano()) // Will trigger warning
client.Send(ctx, req, CallOption{
    IdempotencyKey: badKey, // Validation happens automatically
})
```

## 2. Manual Validation (Explicit)

### ValidateIdempotencyKey() - Direct Usage

**Location:** `framework.go` lines 604-623

Use this function to explicitly validate keys before use:

```go
func ValidateIdempotencyKey(key string) error
```

**Example 1: Validate external keys**

```go
func handleWebhook(ctx restate.Context, webhook WebhookPayload) error {
    // Webhook provides its own idempotency key - validate it!
    if err := ValidateIdempotencyKey(webhook.IdempotencyKey); err != nil {
        ctx.Log().Error("Rejecting webhook with invalid idempotency key", 
            "webhook_id", webhook.ID,
            "error", err)
        return err // Block the webhook
    }
    
    // Safe to use the validated key
    return processWebhook(ctx, webhook)
}
```

**Example 2: Validate before storing in state**

```go
func (OrderService) CreateOrder(ctx restate.ObjectContext, req CreateOrderRequest) error {
    // Validate idempotency key before storing in state
    if req.IdempotencyKey != "" {
        if err := ValidateIdempotencyKey(req.IdempotencyKey); err != nil {
            return restate.TerminalError(
                fmt.Errorf("invalid idempotency key in request: %w", err), 
                400,
            )
        }
    }
    
    // Store validated key in state
    state := NewState[OrderData](ctx, "order")
    return state.Set(OrderData{
        IdempotencyKey: req.IdempotencyKey,
        // ...
    })
}
```

**Example 3: Validate in saga compensation registration**

```go
func registerPaymentCompensation(saga *SagaFramework, paymentID string) error {
    // Create idempotency key for compensation
    compensationKey := fmt.Sprintf("refund:%s", paymentID)
    
    // Validate before registering
    if err := ValidateIdempotencyKey(compensationKey); err != nil {
        return fmt.Errorf("cannot register compensation with invalid key: %w", err)
    }
    
    return saga.Add("refundPayment", map[string]string{
        "paymentID": paymentID,
        "idempotencyKey": compensationKey,
    }, true)
}
```

## 3. Validation Patterns: What Gets Detected

### Pattern 1: Unix Timestamps (10-13 digits)

```go
// ❌ BAD - Will be flagged
badKey1 := fmt.Sprintf("order:%d", time.Now().Unix())        // 10 digits
badKey2 := fmt.Sprintf("order:%d", time.Now().UnixMilli())   // 13 digits
badKey3 := fmt.Sprintf("order:%d", time.Now().UnixNano())    // 19 digits

err := ValidateIdempotencyKey(badKey1)
// Returns: "idempotency key may contain non-deterministic timestamp: order:1700123456"
```

### Pattern 2: Empty Keys

```go
// ❌ BAD - Will be rejected
err := ValidateIdempotencyKey("")
// Returns: "idempotency key cannot be empty"
```

### Pattern 3: Valid Deterministic Keys

```go
// ✅ GOOD - Will pass validation
goodKey1 := "order:user-123:product-456"
goodKey2 := "payment:abc-def-ghi"  // UUID format
goodKey3 := "refund:order-999"     // Business identifiers

err := ValidateIdempotencyKey(goodKey1) // nil
```

## 4. Configuration Options for Stricter Validation

### Option A: Make Validation Blocking

To make validation errors **block** instead of just logging, modify `ServiceClient.Send()`:

```go
// In ServiceClient.Send() method
if opt.IdempotencyKey != "" {
    // STRICT MODE: Block on validation failure
    if err := ValidateIdempotencyKey(opt.IdempotencyKey); err != nil {
        ctx.Log().Error("framework: blocking call due to invalid idempotency key",
            "service", c.ServiceName,
            "handler", c.HandlerName,
            "key", opt.IdempotencyKey,
            "error", err.Error())
        
        // Return a dummy invocation that will fail
        // (Or panic, or return error if you change the method signature)
        panic(fmt.Sprintf("Invalid idempotency key: %v", err))
    }
    sendOpts = append(sendOpts, restate.WithIdempotencyKey(opt.IdempotencyKey))
}
```

### Option B: Add Configuration Flag

Add a global or per-client configuration:

```go
// Add to ServiceClient struct
type ServiceClient[I, O any] struct {
    ServiceName          string
    HandlerName          string
    StrictValidation     bool  // NEW: Enable strict validation
}

// In Send() method
if opt.IdempotencyKey != "" {
    if err := ValidateIdempotencyKey(opt.IdempotencyKey); err != nil {
        if c.StrictValidation {
            // Block in strict mode
            panic(fmt.Sprintf("Invalid idempotency key: %v", err))
        } else {
            // Just log in permissive mode
            ctx.Log().Warn("framework: invalid idempotency key", "error", err)
        }
    }
    sendOpts = append(sendOpts, restate.WithIdempotencyKey(opt.IdempotencyKey))
}
```

## 5. Extending Validation - Custom Rules

### Add Custom Validation Rules

You can extend `ValidateIdempotencyKey()` with additional checks:

```go
func ValidateIdempotencyKey(key string) error {
    if key == "" {
        return restate.TerminalError(fmt.Errorf("idempotency key cannot be empty"), 400)
    }

    // Existing: Check for suspicious timestamps
    if hasSuspiciousTimestamp(key) {
        return restate.TerminalError(
            fmt.Errorf("idempotency key may contain non-deterministic timestamp: %s", key),
            400,
        )
    }

    // NEW: Check for date patterns (YYYY-MM-DD)
    if hasDatePattern(key) {
        return restate.TerminalError(
            fmt.Errorf("idempotency key contains date pattern (non-deterministic): %s", key),
            400,
        )
    }

    // NEW: Check for UUID v4 with high entropy (might be random)
    if hasSuspiciousUUID(key) {
        return restate.TerminalError(
            fmt.Errorf("idempotency key contains potentially random UUID: %s", key),
            400,
        )
    }

    // NEW: Check minimum length
    if len(key) < 3 {
        return restate.TerminalError(
            fmt.Errorf("idempotency key too short (minimum 3 chars): %s", key),
            400,
        )
    }

    return nil
}

// Helper: Detect date patterns like 2024-01-15
func hasDatePattern(key string) bool {
    // Look for YYYY-MM-DD pattern
    for i := 0; i < len(key)-9; i++ {
        if key[i] >= '0' && key[i] <= '9' &&
           key[i+1] >= '0' && key[i+1] <= '9' &&
           key[i+2] >= '0' && key[i+2] <= '9' &&
           key[i+3] >= '0' && key[i+3] <= '9' &&
           key[i+4] == '-' &&
           key[i+5] >= '0' && key[i+5] <= '9' &&
           key[i+6] >= '0' && key[i+6] <= '9' &&
           key[i+7] == '-' &&
           key[i+8] >= '0' && key[i+8] <= '9' &&
           key[i+9] >= '0' && key[i+9] <= '9' {
            return true
        }
    }
    return false
}
```

## 6. Type-Safe Approach (Advanced)

For **compile-time safety**, create a wrapped type:

```go
// IdempotencyKey is a validated, deterministic idempotency key
type IdempotencyKey struct {
    value string
}

// NewIdempotencyKeyFromUUID creates a key from a deterministic UUID
func NewIdempotencyKeyFromUUID(ctx restate.Context, prefix, suffix string) IdempotencyKey {
    uuid := restate.UUID(ctx)
    return IdempotencyKey{
        value: fmt.Sprintf("%s:%s:%s", prefix, suffix, uuid.String()),
    }
}

// NewIdempotencyKeyFromBusinessData creates a key from deterministic business identifiers
func NewIdempotencyKeyFromBusinessData(parts ...string) IdempotencyKey {
    return IdempotencyKey{
        value: path.Join(parts...),
    }
}

// String returns the validated key value
func (k IdempotencyKey) String() string {
    return k.value
}

// Update CallOption to use the type
type CallOption struct {
    IdempotencyKey IdempotencyKey  // Changed from string
    Delay          time.Duration
}

// Now only deterministically-created keys can be used
client.Send(ctx, req, CallOption{
    IdempotencyKey: NewIdempotencyKeyFromUUID(ctx, "payment", "charge"),
})
```

## 7. Testing Validation

### Unit Tests for Validation

```go
func TestValidateIdempotencyKey_RejectsTimestamp(t *testing.T) {
    tests := []struct {
        name    string
        key     string
        wantErr bool
    }{
        {
            name:    "unix timestamp seconds",
            key:     "order:1700123456:payment",
            wantErr: true,
        },
        {
            name:    "unix timestamp millis",
            key:     "order:1700123456789:payment",
            wantErr: true,
        },
        {
            name:    "unix timestamp nanos",
            key:     "order:1700123456789012345:payment",
            wantErr: true,
        },
        {
            name:    "valid business key",
            key:     "order:user-123:product-456",
            wantErr: false,
        },
        {
            name:    "valid UUID",
            key:     "payment:550e8400-e29b-41d4-a716-446655440000",
            wantErr: false,
        },
        {
            name:    "empty key",
            key:     "",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateIdempotencyKey(tt.key)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateIdempotencyKey() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

## 8. Integration Checklist

Use this checklist to ensure validation is properly integrated:

- [x] **Automatic validation in ServiceClient.Send()** - Done (lines 528-536)
- [x] **ValidateIdempotencyKey() function** - Done (lines 604-623)
- [x] **hasSuspiciousTimestamp() detector** - Done (lines 625-641)
- [ ] **Automatic validation in ServiceClient.Call()** - Optional (add if needed)
- [ ] **Validation in ControlPlaneService methods** - Recommended
- [ ] **Custom validation rules** - Optional (extend as needed)
- [ ] **Type-safe IdempotencyKey wrapper** - Advanced (future enhancement)
- [ ] **Configuration for strict vs permissive mode** - Optional
- [ ] **Unit tests for all validation logic** - Recommended
- [ ] **Integration tests with Restate server** - Recommended

## 9. Best Practices Summary

### ✅ DO: Use Framework Helpers

```go
// Use deterministic key generation
cp := NewControlPlaneService(ctx, "workflow", "prefix")
key := cp.GenerateIdempotencyKey(ctx, "operation")  // Uses restate.UUID

// Or use business data
key := cp.GenerateIdempotencyKeyDeterministic("user-123", "order-456")
```

### ✅ DO: Validate External Keys

```go
// Always validate keys from external sources
if err := ValidateIdempotencyKey(externalKey); err != nil {
    return err
}
```

### ❌ DON'T: Use Time-Based Keys

```go
// Never use current time
badKey := fmt.Sprintf("key:%d", time.Now().UnixNano())  // Will fail validation
```

### ❌ DON'T: Use Random UUIDs Directly

```go
import "github.com/google/uuid"

// Don't use non-deterministic UUID libraries
badKey := fmt.Sprintf("key:%s", uuid.New().String())  // Random, not deterministic

// Instead, use Restate's deterministic UUID
goodKey := fmt.Sprintf("key:%s", restate.UUID(ctx).String())  // Deterministic
```

## 10. Monitoring and Observability

### Log Analysis

Monitor your logs for validation warnings:

```bash
# Search for validation warnings in production logs
grep "invalid idempotency key detected" application.log

# Count occurrences
grep -c "non-deterministic timestamp" application.log
```

### Metrics to Track

Consider tracking these metrics:

1. **Validation failure rate**: How often does validation detect issues?
2. **Timestamp pattern detections**: How many keys contain suspicious timestamps?
3. **Empty key rejections**: How often are empty keys submitted?
4. **Service/handler breakdown**: Which services/handlers have the most issues?

## Conclusion

The framework now provides **three layers** of protection against non-deterministic ID usage:

1. **Automatic validation** in `ServiceClient.Send()` - catches issues at call time
2. **Manual validation** via `ValidateIdempotencyKey()` - for explicit checks
3. **Deterministic generation** via `GenerateIdempotencyKey()` - prevents issues at source

This defense-in-depth approach helps ensure idempotency keys remain deterministic across retries and replays, aligning with Restate's core durability guarantees.
