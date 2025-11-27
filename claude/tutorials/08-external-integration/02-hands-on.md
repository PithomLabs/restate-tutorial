# Hands-On: E-Commerce Integration Service

> **Build a real-world service integrating multiple external APIs**

## 🎯 What We're Building

An **Order Orchestrator** that integrates with:
- 💳 **Stripe** - Payment processing
- 📧 **SendGrid** - Email notifications  
- 📦 **Shippo** - Shipping label creation

**Features:**
- Process orders with payment
- Send confirmation emails
- Create shipping labels
- Handle webhooks from all services
- Coordinate multi-service workflows

## 📋 Prerequisites

- ✅ Go 1.23+ installed
- ✅ Completed Module 01 (Foundation)
- ✅ Completed Idempotency module
- ✅ Docker for Restate server
- ⚠️ Optional: API keys for real integrations (or use mock mode)

## 🏗️ Project Setup

### Step 1: Create Project Directory

```bash
mkdir -p ~/restate-tutorials/external-integration/code
cd ~/restate-tutorials/external-integration/code
```

### Step 2: Initialize Go Module

```bash
go mod init external-integration
go get github.com/restatedev/sdk-go@latest
```

### Step 3: Set Environment Variables (Optional)

For real API calls:
```bash
export STRIPE_API_KEY="sk_test_..."
export SENDGRID_API_KEY="SG...."
export SHIPPO_API_KEY="shippo_test_..."
```

For mock mode (recommended for learning):
```bash
export MOCK_MODE=true
```

## 📝 Implementation

### Step 1: Define Types (`types.go`)

```go
package main

import "time"

// Order request from client
type OrderRequest struct {
	Items    []OrderItem      `json:"items"`
	Customer CustomerInfo     `json:"customer"`
	Shipping ShippingAddress  `json:"shipping"`
}

type OrderItem struct {
	ProductID string `json:"productId"`
	Quantity  int    `json:"quantity"`
	Price     int    `json:"price"` // cents
}

type CustomerInfo struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type ShippingAddress struct {
	Street  string `json:"street"`
	City    string `json:"city"`
	State   string `json:"state"`
	Zip     string `json:"zip"`
	Country string `json:"country"`
}

// Order stored in state
type Order struct {
	OrderID       string          `json:"orderId"`
	Items         []OrderItem     `json:"items"`
	Customer      CustomerInfo    `json:"customer"`
	Shipping      ShippingAddress `json:"shipping"`
	Status        string          `json:"status"`
	ChargeID      string          `json:"chargeId,omitempty"`
	LabelID       string          `json:"labelId,omitempty"`
	TrackingNumber string         `json:"trackingNumber,omitempty"`
	CreatedAt     time.Time       `json:"createdAt"`
	UpdatedAt     time.Time       `json:"updatedAt"`
}

// Order result returned to client
type OrderResult struct {
	OrderID        string `json:"orderId"`
	Status         string `json:"status"`
	ChargeID       string `json:"chargeId,omitempty"`
	TrackingNumber string `json:"trackingNumber,omitempty"`
	Message        string `json:"message"`
}

// Stripe integration types
type StripeChargeRequest struct {
	Amount   int    `json:"amount"`
	Currency string `json:"currency"`
	Email    string `json:"email"`
}

type StripeChargeResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// SendGrid integration types
type EmailRequest struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

type EmailResponse struct {
	MessageID string `json:"messageId"`
	Status    string `json:"status"`
}

// Shippo integration types
type ShippingLabelRequest struct {
	Address ShippingAddress `json:"address"`
	Weight  int             `json:"weight"` // grams
}

type ShippingLabelResponse struct {
	LabelID        string `json:"labelId"`
	TrackingNumber string `json:"trackingNumber"`
	LabelURL       string `json:"labelUrl"`
}

// Webhook types
type StripeWebhook struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Timestamp int64                  `json:"created"`
}

type WebhookResult struct {
	WebhookID string `json:"webhookId"`
	Type      string `json:"type"`
	Status    string `json:"status"`
	Message   string `json:"message"`
}
```

### Step 2: Stripe Client (`stripe_client.go`)

