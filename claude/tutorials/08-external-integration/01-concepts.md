# Concepts: External Integration Patterns

> **Learn how to safely integrate with external APIs and services**

## 🎯 What You'll Learn

- Challenges of external API integration
- Journaling external calls for safety
- Webhook processing patterns
- Rate limiting and retry strategies
- Error handling best practices
- Service adapter patterns

---

## 📖 The External Integration Challenge

### What is External Integration?

**External integration** means connecting your service to third-party APIs and services that you don't control:

```
Your Restate Service
    ↓
External APIs (not under your control)
    ├─ Stripe (payments)
    ├─ SendGrid (emails)
    ├─ Twilio (SMS)
    ├─ Slack (notifications)
    └─ Any HTTP API
```

### Why Is It Challenging?

External services introduce unique problems:

#### 1. Network Failures

```
Your Request → [Network Timeout] → External API
❌ Did it succeed? Unknown!
❌ Should I retry? Maybe duplicate!
❌ How to recover? Unclear!
```

#### 2. Non-Idempotent APIs

```go
// Calling twice = different results
POST /api/createUser
→ Creates user (first call)
→ Error: user exists (second call)

// Calling twice = duplicate operations
POST /api/chargeCard
→ Charges $100 (first call)
→ Charges $100 again! (duplicate)
```

#### 3. Unpredictable Responses

```
Status 200: Success
Status 400: Bad request (terminal error)
Status 429: Rate limited (retry later)
Status 500: Server error (retry)
Status 503: Service unavailable (retry with backoff)
```

#### 4. Asynchronous Workflows

```
1. You call API
2. API returns "processing..."
3. Later: API sends webhook when done
4. You must correlate webhook with original request
```

---

## 🛡️ The Restate Solution

### Journaling External Calls

**Problem:** External calls are non-deterministic

```go
// ❌ Direct call - executes on every retry
func ProcessPayment(amount int) string {
    chargeID := stripe.Charge(amount)  // Charges again on retry!
    return chargeID
}
```

**Solution:** Wrap in `restate.Run()`

```go
// ✅ Journaled call - executes once, result cached
func ProcessPayment(ctx restate.ObjectContext, amount int) (string, error) {
    chargeID, err := restate.Run(ctx, func(ctx restate.RunContext) (string, error) {
        return stripe.Charge(amount), nil  // Executes ONCE
    })
    // On retry: returns journaled chargeID, no duplicate charge!
    return chargeID, err
}
```

**How It Works:**

```
First Execution:
1. Call restate.Run()
2. Execute side effect (call API)
3. Journal result
4. Return result

On Retry/Replay:
1. Call restate.Run()
2. Check journal (result exists!)
3. Return journaled result
4. ✅ Side effect NOT executed again
```

### Benefits

- ✅ **Exactly-once execution** - API called once
- ✅ **Deterministic replay** - Same result on retry
- ✅ **Automatic deduplication** - No duplicate operations
- ✅ **State consistency** - Results match reality

---

## 🔄 Integration Patterns

### Pattern 1: Simple HTTP Client

For basic GET/POST requests:

```go
type HTTPClient struct {
    baseURL string
    apiKey  string
}

// Journaled HTTP GET
func (c *HTTPClient) Get(
    ctx restate.ObjectContext,
    path string,
) (Response, error) {
    return restate.Run(ctx, func(ctx restate.RunContext) (Response, error) {
        url := c.baseURL + path
        req, _ := http.NewRequest("GET", url, nil)
        req.Header.Set("Authorization", "Bearer "+c.apiKey)
        
        resp, err := http.DefaultClient.Do(req)
        if err != nil {
            return Response{}, err
        }
        defer resp.Body.Close()
        
        if resp.StatusCode >= 400 {
            return Response{}, fmt.Errorf("API error: %d", resp.StatusCode)
        }
        
        var result Response
        json.NewDecoder(resp.Body).Decode(&result)
        return result, nil
    })
}
```

### Pattern 2: Service Adapter

Encapsulate external service logic:

