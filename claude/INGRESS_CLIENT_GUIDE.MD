# Ingress Client Wrappers - Complete Guide

## Overview

Ingress clients enable external applications (outside Restate handlers) to invoke Restate services. The framework provides type-safe wrappers that match the internal client patterns while adding automatic validation and better error handling.

## When to Use Ingress Clients

✅ **DO use ingress clients when:**
- Calling Restate from web applications
- Invoking from external microservices
- Triggering workflows from HTTP APIs
- Integrating with third-party systems

❌ **DON'T use ingress clients when:**
- Inside Restate handlers (use `ServiceClient`, `ObjectClient`, `WorkflowClient` instead)
- For internal service-to-service calls
- When you have access to `restate.Context`

## Client Types

###1. IngressServiceClient - Stateless Services
### 2. IngressObjectClient - Virtual Objects
### 3. IngressWorkflowClient - Workflows

## Setup

### Create Ingress Client

```go
import (
    "context"
    "github.com/restatedev/sdk-go/server"
)

func main() {
    // Create ingress client for external calls
    ingress := NewIngressClient(
        "http://localhost:8080",  // Restate server URL
        "your-auth-key-optional",  // Auth key (optional)
    )
    
    // Use the client...
}
```

Note: The framework wrappers are examples showing the pattern. For actual usage, you'll need the `restateingress` package:

```go
import restateingress "github.com/restatedev/sdk-go/server"
```

## Usage Examples

### Example 1: Call Stateless Service

```go
package main

import (
    "context"
    "fmt"
    restateingress "github.com/restatedev/sdk-go/server"
)

func callPaymentService() error {
    // Create ingress client
    client := restateingress.NewClient("http://localhost:8080")
    
    // Call service synchronously
    result, err := restateingress.Service[PaymentRequest, PaymentResponse](
        client,
        "PaymentService",
        "ProcessPayment",
    ).Request(context.Background(), PaymentRequest{
        Amount: 100.00,
        Currency: "USD",
    })
    
    if err != nil {
        return fmt.Errorf("payment failed: %w", err)
    }
    
    fmt.Printf("Payment successful: %s\n", result.TransactionID)
    return nil
}
```

### Example 2: Send to Service (Fire-and-Forget)

```go
func sendNotification() error {
    client := restateingress.NewClient("http://localhost:8080")
    
    // Send asynchronously (don't wait for result)
    restateingress.ServiceSend[Notification](
        client,
        "NotificationService",
        "SendEmail",
    ).Send(context.Background(), Notification{
        To:      "user@example.com",
        Subject: "Welcome!",
        Body:    "Thanks for signing up",
    })
    
    return nil
}
```

### Example 3: Call Virtual Object with Key

```go
func updateUserAccount(userID string, update AccountUpdate) error {
    client := restateingress.NewClient("http://localhost:8080")
    
    // Call Virtual Object with specific key
    result, err := restateingress.Object[AccountUpdate, AccountBalance](
        client,
        "Account",        // Service name
        userID,          // Object key - identifies which account
        "UpdateBalance", // Handler name
    ).Request(context.Background(), update)
    
    if err != nil {
        return err
    }
    
    fmt.Printf("New balance: $%.2f\n", result.Balance)
    return nil
}
```

### Example 4: Start Workflow

```go
func startOrderWorkflow(orderID string, order OrderInput) (string, error) {
    client := restateingress.NewClient("http://localhost:8080")
    
    // Submit new workflow instance
    workflowID := fmt.Sprintf("order-%s", orderID)
    
    invocation := restateingress.WorkflowSend[OrderInput](
        client,
        "OrderWorkflow",  // Workflow service name
        workflowID,       // Workflow instance ID
        "run",            // Handler name (usually "run")
    ).Send(context.Background(), order)
    
    fmt.Printf("Workflow started: %s\n", invocation.ID)
    return invocation.ID, nil
}
```

### Example 5: Attach to Running Workflow

```go
func getWorkflowResult(workflowID string) (OrderResult, error) {
    client := restateingress.NewClient("http://localhost:8080")
    
    // Attach to existing workflow and wait for result
    result, err := restateingress.Workflow[OrderInput, OrderResult](
        client,
        "OrderWorkflow",
        workflowID,
        "run",
    ).Request(context.Background(), OrderInput{})  // Empty input for attach
    
    if err != nil {
        return OrderResult{}, fmt.Errorf("workflow failed: %w", err)
    }
    
    return result, nil
}
```

### Example 6: Query Workflow Status