```go
package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"
	
	restate "github.com/restatedev/sdk-go"
)

type StripeClient struct {
	apiKey   string
	mockMode bool
}

func NewStripeClient() *StripeClient {
	return &StripeClient{
		apiKey:   os.Getenv("STRIPE_API_KEY"),
		mockMode: os.Getenv("MOCK_MODE") == "true",
	}
}

// CreateCharge processes a payment (called within restate.Run)
func (c *StripeClient) CreateCharge(
	ctx restate.RunContext,
	req StripeChargeRequest,
) (StripeChargeResponse, error) {
	ctx.Log().Info("Creating Stripe charge",
		"amount", req.Amount,
		"email", req.Email)
	
	if c.mockMode {
		return c.mockCharge(req)
	}
	
	return c.realCharge(req)
}

func (c *StripeClient) mockCharge(req StripeChargeRequest) (StripeChargeResponse, error) {
	// Simulate network delay
	time.Sleep(100 * time.Millisecond)
	
	// Simulate 5% failure rate
	if rand.Float64() < 0.05 {
		return StripeChargeResponse{}, fmt.Errorf("card declined")
	}
	
	return StripeChargeResponse{
		ID:     fmt.Sprintf("ch_mock_%d", time.Now().Unix()),
		Status: "succeeded",
	}, nil
}

func (c *StripeClient) realCharge(req StripeChargeRequest) (StripeChargeResponse, error) {
	// In production, use actual Stripe SDK:
	// stripe.Key = c.apiKey
	// charge, err := charge.New(&stripe.ChargeParams{
	//     Amount:   stripe.Int64(int64(req.Amount)),
	//     Currency: stripe.String(req.Currency),
	//     Source:   &stripe.SourceParams{Token: stripe.String("tok_...")},
	// })
	
	return StripeChargeResponse{}, fmt.Errorf("real Stripe integration not implemented")
}

// RefundCharge refunds a payment
func (c *StripeClient) RefundCharge(
	ctx restate.RunContext,
	chargeID string,
) error {
	ctx.Log().Info("Refunding Stripe charge", "chargeId", chargeID)
	
	if c.mockMode {
		time.Sleep(100 * time.Millisecond)
		return nil
	}
	
	return fmt.Errorf("real Stripe refund not implemented")
}
```

### Step 3: SendGrid Client (`sendgrid_client.go`)

```go
package main

import (
	"fmt"
	"os"
	"time"
	
	restate "github.com/restatedev/sdk-go"
)

type SendGridClient struct {
	apiKey   string
	mockMode bool
}

func NewSendGridClient() *SendGridClient {
	return &SendGridClient{
		apiKey:   os.Getenv("SENDGRID_API_KEY"),
		mockMode: os.Getenv("MOCK_MODE") == "true",
	}
}

// SendEmail sends an email (called within restate.Run)
func (c *SendGridClient) SendEmail(
	ctx restate.RunContext,
	req EmailRequest,
) (EmailResponse, error) {
	ctx.Log().Info("Sending email",
		"to", req.To,
		"subject", req.Subject)
	
	if c.mockMode {
		return c.mockSend(req)
	}
	
	return c.realSend(req)
}

func (c *SendGridClient) mockSend(req EmailRequest) (EmailResponse, error) {
	// Simulate network delay
	time.Sleep(50 * time.Millisecond)
	
	return EmailResponse{
		MessageID: fmt.Sprintf("msg_mock_%d", time.Now().Unix()),
		Status:    "sent",
	}, nil
}

func (c *SendGridClient) realSend(req EmailRequest) (EmailResponse, error) {
	// In production, use actual SendGrid SDK:
	// from := mail.NewEmail("Shop", "orders@shop.com")
	// to := mail.NewEmail(req.To, req.To)
	// message := mail.NewSingleEmail(from, req.Subject, to, req.Body, req.Body)
	// response, err := sendgrid.NewSendClient(c.apiKey).Send(message)
	
	return EmailResponse{}, fmt.Errorf("real SendGrid integration not implemented")
}
```

### Step 4: Shippo Client (`shippo_client.go`)

