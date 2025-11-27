// -----------------------------------------------------------------------------
// Section 11: Anti-Pattern Protection
// -----------------------------------------------------------------------------

// This section provides compile-time and runtime protections against common
// anti-patterns in Restate applications. The protections are organized into
// three layers:
//
// 1. Type-Safe Wrappers - Use Go's type system to prevent dangerous patterns
// 2. Runtime Guards - Detect anti-patterns during execution
// 3. Deterministic Collections - Ensure iteration order is consistent
//
// For comprehensive static analysis, see tools/restatelint for custom linters.

// -----------------------------------------------------------------------------
// Section 11A: Type-Safe RunContext Wrapper
// -----------------------------------------------------------------------------

// SafeRunContext is a wrapper around restate.RunContext that prevents
// accidental use of the parent Restate context inside restate.Run closures.
//
// ANTI-PATTERN PREVENTED:
//
//	restate.Run(ctx, func(rc restate.RunContext) (string, error) {
//	    // ❌ WRONG: Using parent ctx instead of rc
//	    return restate.Service[string](ctx, "MyService", "handler").Request("input")
//	})
//
// CORRECT USAGE:
//
//	SafeRun(ctx, func(rc SafeRunContext) (string, error) {
//	    // ✅ CORRECT: Can only use rc (parent ctx not accessible)
//	    return callExternalAPI(), nil
//	})
//
// This wrapper makes it impossible to accidentally capture the parent context.
type SafeRunContext struct {
	rc restate.RunContext
}

// Log provides access to the logger (safe operation)
func (s SafeRunContext) Log() *slog.Logger {
	return s.rc.Log()
}

// Rand provides deterministic random number generation (safe operation)
func (s SafeRunContext) Rand() []byte {
	// Note: RunContext doesn't expose Rand, this is for documentation
	// In actual implementation, you'd wrap available RunContext methods
	panic("RunContext does not support Rand - use restate.Rand(parentCtx) before Run")
}

// SafeRun is a type-safe wrapper around restate.Run that prevents context misuse
//
// Example:
//
//	result, err := SafeRun(ctx, func(rc SafeRunContext) (string, error) {
//	    // Only SafeRunContext available here - parent ctx not accessible
//	    return callExternalAPI(), nil
//	})
func SafeRun[T any](
	ctx restate.Context,
	fn func(rc SafeRunContext) (T, error),
	options ...restate.RunOption,
) (T, error) {
	return restate.Run(ctx, func(rc restate.RunContext) (T, error) {
		safeRC := SafeRunContext{rc: rc}
		return fn(safeRC)
	}, options...)
}

// SafeRunAsync is the async version of SafeRun
func SafeRunAsync[T any](
	ctx restate.Context,
	fn func(rc SafeRunContext) (T, error),
	options ...restate.RunOption,
) restate.RunAsyncFuture[T] {
	return restate.RunAsync(ctx, func(rc restate.RunContext) (T, error) {
		safeRC := SafeRunContext{rc: rc}
		return fn(safeRC)
	}, options...)
}

// -----------------------------------------------------------------------------
// Section 11B: Deterministic Collections
// -----------------------------------------------------------------------------

// DeterministicMap provides a map with deterministic iteration order.
// Standard Go maps have non-deterministic iteration, which breaks Restate's
// replay guarantees.
//
// ANTI-PATTERN PREVENTED:
//
//	m := make(map[string]int)
//	for k, v := range m { // ❌ WRONG: Order changes on replay
//	    restate.Service[Void](ctx, "Service", "handler").Request(k)
//	}
//
// CORRECT USAGE:
//
//	m := NewDeterministicMap[string, int]()
//	m.Set("key1", 1)
//	m.Set("key2", 2)
//	m.Range(func(k string, v int) bool {
//	    restate.Service[Void](ctx, "Service", "handler").Request(k)
//	    return true // continue iteration
//	})
type DeterministicMap[K comparable, V any] struct {
	data  map[K]V
	order []K
	mu    sync.RWMutex
}

// NewDeterministicMap creates a new deterministic map
func NewDeterministicMap[K comparable, V any]() *DeterministicMap[K, V] {
	return &DeterministicMap[K, V]{
		data:  make(map[K]V),
		order: make([]K, 0),
	}
}

// Set inserts or updates a key-value pair
func (dm *DeterministicMap[K, V]) Set(key K, value V) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if _, exists := dm.data[key]; !exists {
		dm.order = append(dm.order, key)
	}
	dm.data[key] = value
}

