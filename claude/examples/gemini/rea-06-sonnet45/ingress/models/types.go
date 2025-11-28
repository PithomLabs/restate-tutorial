package models

// Order represents a customer order with all required context
type Order struct {
	OrderID         string `json:"order_id"`
	UserID          string `json:"user_id"` // L2: Authenticated user identity
	Items           string `json:"items"`
	AmountCents     int    `json:"amount_cents"`
	ShippingAddress string `json:"shipping_address"`
}

// ShipmentRequest represents a request to initiate shipment
type ShipmentRequest struct {
	OrderID string `json:"order_id"`
	Address string `json:"address"`
}

// PaymentReceipt represents the result of a payment transaction
type PaymentReceipt struct {
	TransactionID string `json:"transaction_id"`
	Amount        int    `json:"amount"`
	Status        string `json:"status"` // "success" or "failed"
	Success       bool   `json:"success"`
}

// PaymentCallback represents a callback from an external payment gateway
// Used to resolve awakeables when payment processing completes
type PaymentCallback struct {
	AwakeableID   string `json:"awakeable_id"`
	TransactionID string `json:"transaction_id"`
	Amount        int    `json:"amount"`
	Status        string `json:"status"` // "success" or "failed"
}
