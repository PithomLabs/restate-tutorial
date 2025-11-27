# Exercises: Idempotency Practice

> **Build your skills with hands-on idempotency challenges**

## 🎯 Exercise Overview

| Exercise | Difficulty | Focus | Time |
|----------|-----------|-------|------|
| 1. Idempotent Order Service | ⭐ | State-based deduplication | 20 min |
| 2. Email Service with Deduplication | ⭐⭐ | Side effect idempotency | 25 min |
| 3. Inventory Reservation | ⭐⭐ | Complex state management | 30 min |
| 4. Subscription Service | ⭐⭐⭐ | Multi-step idempotency | 40 min |
| 5. Idempotent Webhook Handler | ⭐⭐ | External event processing | 25 min |
| 6. Distributed Transaction | ⭐⭐⭐ | Cross-service idempotency | 45 min |

---

## Exercise 1: Idempotent Order Service ⭐

### Objective

Create an `OrderService` virtual object with idempotent order creation.

### Requirements

1. **CreateOrder** handler that:
   - Accepts order details (items, customer ID, total amount)
   - Checks if order already exists
   - Returns existing order if already created
   - Creates new order if it doesn't exist
   - Assigns order number using deterministic generation

2. **GetOrder** handler (read-only)

3. **CancelOrder** handler (idempotent cancellation)

### Data Structures

```go
type OrderRequest struct {
    Items      []OrderItem `json:"items"`
    CustomerID string      `json:"customerId"`
    Total      int         `json:"total"`
}

type OrderItem struct {
    ProductID string `json:"productId"`
    Quantity  int    `json:"quantity"`
    Price     int    `json:"price"`
}

type Order struct {
    OrderID     string      `json:"orderId"`
    OrderNumber string      `json:"orderNumber"`
    Items       []OrderItem `json:"items"`
    CustomerID  string      `json:"customerId"`
    Total       int         `json:"total"`
    Status      string      `json:"status"` // "pending", "confirmed", "cancelled"
    CreatedAt   time.Time   `json:"createdAt"`
}
```

### Hints

- Use object key as order ID
- Check for existing order before creating
- Use `restate.Rand()` for deterministic order numbers
- Track order status to prevent invalid transitions

### Success Criteria

- [x] Duplicate CreateOrder returns same order
- [x] Order number is deterministic (same on retry)
- [x] CancelOrder is idempotent
- [x] Cannot cancel already-cancelled order twice

### Bonus Challenge 🌟

Add `UpdateOrder` handler that allows updating items only if order is "pending".

---

## Exercise 2: Email Service with Deduplication ⭐⭐

### Objective

Build an `EmailService` that ensures emails are sent exactly once, even with retries.

### Requirements

1. **SendEmail** handler that:
   - Accepts recipient, subject, body
   - Checks if email already sent (by email ID)
   - Sends email via external service (use `restate.Run`)
   - Records send timestamp
   - Returns send result

2. **GetEmailStatus** handler

3. **ResendEmail** handler (safe to call multiple times)

### Data Structures

```go
type EmailRequest struct {
    Recipient string `json:"recipient"`
    Subject   string `json:"subject"`
    Body      string `json:"body"`
}

type EmailRecord struct {
    EmailID   string    `json:"emailId"`
    Recipient string    `json:"recipient"`
    Subject   string    `json:"subject"`
    Status    string    `json:"status"` // "pending", "sent", "failed"
    SentAt    time.Time `json:"sentAt,omitempty"`
    MessageID string    `json:"messageId,omitempty"`
    Error     string    `json:"error,omitempty"`
}
```

### Mock Email Gateway

```go
func SendEmailViaSMTP(recipient, subject, body string) (messageID string, err error) {
    // Simulate network delay
    time.Sleep(50 * time.Millisecond)
    
    // Simulate 5% failure rate
    if rand.Float64() < 0.05 {
        return "", fmt.Errorf("SMTP timeout")
    }
    
    return fmt.Sprintf("msg_%s_%d", recipient, time.Now().Unix()), nil
}
```

### Hints

- Use email ID as virtual object key
- Wrap SMTP call in `restate.Run()`
- Store sent emails to prevent duplicates
- Handle failures gracefully (mark as "failed")

### Success Criteria

- [x] Email sent exactly once
- [x] Duplicate requests return same message ID
- [x] Failed sends can be retried with new email ID
- [x] ResendEmail is safe for already-sent emails