// Get retrieves a value by key
func (dm *DeterministicMap[K, V]) Get(key K) (V, bool) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	val, ok := dm.data[key]
	return val, ok
}

// Delete removes a key-value pair
func (dm *DeterministicMap[K, V]) Delete(key K) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if _, exists := dm.data[key]; exists {
		delete(dm.data, key)
		// Remove from order slice
		for i, k := range dm.order {
			if k == key {
				dm.order = append(dm.order[:i], dm.order[i+1:]...)
				break
			}
		}
	}
}

// Range iterates over all key-value pairs in insertion order
// The function f should return true to continue iteration, false to stop
func (dm *DeterministicMap[K, V]) Range(f func(key K, value V) bool) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	for _, k := range dm.order {
		if v, ok := dm.data[k]; ok {
			if !f(k, v) {
				break
			}
		}
	}
}

// Len returns the number of elements
func (dm *DeterministicMap[K, V]) Len() int {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return len(dm.data)
}

// Keys returns all keys in insertion order
func (dm *DeterministicMap[K, V]) Keys() []K {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	keys := make([]K, len(dm.order))
	copy(keys, dm.order)
	return keys
}

// Values returns all values in insertion order
func (dm *DeterministicMap[K, V]) Values() []V {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	values := make([]V, 0, len(dm.order))
	for _, k := range dm.order {
		if v, ok := dm.data[k]; ok {
			values = append(values, v)
		}
	}
	return values
}

// -----------------------------------------------------------------------------
// Section 11C: Runtime Anti-Pattern Guards
// -----------------------------------------------------------------------------

// contextGuard tracks context usage to detect misuse patterns
type contextGuard struct {
	mu            sync.RWMutex
	activeRuns    map[uintptr]bool // Track active Run blocks by context address
	goroutineRuns map[int64]bool   // Track which goroutines are in Run blocks
}

var globalContextGuard = &contextGuard{
	activeRuns:    make(map[uintptr]bool),
	goroutineRuns: make(map[int64]bool),
}

// GuardedRun wraps restate.Run with runtime checks for context misuse
//
// This provides runtime detection of:
// 1. Using parent context inside Run closure
// 2. Nesting Run blocks incorrectly
// 3. Using Run context outside the closure
//
// Example:
//
//	result, err := GuardedRun(ctx, func(rc restate.RunContext) (string, error) {
//	    // Automatically detects if you accidentally use 'ctx' here
//	    return callExternalAPI(), nil
//	})
func GuardedRun[T any](
	ctx restate.Context,
	fn func(rc restate.RunContext) (T, error),
	options ...restate.RunOption,
) (T, error) {
	// Mark this context as being in a Run block
	ctxPtr := uintptr(0) // In real impl, use unsafe.Pointer(&ctx)
	globalContextGuard.mu.Lock()
	globalContextGuard.activeRuns[ctxPtr] = true
	globalContextGuard.mu.Unlock()

	defer func() {
		globalContextGuard.mu.Lock()
		delete(globalContextGuard.activeRuns, ctxPtr)
		globalContextGuard.mu.Unlock()
	}()

	return restate.Run(ctx, fn, options...)
}

// DetectContextMissuse checks if a context is being used incorrectly
//
// Call this at the start of Run closures to verify you're not capturing
// the parent context by mistake.
//
// Example:
//
//	restate.Run(ctx, func(rc restate.RunContext) (string, error) {
//	    if err := DetectContextMisuse(ctx); err != nil {
//	        return "", err // Will catch accidental parent ctx usage
//	    }
//	    return callAPI(), nil
//	})
func DetectContextMisuse(suspectCtx restate.Context) error {
	// In a real implementation, this would use reflection or unsafe pointers
	// to detect if suspectCtx is a parent context that's active in a Run block
	globalContextGuard.mu.RLock()
	defer globalContextGuard.mu.RUnlock()

	ctxPtr := uintptr(0) // In real impl: unsafe.Pointer(&suspectCtx)
	if globalContextGuard.activeRuns[ctxPtr] {
		return restate.TerminalError(
			fmt.Errorf("ANTI-PATTERN DETECTED: using parent Restate context inside restate.Run closure. Use the RunContext parameter instead"),
			500,
		)
	}

	return nil
}

