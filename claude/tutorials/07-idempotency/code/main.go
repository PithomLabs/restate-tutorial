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

	// Register Payment Service Virtual Object
	if err := restateServer.Bind(restate.Reflect(PaymentService{})); err != nil {
		log.Fatal("Failed to bind PaymentService:", err)
	}

	fmt.Println("💳 Starting Payment Service on :9090...")
	fmt.Println("")
	fmt.Println("📝 Virtual Object: PaymentService")
	fmt.Println("Handlers:")
	fmt.Println("  Exclusive (state-modifying, idempotent):")
	fmt.Println("    - CreatePayment")
	fmt.Println("    - RefundPayment")
	fmt.Println("")
	fmt.Println("  Concurrent (read-only):")
	fmt.Println("    - GetPayment")
	fmt.Println("")
	fmt.Println("✓ Ready to accept requests")
	fmt.Println("")
	fmt.Println("💡 Idempotency Features:")
	fmt.Println("  - Automatic request deduplication")
	fmt.Println("  - Journaled side effects (gateway calls)")
	fmt.Println("  - State-based duplicate detection")
	fmt.Println("  - Safe retries")

	if err := restateServer.Start(context.Background(), ":9090"); err != nil {
		log.Fatal("Server error:", err)
	}
}