```go
type StripeAdapter struct {
    apiKey string
}

// High-level business operation
func (a *StripeAdapter) CreateCharge(
    ctx restate.ObjectContext,
    amount int,
    currency string,
    customerID string,
) (ChargeResult, error) {
    // Wrap entire operation
    return restate.Run(ctx, func(ctx restate.RunContext) (ChargeResult, error) {
        // Handle authentication
        client := stripe.New(a.apiKey)
        
        // Make API call
        charge, err := client.Charges.New(&stripe.ChargeParams{
            Amount:   stripe.Int64(int64(amount)),
            Currency: stripe.String(currency),
            Customer: stripe.String(customerID),
        })
        
        if err != nil {
            // Convert Stripe error to our error
            return ChargeResult{}, a.handleError(err)
        }
        
        // Convert Stripe response to our model
        return ChargeResult{
            ID:     charge.ID,
            Amount: int(charge.Amount),
            Status: string(charge.Status),
        }, nil
    })
}

func (a *StripeAdapter) handleError(err error) error {
    if stripeErr, ok := err.(*stripe.Error); ok {
        switch stripeErr.Code {
        case stripe.ErrorCodeCardDeclined:
            return restate.TerminalError(fmt.Errorf("card declined"), 400)
        case stripe.ErrorCodeRateLimit:
            return fmt.Errorf("rate limited")  // Retryable
        default:
            return err
        }
    }
    return err
}
```

### Pattern 3: Multi-Step Integration

Coordinate multiple API calls:

```go
func (OrderService) ProcessOrder(
    ctx restate.ObjectContext,
    order Order,
) (OrderResult, error) {
    orderID := restate.Key(ctx)
    
    // Step 1: Charge customer (journaled)
    chargeID, err := restate.Run(ctx, func(ctx restate.RunContext) (string, error) {
        return stripeClient.Charge(order.Amount)
    })
    if err != nil {
        return OrderResult{}, fmt.Errorf("payment failed: %w", err)
    }
    
    // Step 2: Reserve inventory (journaled)
    reservationID, err := restate.Run(ctx, func(ctx restate.RunContext) (string, error) {
        return inventoryAPI.Reserve(order.Items)
    })
    if err != nil {
        // Compensate: refund charge
        restate.Run(ctx, func(ctx restate.RunContext) (bool, error) {
            return stripeClient.Refund(chargeID), nil
        })
        return OrderResult{}, fmt.Errorf("inventory unavailable: %w", err)
    }
    
    // Step 3: Create shipping label (journaled)
    labelID, err := restate.Run(ctx, func(ctx restate.RunContext) (string, error) {
        return shippoAPI.CreateLabel(order.Shipping)
    })
    if err != nil {
        // Continue anyway, can create label later
        ctx.Log().Warn("Failed to create shipping label", "error", err)
    }
    
    // Step 4: Send confirmation (journaled)
    _, err = restate.Run(ctx, func(ctx restate.RunContext) (string, error) {
        return sendgridAPI.Send(order.Customer.Email, "Order confirmed!")
    })
    
    return OrderResult{
        OrderID:       orderID,
        ChargeID:      chargeID,
        ReservationID: reservationID,
        LabelID:       labelID,
        Status:        "confirmed",
    }, nil
}
```

**Key Points:**
- Each API call is journaled
- Failures handled at each step
- Compensation logic for rollback
- Entire workflow is deterministic

---

## 🪝 Webhook Processing

### What Are Webhooks?

**Webhooks** are HTTP callbacks from external services to notify you of events:

```
1. You call Stripe API to charge customer
2. Stripe returns "charge created" immediately
3. Later: Card processing completes
4. Stripe POSTs webhook to your endpoint
5. Your service updates order status
```

### Challenges

#### 1. Duplicate Webhooks

Services may send the same webhook multiple times:

```
Stripe sends: payment_succeeded webhook
Network timeout
Stripe retries: payment_succeeded webhook (same event!)
```

#### 2. Out-of-Order Delivery

```
Event 1: payment_created (sent second)
Event 2: payment_succeeded (sent first)
```

#### 3. Signature Verification

Must validate webhook is authentic:

```go
// Verify webhook came from Stripe
signature := r.Header.Get("Stripe-Signature")
if !stripe.VerifySignature(payload, signature, secret) {
    return error("invalid signature")
}
```

### Webhook Processing Pattern