### Bonus Challenge 🌟

Add email templates and variable substitution (must be deterministic).

---

## Exercise 3: Inventory Reservation ⭐⭐

### Objective

Create an `InventoryService` that manages stock levels with idempotent reservations.

### Requirements

1. **ReserveStock** handler that:
   - Accepts product ID, quantity, reservation ID
   - Checks current stock levels
   - Checks if reservation already exists
   - Reserves stock if available
   - Returns reservation result

2. **ReleaseReservation** handler (idempotent release)

3. **GetStockLevel** handler

### Data Structures

```go
type ReservationRequest struct {
    ProductID     string `json:"productId"`
    Quantity      int    `json:"quantity"`
    ReservationID string `json:"reservationId"`
}

type Reservation struct {
    ReservationID string    `json:"reservationId"`
    ProductID     string    `json:"productId"`
    Quantity      int       `json:"quantity"`
    Status        string    `json:"status"` // "active", "released", "expired"
    CreatedAt     time.Time `json:"createdAt"`
}

type StockLevel struct {
    ProductID         string `json:"productId"`
    TotalStock        int    `json:"totalStock"`
    AvailableStock    int    `json:"availableStock"`
    ReservedStock     int    `json:"reservedStock"`
}
```

### State Management

```go
// State keys
"stock_level"               // StockLevel
"reservations"              // map[string]Reservation
"reservation:{id}"          // Individual reservation
```

### Hints

- Use product ID as virtual object key
- Track reservations separately
- Update available stock when reserving
- Check for existing reservation before creating new one
- Handle insufficient stock gracefully

### Success Criteria

- [x] Stock levels accurate
- [x] Duplicate reservations don't reduce stock twice
- [x] Releasing reservation restores stock
- [x] Concurrent reservations handled correctly

### Bonus Challenge 🌟

Add automatic reservation expiration after timeout (use `restate.Sleep`).

---

## Exercise 4: Subscription Service ⭐⭐⭐

### Objective

Build a complex `SubscriptionService` with multiple idempotent operations.

### Requirements

1. **CreateSubscription** handler:
   - Validates customer and plan
   - Charges initial payment (idempotent)
   - Creates subscription record
   - Sends welcome email (idempotent)

2. **CancelSubscription** handler:
   - Validates subscription exists and is active
   - Processes final prorated charge/refund
   - Marks subscription as cancelled
   - Sends cancellation email

3. **UpdatePlan** handler:
   - Changes subscription plan
   - Calculates prorated amount
   - Charges or refunds difference
   - Updates billing cycle

### Data Structures

```go
type SubscriptionRequest struct {
    CustomerID string `json:"customerId"`
    PlanID     string `json:"planId"`
    StartDate  string `json:"startDate"`
}

type Subscription struct {
    SubscriptionID string    `json:"subscriptionId"`
    CustomerID     string    `json:"customerId"`
    PlanID         string    `json:"planId"`
    Status         string    `json:"status"` // "active", "cancelled", "expired"
    StartDate      time.Time `json:"startDate"`
    EndDate        time.Time `json:"endDate,omitempty"`
    CreatedAt      time.Time `json:"createdAt"`
}

type Plan struct {
    PlanID      string `json:"planId"`
    Name        string `json:"name"`
    Price       int    `json:"price"`
    BillingCycle string `json:"billingCycle"` // "monthly", "yearly"
}
```

### Multi-Step Idempotency

Each step must be idempotent:

```go
// Step 1: Charge customer
chargeID := chargeCustomer()  // Journaled

// Step 2: Create subscription
createSubscription()  // State-based

// Step 3: Send email
sendEmail()  // Journaled
```

### Hints

- Use subscription ID as object key
- Check state at each step before proceeding
- Store intermediate results (charge ID, email ID)
- Make each operation idempotent independently

### Success Criteria

- [x] Creating subscription twice doesn't double-charge
- [x] Cancellation is idempotent
- [x] Plan updates handle edge cases
- [x] All emails sent exactly once
- [x] State is always consistent

### Bonus Challenge 🌟

Add automatic renewal using workflows with scheduled tasks.

---

## Exercise 5: Idempotent Webhook Handler ⭐⭐

### Objective

Create a `WebhookService` that processes external webhooks idempotently.

### Requirements

