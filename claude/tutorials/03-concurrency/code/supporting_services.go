package main

import (
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
)

// ============================================
// Inventory Service
// ============================================

type InventoryService struct{}

type InventoryRequest struct {
	ProductID string `json:"productId"`
	Quantity  int    `json:"quantity"`
	Warehouse string `json:"warehouse"`
}

type InventoryResponse struct {
	Available bool   `json:"available"`
	Warehouse string `json:"warehouse"`
	Quantity  int    `json:"quantity"`
}

func (InventoryService) CheckInventory(
	ctx restate.Context,
	req InventoryRequest,
) (InventoryResponse, error) {
	// Simulate database query delay
	err := restate.Sleep(ctx, 80*time.Millisecond)
	if err != nil {
		return InventoryResponse{}, err
	}

	// Simulate occasional failures
	if restate.Rand(ctx).Float64() < 0.05 {
		return InventoryResponse{}, fmt.Errorf("warehouse %s temporarily unavailable", req.Warehouse)
	}

	// Simulate inventory availability (80% available)
	available := restate.Rand(ctx).Float64() < 0.8

	return InventoryResponse{
		Available: available,
		Warehouse: req.Warehouse,
		Quantity:  req.Quantity,
	}, nil
}

// ============================================
// Payment Service
// ============================================

type PaymentService struct{}

type PaymentRequest struct {
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	CardLast4 string  `json:"cardLast4"`
}

type PaymentResponse struct {
	Authorized    bool   `json:"authorized"`
	TransactionID string `json:"transactionId"`
}

func (PaymentService) AuthorizePayment(
	ctx restate.Context,
	req PaymentRequest,
) (PaymentResponse, error) {
	// Simulate payment gateway delay
	err := restate.Sleep(ctx, 120*time.Millisecond)
	if err != nil {
		return PaymentResponse{}, err
	}

	// Simulate failures
	if restate.Rand(ctx).Float64() < 0.02 {
		return PaymentResponse{}, fmt.Errorf("payment gateway timeout")
	}

	// Simulate authorization (95% success)
	authorized := restate.Rand(ctx).Float64() < 0.95

	txID := ""
	if authorized {
		txID = fmt.Sprintf("tx_%s", restate.UUID(ctx).String()[:8])
	}

	return PaymentResponse{
		Authorized:    authorized,
		TransactionID: txID,
	}, nil
}

// ============================================
// Fraud Detection Service
// ============================================

type FraudService struct{}

type FraudRequest struct {
	UserID    string  `json:"userId"`
	Amount    float64 `json:"amount"`
	IPAddress string  `json:"ipAddress"`
}

type FraudResponse struct {
	RiskScore float64 `json:"riskScore"` // 0-100
	Flagged   bool    `json:"flagged"`
}

func (FraudService) CheckFraud(
	ctx restate.Context,
	req FraudRequest,
) (FraudResponse, error) {
	// Simulate ML model inference delay
	err := restate.Sleep(ctx, 150*time.Millisecond)
	if err != nil {
		return FraudResponse{}, err
	}

	// Simulate risk score (mostly low risk)
	riskScore := restate.Rand(ctx).Float64() * 50 // 0-50 (low risk)

	// Occasionally flag high risk
	if restate.Rand(ctx).Float64() < 0.1 {
		riskScore = 50 + restate.Rand(ctx).Float64()*50 // 50-100 (high risk)
	}

	return FraudResponse{
		RiskScore: riskScore,
		Flagged:   riskScore > 70,
	}, nil
}

// ============================================
// Shipping Service
// ============================================

type ShippingService struct{}

type ShippingRequest struct {
	Weight      float64 `json:"weight"`
	Destination string  `json:"destination"`
	Carrier     string  `json:"carrier"`
}

type ShippingResponse struct {
	Carrier       string  `json:"carrier"`
	Cost          float64 `json:"cost"`
	EstimatedDays int     `json:"estimatedDays"`
}

func (ShippingService) CalculateShipping(
	ctx restate.Context,
	req ShippingRequest,
) (ShippingResponse, error) {
	// Simulate API call delay
	err := restate.Sleep(ctx, 100*time.Millisecond)
	if err != nil {
		return ShippingResponse{}, err
	}

	// Simulate carrier-specific costs
	baseCost := req.Weight * 2.5
	switch req.Carrier {
	case "FastShip":
		baseCost *= 1.5
	case "Standard":
		baseCost *= 1.0
	case "Economy":
		baseCost *= 0.7
	}

	days := 3
	if req.Carrier == "FastShip" {
		days = 1
	} else if req.Carrier == "Economy" {
		days = 7
	}

	return ShippingResponse{
		Carrier:       req.Carrier,
		Cost:          baseCost,
		EstimatedDays: days,
	}, nil
}