```go
package main

import (
	"fmt"
	"os"
	"time"
	
	restate "github.com/restatedev/sdk-go"
)

type ShippoClient struct {
	apiKey   string
	mockMode bool
}

func NewShippoClient() *ShippoClient {
	return &ShippoClient{
		apiKey:   os.Getenv("SHIPPO_API_KEY"),
		mockMode: os.Getenv("MOCK_MODE") == "true",
	}
}

// CreateLabel creates a shipping label (called within restate.Run)
func (c *ShippoClient) CreateLabel(
	ctx restate.RunContext,
	req ShippingLabelRequest,
) (ShippingLabelResponse, error) {
	ctx.Log().Info("Creating shipping label",
		"city", req.Address.City,
		"state", req.Address.State)
	
	if c.mockMode {
		return c.mockLabel(req)
	}
	
	return c.realLabel(req)
}

func (c *ShippoClient) mockLabel(req ShippingLabelRequest) (ShippingLabelResponse, error) {
	// Simulate network delay
	time.Sleep(150 * time.Millisecond)
	
	return ShippingLabelResponse{
		LabelID:        fmt.Sprintf("label_mock_%d", time.Now().Unix()),
		TrackingNumber: fmt.Sprintf("1Z999AA1%d", time.Now().Unix()%10000000000),
		LabelURL:       "https://example.com/label.pdf",
	}, nil
}

func (c *ShippoClient) realLabel(req ShippingLabelRequest) (ShippingLabelResponse, error) {
	// In production, use actual Shippo SDK
	return ShippingLabelResponse{}, fmt.Errorf("real Shippo integration not implemented")
}
```

### Step 5: Order Orchestrator (`order_orchestrator.go`)

The main service coordinating all integrations:

```go
package main

import (
	"fmt"
	"time"
	
	restate "github.com/restatedev/sdk-go"
)

type OrderOrchestrator struct{}

var (
	stripeClient   = NewStripeClient()
	sendgridClient = NewSendGridClient()
	shippoClient   = NewShippoClient()
)

// ProcessOrder orchestrates the entire order workflow
func (OrderOrchestrator) ProcessOrder(
	ctx restate.ObjectContext,
	req OrderRequest,
) (OrderResult, error) {
	orderID := restate.Key(ctx)
	
	ctx.Log().Info("Processing order",
		"orderId", orderID,
		"customer", req.Customer.Email)
	
	// Check if order already exists (idempotent)
	existingOrder, err := restate.Get[*Order](ctx, "order")
	if err != nil {
		return OrderResult{}, err
	}
	
	if existingOrder != nil {
		ctx.Log().Info("Order already exists", "status", existingOrder.Status)
		return OrderResult{
			OrderID:        existingOrder.OrderID,
			Status:         existingOrder.Status,
			ChargeID:       existingOrder.ChargeID,
			TrackingNumber: existingOrder.TrackingNumber,
			Message:        "Order already processed",
		}, nil
	}
	
	// Calculate total
	total := 0
	for _, item := range req.Items {
		total += item.Price * item.Quantity
	}
	
	// Create order in pending state
	order := Order{
		OrderID:   orderID,
		Items:     req.Items,
		Customer:  req.Customer,
		Shipping:  req.Shipping,
		Status:    "pending",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	restate.Set(ctx, "order", order)
	
	// Step 1: Charge customer (JOURNALED)
	ctx.Log().Info("Charging customer", "amount", total)
	
	chargeResp, err := restate.Run(ctx, func(ctx restate.RunContext) (StripeChargeResponse, error) {
		return stripeClient.CreateCharge(ctx, StripeChargeRequest{
			Amount:   total,
			Currency: "usd",
			Email:    req.Customer.Email,
		})
	})
	
	if err != nil {
		ctx.Log().Error("Payment failed", "error", err)
		order.Status = "payment_failed"
		restate.Set(ctx, "order", order)
		
		return OrderResult{
			OrderID: orderID,
			Status:  "payment_failed",
			Message: fmt.Sprintf("Payment failed: %s", err.Error()),
		}, nil
	}
	
	order.ChargeID = chargeResp.ID
	order.Status = "paid"
	order.UpdatedAt = time.Now()
	restate.Set(ctx, "order", order)
	
	ctx.Log().Info("Payment successful", "chargeId", chargeResp.ID)
	
	// Step 2: Create shipping label (JOURNALED)
	ctx.Log().Info("Creating shipping label")
	
	labelResp, err := restate.Run(ctx, func(ctx restate.RunContext) (ShippingLabelResponse, error) {
		return shippoClient.CreateLabel(ctx, ShippingLabelRequest{
			Address: req.Shipping,
			Weight:  1000, // 1kg default
		})
	})
	
	if err != nil {
		// Label creation failed, but order still valid
		ctx.Log().Warn("Failed to create label", "error", err)
	} else {
		order.LabelID = labelResp.LabelID
		order.TrackingNumber = labelResp.TrackingNumber
		order.UpdatedAt = time.Now()
		restate.Set(ctx, "order", order)
		
		ctx.Log().Info("Shipping label created",
			"labelId", labelResp.LabelID,
			"tracking", labelResp.TrackingNumber)
	}
	
	// Step 3: Send confirmation email (JOURNALED)
	ctx.Log().Info("Sending confirmation email")
	
	emailBody := fmt.Sprintf(`
		Hi %s,
		
		Thank you for your order!
		
		Order ID: %s
		Total: $%.2f
		Tracking: %s
		
		Your order will ship soon!
	`, req.Customer.Name, orderID, float64(total)/100, order.TrackingNumber)
	
	_, err = restate.Run(ctx, func(ctx restate.RunContext) (EmailResponse, error) {
		return sendgridClient.SendEmail(ctx, EmailRequest{
			To:      req.Customer.Email,
			Subject: fmt.Sprintf("Order Confirmation - %s", orderID),
			Body:    emailBody,
		})
	})
	
	if err != nil {
		ctx.Log().Warn("Failed to send email", "error", err)
	} else {
		ctx.Log().Info("Confirmation email sent")
	}
	
	// Mark order as confirmed
	order.Status = "confirmed"
	order.UpdatedAt = time.Now()
	restate.Set(ctx, "order", order)
	
	return OrderResult{
		OrderID:        orderID,
		Status:         "confirmed",
		ChargeID:       order.ChargeID,
		TrackingNumber: order.TrackingNumber,
		Message:        "Order processed successfully",
	}, nil
}

// GetOrder retrieves order status
func (OrderOrchestrator) GetOrder(
	ctx restate.ObjectSharedContext,
	_ restate.Void,
) (Order, error) {
	orderID := restate.Key(ctx)
	
	order, err := restate.Get[Order](ctx, "order")
	if err != nil {
		return Order{}, err
	}
	
	ctx.Log().Info("Retrieved order", "orderId", orderID, "status", order.Status)
	
	return order, nil
}
```

