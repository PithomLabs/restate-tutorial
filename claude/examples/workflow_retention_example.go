package main

import (
	"fmt"
	"log/slog"
	"time"

	restate "github.com/restatedev/sdk-go"
)

// -----------------------------------------------------------------------------
// Example 1: Production Order Workflow (High Reliability)
// -----------------------------------------------------------------------------

type OrderWorkflow struct{}

// Configuration: Production profile (90 days retention, durable status)
func (w *OrderWorkflow) GetConfig() WorkflowConfig {
	return ProductionWorkflowConfig()
}

func (w *OrderWorkflow) Run(ctx restate.WorkflowContext, orderID string) (string, error) {
	// 1. Log configuration for visibility
	w.GetConfig().LogConfiguration(ctx.Log(), "OrderWorkflow")

	// 2. Validate configuration (good practice)
	if err := w.GetConfig().Validate(ctx.Log()); err != nil {
		ctx.Log().Warn("Invalid workflow configuration", "error", err)
	}

	ctx.Log().Info("Processing order", "orderId", orderID)

	// Simulate long-running process
	if err := restate.Sleep(ctx, 1*time.Second); err != nil {
		return "", err
	}

	return "completed", nil
}

// -----------------------------------------------------------------------------
// Example 2: High-Volume Notification Workflow (Cost Optimized)
// -----------------------------------------------------------------------------

type NotificationWorkflow struct{}

// Configuration: High-volume profile (7 days retention, auto-cleanup)
func (w *NotificationWorkflow) GetConfig() WorkflowConfig {
	return HighVolumeWorkflowConfig()
}

func (w *NotificationWorkflow) Run(ctx restate.WorkflowContext, email string) (string, error) {
	// 1. Log configuration
	w.GetConfig().LogConfiguration(ctx.Log(), "NotificationWorkflow")

	// 2. Check storage cost estimate (for monitoring)
	// Estimate: 1M workflows/day, 10KB state each
	monthlyGB := w.GetConfig().EstimateStorageCost(1000000, 10)
	if monthlyGB > 100 {
		ctx.Log().Warn("High projected storage cost", "est_gb_month", monthlyGB)
	}

	ctx.Log().Info("Sending notification", "email", email)

	// Simulate work
	return "sent", nil
}

// -----------------------------------------------------------------------------
// Example 3: Custom Compliance Workflow (Specific Requirements)
// -----------------------------------------------------------------------------

type AuditWorkflow struct{}

// Configuration: Custom (60 days retention, strict state limit)
func (w *AuditWorkflow) GetConfig() WorkflowConfig {
	cfg := DefaultWorkflowConfig()
	cfg.StateRetentionDays = 60             // Legal requirement
	cfg.MaxStateSizeBytes = 5 * 1024 * 1024 // 5MB limit
	cfg.AutoCleanupOnCompletion = false     // Must keep for audit
	return cfg
}

func (w *AuditWorkflow) Run(ctx restate.WorkflowContext, auditID string) (string, error) {
	cfg := w.GetConfig()
	cfg.LogConfiguration(ctx.Log(), "AuditWorkflow")

	// Runtime check for state size (simulated)
	currentStateSize := int64(1024 * 1024) // 1MB
	if currentStateSize > cfg.MaxStateSizeBytes {
		ctx.Log().Error("State size exceeded limit",
			"current", currentStateSize,
			"limit", cfg.MaxStateSizeBytes)
		// Handle error or cleanup
	}

	return "audited", nil
}

// -----------------------------------------------------------------------------
// Main: Registering Services
// -----------------------------------------------------------------------------

func main() {
	// In a real application, you would register these with the server
	// server := restate.NewServer()
	// server.Bind(restate.Reflect(&OrderWorkflow{}))
	// server.Bind(restate.Reflect(&NotificationWorkflow{}))
	// server.Bind(restate.Reflect(&AuditWorkflow{}))
	// server.Start(":9080")

	// For demonstration, just print the configs
	fmt.Println("--- Workflow Retention Configuration Examples ---")

	fmt.Println("\n1. Production Order Workflow:")
	order := &OrderWorkflow{}
	printConfig(order.GetConfig())

	fmt.Println("\n2. High-Volume Notification Workflow:")
	notif := &NotificationWorkflow{}
	printConfig(notif.GetConfig())

	fmt.Println("\n3. Custom Audit Workflow:")
	audit := &AuditWorkflow{}
	printConfig(audit.GetConfig())
}

func printConfig(cfg WorkflowConfig) {
	fmt.Printf("  Retention: %d days\n", cfg.StateRetentionDays)
	fmt.Printf("  Status Persistence: %v\n", cfg.EnableStatusPersistence)
	fmt.Printf("  Auto-Cleanup: %v\n", cfg.AutoCleanupOnCompletion)
	fmt.Printf("  Max State Size: %.2f MB\n", float64(cfg.MaxStateSizeBytes)/(1024*1024))
	if cfg.AutoCleanupOnCompletion {
		fmt.Printf("  Cleanup Grace Period: %s\n", cfg.CleanupGracePeriod)
	}
}

// -----------------------------------------------------------------------------
// Stub definitions for framework types (to make example compilable standalone)
// -----------------------------------------------------------------------------

// In a real project, you would import these from the framework package
// import "github.com/your-org/your-project/framework"

type WorkflowConfig struct {
	StateRetentionDays      int
	EnableStatusPersistence bool
	AutoCleanupOnCompletion bool
	MaxStateSizeBytes       int64
	CleanupGracePeriod      time.Duration
}

func DefaultWorkflowConfig() WorkflowConfig {
	return WorkflowConfig{
		StateRetentionDays:      30,
		EnableStatusPersistence: true,
		AutoCleanupOnCompletion: false,
		MaxStateSizeBytes:       1048576,
		CleanupGracePeriod:      24 * time.Hour,
	}
}

func ProductionWorkflowConfig() WorkflowConfig {
	return WorkflowConfig{
		StateRetentionDays:      90,
		EnableStatusPersistence: true,
		AutoCleanupOnCompletion: false,
		MaxStateSizeBytes:       10 * 1024 * 1024,
		CleanupGracePeriod:      7 * 24 * time.Hour,
	}
}

func HighVolumeWorkflowConfig() WorkflowConfig {
	return WorkflowConfig{
		StateRetentionDays:      7,
		EnableStatusPersistence: false,
		AutoCleanupOnCompletion: true,
		MaxStateSizeBytes:       524288,
		CleanupGracePeriod:      time.Hour,
	}
}

func (cfg WorkflowConfig) Validate(logger *slog.Logger) error {
	return nil // Stub
}

func (cfg WorkflowConfig) LogConfiguration(logger *slog.Logger, name string) {
	// Stub
}

func (cfg WorkflowConfig) EstimateStorageCost(count int, sizeKB int) float64 {
	return float64(count*cfg.StateRetentionDays*sizeKB) / (1024 * 1024)
}
