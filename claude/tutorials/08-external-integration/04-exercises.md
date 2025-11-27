# Exercises: External Integration Practice

> **Build your own external integrations**

## 🎯 Learning Objectives

Practice building resilient external integrations using Restate patterns you've learned.

---

## Exercise 1: Twilio SMS Notifications ⭐

**Goal:** Add SMS notification integration to the order service

### Requirements

1. Create `TwilioClient` with `SendSMS` method
2. Send SMS when order is confirmed
3. Use mock mode for development
4. Journal the SMS call
5. Handle failures gracefully (order succeeds even if SMS fails)

### Data Structures

```go
type SMSRequest struct {
    To      string
    Message string
}

type SMSResponse struct {
    MessageSID string
    Status     string
}
```

### Hints

- Wrap Twilio call in `restate.Run()`
- Use environment variable `TWILIO_API_KEY`
- Mock mode returns `fmt.Sprintf("sms_mock_%d", time.Now().Unix())`
- Call after email is sent
- Log but don't fail if SMS fails

### Success Criteria

- [ ] SMS sent after order confirmation
- [ ] Journaled (doesn't retry on system restart)
- [ ] Mock mode works
- [ ] Order succeeds even if SMS fails

---

## Exercise 2: Slack Notification Integration ⭐⭐

**Goal:** Send Slack notifications for important order events

### Requirements

1. Create `SlackClient` with `PostMessage` method
2. Send notifications for:
   - New order (with total amount)
   - Payment failure
   - Order confirmation
3. Use Slack webhooks API
4. Include order details in message

### Data Structures

```go
type SlackMessage struct {
    Channel string
    Text    string
    Fields  []SlackField
}

type SlackField struct {
    Title string
    Value string
}
```

### Hints

- Use Slack incoming webhooks
- Format messages with order ID, customer, total
- Different channels for different events
- Mock mode logs to console
- Non-blocking (failures don't affect orders)

### Success Criteria

- [ ] Notifications sent for all events
- [ ] Messages include relevant details
- [ ] Journaled externally calls
- [ ] Failures logged but don't block

---

## Exercise 3: Inventory Service Integration ⭐⭐

**Goal:** Check and reserve inventory before processing order

### Requirements

1. Create `InventoryClient` with:
   - `CheckAvailability(productID, quantity)`
   - `ReserveInventory(items[])`
   - `ReleaseReservation(reservationID)`
2. Check inventory before charging customer
3. Reserve inventory after successful payment
4. Release reservation if shipping fails
5. Handle "out of stock" scenarios

### Data Structures

```go
type InventoryCheck struct {
    ProductID string
    Quantity  int
    Available bool
}

type Reservation struct {
    ID    string
    Items []OrderItem
}
```

###Hints

- Check availability first (before payment)
- Reserve after charge succeeds
- Use compensating action (release) on failure
- Return terminal error if out of stock
- Mock inventory with simple map

### Success Criteria

- [ ] Out of stock returns terminal error
- [ ] Inventory reserved after payment
- [ ] Reservation released on failure
- [ ] All calls journaled

---

## Exercise 4: Multi-Webhook Aggregator ⭐⭐⭐

**Goal:** Handle webhooks from multiple services (Stripe, Shippo, SendGrid)

### Requirements

1. Extend `WebhookProcessor` to handle:
   - Stripe webhooks (charge events)
   - Shippo webhooks (tracking updates)
   - SendGrid webhooks (email delivery status)
2. Update order state based on webhooks
3. Verify webhook signatures
4. Handle out-of-order webhooks

### Webhook Types

```go
// Stripe
- charge.succeeded
- charge.failed
- charge.refunded

// Shippo
- tracking.created
- tracking.in_transit
- tracking.delivered

// SendGrid
- delivered
- bounced
- opened
```

### Hints

- Different handler methods for each service
- Use webhook ID as object key
- Verify signatures (mock in development)
- State transitions: delivered → in_transit → delivered
- Call back to `OrderOrchestrator` to update status

### Success Criteria

- [ ] All three services supported
- [ ] Signatures verified
- [ ] Idempotent processing
- [ ] Order state updated correctly
- [ ] Out-of-order handled

---

## Exercise 5: GitHub Deployment Integration ⭐⭐

**Goal:** Trigger deployments via GitHub API

### Requirements

1. Create `GitHubClient` with:
   - `CreateDeployment(repo, ref, environment)`
   - `GetDeploymentStatus(deploymentID)`
2. Create Workflow service that:
   - Triggers deployment
   - Waits for webhook callback
   - Returns final status
3. Use awakeables for async callback

### Data Structures

```go
type DeploymentRequest struct {
    Repository  string
    Ref         string
    Environment string
}

type Deployment struct {
    ID     string
    Status string // pending, in_progress, success, failure
}
```

### Hints

- Create awakeable after triggering deployment
- Store awakeable ID in state
- Webhook resolves awakeable with result
- Use `restate.Awakeable[DeploymentStatus](ctx)`
- Timeout after 10 minutes

### Success Criteria

- [ ] Deployment triggered via API
- [ ] Awaits webhook callback
- [ ] Awakeable resolved with result
- [ ] Timeout handled
- [ ] All journaled

---

## Exercise 6: Distributed Order Fulfillment ⭐⭐⭐

**Goal:** Build complete fulfillment pipeline with multiple services

### Requirements

1. Services involved:
   - `FulfillmentOrchestrator` - Main coordinator
   - `InventoryService` - Stock management
   - `WarehouseService` - Picking/packing
   - `CarrierService` - Shipping
   - `NotificationService` - Customer updates
2. Workflow:
   - Check inventory
   - Create pick ticket
   - Wait for packing webhook
   - Schedule carrier pickup
   - Send tracking notification
3. Handle failures at each step
4. Compensating actions

### Architecture

```
FulfillmentOrchestrator
├→ InventoryService.Reserve()
├→ WarehouseService.CreatePickTicket()
├→ await WarehouseWebhook (packed)
├→ CarrierService.SchedulePickup()
├→ await CarrierWebhook (picked up)
└→ NotificationService.SendTracking()
```

### Hints

- Use object-to-object calls
- Awakeables for warehouse/carrier webhooks
- State machine for order status
- Retries with backoff
- Comprehensive error handling

### Success Criteria

- [ ] Complete workflow implemented
- [ ] All services integrated
- [ ] Webhooks for async steps
- [ ] Compensating actions
- [ ] Failure recovery
- [ ] Status tracking

---

## 💡 General Tips

### External Integration Best Practices

1. **Always journal external calls**
   ```go
   restate.Run(ctx, func(ctx restate.RunContext) (T, error) {
       return externalAPI.Call()
   })
   ```

2. **Classify errors correctly**
   ```go
   if isClientError(err) {
       return restate.TerminalError(err)
   }
   return err // Retryable
   ```

3. **Use mock mode for development**
   ```go
   if os.Getenv("MOCK_MODE") == "true" {
       return mockImplementation()
   }
   ```

4. **Handle failures gracefully**
   - Critical operations (payment) - fail fast
   - Non-critical (email) - log and continue

## 📚 Resources

- [Module Concepts](./01-concepts.md)
- [Hands-On Tutorial](./02-hands-on.md)
- [Validation Guide](./03-validation.md)
- [Code Examples](./code/)

## 🎓 Learning Path

**Recommended Order:**
1. Exercise 1 (Twilio) - Simple single integration
2. Exercise 2 (Slack) - Multiple event types
3. Exercise 3 (Inventory) - Compensating actions
4. Exercise 5 (GitHub) - Async webhooks with awakeables
5. Exercise 4 (Multi-webhook) - Complex webhook handling
6. Exercise 6 (Fulfillment) - Complete distributed system

---

**Good luck!** 🚀 Build resilient integrations!
