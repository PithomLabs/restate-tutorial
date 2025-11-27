package main

import (
	"context"
	"fmt"
	"log"
	"os"

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

	mockMode := "✅ MOCK_MODE enabled (no real API calls)"
	if os.Getenv("MOCK_MODE") != "true" {
		mockMode = "⚠️  REAL MODE (using actual APIs)"
	}
	fmt.Println("⚙️  Mode:", mockMode)
	fmt.Println("")

	if err := restateServer.Start(context.Background(), ":9090"); err != nil {
		log.Fatal("Server error:", err)
	}
}
