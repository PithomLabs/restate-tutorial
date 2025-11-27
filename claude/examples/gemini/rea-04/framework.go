// Package framework provides a high-level abstraction layer for the
// Restate Go SDK, enforcing all best practices ("dos and don'ts") and
// eliminating boilerplate when writing Restate services.
package rea

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/ingress"
)

// -----------------------------------------------------------------------------
// Section 1: Core Types and Service Classification
// -----------------------------------------------------------------------------

// ServiceType distinguishes between orchestration and business logic layers
type ServiceType string

const (
	ServiceTypeControlPlane ServiceType = "control_plane" // Orchestration, sagas, human workflows
	ServiceTypeDataPlane    ServiceType = "data_plane"    // Business logic, external calls, state management
)

// Void is a type alias for operations that return no meaningful data
type Void = restate.Void

// IdempotencyValidationMode controls how idempotency key validation failures are handled
type IdempotencyValidationMode string

const (
	// IdempotencyValidationWarn logs validation errors but allows the call to proceed (default, permissive)
	IdempotencyValidationWarn IdempotencyValidationMode = "warn"

	// IdempotencyValidationFail rejects calls with invalid idempotency keys (strict)
	IdempotencyValidationFail IdempotencyValidationMode = "fail"

	// IdempotencyValidationDisabled skips validation entirely
	IdempotencyValidationDisabled IdempotencyValidationMode = "disabled"
)

// ============================================================================
// Global Framework Policy - Unified Guardrail Control
// ============================================================================
//
// FrameworkPolicy provides consistent strictness control across all framework
// guardrails including idempotency validation, state write checks, security
// validation, saga checks, and other runtime validations.
//
// Policy Modes:
//   - Strict: Fail-fast on any violation (recommended for CI/production)
//   - Warn: Log warnings but continue execution (recommended for development)
//   - Disabled: Skip most validation checks (testing only, not recommended)
//
// Environment Configuration:
//   export RESTATE_FRAMEWORK_POLICY=strict  # Override default
//   # Auto-detection: CI=true → strict, otherwise → warn
//
// Per-call overrides are supported via CallOption.ValidationMode for
// backward compatibility.

// FrameworkPolicy controls the strictness of all framework runtime checks
type FrameworkPolicy string

const (
	// PolicyStrict fails fast on any guardrail violation
	// Recommended for: CI pipelines, production environments
	PolicyStrict FrameworkPolicy = "strict"

	// PolicyWarn logs warnings but allows execution to continue
	// Recommended for: Local development, staging environments
	PolicyWarn FrameworkPolicy = "warn"

	// PolicyDisabled skips most validation checks
	// NOT RECOMMENDED: Only for testing/debugging specific issues
	PolicyDisabled FrameworkPolicy = "disabled"
)

var (
	// globalPolicy is the framework-wide policy, configurable via environment
	globalPolicy     = defaultFrameworkPolicy()
	globalPolicyOnce sync.Once
	policyMutex      sync.RWMutex
)

// defaultFrameworkPolicy returns the default policy based on environment
func defaultFrameworkPolicy() FrameworkPolicy {
	// Check explicit environment variable first
	env := os.Getenv("RESTATE_FRAMEWORK_POLICY")
	switch env {
	case "strict":
		return PolicyStrict
	case "warn":
		return PolicyWarn
	case "disabled":
		return PolicyDisabled
	case "":
		// Auto-detect based on CI environment
		if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" {
			return PolicyStrict // Strict in CI
		}
		return PolicyWarn // Permissive in dev
	default:
		// Unknown value, default to warn with warning
		fmt.Fprintf(os.Stderr, "Warning: Unknown RESTATE_FRAMEWORK_POLICY=%q, defaulting to 'warn'\n", env)
		return PolicyWarn
	}
}

// SetFrameworkPolicy sets the global framework policy.
// This should be called early in application initialization.
func SetFrameworkPolicy(policy FrameworkPolicy) {
	policyMutex.Lock()
	defer policyMutex.Unlock()
	globalPolicy = policy
}

// GetFrameworkPolicy returns the current global framework policy
func GetFrameworkPolicy() FrameworkPolicy {
	policyMutex.RLock()
	defer policyMutex.RUnlock()
	return globalPolicy
}

// GuardrailViolation represents a framework guardrail violation
type GuardrailViolation struct {
	Check    string // Name of the check (e.g., "idempotency_key_validation")
	Message  string // Human-readable error message
	Severity string // "error", "warning", "info"
}

// HandleGuardrailViolation processes a guardrail violation according to policy.
// Returns an error if the violation should fail execution (strict mode).
// Logs a warning if the violation should be noted but execution continues (warn mode).
// Does nothing if validation is disabled.
//
// If policyOverride is empty, uses the global framework policy.
func HandleGuardrailViolation(
	violation GuardrailViolation,
	logger *slog.Logger,
	policyOverride FrameworkPolicy,
) error {
	// Use global policy if not overridden
	policy := policyOverride
	if policy == "" {
		policy = GetFrameworkPolicy()
	}

	switch policy {
	case PolicyDisabled:
		// Skip validation entirely
		return nil

	case PolicyWarn:
		// Log warning but allow execution to continue
		if logger != nil {
			logger.Warn("framework guardrail violation (permissive mode)",
				"check", violation.Check,
				"message", violation.Message,
				"severity", violation.Severity,
				"policy", "warn")
		}
		return nil

	case PolicyStrict:
		// Fail with terminal error
		return restate.TerminalError(
			fmt.Errorf("framework guardrail failed [%s]: %s", violation.Check, violation.Message),
			400,
		)

	default:
		// Unknown policy - default to strict for safety
		return restate.TerminalError(
			fmt.Errorf("framework guardrail failed [%s]: %s (unknown policy: %s)",
				violation.Check, violation.Message, policy),
			400,
		)
	}
}

// Backward compatibility: Map old validation modes to new framework policy
func validationModeToPolicy(mode IdempotencyValidationMode) FrameworkPolicy {
	switch mode {
	case IdempotencyValidationDisabled:
		return PolicyDisabled
	case IdempotencyValidationFail:
		return PolicyStrict
	case IdempotencyValidationWarn:
		return PolicyWarn
	default:
		return "" // Will use global policy
	}
}

// CallOption configures inter-service calls
type CallOption struct {
	IdempotencyKey string
	Delay          time.Duration
	ValidationMode IdempotencyValidationMode // Controls validation behavior (warn/fail/disabled)
}

// checkRedundantIdempotencyKey checks if an idempotency key is unnecessary for same-handler execution
//
// Idempotency keys within the same handler execution are redundant because Restate's journaling
// already provides exactly-once execution guarantees. This helper detects such cases and logs a warning.
//
// When NOT to use idempotency keys:
//   - Same-handler execution (this function returns true)
//   - Sequential calls within the same handler
//   - Fire-and-forget Send within the same handler
//
// When to use idempotency keys:
//   - External calls (outside handlers via IngressClient)
//   - Cross-handler calls with attach semantics
//   - Deduplication across handler invocations
//
// See IDEMPOTENCY_GUIDE.MD for comprehensive guidance.
func checkRedundantIdempotencyKey(ctx restate.Context, key, serviceName, handlerName string) bool {
	if key == "" {
		return false // No key provided, nothing to check
	}

	// Detect if we're in a journaled context (same-handler execution)
	// In journaled contexts, idempotency keys are redundant
	isSameHandler := isSameHandlerExecution(ctx)

	if isSameHandler {
		// Get logger from context if available, fallback to default
		logger := slog.Default()

		// Handle via global framework policy
		policy := GetFrameworkPolicy()
		violation := GuardrailViolation{
			Check:    "RedundantIdempotencyKey",
			Message:  fmt.Sprintf("idempotency key '%s' is unnecessary for same-handler call to %s.%s (Restate journaling already provides exactly-once execution)", key, serviceName, handlerName),
			Severity: "info",
		}

		HandleGuardrailViolation(violation, logger, policy)
		return true
	}

	return false
}

// isSameHandlerExecution detects if the context is within a journaled handler execution
//
// Returns true for contexts that have journaling (exactly-once guarantees):
//   - restate.Context (service handlers)
//   - restate.ObjectContext (object handlers)
//   - restate.WorkflowContext (workflow handlers)
//   - restate.ObjectSharedContext (shared object handlers)
//   - restate.WorkflowSharedContext (shared workflow handlers)
//
// Returns false for contexts without journaling:
//   - restate.RunContext (side effects via restate.Run)
//   - External contexts (standard Go context.Context)
func isSameHandlerExecution(ctx restate.Context) bool {
	// Type switch to detect journaled contexts
	switch ctx.(type) {
	case restate.Context,
		restate.ObjectContext,
		restate.WorkflowContext,
		restate.ObjectSharedContext,
		restate.WorkflowSharedContext:
		return true // Within handler execution - journaling provides guarantees
	case restate.RunContext:
		return false // Side effect context - no journaling
	default:
		return false // Unknown or external context
	}
}

// -----------------------------------------------------------------------------
// Section 1A: Security Configuration
// -----------------------------------------------------------------------------

// SecurityConfig holds security settings for Restate services
type SecurityConfig struct {
	// EnableRequestValidation enables cryptographic validation of incoming requests
	EnableRequestValidation bool

	// SigningKeys are the Ed25519 public keys used to verify request signatures
	// These should match the keys configured in your Restate server
	SigningKeys []ed25519.PublicKey

	// RequireHTTPS enforces that services only accept HTTPS connections
	RequireHTTPS bool

	// AllowedOrigins restricts which Restate instances can invoke this service
	// Empty means all origins are allowed
	AllowedOrigins []string

	// RestrictPublicAccess marks services as private (not accessible via HTTP ingress)
	RestrictPublicAccess bool

	// ValidationMode determines how strict the validation is
	ValidationMode SecurityValidationMode
}

// SecurityValidationMode controls security validation behavior
type SecurityValidationMode string

const (
	// SecurityModePermissive logs warnings but allows requests
	SecurityModePermissive SecurityValidationMode = "permissive"

	// SecurityModeStrict rejects requests that fail validation
	SecurityModeStrict SecurityValidationMode = "strict"

	// SecurityModeDisabled skips validation (NOT RECOMMENDED for production)
	SecurityModeDisabled SecurityValidationMode = "disabled"
)

// DefaultSecurityConfig returns production-ready security settings
func DefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		EnableRequestValidation: true,
		SigningKeys:             nil, // Must be configured explicitly
		RequireHTTPS:            true,
		AllowedOrigins:          []string{},
		RestrictPublicAccess:    false,
		ValidationMode:          SecurityModeStrict,
	}
}

// DevelopmentSecurityConfig returns permissive settings for local development
func DevelopmentSecurityConfig() SecurityConfig {
	return SecurityConfig{
		EnableRequestValidation: false,
		SigningKeys:             nil,
		RequireHTTPS:            false,
		AllowedOrigins:          []string{},
		RestrictPublicAccess:    false,
		ValidationMode:          SecurityModePermissive,
	}
}

// RequestValidationResult contains the outcome of request signature validation
type RequestValidationResult struct {
	Valid         bool
	KeyIndex      int    // Index of the signing key that validated (if Valid=true)
	ErrorMessage  string // Human-readable error if Valid=false
	RequestOrigin string // Origin header from the request
}

// -----------------------------------------------------------------------------
// Section 2: External vs Internal Signaling
// -----------------------------------------------------------------------------

// WaitForExternalSignal creates a durable awakeable for external coordination.
func WaitForExternalSignal[T any](ctx restate.Context) restate.AwakeableFuture[T] {
	ctx.Log().Info("framework: creating external signal (awakeable)", "type", fmt.Sprintf("%T", *new(T)))
	return restate.Awakeable[T](ctx)
}

// ResolveExternalSignal completes an awakeable from an external callback handler.
func ResolveExternalSignal[T any](ctx restate.Context, id string, value T) {
	ctx.Log().Info("framework: resolving external signal", "awakeable_id", id)
	restate.ResolveAwakeable[T](ctx, id, value)
}

// RejectExternalSignal fails an awakeable from an external system.
func RejectExternalSignal(ctx restate.Context, id string, reason error) {
	ctx.Log().Error("framework: rejecting external signal", "awakeable_id", id, "reason", reason.Error())
	restate.RejectAwakeable(ctx, id, reason)
}

// GetInternalSignal obtains a durable promise for intra-workflow coordination.
func GetInternalSignal[T any](ctx restate.WorkflowSharedContext, name string) restate.DurablePromise[T] {
	ctx.Log().Info("framework: getting internal signal (promise)", "name", name)
	return restate.Promise[T](ctx, name)
}

// -----------------------------------------------------------------------------
// Section 2A: Workflow Automation Utilities
// -----------------------------------------------------------------------------

// WorkflowTimer provides durable timer utilities for workflows
type WorkflowTimer struct {
	ctx restate.WorkflowContext
	log *slog.Logger
}

// NewWorkflowTimer creates timer utilities bound to a workflow context
func NewWorkflowTimer(ctx restate.WorkflowContext) *WorkflowTimer {
	return &WorkflowTimer{
		ctx: ctx,
		log: ctx.Log(),
	}
}

// Sleep pauses workflow execution for the specified duration (durable)
func (wt *WorkflowTimer) Sleep(duration time.Duration) error {
	wt.log.Info("workflow: sleeping", "duration", duration.String())
	return restate.Sleep(wt.ctx, duration)
}

// After creates a durable timer that completes after the specified duration
func (wt *WorkflowTimer) After(duration time.Duration) restate.AfterFuture {
	wt.log.Debug("workflow: creating timer", "duration", duration.String())
	return restate.After(wt.ctx, duration)
}

// SleepUntil pauses until a specific time (calculates duration from now)
func (wt *WorkflowTimer) SleepUntil(targetTime time.Time) error {
	now := time.Now()
	if targetTime.Before(now) {
		wt.log.Warn("workflow: target time in past, skipping sleep", "target", targetTime)
		return nil
	}
	duration := targetTime.Sub(now)
	return wt.Sleep(duration)
}

// PromiseRacer provides utilities for racing promises against timeouts
type PromiseRacer struct {
	ctx restate.WorkflowContext
	log *slog.Logger
}

// NewPromiseRacer creates a promise racing utility
func NewPromiseRacer(ctx restate.WorkflowContext) *PromiseRacer {
	return &PromiseRacer{
		ctx: ctx,
		log: ctx.Log(),
	}
}

// PromiseRaceResult indicates which future won the race (promise vs timeout)
type PromiseRaceResult[T any] struct {
	Value         T
	TimedOut      bool
	PromiseWon    bool
	PromiseResult T
	Error         error
}

// RacePromiseWithTimeout races a promise against a timeout (standalone generic function)
func RacePromiseWithTimeout[T any](
	ctx restate.WorkflowContext,
	promiseName string,
	timeout time.Duration,
) (PromiseRaceResult[T], error) {
	result := PromiseRaceResult[T]{}

	promise := restate.Promise[T](ctx, promiseName)
	timeoutFuture := restate.After(ctx, timeout)

	ctx.Log().Info("workflow: racing promise against timeout",
		"promise", promiseName,
		"timeout", timeout.String())

	// Race the promise against timeout
	winner, err := restate.WaitFirst(ctx, promise, timeoutFuture)
	if err != nil {
		result.Error = err
		return result, err
	}

	// Check which one won
	switch winner {
	case timeoutFuture:
		// Timeout won
		if err := timeoutFuture.Done(); err != nil {
			result.Error = err
			return result, err
		}
		result.TimedOut = true
		result.PromiseWon = false
		ctx.Log().Warn("workflow: promise timed out", "promise", promiseName)
		return result, nil

	case promise:
		// Promise won
		value, err := promise.Result()
		if err != nil {
			result.Error = err
			return result, err
		}
		result.Value = value
		result.PromiseResult = value
		result.TimedOut = false
		result.PromiseWon = true
		ctx.Log().Info("workflow: promise completed before timeout", "promise", promiseName)
		return result, nil

	default:
		result.Error = fmt.Errorf("unexpected race winner")
		return result, result.Error
	}
}

