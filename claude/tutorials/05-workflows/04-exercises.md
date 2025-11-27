# Exercises: Practice Workflows

> **Master long-running orchestrations with hands-on exercises**

## 🎯 Objectives

Practice:
- Building workflows with durable promises
- Human-in-the-loop patterns
- Timeout handling
- Multi-step orchestrations
- Sequential and parallel promise patterns

## 📚 Exercise Levels

- 🟢 **Beginner** - Basic workflow patterns
- 🟡 **Intermediate** - Multi-step workflows
- 🔴 **Advanced** - Complex orchestrations

---

## Exercise 1: Email Verification Workflow 🟢

**Goal:** Build a user signup workflow with email verification

### Requirements

1. Create `EmailVerificationWorkflow` with:
   - `Run(user)` - Main workflow
   - `VerifyEmail(token)` - Verify email with token
   - `GetStatus()` - Check verification status

2. Flow:
   - Send verification email
   - Wait for user to click link (max 24 hours)
   - If verified: activate account
   - If timeout: mark as expired

### Starter Code

```go
type EmailVerificationWorkflow struct{}

type User struct {
    Email     string `json:"email"`
    Name      string `json:"name"`
    SignupAt  time.Time `json:"signupAt"`
}

type VerificationToken struct {
    Token      string `json:"token"`
    VerifiedAt time.Time `json:"verifiedAt"`
}

type VerificationResult struct {
    Status      string `json:"status"` // "verified", "expired"
    VerifiedAt  *time.Time `json:"verifiedAt,omitempty"`
}

func (EmailVerificationWorkflow) Run(
    ctx restate.WorkflowContext,
    user User,
) (VerificationResult, error) {
    // TODO: Send verification email (side effect)
    token := generateToken(ctx)
    sendVerificationEmail(ctx, user.Email, token)
    
    // TODO: Create promise for verification
    promise := restate.Promise[VerificationToken](ctx, "verification")
    
    // TODO: Create 24-hour timeout
    timeout := restate.After(ctx, 24*time.Hour)
    
    // TODO: Wait for first to complete
    winner, _ := restate.WaitFirst(ctx, promise, timeout)
    
    // TODO: Handle result
    switch winner {
    case promise:
        // Verified!
        verification, _ := promise.Result()
        return VerificationResult{
            Status: "verified",
            VerifiedAt: &verification.VerifiedAt,
        }, nil
    case timeout:
        // Expired
        return VerificationResult{Status: "expired"}, nil
    }
    
    return VerificationResult{}, nil
}

func (EmailVerificationWorkflow) VerifyEmail(
    ctx restate.WorkflowSharedContext,
    token VerificationToken,
) error {
    // TODO: Resolve promise
    return restate.Promise[VerificationToken](ctx, "verification").
        Resolve(token)
}

func (EmailVerificationWorkflow) GetStatus(
    ctx restate.WorkflowSharedContext,
    _ restate.Void,
) (string, error) {
    // TODO: Return current status
}
```

### Test

```bash
# Start workflow
curl -X POST http://localhost:9080/EmailVerificationWorkflow/user123/Run \
  -H 'Content-Type: application/json' \
  -d '{
    "email": "alice@example.com",
    "name": "Alice",
    "signupAt": "2024-01-15T10:00:00Z"
  }'

# User clicks verification link
curl -X POST http://localhost:9080/EmailVerificationWorkflow/user123/VerifyEmail \
  -H 'Content-Type: application/json' \
  -d '{
    "token": "abc123",
    "verifiedAt": "2024-01-15T10:05:00Z"
  }'
```

---

## Exercise 2: Order Fulfillment Workflow 🟡

**Goal:** Multi-step order processing with external events

### Requirements

1. Create `OrderFulfillmentWorkflow` with handlers:
   - `Run(order)` - Main workflow
   - `ConfirmPayment(payment)` - External payment confirmation
   - `ConfirmShipment(tracking)` - External shipping confirmation
   - `GetStatus()` - Query order status

