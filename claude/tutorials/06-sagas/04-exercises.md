# Exercises: Practice Saga Patterns

> **Master distributed transactions with hands-on saga exercises**

## 🎯 Objectives

Practice:
- Building multi-step sagas
- Implementing compensation logic
- Handling partial failures
- Ensuring idempotency
- Managing saga state

## 📚 Exercise Levels

- 🟢 **Beginner** - Basic saga patterns
- 🟡 **Intermediate** - Complex compensations
- 🔴 **Advanced** - Advanced saga orchestrations

---

## Exercise 1: E-Commerce Order Saga 🟢

**Goal:** Build an order processing saga with payment, inventory, and shipping

### Requirements

1. Create `OrderSaga` with steps:
   - Charge payment
   - Reserve inventory
   - Create shipment
   - Send confirmation email

2. Compensation logic:
   - If inventory fails → refund payment
   - If shipment fails → release inventory + refund payment
   - If email fails → continue (non-critical)

### Starter Code

```go
type OrderSaga struct{}

type OrderRequest struct {
    OrderID    string  `json:"orderId"`
    CustomerID string  `json:"customerId"`
    Items      []Item  `json:"items"`
    Amount     float64 `json:"amount"`
}

type OrderResult struct {
    Status        string `json:"status"`
    PaymentID     string `json:"paymentId,omitempty"`
    ReservationID string `json:"reservationId,omitempty"`
    ShipmentID    string `json:"shipmentId,omitempty"`
}

func (OrderSaga) Run(
    ctx restate.WorkflowContext,
    order OrderRequest,
) (OrderResult, error) {
    // TODO: Implement saga
    // Step 1: Charge payment
    // Step 2: Reserve inventory (compensate payment on fail)
    // Step 3: Create shipment (compensate inventory+payment on fail)
    // Step 4: Send email (optional, don't fail saga)
    
    return OrderResult{Status: "completed"}, nil
}
```

### Test

```bash
curl -X POST http://localhost:9080/OrderSaga/order-001/Run \
  -H 'Content-Type: application/json' \
  -d '{
    "orderId": "order-001",
    "customerId": "cust-123",
    "items": [{"sku": "ITEM-001", "quantity": 2}],
    "amount": 99.99
  }'
```

---

## Exercise 2: Money Transfer Saga 🟡

**Goal:** Transfer money between accounts with compensation

### Requirements

1. Create `MoneyTransferSaga` with steps:
   - Debit source account
   - Credit destination account
   - Record transaction

2. Challenges:
   - Ensure atomicity (all or nothing)
   - Handle insufficient funds
   - Idempotent debit/credit operations

### Starter Code

```go
type MoneyTransferSaga struct{}

type TransferRequest struct {
    TransferID  string  `json:"transferId"`
    FromAccount string  `json:"fromAccount"`
    ToAccount   string  `json:"toAccount"`
    Amount      float64 `json:"amount"`
}

type TransferResult struct {
    Status      string `json:"status"`
    DebitTxID   string `json:"debitTxId,omitempty"`
    CreditTxID  string `json:"creditTxId,omitempty"`
}

func (MoneyTransferSaga) Run(
    ctx restate.WorkflowContext,
    req TransferRequest,
) (TransferResult, error) {
    // TODO: Implement saga
    // Step 1: Debit from account
    debitTxID, err := debitAccount(ctx, req.FromAccount, req.Amount)
    if err != nil {
        // Handle insufficient funds
        return TransferResult{Status: "failed"}, nil
    }
    
    // Step 2: Credit to account
    creditTxID, err := creditAccount(ctx, req.ToAccount, req.Amount)
    if err != nil {
        // COMPENSATE: Reverse debit
        reverseDebit(ctx, debitTxID)
        return TransferResult{Status: "failed"}, nil
    }
    
    // Step 3: Record transaction
    recordTransaction(ctx, req)
    
    return TransferResult{
        Status:     "completed",
        DebitTxID:  debitTxID,
        CreditTxID: creditTxID,
    }, nil
}
```

---

## Exercise 3: Event Ticketing Saga 🟡

**Goal:** Reserve tickets, process payment, and issue tickets

### Requirements

1. Create `TicketingSaga` with steps:
   - Reserve seats (with timeout)
   - Process payment
   - Issue tickets
   - Send confirmation

