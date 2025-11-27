package main

import (
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	//	"github.com/yourorg/framework" // Import your framework package
)

// -----------------------------------------------------------------------------
// Example 1: Safe Run Context Usage
// -----------------------------------------------------------------------------

// ❌ BAD: Context misuse leads to runtime panics
func BadExternalAPICall(ctx restate.Context, url string) (string, error) {
	return restate.Run(ctx, func(rc restate.RunContext) (string, error) {
		// WRONG: Using parent 'ctx' instead of 'rc'
		// This will cause non-deterministic behavior during replay
		time.Sleep(1 * time.Second) // Also wrong - should use restate.Sleep
		return callHTTPAPI(url)
	})
}

// ✅ GOOD: Using SafeRun prevents accidental context capture
func GoodExternalAPICall(ctx restate.Context, url string) (string, error) {
	return SafeRun(ctx, func(rc SafeRunContext) (string, error) {
		// CORRECT: Only SafeRunContext available
		// Trying to use 'ctx' here would be a compile error
		return callHTTPAPI(url)
	})
}

// ✅ ALTERNATIVE: Manual runtime check
func ExternalAPICallWithGuard(ctx restate.Context, url string) (string, error) {
	return restate.Run(ctx, func(rc restate.RunContext) (string, error) {
		// Runtime guard - catches accidental ctx usage
		if err := DetectContextMisuse(ctx); err != nil {
			return "", err
		}
		return callHTTPAPI(url)
	})
}

// -----------------------------------------------------------------------------
// Example 2: Deterministic Map Iteration
// -----------------------------------------------------------------------------

// ❌ BAD: Standard map iteration is non-deterministic
func BadBatchProcessor(ctx restate.Context, items map[string]string) error {
	// WRONG: Iteration order changes between executions
	for key, value := range items {
		_, err := restate.Service[restate.Void](ctx, "Processor", "process").
			Request(map[string]string{"key": key, "value": value})
		if err != nil {
			return err
		}
	}
	return nil
}

// ✅ GOOD: DeterministicMap ensures consistent iteration order
func GoodBatchProcessor(ctx restate.Context, items *DeterministicMap[string, string]) error {
	// CORRECT: Iteration order is deterministic (insertion order)
	items.Range(func(key string, value string) bool {
		_, err := restate.Service[restate.Void](ctx, "Processor", "process").
			Request(map[string]string{"key": key, "value": value})
		if err != nil {
			ctx.Log().Error("processing failed", "key", key, "error", err)
			return false // stop iteration
		}
		return true // continue
	})
	return nil
}

// Example: Building a DeterministicMap
func BuildOrderedMap(ctx restate.Context) *DeterministicMap[string, int] {
	m := NewDeterministicMap[string, int]()

	// Insert in specific order
	m.Set("first", 1)
	m.Set("second", 2)
	m.Set("third", 3)

	// Update existing key (preserves position)
	m.Set("first", 100)

	// Iteration will always be: first(100), second(2), third(3)
	return m
}

// -----------------------------------------------------------------------------
// Example 3: Avoiding Blocking Operations
// -----------------------------------------------------------------------------

// ❌ BAD: time.Sleep blocks the object handler
type BadOrderObject struct{}

func (o *BadOrderObject) ProcessOrder(ctx restate.ObjectContext, orderID string) error {
	// WRONG: Blocks all requests for this order key
	time.Sleep(30 * time.Second)

	// Process order...
	return nil
}

// ✅ GOOD: Using durable sleep
type GoodOrderObject struct{}

func (o *GoodOrderObject) ProcessOrder(ctx restate.ObjectContext, orderID string) error {
	// CORRECT: Durable sleep releases the handler
	if err := restate.Sleep(ctx, 30*time.Second); err != nil {
		return err
	}

	// Process order...
	return nil
}

// ✅ ALSO GOOD: Monitor for unexpected blocking
type MonitoredOrderObject struct{}

func (o *MonitoredOrderObject) ProcessOrder(ctx restate.ObjectContext, orderID string) error {
	// Warn if handler takes >5 seconds
	defer WarnOnBlockingCall(ctx, 5*time.Second)()

	// If the processing below takes >5s, a warning is logged
	return processComplexOrder(orderID)
}

