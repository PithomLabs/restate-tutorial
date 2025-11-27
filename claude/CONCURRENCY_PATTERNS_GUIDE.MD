# Concurrency Pattern Helpers - Complete Guide

## Overview

The framework provides production-ready helpers for common concurrency patterns like fan-out/fan-in, parallel mapping, and batch processing. These utilities simplify concurrent execution while maintaining Restate's durability guarantees.

## Concurrency Patterns

### 1. Fan-Out/Fan-In
Execute multiple operations concurrently and aggregate results.

### 2. Parallel Map
Apply a function to each item in a collection concurrently.

### 3. Batch Processing
Process large datasets with controlled concurrency.

### 4. Parallel Service Invocation
Invoke multiple services concurrently.

## FanOut - Concurrent Execution

### FanOut (Best Effort)
Executes all operations and collects both successes and failures.

```go
func FanOut[T any](
    ctx restate.Context,
    operations []func() (T, error),
) FanOutResult[T]

type FanOutResult[T any] struct {
    Results []T      // All results (including zero values for failed ops)
    Errors  []error  // Errors for each operation (nil for success)
    Failed  int      // Count of failed operations
    Success int      // Count of successful operations
}
```

**When to use:**
- Partial success is acceptable
- You need to know which operations failed
- Collecting metrics or statistics

### FanOutFail (Fail Fast)
Executes all operations but fails on first error.

```go
func FanOutFail[T any](
    ctx restate.Context,
    operations []func() (T, error),
) ([]T, error)
```

**When to use:**
- All operations must succeed
- Atomic batch processing
- Critical workflows

## Usage Examples

### Example 1: Fan-Out Multiple Service Calls

```go
type ProductService struct{}

func (ProductService) EnrichProducts(
    ctx restate.Context,
    productIDs []string,
) ([]Product, error) {
    inventoryClient := ServiceClient[string, InventoryInfo]{
        ServiceName: "Inventory",
        HandlerName: "Get",
    }
    
    pricingClient := ServiceClient[string, PriceInfo]{
        ServiceName: "Pricing",
        HandlerName: "Get",
    }
    
    // Fan-out: Fetch inventory and pricing concurrently for each product
    operations := make([]func() (ProductData, error), 0)
    
    for _, productID := range productIDs {
        pid := productID
        operations = append(operations, func() (ProductData, error) {
            // Concurrent fetch of inventory and price
            invFut := restate.RunAsync(ctx, func(rc restate.RunContext) (InventoryInfo, error) {
                return inventoryClient.Call(ctx, pid)
            })
            
            priceFut := restate.RunAsync(ctx, func(rc restate.RunContext) (PriceInfo, error) {
                return pricingClient.Call(ctx, pid)
            })
            
            // Wait for both
            inv, err1 := invFut.Response()
            price, err2 := priceFut.Response()
            
            if err1 != nil || err2 != nil {
                return ProductData{}, fmt.Errorf("failed to fetch data for %s", pid)
            }
            
            return ProductData{
                ProductID: pid,
                Inventory: inv,
                Price:     price,
            }, nil
        })
    }
    
    // Fan-in: Collect all results
    result := FanOut(ctx, operations)
    
    if result.Failed > 0 {
        ctx.Log().Warn("Some products failed to enrich",
            "failed", result.Failed,
            "success", result.Success)
    }
    
    // Convert to products (filter out failures)
    products := make([]Product, 0, result.Success)
    for i, data := range result.Results {
        if result.Errors[i] == nil {
            products = append(products, Product{
                ID:        data.ProductID,
                Stock:     data.Inventory.Stock,
                Price:     data.Price.Amount,
            })
        }
    }
    
    return products, nil
}
```

### Example 2: FanOutFail - All or Nothing