2. Flow (sequential):
   - Wait for payment confirmation (max 1 hour)
   - If payment confirmed: Wait for shipment (max 48 hours)
   - If any timeout: Cancel order

### Starter Code

```go
type OrderFulfillmentWorkflow struct{}

type Order struct {
    OrderID string  `json:"orderId"`
    Items   []Item  `json:"items"`
    Total   float64 `json:"total"`
}

type Payment struct {
    PaymentID string    `json:"paymentId"`
    Amount    float64   `json:"amount"`
    PaidAt    time.Time `json:"paidAt"`
}

type Shipment struct {
    TrackingNumber string    `json:"trackingNumber"`
    ShippedAt      time.Time `json:"shippedAt"`
}

type FulfillmentResult struct {
    Status         string `json:"status"` // "completed", "cancelled"
    PaymentID      string `json:"paymentId,omitempty"`
    TrackingNumber string `json:"trackingNumber,omitempty"`
}

func (OrderFulfillmentWorkflow) Run(
    ctx restate.WorkflowContext,
    order Order,
) (FulfillmentResult, error) {
    // TODO: Step 1 - Wait for payment (1 hour timeout)
    paymentPromise := restate.Promise[Payment](ctx, "payment")
    paymentTimeout := restate.After(ctx, 1*time.Hour)
    
    winner, _ := restate.WaitFirst(ctx, paymentPromise, paymentTimeout)
    if winner == paymentTimeout {
        return FulfillmentResult{Status: "cancelled"}, nil
    }
    
    payment, _ := paymentPromise.Result()
    
    // TODO: Step 2 - Wait for shipment (48 hour timeout)
    shipmentPromise := restate.Promise[Shipment](ctx, "shipment")
    shipmentTimeout := restate.After(ctx, 48*time.Hour)
    
    winner, _ = restate.WaitFirst(ctx, shipmentPromise, shipmentTimeout)
    if winner == shipmentTimeout {
        // Cancel - refund payment
        return FulfillmentResult{Status: "cancelled"}, nil
    }
    
    shipment, _ := shipmentPromise.Result()
    
    // TODO: Complete!
    return FulfillmentResult{
        Status:         "completed",
        PaymentID:      payment.PaymentID,
        TrackingNumber: shipment.TrackingNumber,
    }, nil
}

// TODO: Implement ConfirmPayment, ConfirmShipment, GetStatus
```

---

## Exercise 3: Multi-Approver Workflow 🔴

**Goal:** Document requires approval from multiple people

### Requirements

1. Create `MultiApproverWorkflow`:
   - `Run(document, approvers)` - Start workflow
   - `Approve(approver, decision)` - Approve/reject by specific approver
   - `GetApprovalStatus()` - View approval progress

2. Logic:
   - Need approval from ALL approvers
   - ANY rejection cancels entire process
   - 48-hour timeout for each approver

### Starter Code

```go
type MultiApproverWorkflow struct{}

type ApprovalRequest struct {
    Document  Document `json:"document"`
    Approvers []string `json:"approvers"` // List of approver IDs
}

type ApproverDecision struct {
    ApproverID string `json:"approverId"`
    Approved   bool   `json:"approved"`
    Comments   string `json:"comments"`
}

type MultiApprovalResult struct {
    Status     string             `json:"status"` // "approved", "rejected", "timeout"
    Decisions  []ApproverDecision `json:"decisions"`
}

func (MultiApproverWorkflow) Run(
    ctx restate.WorkflowContext,
    req ApprovalRequest,
) (MultiApprovalResult, error) {
    // TODO: Create promise for each approver
    var promises []restate.Promise[ApproverDecision]
    for _, approver := range req.Approvers {
        p := restate.Promise[ApproverDecision](ctx, "approval_"+approver)
        promises = append(promises, p)
    }
    
    // TODO: Create timeout
    timeout := restate.After(ctx, 48*time.Hour)
    
    // TODO: Wait for all approvals (or first rejection/timeout)
    // Hint: Use loop to check each promise as it completes
    
    // TODO: Return result
}

func (MultiApproverWorkflow) Approve(
    ctx restate.WorkflowSharedContext,
    decision ApproverDecision,
) error {
    // TODO: Resolve the specific approver's promise
    promiseName := "approval_" + decision.ApproverID
    return restate.Promise[ApproverDecision](ctx, promiseName).
        Resolve(decision)
}
```

