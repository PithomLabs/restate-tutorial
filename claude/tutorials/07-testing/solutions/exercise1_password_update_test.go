package main

import (
	"testing"

	restate "github.com/restatedev/sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Exercise 1 Tests

func TestUserService_UpdatePassword_Success(t *testing.T) {
	// Setup
	ctx := restate.NewMockObjectContext(restate.WithKey("user-123"))
	service := UserService{}

	// Pre-populate with known password
	profile := UserProfile{
		UserID:       "user-123",
		Email:        "alice@example.com",
		PasswordHash: hashPassword("oldpass"),
	}
	restate.Set(ctx, stateKeyProfile, profile)

	// Test update
	req := UpdatePasswordRequest{
		OldPassword: "oldpass",
		NewPassword: "newpass123",
	}
	err := service.UpdatePassword(ctx, req)

	// Verify
	require.NoError(t, err)

	// Verify password actually updated
	updatedProfile, _ := restate.Get[UserProfile](ctx, stateKeyProfile)
	assert.Equal(t, hashPassword("newpass123"), updatedProfile.PasswordHash)
	assert.NotEqual(t, hashPassword("oldpass"), updatedProfile.PasswordHash)
}

func TestUserService_UpdatePassword_WrongOldPassword(t *testing.T) {
	// Setup
	ctx := restate.NewMockObjectContext(restate.WithKey("user-123"))
	service := UserService{}

	// Pre-populate
	profile := UserProfile{
		UserID:       "user-123",
		PasswordHash: hashPassword("correctpass"),
	}
	restate.Set(ctx, stateKeyProfile, profile)

	// Try with wrong old password
	req := UpdatePasswordRequest{
		OldPassword: "wrongpass",
		NewPassword: "newpass123",
	}
	err := service.UpdatePassword(ctx, req)

	// Verify
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid old password")
	assert.True(t, restate.IsTerminalError(err), "Should be terminal error")

	// Verify password unchanged
	unchangedProfile, _ := restate.Get[UserProfile](ctx, stateKeyProfile)
	assert.Equal(t, hashPassword("correctpass"), unchangedProfile.PasswordHash)
}

func TestUserService_UpdatePassword_UserNotFound(t *testing.T) {
	// Setup - no pre-existing user
	ctx := restate.NewMockObjectContext(restate.WithKey("user-123"))
	service := UserService{}

	// Try to update password for non-existent user
	req := UpdatePasswordRequest{
		OldPassword: "oldpass",
		NewPassword: "newpass",
	}
	err := service.UpdatePassword(ctx, req)

	// Verify - should fail gracefully
	require.Error(t, err)
}
