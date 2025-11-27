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

	// Register saga
	if err := restateServer.Bind(restate.Reflect(TravelSaga{})); err != nil {
		log.Fatal("Failed to bind TravelSaga:", err)
	}

	// Register supporting services
	if err := restateServer.Bind(restate.Reflect(FlightService{})); err != nil {
		log.Fatal("Failed to bind FlightService:", err)
	}

	if err := restateServer.Bind(restate.Reflect(HotelService{})); err != nil {
		log.Fatal("Failed to bind HotelService:", err)
	}

	if err := restateServer.Bind(restate.Reflect(CarService{})); err != nil {
		log.Fatal("Failed to bind CarService:", err)
	}

	fmt.Println("🌍 Starting Travel Booking Saga Service on :9090...")
	fmt.Println("📋 Registered:")
	fmt.Println("  - TravelSaga (workflow)")
	fmt.Println("  - FlightService")
	fmt.Println("  - HotelService")
	fmt.Println("  - CarService")
	fmt.Println("")
	fmt.Println("✓ Ready to accept bookings")

	if err := restateServer.Start(context.Background(), ":9090"); err != nil {
		log.Fatal("Server error:", err)
	}
}