```go
func (OrderService) ValidateOrder(
    ctx restate.Context,
    order Order,
) error {
    // All validations must pass
    validations := []func() (bool, error){
        func() (bool, error) {
            // Check inventory
            inventoryClient := ServiceClient[CheckInventoryRequest, bool]{
                ServiceName: "Inventory",
                HandlerName: "Check",
            }
            return inventoryClient.Call(ctx, CheckInventoryRequest{
                Items: order.Items,
            })
        },
        
        func() (bool, error) {
            // Verify payment method
            paymentClient := ServiceClient[VerifyPaymentRequest, bool]{
                ServiceName: "Payment",
                HandlerName: "Verify",
            }
            return paymentClient.Call(ctx, VerifyPaymentRequest{
                PaymentMethod: order.PaymentMethod,
            })
        },
        
        func() (bool, error) {
            // Check delivery availability
            deliveryClient := ServiceClient[CheckDeliveryRequest, bool]{
                ServiceName: "Delivery",
                HandlerName: "CheckAvailability",
            }
            return deliveryClient.Call(ctx, CheckDeliveryRequest{
                Address: order.ShippingAddress,
            })
        },
    }
    
    // Fail fast if any validation fails
    results, err := FanOutFail(ctx, validations)
    if err != nil {
        return fmt.Errorf("order validation failed: %w", err)
    }
    
    // Check all validations passed
    for i, valid := range results {
        if !valid {
            return restate.TerminalError(
                fmt.Errorf("validation %d failed", i),
                400,
            )
        }
    }
    
    return nil
}
```

### Example 3: MapConcurrent - Parallel Transformation

```go
func (ImageService) ProcessImages(
    ctx restate.Context,
    imageURLs []string,
) ([]ProcessedImage, error) {
    // Apply processing to each image concurrently
    results, err := MapConcurrent(
        ctx,
        imageURLs,
        func(url string) (ProcessedImage, error) {
            // Process image
            processed, err := restate.Run(ctx, func(rc restate.RunContext) (ProcessedImage, error) {
                return processImage(url)
            })
            if err != nil {
                return ProcessedImage{}, err
            }
            
            return processed, nil
        },
    )
    
    if err != nil {
        return nil, fmt.Errorf("image processing failed: %w", err)
    }
    
    return results, nil
}
```

### Example 4: Batch Processing with Concurrency Control

```go
func (ReportService) GenerateReports(
    ctx restate.Context,
    userIDs []string,
) error {
    // Create batch processor with max 5 concurrent operations
    processor := NewBatchProcessor(ctx, 5)
    
    // Process users in controlled batches
    reports, err := processor.ProcessBatch(
        userIDs,
        func(userID string) (Report, error) {
            // Generate report for user
            reportClient := ServiceClient[GenerateReportRequest, Report]{
                ServiceName: "ReportGenerator",
                HandlerName: "Generate",
            }
            
            return reportClient.Call(ctx, GenerateReportRequest{
                UserID: userID,
                Type:   "monthly",
            })
        },
    )
    
    if err != nil {
        return fmt.Errorf("batch report generation failed: %w", err)
    }
    
    ctx.Log().Info("Reports generated", "count", len(reports))
    
    // Store reports
    for _, report := range reports {
        storeReport(ctx, report)
    }
    
    return nil
}
```

### Example 5: Parallel Service Invocation

```go
func (DashboardService) GetDashboardData(
    ctx restate.Context,
    userID string,
) (Dashboard, error) {
    // Define all service calls
    clients := []ServiceClient[any, any]{
        {ServiceName: "UserStats", HandlerName: "Get"},
        {ServiceName: "RecentActivity", HandlerName: "Get"},
        {ServiceName: "Notifications", HandlerName: "GetUnread"},
    }
    
    inputs := []any{
        UserStatsRequest{UserID: userID},
        ActivityRequest{UserID: userID, Limit: 10},
        NotificationRequest{UserID: userID},
    }
    
    // Invoke all concurrently
    results, err := ParallelInvoke(ctx, clients, inputs)
    if err != nil {
        return Dashboard{}, err
    }
    
    // Type assert results
    stats := results[0].(UserStats)
    activity := results[1].([]Activity)
    notifications := results[2].([]Notification)
    
    return Dashboard{
        UserID:        userID,
        Stats:         stats,
        RecentActivity: activity,
        Notifications: notifications,
    }, nil
}
```