```go
func checkWorkflowProgress(workflowID string) (WorkflowStatus, error) {
    client := restateingress.NewClient("http://localhost:8080")
    
    // Query status via shared handler
    status, err := restateingress.Workflow[struct{}, WorkflowStatus](
        client,
        "OrderWorkflow",
        workflowID,
        "GetStatus",  // Shared handler name
    ).Request(context.Background(), struct{}{})
    
    if err != nil {
        return WorkflowStatus{}, err
    }
    
    fmt.Printf("Workflow progress: %.0f%%\n", status.Progress*100)
    return status, nil
}
```

### Example 7: With Idempotency Key

```go
func processPaymentIdempotent(paymentID string, req PaymentRequest) error {
    client := restateingress.NewClient("http://localhost:8080")
    
    // Generate idempotency key (deterministic based on payment ID)
    idempotencyKey := fmt.Sprintf("payment-%s", paymentID)
    
    // Call with idempotency key
    _, err := restateingress.Service[PaymentRequest, PaymentResponse](
        client,
        "PaymentService",
        "Charge",
    ).Request(
        context.Background(),
        req,
        restateingress.WithIdempotencyKey(idempotencyKey),
    )
    
    return err
}
```

### Example 8: With Delay

```go
func scheduleReminder(reminderID string, delay time.Duration) error {
    client := restateingress.NewClient("http://localhost:8080")
    
    // Send with delay (scheduled execution)
    restateingress.ServiceSend[Reminder](
        client,
        "ReminderService",
        "SendReminder",
    ).Send(
        context.Background(),
        Reminder{ID: reminderID},
        restateingress.WithDelay(delay),  // Execute after delay
    )
    
    return nil
}
```

### Example 9: HTTP API Integration

```go
package main

import (
    "encoding/json"
    "net/http"
    restateingress "github.com/restatedev/sdk-go/server"
)

// HTTP handler that invokes Restate workflow
func handleCreateOrder(w http.ResponseWriter, r *http.Request) {
    var req CreateOrderRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Create Restate client
    restateClient := restateingress.NewClient("http://localhost:8080")
    
    // Start workflow
    workflowID := fmt.Sprintf("order-%s", req.OrderID)
    invocation := restateingress.WorkflowSend[OrderInput](
        restateClient,
        "OrderWorkflow",
        workflowID,
        "run",
    ).Send(r.Context(), OrderInput{
        OrderID: req.OrderID,
        Items:   req.Items,
    })
    
    // Return workflow ID to client
    json.NewEncoder(w).Encode(map[string]string{
        "workflow_id": workflowID,
        "invocation_id": invocation.ID,
    })
}

func main() {
    http.HandleFunc("/api/orders", handleCreateOrder)
    http.ListenAndServe(":3000", nil)
}
```

### Example 10: Webhook Integration

```go
// Webhook handler that resolves Restate awakeable
func handlePaymentWebhook(w http.ResponseWriter, r *http.Request) {
    var webhook PaymentWebhook
    if err := json.NewDecoder(r.Body).Decode(&webhook); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Extract awakeable ID from webhook
    awakeableID := webhook.AwakeableID
    
    // Resolve awakeable via Restate API
    restateClient := restateingress.NewClient("http://localhost:8080")
    
    if webhook.Status == "success" {
        // Resolve with success
        err := restateingress.ResolveAwakeable[PaymentResult](
            restateClient,
            awakeableID,
            PaymentResult{
                Success: true,
                TransactionID: webhook.TransactionID,
            },
        )
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
    } else {
        // Reject with error
        err := restateingress.RejectAwakeable(
            restateClient,
            awakeableID,
            fmt.Errorf("payment failed: %s", webhook.ErrorMessage),
        )
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
    }
    
    w.WriteHeader(http.StatusOK)
}
```

## Framework Wrapper Usage

The framework provides type-safe wrappers that mirror the SDK patterns:

```go
// Create framework ingress wrapper
ingress := NewIngressClient("http://localhost:8080", "auth-key")

// Service client
serviceClient := ingress.Service("PaymentService", "Charge")

// Object client
objectClient := ingress.Object("Account", "Debit")

// Workflow client
workflowClient := ingress.Workflow("OrderWorkflow", "run")
```

**Note:** These are wrapper patterns showing the API design. For production use, import and use `restateingress` package directly as shown in  examples above.

## Best Practices

### ✅ DO: Use Idempotency Keys

```go
// ✓ Use deterministic idempotency keys for important operations
idempKey := fmt.Sprintf("payment-%s-%s", userID, orderID)

restateingress.Service[Req, Resp](client, "Service", "Handler").
    Request(ctx, req, restateingress.WithIdempotencyKey(idempKey))
```

### ✅ DO: Handle Errors Appropriately

