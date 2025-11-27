// Exercise 2 Solution: Language Support

package main

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
)

type GreetRequest struct {
	Name       string `json:"name"`
	Language   string `json:"language"`
	ShouldFail bool   `json:"shouldFail"`
}

type GreetResponse struct {
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

type GreetingService struct{}

func (GreetingService) Greet(
	ctx restate.Context,
	req GreetRequest,
) (GreetResponse, error) {
	ctx.Log().Info("Processing greeting",
		"name", req.Name,
		"language", req.Language)

	// Validation
	if req.Name == "" {
		return GreetResponse{}, restate.TerminalError(
			fmt.Errorf("name cannot be empty"),
			400,
		)
	}

	// Default to English if not specified
	if req.Language == "" {
		req.Language = "en"
	}

	// Select greeting based on language
	var greeting string
	switch req.Language {
	case "en":
		greeting = fmt.Sprintf("Hello, %s!", req.Name)
	case "es":
		greeting = fmt.Sprintf("¡Hola, %s!", req.Name)
	case "fr":
		greeting = fmt.Sprintf("Bonjour, %s!", req.Name)
	case "de":
		greeting = fmt.Sprintf("Hallo, %s!", req.Name)
	default:
		return GreetResponse{}, restate.TerminalError(
			fmt.Errorf("unsupported language: %s (supported: en, es, fr, de)", req.Language),
			400,
		)
	}

	requestID := restate.UUID(ctx).String()

	return GreetResponse{
		Message:   greeting,
		Timestamp: requestID,
	}, nil
}