### Example 6: Fan-Out with Partial Failures

```go
func (NotificationService) SendBulkNotifications(
    ctx restate.Context,
    userIDs []string,
    message Message,
) (SendResult, error) {
    notifyClient := ObjectClient[Message, SendStatus]{
        ServiceName: "UserNotification",
        HandlerName: "Send",
    }
    
    // Create operations for each user
    operations := make([]func() (SendStatus, error), len(userIDs))
    for i, userID := range userIDs {
        uid := userID
        operations[i] = func() (SendStatus, error) {
            return notifyClient.Call(ctx, uid, message)
        }
    }
    
    // Fan-out: Send to all users (don't fail on individual failures)
    result := FanOut(ctx, operations)
    
    // Collect results
    sendResult := SendResult{
        TotalSent:   result.Success,
        TotalFailed: result.Failed,
        FailedUsers: make([]string, 0),
    }
    
    for i, err := range result.Errors {
        if err != nil {
            sendResult.FailedUsers = append(sendResult.FailedUsers, userIDs[i])
            ctx.Log().Warn("Failed to send notification",
                "user_id", userIDs[i],
                "error", err.Error())
        }
    }
    
    return sendResult, nil
}
```

### Example 7: Nested Fan-Out

```go
func (DataSyncService) SyncAllData(
    ctx restate.Context,
    tenantIDs []string,
) error {
    // Outer fan-out: Process each tenant
    tenantOps := make([]func() (SyncResult, error), len(tenantIDs))
    
    for i, tenantID := range tenantIDs {
        tid := tenantID
        tenantOps[i] = func() (SyncResult, error) {
            // Get all datasets for this tenant
            datasets, err := getDatasets(ctx, tid)
            if err != nil {
                return SyncResult{}, err
            }
            
            // Inner fan-out: Sync each dataset concurrently
            datasetResults, err := MapConcurrent(
                ctx,
                datasets,
                func(dataset Dataset) (DatasetSyncStatus, error) {
                    return syncDataset(ctx, tid, dataset)
                },
            )
            
            if err != nil {
                return SyncResult{}, err
            }
            
            return SyncResult{
                TenantID: tid,
                Datasets: datasetResults,
            }, nil
        }
    }
    
    // Execute outer fan-out
    result := FanOut(ctx, tenantOps)
    
    if result.Failed > 0 {
        ctx.Log().Error("Some tenants failed to sync",
            "failed", result.Failed,
            "success", result.Success)
    }
    
    return nil
}
```

### Example 8: Rate-Limited Batch Processing

```go
func (ApiService) FetchFromExternalAPI(
    ctx restate.Context,
    ids []string,
) ([]APIResponse, error) {
    // Process in batches of 10 to respect API rate limits
    processor := NewBatchProcessor(ctx, 10)
    timer := NewWorkflowTimer(ctx.(restate.WorkflowContext))
    
    var allResponses []APIResponse
    batchSize := 10
    
    for i := 0; i < len(ids); i += batchSize {
        end := i + batchSize
        if end > len(ids) {
            end = len(ids)
        }
        
        batch := ids[i:end]
        
        // Process batch
        responses, err := processor.ProcessBatch(
            batch,
            func(id string) (APIResponse, error) {
                return callExternalAPI(ctx, id)
            },
        )
        
        if err != nil {
            return nil, err
        }
        
        allResponses = append(allResponses, responses...)
        
        // Rate limit: Wait between batches
        if end < len(ids) {
            timer.Sleep(1 * time.Second)
        }
    }
    
    return allResponses, nil
}
```

## Best Practices

### ✅ DO: Use FanOut for Independent Operations

```go
// ✓ Operations don't depend on each other
result := FanOut(ctx, []func() (Data, error){
    func() (Data, error) { return fetchFromDB(ctx) },
    func() (Data, error) { return fetchFromCache(ctx) },
    func() (Data, error) { return fetchFromAPI(ctx) },
})
```

