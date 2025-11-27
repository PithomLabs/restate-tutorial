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

	// Register Shopping Cart Virtual Object
	if err := restateServer.Bind(restate.Reflect(ShoppingCart{})); err != nil {
		log.Fatal("Failed to bind ShoppingCart:", err)
	}

	fmt.Println("🛒 Starting Shopping Cart Service on :9090...")
	fmt.Println("📝 Virtual Object: ShoppingCart")
	fmt.Println("")
	fmt.Println("Handlers:")
	fmt.Println("  Exclusive (modify state):")
	fmt.Println("    - AddItem")
	fmt.Println("    - RemoveItem")
	fmt.Println("    - UpdateQuantity")
	fmt.Println("    - ApplyCoupon")
	fmt.Println("    - Checkout")
	fmt.Println("    - ClearCart")
	fmt.Println("")
	fmt.Println("  Concurrent (read-only):")
	fmt.Println("    - GetCart")
	fmt.Println("    - GetItemCount")
	fmt.Println("")
	fmt.Println("✓ Ready to accept requests")

	if err := restateServer.Start(context.Background(), ":9090"); err != nil {
		log.Fatal("Server error:", err)
	}
}
