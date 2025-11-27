# Workflow Retention and State Management Guide

> **Managing workflow lifecycle, storage costs, and compliance in Restate**

## Overview

Restate workflows are durable, meaning their state (execution history, variables, promises, timers) is stored on disk. This durability enables long-running processes but requires careful management of storage and lifecycle.

This guide explains how to configure retention policies using `WorkflowConfig` to balance:
- **Reliability**: Ensuring workflows complete and are recoverable
- **Cost**: Minimizing storage usage
- **Compliance**: Meeting data retention requirements
- **Performance**: Keeping state size manageable

## Workflow Lifecycle

1. **Active**: Workflow is running or suspended (waiting for input/timer)
2. **Completed**: Workflow has returned a result or failed
3. **Retained**: State is kept for a grace period (for queries/debugging)
4. **Purged**: State is permanently deleted

## Configuration (`WorkflowConfig`)

The framework provides a typed configuration object to manage these policies.

```go
type WorkflowConfig struct {
    StateRetentionDays      int           // How long to keep state (1-90 days)
    EnableStatusPersistence bool          // Store status durably?
    AutoCleanupOnCompletion bool          // Delete immediately on success?
    MaxStateSizeBytes       int64         // Warning threshold for state size
    CleanupGracePeriod      time.Duration // Time to wait before auto-cleanup
}
```

### Pre-defined Profiles

| Profile | Retention | Status | Auto-Cleanup | Use Case |
|---------|-----------|--------|--------------|----------|
| **Default** | 30 days | ✅ | ❌ | General purpose, development |
| **Production** | 90 days | ✅ | ❌ | Critical business processes, audit |
| **HighVolume** | 7 days | ❌ | ✅ | High throughput, short-lived, cost-sensitive |

## Usage Patterns

### 1. Standard Production Workflow

For critical business processes (e.g., Order Processing) where audit trails and recoverability are paramount.

```go
func OrderWorkflowConfig() WorkflowConfig {
    return ProductionWorkflowConfig()
    // - 90 day retention
    // - Durable status
    // - No auto-cleanup (keep for audit)
}
```

### 2. High-Volume / Short-Lived

For millions of small tasks (e.g., Email sending, IoT data processing) where storage cost is the main concern.

```go
func EmailWorkflowConfig() WorkflowConfig {
    return HighVolumeWorkflowConfig()
    // - 7 day retention
    // - Ephemeral status (save space)
    // - Auto-cleanup on completion
}
```

### 3. Custom Compliance Policy

For regulated industries requiring specific retention periods.

```go
func FinancialWorkflowConfig() WorkflowConfig {
    cfg := ProductionWorkflowConfig()
    cfg.StateRetentionDays = 60 // Specific compliance requirement
    cfg.MaxStateSizeBytes = 5 * 1024 * 1024 // 5MB limit
    return cfg
}
```

### 4. Fluent Configuration API

Build configs using chainable methods:

```go
func CustomWorkflowConfig() WorkflowConfig {
    return DefaultWorkflowConfig().
        WithCustomRetention(45).
        WithAutoCleanup(true, 6*time.Hour).
        WithMaxStateSize(2 * 1024 * 1024) // 2MB
}
```

## Integrating with Restate SDK

The framework provides helpers to bridge `WorkflowConfig` to native Restate SDK options.

### Option 1: Using ToRestateOptions()

```go
import (
    restate "github.com/restatedev/sdk-go"
)

func RegisterWorkflows(server *server.Restate) {
    config := ProductionWorkflowConfig()
    
    // Apply config to workflow definition
    workflow := restate.NewWorkflow("MyWorkflow", config.ToRestateOptions()...)
    
    server.Bind(workflow)
}
```

### Option 2: Inline Configuration

```go
func main() {
    server := server.NewRestate()
    
    // Short-lived notification workflow
    notifConfig := HighVolumeWorkflowConfig()
    server.Bind(restate.NewWorkflow("NotificationWorkflow", notifConfig.ToRestateOptions()...))
    
    // Long-running order workflow
    orderConfig := ProductionWorkflowConfig()
    server.Bind(restate.NewWorkflow("OrderWorkflow", orderConfig.ToRestateOptions()...))
    
    server.Start(context.Background(), ":9080")
}
```

### What ToRestateOptions() Does

The `ToRestateOptions()` method translates our `WorkflowConfig` to Restate SDK options:

- `StateRetentionDays` → `restate.WithWorkflowRetention(duration)`
- Also sets `restate.WithIdempotencyRetention(duration)` to match workflow retention

**Note:** `AutoCleanupOnCompletion` must be handled in application logic (see next section).

## Best Practices