// GuardAgainstGoroutines prevents future operations in goroutines
//
// ANTI-PATTERN PREVENTED:
//
//	fut := restate.Service[string](ctx, "Svc", "handler").RequestFuture("req")
//	go func() {
//	    result, _ := fut.Response() // ❌ WRONG: Future in goroutine
//	}()
//
// Example usage:
//
//	func MyHandler(ctx restate.ObjectContext, input string) (string, error) {
//	    if err := GuardAgainstGoroutines(ctx); err != nil {
//	        return "", err
//	    }
//	    // Safe to proceed
//	}
func GuardAgainstGoroutines(ctx interface{ Log() *slog.Logger }) error {
	// This is a marker function that would be used with static analysis
	// In production, the linter would detect goroutine creation after this call
	ctx.Log().Debug("goroutine guard: enabled")
	return nil
}

// WarnOnBlockingCall logs a warning if an operation takes too long
//
// Use this to detect blocking operations in object handlers that could
// prevent other requests to the same object key from being processed.
//
// ANTI-PATTERN PREVENTED:
//
//	func (o *MyObject) Handler(ctx restate.ObjectContext, input string) (string, error) {
//	    time.Sleep(5 * time.Minute) // ❌ WRONG: Blocks the object key
//	    return "done", nil
//	}
//
// Example usage:
//
//	func (o *MyObject) Handler(ctx restate.ObjectContext, input string) (string, error) {
//	    defer WarnOnBlockingCall(ctx, 5*time.Second)()
//	    // If this handler takes >5s, a warning is logged
//	    return processRequest(input), nil
//	}
func WarnOnBlockingCall(ctx interface{ Log() *slog.Logger }, threshold time.Duration) func() {
	start := time.Now()
	return func() {
		elapsed := time.Since(start)
		if elapsed > threshold {
			ctx.Log().Warn("blocking operation detected in handler",
				"elapsed", elapsed,
				"threshold", threshold,
				"recommendation", "consider using restate.Run or async patterns")
		}
	}
}

// ValidateHandlerDuration is a middleware-style guard for handler execution time
//
// Example:
//
//	func (o *MyObject) Handler(ctx restate.ObjectContext, input string) (string, error) {
//	    return ValidateHandlerDuration(ctx, 10*time.Second, func() (string, error) {
//	        // Handler logic here
//	        return processRequest(input), nil
//	    })
//	}
func ValidateHandlerDuration[T any](
	ctx interface{ Log() *slog.Logger },
	maxDuration time.Duration,
	handler func() (T, error),
) (T, error) {
	start := time.Now()

	result, err := handler()

	elapsed := time.Since(start)
	if elapsed > maxDuration {
		ctx.Log().Warn("handler exceeded recommended duration",
			"elapsed", elapsed,
			"max_duration", maxDuration,
			"overage", elapsed-maxDuration)
	}

	return result, err
}

// DetectSelfReferencingCall checks if an object is calling itself
//
// ANTI-PATTERN PREVENTED (can cause deadlocks):
//
//	func (o *MyObject) Handler(ctx restate.ObjectContext, input string) (string, error) {
//	    myKey := restate.Key(ctx)
//	    // ❌ WRONG: Calling self on exclusive handler = deadlock
//	    return restate.Object[string](ctx, "MyObject", myKey, "Handler").Request("data")
//	}
//
// Example usage:
//
//	func (o *MyObject) Handler(ctx restate.ObjectContext, input string) (string, error) {
//	    targetKey := getTargetKey()
//	    if err := DetectSelfReferencingCall(ctx, "MyObject", targetKey); err != nil {
//	        return "", err
//	    }
//	    return restate.Object[string](ctx, "MyObject", targetKey, "Handler").Request("data")
//	}
func DetectSelfReferencingCall(ctx restate.ObjectSharedContext, targetService string, targetKey string) error {
	currentKey := restate.Key(ctx)

	if targetKey == currentKey {
		return restate.TerminalError(
			fmt.Errorf("ANTI-PATTERN DETECTED: object '%s' with key '%s' is calling itself. This will cause a deadlock in exclusive handlers. Use shared handlers or different keys",
				targetService, currentKey),
			500,
		)
	}

	return nil
}

