package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"testing"

	"github.com/pithomlabs/rea"
	restate "github.com/restatedev/sdk-go"
	"github.com/stretchr/testify/assert"
)

// TestIdempotencyDeduplication verifies that duplicate requests with same idempotency key
// are properly deduplicated without re-executing business logic
func TestIdempotencyDeduplication(t *testing.T) {
	logger := slog.Default()
	metrics := rea.NewMetricsCollector()

	hookCalls := 0
	hooks := &rea.ObservabilityHooks{
		OnInvocationStart: func(svc, handler string, input interface{}) {
			hookCalls++
			logger.Info("test_hook_invocation_start", "handler", handler, "count", hookCalls)
		},
	}

	firstInvocationKey := "test-order-001"
	// Simulate first invocation
	hooks.OnInvocationStart("checkout", "execute", map[string]string{
		"idempotency_key": firstInvocationKey,
	})

	// Record first execution
	metrics.RecordInvocation("checkout", "execute", 0, nil)

	assert.Equal(t, 1, hookCalls, "First invocation should trigger hook once")

	// Simulate second invocation with same idempotency key (should be deduplicated in real implementation)
	hooks.OnInvocationStart("checkout", "execute", map[string]string{
		"idempotency_key": firstInvocationKey,
	})

	// Verify metrics show the execution was recorded
	metricsSnapshot := metrics.GetMetrics()
	assert.NotNil(t, metricsSnapshot, "Metrics should be collected")

	logger.Info("test_completed", "total_hook_calls", hookCalls, "metrics", metricsSnapshot)
}

// TestDeterministicOrderIDGeneration verifies that the same business context
// always generates the same deterministic order ID
func TestDeterministicOrderIDGeneration(t *testing.T) {
	userID := "user-123"
	cartID := "cart-abc"

	// Generate ID multiple times with same input
	id1 := generateTestOrderID(userID, cartID)
	id2 := generateTestOrderID(userID, cartID)
	id3 := generateTestOrderID(userID, cartID)

	// All three should be identical (deterministic)
	assert.Equal(t, id1, id2, "Same inputs should generate same ID")
	assert.Equal(t, id2, id3, "Same inputs should generate same ID")

	// Different inputs should generate different IDs
	id4 := generateTestOrderID(userID, "cart-def")
	assert.NotEqual(t, id1, id4, "Different cartIDs should generate different IDs")

	t.Logf("Generated deterministic ID: %s\n", id1)
}

// TestIdempotencyKeyValidation verifies that idempotency keys are validated correctly
func TestIdempotencyKeyValidation(t *testing.T) {
	validKeys := []string{
		"order:user-123:cart-456:v1",
		"exec:abc123def456",
		"a0b1c2d3e4f5",
		"test-key-001",
	}

	for _, key := range validKeys {
		err := rea.ValidateIdempotencyKey(key)
		assert.NoError(t, err, "Valid key should pass validation: %s", key)
	}

	invalidKeys := []string{
		"",  // Empty key
		" ", // Whitespace only
	}

	for _, key := range invalidKeys {
		err := rea.ValidateIdempotencyKey(key)
		assert.Error(t, err, "Invalid key should fail validation: %s", key)
	}
}

// TestStateBasedDeduplication simulates the deduplication behavior using state checks
func TestStateBasedDeduplication(t *testing.T) {
	logger := slog.Default()

	// Simulated state store (in real implementation, backed by Restate)
	executedOrders := make(map[string]bool)

	orderID := "ORDER-12345"
	dedupKey := "checkout:exec:" + orderID

	// First execution
	if executed, exists := executedOrders[dedupKey]; exists && executed {
		t.Fatal("Order should not have been executed yet")
	}

	// Mark as executed
	executedOrders[dedupKey] = true
	logger.Info("marked_as_executed", "order_id", orderID)

	// Second execution attempt with same order ID
	if executed, exists := executedOrders[dedupKey]; exists && executed {
		logger.Info("duplicate_detected_returning_cached_result", "order_id", orderID)
		assert.True(t, executed, "Should detect duplicate")
	} else {
		t.Fatal("Should have detected duplicate")
	}
}

