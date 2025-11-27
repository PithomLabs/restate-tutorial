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

	// Register all services
	services := []interface{}{
		OrderService{},
		OrderServiceSlow{},
		InventoryService{},
		PaymentService{},
		FraudService{},
		ShippingService{},
	}

	for _, svc := range services {
		if err := restateServer.Bind(restate.Reflect(svc)); err != nil {
			log.Fatal("Failed to bind service:", err)
		}
	}

	fmt.Println("🛒 Starting Order Processing Services on :9090...")
	fmt.Println("📝 Services:")
	fmt.Println("  - OrderService (parallel)")
	fmt.Println("  - OrderServiceSlow (sequential)")
	fmt.Println("  - InventoryService")
	fmt.Println("  - PaymentService")
	fmt.Println("  - FraudService")
	fmt.Println("  - ShippingService")
	fmt.Println("")
	fmt.Println("✓ Ready to accept requests")

	if err := restateServer.Start(context.Background(), ":9090"); err != nil {
		log.Fatal("Server error:", err)
	}
}
