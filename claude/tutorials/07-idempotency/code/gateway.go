package main

import (
	"fmt"
	"math/rand"
	"time"
)

// MockPaymentGateway simulates an external payment processor
type MockPaymentGateway struct{}

// ChargeResponse from gateway
type ChargeResponse struct {
	ChargeID string
	Success  bool
	ErrorMsg string
}

// Charge processes a payment (simulated)
func (g *MockPaymentGateway) Charge(
	amount int,
	currency string,
	customerID string,
) ChargeResponse {
	// Simulate network delay
	time.Sleep(100 * time.Millisecond)

	// Simulate occasional failures (10% of the time)
	if rand.Float64() < 0.1 {
		return ChargeResponse{
			Success:  false,
			ErrorMsg: "insufficient funds",
		}
	}

	// Generate charge ID
	chargeID := fmt.Sprintf("ch_%s_%d", customerID, time.Now().Unix())

	return ChargeResponse{
		ChargeID: chargeID,
		Success:  true,
	}
}

// Refund processes a refund (simulated)
func (g *MockPaymentGateway) Refund(
	chargeID string,
	amount int,
) ChargeResponse {
	// Simulate network delay
	time.Sleep(100 * time.Millisecond)

	// Generate refund ID
	refundID := fmt.Sprintf("re_%s_%d", chargeID, time.Now().Unix())

	return ChargeResponse{
		ChargeID: refundID,
		Success:  true,
	}
}