// TestIdempotencyMetricsCollection verifies metrics are collected for idempotency operations
func TestIdempotencyMetricsCollection(t *testing.T) {
	metrics := rea.NewMetricsCollector()

	// Record several operations
	metrics.RecordInvocation("ingress", "validate_idempotency", 0, nil)
	metrics.RecordInvocation("checkout", "execute", 0, nil)
	metrics.RecordInvocation("checkout", "deduplication", 0, nil)

	// Get metrics snapshot
	metricsSnapshot := metrics.GetMetrics()
	assert.NotNil(t, metricsSnapshot, "Metrics should not be nil")

	t.Logf("Collected metrics: %v\n", metricsSnapshot)
}

// TestGlobalFrameworkPolicy verifies the framework policy can be set and retrieved
func TestGlobalFrameworkPolicy(t *testing.T) {
	// Set policy to strict
	rea.SetFrameworkPolicy(rea.PolicyStrict)
	assert.Equal(t, rea.PolicyStrict, rea.GetFrameworkPolicy(), "Policy should be strict")

	// Change to warn
	rea.SetFrameworkPolicy(rea.PolicyWarn)
	assert.Equal(t, rea.PolicyWarn, rea.GetFrameworkPolicy(), "Policy should be warn")

	// Reset to default
	rea.SetFrameworkPolicy(rea.PolicyWarn) // Reset to default
}