```go
type WebhookProcessor struct{}

func (WebhookProcessor) ProcessStripeWebhook(
    ctx restate.ObjectContext,
    webhook StripeWebhook,
) (WebhookResult, error) {
    webhookID := restate.Key(ctx)  // Use webhook.ID as key
    
    // Check if already processed (IDEMPOTENT)
    existing, _ := restate.Get[*WebhookResult](ctx, "result")
    if existing != nil {
        ctx.Log().Info("Webhook already processed")
        return *existing, nil  // Return immediately
    }
    
    // Process based on event type
    var result WebhookResult
    
    switch webhook.Type {
    case "payment_succeeded":
        result = processPaymentSucceeded(ctx, webhook)
    case "payment_failed":
        result = processPaymentFailed(ctx, webhook)
    default:
        ctx.Log().Warn("Unknown webhook type", "type", webhook.Type)
    }
    
    // Store result (prevents reprocessing)
    restate.Set(ctx, "result", result)
    
    return result, nil
}

func processPaymentSucceeded(
    ctx restate.ObjectContext,
    webhook StripeWebhook,
) WebhookResult {
    paymentID := webhook.Data["payment_id"].(string)
    
    // Update order status (idempotent service call)
    _, err := restate.Service[bool](
        ctx,
        "OrderService",
        "MarkAsPaid",
    ).Request(paymentID)
    
    if err != nil {
        return WebhookResult{Status: "failed", Error: err.Error()}
    }
    
    // Send confirmation email (journaled)
    _, err = restate.Run(ctx, func(ctx restate.RunContext) (bool, error) {
        sendConfirmationEmail(paymentID)
        return true, nil
    })
    
    return WebhookResult{
        WebhookID: webhook.ID,
        Type:      webhook.Type,
        Status:    "processed",
    }
}
```

**Key Points:**
- Use webhook ID as virtual object key
- Check for existing result first (idempotent)
- Store processing result
- Handle different event types
- Journal any side effects

---

## ⚡ Rate Limiting & Retry Strategies

### Understanding Rate Limits

APIs limit requests per time period:

```
Stripe: 100 requests/second
SendGrid: 600 emails/minute
Twilio: 1 SMS/second per number
```

**Responses:**
```
HTTP 429 Too Many Requests
Retry-After: 30
```

### Handling Rate Limits

#### Strategy 1: Exponential Backoff

```go
func callAPIWithBackoff(
    ctx restate.RunContext,
    fn func() (Response, error),
) (Response, error) {
    maxRetries := 5
    baseDelay := time.Second
    
    for attempt := 0; attempt < maxRetries; attempt++ {
        resp, err := fn()
        
        if err == nil {
            return resp, nil
        }
        
        // Check if rate limited
        if isRateLimited(err) {
            delay := baseDelay * time.Duration(math.Pow(2, float64(attempt)))
            ctx.Log().Info("Rate limited, backing off",
                "attempt", attempt,
                "delay", delay)
            time.Sleep(delay)
            continue
        }
        
        // Other error, fail immediately
        return Response{}, err
    }
    
    return Response{}, fmt.Errorf("max retries exceeded")
}
```

#### Strategy 2: Respect Retry-After

```go
if resp.StatusCode == 429 {
    retryAfter := resp.Header.Get("Retry-After")
    if duration, err := time.ParseDuration(retryAfter + "s"); err == nil {
        time.Sleep(duration)
        // Retry...
    }
}
```

#### Strategy 3: Token Bucket

```go
type RateLimiter struct {
    tokens int64
    max    int64
    refill time.Duration
}

func (rl *RateLimiter) Acquire(ctx context.Context) error {
    for atomic.LoadInt64(&rl.tokens) <=0 {
        select {
        case <-time.After(rl.refill):
            atomic.StoreInt64(&rl.tokens, rl.max)
        case <-ctx.Done():
            return ctx.Err()
        }
    }
    atomic.AddInt64(&rl.tokens, -1)
    return nil
}
```

### Restate's Built-In Retries

Restate automatically retries failed operations:

```go
// Restate retries this automatically on transient failures
result, err := restate.Run(ctx, func(ctx restate.RunContext) (T, error) {
    return externalAPI.Call()
})

// Control retry behavior
return restate.TerminalError(err, 400)  // Don't retry (4xx)
return err  // Retry (5xx, network errors)
```

---

## 🎯 Error Handling Strategies

### Classifying Errors

Not all errors should be retried:

```go
func handleAPIError(err error) error {
    switch {
    case isNetworkError(err):
        return err  // Retry
        
    case isTimeout(err):
        return err  // Retry
        
    case isServerError(err):  // 5xx
        return err  // Retry
        
    case isClientError(err):  // 4xx
        return restate.TerminalError(err)  // Don't retry
        
    case isAuthError(err):  // 401, 403
        return restate.TerminalError(err)  // Don't retry
        
    default:
        return err  //Retry by default
    }
}
```

### Error Classification Matrix