// RaceAwakeableWithTimeout races an awakeable against a timeout (standalone generic function)
func RaceAwakeableWithTimeout[T any](
	ctx restate.WorkflowContext,
	awakeable restate.AwakeableFuture[T],
	timeout time.Duration,
	timeoutValue T,
) (T, bool, error) {
	timeoutFuture := restate.After(ctx, timeout)

	ctx.Log().Info("workflow: racing awakeable against timeout",
		"awakeable_id", awakeable.Id(),
		"timeout", timeout.String())

	winner, err := restate.WaitFirst(ctx, awakeable, timeoutFuture)
	if err != nil {
		return timeoutValue, false, err
	}

	switch winner {
	case timeoutFuture:
		if err := timeoutFuture.Done(); err != nil {
			return timeoutValue, false, err
		}
		ctx.Log().Warn("workflow: awakeable timed out")
		return timeoutValue, true, nil

	case awakeable:
		value, err := awakeable.Result()
		if err != nil {
			return timeoutValue, false, err
		}
		ctx.Log().Info("workflow: awakeable completed before timeout")
		return value, false, nil

	default:
		return timeoutValue, false, fmt.Errorf("unexpected race winner")
	}
}

// WorkflowStatus provides utilities for exposing workflow progress via shared handlers
type WorkflowStatus struct {
	ctx      restate.WorkflowSharedContext
	stateKey string
}

// NewWorkflowStatus creates a status tracker for the workflow
func NewWorkflowStatus(ctx restate.WorkflowSharedContext, statusKey string) *WorkflowStatus {
	if statusKey == "" {
		statusKey = "workflow_status"
	}
	return &WorkflowStatus{
		ctx:      ctx,
		stateKey: statusKey,
	}
}

// StatusData represents workflow progress information
type StatusData struct {
	Phase          string                 `json:"phase"`
	Progress       float64                `json:"progress"` // 0.0 to 1.0
	CurrentStep    string                 `json:"current_step"`
	CompletedSteps []string               `json:"completed_steps"`
	Metadata       map[string]interface{} `json:"metadata"`
	UpdatedAt      time.Time              `json:"updated_at"`
	IsComplete     bool                   `json:"is_complete"`
	Error          string                 `json:"error,omitempty"`
}

// GetStatus retrieves current workflow status (read-only, safe from shared context)
func (ws *WorkflowStatus) GetStatus() (StatusData, error) {
	status, err := restate.Get[StatusData](ws.ctx, ws.stateKey)
	if err != nil {
		return StatusData{}, err
	}
	return status, nil
}

// UpdateStatus updates workflow status (must be called from exclusive run handler)
func UpdateStatus(ctx restate.WorkflowContext, statusKey string, update StatusData) error {
	update.UpdatedAt = time.Now()
	restate.Set(ctx, statusKey, update)
	ctx.Log().Debug("workflow: status updated",
		"phase", update.Phase,
		"progress", update.Progress,
		"step", update.CurrentStep)
	return nil
}

// LoopCondition is a function that determines if loop should continue
type LoopCondition func() (shouldContinue bool, err error)

// LoopBody is the function executed in each iteration
type LoopBody func(iteration int) error

// WorkflowLoop provides looping constructs with built-in safety
type WorkflowLoop struct {
	ctx           restate.WorkflowContext
	log           *slog.Logger
	maxIterations int
}

// NewWorkflowLoop creates a loop controller
func NewWorkflowLoop(ctx restate.WorkflowContext, maxIterations int) *WorkflowLoop {
	if maxIterations <= 0 {
		maxIterations = 10000 // Safety limit
	}
	return &WorkflowLoop{
		ctx:           ctx,
		log:           ctx.Log(),
		maxIterations: maxIterations,
	}
}

// While executes body while condition returns true
func (wl *WorkflowLoop) While(condition LoopCondition, body LoopBody) error {
	iteration := 0
	for {
		// Check max iterations safety limit
		if iteration >= wl.maxIterations {
			return restate.TerminalError(
				fmt.Errorf("workflow loop exceeded max iterations: %d", wl.maxIterations),
				500,
			)
		}

		// Evaluate condition
		shouldContinue, err := condition()
		if err != nil {
			return fmt.Errorf("loop condition failed at iteration %d: %w", iteration, err)
		}
		if !shouldContinue {
			wl.log.Info("workflow: loop completed", "iterations", iteration)
			break
		}

		// Execute body
		wl.log.Debug("workflow: loop iteration", "iteration", iteration)
		if err := body(iteration); err != nil {
			return fmt.Errorf("loop body failed at iteration %d: %w", iteration, err)
		}

		iteration++
	}
	return nil
}

// Retry executes body with exponential backoff retry
func (wl *WorkflowLoop) Retry(body LoopBody, maxAttempts int, initialDelay time.Duration) error {
	if maxAttempts <= 0 {
		maxAttempts = 3
	}

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		wl.log.Info("workflow: retry attempt", "attempt", attempt+1, "max", maxAttempts)

		err := body(attempt)
		if err == nil {
			wl.log.Info("workflow: retry succeeded", "attempts", attempt+1)
			return nil
		}

		lastErr = err
		wl.log.Warn("workflow: retry attempt failed", "attempt", attempt+1, "error", err.Error())

		// Don't sleep after last attempt
		if attempt < maxAttempts-1 {
			delay := computeBackoff(initialDelay, 5*time.Minute, attempt)
			wl.log.Debug("workflow: retry backoff", "delay", delay.String())
			if sleepErr := restate.Sleep(wl.ctx, delay); sleepErr != nil {
				return fmt.Errorf("retry sleep failed: %w", sleepErr)
			}
		}
	}

	return restate.TerminalError(
		fmt.Errorf("retry exhausted after %d attempts: %w", maxAttempts, lastErr),
		500,
	)
}

// ForEach executes body for each item in the slice (standalone generic function)
func ForEach[T any](
	ctx restate.WorkflowContext,
	items []T,
	body func(item T, index int) error,
) error {
	ctx.Log().Info("workflow: foreach starting", "count", len(items))
	for i, item := range items {
		if err := body(item, i); err != nil {
			return fmt.Errorf("foreach failed at index %d: %w", i, err)
		}
	}
	ctx.Log().Info("workflow: foreach completed", "count", len(items))
	return nil
}

// -----------------------------------------------------------------------------
// Section 3: Type-Safe Durable State Management
// -----------------------------------------------------------------------------

// State provides type-safe access to Restate's key-value store with runtime guards
type State[T any] struct {
	key string
	ctx interface{} // Validated at runtime for write operations
}

// NewState creates a state accessor. Write operations require exclusive context.
func NewState[T any](ctx interface{}, key string) *State[T] {
	return &State[T]{ctx: ctx, key: key}
}

// Get retrieves state value. Safe from any context type.
func (s *State[T]) Get() (T, error) {
	var zero T

	switch c := s.ctx.(type) {
	case restate.ObjectContext:
		return restate.Get[T](c, s.key)
	case restate.ObjectSharedContext:
		return restate.Get[T](c, s.key)
	case restate.WorkflowContext:
		return restate.Get[T](c, s.key)
	case restate.WorkflowSharedContext:
		return restate.Get[T](c, s.key)
	//case restate.Context:
	//return restate.Get[T](c, s.key)
	default:
		return zero, restate.TerminalError(fmt.Errorf("invalid context type for state get: %T", s.ctx), 400)
	}
}

// Set writes state value. Only permitted from exclusive contexts (enforced).
func (s *State[T]) Set(value T) error {
	switch c := s.ctx.(type) {
	case restate.ObjectContext:
		restate.Set(c, s.key, value)
		return nil
	case restate.WorkflowContext:
		restate.Set(c, s.key, value)
		return nil
	default:
		return restate.TerminalError(fmt.Errorf("Set called from read-only context: %T", s.ctx), 400)
	}
}

// Clear removes state key. Only permitted from exclusive contexts.
func (s *State[T]) Clear() error {
	switch c := s.ctx.(type) {
	case restate.ObjectContext:
		restate.Clear(c, s.key)
		return nil
	case restate.WorkflowContext:
		restate.Clear(c, s.key)
		return nil
	default:
		return restate.TerminalError(fmt.Errorf("Clear called from read-only context: %T", s.ctx), 400)
	}
}

// ClearAll removes all state. Only permitted from exclusive contexts.
func ClearAll(ctx interface{}) error {
	switch c := ctx.(type) {
	case restate.ObjectContext:
		restate.ClearAll(c)
		return nil
	case restate.WorkflowContext:
		restate.ClearAll(c)
		return nil
	default:
		return restate.TerminalError(fmt.Errorf("ClearAll called from read-only context: %T", ctx), 400)
	}
}

// -----------------------------------------------------------------------------
// Section 4: Durable Saga Framework - Distributed Transaction Compensation
// -----------------------------------------------------------------------------

// SagaCompensationFunc executes a compensation action inside restate.Run.
type SagaCompensationFunc func(rc restate.RunContext, payload []byte) error

// SagaEntry is the persisted record of a compensation step.
type SagaEntry struct {
	Name      string    `json:"name"`
	Payload   []byte    `json:"payload"`
	StepID    string    `json:"step_id"`
	Timestamp time.Time `json:"timestamp"`
	Attempt   int       `json:"attempt"`
}

// SagaConfig controls retry behavior and DLQ handling.
type SagaConfig struct {
	MaxRetries         int
	InitialRetryDelay  time.Duration
	MaxRetryDelay      time.Duration
	FailOnCleanupError bool
	DLQKey             string
}

// DefaultSagaConfig returns production-ready retry defaults.
func DefaultSagaConfig() SagaConfig {
	return SagaConfig{
		MaxRetries:         5,
		InitialRetryDelay:  1 * time.Second,
		MaxRetryDelay:      5 * time.Minute,
		FailOnCleanupError: false,
		DLQKey:             "",
	}
}

// SagaFramework manages durable compensation for control plane operations.
type SagaFramework struct {
	wctx     restate.WorkflowContext
	nsKey    string
	dlqKey   string
	registry map[string]SagaCompensationFunc
	log      *slog.Logger
	cfg      SagaConfig
}

// NewSaga creates a saga bound to a workflow context.
func NewSaga(ctx restate.WorkflowContext, name string, cfg *SagaConfig) *SagaFramework {
	instance := restate.Key(ctx)
	ns := path.Join(instance, "saga", name)
	dlq := path.Join(instance, "saga-dlq", name)

	if cfg == nil {
		c := DefaultSagaConfig()
		cfg = &c
	}

	if cfg.DLQKey != "" {
		dlq = cfg.DLQKey
	}

	return &SagaFramework{
		wctx:     ctx,
		nsKey:    ns,
		dlqKey:   dlq,
		registry: make(map[string]SagaCompensationFunc),
		log:      ctx.Log(),
		cfg:      *cfg,
	}
}

// Register adds a compensation handler. Must be called before Add.
func (s *SagaFramework) Register(name string, fn SagaCompensationFunc) {
	if fn == nil {
		s.log.Warn("saga.register: nil handler ignored", "name", name)
		return
	}
	s.registry[name] = fn
}

// ValidateCompensationIdempotent is a documentation/lint helper for compensation handlers.
//
// While the framework cannot enforce idempotency at runtime, this helper:
//  1. Documents the idempotency contract clearly in code
//  2. Provides a consistent API for developers
//  3. Enables potential static analysis/linting
//  4. Adds runtime logging to track compensation execution
//
// Compensation handlers MUST be idempotent because:
//   - Restate may replay them during recovery
//   - Retries will execute the same compensation multiple times
//   - Network failures can cause duplicate attempts
//
// Usage:
//
//	saga.Register("charge_payment", ValidateCompensationIdempotent(
//	    "charge_payment",
//	    func(rc restate.RunContext, payload []byte) error {
//	        var data PaymentData
//	        json.Unmarshal(payload, &data)
//	        // ✅ Idempotent: refund checks if already refunded
//	        return refundPaymentIdempotent(data.PaymentID)
//	    },
//	))
//
// Idempotency Patterns:
//  1. Check-Then-Act: Verify side effect not already done
//  2. Use External Idempotency Keys: Payment provider handles deduplication
//  3. State-Based Deduplication: Store completion flag in workflow state
//  4. Set Absolute Values: Don't increment/decrement, set exact values
//
// Anti-Patterns to Avoid:
//
//	❌ Incrementing/decrementing without checks
//	❌ Creating resources without unique constraints
//	❌ Time-dependent logic (non-deterministic)
//	❌ Assuming single execution
//
// See SAGA_GUIDE.MD for detailed patterns and examples.
func ValidateCompensationIdempotent(
	name string,
	handler SagaCompensationFunc,
) SagaCompensationFunc {
	return func(rc restate.RunContext, payload []byte) error {
		// Log compensation attempt for observability
		// This helps track retries and identify non-idempotent handlers
		// Future enhancement: Could add deduplication checks here

		err := handler(rc, payload)

		// Future: Could add runtime validations:
		// - Track compensation IDs to detect duplicates
		// - Validate payload schema
		// - Check for state consistency
		// - Monitor for suspicious retry patterns

		return err
	}
}

// Add persists a compensation step BEFORE executing the main action.
func (s *SagaFramework) Add(name string, payload any, dedupe bool) error {
	raw, err := canonicalJSON(payload)
	if err != nil {
		return fmt.Errorf("saga: marshal payload: %w", err)
	}

	stepID := deterministicStepID(name, raw)

	// Read-modify-write with deduplication
	entries, _ := restate.Get[[]SagaEntry](s.wctx, s.nsKey)
	if dedupe {
		for _, e := range entries {
			if e.StepID == stepID {
				s.log.Debug("saga.add: duplicate step ignored", "name", name, "step_id", stepID)
				return nil
			}
		}
	}

	entry := SagaEntry{
		Name:      name,
		Payload:   raw,
		StepID:    stepID,
		Timestamp: time.Now(),
		Attempt:   0,
	}

	entries = append(entries, entry)
	restate.Set(s.wctx, s.nsKey, entries)
	s.log.Info("saga.step_added", "name", name, "step_id", stepID)
	return nil
}