2. Features:
   - Release seats if payment fails
   - Handle sold-out scenarios
   - Send tickets via email

### Starter Code

```go
type TicketingSaga struct{}

type TicketRequest struct {
    BookingID   string   `json:"bookingId"`
    EventID     string   `json:"eventId"`
    SeatIDs     []string `json:"seatIds"`
    CustomerID  string   `json:"customerId"`
    TotalAmount float64  `json:"totalAmount"`
}

func (TicketingSaga) Run(
    ctx restate.WorkflowContext,
    req TicketRequest,
) (TicketResult, error) {
    // TODO: Reserve seats
    reservationID, err := reserveSeats(ctx, req.EventID, req.SeatIDs)
    if err != nil {
        return TicketResult{Status: "sold_out"}, nil
    }
    
    // TODO: Process payment with timeout
    // Hold reservation for 15 minutes
    
    // TODO: Issue tickets
    
    // TODO: Send confirmation
    
    return TicketResult{Status: "confirmed"}, nil
}
```

---

## Exercise 4: Multi-Stage Deployment Saga 🔴

**Goal:** Deploy application across environments with rollback

### Requirements

1. Create `DeploymentSaga` with stages:
   - Deploy to staging
   - Run integration tests
   - Deploy to production
   - Update DNS

2. Compensation:
   - Rollback production if DNS update fails
   - Rollback staging if tests fail
   - Preserve logs for debugging

### Starter Code

```go
type DeploymentSaga struct{}

type DeploymentRequest struct {
    DeploymentID string `json:"deploymentId"`
    Version      string `json:"version"`
    ImageTag     string `json:"imageTag"`
}

func (DeploymentSaga) Run(
    ctx restate.WorkflowContext,
    req DeploymentRequest,
) (DeploymentResult, error) {
    // Stage 1: Deploy to staging
    stagingID, err := deployToEnvironment(ctx, "staging", req)
    if err != nil {
        return DeploymentResult{Status: "failed"}, nil
    }
    
    // Stage 2: Run tests
    testsPassed, err := runIntegrationTests(ctx, "staging")
    if err != nil || !testsPassed {
        // COMPENSATE: Rollback staging
        rollbackDeployment(ctx, "staging", stagingID)
        return DeploymentResult{Status: "tests_failed"}, nil
    }
    
    // Stage 3: Deploy to production
    prodID, err := deployToEnvironment(ctx, "production", req)
    if err != nil {
        // COMPENSATE: Rollback staging
        rollbackDeployment(ctx, "staging", stagingID)
        return DeploymentResult{Status: "failed"}, nil
    }
    
    // Stage 4: Update DNS
    err = updateDNS(ctx, req.Version)
    if err != nil {
        // COMPENSATE: Rollback production and staging
        rollbackDeployment(ctx, "production", prodID)
        rollbackDeployment(ctx, "staging", stagingID)
        return DeploymentResult{Status: "failed"}, nil
    }
    
    return DeploymentResult{Status: "deployed"}, nil
}
```

---

## Exercise 5: Insurance Claim Processing 🔴

**Goal:** Multi-step claim with approvals and payments

### Requirements

1. Create `ClaimProcessingSaga`:
   - Validate claim
   - Request medical records
   - Get approval (human-in-the-loop)
   - Process payment
   - Close claim

2. Features:
   - Wait for external approval (promise)
   - Timeout after 30 days
   - Handle rejection at any stage

### Starter Code