---

## Exercise 4: Scheduled Task Workflow 🟡

**Goal:** Execute task at specific time

### Requirements

1. Create `ScheduledTaskWorkflow`:
   - `Schedule(task, executeAt)` - Schedule task for future
   - `Cancel()` - Cancel before execution
   - `GetStatus()` - Check if executed

2. Logic:
   - Wait until specified time
   - Execute task
   - Allow cancellation before execution

### Starter Code

```go
type ScheduledTaskWorkflow struct{}

type ScheduledTask struct {
    TaskID    string    `json:"taskId"`
    Action    string    `json:"action"`
    ExecuteAt time.Time `json:"executeAt"`
}

type TaskResult struct {
    Status      string    `json:"status"` // "executed", "cancelled"
    ExecutedAt  *time.Time `json:"executedAt,omitempty"`
}

func (ScheduledTaskWorkflow) Schedule(
    ctx restate.WorkflowContext,
    task ScheduledTask,
) (TaskResult, error) {
    // TODO: Calculate delay until executeAt
    delay := time.Until(task.ExecuteAt)
    
    // TODO: Create sleep for delay
    sleepFuture := restate.After(ctx, delay)
    
    // TODO: Create promise for cancellation
    cancelPromise := restate.Promise[bool](ctx, "cancel")
    
    // TODO: Wait for sleep or cancellation
    winner, _ := restate.WaitFirst(ctx, sleepFuture, cancelPromise)
    
    switch winner {
    case sleepFuture:
        // Time to execute!
        executeTask(ctx, task)
        now := time.Now()
        return TaskResult{
            Status:     "executed",
            ExecutedAt: &now,
        }, nil
    case cancelPromise:
        // Cancelled
        return TaskResult{Status: "cancelled"}, nil
    }
    
    return TaskResult{}, nil
}

func (ScheduledTaskWorkflow) Cancel(
    ctx restate.WorkflowSharedContext,
    _ restate.Void,
) error {
    // TODO: Resolve cancellation promise
    return restate.Promise[bool](ctx, "cancel").Resolve(true)
}
```

---

## Exercise 5: Saga Pattern - Travel Booking 🔴

**Goal:** Coordinate multiple services with compensation

### Requirements

1. Create `TravelBookingWorkflow`:
   - Book flight, hotel, and car rental
   - If any fails: Compensating rollback all previous bookings
   - Use promises for external booking confirmations

2. Flow:
   - Reserve flight → confirm or timeout (1 hour)
   - Reserve hotel → confirm or timeout (1 hour)
   - Reserve car → confirm or timeout (1 hour)
   - If any failure: Cancel all previous reservations

### Starter Code

