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
	return ShippingLabelResponse{}, fmt.Errorf("real Shippo integration not implemented - set MOCK_MODE=true")
}