```go
type ClaimSaga struct{}

func (ClaimSaga) Run(
    ctx restate.WorkflowContext,
    claim Claim,
) (ClaimResult, error) {
    // Step 1: Validate claim
    valid, err := validateClaim(ctx, claim)
    if err != nil || !valid {
        return ClaimResult{Status: "rejected_invalid"}, nil
    }
    
    // Step 2: Request medical records
    recordsID, err := requestMedicalRecords(ctx, claim)
    if err != nil {
        return ClaimResult{Status: "failed"}, nil
    }
    
    // Step 3: Wait for approval (with 30-day timeout)
    approvalPromise := restate.Promise[Approval](ctx, "approval")
    timeout := restate.After(ctx, 30*24*time.Hour)
    
    winner, _ := restate.WaitFirst(ctx, approvalPromise, timeout)
    
    if winner == timeout {
        // Timeout - close claim
        return ClaimResult{Status: "timeout"}, nil
    }
    
    approval, _ := approvalPromise.Result()
    if !approval.Approved {
        return ClaimResult{Status: "rejected"}, nil
    }
    
    // Step 4: Process payment
    paymentID, err := processPayment(ctx, claim.Amount)
    if err != nil {
        // Can't compensate approval, log for manual review
        return ClaimResult{Status: "payment_failed"}, nil
    }
    
    // Step 5: Close claim
    closeClaim(ctx, claim.ClaimID, paymentID)
    
    return ClaimResult{
        Status:    "paid",
        PaymentID: paymentID,
    }, nil
}

func (ClaimSaga) Approve(
    ctx restate.WorkflowSharedContext,
    approval Approval,
) error {
    return restate.Promise[Approval](ctx, "approval").Resolve(approval)
}
```

---

## Exercise 6: Microservices Migration Saga 🔴

**Goal:** Migrate data between systems with zero downtime

### Requirements

1. Create `MigrationSaga`:
   - Enable dual-write mode
   - Migrate historical data
   - Verify data consistency
   - Switch traffic
   - Disable old system

2. Challenges:
   - Long-running (hours/days)
   - Cannot fully rollback after traffic switch
   - Must handle partial migrations

### Starter Code

```go
type MigrationSaga struct{}

func (MigrationSaga) Run(
    ctx restate.WorkflowContext,
    migration Migration,
) (MigrationResult, error) {
    // Phase 1: Enable dual-write
    err := enableDualWrite(ctx, migration.ServiceID)
    if err != nil {
        return MigrationResult{Status: "failed"}, nil
    }
    
    // Phase 2: Migrate historical data (long-running)
    batchSize := migration.TotalRecords / 100
    for batch := 0; batch < 100; batch++ {
        err := migrateDataBatch(ctx, batch, batchSize)
        if err != nil {
            // COMPENSATE: Disable dual-write
            disableDualWrite(ctx, migration.ServiceID)
            return MigrationResult{Status: "migration_failed"}, nil
        }
        
        // Sleep between batches
        restate.Sleep(ctx, 1*time.Minute)
    }
    
    // Phase 3: Verify consistency
    consistent, err := verifyDataConsistency(ctx, migration.ServiceID)
    if err != nil || !consistent {
        disableDualWrite(ctx, migration.ServiceID)
        return MigrationResult{Status: "verification_failed"}, nil
    }
    
    // Phase 4: Switch traffic (POINT OF NO RETURN)
    err = switchTraffic(ctx, migration.ServiceID)
    if err != nil {
        // Can't easily rollback - use forward recovery
        ctx.Log().Error("Traffic switch failed - manual intervention needed")
        return MigrationResult{Status: "manual_intervention_required"}, nil
    }
    
    // Phase 5: Disable old system
    disableOldSystem(ctx, migration.ServiceID)
    
    return MigrationResult{Status: "completed"}, nil
}
```

---

## ✅ Exercise Checklist

- [ ] Exercise 1: E-Commerce Order (Beginner)
- [ ] Exercise 2: Money Transfer (Intermediate)
- [ ] Exercise 3: Event Ticketing (Intermediate)
- [ ] Exercise 4: Multi-Stage Deployment (Advanced)
- [ ] Exercise 5: Insurance Claim Processing (Advanced)
- [ ] Exercise 6: Microservices Migration (Advanced)

## 📁 Solutions

Complete solutions available in [solutions/](./solutions/):

- `exercise1_ecommerce_order.go`
- `exercise2_money_transfer.go`
- `exercise3_event_ticketing.go`
- `exercise4_deployment.go`
- `exercise5_insurance_claim.go`
- `exercise6_data_migration.go`

## 🎯 Next Module

Congratulations! You've mastered Sagas!

You now understand:
- ✅ Distributed transactions
- ✅ Compensation patterns
- ✅ Forward vs backward recovery
- ✅ Idempotent operations
- ✅ Complex saga orchestrations

Ready to learn about advanced patterns?

👉 **Continue to [Module 7: Testing](../07-testing/README.md)**

Learn to test your Restate applications thoroughly!

---

**Questions?** Review [concepts](./01-concepts.md) or check the [complete code](./code/)!