1. **ProcessWebhook** handler that:
   - Accepts webhook payload with unique ID
   - Checks if webhook already processed
   - Validates webhook signature
   - Processes webhook (business logic)
   - Stores result

2. **GetWebhookStatus** handler

### Scenario

You're receiving webhooks from Stripe for payment events:

```go
type StripeWebhook struct {
    ID        string                 `json:"id"`
    Type      string                 `json:"type"` // "payment.succeeded", "payment.failed"
    Data      map[string]interface{} `json:"data"`
    Timestamp int64                  `json:"timestamp"`
}
```

### Challenge

Stripe may send the same webhook multiple times (retries). Your handler must:
- Process each webhook exactly once
- Return success immediately if already processed
- Handle different webhook types

### Hints

- Use webhook ID as object key
- Store processing result
- Return early if already processed
- Use `restate.Run()` for any side effects

### Success Criteria

- [x] Webhook processed exactly once
- [x] Duplicate webhooks return success immediately
- [x] Different webhooks processed independently
- [x] Side effects (notifications, DB updates) happen once

### Bonus Challenge 🌟

Add webhook replay capability for debugging.

---

## Exercise 6: Distributed Transaction ⭐⭐⭐

### Objective

Implement an idempotent distributed transaction across multiple services.

### Scenario

Process an order that requires coordinating multiple services:

1. **OrderService** - Creates order
2. **InventoryService** - Reserves stock
3. **PaymentService** - Charges customer
4. **ShippingService** - Creates shipping label
5. **NotificationService** - Sends confirmation

All must be idempotent, and the entire transaction must be atomic.

### Requirements

```go
type OrderOrchestrator struct{}

func (OrderOrchestrator) ProcessOrder(
    ctx restate.ObjectContext,
    req OrderRequest,
) (OrderResult, error) {
    orderID := restate.Key(ctx)
    
    // Check if order already processed
    existingResult, _ := restate.Get[*OrderResult](ctx, "result")
    if existingResult != nil {
        return *existingResult, nil
    }
    
    // Step 1: Reserve inventory (idempotent call)
    reservation, err := restate.Service[Reservation](
        ctx,
        "InventoryService",
        "ReserveStock",
    ).Request(req.Items, restate.WithIdempotencyKey(orderID+"-inventory"))
    if err != nil {
        return OrderResult{}, err
    }
    
    // Step 2: Charge customer (idempotent call)
    charge, err := restate.Service[Charge](
        ctx,
        "PaymentService",
        "CreatePayment",
    ).Request(req.Payment, restate.WithIdempotencyKey(orderID+"-payment"))
    if err != nil {
        // Release inventory on payment failure
        restate.ServiceSend(ctx, "InventoryService", "ReleaseReservation").
            Send(reservation.ID)
        return OrderResult{}, err
    }
    
    // Step 3: Create shipping label (idempotent call)
    // Step 4: Send notification (idempotent call)
    // ...
    
    result := OrderResult{
        OrderID:       orderID,
        Status:        "completed",
        ReservationID: reservation.ID,
        ChargeID:      charge.ID,
    }
    
    restate.Set(ctx, "result", result)
    return result, nil
}
```

### Hints

- Use idempotency keys for each service call
- Chain operations carefully
- Handle rollback on failures
- Test retry scenarios extensively

### Success Criteria

- [x] Entire transaction is idempotent
- [x] Each service called with idempotency key
- [x] Failures trigger compensating actions
- [x] Retrying the transaction is safe
- [x] No duplicate operations across services

### Bonus Challenge 🌟

Convert this to a Saga pattern with compensation handlers (see Module 06).

---

## 🎓 Learning Objectives

After completing these exercises, you should be able to:

- ✅ Implement state-based deduplication
- ✅ Use `restate.Run()` for idempotent side effects
- ✅ Design idempotent APIs
- ✅ Handle complex multi-step transactions
- ✅ Coordinate idempotent operations across services
- ✅ Test idempotency guarantees

---

## 📚 Solutions

Solutions for all exercises are available in the [`solutions/`](./solutions/) directory.

Try solving the exercises yourself first before checking the solutions!

---

## 🚀 Next Steps

Completed the exercises?

- Review your solutions with the provided answers
- Test edge cases and retry scenarios
- Apply these patterns to your own projects
- Continue to the next module!

---

**Questions?** Review [concepts](./01-concepts.md) or [hands-on tutorial](./02-hands-on.md).