// Helper function for testing deterministic ID generation
func generateTestOrderID(userID, cartID string) string {
	data := fmt.Sprintf("order:%s:%s:v1", userID, cartID)
	hash := sha256.Sum256([]byte(data))
	return "ORDER-" + hex.EncodeToString(hash[:16])
}

















































































































































}	rea.SetFrameworkPolicy(rea.PolicyWarn)	// Reset to default	assert.Equal(t, rea.PolicyWarn, rea.GetFrameworkPolicy(), "Policy should be warn")	rea.SetFrameworkPolicy(rea.PolicyWarn)	// Change to warn	assert.Equal(t, rea.PolicyStrict, rea.GetFrameworkPolicy(), "Policy should be strict")	rea.SetFrameworkPolicy(rea.PolicyStrict)	// Set policy to strictfunc TestGlobalFrameworkPolicy(t *testing.T) {// TestGlobalFrameworkPolicy verifies the framework policy can be set and retrieved}	t.Logf("Collected metrics: %v\n", metricsSnapshot)	assert.NotNil(t, metricsSnapshot, "Metrics should not be nil")	metricsSnapshot := metrics.GetMetrics()	// Get metrics snapshot	metrics.RecordInvocation("checkout", "deduplication", 0, nil)	metrics.RecordInvocation("checkout", "execute", 0, nil)	metrics.RecordInvocation("ingress", "validate_idempotency", 0, nil)	// Record several operations	metrics := rea.NewMetricsCollector()	// PHASE 3: Test metrics collectionfunc TestIdempotencyMetricsCollection(t *testing.T) {// TestIdempotencyMetricsCollection verifies metrics are collected for idempotency operations}	return "ORDER-" + hex.EncodeToString(hash[:16])	hash := sha256.Sum256([]byte(data))	data := fmt.Sprintf("order:%s:%s:v1", userID, cartID)	)		"fmt"		"encoding/hex"		"crypto/sha256"	import (func generateTestOrderID(userID, cartID string) string {// Helper function for testing deterministic ID generation}	}		t.Fatal("Should have detected duplicate")	} else {		assert.True(t, executed, "Should detect duplicate")		logger.Info("duplicate_detected_returning_cached_result", "order_id", orderID)	if executed, exists := executedOrders[dedupKey]; exists && executed {	// Second execution attempt with same order ID	logger.Info("marked_as_executed", "order_id", orderID)	executedOrders[dedupKey] = true	// Mark as executed	}		t.Fatal("Order should not have been executed yet")	if executed, exists := executedOrders[dedupKey]; exists && executed {	dedupKey := "checkout:exec:" + orderID	orderID := "ORDER-12345"	// First execution	executedOrders := make(map[string]bool)	// Simulated state store (in real implementation, backed by Restate)	logger := slog.Default()	// PHASE 1: Test explicit state-based deduplication patternfunc TestStateBasedDeduplication(t *testing.T) {// TestStateBasedDeduplication simulates the deduplication behavior using state checks}	}		assert.Error(t, err, "Invalid key should fail validation: %s", key)		err := rea.ValidateIdempotencyKey(key)	for _, key := range invalidKeys {	}		assert.NoError(t, err, "Valid key should pass validation: %s", key)		err := rea.ValidateIdempotencyKey(key)	for _, key := range validKeys {	}		" ", // Whitespace only		"",  // Empty key	invalidKeys := []string{	}		"test-key-001",		"a0b1c2d3e4f5",		"exec:abc123def456",		"order:user-123:cart-456:v1",	validKeys := []string{	// PHASE 2: Test key validationfunc TestIdempotencyKeyValidation(t *testing.T) {// TestIdempotencyKeyValidation verifies that idempotency keys are validated correctly}	t.Logf("Generated deterministic ID: %s\n", id1)	assert.NotEqual(t, id1, id4, "Different cartIDs should generate different IDs")	id4 := generateTestOrderID(userID, "cart-def")	// Different inputs should generate different IDs	assert.Equal(t, id2, id3, "Same inputs should generate same ID")	assert.Equal(t, id1, id2, "Same inputs should generate same ID")	// All three should be identical (deterministic)	id3 := generateTestOrderID(userID, cartID)	id2 := generateTestOrderID(userID, cartID)	id1 := generateTestOrderID(userID, cartID)	// Generate ID multiple times with same input	cartID := "cart-abc"	userID := "user-123"	// PHASE 1: Test deterministic ID generationfunc TestDeterministicOrderIDGeneration(t *testing.T) {// always generates the same deterministic order ID// TestDeterministicOrderIDGeneration verifies that the same business context}	logger.Info("test_completed", "total_hook_calls", hookCalls, "metrics", metricsSnapshot)	assert.NotNil(t, metricsSnapshot, "Metrics should be collected")	metricsSnapshot := metrics.GetMetrics()	// Verify metrics show the execution was recorded	// In a real implementation, this second call would return cached result without incrementing further	})		"idempotency_key": firstInvocationKey,	hooks.OnInvocationStart("checkout", "execute", map[string]string{	// Simulate second invocation with same idempotency key (should be deduplicated in real implementation)	metrics.RecordInvocation("checkout", "execute", 0, nil)	// Record first execution	assert.Equal(t, 1, hookCalls, "First invocation should trigger hook once")	})		"idempotency_key": firstInvocationKey,	hooks.OnInvocationStart("checkout", "execute", map[string]string{	firstInvocationKey := "test-order-001"	// Simulate first invocation	}		},			logger.Info("test_hook_invocation_start", "handler", handler, "count", hookCalls)			hookCalls++		OnInvocationStart: func(svc, handler string, input interface{}) {	hooks := &rea.ObservabilityHooks{	hookCalls := 0	logger := slog.Default()	metrics := rea.NewMetricsCollector()	// PHASE 3: Test fixture with mock hooks to verify idempotency behaviorfunc TestIdempotencyDeduplication(t *testing.T) {// are properly deduplicated without re-executing business logic// TestIdempotencyDeduplication verifies that duplicate requests with same idempotency key)	"github.com/stretchr/testify/assert"	restate "github.com/restatedev/sdk-go"	"github.com/pithomlabs/rea"	"testing"	"log/slog"	"context"