### Step 6: Webhook Processor (`webhook_processor.go`)

```go
package main

import (
	restate "github.com/restatedev/sdk-go"
)

type WebhookProcessor struct{}

// ProcessStripeWebhook handles Stripe webhooks (IDEMPOTENT)
func (WebhookProcessor) ProcessStripeWebhook(
	ctx restate.ObjectContext,
	webhook StripeWebhook,
) (WebhookResult, error) {
	webhookID := restate.Key(ctx)
	
	ctx.Log().Info("Processing Stripe webhook",
		"webhookId", webhookID,
		"type", webhook.Type)
	
	// Check if already processed
	existing, err := restate.Get[*WebhookResult](ctx, "result")
	if err != nil {
		return WebhookResult{}, err
	}
	
	if existing != nil {
		ctx.Log().Info("Webhook already processed")
		return *existing, nil
	}
	
	// Process based on type
	var result WebhookResult
	
	switch webhook.Type {
	case "charge.succeeded":
		result = WebhookResult{
			WebhookID: webhookID,
			Type:      webhook.Type,
			Status:    "processed",
			Message:   "Charge succeeded",
		}
		
	case "charge.failed":
		result = WebhookResult{
			WebhookID: webhookID,
			Type:      webhook.Type,
			Status:    "processed",
			Message:   "Charge failed",
		}
		
	default:
		ctx.Log().Warn("Unknown webhook type", "type", webhook.Type)
		result = WebhookResult{
			WebhookID: webhookID,
			Type:      webhook.Type,
			Status:    "skipped",
			Message:   "Unknown webhook type",
		}
	}
	
	// Store result
	restate.Set(ctx, "result", result)
	
	return result, nil
}
```

### Step 7: Main Server (`main.go`)