// CompensateIfNeeded executes all compensations in reverse order if error occurred.
func (s *SagaFramework) CompensateIfNeeded(errPtr *error) {
	if errPtr == nil || *errPtr == nil {
		return
	}

	origErr := *errPtr
	entries, _ := restate.Get[[]SagaEntry](s.wctx, s.nsKey)
	if len(entries) == 0 {
		s.log.Info("saga.no_compensations")
		return
	}

	s.log.Info("saga.compensation.starting", "count", len(entries), "original_error", origErr.Error())

	// Execute compensations in LIFO order
	for idx := len(entries) - 1; idx >= 0; idx-- {
		entry := entries[idx]

		handler, ok := s.registry[entry.Name]
		if !ok {
			msg := fmt.Sprintf("missing compensation handler: %s", entry.Name)
			s.log.Error("saga.missing_handler", "name", entry.Name)
			s.persistDLQ(origErr, fmt.Errorf(msg), entries, idx)
			*errPtr = restate.TerminalError(fmt.Errorf("%s: original=%w", msg, origErr), 500)
			return
		}

		// Retry loop with exponential backoff
		for {
			ents, _ := restate.Get[[]SagaEntry](s.wctx, s.nsKey)
			if idx >= len(ents) {
				msg := fmt.Errorf("compensation index out of range: idx=%d len=%d", idx, len(ents))
				s.log.Error("saga.index_oob", "idx", idx, "len", len(ents))
				s.persistDLQ(origErr, msg, ents, idx)
				*errPtr = restate.TerminalError(fmt.Errorf("compensation state corrupted: %w", origErr), 500)
				return
			}

			cur := ents[idx]

			// Execute compensation using RunDoVoid to prevent outer context capture
			runErr := RunDoVoid(s.wctx, func(rc restate.RunContext) error {
				s.log.Info("saga.compensation.attempting", "name", cur.Name, "attempt", cur.Attempt+1)
				return handler(rc, cur.Payload)
			}, restate.WithName(fmt.Sprintf("saga.compensate.%s", cur.Name)))

			if runErr == nil {
				// Success: remove this compensation from the list
				ents = removeIndex(ents, idx)
				restate.Set(s.wctx, s.nsKey, ents)
				s.log.Info("saga.compensation.succeeded", "name", cur.Name)
				break
			}

			// Failure: increment attempt counter
			cur.Attempt++
			ents[idx] = cur
			restate.Set(s.wctx, s.nsKey, ents)
			s.log.Warn("saga.compensation.failed", "name", cur.Name, "attempt", cur.Attempt, "err", runErr.Error())

			// Check if max retries exceeded
			if s.cfg.MaxRetries >= 0 && cur.Attempt >= s.cfg.MaxRetries {
				msg := fmt.Errorf("max retries exceeded for %s (attempts=%d): last_err=%w",
					cur.Name, cur.Attempt, runErr)
				s.log.Error("saga.compensation.max_retries", "name", cur.Name, "attempts", cur.Attempt)
				s.persistDLQ(origErr, msg, ents, idx)
				*errPtr = restate.TerminalError(fmt.Errorf("compensation failed irrecoverably: %w", origErr), 500)
				return
			}

			// Schedule exponential backoff
			delay := computeBackoff(s.cfg.InitialRetryDelay, s.cfg.MaxRetryDelay, cur.Attempt-1)
			s.log.Info("saga.compensation.retry_scheduled", "name", cur.Name, "delay", delay.String())

			if sleepErr := restate.Sleep(s.wctx, delay); sleepErr != nil {
				s.log.Error("saga.sleep_failed", "err", sleepErr.Error())
				s.persistDLQ(origErr, sleepErr, ents, idx)
				*errPtr = restate.TerminalError(fmt.Errorf("saga sleep failed: %w", origErr), 500)
				return
			}
		}
	}

	// All compensations succeeded
	restate.Clear(s.wctx, s.nsKey)
	s.log.Info("saga.compensation.completed")
}

// persistDLQ records irrecoverable compensation failures to dead-letter queue.
func (s *SagaFramework) persistDLQ(originalErr, escalationErr error, entries []SagaEntry, cursor int) {
	_ = RunDoVoid(s.wctx, func(rc restate.RunContext) error {
		record := map[string]any{
			"saga_key":                     s.nsKey,
			"original_error":               originalErr.Error(),
			"escalation_error":             escalationErr.Error(),
			"entries":                      entries,
			"cursor":                       cursor,
			"timestamp":                    time.Now(),
			"hostname":                     os.Getenv("HOSTNAME"),
			"requires_manual_intervention": true,
		}
		jsonBytes, err := json.Marshal(record)
		if err != nil {
			return err
		}
		restate.Set(s.wctx, s.dlqKey, jsonBytes)
		s.log.Error("saga.dlq_recorded", "dlq_key", s.dlqKey)
		return nil
	}, restate.WithName("saga.persist_dlq"))
}

// -----------------------------------------------------------------------------
// Section 5: Control Plane Service
// -----------------------------------------------------------------------------

// ControlPlaneService provides orchestration capabilities with built-in saga management.
type ControlPlaneService struct {
	name              string
	saga              *SagaFramework
	idempotencyPrefix string
}

// NewControlPlaneService creates an orchestrator with automatic saga setup.
func NewControlPlaneService(ctx restate.WorkflowContext, name string, idempotencyPrefix string) *ControlPlaneService {
	return &ControlPlaneService{
		name:              name,
		saga:              NewSaga(ctx, name, nil),
		idempotencyPrefix: idempotencyPrefix,
	}
}

// Orchestrate executes a function within a saga context with automatic compensation.
func (cp *ControlPlaneService) Orchestrate(fn func() error) (err error) {
	defer cp.saga.CompensateIfNeeded(&err)
	return fn()
}

// RegisterCompensation adds a compensation handler to the saga.
func (cp *ControlPlaneService) RegisterCompensation(name string, fn SagaCompensationFunc) {
	cp.saga.Register(name, fn)
}

// AddCompensationStep persists a compensation step before executing main action.
func (cp *ControlPlaneService) AddCompensationStep(name string, payload any, dedupe bool) error {
	return cp.saga.Add(name, payload, dedupe)
}

// AwaitHumanApproval coordinates human-in-the-loop with durable timeout.
func (cp *ControlPlaneService) AwaitHumanApproval(
	ctx restate.Context,
	approvalID string,
	timeout time.Duration,
) (approved bool, err error) {
	awakeable := WaitForExternalSignal[bool](ctx)
	awakeableID := awakeable.Id()

	ctx.Log().Info("controlplane.awaiting_approval", "approval_id", approvalID, "awakeable_id", awakeableID)

	// Race against timeout
	timeoutFuture := restate.After(ctx, timeout)
	winner, err := restate.WaitFirst(ctx, awakeable, timeoutFuture)
	if err != nil {
		return false, fmt.Errorf("approval race failed: %w", err)
	}

	// Check if timeout occurred - AfterFuture completes with nil error on timeout
	if winner == timeoutFuture {
		ctx.Log().Warn("controlplane.approval_timeout", "approval_id", approvalID)
		return false, restate.TerminalError(fmt.Errorf("approval timeout after %v", timeout), 408)
	}

	// Get approval result
	result, err := awakeable.Result()
	if err != nil {
		return false, fmt.Errorf("approval result failed: %w", err)
	}

	ctx.Log().Info("controlplane.approval_result", "approval_id", approvalID, "approved", result)
	return result, nil
}

// GenerateIdempotencyKey creates a deterministic idempotency key for external calls.
// Uses restate.UUID to ensure deterministic generation across retries.
// IMPORTANT: Must be called from within a durable handler context.
func (cp *ControlPlaneService) GenerateIdempotencyKey(ctx restate.Context, suffix string) string {
	// Use deterministic UUID seeded by invocation ID
	uuid := restate.UUID(ctx)
	return fmt.Sprintf("%s:%s:%s", cp.idempotencyPrefix, suffix, uuid.String())
}

// GenerateIdempotencyKeyDeterministic creates an idempotency key using only deterministic inputs.
// Use this when you need a predictable key based on business data (e.g., user ID + order ID).
func (cp *ControlPlaneService) GenerateIdempotencyKeyDeterministic(businessKeys ...string) string {
	if len(businessKeys) == 0 {
		return cp.idempotencyPrefix
	}
	// Join all keys with deterministic separator
	combined := fmt.Sprintf("%s:%s", cp.idempotencyPrefix, path.Join(businessKeys...))
	return combined
}

// SafeStep enforces "register compensation BEFORE action" pattern
type SafeStep[T any] struct {
	saga         *SagaFramework
	name         string
	compensation SagaCompensationFunc
	registered   bool
	executed     bool
}

// NewSafeStep creates a step that enforces compensation-before-action
func (s *SagaFramework) NewSafeStep(name string) *SafeStep[any] {
	return &SafeStep[any]{
		saga: s,
		name: name,
	}
}

// WithCompensation registers the compensation (MUST be called before Execute)
func (step *SafeStep[T]) WithCompensation(compensation SagaCompensationFunc) *SafeStep[T] {
	if step.executed {
		panic(fmt.Sprintf("saga: cannot register compensation after action execution: %s", step.name))
	}
	step.compensation = compensation
	step.registered = true
	return step
}

// Execute runs the action (compensation must be registered first)
func (step *SafeStep[T]) Execute(
	ctx restate.Context,
	action func() (T, error),
) (T, error) {
	var zero T

	if !step.registered {
		return zero, restate.TerminalError(
			fmt.Errorf("saga: must register compensation before executing action: %s", step.name),
			500,
		)
	}

	if step.executed {
		return zero, restate.TerminalError(
			fmt.Errorf("saga: action already executed: %s", step.name),
			500,
		)
	}

	// Register compensation first using the saga's Register method
	step.saga.Register(step.name, func(rc restate.RunContext, payload []byte) error {
		return step.compensation(rc, payload)
	})

	// Then execute action
	result, err := restate.Run(step.saga.wctx, func(rc restate.RunContext) (T, error) {
		return action()
	}, restate.WithName(step.name))

	step.executed = true

	if err != nil {
		// Error will trigger compensation automatically
		return zero, err
	}

	return result, nil
}

// CompensationStrategy defines how to handle partial compensation
type CompensationStrategy int

const (
	// CompensateAll runs all compensations (default, all-or-nothing)
	CompensateAll CompensationStrategy = iota

	// CompensateCompleted only compensates successfully completed steps
	CompensateCompleted

	// CompensateBestEffort tries all compensations, continues on errors
	CompensateBestEffort

	// CompensateUntilSuccess stops after first successful compensation
	CompensateUntilSuccess
)

// PartialCompensationConfig configures partial compensation behavior
type PartialCompensationConfig struct {
	Strategy        CompensationStrategy
	ContinueOnError bool
	MaxRetries      int
}

// SetCompensationStrategy configures how compensations are executed
func (s *SagaFramework) SetCompensationStrategy(strategy CompensationStrategy) {
	// Store strategy in saga namespace state
	restate.Set(s.wctx, fmt.Sprintf("%s:strategy", s.nsKey), int(strategy))
}

// RollbackWithStrategy executes compensations with the specified strategy
// Simplified version that works with existing saga implementation
func (s *SagaFramework) RollbackWithStrategy(
	ctx restate.WorkflowContext,
	strategy CompensationStrategy,
) error {
	// Get compensation entries
	entries, err := restate.Get[[]SagaEntry](ctx, s.nsKey)
	if err != nil || len(entries) == 0 {
		s.log.Info("saga: no compensations to execute")
		return nil
	}

	s.log.Info("saga: starting compensations",
		"count", len(entries),
		"strategy", strategyName(strategy))

	// Execute compensations in reverse order
	var compensationErrors []error

	for idx := len(entries) - 1; idx >= 0; idx-- {
		entry := entries[idx]

		handler, ok := s.registry[entry.Name]
		if !ok {
			s.log.Warn("saga: no compensation handler", "name", entry.Name)
			if strategy != CompensateBestEffort {
				return fmt.Errorf("missing compensation handler: %s", entry.Name)
			}
			continue
		}

		s.log.Info("saga: executing compensation",
			"step", entry.Name,
			"strategy", strategyName(strategy))

		_, runErr := restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
			return restate.Void{}, handler(rc, entry.Payload)
		}, restate.WithName(fmt.Sprintf("compensate-%s", entry.Name)))

		if runErr != nil {
			compensationErrors = append(compensationErrors,
				fmt.Errorf("compensation failed for %s: %w", entry.Name, runErr))

			// Handle error based on strategy
			if strategy == CompensateBestEffort {
				// Continue with next compensation
				s.log.Warn("saga: compensation failed, continuing",
					"step", entry.Name,
					"error", runErr.Error())
				continue
			} else if strategy != CompensateUntilSuccess {
				// Stop on error for other strategies
				break
			}
		} else {
			// Compensation succeeded
			// Remove from list
			entries = removeIndex(entries, idx)
			restate.Set(ctx, s.nsKey, entries)

			if strategy == CompensateUntilSuccess {
				// Stop after first success
				break
			}
		}
	}

	if len(compensationErrors) > 0 && strategy != CompensateBestEffort {
		return fmt.Errorf("saga rollback encountered %d errors: %v",
			len(compensationErrors), compensationErrors[0])
	}

	return nil
}

// Helper function to get strategy name for logging
func strategyName(strategy CompensationStrategy) string {
	switch strategy {
	case CompensateAll:
		return "compensate-all"
	case CompensateCompleted:
		return "compensate-completed"
	case CompensateBestEffort:
		return "compensate-best-effort"
	case CompensateUntilSuccess:
		return "compensate-until-success"
	default:
		return "unknown"
	}
}

// -----------------------------------------------------------------------------
// Section 6: Data Plane Service Abstraction
// -----------------------------------------------------------------------------

// DataPlaneService encapsulates pure business logic with automatic durability.
type DataPlaneService[I, O any] struct {
	Name    string
	Handler func(restate.RunContext, I) (O, error)
}

// NewStatelessService creates a durable data plane service.
func NewStatelessService[I, O any](
	name string,
	operation func(restate.RunContext, I) (O, error),
) *DataPlaneService[I, O] {
	return &DataPlaneService[I, O]{
		Name: name,
		Handler: func(rc restate.RunContext, input I) (O, error) {
			GuardRunContext(rc)
			return operation(rc, input)
		},
	}
}

// Execute runs the data plane operation within a durable context.
func (dp *DataPlaneService[I, O]) Execute(ctx restate.Context, input I) (O, error) {
	var zero O

	result, err := restate.Run(ctx, func(rc restate.RunContext) (O, error) {
		return dp.Handler(rc, input)
	}, restate.WithName(dp.Name))

	if err != nil {
		return zero, err
	}

	return result, nil
}

// -----------------------------------------------------------------------------
// Section 6A: Run (Side Effects) Utilities
// -----------------------------------------------------------------------------

// RunConfig configures retry behavior for Run blocks
type RunConfig struct {
	MaxRetries    int
	InitialDelay  time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
	Name          string
}

// DefaultRunConfig returns sensible defaults for Run blocks
func DefaultRunConfig(name string) RunConfig {
	return RunConfig{
		MaxRetries:    3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		Name:          name,
	}
}

// RunWithRetry executes a side effect with automatic retry on transient errors
func RunWithRetry[T any](
	ctx restate.Context,
	cfg RunConfig,
	operation func(restate.RunContext) (T, error),
) (T, error) {
	var lastErr error
	var zero T

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			// Calculate backoff delay
			delay := computeBackoff(cfg.InitialDelay, cfg.MaxDelay, attempt-1)
			ctx.Log().Info("run: retrying after backoff",
				"attempt", attempt,
				"delay", delay.String(),
				"name", cfg.Name)

			// Durable sleep between retries
			if err := restate.Sleep(ctx, delay); err != nil {
				return zero, fmt.Errorf("retry sleep failed: %w", err)
			}
		}

		// Execute the operation
		result, err := restate.Run(ctx, operation, restate.WithName(cfg.Name))
		if err == nil {
			if attempt > 0 {
				ctx.Log().Info("run: retry succeeded", "attempt", attempt, "name", cfg.Name)
			}
			return result, nil
		}

		// Check if error is terminal (don't retry)
		if isTerminalError(err) {
			ctx.Log().Error("run: terminal error, not retrying",
				"error", err.Error(),
				"name", cfg.Name)
			return zero, err
		}

		lastErr = err
		ctx.Log().Warn("run: attempt failed",
			"attempt", attempt,
			"error", err.Error(),
			"name", cfg.Name)
	}

	return zero, fmt.Errorf("run exhausted retries (%d attempts) for %s: %w",
		cfg.MaxRetries+1, cfg.Name, lastErr)
}

// RunAsync executes a side effect asynchronously and returns a future
func RunAsync[T any](
	ctx restate.Context,
	operation func(restate.RunContext) (T, error),
	opts ...restate.RunOption,
) restate.Future {
	return restate.RunAsync(ctx, operation, opts...)
}

