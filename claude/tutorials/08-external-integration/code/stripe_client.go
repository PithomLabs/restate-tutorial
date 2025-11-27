package main

import (
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

	return StripeChargeResponse{}, fmt.Errorf("real Stripe integration not implemented - set MOCK_MODE=true")
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

	return fmt.Errorf("real Stripe refund not implemented - set MOCK_MODE=true")
}
