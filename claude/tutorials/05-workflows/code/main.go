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

	// Register Approval Workflow
	if err := restateServer.Bind(restate.Reflect(ApprovalWorkflow{})); err != nil {
		log.Fatal("Failed to bind ApprovalWorkflow:", err)
	}

	fmt.Println("📄 Starting Approval Workflow Service on :9090...")
	fmt.Println("🔄 Workflow: ApprovalWorkflow")
	fmt.Println("")
	fmt.Println("Handlers:")
	fmt.Println("  Main:")
	fmt.Println("    - Run (workflow orchestration)")
	fmt.Println("")
	fmt.Println("  Shared (external interactions):")
	fmt.Println("    - Approve")
	fmt.Println("    - Reject")
	fmt.Println("    - GetStatus")
	fmt.Println("")
	fmt.Println("✓ Ready to accept requests")

	if err := restateServer.Start(context.Background(), ":9090"); err != nil {
		log.Fatal("Server error:", err)
	}
}