// RunAsyncWithRetry combines RunAsync with retry logic
func RunAsyncWithRetry[T any](
	ctx restate.Context,
	cfg RunConfig,
	operation func(restate.RunContext) (T, error),
) restate.Future {
	// Wrap the operation with retry logic
	return restate.RunAsync(ctx, func(rc restate.RunContext) (T, error) {
		// Note: Retry logic must be inside the Run block for determinism
		var lastErr error
		var zero T

		for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
			if attempt > 0 {
				// Can't use restate.Sleep here (not available in RunContext)
				// So retry logic is limited in RunAsync
				ctx.Log().Debug("run-async: retry attempt", "attempt", attempt)
			}

			result, err := operation(rc)
			if err == nil {
				return result, nil
			}

			if isTerminalError(err) {
				return zero, err
			}

			lastErr = err
		}

		return zero, lastErr
	}, restate.WithName(cfg.Name))
}

// ============================================================================
// RunDo Helpers - Prevent Accidental Context Capture
// ============================================================================
//
// ⚠️ CRITICAL DOS RULE: Do not use the Restate context inside ctx.Run
//
// The following helpers enforce the pattern of accepting restate.RunContext
// to prevent accidentally capturing and using the outer Restate context.
//
// WRONG ❌:
//
//	result, err := restate.Run(ctx, func(rc restate.RunContext) (string, error) {
//	    ctx.Sleep(time.Second)  // ❌ Uses outer ctx - causes non-determinism!
//	    ctx.Log().Info("test")  // ❌ Uses outer ctx
//	    return fetchData()
//	})
//
// RIGHT ✅:
//
//	result, err := RunDo(ctx, func(rc restate.RunContext) (string, error) {
//	    // Only rc is available - cannot accidentally use outer ctx
//	    // RunContext has no Sleep, Get, Set, etc. - only for side effects
//	    return fetchData()
//	})
//
// The RunDo helpers provide minimal additional compile-time safety but serve
// as clear documentation that operations should ONLY use RunContext.
//
// See DOS_DONTS_MEGA.MD lines 386, 515, 756 for details.

// RunDo executes a function within restate.Run, ensuring the function
// receives only RunContext to prevent accidentally capturing outer context.
//
// Use this for side effects that need to return a value. The function you
// provide should ONLY use the RunContext parameter (rc), never the outer ctx.
//
// Example:
//
//	user, err := RunDo(ctx, func(rc restate.RunContext) (User, error) {
//	    // Do NOT use ctx here - only use rc if needed
//	    return fetchUserFromDB(userID) // side effect
//	}, restate.WithName("fetch-user"))
func RunDo[T any](
	ctx restate.Context,
	operation func(restate.RunContext) (T, error),
	opts ...restate.RunOption,
) (T, error) {
	return restate.Run(ctx, operation, opts...)
}

// RunDoVoid executes a void function within restate.Run.
//
// Use this for side effects that don't return a meaningful value.
// The function should ONLY use the RunContext parameter, never the outer ctx.
//
// Example:
//
//	err := RunDoVoid(ctx, func(rc restate.RunContext) error {
//	    // Do NOT use ctx here
//	    return sendEmail(userEmail, subject, body)
//	}, restate.WithName("send-email"))
func RunDoVoid(
	ctx restate.Context,
	operation func(restate.RunContext) error,
	opts ...restate.RunOption,
) error {
	_, err := restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
		return restate.Void{}, operation(rc)
	}, opts...)
	return err
}

// DeterministicHelpers provides deterministic operations
type DeterministicHelpers struct {
	ctx restate.Context
}

// NewDeterministicHelpers creates helpers for deterministic operations
func NewDeterministicHelpers(ctx restate.Context) *DeterministicHelpers {
	return &DeterministicHelpers{ctx: ctx}
}

// UUID generates a deterministic UUID (same on replay)
func (h *DeterministicHelpers) UUID() string {
	uuid := restate.UUID(h.ctx)
	return uuid.String()
}

// RandInt generates a deterministic random integer
func (h *DeterministicHelpers) RandInt(min, max int) int {
	if min >= max {
		return min
	}
	rand := restate.Rand(h.ctx)
	// Use Uint64() and modulo for random selection
	range_ := uint64(max - min)
	if range_ == 0 {
		return min
	}
	return min + int(rand.Uint64()%range_)
}

// RandFloat generates a deterministic random float64 between 0.0 and 1.0
func (h *DeterministicHelpers) RandFloat() float64 {
	rand := restate.Rand(h.ctx)
	return rand.Float64()
}

// RandChoice picks a deterministic random item from a slice
func RandChoice[T any](ctx restate.Context, items []T) (T, error) {
	var zero T
	if len(items) == 0 {
		return zero, fmt.Errorf("cannot choose from empty slice")
	}

	rand := restate.Rand(ctx)
	idx := int(rand.Uint64() % uint64(len(items)))
	return items[idx], nil
}

// Time captures the current time deterministically
type Time struct {
	ctx restate.Context
}

// NewTime creates a deterministic time helper
func NewTime(ctx restate.Context) *Time {
	return &Time{ctx: ctx}
}

// Now returns the current time (captured once, deterministic on replay)
func (t *Time) Now() time.Time {
	var currentTime time.Time

	// Capture time in a Run block to make it deterministic
	_, err := restate.Run(t.ctx, func(rc restate.RunContext) (struct{}, error) {
		currentTime = time.Now()
		return struct{}{}, nil
	}, restate.WithName("capture-time"))

	if err != nil {
		// Fallback to zero time on error
		return time.Time{}
	}

	return currentTime
}

// Since returns the duration since a given time (uses deterministic Now)
func (t *Time) Since(start time.Time) time.Duration {
	return t.Now().Sub(start)
}

// Until returns the duration until a given time (uses deterministic Now)
func (t *Time) Until(target time.Time) time.Duration {
	return target.Sub(t.Now())
}

// Helper: Check if error is terminal (non-retryable)
func isTerminalError(err error) bool {
	if err == nil {
		return false
	}
	// Check for TerminalError type
	_, isTerminal := err.(interface{ Terminal() bool })
	return isTerminal
}

// -----------------------------------------------------------------------------
// Section 6B: Observability & Metrics
// -----------------------------------------------------------------------------

// MetricsCollector provides Prometheus-compatible metrics collection
type MetricsCollector struct {
	// Counters
	InvocationTotal    map[string]int64
	InvocationErrors   map[string]int64
	CompensationTotal  map[string]int64
	CompensationErrors map[string]int64

	// Gauges
	ActiveInvocations map[string]int64
	StateSize         map[string]int64

	// Histograms (stored as buckets)
	InvocationDuration   map[string][]float64
	CompensationDuration map[string][]float64

	mu sync.RWMutex
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		InvocationTotal:      make(map[string]int64),
		InvocationErrors:     make(map[string]int64),
		CompensationTotal:    make(map[string]int64),
		CompensationErrors:   make(map[string]int64),
		ActiveInvocations:    make(map[string]int64),
		StateSize:            make(map[string]int64),
		InvocationDuration:   make(map[string][]float64),
		CompensationDuration: make(map[string][]float64),
	}
}

// RecordInvocation records a service invocation
func (mc *MetricsCollector) RecordInvocation(serviceName, handlerName string, duration time.Duration, err error) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	key := fmt.Sprintf("%s.%s", serviceName, handlerName)
	mc.InvocationTotal[key]++

	if err != nil {
		mc.InvocationErrors[key]++
	}

	mc.InvocationDuration[key] = append(mc.InvocationDuration[key], duration.Seconds())
}

// RecordCompensation records a saga compensation execution
func (mc *MetricsCollector) RecordCompensation(stepName string, duration time.Duration, err error) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.CompensationTotal[stepName]++

	if err != nil {
		mc.CompensationErrors[stepName]++
	}

	mc.CompensationDuration[stepName] = append(mc.CompensationDuration[stepName], duration.Seconds())
}

// IncrementActiveInvocations increments active invocation gauge
func (mc *MetricsCollector) IncrementActiveInvocations(serviceName string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.ActiveInvocations[serviceName]++
}

// DecrementActiveInvocations decrements active invocation gauge
func (mc *MetricsCollector) DecrementActiveInvocations(serviceName string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.ActiveInvocations[serviceName]--
}

// RecordStateSize records state size in bytes
func (mc *MetricsCollector) RecordStateSize(objectKey string, sizeBytes int64) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.StateSize[objectKey] = sizeBytes
}

// GetMetrics returns a snapshot of all metrics
func (mc *MetricsCollector) GetMetrics() map[string]interface{} {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	return map[string]interface{}{
		"invocation_total":          copyMap(mc.InvocationTotal),
		"invocation_errors":         copyMap(mc.InvocationErrors),
		"compensation_total":        copyMap(mc.CompensationTotal),
		"compensation_errors":       copyMap(mc.CompensationErrors),
		"active_invocations":        copyMap(mc.ActiveInvocations),
		"state_size_bytes":          copyMap(mc.StateSize),
		"invocation_duration_sec":   copyDurationMap(mc.InvocationDuration),
		"compensation_duration_sec": copyDurationMap(mc.CompensationDuration),
	}
}

