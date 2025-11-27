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

	// Register the weather service
	if err := restateServer.Bind(
		restate.Reflect(WeatherService{}),
	); err != nil {
		log.Fatal("Failed to bind service:", err)
	}

	fmt.Println("🌤️  Starting Weather Aggregation Service on :9090...")
	fmt.Println("📝 Service: WeatherService")
	fmt.Println("📌 Handler: GetWeather")
	fmt.Println("")
	fmt.Println("Register with:")
	fmt.Println("  curl -X POST http://localhost:8080/deployments \\")
	fmt.Println("    -H 'Content-Type: application/json' \\")
	fmt.Println("    -d '{\"uri\": \"http://localhost:9090\"}'")
	fmt.Println("")

	if err := restateServer.Start(context.Background(), ":9090"); err != nil {
		log.Fatal("Server error:", err)
	}
}