```go
type TravelBookingWorkflow struct{}

type BookingRequest struct {
    TripID     string `json:"tripId"`
    FlightID   string `json:"flightId"`
    HotelID    string `json:"hotelId"`
    CarID      string `json:"carId"`
}

type Confirmation struct {
    BookingID     string `json:"bookingId"`
    ConfirmedAt   time.Time `json:"confirmedAt"`
}

type BookingResult struct {
    Status             string `json:"status"`
    FlightConfirmation string `json:"flightConfirmation,omitempty"`
    HotelConfirmation  string `json:"hotelConfirmation,omitempty"`
    CarConfirmation    string `json:"carConfirmation,omitempty"`
}

func (TravelBookingWorkflow) Run(
    ctx restate.WorkflowContext,
    req BookingRequest,
) (BookingResult, error) {
    var bookings []string // Track successful bookings for rollback
    
    // TODO: Try to book flight
   flightPromise := restate.Promise[Confirmation](ctx, "flight")
    timeout1 := restate.After(ctx, 1*time.Hour)
    
    winner, _ := restate.WaitFirst(ctx, flightPromise, timeout1)
    if winner == timeout1 {
        return BookingResult{Status: "failed"}, nil
    }
    
    flightConfirm, _ := flightPromise.Result()
    bookings = append(bookings, flightConfirm.BookingID)
    
    // TODO: Try to book hotel
    hotelPromise := restate.Promise[Confirmation](ctx, "hotel")
    timeout2 := restate.After(ctx, 1*time.Hour)
    
    winner, _ = restate.WaitFirst(ctx, hotelPromise, timeout2)
    if winner == timeout2 {
        // Cancel flight
        cancelBooking(ctx, flightConfirm.BookingID)
        return BookingResult{Status: "failed"}, nil
    }
    
    hotelConfirm, _ := hotelPromise.Result()
    bookings = append(bookings, hotelConfirm.BookingID)
    
    // TODO: Try to book car
    // TODO: If fails, cancel all previous bookings
    
    // TODO: All succeeded!
}

// TODO: Implement confirmation handlers
```

---

## Exercise 6: Drip Campaign Workflow 🟡

**Goal:** Send series of emails over time

### Requirements

1. Create `DripCampaignWorkflow`:
   - Send email sequence over days/weeks
   - Allow user to unsubscribe (cancels future emails)
   - Track email opens/clicks

### Starter Code

```go
type DripCampaignWorkflow struct{}

type Campaign struct {
    UserID    string  `json:"userId"`
    Emails    []Email `json:"emails"` // Sequence of emails
}

type Email struct {
    Subject string        `json:"subject"`
    Body    string        `json:"body"`
    Delay   time.Duration `json:"delay"` // Wait before sending
}

func (DripCampaignWorkflow) Run(
    ctx restate.WorkflowContext,
    campaign Campaign,
) error {
    // Create unsubscribe promise
    unsubPromise := restate.Promise[bool](ctx, "unsubscribe")
    
    // TODO: Loop through emails
    for i, email := range campaign.Emails {
        // Wait for delay or unsubscribe
        sleepFuture := restate.After(ctx, email.Delay)
        
        winner, _ := restate.WaitFirst(ctx, sleepFuture, unsubPromise)
        
        if winner == unsubPromise {
            // User unsubscribed - stop
            return nil
        }
        
        // Send email
        sendEmail(ctx, campaign.UserID, email)
    }
    
    return nil
}

func (DripCampaignWorkflow) Unsubscribe(
    ctx restate.WorkflowSharedContext,
    _ restate.Void,
) error {
    // TODO: Resolve unsubscribe promise
    return restate.Promise[bool](ctx, "unsubscribe").Resolve(true)
}
```

---

## ✅ Exercise Checklist

- [ ] Exercise 1: Email Verification (Beginner)
- [ ] Exercise 2: Order Fulfillment (Intermediate)
- [ ] Exercise 3: Multi-Approver (Advanced)
- [ ] Exercise 4: Scheduled Task (Intermediate)
- [ ] Exercise 5: Travel Booking Saga (Advanced)
- [ ] Exercise 6: Drip Campaign (Intermediate)

## 📁 Solutions

Complete solutions available in [solutions/](./solutions/):

- `exercise1_email_verification.go`
- `exercise2_order_fulfillment.go`
- `exercise3_multi_approver.go`
- `exercise4_scheduled_task.go`
- `exercise5_travel_saga.go`
- `exercise6_drip_campaign.go`

## 🎯 Next Module

Congratulations! You've mastered Workflows!

You now understand:
- ✅ Long-running orchestrations
- ✅ Durable promises
- ✅ Human-in-the-loop patterns
- ✅ Timeout handling
- ✅ Multi-step workflows
- ✅ Saga patterns

Ready to learn about distributed transactions?

👉 **Continue to [Module 6: Sagas](../06-sagas/README.md)**

Learn to build reliable distributed transactions with compensation!

---

**Questions?** Review [concepts](./01-concepts.md) or check the [complete code](./code/)!