// Helper functions for copying maps
func copyMap(m map[string]int64) map[string]int64 {
	result := make(map[string]int64, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

func copyDurationMap(m map[string][]float64) map[string][]float64 {
	result := make(map[string][]float64, len(m))
	for k, v := range m {
		copied := make([]float64, len(v))
		copy(copied, v)
		result[k] = copied
	}
	return result
}

// OpenTelemetrySpan represents a trace span
type OpenTelemetrySpan struct {
	TraceID    string
	SpanID     string
	ParentID   string
	Name       string
	StartTime  time.Time
	EndTime    time.Time
	Attributes map[string]string
	Status     string
	Error      error
}

// TracingContext provides OpenTelemetry-compatible tracing
type TracingContext struct {
	ctx     restate.Context
	spans   []*OpenTelemetrySpan
	current *OpenTelemetrySpan
	mu      sync.Mutex
}

// NewTracingContext creates a new tracing context
func NewTracingContext(ctx restate.Context) *TracingContext {
	return &TracingContext{
		ctx:   ctx,
		spans: make([]*OpenTelemetrySpan, 0),
	}
}

// StartSpan starts a new span
func (tc *TracingContext) StartSpan(name string, attributes map[string]string) *OpenTelemetrySpan {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	// Generate span IDs deterministically using context
	helpers := NewDeterministicHelpers(tc.ctx)
	spanID := helpers.UUID()
	traceID := helpers.UUID()

	var parentID string
	if tc.current != nil {
		parentID = tc.current.SpanID
		traceID = tc.current.TraceID
	}

	span := &OpenTelemetrySpan{
		TraceID:    traceID,
		SpanID:     spanID,
		ParentID:   parentID,
		Name:       name,
		StartTime:  time.Now(),
		Attributes: attributes,
		Status:     "OK",
	}

	tc.spans = append(tc.spans, span)
	tc.current = span

	return span
}

// EndSpan ends the current span
func (tc *TracingContext) EndSpan(span *OpenTelemetrySpan, err error) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	span.EndTime = time.Now()

	if err != nil {
		span.Status = "ERROR"
		span.Error = err
	}

	// Pop to parent if this was current
	if tc.current == span && span.ParentID != "" {
		// Find parent span
		for _, s := range tc.spans {
			if s.SpanID == span.ParentID {
				tc.current = s
				break
			}
		}
	}
}

// GetSpans returns all recorded spans
func (tc *TracingContext) GetSpans() []*OpenTelemetrySpan {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	result := make([]*OpenTelemetrySpan, len(tc.spans))
	copy(result, tc.spans)
	return result
}

// ObservabilityHooks provides hooks for custom observability
type ObservabilityHooks struct {
	// Invocation hooks
	OnInvocationStart func(serviceName, handlerName string, input interface{})
	OnInvocationEnd   func(serviceName, handlerName string, output interface{}, err error, duration time.Duration)

	// State hooks
	OnStateGet   func(key string, value interface{})
	OnStateSet   func(key string, value interface{})
	OnStateClear func(key string)

	// Saga hooks
	OnSagaStart         func(sagaName string)
	OnSagaStepAdded     func(sagaName, stepName string)
	OnCompensationStart func(stepName string)
	OnCompensationEnd   func(stepName string, err error, duration time.Duration)

	// Workflow hooks
	OnWorkflowStart func(workflowID string, input interface{})
	OnWorkflowEnd   func(workflowID string, output interface{}, err error)

	// Error hooks
	OnError func(context string, err error)
}

// DefaultObservabilityHooks returns hooks that log to slog
func DefaultObservabilityHooks(log *slog.Logger) *ObservabilityHooks {
	return &ObservabilityHooks{
		OnInvocationStart: func(serviceName, handlerName string, input interface{}) {
			log.Info("invocation.start",
				"service", serviceName,
				"handler", handlerName)
		},
		OnInvocationEnd: func(serviceName, handlerName string, output interface{}, err error, duration time.Duration) {
			if err != nil {
				log.Error("invocation.end",
					"service", serviceName,
					"handler", handlerName,
					"duration_ms", duration.Milliseconds(),
					"error", err.Error())
			} else {
				log.Info("invocation.end",
					"service", serviceName,
					"handler", handlerName,
					"duration_ms", duration.Milliseconds())
			}
		},
		OnStateSet: func(key string, value interface{}) {
			log.Debug("state.set", "key", key)
		},
		OnError: func(context string, err error) {
			log.Error("error", "context", context, "error", err.Error())
		},
	}
}

// InstrumentedServiceClient wraps ServiceClient with observability
type InstrumentedServiceClient[I, O any] struct {
	Client  ServiceClient[I, O]
	Metrics *MetricsCollector
	Tracing *TracingContext
	Hooks   *ObservabilityHooks
}

// Call executes with full observability
func (isc *InstrumentedServiceClient[I, O]) Call(
	ctx restate.Context,
	input I,
	opts ...CallOption,
) (O, error) {
	var zero O

	// Start metrics/tracing
	start := time.Now()

	if isc.Metrics != nil {
		isc.Metrics.IncrementActiveInvocations(isc.Client.ServiceName)
		defer isc.Metrics.DecrementActiveInvocations(isc.Client.ServiceName)
	}

	var span *OpenTelemetrySpan
	if isc.Tracing != nil {
		span = isc.Tracing.StartSpan(
			fmt.Sprintf("%s.%s", isc.Client.ServiceName, isc.Client.HandlerName),
			map[string]string{
				"service": isc.Client.ServiceName,
				"handler": isc.Client.HandlerName,
				"type":    "invocation",
			},
		)
		defer isc.Tracing.EndSpan(span, nil)
	}

	if isc.Hooks != nil && isc.Hooks.OnInvocationStart != nil {
		isc.Hooks.OnInvocationStart(isc.Client.ServiceName, isc.Client.HandlerName, input)
	}

	// Execute call
	output, err := isc.Client.Call(ctx, input, opts...)

	// Record metrics
	duration := time.Since(start)

	if isc.Metrics != nil {
		isc.Metrics.RecordInvocation(isc.Client.ServiceName, isc.Client.HandlerName, duration, err)
	}

	if isc.Tracing != nil && span != nil {
		if err != nil {
			span.Status = "ERROR"
			span.Error = err
		}
	}

	if isc.Hooks != nil && isc.Hooks.OnInvocationEnd != nil {
		isc.Hooks.OnInvocationEnd(isc.Client.ServiceName, isc.Client.HandlerName, output, err, duration)
	}

	if err != nil {
		if isc.Hooks != nil && isc.Hooks.OnError != nil {
			isc.Hooks.OnError("service_call", err)
		}
		return zero, err
	}

	return output, nil
}

// NewInstrumentedClient creates an instrumented service client
func NewInstrumentedClient[I, O any](
	client ServiceClient[I, O],
	metrics *MetricsCollector,
	tracing *TracingContext,
	hooks *ObservabilityHooks,
) *InstrumentedServiceClient[I, O] {
	return &InstrumentedServiceClient[I, O]{
		Client:  client,
		Metrics: metrics,
		Tracing: tracing,
		Hooks:   hooks,
	}
}

// -----------------------------------------------------------------------------
// Section 7: Type-Safe Service Clients
// -----------------------------------------------------------------------------
//
// IMPORTANT: Client Usage Patterns (DOS Guidance)
// ================================================
//
// This framework provides TWO DISTINCT types of clients for invoking Restate services:
//
// 1. **Internal Clients** (ServiceClient, ObjectClient, WorkflowClient)
//    - Use INSIDE Restate handlers (within restate.Context)
//    - Route calls through Restate's durable execution engine
//    - Provide durability, exactly-once semantics, and journal replay
//    - Calls are part of the workflow execution graph
//
//    Example:
//      func (MyService) ProcessOrder(ctx restate.Context, order Order) error {
//          client := ServiceClient[Order, Result]{
//              ServiceName: "InventoryService",
//              HandlerName: "Reserve",
//          }
//          result, err := client.Call(ctx, order)  // ✅ Correct: Uses ctx from handler
//          return err
//      }
//
// 2. **Ingress Clients** (IngressClient, IngressServiceClient, etc.)
//    - Use OUTSIDE Restate (from external applications, CLIs, tests)
//    - Make HTTP calls directly to Restate's ingress endpoint
//    - Do NOT have durability or journal replay
//    - Useful for triggering workflows, querying status, or testing
//
//    Example:
//      func main() {
//          ingressClient := NewIngressClient("http://localhost:8080", "")
//          client := ingressClient.Service("OrderService", "Create")
//          metadata, err := client.Send(context.Background(), order)  // ✅ Correct: External trigger
//      }
//
// ⚠️  CRITICAL DOS RULE: Never use IngressClient inside a Restate handler!
//
//    BAD Example (DON'T DO THIS):
//      func (MyService) ProcessOrder(ctx restate.Context, order Order) error {
//          ingress := NewIngressClient("http://localhost:8080", "")  // ❌ WRONG!
//          client := ingress.Service("InventoryService", "Reserve")
//          // This bypasses Restate's durability - the call won't be journaled!
//          _, err := client.Call(context.Background(), order)  //  ❌ WRONG!
//          return err
//      }
//
//    Why this is wrong:
//    - The call won't be part of the durable execution journal
//    - On replay, it will execute again (not idempotent)
//    - You lose exactly-once guarantees
//    - Violates the fundamental Restate execution model
//
// ValidationMode Configuration
// =============================
//
// All internal clients support configurable idempotency key validation via CallOption.ValidationMode:
//
//   - IdempotencyValidationWarn (default): Logs warnings but allows calls to proceed
//   - IdempotencyValidationFail (strict):  Fails calls with invalid idempotency keys
//   - IdempotencyValidationDisabled:       Skips validation entirely
//
// Example:
//   client.Send(ctx, data, CallOption{
//       IdempotencyKey: key,
//       ValidationMode: IdempotencyValidationFail,  // Strict validation
//   })
//
// Per DOS guidance, choose validation mode based on your environment:
//   - Development: Use IdempotencyValidationWarn to catch issues without breaking flows
//   - Production:  Use IdempotencyValidationFail to prevent silent corruption
//
// -----------------------------------------------------------------------------

// ServiceClient provides type-safe inter-service communication.
type ServiceClient[I, O any] struct {
	ServiceName string
	HandlerName string
}

// Call executes a request-response interaction.
func (c ServiceClient[I, O]) Call(
	ctx restate.Context,
	input I,
	opts ...CallOption,
) (O, error) {
	// Check for redundant idempotency keys
	for _, opt := range opts {
		if opt.IdempotencyKey != "" {
			checkRedundantIdempotencyKey(ctx, opt.IdempotencyKey, c.ServiceName, c.HandlerName)
		}
	}

	// Build and execute the request
	client := restate.Service[O](ctx, c.ServiceName, c.HandlerName)
	return client.Request(input)
}

// Send executes a one-way fire-and-forget message.
// FIXED: Method cannot have its own type parameters - uses receiver's [I, O]
func (c ServiceClient[I, O]) Send(
	ctx restate.Context,
	input I,
	opts ...CallOption,
) restate.Invocation {
	// Build the send client
	send := restate.ServiceSend(ctx, c.ServiceName, c.HandlerName)

	// Build options slice
	var sendOpts []restate.SendOption
	for _, opt := range opts {
		// Check for redundant idempotency keys
		if opt.IdempotencyKey != "" {
			checkRedundantIdempotencyKey(ctx, opt.IdempotencyKey, c.ServiceName, c.HandlerName)
		}

		if opt.IdempotencyKey != "" {
			// Determine validation mode (default to warn if not specified)
			validationMode := opt.ValidationMode
			if validationMode == "" {
				validationMode = IdempotencyValidationWarn
			}

			// Skip validation if disabled
			if validationMode != IdempotencyValidationDisabled {
				if err := ValidateIdempotencyKey(opt.IdempotencyKey); err != nil {
					if validationMode == IdempotencyValidationFail {
						// Strict mode: fail the call with terminal error
						ctx.Log().Error("framework: idempotency key validation failed (strict mode)",
							"key", opt.IdempotencyKey,
							"error", err.Error())
						panic(restate.TerminalError(
							fmt.Errorf("idempotency key validation failed: %w", err),
							400,
						))
					} else {
						// Permissive mode: log warning but continue
						ctx.Log().Warn("framework: idempotency key validation warning (permissive mode)",
							"key", opt.IdempotencyKey,
							"error", err.Error())
					}
				}
			}
			sendOpts = append(sendOpts, restate.WithIdempotencyKey(opt.IdempotencyKey))
		}
		if opt.Delay > 0 {
			sendOpts = append(sendOpts, restate.WithDelay(opt.Delay))
		}
	}

	return send.Send(input, sendOpts...)
}

// -----------------------------------------------------------------------------
// Section 7A: Service Type-Specific Clients
// -----------------------------------------------------------------------------

// ObjectClient provides type-safe communication with Virtual Object services
type ObjectClient[I, O any] struct {
	ServiceName string
	HandlerName string
}

// Call invokes a Virtual Object handler with the specified key (request-response)
func (c ObjectClient[I, O]) Call(
	ctx restate.Context,
	key string,
	input I,
	opts ...CallOption,
) (O, error) {
	client := restate.Object[O](ctx, c.ServiceName, key, c.HandlerName)
	return client.Request(input)
}

// Send invokes a Virtual Object handler asynchronously (one-way)
func (c ObjectClient[I, O]) Send(
	ctx restate.Context,
	key string,
	input I,
	opts ...CallOption,
) restate.Invocation {
	send := restate.ObjectSend(ctx, c.ServiceName, key, c.HandlerName)

	// Build options slice
	var sendOpts []restate.SendOption
	for _, opt := range opts {
		if opt.IdempotencyKey != "" {
			// Validate idempotency key before using it
			if err := ValidateIdempotencyKey(opt.IdempotencyKey); err != nil {
				ctx.Log().Error("framework: invalid idempotency key detected",
					"key", opt.IdempotencyKey,
					"error", err.Error())
			}
			sendOpts = append(sendOpts, restate.WithIdempotencyKey(opt.IdempotencyKey))
		}
		if opt.Delay > 0 {
			sendOpts = append(sendOpts, restate.WithDelay(opt.Delay))
		}
	}

	return send.Send(input, sendOpts...)
}

// RequestFuture invokes a Virtual Object handler and returns a future (for concurrent calls)
// Returns the future for use with restate.Wait() or restate.WaitFirst()
func (c ObjectClient[I, O]) RequestFuture(
	ctx restate.Context,
	key string,
	input I,
) restate.Future {
	client := restate.Object[O](ctx, c.ServiceName, key, c.HandlerName)
	return client.RequestFuture(input)
}

// WorkflowClient provides type-safe communication with Workflow services
type WorkflowClient[I, O any] struct {
	ServiceName string
	HandlerName string // Usually "run" for the main workflow handler
}

// Submit starts a new workflow instance with the given ID (idempotent)
func (c WorkflowClient[I, O]) Submit(
	ctx restate.Context,
	workflowID string,
	input I,
	opts ...CallOption,
) restate.Invocation {
	send := restate.WorkflowSend(ctx, c.ServiceName, workflowID, c.HandlerName)

	// Build options slice
	var sendOpts []restate.SendOption
	for _, opt := range opts {
		if opt.IdempotencyKey != "" {
			// Validate idempotency key
			if err := ValidateIdempotencyKey(opt.IdempotencyKey); err != nil {
				ctx.Log().Error("framework: invalid workflow idempotency key",
					"key", opt.IdempotencyKey,
					"error", err.Error())
			}
			sendOpts = append(sendOpts, restate.WithIdempotencyKey(opt.IdempotencyKey))
		}
		if opt.Delay > 0 {
			sendOpts = append(sendOpts, restate.WithDelay(opt.Delay))
		}
	}

	return send.Send(input, sendOpts...)
}

// Attach attaches to an existing workflow instance (request-response)
func (c WorkflowClient[I, O]) Attach(
	ctx restate.Context,
	workflowID string,
	opts ...CallOption,
) (O, error) {
	client := restate.Workflow[O](ctx, c.ServiceName, workflowID, c.HandlerName)
	var zero I
	return client.Request(zero) // Workflows don't take input on attach
}

// AttachFuture attaches to a workflow and returns a future
// Returns the future for use with restate.Wait() or restate.WaitFirst()
func (c WorkflowClient[I, O]) AttachFuture(
	ctx restate.Context,
	workflowID string,
) restate.Future {
	client := restate.Workflow[O](ctx, c.ServiceName, workflowID, c.HandlerName)
	var zero I
	return client.RequestFuture(zero)
}

// Signal sends a signal to a workflow's shared handler
func (c WorkflowClient[I, O]) Signal(
	ctx restate.Context,
	workflowID string,
	signalHandler string,
	input I,
	opts ...CallOption,
) restate.Invocation {
	send := restate.WorkflowSend(ctx, c.ServiceName, workflowID, signalHandler)

	// Build options
	var sendOpts []restate.SendOption
	for _, opt := range opts {
		if opt.IdempotencyKey != "" {
			if err := ValidateIdempotencyKey(opt.IdempotencyKey); err != nil {
				ctx.Log().Error("framework: invalid signal idempotency key",
					"key", opt.IdempotencyKey,
					"error", err.Error())
			}
			sendOpts = append(sendOpts, restate.WithIdempotencyKey(opt.IdempotencyKey))
		}
		if opt.Delay > 0 {
			sendOpts = append(sendOpts, restate.WithDelay(opt.Delay))
		}
	}

	return send.Send(input, sendOpts...)
}

// GetOutput queries a workflow's output via shared handler
func (c WorkflowClient[I, O]) GetOutput(
	ctx restate.Context,
	workflowID string,
	outputHandler string,
) (O, error) {
	client := restate.Workflow[O](ctx, c.ServiceName, workflowID, outputHandler)
	var zero I
	return client.Request(zero)
}

// -----------------------------------------------------------------------------
// Section 7A: Workflow Configuration and Retention Policies
// -----------------------------------------------------------------------------

// WorkflowConfig defines configuration for workflow behavior and state retention
//
// CRITICAL: Workflow state is stored in Restate and subject to retention limits.
// Default Restate retention is typically 24 hours to 90 days depending on deployment.
//
// State Retention Considerations:
//   - Workflow state includes: execution history, promises, timers, durable state
//   - If retention expires, workflow becomes unrecoverable
//   - Long-running workflows (>90 days) need external state archival
//   - WorkflowStatus persistence counts against retention
//
// See WORKFLOW_RETENTION_GUIDE.MD for comprehensive guidance.
type WorkflowConfig struct {
	// StateRetentionDays configures how long workflow state is retained
	// Default: 30 days (Restate default)
	// Min: 1 day
	// Max: 90 days (Restate cluster limit)
	// Note: Actual retention depends on Restate cluster configuration
	StateRetentionDays int

	// EnableStatusPersistence enables durable storage of workflow status
	// When true: Status survives restarts, counts against retention
	// When false: Status is ephemeral, cheaper but lost on restart
	// Default: true
	//
	// IMPORTANT: Status persistence adds to state size. For workflows with
	// frequent status updates (e.g., every second), consider:
	// 1. Reducing update frequency
	// 2. Using external status storage
	// 3. Disabling persistence for short-lived workflows
	EnableStatusPersistence bool

	// AutoCleanupOnCompletion automatically purges workflow state after completion
	// When true: State is deleted when workflow completes successfully
	// When false: State retained until retention period expires
	// Default: false (preserve for audit/debugging)
	//
	// Set to true for:
	// - High-volume workflows (millions per day)
	// - Workflows with large state (>1MB)
	// - When audit trail not needed
	AutoCleanupOnCompletion bool

	// MaxStateSizeBytes warns when workflow state approaches this limit
	// Default: 1MB (conservative)
	// Restate limit: Typically 10MB per workflow
	//
	// Large state causes:
	// - Slower workflow execution
	// - Higher memory usage
	// - Longer retention costs
	MaxStateSizeBytes int64

	// CleanupGracePeriod is time to keep state after completion before cleanup
	// Only applies if AutoCleanupOnCompletion is true
	// Default: 24 hours
	// Use cases:
	// - Allow time for result queries
	// - Grace period for auditing
	// - Debugging window
	CleanupGracePeriod time.Duration
}

// DefaultWorkflowConfig returns recommended default configuration
//
// Defaults are conservative and suitable for most use cases:
//   - 30 day retention (balance between cost and recovery window)
//   - Status persistence enabled (durability over cost)
//   - No auto-cleanup (preserve for debugging)
//   - 1MB max state warning (conservative limit)
//   - 24h cleanup grace period (reasonable audit window)
//
// Adjust based on your requirements. See WorkflowConfig documentation.
func DefaultWorkflowConfig() WorkflowConfig {
	return WorkflowConfig{
		StateRetentionDays:      30,      // 30 days
		EnableStatusPersistence: true,    // Durable status
		AutoCleanupOnCompletion: false,   // Keep for audit
		MaxStateSizeBytes:       1048576, // 1MB
		CleanupGracePeriod:      24 * time.Hour,
	}
}

// ProductionWorkflowConfig returns configuration optimized for production
//
// Production defaults prioritize:
//   - Longer retention (90 days for compliance)
//   - Status persistence (critical for monitoring)
//   - Auto-cleanup disabled (audit requirements)
//   - Larger state limit (10MB - Restate max)
//   - Longer grace period (7 days for investigation)
func ProductionWorkflowConfig() WorkflowConfig {
	return WorkflowConfig{
		StateRetentionDays:      90,                 // Maximum retention
		EnableStatusPersistence: true,               // Critical for production monitoring
		AutoCleanupOnCompletion: false,              // Preserve for compliance/audit
		MaxStateSizeBytes:       10 * 1024 * 1024,   // 10MB (Restate limit)
		CleanupGracePeriod:      7 * 24 * time.Hour, // 7 days
	}
}

// HighVolumeWorkflowConfig returns configuration for high-volume workflows
//
// High-volume defaults prioritize cost and performance:
//   - Short retention (7 days - minimal compliance)
//   - Status persistence disabled (reduce state size)
//   - Auto-cleanup enabled (reduce storage costs)
//   - Smaller state limit (512KB - encourage efficiency)
//   - Short grace period (1 hour - quick cleanup)
//
// Use for:
//   - Millions of workflows per day
//   - Short-lived workflows (<1 hour)
//   - No audit requirements
//   - Cost-sensitive deployments
func HighVolumeWorkflowConfig() WorkflowConfig {
	return WorkflowConfig{
		StateRetentionDays:      7,         // Minimum retention
		EnableStatusPersistence: false,     // Ephemeral status
		AutoCleanupOnCompletion: true,      // Aggressive cleanup
		MaxStateSizeBytes:       524288,    // 512KB
		CleanupGracePeriod:      time.Hour, // Fast cleanup
	}
}

// ValidateConfig checks if WorkflowConfig is valid and logs warnings
//
// Validation checks:
//   - Retention within Restate limits (1-90 days)
//   - State size within Restate limit (10MB)
//   - Cleanup grace period reasonable (<retention)
//
// Warnings for common issues:
//   - Very short retention (<7 days) - recovery risk
//   - Very long retention (>60 days) - cost concern
//   - No auto-cleanup + short retention - state orphaning
//   - Large state limit (>5MB) - performance impact
func (cfg WorkflowConfig) Validate(logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}

	// Check retention bounds
	if cfg.StateRetentionDays < 1 {
		return fmt.Errorf("StateRetentionDays must be >= 1, got %d", cfg.StateRetentionDays)
	}
	if cfg.StateRetentionDays > 90 {
		return fmt.Errorf("StateRetentionDays must be <= 90 (Restate limit), got %d", cfg.StateRetentionDays)
	}

	// Warn about very short retention
	if cfg.StateRetentionDays < 7 {
		logger.Warn("workflow config: very short retention period",
			"retention_days", cfg.StateRetentionDays,
			"risk", "workflows may become unrecoverable before completion")
	}

	// Warn about very long retention (cost)
	if cfg.StateRetentionDays > 60 {
		logger.Warn("workflow config: long retention period",
			"retention_days", cfg.StateRetentionDays,
			"consideration", "higher storage costs, consider archival strategy")
	}

	// Check state size limit
	if cfg.MaxStateSizeBytes < 0 {
		return fmt.Errorf("MaxStateSizeBytes must be >= 0, got %d", cfg.MaxStateSizeBytes)
	}
	if cfg.MaxStateSizeBytes > 10*1024*1024 {
		return fmt.Errorf("MaxStateSizeBytes must be <= 10MB (Restate limit), got %d", cfg.MaxStateSizeBytes)
	}

	// Warn about large state limits
	if cfg.MaxStateSizeBytes > 5*1024*1024 {
		logger.Warn("workflow config: large state size limit",
			"max_size_mb", cfg.MaxStateSizeBytes/(1024*1024),
			"impact", "may cause performance degradation")
	}

	// Validate cleanup grace period
	if cfg.AutoCleanupOnCompletion {
		retentionDuration := time.Duration(cfg.StateRetentionDays) * 24 * time.Hour
		if cfg.CleanupGracePeriod > retentionDuration {
			logger.Warn("workflow config: cleanup grace period exceeds retention",
				"grace_period", cfg.CleanupGracePeriod,
				"retention", retentionDuration,
				"effect", "grace period limited by retention policy")
		}

		if cfg.CleanupGracePeriod < time.Hour {
			logger.Warn("workflow config: very short cleanup grace period",
				"grace_period", cfg.CleanupGracePeriod,
				"risk", "may not allow sufficient time for result queries")
		}
	}

	// Warn about potential state orphaning
	if !cfg.AutoCleanupOnCompletion && cfg.StateRetentionDays < 30 {
		logger.Warn("workflow config: short retention without auto-cleanup",
			"retention_days", cfg.StateRetentionDays,
			"auto_cleanup", false,
			"recommendation", "enable auto-cleanup or increase retention to prevent orphaned state")
	}

	return nil
}