// ✅ ENFORCE: Validate handler duration
type ValidatedOrderObject struct{}

func (o *ValidatedOrderObject) ProcessOrder(ctx restate.ObjectContext, orderID string) error {
	return ValidateHandlerDuration(ctx, 10*time.Second, func() error {
		// Handler logic here
		return processComplexOrder(orderID)
	})
}

// -----------------------------------------------------------------------------
// Example 4: Preventing Self-Referencing Calls
// -----------------------------------------------------------------------------

// ❌ BAD: Object calling itself = deadlock
type BadUserObject struct{}

func (o *BadUserObject) UpdateProfile(ctx restate.ObjectContext, profile UserProfile) error {
	myKey := restate.Key(ctx)

	// WRONG: Calling self on same key with exclusive handler = deadlock
	valid, err := restate.Object[bool](ctx, "UserObject", myKey, "ValidateProfile").
		Request(profile)
	if err != nil {
		return err
	}

	if !valid {
		return fmt.Errorf("invalid profile")
	}

	// Update...
	return nil
}

// ✅ GOOD: Using shared handler for validation
type GoodUserObject struct{}

// Shared handler - allows concurrent access
func (o *GoodUserObject) ValidateProfile(ctx restate.ObjectSharedContext, profile UserProfile) (bool, error) {
	// Read-only validation
	currentProfile, err := restate.Get[UserProfile](ctx, "profile")
	if err != nil {
		return false, err
	}

	return validateProfileTransition(currentProfile, profile), nil
}

// Exclusive handler - updates state
func (o *GoodUserObject) UpdateProfile(ctx restate.ObjectContext, profile UserProfile) error {
	myKey := restate.Key(ctx)

	// CORRECT: Calling shared handler is safe (no deadlock)
	valid, err := restate.Object[bool](ctx, "UserObject", myKey, "ValidateProfile").
		Request(profile)
	if err != nil {
		return err
	}

	if !valid {
		return restate.TerminalError(fmt.Errorf("invalid profile"), 400)
	}

	// Update state
	restate.Set(ctx, "profile", profile)
	return nil
}

// ✅ ALTERNATIVE: Runtime detection with guard
type GuardedUserObject struct{}

func (o *GuardedUserObject) UpdateProfile(ctx restate.ObjectContext, profile UserProfile) error {
	targetKey := getTargetUserKey(profile)

	// Guard against self-call
	if err := DetectSelfReferencingCall(ctx, "UserObject", targetKey); err != nil {
		return err
	}

	// Safe to call different key
	_, err := restate.Object[restate.Void](ctx, "UserObject", targetKey, "Notify").
		Request(profile)
	return err
}

// -----------------------------------------------------------------------------
// Example 5: Proper Future Handling
// -----------------------------------------------------------------------------

// ❌ BAD: Handling futures in goroutines
func BadConcurrentAPICalls(ctx restate.Context, urls []string) error {
	futures := make([]restate.ResponseFuture[string], len(urls))

	for i, url := range urls {
		futures[i] = restate.Service[string](ctx, "APIService", "fetch").
			RequestFuture(url)
	}

	// WRONG: Goroutines with futures = non-deterministic
	for _, fut := range futures {
		go func(f restate.ResponseFuture[string]) {
			result, _ := f.Response()
			fmt.Println(result)
		}(fut)
	}

	return nil
}

// ✅ GOOD: Using restate.Wait for deterministic concurrency
func GoodConcurrentAPICalls(ctx restate.Context, urls []string) ([]string, error) {
	futures := make([]restate.Future, len(urls))

	for i, url := range urls {
		futures[i] = restate.Service[string](ctx, "APIService", "fetch").
			RequestFuture(url)
	}

	// CORRECT: Wait for all futures deterministically
	results := make([]string, 0, len(urls))
	for fut, err := range restate.Wait(ctx, futures...) {
		if err != nil {
			return nil, fmt.Errorf("API call failed: %w", err)
		}

		response, err := fut.(restate.ResponseFuture[string]).Response()
		if err != nil {
			return nil, err
		}
		results = append(results, response)
	}

	return results, nil
}

