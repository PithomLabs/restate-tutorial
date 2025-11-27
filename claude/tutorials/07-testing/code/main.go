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

	// Register User Service Virtual Object
	if err := restateServer.Bind(restate.Reflect(UserService{})); err != nil {
		log.Fatal("Failed to bind UserService:", err)
	}

	// Register Email Service
	if err := restateServer.Bind(restate.Reflect(EmailService{})); err != nil {
		log.Fatal("Failed to bind EmailService:", err)
	}

	fmt.Println("🧪 Starting User Registration Service on :9090...")
	fmt.Println("")
	fmt.Println("📝 Virtual Object: UserService")
	fmt.Println("Handlers:")
	fmt.Println("  Exclusive (modify state):")
	fmt.Println("    - Register")
	fmt.Println("    - VerifyEmail")
	fmt.Println("")
	fmt.Println("  Concurrent (read-only):")
	fmt.Println("    - GetProfile")
	fmt.Println("")
	fmt.Println("📧 Service: EmailService")
	fmt.Println("Handlers:")
	fmt.Println("    - SendVerificationEmail")
	fmt.Println("")
	fmt.Println("✓ Ready to accept requests")

	if err := restateServer.Start(context.Background(), ":9090"); err != nil {
		log.Fatal("Server error:", err)
	}
}