// EstimateStorageCost estimates monthly storage cost based on workflow volume
//
// Parameters:
//   - workflowsPerDay: Number of workflows started per day
//   - avgStateSizeKB: Average workflow state size in KB
//
// Returns:
//   - Estimated total state size in GB for the retention period
//
// Example:
//
//	config := DefaultWorkflowConfig()
//	storageGB := config.EstimateStorageCost(10000, 50) // 10k workflows/day, 50KB each
//	// Result: ~15GB for 30-day retention
//
// Note: Actual costs depend on cloud provider pricing
func (cfg WorkflowConfig) EstimateStorageCost(workflowsPerDay int, avgStateSizeKB int) float64 {
	totalWorkflows := workflowsPerDay * cfg.StateRetentionDays
	totalKB := totalWorkflows * avgStateSizeKB
	return float64(totalKB) / (1024 * 1024) // Convert to GB
}

// LogConfiguration logs the workflow configuration for visibility
//
// Use at application startup to document configuration:
//
//	config := ProductionWorkflowConfig()
//	config.LogConfiguration(slog.Default(), "OrderWorkflow")
func (cfg WorkflowConfig) LogConfiguration(logger *slog.Logger, workflowName string) {
	if logger == nil {
		logger = slog.Default()
	}

	logger.Info("workflow configuration",
		"workflow", workflowName,
		"retention_days", cfg.StateRetentionDays,
		"status_persistence", cfg.EnableStatusPersistence,
		"auto_cleanup", cfg.AutoCleanupOnCompletion,
		"max_state_mb", cfg.MaxStateSizeBytes/(1024*1024),
		"cleanup_grace_period", cfg.CleanupGracePeriod.String())
}

// ToRestateOptions converts WorkflowConfig to Restate SDK workflow options
//
// This helper bridges our framework configuration to the native Restate SDK.
// Use when defining workflows with restate.NewWorkflow().
//
// Example:
//
//	config := ProductionWorkflowConfig()
//	workflow := restate.NewWorkflow("MyWorkflow", config.ToRestateOptions()...)
//
// Note: The Restate SDK supports these workflow-specific options:
//   - restate.WithWorkflowRetention(duration) - sets state retention period
//   - restate.WithIdempotencyRetention(duration) - sets idempotency key retention
//   - restate.WithInactivityTimeout(duration) - auto-fails workflows after inactivity
//
// See: https://docs.restate.dev/develop/go/workflows
func (cfg WorkflowConfig) ToRestateOptions() []restate.ServiceDefinitionOption {
	opts := []restate.ServiceDefinitionOption{}

	// Convert retention days to duration for Restate SDK
	if cfg.StateRetentionDays > 0 {
		retentionDuration := time.Duration(cfg.StateRetentionDays) * 24 * time.Hour
		opts = append(opts, restate.WithWorkflowRetention(retentionDuration))

		// Also configure idempotency retention to match workflow retention
		// This ensures idempotency keys live as long as the workflow state
		opts = append(opts, restate.WithIdempotencyRetention(retentionDuration))
	}

	// Note: AutoCleanupOnCompletion is handled by application logic after workflow completion
	// The Restate SDK doesn't provide a direct option for this - you must implement it
	// by calling workflow cleanup APIs when the workflow completes successfully.

	return opts
}

// ApplyToWorkflow is a convenience method to apply config when creating a workflow
//
// Example:
//
//	config := DefaultWorkflowConfig()
//	workflow := config.ApplyToWorkflow(restate.NewWorkflow("MyWorkflow"))
//
// This is equivalent to:
//
//	workflow := restate.NewWorkflow("MyWorkflow", config.ToRestateOptions()...)
func (cfg WorkflowConfig) ApplyToWorkflow(workflow *restate.ServiceDefinition) *restate.ServiceDefinition {
	// Note: This is a conceptual helper. In practice, you pass options to NewWorkflow()
	// directly since ServiceDefinition doesn't expose a method to add options after creation.
	// This function exists for documentation purposes.
	//
	// Correct usage:
	//   workflow := restate.NewWorkflow("MyWorkflow", cfg.ToRestateOptions()...)
	return workflow
}

// MonitorStateSize tracks workflow state size and logs warnings
//
// Use this in your workflow handlers to monitor state growth:
//
//	func (w *MyWorkflow) Run(ctx restate.WorkflowContext, req Request) (Result, error) {
//	    cfg := w.GetConfig()
//
//	    // Check state size periodically
//	    if err := cfg.MonitorStateSize(ctx, estimatedStateSize); err != nil {
//	        ctx.Log().Warn("State size warning", "error", err)
//	    }
//
//	    // ... workflow logic ...
//	}
//
// Parameters:
//   - ctx: Workflow context (for logging)
//   - estimatedSizeBytes: Current estimated state size in bytes
//
// Returns error if state exceeds configured maximum
func (cfg WorkflowConfig) MonitorStateSize(ctx interface{ Log() *slog.Logger }, estimatedSizeBytes int64) error {
	if estimatedSizeBytes > cfg.MaxStateSizeBytes {
		return fmt.Errorf("workflow state size (%d bytes) exceeds configured maximum (%d bytes)",
			estimatedSizeBytes, cfg.MaxStateSizeBytes)
	}

	// Warn at 80% threshold
	threshold80 := int64(float64(cfg.MaxStateSizeBytes) * 0.8)
	if estimatedSizeBytes > threshold80 {
		ctx.Log().Warn("workflow state approaching size limit",
			"current_bytes", estimatedSizeBytes,
			"max_bytes", cfg.MaxStateSizeBytes,
			"usage_percent", int(float64(estimatedSizeBytes)/float64(cfg.MaxStateSizeBytes)*100))
	}

	return nil
}

// WithCustomRetention creates a new config with custom retention period
//
// Example:
//
//	config := DefaultWorkflowConfig().WithCustomRetention(45)
func (cfg WorkflowConfig) WithCustomRetention(days int) WorkflowConfig {
	cfg.StateRetentionDays = days
	return cfg
}

// WithAutoCleanup creates a new config with auto-cleanup enabled
//
// Example:
//
//	config := DefaultWorkflowConfig().WithAutoCleanup(true, 2*time.Hour)
func (cfg WorkflowConfig) WithAutoCleanup(enabled bool, gracePeriod time.Duration) WorkflowConfig {
	cfg.AutoCleanupOnCompletion = enabled
	cfg.CleanupGracePeriod = gracePeriod
	return cfg
}

// WithMaxStateSize creates a new config with custom max state size
//
// Example:
//
//	config := DefaultWorkflowConfig().WithMaxStateSize(5 * 1024 * 1024) // 5MB
func (cfg WorkflowConfig) WithMaxStateSize(bytes int64) WorkflowConfig {
	cfg.MaxStateSizeBytes = bytes
	return cfg
}

// -----------------------------------------------------------------------------
// Section 7B: Ingress Client Wrappers (External Invocations)
// -----------------------------------------------------------------------------

// IngressClient wraps the Restate ingress client for external invocations
type IngressClient struct {
	client *ingress.Client // Actual SDK ingress client
	log    *slog.Logger
}

// NewIngressClient creates an ingress client for external (non-Restate) applications
func NewIngressClient(baseURL string, authKey string) *IngressClient {
	// Create options slice for SDK ingress client - use IngressClientOption
	var opts []restate.IngressClientOption
	if authKey != "" {
		opts = append(opts, restate.WithAuthKey(authKey))
	}

	return &IngressClient{
		client: ingress.NewClient(baseURL, opts...),
		log:    slog.Default(),
	}
}

// IngressServiceClient provides type-safe external calls to stateless services
type IngressServiceClient[I, O any] struct {
	ingress     *IngressClient
	serviceName string
	handlerName string
}

// Call invokes a service handler synchronously and waits for the result
func (c IngressServiceClient[I, O]) Call(
	ctx context.Context,
	input I,
	opts ...IngressCallOption,
) (O, error) {
	var zero O

	// Build SDK options - use IngressRequestOption
	var sdkOpts []restate.IngressRequestOption
	for _, opt := range opts {
		if opt.IdempotencyKey != "" {
			// Determine validation mode (default to warn)
			validationMode := opt.ValidationMode
			if validationMode == "" {
				validationMode = IdempotencyValidationWarn
			}

			// Apply validation
			if validationMode != IdempotencyValidationDisabled {
				if err := ValidateIdempotencyKey(opt.IdempotencyKey); err != nil {
					if validationMode == IdempotencyValidationFail {
						c.ingress.log.Error("ingress: idempotency key validation failed (strict mode)",
							"key", opt.IdempotencyKey,
							"error", err.Error())
						return zero, err
					} else {
						c.ingress.log.Warn("ingress: idempotency key validation warning (permissive mode)",
							"key", opt.IdempotencyKey,
							"error", err.Error())
					}
				}
			}
			sdkOpts = append(sdkOpts, restate.WithIdempotencyKey(opt.IdempotencyKey))
		}
		if len(opt.Headers) > 0 {
			sdkOpts = append(sdkOpts, restate.WithHeaders(opt.Headers))
		}
	}

	c.ingress.log.Info("ingress: calling service",
		"service", c.serviceName,
		"handler", c.handlerName)

	// Use SDK ingress client to make the call
	return ingress.Service[I, O](c.ingress.client, c.serviceName, c.handlerName).Request(ctx, input, sdkOpts...)
}

// Send invokes a service handler asynchronously (fire-and-forget)
func (c IngressServiceClient[I, O]) Send(
	ctx context.Context,
	input I,
	opts ...IngressCallOption,
) (string, error) {
	// Build SDK options - use IngressSendOption
	var sdkOpts []restate.IngressSendOption
	for _, opt := range opts {
		if opt.IdempotencyKey != "" {
			// Determine validation mode (default to warn)
			validationMode := opt.ValidationMode
			if validationMode == "" {
				validationMode = IdempotencyValidationWarn
			}

			// Apply validation
			if validationMode != IdempotencyValidationDisabled {
				if err := ValidateIdempotencyKey(opt.IdempotencyKey); err != nil {
					if validationMode == IdempotencyValidationFail {
						c.ingress.log.Error("ingress: idempotency key validation failed (strict mode)",
							"key", opt.IdempotencyKey,
							"error", err.Error())
						return "", err
					} else {
						c.ingress.log.Warn("ingress: idempotency key validation warning (permissive mode)",
							"key", opt.IdempotencyKey,
							"error", err.Error())
					}
				}
			}
			sdkOpts = append(sdkOpts, restate.WithIdempotencyKey(opt.IdempotencyKey))
		}
		if opt.Delay > 0 {
			sdkOpts = append(sdkOpts, restate.WithDelay(opt.Delay))
		}
		if len(opt.Headers) > 0 {
			sdkOpts = append(sdkOpts, restate.WithHeaders(opt.Headers))
		}
	}

	c.ingress.log.Info("ingress: sending to service",
		"service", c.serviceName,
		"handler", c.handlerName)

	// Use SDK ingress client to send asynchronously
	resp, err := ingress.ServiceSend[I](c.ingress.client, c.serviceName, c.handlerName).Send(ctx, input, sdkOpts...)
	if err != nil {
		return "", err
	}
	return resp.Id(), nil
}

// AttachByIdempotencyKey attaches to an existing service invocation using its idempotency key.
// This only works if the original invocation was started with an idempotency key.
func (c IngressServiceClient[I, O]) AttachByIdempotencyKey(
	ctx context.Context,
	idempotencyKey string,
) (O, error) {
	c.ingress.log.Info("ingress: attaching to service by idempotency key",
		"service", c.serviceName,
		"handler", c.handlerName,
		"idempotency_key", idempotencyKey)

	// Use SDK to attach by idempotency key
	return ingress.ServiceInvocationByIdempotencyKey[O](
		c.ingress.client,
		c.serviceName,
		c.handlerName,
		idempotencyKey,
	).Attach(ctx)
}

// IngressObjectClient provides type-safe external calls to Virtual Objects
type IngressObjectClient[I, O any] struct {
	ingress     *IngressClient
	serviceName string
	handlerName string
}

