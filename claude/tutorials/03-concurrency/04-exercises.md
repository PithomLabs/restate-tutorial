# Exercises: Practice Concurrent Execution

> **Master parallel patterns with hands-on exercises**

## 🎯 Objectives

Practice:
- Building fan-out/fan-in patterns
- Using futures effectively
- Handling partial failures
- Optimizing service latency
- Racing multiple options

## 📚 Exercise Levels

- 🟢 **Beginner** - Direct application
- 🟡 **Intermediate** - Combining patterns
- 🔴 **Advanced** - Complex orchestration

---

## Exercise 1: Add Product Recommendations 🟢

**Goal:** Fetch recommendations from multiple sources in parallel

### Requirements

1. Create `RecommendationService` with three methods:
   - `GetTrendingProducts` (100ms delay)
   - `GetPersonalizedProducts` (120ms delay)
   - `GetRelatedProducts` (80ms delay)

2. Create `GetAllRecommendations` handler that calls all three in parallel

3. Aggregate and deduplicate results

### Starter Code

```go
type RecommendationService struct{}

type Product struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Price float64 `json:"price"`
}

func (RecommendationService) GetTrendingProducts(
    ctx restate.Context,
    _ restate.Void,
) ([]Product, error) {
    restate.Sleep(ctx, 100*time.Millisecond)
    
    return []Product{
        {ID: "prod1", Name: "Trending Item 1", Price: 29.99},
        {ID: "prod2", Name: "Trending Item 2", Price: 39.99},
    }, nil
}

// TODO: Implement GetPersonalizedProducts and GetRelatedProducts

func (RecommendationService) GetAllRecommendations(
    ctx restate.Context,
    userID string,
) ([]Product, error) {
    // TODO: Call all three methods in parallel using futures
    // Hint: Use RequestFuture for each
    
    trendingFut := restate.Service[[]Product](
        ctx, "RecommendationService", "GetTrendingProducts",
    ).RequestFuture(restate.Void{})
    
    // TODO: Create futures for personalized and related
    
    // TODO: Use restate.Wait to collect all results
    
    // TODO: Deduplicate by product ID
    
    // TODO: Return combined list
}
```

### Test

```bash
curl -X POST http://localhost:9080/RecommendationService/GetAllRecommendations \
  -H 'Content-Type: application/json' \
  -d '"user123"'

# Should return all products in ~120ms (not 300ms!)
```

---

## Exercise 2: Fastest Response Wins 🟡

**Goal:** Use `WaitFirst` to get the fastest result from redundant calls

### Requirements

1. Create multiple mock price-check APIs with different latencies
2. Call all simultaneously
3. Return the first result that completes
4. Cancel remaining operations (Restate handles this)

### Starter Code

```go
type PriceService struct{}

func (PriceService) CheckPriceAPI1(
    ctx restate.Context,
    productID string,
) (float64, error) {
    // Slow but reliable
    restate.Sleep(ctx, 200*time.Millisecond)
    return 99.99, nil
}

func (PriceService) CheckPriceAPI2(
    ctx restate.Context,
    productID string,
) (float64, error) {
    // Fast but sometimes fails
    if rand.Float64() < 0.3 {
        return 0, fmt.Errorf("API2 unavailable")
    }
    restate.Sleep(ctx, 50*time.Millisecond)
    return 99.99, nil
}

func (PriceService) CheckPriceAPI3(
    ctx restate.Context,
    productID string,
) (float64, error) {
    // Medium speed
    restate.Sleep(ctx, 120*time.Millisecond)
    return 99.99, nil
}

func (PriceService) GetBestPrice(
    ctx restate.Context,
    productID string,
) (float64, error) {
    // TODO: Call all three APIs in parallel
    fut1 := restate.Service[float64](ctx, "PriceService", "CheckPriceAPI1").
        RequestFuture(productID)
    fut2 := restate.Service[float64](ctx, "PriceService", "CheckPriceAPI2").
        RequestFuture(productID)
    fut3 := restate.Service[float64](ctx, "PriceService", "CheckPriceAPI3").
        RequestFuture(productID)
    
    // TODO: Use restate.WaitFirst to get fastest result
    winner, err := restate.WaitFirst(ctx, fut1, fut2, fut3)
    if err != nil {
        return 0, err
    }
    
    // TODO: Extract price from winning future
    switch winner {
    case fut1:
        return fut1.Response()
    case fut2:
        return fut2.Response()
    case fut3:
        return fut3.Response()
    }
    
    return 0, fmt.Errorf("no winner")
}
```

### Test

```bash
# Run multiple times - should usually return in ~50-120ms
for i in {1..10}; do
  time curl -s -X POST http://localhost:9080/PriceService/GetBestPrice \
    -H 'Content-Type: application/json' \
    -d '"prod456"'
  echo ""
done
```

