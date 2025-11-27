package main

import (
	"context"
	"fmt"
	"log"

	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/server"
)

func main() {
	// Create a new Restate server
	restateServer := server.NewRestate()

	// Register our services
	if err := restateServer.Bind(
		restate.Reflect(GreetingService{}),
	); err != nil {
		log.Fatal("Failed to bind GreetingService:", err)
	}

	// Start the server on port 9090
	fmt.Println("🚀 Starting Greeting Service on :9090...")
	fmt.Println("📝 Service: GreetingService")
	fmt.Println("📌 Handlers: Greet")
	fmt.Println("")
	fmt.Println("Register with Restate:")
	fmt.Println("  curl -X POST http://localhost:8080/deployments \\")
	fmt.Println("    -H 'Content-Type: application/json' \\")
	fmt.Println("    -d '{\"uri\": \"http://localhost:9090\"}'")
	fmt.Println("")

	if err := restateServer.Start(context.Background(), ":9090"); err != nil {
		log.Fatal("Server error:", err)
	}
}