### ✅ DO: Use FanOutFail for Critical Operations

```go
// ✓ All must succeed for consistency
results, err := FanOutFail(ctx, criticalOperations)
if err != nil {
    return fmt.Errorf("critical operation failed: %w", err)
}
```

### ✅ DO: Limit Concurrency for External APIs

```go
// ✓ Respect rate limits
processor := NewBatchProcessor(ctx, 5)  // Max 5 concurrent
results, _ := processor.ProcessBatch(items, externalAPICall)
```

### ✅ DO: Handle Partial Failures Gracefully

```go
// ✓ Check which operations failed
result := FanOut(ctx, operations)
for i, err := range result.Errors {
    if err != nil {
        ctx.Log().Warn("Operation failed", "index", i, "error", err)
        // Implement retry or compensation logic
    }
}
```

### ❌ DON'T: Fan-Out Too Many Operations

```go
// ✗ Don't fan-out thousands of operations
operations := make([]func() (T, error), 10000)  // Too many!

// ✓ Use batch processing instead
processor := NewBatchProcessor(ctx, 100)
processor.ProcessBatch(items, operation)
```

### ❌ DON'T: Ignore Errors in FanOut

```go
// ✗ Don't ignore failed operations
result := FanOut(ctx, operations)
// Missing error handling!

// ✓ Check and handle failures
if result.Failed > 0 {
    // Log, retry, or compensate
}
```

## Performance Considerations

### Optimal Batch Size

```go
// Small batches (<10): Lower throughput, better error isolation
processor := NewBatchProcessor(ctx, 5)

// Medium batches (10-50): Good balance
processor := NewBatchProcessor(ctx, 20)

// Large batches (>50): Higher throughput, harder to debug
processor := NewBatchProcessor(ctx, 100)
```

### Memory Usage

```go
// ✓ Process in chunks for large datasets
total := 100000
chunkSize := 1000

for i := 0; i < total; i += chunkSize {
    chunk := items[i:min(i+chunkSize, total)]
    processor.ProcessBatch(chunk, operation)
}
```

## Testing

```go
func TestFanOut_AllSuccess(t *testing.T) {
    ctx := mockContext()
    
    ops := []func() (int, error){
        func() (int, error) { return 1, nil },
        func() (int, error) { return 2, nil },
        func() (int, error) { return 3, nil },
    }
    
    result := FanOut(ctx, ops)
    
    assert.Equal(t, 3, result.Success)
    assert.Equal(t, 0, result.Failed)
    assert.Equal(t, []int{1, 2, 3}, result.Results)
}

func TestFanOutFail_FailsOnError(t *testing.T) {
    ctx := mockContext()
    
    ops := []func() (int, error){
        func() (int, error) { return 1, nil },
        func() (int, error) { return 0, errors.New("fail") },
        func() (int, error) { return 3, nil },
    }
    
    _, err := FanOutFail(ctx, ops)
    
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "fail")
}

func TestMapConcurrent(t *testing.T) {
    ctx := mockContext()
    
    items := []int{1, 2, 3, 4, 5}
    
    results, err := MapConcurrent(ctx, items, func(n int) (int, error) {
        return n * 2, nil
    })
    
    assert.NoError(t, err)
    assert.Equal(t, []int{2, 4, 6, 8, 10}, results)
}
```

## Conclusion

Concurrency pattern helpers provide:

- ✅ **Fan-out/fan-in** - Concurrent execution with result aggregation
- ✅ **Partial failure handling** - Continue despite individual failures
- ✅ **Controlled concurrency** - Batch processing with limits
- ✅ **Parallel mapping** - Apply functions concurrently
- ✅ **Type safety** - Generic implementations
- ✅ **Durable execution** - All operations logged for replay

**Use these patterns** to maximize throughput while maintaining Restate's durability guarantees and proper error handling.