```go
// ✓ Differentiate between transient and permanent errors
result, err := restateingress.Service[Req, Resp](
    client, "Service", "Handler",
).Request(ctx, req)

if err != nil {
    if isTerminalError(err) {
        // Don't retry terminal errors
        return err
    }
    // Retry transient errors
    return retryWithBackoff(operation)
}
```

### ✅ DO: Use Context for Timeouts

```go
// ✓ Set timeout for external calls
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

result, err := restateingress.Service[Req, Resp](
    client, "Service", "Handler",
).Request(ctx, req)
```

### ✅ DO: Validate Before Sending

```go
// ✓ Validate idempotency keys before use
idempKey := generateKey()
if err := ValidateIdempotencyKey(idempKey); err != nil {
    return fmt.Errorf("invalid key: %w", err)
}

restateingress.Service[Req, Resp](client, "Svc", "Handler").
    Request(ctx, req, restateingress.WithIdempotencyKey(idempKey))
```

### ❌ DON'T: Use Ingress Clients Inside Handlers

```go
// ✗ DON'T: Use ingress client from inside Restate handler
func (MyService) Handler(ctx restate.Context, req Request) error {
    // BAD: Using ingress client
    client := restateingress.NewClient("http://localhost:8080")
    result, _ := restateingress.Service[Req, Resp](
        client, "OtherService", "Handler",
    ).Request(context.Background(), req)
    
    return nil
}

// ✓ DO: Use internal SDK clients
func (MyService) Handler(ctx restate.Context, req Request) error {
    // GOOD: Using internal client
    client := ServiceClient[Req, Resp]{
        ServiceName: "OtherService",
        HandlerName: "Handler",
    }
    result, _ := client.Call(ctx, req)
    
    return nil
}
```

### ❌ DON'T: Forget Workflow IDs

```go
// ✗ DON'T: Random workflow IDs (can't attach later)
workflowID := uuid.New().String()

// ✓ DO: Deterministic workflow IDs
workflowID := fmt.Sprintf("order-%s", orderID)
```

## Security Considerations

### Authentication

```go
// Create client with auth key
client := restateingress.NewClient(
    "http://localhost:8080",
    restateingress.WithAuthKey("your-secret-key"),
)
```

### HTTPS in Production

```go
// ✓ Always use HTTPS in production
client := restateingress.NewClient("https://restate.production.com")

// ✗ Don't use HTTP in production
// client := restateingress.NewClient("http://restate.production.com")
```

## Testing

```go
func TestIngressClient_CallService(t *testing.T) {
    // Use test Restate server
    client := restateingress.NewClient("http://localhost:8080")
    
    result, err := restateingress.Service[TestReq, TestResp](
        client,
        "TestService",
        "TestHandler",
    ).Request(context.Background(), TestReq{Data: "test"})
    
    assert.NoError(t, err)
    assert.Equal(t, "expected", result.Value)
}
```

## Comparison: Internal vs Ingress Clients

| Feature | Internal Clients | Ingress Clients |
|---------|-----------------|-----------------|
| **Context** | `restate.Context` | `context.Context` |
| **Usage** | Inside handlers | Outside Restate |
| **Durability** | Automatic | Manual (via idempotency) |
| **Package** | `github.com/restatedev/sdk-go` | `github.com/restatedev/sdk-go/server` |
| **Example** | `ServiceClient[I,O]` | `restateingress.Service[I,O]` |

## Troubleshooting

### Issue: "connection refused"

**Cause:** Restate server not running

**Solution:**
```bash
# Start Restate server
docker run -p 8080:8080 -p 9090:9090 docker.io/restatedev/restate:latest
```

### Issue: "handler not found"

**Cause:** Service not registered

**Solution:**
```bash
# Register service with Restate
curl -X POST http://localhost:9090/endpoints \
  -H 'Content-Type: application/json' \
  -d '{"uri": "http://localhost:9080"}'
```

### Issue: "idempotency key already used"

**Cause:** Reusing idempotency key for different request

**Solution:** Use unique keys per operation:
```go
// Include operation-specific data in key
key := fmt.Sprintf("payment-%s-%s-%d", userID, orderID, timestamp)
```

## Conclusion

Ingress clients enable:
- ✅ **External invocation** from web apps and services
- ✅ **Type safety** with generics
- ✅ **Idempotency support** for reliability
- ✅ **Workflow lifecycle** management (submit/attach)
- ✅ **Status queries** via shared handlers
- ✅ **Awakeable resolution** for webhook integration

**Use** `restateingress` package for production. The framework wrappers demonstrate the clean API pattern for building on top of the SDK.