| Error Type | Status | Action | Example |
|------------|--------|--------|---------|
| **Network** | - | Retry | Connection refused, DNS failure |
| **Timeout** | - | Retry | Read timeout, deadline exceeded |
| **Server Error** | 500-599 | Retry | Internal server error, service unavailable |
| **Rate Limit** | 429 | Retry (backoff) | Too many requests |
| **Client Error** | 400-499 | Terminal | Bad request, not found |
| **Auth Error** | 401, 403 | Terminal | Unauthorized, forbidden |
| **Validation** | 422 | Terminal | Invalid input |

### Implementing Error Handling

```go
func callExternalAPI(
    ctx restate.ObjectContext,
    req Request,
) (Response, error) {
    return restate.Run(ctx, func(ctx restate.RunContext) (Response, error) {
        resp, err := http.Post(url, "application/json", body)
        
        if err != nil {
            // Network error - retry
            return Response{}, err
        }
        
        defer resp.Body.Close()
        
        // Check status code
        switch {
        case resp.StatusCode >= 200 && resp.StatusCode < 300:
            // Success
            var result Response
            json.NewDecoder(resp.Body).Decode(&result)
            return result, nil
            
        case resp.StatusCode == 429:
            // Rate limited - retry with backoff
            ctx.Log().Warn("Rate limited")
            return Response{}, fmt.Errorf("rate limited")
            
        case resp.StatusCode >= 500:
            // Server error - retry
            return Response{}, fmt.Errorf("server error: %d", resp.StatusCode)
            
        case resp.StatusCode >= 400:
            // Client error - terminal
            body, _ := io.ReadAll(resp.Body)
            return Response{}, restate.TerminalError(
                fmt.Errorf("client error: %s", body), resp.StatusCode)
        }
        
        return Response{}, fmt.Errorf("unexpected status: %d", resp.StatusCode)
    })
}
```

---

## 🔐 Authentication Patterns

### API Key Authentication

```go
type APIClient struct {
    apiKey string
}

func (c *APIClient) request(
    ctx restate.RunContext,
    method, path string,
    body interface{},
) (Response, error) {
    req, _ := http.NewRequest(method, c.baseURL+path, marshalBody(body))
    req.Header.Set("Authorization", "Bearer "+c.apiKey)
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := http.DefaultClient.Do(req)
    // ... handle response
}
```

### OAuth 2.0 with Token Refresh

```go
type OAuthClient struct {
    clientID     string
    clientSecret string
    accessToken  string
    refreshToken string
    expiresAt    time.Time
}

func (c *OAuthClient) ensureValidToken(ctx restate.ObjectContext) error {
    if time.Now().Before(c.expiresAt) {
        return nil  // Token still valid
    }
    
    // Refresh token (journaled!)
    tokens, err := restate.Run(ctx, func(ctx restate.RunContext) (Tokens, error) {
        return c.refreshAccessToken()
    })
    
    if err != nil {
        return err
    }
    
    c.accessToken = tokens.AccessToken
    c.refreshToken = tokens.RefreshToken
    c.expiresAt = time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)
    
    return nil
}
```

---

## ✅ Best Practices Summary

### DO's ✅

1. **Always journal external calls**
   ```go
   restate.Run(ctx, func(ctx restate.RunContext) (T, error) {
       return externalAPI.Call()
   })
   ```

2. **Process webhooks idempotently**
   ```go
   existing, _ := restate.Get[*Result](ctx, "result")
   if existing != nil { return existing }
   ```

3. **Classify errors correctly**
   ```go
   if isClientError(err) {
       return restate.TerminalError(err)
   }
   ```

4. **Use service adapters**
   ```go
   type StripeAdapter struct { /* ... */ }
   ```

5. **Handle rate limits gracefully**
   ```go
   if isRateLimited(err) {
       // Backoff and retry
   }
   ```

### DON'Ts ❌

1. **Don't call APIs directly**
   ```go
   // ❌ NOT journaled
   stripe.Charge(amount)
   ```

2. **Don't process webhooks multiple times**
   ```go
   // ❌ No deduplication
   processWebhookWithoutChecking(webhook)
   ```

3. **Don't retry terminal errors**
   ```go
   // ❌ Retrying 400 errors wastes resources
   if resp.StatusCode == 400 {
       return err  // Should be TerminalError!
   }
   ```

4. **Don't ignore error classification**
   ```go
   // ❌ All errors treated the same
   if err != nil { return err }
   ```

---

## 🚀 Next Steps

You now understand external integration patterns!

👉 **Continue to [Hands-On Tutorial](./02-hands-on.md)**

Build a real e-commerce integration service!

---

**Questions?** Review this document or check the [module README](./README.md).
