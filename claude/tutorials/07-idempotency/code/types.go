package main

import "time"

// Payment request from client
type PaymentRequest struct {
	Amount      int    `json:"amount"`      // Amount in cents
	Currency    string `json:"currency"`    // USD, EUR, etc.
	Description string `json:"description"` // Payment description
	CustomerID  string `json:"customerId"`  // Customer identifier
}

// Payment stored in state
type Payment struct {
	PaymentID   string    `json:"paymentId"`
	Amount      int       `json:"amount"`
	Currency    string    `json:"currency"`
	Description string    `json:"description"`
	CustomerID  string    `json:"customerId"`
	Status      string    `json:"status"`   // "pending", "completed", "failed"
	ChargeID    string    `json:"chargeId"` // External gateway charge ID
	CreatedAt   time.Time `json:"createdAt"`
	CompletedAt time.Time `json:"completedAt,omitempty"`
	ErrorMsg    string    `json:"errorMsg,omitempty"`
}

// Payment result returned to client
type PaymentResult struct {
	PaymentID string `json:"paymentId"`
	Status    string `json:"status"`
	ChargeID  string `json:"chargeId,omitempty"`
	Message   string `json:"message,omitempty"`
}

// Refund request
type RefundRequest struct {
	Reason string `json:"reason"`
	Amount int    `json:"amount,omitempty"` // Optional partial refund
}

// Refund result
type RefundResult struct {
	RefundID string `json:"refundId"`
	Status   string `json:"status"`
	Amount   int    `json:"amount"`
	Message  string `json:"message,omitempty"`
}