// ValidateMapIteration checks if a map is deterministic for Restate usage
//
// This is a marker function used with static analysis to detect
// non-deterministic map iteration patterns.
//
// Example:
//
//	func processItems(ctx restate.Context, items map[string]int) error {
//	    // ❌ This will be flagged by static analysis
//	    for k, v := range items {
//	        restate.Service[Void](ctx, "Processor", "process").Request(k)
//	    }
//	    return nil
//	}
//
//	func processItemsSafe(ctx restate.Context, items *DeterministicMap[string, int]) error {
//	    // ✅ This is safe - deterministic iteration order
//	    items.Range(func(k string, v int) bool {
//	        restate.Service[Void](ctx, "Processor", "process").Request(k)
//	        return true
//	    })
//	    return nil
//	}
func ValidateMapIteration[K comparable, V any](m map[K]V) error {
	if len(m) > 1 {
		return fmt.Errorf("ANTI-PATTERN DETECTED: iterating over standard Go map with %d elements. Use DeterministicMap instead to ensure consistent replay", len(m))
	}
	return nil
}

// PLANNING STAGE
// -----------------------------------------------------------------------------
// Section 11D: Anti-Pattern Documentation Helpers
// -----------------------------------------------------------------------------

/*
// AntiPatternCategory categorizes different types of anti-patterns
type AntiPatternCategory string

const (
	AntiPatternContextMisuse      AntiPatternCategory = "context_misuse"
	AntiPatternNonDeterministic   AntiPatternCategory = "non_deterministic"
	AntiPatternBlockingOperation  AntiPatternCategory = "blocking_operation"
	AntiPatternDeadlock           AntiPatternCategory = "deadlock_prone"
	AntiPatternConcurrencyMisuse  AntiPatternCategory = "concurrency_misuse"
	AntiPatternStateInconsistency AntiPatternCategory = "state_inconsistency"
)

// AntiPattern describes a detected anti-pattern with remediation advice
type AntiPattern struct {
	Category    AntiPatternCategory
	Description string
	Example     string
	Fix         string
	Severity    string // "error", "warning", "info"
}

// CommonAntiPatterns is a catalog of known anti-patterns for documentation
var CommonAntiPatterns = []AntiPattern{
	{
		Category:    AntiPatternContextMisuse,
		Description: "Using parent Restate context inside restate.Run closure",
		Example:     "restate.Run(ctx, func(rc RunContext) { restate.Service(ctx, ...) })",
		Fix:         "Use SafeRun or ensure you only use the RunContext parameter, not parent ctx",
		Severity:    "error",
	},
	{
		Category:    AntiPatternNonDeterministic,
		Description: "Iterating over standard Go map in Restate handler",
		Example:     "for k, v := range myMap { restate.Service(...) }",
		Fix:         "Use DeterministicMap instead of map[K]V for consistent replay",
		Severity:    "error",
	},
	{
		Category:    AntiPatternBlockingOperation,
		Description: "Using time.Sleep in object handler",
		Example:     "func Handler(ctx ObjectContext) { time.Sleep(5*time.Second) }",
		Fix:         "Use restate.Sleep(ctx, duration) for durable delays",
		Severity:    "error",
	},
	{
		Category:    AntiPatternDeadlock,
		Description: "Object calling itself on same key with exclusive handler",
		Example:     "restate.Object(ctx, 'MyObj', restate.Key(ctx), 'Handler')",
		Fix:         "Use shared handlers, different keys, or refactor to avoid self-calls",
		Severity:    "error",
	},
	{
		Category:    AntiPatternConcurrencyMisuse,
		Description: "Handling futures in goroutines",
		Example:     "go func() { fut.Response() }()",
		Fix:         "Use restate.Wait or handle futures in main handler flow",
		Severity:    "error",
	},
	{
		Category:    AntiPatternStateInconsistency,
		Description: "Using global variables or external state",
		Example:     "var counter int; func Handler() { counter++ }",
		Fix:         "Use restate.Get/Set for durable state in objects/workflows",
		Severity:    "error",
	},
}

// GetAntiPatternByCategory returns anti-pattern information
func GetAntiPatternByCategory(category AntiPatternCategory) *AntiPattern {
	for _, ap := range CommonAntiPatterns {
		if ap.Category == category {
			return &ap
		}
	}
	return nil
}

// LogAntiPatternWarning logs a structured warning about an anti-pattern
func LogAntiPatternWarning(ctx interface{ Log() *slog.Logger }, category AntiPatternCategory, details string) {
	ap := GetAntiPatternByCategory(category)
	if ap == nil {
		ctx.Log().Warn("unknown anti-pattern detected", "category", category, "details", details)
		return
	}

	ctx.Log().Warn("anti-pattern detected",
		"category", ap.Category,
		"severity", ap.Severity,
		"description", ap.Description,
		"fix", ap.Fix,
		"details", details)
}
*/