```go
package main

import (
	"context"
	"fmt"
	"log"
	
	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/server"
)

func main() {
	restateServer := server.NewRestate()
	
	// Register Order Orchestrator
	if err := restateServer.Bind(restate.Reflect(OrderOrchestrator{})); err != nil {
		log.Fatal("Failed to bind OrderOrchestrator:", err)
	}
	
	// Register Webhook Processor
	if err := restateServer.Bind(restate.Reflect(WebhookProcessor{})); err != nil {
		log.Fatal("Failed to bind WebhookProcessor:", err)
	}
	
	fmt.Println("🛍️  Starting E-Commerce Integration Service on :9090...")
	fmt.Println("")
	fmt.Println("📝 Services:")
	fmt.Println("  OrderOrchestrator:")
	fmt.Println("    - ProcessOrder (orchestrates Stripe + SendGrid + Shippo)")
	fmt.Println("    - GetOrder (retrieve order status)")
	fmt.Println("")
	fmt.Println("  WebhookProcessor:")
	fmt.Println("    - ProcessStripeWebhook (handle Stripe events)")
	fmt.Println("")
	fmt.Println("🔌 External Integrations:")
	fmt.Println("  💳 Stripe - Payment processing")
	fmt.Println("  📧 SendGrid - Email notifications")
	fmt.Println("  📦 Shippo - Shipping labels")
	fmt.Println("")
	
	mockMode := "MOCK_MODE enabled (no real API calls)"
	if os.Getenv("MOCK_MODE") != "true" {
		mockMode = "REAL MODE (using actual APIs)"
	}
	fmt.Println("⚙️  Mode:", mockMode)
	
	if err := restateServer.Start(context.Background(), ":9090"); err != nil {
		log.Fatal("Server error:", err)
	}
}
```

### Step 8: Go Module (`go.mod`)

```go
module external-integration

go 1.23

require github.com/restatedev/sdk-go v0.13.1
```

## 🚀 Running the Service

### Step 1: Start Restate Server

```bash
docker run --name restate_dev --rm \
  -p 8080:8080 -p 9070:9070 -p 9091:9091 \
  --add-host=host.docker.internal:host-gateway \
  docker.io/restatedev/restate:latest
```

### Step 2: Install Dependencies

```bash
go mod tidy
```

### Step 3: Start Integration Service

```bash
export MOCK_MODE=true
go run .
```

### Step 4: Register with Restate

```bash
curl -X POST http://localhost:9070/deployments \
  -H 'Content-Type: application/json' \
  -d '{"uri": "http://host.docker.internal:9090"}'
```

## 🧪 Testing the Integration

### Test 1: Process Complete Order

```bash
curl -X POST http://localhost:8080/OrderOrchestrator/order-001/ProcessOrder \
  -H 'Content-Type: application/json' \
  -d '{
    "items": [
      {"productId": "prod-123", "quantity": 2, "price": 2500},
      {"productId": "prod-456", "quantity": 1, "price": 5000}
    ],
    "customer": {
      "email": "alice@example.com",
      "name": "Alice Smith"
    },
    "shipping": {
      "street": "123 Main St",
      "city": "San Francisco",
      "state": "CA",
      "zip": "94105",
      "country": "US"
    }
  }'
```

**Expected Response:**
```json
{
  "orderId": "order-001",
  "status": "confirmed",
  "chargeId": "ch_mock_1700000001",
  "trackingNumber": "1Z999AA11700000001",
  "message": "Order processed successfully"
}
```

### Test 2: Verify Idempotency

```bash
# Send exact same request again
curl -X POST http://localhost:8080/OrderOrchestrator/order-001/ProcessOrder \
  [same payload as above]
```

**Expected:** Same result, no duplicate charges!

### Test 3: Get Order Status

```bash
curl -X POST http://localhost:8080/OrderOrchestrator/order-001/GetOrder \
  -H 'Content-Type: application/json' \
  -d '{}'
```

## 🎓 What You Learned

### External Integration Patterns

1. **Journaled API Calls**
   ```go
   resp, err := restate.Run(ctx, func(ctx restate.RunContext) (T, error) {
       return externalClient.Call(params)
   })
   ```

2. **Service Orchestration**
   - Stripe charge → Shippo label → SendGrid email
   - Each step journaled independently
   - Failure handling at each step

3. **Mock Mode for Development**
   - Test without real API keys
   - Simulated delays and failures
   - Easy local development

4. **Idempotent Processing**
   - Check state before creating order
   - All API calls journaled
   - Safe retries

## 🚀 Next Steps

Excellent! You've built a multi-service integration!

👉 **Continue to [Validation](./03-validation.md)**

Test webhook processing and failure scenarios!

---

**Questions?** Review [concepts](./01-concepts.md) or the [module README](./README.md).