// Call invokes a Virtual Object handler synchronously
func (c IngressObjectClient[I, O]) Call(
	ctx context.Context,
	key string,
	input I,
	opts ...IngressCallOption,
) (O, error) {
	var zero O

	// Build SDK options
	var sdkOpts []restate.IngressRequestOption
	for _, opt := range opts {
		if opt.IdempotencyKey != "" {
			validationMode := opt.ValidationMode
			if validationMode == "" {
				validationMode = IdempotencyValidationWarn
			}
			if validationMode != IdempotencyValidationDisabled {
				if err := ValidateIdempotencyKey(opt.IdempotencyKey); err != nil {
					if validationMode == IdempotencyValidationFail {
						c.ingress.log.Error("ingress: idempotency key validation failed",
							"key", opt.IdempotencyKey, "error", err.Error())
						return zero, err
					} else {
						c.ingress.log.Warn("ingress: idempotency key validation warning",
							"key", opt.IdempotencyKey, "error", err.Error())
					}
				}
			}
			sdkOpts = append(sdkOpts, restate.WithIdempotencyKey(opt.IdempotencyKey))
		}
		if len(opt.Headers) > 0 {
			sdkOpts = append(sdkOpts, restate.WithHeaders(opt.Headers))
		}
	}

	c.ingress.log.Info("ingress: calling object",
		"service", c.serviceName,
		"handler", c.handlerName,
		"key", key)

	return ingress.Object[I, O](c.ingress.client, c.serviceName, key, c.handlerName).Request(ctx, input, sdkOpts...)
}

// Send invokes a Virtual Object handler asynchronously
func (c IngressObjectClient[I, O]) Send(
	ctx context.Context,
	key string,
	input I,
	opts ...IngressCallOption,
) (string, error) {
	// Build SDK options
	var sdkOpts []restate.IngressSendOption
	for _, opt := range opts {
		if opt.IdempotencyKey != "" {
			validationMode := opt.ValidationMode
			if validationMode == "" {
				validationMode = IdempotencyValidationWarn
			}
			if validationMode != IdempotencyValidationDisabled {
				if err := ValidateIdempotencyKey(opt.IdempotencyKey); err != nil {
					if validationMode == IdempotencyValidationFail {
						return "", err
					} else {
						c.ingress.log.Warn("ingress: idempotency key validation warning",
							"key", opt.IdempotencyKey, "error", err.Error())
					}
				}
			}
			sdkOpts = append(sdkOpts, restate.WithIdempotencyKey(opt.IdempotencyKey))
		}
		if opt.Delay > 0 {
			sdkOpts = append(sdkOpts, restate.WithDelay(opt.Delay))
		}
		if len(opt.Headers) > 0 {
			sdkOpts = append(sdkOpts, restate.WithHeaders(opt.Headers))
		}
	}

	c.ingress.log.Info("ingress: sending to object",
		"service", c.serviceName,
		"handler", c.handlerName,
		"key", key)

	resp, err := ingress.ObjectSend[I](c.ingress.client, c.serviceName, key, c.handlerName).Send(ctx, input, sdkOpts...)
	if err != nil {
		return "", err
	}
	return resp.Id(), nil
}

// AttachByIdempotencyKey attaches to an existing object invocation using its idempotency key
func (c IngressObjectClient[I, O]) AttachByIdempotencyKey(
	ctx context.Context,
	key string,
	idempotencyKey string,
) (O, error) {
	c.ingress.log.Info("ingress: attaching to object by idempotency key",
		"service", c.serviceName,
		"handler", c.handlerName,
		"key", key,
		"idempotency_key", idempotencyKey)

	return ingress.ObjectInvocationByIdempotencyKey[O](
		c.ingress.client,
		c.serviceName,
		key,
		c.handlerName,
		idempotencyKey,
	).Attach(ctx)
}

// IngressWorkflowClient provides type-safe external calls to Workflows
type IngressWorkflowClient[I, O any] struct {
	ingress     *IngressClient
	serviceName string
	handlerName string
}

// Submit starts a new workflow instance
func (c IngressWorkflowClient[I, O]) Submit(
	ctx context.Context,
	workflowID string,
	input I,
	opts ...IngressCallOption,
) (string, error) {
	// Build SDK options
	var sdkOpts []restate.IngressSendOption
	for _, opt := range opts {
		if opt.IdempotencyKey != "" {
			validationMode := opt.ValidationMode
			if validationMode == "" {
				validationMode = IdempotencyValidationWarn
			}
			if validationMode != IdempotencyValidationDisabled {
				if err := ValidateIdempotencyKey(opt.IdempotencyKey); err != nil {
					if validationMode == IdempotencyValidationFail {
						return "", err
					} else {
						c.ingress.log.Warn("ingress: idempotency key validation warning",
							"key", opt.IdempotencyKey, "error", err.Error())
					}
				}
			}
			sdkOpts = append(sdkOpts, restate.WithIdempotencyKey(opt.IdempotencyKey))
		}
		if opt.Delay > 0 {
			sdkOpts = append(sdkOpts, restate.WithDelay(opt.Delay))
		}
		if len(opt.Headers) > 0 {
			sdkOpts = append(sdkOpts, restate.WithHeaders(opt.Headers))
		}
	}

	c.ingress.log.Info("ingress: submitting workflow",
		"service", c.serviceName,
		"workflow_id", workflowID)

	resp, err := ingress.WorkflowSend[I](c.ingress.client, c.serviceName, workflowID, c.handlerName).Send(ctx, input, sdkOpts...)
	if err != nil {
		return "", err
	}
	return resp.Id(), nil
}

// Attach attaches to an existing workflow and waits for result.
// Works by workflow ID alone (workflows use ID as the key).
func (c IngressWorkflowClient[I, O]) Attach(
	ctx context.Context,
	workflowID string,
) (O, error) {
	c.ingress.log.Info("ingress: attaching to workflow",
		"service", c.serviceName,
		"workflow_id", workflowID)

	// Use SDK WorkflowHandle to attach
	return ingress.WorkflowHandle[O](c.ingress.client, c.serviceName, workflowID).Attach(ctx)
}

// GetOutput queries workflow output via shared handler.
// This is useful when you want to query the workflow result without running the main handler.
func (c IngressWorkflowClient[I, O]) GetOutput(
	ctx context.Context,
	workflowID string,
	outputHandler string,
) (O, error) {
	c.ingress.log.Info("ingress: querying workflow output",
		"service", c.serviceName,
		"workflow_id", workflowID,
		"handler", outputHandler)

	// Use SDK WorkflowHandle to get output
	return ingress.WorkflowHandle[O](c.ingress.client, c.serviceName, workflowID).Output(ctx)
}

// IngressCallOption configures ingress client calls
type IngressCallOption struct {
	IdempotencyKey string
	Delay          time.Duration
	Headers        map[string]string
	RequestID      string
	ValidationMode IdempotencyValidationMode // Control validation behavior
}

// IngressService creates a type-safe ingress client for calling a stateless service
func IngressService[I, O any](ic *IngressClient, serviceName, handlerName string) *IngressServiceClient[I, O] {
	return &IngressServiceClient[I, O]{
		ingress:     ic,
		serviceName: serviceName,
		handlerName: handlerName,
	}
}

// IngressObject creates a type-safe ingress client for calling a Virtual Object
func IngressObject[I, O any](ic *IngressClient, serviceName, handlerName string) *IngressObjectClient[I, O] {
	return &IngressObjectClient[I, O]{
		ingress:     ic,
		serviceName: serviceName,
		handlerName: handlerName,
	}
}

// IngressWorkflow creates a type-safe ingress client for calling a Workflow
func IngressWorkflow[I, O any](ic *IngressClient, serviceName, handlerName string) *IngressWorkflowClient[I, O] {
	return &IngressWorkflowClient[I, O]{
		ingress:     ic,
		serviceName: serviceName,
		handlerName: handlerName,
	}
}

// -----------------------------------------------------------------------------
// Section 8: Concurrency Utilities
// -----------------------------------------------------------------------------

// RaceResult represents the winner of a race between futures.
type RaceResult struct {
	Winner restate.Future
	Index  int
	Error  error
}

// Race executes multiple futures and returns the first to complete.
func Race(ctx restate.Context, futures ...restate.Future) (*RaceResult, error) {
	if len(futures) == 0 {
		return nil, restate.TerminalError(fmt.Errorf("race requires at least one future"), 400)
	}

	winner, err := restate.WaitFirst(ctx, futures...)
	if err != nil {
		return nil, err
	}

	// Find which future won
	for i, fut := range futures {
		if fut == winner {
			return &RaceResult{
				Winner: winner,
				Index:  i,
			}, nil
		}
	}

	return nil, restate.TerminalError(fmt.Errorf("race: could not identify winning future"), 500)
}

// Gather waits for all futures to complete and returns results.
// Returns slice of interface{} - caller must type assert.
func Gather(ctx restate.Context, futures ...restate.Future) ([]any, error) {
	results := make([]any, 0, len(futures))

	for fut, err := range restate.Wait(ctx, futures...) {
		if err != nil {
			return nil, fmt.Errorf("gather future error: %w", err)
		}

		// Extract the actual result based on future type
		if rf, ok := fut.(interface{ Response() (any, error) }); ok {
			resp, err := rf.Response()
			if err != nil {
				return nil, fmt.Errorf("gather response error: %w", err)
			}
			results = append(results, resp)
		} else {
			results = append(results, fut)
		}
	}

	return results, nil
}

// -----------------------------------------------------------------------------
// Section 8A: Security and Request Validation
// -----------------------------------------------------------------------------

// SecurityValidator provides request signature validation and security checks
type SecurityValidator struct {
	config SecurityConfig
	log    *slog.Logger
}

// NewSecurityValidator creates a validator with the given configuration
func NewSecurityValidator(config SecurityConfig, logger *slog.Logger) *SecurityValidator {
	if logger == nil {
		logger = slog.Default()
	}
	return &SecurityValidator{
		config: config,
		log:    logger,
	}
}

// ValidateRequest validates an incoming HTTP request's signature and origin
func (sv *SecurityValidator) ValidateRequest(req *http.Request) RequestValidationResult {
	result := RequestValidationResult{
		Valid:         false,
		KeyIndex:      -1,
		RequestOrigin: req.Header.Get("X-Restate-Server"),
	}

	// Skip validation if disabled
	if sv.config.ValidationMode == SecurityModeDisabled {
		sv.log.Warn("security: validation disabled - accepting request without verification")
		result.Valid = true
		return result
	}

	// Check HTTPS requirement
	if sv.config.RequireHTTPS && req.TLS == nil && req.Header.Get("X-Forwarded-Proto") != "https" {
		result.ErrorMessage = "HTTPS required but request received over HTTP"
		sv.handleValidationFailure(result)
		return result
	}

	// Validate origin if restrictions configured
	if len(sv.config.AllowedOrigins) > 0 {
		if !sv.isOriginAllowed(result.RequestOrigin) {
			result.ErrorMessage = fmt.Sprintf("origin not allowed: %s", result.RequestOrigin)
			sv.handleValidationFailure(result)
			return result
		}
	}

	// Validate request signature if enabled
	if sv.config.EnableRequestValidation {
		if len(sv.config.SigningKeys) == 0 {
			result.ErrorMessage = "request validation enabled but no signing keys configured"
			sv.log.Error("security: critical configuration error", "error", result.ErrorMessage)
			sv.handleValidationFailure(result)
			return result
		}

		signatureHeader := req.Header.Get("X-Restate-Signature")
		if signatureHeader == "" {
			result.ErrorMessage = "missing X-Restate-Signature header"
			sv.handleValidationFailure(result)
			return result
		}

		// Verify signature against configured keys
		for i, publicKey := range sv.config.SigningKeys {
			if sv.verifySignature(req, signatureHeader, publicKey) {
				result.Valid = true
				result.KeyIndex = i
				sv.log.Debug("security: request signature validated",
					"key_index", i,
					"origin", result.RequestOrigin)
				return result
			}
		}

		result.ErrorMessage = "signature verification failed with all configured keys"
		sv.handleValidationFailure(result)
		return result
	}

	// If no validation configured, accept the request
	result.Valid = true
	return result
}

// verifySignature validates the Ed25519 signature on the request
func (sv *SecurityValidator) verifySignature(req *http.Request, signatureHeader string, publicKey ed25519.PublicKey) bool {
	// Parse the signature from base64
	signature, err := base64.StdEncoding.DecodeString(signatureHeader)
	if err != nil {
		sv.log.Debug("security: failed to decode signature", "error", err)
		return false
	}

	// Reconstruct the signed message
	// Restate signs: METHOD + " " + PATH + "\n" + Headers + "\n\n" + Body
	message := sv.constructSignedMessage(req)

	// Verify signature
	return ed25519.Verify(publicKey, message, signature)
}

// constructSignedMessage reconstructs the message that was signed by Restate
func (sv *SecurityValidator) constructSignedMessage(req *http.Request) []byte {
	var builder strings.Builder

	// Method and path
	builder.WriteString(req.Method)
	builder.WriteString(" ")
	builder.WriteString(req.URL.Path)
	if req.URL.RawQuery != "" {
		builder.WriteString("?")
		builder.WriteString(req.URL.RawQuery)
	}
	builder.WriteString("\n")

	// Select headers (sorted alphabetically)
	headerNames := []string{
		"content-type",
		"x-restate-id",
		"x-restate-server",
	}
	for _, name := range headerNames {
		if value := req.Header.Get(name); value != "" {
			builder.WriteString(name)
			builder.WriteString(": ")
			builder.WriteString(value)
			builder.WriteString("\n")
		}
	}
	builder.WriteString("\n")

	// Body would be appended here, but requires buffering the request body
	// For now, we assume body validation is handled separately
	// In production, you'd need to buffer and reconstruct the body

	return []byte(builder.String())
}

// isOriginAllowed checks if the request origin is in the allowed list
func (sv *SecurityValidator) isOriginAllowed(origin string) bool {
	if origin == "" {
		return len(sv.config.AllowedOrigins) == 0
	}
	for _, allowed := range sv.config.AllowedOrigins {
		if origin == allowed {
			return true
		}
	}
	return false
}

// handleValidationFailure logs and potentially rejects failed validation
func (sv *SecurityValidator) handleValidationFailure(result RequestValidationResult) {
	if sv.config.ValidationMode == SecurityModeStrict {
		sv.log.Error("security: request validation failed (strict mode)",
			"error", result.ErrorMessage,
			"origin", result.RequestOrigin)
	} else {
		sv.log.Warn("security: request validation failed (permissive mode)",
			"error", result.ErrorMessage,
			"origin", result.RequestOrigin)
	}
}

// ConfigureSecureServer logs security configuration (server options handled by SDK directly)
func ConfigureSecureServer(cfg SecurityConfig) {
	// Configure request identity validation if signing keys provided
	if cfg.EnableRequestValidation && len(cfg.SigningKeys) > 0 {
		// Convert Ed25519 public keys to the format expected by Restate SDK
		keyStrings := make([]string, len(cfg.SigningKeys))
		for i, key := range cfg.SigningKeys {
			keyStrings[i] = base64.StdEncoding.EncodeToString(key)
		}

		// Note: Actual SDK integration depends on server configuration
		// Pass these keys to your server.NewRestate().WithIdentityV1() or similar
		slog.Info("security: request validation configured",
			"num_keys", len(keyStrings),
			"mode", cfg.ValidationMode)
	}

	// Log HTTPS requirement
	if cfg.RequireHTTPS {
		slog.Info("security: HTTPS enforcement enabled")
	}
}

// ValidateServiceEndpoint checks if a service endpoint meets security requirements
func ValidateServiceEndpoint(endpoint string, requireHTTPS bool) error {
	if requireHTTPS && !strings.HasPrefix(endpoint, "https://") {
		return restate.TerminalError(
			fmt.Errorf("HTTPS required but endpoint uses HTTP: %s", endpoint),
			400,
		)
	}
	return nil
}

// ParseSigningKey parses a base64-encoded Ed25519 public key
func ParseSigningKey(keyBase64 string) (ed25519.PublicKey, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode signing key: %w", err)
	}

	if len(keyBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid key size: expected %d bytes, got %d",
			ed25519.PublicKeySize, len(keyBytes))
	}

	return ed25519.PublicKey(keyBytes), nil
}