// ✅ ALTERNATIVE: Using framework helper
func EvenBetterConcurrentAPICalls(ctx restate.Context, urls []string) ([]string, error) {
	// Use framework's MapConcurrent helper
	return MapConcurrent(ctx, urls, func(url string) (string, error) {
		return SafeRun(ctx, func(rc SafeRunContext) (string, error) {
			return callHTTPAPI(url)
		})
	})
}

// -----------------------------------------------------------------------------
// Example 6: Avoiding Global State
// -----------------------------------------------------------------------------

// ❌ BAD: Global state doesn't survive restarts
var globalCounter int
var globalCache = make(map[string]string)

func BadCounterService(ctx restate.Context) error {
	// WRONG: Lost on restart
	globalCounter++
	globalCache["key"] = "value"
	return nil
}

// ✅ GOOD: Using Restate state
type GoodCounterObject struct{}

func (o *GoodCounterObject) Increment(ctx restate.ObjectContext) (int, error) {
	// CORRECT: Durable state
	counter, err := restate.Get[int](ctx, "counter")
	if err != nil {
		return 0, err
	}

	counter++
	restate.Set(ctx, "counter", counter)

	return counter, nil
}

func (o *GoodCounterObject) SetCache(ctx restate.ObjectContext, key string, value string) error {
	// CORRECT: Using DeterministicMap for state
	cache, err := restate.Get[*DeterministicMap[string, string]](ctx, "cache")
	if err != nil {
		cache = NewDeterministicMap[string, string]()
	}

	cache.Set(key, value)
	restate.Set(ctx, "cache", cache)

	return nil
}

// -----------------------------------------------------------------------------
// Comprehensive Example: Combining All Protections
// -----------------------------------------------------------------------------

type SecureOrderProcessor struct{}

func (p *SecureOrderProcessor) ProcessOrder(ctx restate.ObjectContext, order Order) (OrderResult, error) {
	// 1. Monitor for blocking operations
	defer WarnOnBlockingCall(ctx, 5*time.Second)()

	// 2. Guard against self-calls
	if err := DetectSelfReferencingCall(ctx, "OrderProcessor", order.ID); err != nil {
		return OrderResult{}, err
	}

	// 3. Use deterministic collections
	tasks := NewDeterministicMap[string, Task]()
	tasks.Set("payment", Task{Name: "ProcessPayment", Status: "pending"})
	tasks.Set("inventory", Task{Name: "CheckInventory", Status: "pending"})
	tasks.Set("shipping", Task{Name: "ArrangeShipping", Status: "pending"})

	// 4. Process tasks deterministically
	results := make([]string, 0)
	tasks.Range(func(taskName string, task Task) bool {
		// 5. Use SafeRun for external calls
		result, err := SafeRun(ctx, func(rc SafeRunContext) (string, error) {
			return processTask(task)
		})

		if err != nil {
			ctx.Log().Error("task failed", "task", taskName, "error", err)
			return false // stop processing
		}

		results = append(results, result)
		return true
	})

	// 6. Use durable state
	restate.Set(ctx, "results", results)
	restate.Set(ctx, "status", "completed")

	return OrderResult{
		OrderID: order.ID,
		Tasks:   results,
	}, nil
}

// -----------------------------------------------------------------------------
// Helper Functions and Types (Stubs)
// -----------------------------------------------------------------------------

type Order struct {
	ID    string
	Items []string
}

type OrderResult struct {
	OrderID string
	Tasks   []string
}

type Task struct {
	Name   string
	Status string
}

type UserProfile struct {
	Name  string
	Email string
}

func callHTTPAPI(url string) (string, error) {
	return "api response", nil
}

func processComplexOrder(orderID string) error {
	return nil
}

func validateProfileTransition(old, new UserProfile) bool {
	return true
}

func getTargetUserKey(profile UserProfile) string {
	return "user-key"
}

func processTask(task Task) (string, error) {
	return "task completed", nil
}

// -----------------------------------------------------------------------------
// Main: Demonstration
// -----------------------------------------------------------------------------

func main() {
	fmt.Println("=== Anti-Pattern Protection Examples ===")
	fmt.Println()
	fmt.Println("✅ All examples demonstrate safe Restate patterns")
	fmt.Println("❌ Commented code shows anti-patterns to avoid")
	fmt.Println()
	fmt.Println("Run with actual Restate server to see protections in action")
}
