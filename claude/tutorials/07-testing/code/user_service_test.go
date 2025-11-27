package main

import (
	"testing"

	restate "github.com/restatedev/sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test successful user registration
func TestUserService_Register_Success(t *testing.T) {
	// Setup
	ctx := restate.NewMockObjectContext(restate.WithKey("user-123"))
	service := UserService{}

	req := RegisterRequest{
		Email:    "alice@example.com",
		Username: "alice",
		Password: "secret123",
	}

	// Execute
	result, err := service.Register(ctx, req)

	// Verify
	require.NoError(t, err)
	assert.Equal(t, "user-123", result.UserID)
	assert.Equal(t, "created", result.Status)

	// Verify state was saved
	profile, err := restate.Get[UserProfile](ctx, stateKeyProfile)
	require.NoError(t, err)
	assert.Equal(t, "alice@example.com", profile.Email)
	assert.Equal(t, "alice", profile.Username)
	assert.False(t, profile.Verified)
	assert.NotEmpty(t, profile.PasswordHash)
}

// Test duplicate registration
func TestUserService_Register_AlreadyExists(t *testing.T) {
	// Setup
	ctx := restate.NewMockObjectContext(restate.WithKey("user-123"))
	service := UserService{}

	// Pre-populate existing user
	existingProfile := UserProfile{
		UserID:   "user-123",
		Email:    "alice@example.com",
		Username: "alice",
		Verified: true,
	}
	restate.Set(ctx, stateKeyProfile, existingProfile)

	req := RegisterRequest{
		Email:    "alice@example.com",
		Username: "alice",
		Password: "secret123",
	}

	// Execute
	result, err := service.Register(ctx, req)

	// Verify
	require.NoError(t, err)
	assert.Equal(t, "already_exists", result.Status)

	// Verify state unchanged
	profile, _ := restate.Get[UserProfile](ctx, stateKeyProfile)
	assert.True(t, profile.Verified) // Still verified
}

// Test email verification
func TestUserService_VerifyEmail_Success(t *testing.T) {
	// Setup
	ctx := restate.NewMockObjectContext(restate.WithKey("user-123"))
	service := UserService{}

	// Pre-populate user and token
	profile := UserProfile{
		UserID:   "user-123",
		Email:    "alice@example.com",
		Verified: false,
	}
	restate.Set(ctx, stateKeyProfile, profile)
	restate.Set(ctx, stateKeyToken, "valid-token-123")

	req := VerifyEmailRequest{
		Token: "valid-token-123",
	}

	// Execute
	err := service.VerifyEmail(ctx, req)

	// Verify
	require.NoError(t, err)

	// Verify state updated
	updatedProfile, _ := restate.Get[UserProfile](ctx, stateKeyProfile)
	assert.True(t, updatedProfile.Verified)
}

// Test invalid verification token
func TestUserService_VerifyEmail_InvalidToken(t *testing.T) {
	// Setup
	ctx := restate.NewMockObjectContext(restate.WithKey("user-123"))
	service := UserService{}

	profile := UserProfile{
		UserID:   "user-123",
		Email:    "alice@example.com",
		Verified: false,
	}
	restate.Set(ctx, stateKeyProfile, profile)
	restate.Set(ctx, stateKeyToken, "valid-token-123")

	req := VerifyEmailRequest{
		Token: "wrong-token",
	}

	// Execute
	err := service.VerifyEmail(ctx, req)

	// Verify
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid verification token")

	// Verify state unchanged
	updatedProfile, _ := restate.Get[UserProfile](ctx, stateKeyProfile)
	assert.False(t, updatedProfile.Verified)
}

// Test get profile
func TestUserService_GetProfile(t *testing.T) {
	// Setup
	ctx := restate.NewMockObjectSharedContext(restate.WithKey("user-123"))
	service := UserService{}

	expectedProfile := UserProfile{
		UserID:   "user-123",
		Email:    "alice@example.com",
		Username: "alice",
		Verified: true,
	}
	restate.Set(ctx, stateKeyProfile, expectedProfile)

	// Execute
	profile, err := service.GetProfile(ctx, restate.Void{})

	// Verify
	require.NoError(t, err)
	assert.Equal(t, expectedProfile.Email, profile.Email)
	assert.Equal(t, expectedProfile.Username, profile.Username)
	assert.True(t, profile.Verified)
}

// Table-driven test for password hashing
func TestHashPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantLen  int
	}{
		{"short password", "abc", 64},
		{"long password", "verylongpasswordwithmanycharacters", 64},
		{"special chars", "p@ssw0rd!#$%", 64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := hashPassword(tt.password)
			assert.Len(t, hash, tt.wantLen)

			// Same password should produce same hash
			hash2 := hashPassword(tt.password)
			assert.Equal(t, hash, hash2)
		})
	}
}

// Test deterministic token generation with seeded context
func TestGenerateToken_Deterministic(t *testing.T) {
	// Create two contexts with same seed
	ctx1 := restate.NewMockObjectContext(
		restate.WithKey("user-123"),
		restate.WithRandomSeed(42),
	)
	ctx2 := restate.NewMockObjectContext(
		restate.WithKey("user-123"),
		restate.WithRandomSeed(42),
	)

	token1 := generateToken(ctx1)
	token2 := generateToken(ctx2)

	// Should be same with same seed
	assert.Equal(t, token1, token2)
}