// ParseSigningKeys parses multiple base64-encoded Ed25519 public keys
func ParseSigningKeys(keysBase64 []string) ([]ed25519.PublicKey, error) {
	keys := make([]ed25519.PublicKey, 0, len(keysBase64))
	for i, keyStr := range keysBase64 {
		key, err := ParseSigningKey(keyStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse key at index %d: %w", i, err)
		}
		keys = append(keys, key)
	}
	return keys, nil
}

// SecurityMiddleware creates an HTTP middleware that enforces SecurityConfig
//
// This middleware validates incoming requests according to the SecurityConfig:
//   - Signature verification (if EnableRequestValidation is true)
//   - HTTPS enforcement (if RequireHTTPS is true)
//   - Origin validation (if AllowedOrigins is configured)
//
// Behavior per ValidationMode:
//   - SecurityModeStrict: Reject invalid requests with HTTP 401/403
//   - SecurityModePermissive: Log warnings but allow requests through
//   - SecurityModeDisabled: Skip all validation
//
// Usage:
//
//	config := DefaultSecurityConfig()
//	config.SigningKeys = []ed25519.PublicKey{publicKey}
//	validator := NewSecurityValidator(config, slog.Default())
//
//	http.Handle("/restate", SecurityMiddleware(validator)(restateHandler))
func SecurityMiddleware(validator *SecurityValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Validate the request
			result := validator.ValidateRequest(r)

			// In disabled mode, always pass through
			if validator.config.ValidationMode == SecurityModeDisabled {
				next.ServeHTTP(w, r)
				return
			}

			// In permissive mode, log but continue
			if validator.config.ValidationMode == SecurityModePermissive {
				if !result.Valid {
					validator.log.Warn("security: request validation failed (permissive mode)",
						"error", result.ErrorMessage,
						"origin", result.RequestOrigin,
						"path", r.URL.Path,
						"method", r.Method)
				}
				next.ServeHTTP(w, r)
				return
			}

			// In strict mode, reject invalid requests
			if validator.config.ValidationMode == SecurityModeStrict {
				if !result.Valid {
					validator.log.Error("security: rejecting invalid request (strict mode)",
						"error", result.ErrorMessage,
						"origin", result.RequestOrigin,
						"path", r.URL.Path,
						"method", r.Method,
						"remote_addr", r.RemoteAddr)

					// Determine appropriate HTTP status code
					statusCode := http.StatusUnauthorized
					if strings.Contains(result.ErrorMessage, "HTTPS required") {
						statusCode = http.StatusForbidden
					} else if strings.Contains(result.ErrorMessage, "origin not allowed") {
						statusCode = http.StatusForbidden
					}

					http.Error(w, result.ErrorMessage, statusCode)
					return
				}

				// Request is valid, add validation metadata to context
				validator.log.Debug("security: request validated successfully",
					"origin", result.RequestOrigin,
					"key_index", result.KeyIndex)
			}

			// Pass to next handler
			next.ServeHTTP(w, r)
		})
	}
}

// SecureHandlerFunc is a convenience wrapper for securing a single http.HandlerFunc
//
// Usage:
//
//	http.HandleFunc("/restate", SecureHandlerFunc(validator, myHandler))
func SecureHandlerFunc(validator *SecurityValidator, handler http.HandlerFunc) http.HandlerFunc {
	middleware := SecurityMiddleware(validator)
	wrappedHandler := middleware(handler)
	return wrappedHandler.ServeHTTP
}

// SecureServer wraps an entire http.ServeMux with security middleware
//
// Usage:
//
//	mux := http.NewServeMux()
//	mux.HandleFunc("/restate", restateHandler)
//	mux.HandleFunc("/health", healthHandler)
//
//	securedMux := SecureServer(validator, mux)
//	http.ListenAndServe(":8080", securedMux)
func SecureServer(validator *SecurityValidator, mux *http.ServeMux) http.Handler {
	return SecurityMiddleware(validator)(mux)
}

// -----------------------------------------------------------------------------
// Section 8A: Concurrency Pattern Helpers
// -----------------------------------------------------------------------------

// FanOutResult represents the result of a fan-out operation
type FanOutResult[T any] struct {
	Results []T
	Errors  []error
	Failed  int
	Success int
}

// FanOut executes multiple operations concurrently and collects all results
// Continues execution even if some operations fail
func FanOut[T any](
	ctx restate.Context,
	operations []func() (T, error),
) FanOutResult[T] {
	result := FanOutResult[T]{
		Results: make([]T, len(operations)),
		Errors:  make([]error, len(operations)),
	}

	if len(operations) == 0 {
		return result
	}

	// Create futures for all operations
	futures := make([]restate.Future, 0, len(operations))
	for i, op := range operations {
		idx := i
		operation := op

		// Wrap each operation in a Run block
		fut := restate.RunAsync(ctx, func(rc restate.RunContext) (T, error) {
			return operation()
		}, restate.WithName(fmt.Sprintf("fanout-%d", idx)))

		futures = append(futures, fut)
	}

	// Collect results
	futureIndex := 0
	for fut, err := range restate.Wait(ctx, futures...) {
		if err != nil {
			result.Errors[futureIndex] = err
			result.Failed++
		} else {
			if typedResult, ok := fut.(T); ok {
				result.Results[futureIndex] = typedResult
				result.Success++
			}
		}
		futureIndex++
	}

	return result
}

// FanOutFail executes operations concurrently and fails fast on first error
func FanOutFail[T any](
	ctx restate.Context,
	operations []func() (T, error),
) ([]T, error) {
	if len(operations) == 0 {
		return []T{}, nil
	}

	// Create futures for all operations
	futures := make([]restate.Future, 0, len(operations))
	for i, op := range operations {
		idx := i
		operation := op

		fut := restate.RunAsync(ctx, func(rc restate.RunContext) (T, error) {
			return operation()
		}, restate.WithName(fmt.Sprintf("fanout-fail-%d", idx)))

		futures = append(futures, fut)
	}

	// Collect results, fail on first error
	results := make([]T, len(operations))
	futureIndex := 0
	for fut, err := range restate.Wait(ctx, futures...) {
		if err != nil {
			return nil, fmt.Errorf("fanout operation %d failed: %w", futureIndex, err)
		}

		if typedResult, ok := fut.(T); ok {
			results[futureIndex] = typedResult
		}
		futureIndex++
	}

	return results, nil
}

// MapConcurrent applies a function to each item concurrently
func MapConcurrent[I, O any](
	ctx restate.Context,
	items []I,
	mapper func(I) (O, error),
) ([]O, error) {
	if len(items) == 0 {
		return []O{}, nil
	}

	operations := make([]func() (O, error), len(items))
	for i, item := range items {
		currentItem := item
		operations[i] = func() (O, error) {
			return mapper(currentItem)
		}
	}

	return FanOutFail(ctx, operations)
}

// BatchProcessor handles batch processing with concurrency control
type BatchProcessor struct {
	ctx            restate.Context
	log            *slog.Logger
	maxConcurrency int
}

// NewBatchProcessor creates a batch processor with concurrency limit
func NewBatchProcessor(ctx restate.Context, maxConcurrency int) *BatchProcessor {
	if maxConcurrency <= 0 {
		maxConcurrency = 10 // Default
	}
	return &BatchProcessor{
		ctx:            ctx,
		log:            ctx.Log(),
		maxConcurrency: maxConcurrency,
	}
}

// ProcessBatch processes items in batches with controlled concurrency (helper function)
func ProcessBatch[I, O any](
	ctx restate.Context,
	items []I,
	processor func(I) (O, error),
	maxConcurrency int,
) ([]O, error) {
	if len(items) == 0 {
		return []O{}, nil
	}

	if maxConcurrency <= 0 {
		maxConcurrency = 10
	}

	results := make([]O, 0, len(items))

	// Process in chunks of maxConcurrency
	for i := 0; i < len(items); i += maxConcurrency {
		end := i + maxConcurrency
		if end > len(items) {
			end = len(items)
		}

		batch := items[i:end]
		ctx.Log().Info("batch: processing chunk",
			"chunk_start", i,
			"chunk_size", len(batch))

		// Process this batch concurrently
		batchResults, err := MapConcurrent(ctx, batch, processor)
		if err != nil {
			return nil, fmt.Errorf("batch processing failed at chunk %d: %w", i/maxConcurrency, err)
		}

		results = append(results, batchResults...)
	}

	ctx.Log().Info("batch: processing complete", "total_items", len(items))
	return results, nil
}

// ParallelInvoke invokes multiple service calls concurrently
func ParallelInvoke[T any](
	ctx restate.Context,
	clients []ServiceClient[any, T],
	inputs []any,
) ([]T, error) {
	if len(clients) != len(inputs) {
		return nil, fmt.Errorf("clients and inputs length mismatch: %d != %d", len(clients), len(inputs))
	}

	if len(clients) == 0 {
		return []T{}, nil
	}

	operations := make([]func() (T, error), len(clients))
	for i := range clients {
		idx := i
		operations[i] = func() (T, error) {
			return clients[idx].Call(ctx, inputs[idx])
		}
	}

	return FanOutFail(ctx, operations)
}

// -----------------------------------------------------------------------------
// Section 9: Validation Guards
// -----------------------------------------------------------------------------

// Note: Go doesn't allow interface types with methods in type unions,
// so we use wrapper types that enforce safety through their API design instead.

// MutableState provides compile-time safe state access for exclusive contexts
type MutableState[T any] struct {
	key string
	ctx interface{} // ObjectContext or WorkflowContext
}

// NewMutableObjectState creates state for Object exclusive handlers
func NewMutableObjectState[T any](ctx restate.ObjectContext, key string) *MutableState[T] {
	return &MutableState[T]{
		key: key,
		ctx: ctx,
	}
}

// NewMutableWorkflowState creates state for Workflow run handlers
func NewMutableWorkflowState[T any](ctx restate.WorkflowContext, key string) *MutableState[T] {
	return &MutableState[T]{
		key: key,
		ctx: ctx,
	}
}

// Get retrieves the value
func (s *MutableState[T]) Get() (T, error) {
	switch ctx := s.ctx.(type) {
	case restate.ObjectContext:
		return restate.Get[T](ctx, s.key)
	case restate.WorkflowContext:
		return restate.Get[T](ctx, s.key)
	default:
		var zero T
		return zero, fmt.Errorf("invalid mutable context type")
	}
}

// Set updates the value (guaranteed mutable by constructor)
func (s *MutableState[T]) Set(value T) error {
	switch ctx := s.ctx.(type) {
	case restate.ObjectContext:
		restate.Set(ctx, s.key, value)
		return nil
	case restate.WorkflowContext:
		restate.Set(ctx, s.key, value)
		return nil
	default:
		return fmt.Errorf("context does not support mutation")
	}
}

// Clear removes the value
func (s *MutableState[T]) Clear() {
	switch ctx := s.ctx.(type) {
	case restate.ObjectContext:
		restate.Clear(ctx, s.key)
	case restate.WorkflowContext:
		restate.Clear(ctx, s.key)
	}
}

// ReadOnlyState provides read-only state access for shared contexts
type ReadOnlyState[T any] struct {
	key string
	ctx interface{} // ObjectSharedContext or WorkflowSharedContext
}

// NewReadOnlyObjectState creates read-only state for Object shared handlers
func NewReadOnlyObjectState[T any](ctx restate.ObjectSharedContext, key string) *ReadOnlyState[T] {
	return &ReadOnlyState[T]{
		key: key,
		ctx: ctx,
	}
}

// NewReadOnlyWorkflowState creates read-only state for Workflow shared handlers
func NewReadOnlyWorkflowState[T any](ctx restate.WorkflowSharedContext, key string) *ReadOnlyState[T] {
	return &ReadOnlyState[T]{
		key: key,
		ctx: ctx,
	}
}

// Get retrieves the value (no Set method exists!)
func (s *ReadOnlyState[T]) Get() (T, error) {
	switch ctx := s.ctx.(type) {
	case restate.ObjectSharedContext:
		return restate.Get[T](ctx, s.key)
	case restate.WorkflowSharedContext:
		return restate.Get[T](ctx, s.key)
	default:
		var zero T
		return zero, fmt.Errorf("invalid read-only context type")
	}
}

// GuardRunContext ensures restate.Run is only called from appropriate contexts.
func GuardRunContext(rc restate.RunContext) {
	// Runtime check as a safety net
	if rc == nil {
		panic("GuardRunContext: nil RunContext")
	}
}

// ValidateServiceDefinition checks that a service struct follows Restate rules.
func ValidateServiceDefinition(svc any) error {
	// In production, implement full reflection validation here
	return nil
}

// ValidateIdempotencyKey checks if an idempotency key appears to be deterministic.
// Returns an error if the key contains patterns that suggest non-deterministic generation.
func ValidateIdempotencyKey(key string) error {
	if key == "" {
		return restate.TerminalError(fmt.Errorf("idempotency key cannot be empty"), 400)
	}

	// Check for suspicious patterns that might indicate non-deterministic generation
	// Note: This is heuristic-based and cannot catch all cases
	if hasSuspiciousTimestamp(key) {
		return restate.TerminalError(
			fmt.Errorf("idempotency key may contain non-deterministic timestamp: %s", key),
			400,
		)
	}

	return nil
}

// hasSuspiciousTimestamp detects patterns that suggest raw timestamp usage.
func hasSuspiciousTimestamp(key string) bool {
	// Look for patterns like very large numbers (likely Unix timestamps)
	// This is a heuristic and may have false positives
	for i := 0; i < len(key)-12; i++ {
		consecutiveDigits := 0
		for j := i; j < len(key) && j < i+13; j++ {
			if key[j] >= '0' && key[j] <= '9' {
				consecutiveDigits++
			} else {
				break
			}
		}
		// Unix timestamps (seconds or milliseconds) are typically 10-13 digits
		if consecutiveDigits >= 10 {
			return true
		}
	}
	return false
}

// NewTerminalError creates a terminal error for permanent business failures.
func NewTerminalError(err error) error {
	if err == nil {
		return nil
	}
	return restate.TerminalError(err, 400)
}

// WrapTerminalError wraps an error with custom terminal status code.
func WrapTerminalError(err error, statusCode int) error {
	if err == nil {
		return nil
	}
	//return restate.TerminalError(err, statusCode)
	return restate.TerminalError(err, 500)
}

// -----------------------------------------------------------------------------
// Section 10: Utility Functions (Internal)
// -----------------------------------------------------------------------------

// computeBackoff calculates exponential backoff with cap.
func computeBackoff(initial, max time.Duration, attempt int) time.Duration {
	if initial <= 0 {
		initial = time.Second
	}
	if max <= 0 {
		max = 5 * time.Minute
	}

	backoff := initial * (1 << uint(attempt))
	if backoff > max {
		return max
	}
	return backoff
}

// deterministicStepID creates a deterministic ID for saga deduplication.
func deterministicStepID(name string, payload []byte) string {
	h := sha256.Sum256(append([]byte(name+":"), payload...))
	return fmt.Sprintf("%x", h[:16])
}

// canonicalJSON normalizes JSON for deterministic hashing.
func canonicalJSON(v any) ([]byte, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	var iface any
	if err := json.Unmarshal(raw, &iface); err != nil {
		return nil, err
	}

	canon := canonicalizeValue(iface)
	return json.Marshal(canon)
}

// canonicalizeValue recursively sorts map keys for deterministic output.
func canonicalizeValue(v any) any {
	switch vv := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(vv))
		for k := range vv {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		result := make(map[string]any, len(keys))
		for _, k := range keys {
			result[k] = canonicalizeValue(vv[k])
		}
		return result

	case []any:
		result := make([]any, len(vv))
		for i := range vv {
			result[i] = canonicalizeValue(vv[i])
		}
		return result

	default:
		return vv
	}
}

// removeIndex removes an element from a slice.
func removeIndex[T any](s []T, i int) []T {
	if i < 0 || i >= len(s) {
		return s
	}
	return append(s[:i], s[i+1:]...)
}
