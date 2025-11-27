package main

import "time"

// CartItem represents a product in the cart
type CartItem struct {
	ProductID   string  `json:"productId"`
	ProductName string  `json:"productName"`
	Quantity    int     `json:"quantity"`
	Price       float64 `json:"price"`
}

// Cart represents the complete cart state
type Cart struct {
	Items       []CartItem `json:"items"`
	CouponCode  string     `json:"couponCode"`
	Discount    float64    `json:"discount"` // Percentage (0-100)
	LastUpdated time.Time  `json:"lastUpdated"`
}

// AddItemRequest for adding items
type AddItemRequest struct {
	ProductID   string  `json:"productId"`
	ProductName string  `json:"productName"`
	Quantity    int     `json:"quantity"`
	Price       float64 `json:"price"`
}

// UpdateQuantityRequest for updating item quantities
type UpdateQuantityRequest struct {
	ProductID string `json:"productId"`
	Quantity  int    `json:"quantity"`
}

// CartSummary for viewing cart
type CartSummary struct {
	Items       []CartItem `json:"items"`
	Subtotal    float64    `json:"subtotal"`
	Discount    float64    `json:"discount"`
	Tax         float64    `json:"tax"`
	Total       float64    `json:"total"`
	ItemCount   int        `json:"itemCount"`
	CouponCode  string     `json:"couponCode,omitempty"`
	LastUpdated time.Time  `json:"lastUpdated"`
}

// CheckoutResult returned after checkout
type CheckoutResult struct {
	OrderID      string    `json:"orderId"`
	Total        float64   `json:"total"`
	ItemCount    int       `json:"itemCount"`
	CouponCode   string    `json:"couponCode,omitempty"`
	CheckedOutAt time.Time `json:"checkedOutAt"`
}
