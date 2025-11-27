package main

import (
	"fmt"
	"strings"
	"testing"

	restate "github.com/restatedev/sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Exercise 5 Solution: Table-Driven Error Scenarios

func TestUserService_Register_ValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		req         RegisterRequest
		wantErr     bool
		errContains string
	}{
		{
			name: "valid registration",
			req: RegisterRequest{
				Email:    "alice@example.com",
				Username: "alice",
				Password: "password123",
			},
			wantErr: false,
		},
		{
			name: "empty email",
			req: RegisterRequest{
				Email:    "",
				Username: "alice",
				Password: "password123",
			},
			wantErr:     true,
			errContains: "email required",
		},
		{
			name: "invalid email format",
			req: RegisterRequest{
				Email:    "not-an-email",
				Username: "alice",
				Password: "password123",
			},
			wantErr:     true,
			errContains: "invalid email",
		},
		{
			name: "empty username",
			req: RegisterRequest{
				Email:    "alice@example.com",
				Username: "",
				Password: "password123",
			},
			wantErr:     true,
			errContains: "username required",
		},
		{
			name: "short password",
			req: RegisterRequest{
				Email:    "alice@example.com",
				Username: "alice",
				Password: "short",
			},
			wantErr:     true,
			errContains: "password must be at least 8 characters",
		},
		{
			name: "username too short",
			req: RegisterRequest{
				Email:    "alice@example.com",
				Username: "ab",
				Password: "password123",
			},
			wantErr:     true,
			errContains: "username must be at least 3 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := restate.NewMockObjectContext(restate.WithKey("user-test"))
			service := UserServiceWithValidation{}

			_, err := service.Register(ctx, tt.req)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.True(t, restate.IsTerminalError(err),
					"Validation errors should be terminal")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// Enhanced UserService with validation
type UserServiceWithValidation struct{}

func (UserServiceWithValidation) Register(
	ctx restate.ObjectContext,
	req RegisterRequest,
) (RegisterResult, error) {
	userID := restate.Key(ctx)

	// Validate email
	if req.Email == "" {
		return RegisterResult{}, restate.TerminalError(
			fmt.Errorf("email required"), 400)
	}
	if !strings.Contains(req.Email, "@") {
		return RegisterResult{}, restate.TerminalError(
			fmt.Errorf("invalid email format"), 400)
	}

	// Validate username
	if req.Username == "" {
		return RegisterResult{}, restate.TerminalError(
			fmt.Errorf("username required"), 400)
	}
	if len(req.Username) < 3 {
		return RegisterResult{}, restate.TerminalError(
			fmt.Errorf("username must be at least 3 characters"), 400)
	}

	// Validate password
	if len(req.Password) < 8 {
		return RegisterResult{}, restate.TerminalError(
			fmt.Errorf("password must be at least 8 characters"), 400)
	}

	// ... rest of registration logic same as UserService.Register

	// Check if user already exists
	existingProfile, err := restate.Get[*UserProfile](ctx, stateKeyProfile)
	if err != nil {
		return RegisterResult{}, err
	}

	if existingProfile != nil {
		return RegisterResult{
			UserID: userID,
			Status: "already_exists",
		}, nil
	}

	// Create user profile
	profile := UserProfile{
		UserID:       userID,
		Email:        req.Email,
		Username:     req.Username,
		PasswordHash: hashPassword(req.Password),
		Verified:     false,
	}

	restate.Set(ctx, stateKeyProfile, profile)

	return RegisterResult{
		UserID: userID,
		Status: "created",
	}, nil
}