**Expected:** Most calls complete in 50-120ms (not 200ms!)

---

## Exercise 3: Parallel Data Enrichment 🟡

**Goal:** Enrich product data from multiple sources

### Requirements

1. Base product info
2. Fetch reviews (async)
3. Fetch availability (async)
4. Fetch related products (async)
5. Combine all into enriched response

### Starter Code

```go
type ProductService struct{}

type ProductInfo struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Price float64 `json:"price"`
}

type EnrichedProduct struct {
    Info            ProductInfo `json:"info"`
    AverageRating   float64     `json:"averageRating"`
    ReviewCount     int         `json:"reviewCount"`
    Available       bool        `json:"available"`
    RelatedProducts []string    `json:"relatedProducts"`
}

func (ProductService) GetProductInfo(
    ctx restate.Context,
    productID string,
) (ProductInfo, error) {
    return ProductInfo{
        ID:    productID,
        Name:  "Sample Product",
        Price: 99.99,
    }, nil
}

func (ProductService) GetReviews(
    ctx restate.Context,
    productID string,
) (struct{ Rating float64; Count int }, error) {
    restate.Sleep(ctx, 100*time.Millisecond)
    return struct{ Rating float64; Count int }{4.5, 128}, nil
}

func (ProductService) CheckAvailability(
    ctx restate.Context,
    productID string,
) (bool, error) {
    restate.Sleep(ctx, 80*time.Millisecond)
    return true, nil
}

func (ProductService) GetRelated(
    ctx restate.Context,
    productID string,
) ([]string, error) {
    restate.Sleep(ctx, 90*time.Millisecond)
    return []string{"prod2", "prod3", "prod4"}, nil
}

func (ProductService) GetEnrichedProduct(
    ctx restate.Context,
    productID string,
) (EnrichedProduct, error) {
    // Get base info (no need to parallelize this)
    info, err := restate.Service[ProductInfo](
        ctx, "ProductService", "GetProductInfo",
    ).Request(productID)
    if err != nil {
        return EnrichedProduct{}, err
    }
    
    // TODO: Fetch reviews, availability, and related products in parallel
    
    // TODO: Combine into EnrichedProduct
    
    return EnrichedProduct{
        Info: info,
        // TODO: Add enriched data
    }, nil
}
```

### Test

```bash
curl -X POST http://localhost:9080/ProductService/GetEnrichedProduct \
  -H 'Content-Type: application/json' \
  -d '"prod123"'

# Should return in ~100ms (all parallel), not 270ms (sequential)
```

---

## Exercise 4: Batch Processing with Limits 🔴

**Goal:** Process multiple items in parallel with concurrency limit

### Requirements

1. Process array of items
2. Limit to 5 concurrent operations
3. Use batching to avoid overwhelming services
4. Return all results

### Starter Code

```go
func (OrderService) ProcessBulkOrders(
    ctx restate.Context,
    orders []OrderRequest,
) ([]OrderResult, error) {
    const maxConcurrent = 5
    var results []OrderResult
    
    // Process in batches
    for i := 0; i < len(orders); i += maxConcurrent {
        end := i + maxConcurrent
        if end > len(orders) {
            end = len(orders)
        }
        
        batch := orders[i:end]
        
        // TODO: Create futures for this batch
        var futures []restate.ResponseFuture[OrderResult]
        for _, order := range batch {
            fut := restate.Service[OrderResult](
                ctx, "OrderService", "ProcessOrder",
            ).RequestFuture(order)
            futures = append(futures, fut)
        }
        
        // TODO: Wait for batch to complete
        for fut, err := range restate.Wait(ctx, futures...) {
            if err != nil {
                // Log error but continue
                ctx.Log().Error("Order failed", "error", err)
                continue
            }
            result, _ := fut.(restate.ResponseFuture[OrderResult]).Response()
            results = append(results, result)
        }
    }
    
    return results, nil
}
```

### Test

```bash
# Create test orders file
cat > bulk_orders.json <<EOF
[
  {"userId": "user1", "productId": "prod1", "quantity": 1, "amount": 50.00, "weight": 2.0, "destination": "NYC"},
  {"userId": "user2", "productId": "prod2", "quantity": 2, "amount": 75.00, "weight": 3.0, "destination": "LA"},
  {"userId": "user3", "productId": "prod3", "quantity": 1, "amount": 100.00, "weight": 5.0, "destination": "Chicago"},
  {"userId": "user4", "productId": "prod4", "quantity": 3, "amount": 150.00, "weight": 10.0, "destination": "Houston"},
  {"userId": "user5", "productId": "prod5", "quantity": 1, "amount": 60.00, "weight": 2.5, "destination": "Phoenix"},
  {"userId": "user6", "productId": "prod6", "quantity": 2, "amount": 90.00, "weight": 4.0, "destination": "Philly"},
  {"userId": "user7", "productId": "prod7", "quantity": 1, "amount": 120.00, "weight": 6.0, "destination": "Austin"}
]
EOF

curl -X POST http://localhost:9080/OrderService/ProcessBulkOrders \
  -H 'Content-Type: application/json' \
  -d @bulk_orders.json | jq 'length'

# Should process 7 orders in 2 batches (5 + 2)
```

