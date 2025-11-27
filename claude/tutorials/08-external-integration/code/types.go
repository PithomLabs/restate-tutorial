package main

import "time"

// Order request from client
type OrderRequest struct {
	Items    []OrderItem     `json:"items"`
	Customer CustomerInfo    `json:"customer"`
	Shipping ShippingAddress `json:"shipping"`
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
	OrderID        string          `json:"orderId"`
	Items          []OrderItem     `json:"items"`
	Customer       CustomerInfo    `json:"customer"`
	Shipping       ShippingAddress `json:"shipping"`
	Status         string          `json:"status"`
	ChargeID       string          `json:"chargeId,omitempty"`
	LabelID        string          `json:"labelId,omitempty"`
	TrackingNumber string          `json:"trackingNumber,omitempty"`
	CreatedAt      time.Time       `json:"createdAt"`
	UpdatedAt      time.Time       `json:"updatedAt"`
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