### 1. Estimate Storage Costs

Use the helper to estimate impact before deploying:

```go
config := DefaultWorkflowConfig()
// Estimate: 10k workflows/day, 50KB state each
gbPerMonth := config.EstimateStorageCost(10000, 50)
fmt.Printf("Estimated storage: %.2f GB\n", gbPerMonth)
```

### 2. Monitor State Size

Large state slows down recovery and increases costs.
- **Avoid**: Storing large blobs (images, PDFs) in workflow state.
- **Prefer**: Storing references (S3 URLs, database IDs).
- **Use**: `MaxStateSizeBytes` to get warnings in logs.

### 3. Manage Status Persistence

`EnableStatusPersistence` controls whether `WorkflowStatus` is stored durably.
- **Enable** for long-running workflows where you need to query "What is the status of Order 123?"
- **Disable** for high-frequency workflows where you only care about the final result.

### 4. Handle "Zombie" Workflows

Workflows that fail to complete consume retention quota.
- Implement timeouts using `restate.After` or `WorkflowTimer`.
- Use `AutoCleanupOnCompletion` to ensure successful ones don't eat up quota.

## Implementation Example

See [workflow_retention_example.go](examples/workflow_retention_example.go) for a complete standalone demo.

### Complete Server Setup

```go
package main

import (
    "context"
    "log/slog"
    
    restate "github.com/restatedev/sdk-go"
    "github.com/restatedev/sdk-go/server"
)

// Define your workflow
type OrderWorkflow struct{}

func (w *OrderWorkflow) GetConfig() WorkflowConfig {
    return ProductionWorkflowConfig()
}

func (w *OrderWorkflow) Run(ctx restate.WorkflowContext, orderID string) (string, error) {
    cfg := w.GetConfig()
    
    // 1. Validate and log configuration
    if err := cfg.Validate(ctx.Log()); err != nil {
        ctx.Log().Warn("Invalid workflow config", "error", err)
    }
    cfg.LogConfiguration(ctx.Log(), "OrderWorkflow")
    
    // 2. Monitor state size if needed
    estimatedSize := int64(512 * 1024) // 512KB estimate
    if err := cfg.MonitorStateSize(ctx, estimatedSize); err != nil {
        ctx.Log().Warn("State size warning", "error", err)
    }
    
    // 3. Your workflow logic here
    ctx.Log().Info("Processing order", "orderId", orderID)
    
    return "completed", nil
}

func main() {
    // Create server
    srv := server.NewRestate()
    
    // Get workflow config
    config := (&OrderWorkflow{}).GetConfig()
    
    // Register workflow WITH retention options
    srv.Bind(restate.NewWorkflow("OrderWorkflow", config.ToRestateOptions()...).
        Handler("run", restate.NewWorkflowHandler(
            func(ctx restate.WorkflowContext, req string) (string, error) {
                return (&OrderWorkflow{}).Run(ctx, req)
            },
        )),
    )
    
    // Start server
    if err := srv.Start(context.Background(), ":9080"); err != nil {
        slog.Error("Server failed", "error", err)
    }
}
```

### Multi-Workflow Server

```go
func main() {
    srv := server.NewRestate()
    
    // Production workflow (90 days retention)
    orderConfig := ProductionWorkflowConfig()
    srv.Bind(restate.NewWorkflow("OrderWorkflow", orderConfig.ToRestateOptions()...))
    
    // High-volume workflow (7 days retention, auto-cleanup)
    notifConfig := HighVolumeWorkflowConfig()
    srv.Bind(restate.NewWorkflow("NotificationWorkflow", notifConfig.ToRestateOptions()...))
    
    // Custom compliance workflow (60 days)
    auditConfig := DefaultWorkflowConfig().WithCustomRetention(60)
    srv.Bind(restate.NewWorkflow("AuditWorkflow", auditConfig.ToRestateOptions()...))
    
    srv.Start(context.Background(), ":9080")
}
```

## Troubleshooting

**Warning: "very short retention period"**
- **Cause**: `StateRetentionDays` < 7.
- **Risk**: If a workflow fails, you might lose the evidence before you can debug it.
- **Fix**: Increase retention or ensure you have external logs.

**Warning: "large state size limit"**
- **Cause**: `MaxStateSizeBytes` > 5MB.
- **Risk**: Replay performance will degrade.
- **Fix**: Offload large data to object storage (S3) and store only pointers.

**Warning: "cleanup grace period exceeds retention"**
- **Cause**: `CleanupGracePeriod` > `StateRetentionDays`.
- **Risk**: State will be deleted by retention policy before grace period ends.
- **Fix**: Reduce grace period or increase retention.