---

## Exercise 5: Cascade Pattern 🔴

**Goal:** Chain parallel operations where next stage depends on previous results

### Requirements

1. Stage 1: Fetch user profile
2. Stage 2 (parallel): Based on user preferences, fetch relevant data
3. Stage 3 (parallel): Process each data item
4. Stage 4: Aggregate final results

### Starter Code

```go
type UserService struct{}

type UserProfile struct {
    ID          string   `json:"id"`
    Preferences []string `json:"preferences"` // e.g., ["sports", "tech", "gaming"]
}

type ContentItem struct {
    Category string `json:"category"`
    Title    string `json:"title"`
}

type PersonalizedFeed struct {
    UserID  string        `json:"userId"`
    Items   []ContentItem `json:"items"`
}

func (UserService) GetUserProfile(
    ctx restate.Context,
    userID string,
) (UserProfile, error) {
    restate.Sleep(ctx, 50*time.Millisecond)
    return UserProfile{
        ID:          userID,
        Preferences: []string{"sports", "tech", "gaming"},
    }, nil
}

func (UserService) FetchCategoryContent(
    ctx restate.Context,
    category string,
) ([]ContentItem, error) {
    restate.Sleep(ctx, 100*time.Millisecond)
    return []ContentItem{
        {Category: category, Title: category + " Article 1"},
        {Category: category, Title: category + " Article 2"},
    }, nil
}

func (UserService) GetPersonalizedFeed(
    ctx restate.Context,
    userID string,
) (PersonalizedFeed, error) {
    // Stage 1: Get user profile
    profile, err := restate.Service[UserProfile](
        ctx, "UserService", "GetUserProfile",
    ).Request(userID)
    if err != nil {
        return PersonalizedFeed{}, err
    }
    
    // Stage 2: Fetch content for each preference (parallel)
    var futures []restate.ResponseFuture[[]ContentItem]
    for _, pref := range profile.Preferences {
        fut := restate.Service[[]ContentItem](
            ctx, "UserService", "FetchCategoryContent",
        ).RequestFuture(pref)
        futures = append(futures, fut)
    }
    
    // Stage 3: Collect and combine results
    var allItems []ContentItem
    for fut, err := range restate.Wait(ctx, futures...) {
        if err != nil {
            continue
        }
        items, _ := fut.(restate.ResponseFuture[[]ContentItem]).Response()
        allItems = append(allItems, items...)
    }
    
    return PersonalizedFeed{
        UserID: userID,
        Items:  allItems,
    }, nil
}
```

### Test

```bash
curl -X POST http://localhost:9080/UserService/GetPersonalizedFeed \
  -H 'Content-Type: application/json' \
  -d '"user123"'

# Should complete in ~150ms (50ms + max(100ms parallel))
# Not 350ms (50ms + 3*100ms sequential)
```

---

## ✅ Exercise Checklist

- [ ] Exercise 1: Product Recommendations (Beginner)
- [ ] Exercise 2: Fastest Response Wins (Intermediate)
- [ ] Exercise 3: Data Enrichment (Intermediate)
- [ ] Exercise 4: Batch Processing (Advanced)
- [ ] Exercise 5: Cascade Pattern (Advanced)

## 📁 Solutions

Complete solutions are available in [solutions/](./solutions/):

- `exercise1.go` - Recommendations aggregation
- `exercise2.go` - Race pattern with WaitFirst
- `exercise3.go` - Product enrichment
- `exercise4.go` - Batch processing
- `exercise5.go` - Cascade pipelines

## 🎯 Next Module

Congratulations! You've mastered concurrent execution.

You now understand:
- ✅ Fan-out/fan-in patterns
- ✅ Using futures for parallelism
- ✅ `restate.Wait` and `restate.WaitFirst`
- ✅ Handling partial failures in parallel execution
- ✅ Optimizing service latency

Ready to add state to your services?

👉 **Continue to [Module 4: Virtual Objects](../04-virtual-objects/README.md)**

Learn to build stateful, key-addressable services with durable state!

---

**Questions?** Review [concepts](./01-concepts.md) or check the [complete code](./code/)!
