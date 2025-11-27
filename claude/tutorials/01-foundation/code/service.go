package main

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
)

// GreetingService is a Basic Service (stateless)
type GreetingService struct{}

// GreetRequest defines our input structure
type GreetRequest struct {
	Name       string `json:"name"`
	ShouldFail bool   `json:"shouldFail"` // For testing retry behavior
}

// GreetResponse defines our output structure
type GreetResponse struct {
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

// Greet is our main handler function
// It demonstrates:
// - Context logging (no duplication on replay)
// - Deterministic UUID generation
// - Error handling (Terminal vs Retriable)
func (GreetingService) Greet(
	ctx restate.Context,
	req GreetRequest,
) (GreetResponse, error) {
	// Log the request using context logger
	// This won't duplicate on replay!
	ctx.Log().Info("Processing greeting request",
		"name", req.Name,
		"shouldFail", req.ShouldFail)

	// Input validation - use Terminal error
	if req.Name == "" {
		ctx.Log().Warn("Validation failed: empty name")
		return GreetResponse{}, restate.TerminalError(
			fmt.Errorf("name cannot be empty"),
			400, // HTTP status code
		)
	}

	// Simulate a transient failure for testing retry behavior
	if req.ShouldFail {
		ctx.Log().Error("Simulating transient failure")
		// Regular error - Restate will retry this
		return GreetResponse{}, fmt.Errorf("simulated transient failure")
	}

	// Generate a deterministic UUID
	// This will be the same UUID on replay for the same invocation
	requestID := restate.UUID(ctx).String()

	// Create response
	response := GreetResponse{
		Message:   fmt.Sprintf("Hello, %s! You're awesome!", req.Name),
		Timestamp: requestID,
	}

	ctx.Log().Info("Greeting generated successfully",
		"requestID", requestID,
		"name", req.Name)

	return response, nil
}
