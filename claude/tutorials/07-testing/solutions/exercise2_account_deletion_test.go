package main

import (
	"testing"

	restate "github.com/restatedev/sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Exercise 2 Tests

func TestUserService_DeleteAccount_Success(t *testing.T) {
	// Setup
	ctx := restate.NewMockObjectContext(restate.WithKey("user-123"))
	service := UserService{}

	// Pre-populate user with state
	profile := UserProfile{
		UserID:       "user-123",
		Email:        "alice@example.com",
		PasswordHash: hashPassword("password123"),
		Verified:     true,
	}
	restate.Set(ctx, stateKeyProfile, profile)
	restate.Set(ctx, stateKeyToken, "some-token")

	// Verify state exists before deletion
	keys, _ := restate.Keys(ctx)
	assert.NotEmpty(t, keys, "Should have state before deletion")

	// Delete account
	err := service.DeleteAccount(ctx, "password123")

	// Verify
	require.NoError(t, err)

	// Verify all state cleared
	keysAfter, _ := restate.Keys(ctx)
	assert.Empty(t, keysAfter, "All state should be cleared")

	// Verify profile is gone
	deletedProfile, err := restate.Get[*UserProfile](ctx, stateKeyProfile)
	require.NoError(t, err)
	assert.Nil(t, deletedProfile, "Profile should be nil after deletion")
}

func TestUserService_DeleteAccount_WrongPassword(t *testing.T) {
	// Setup
	ctx := restate.NewMockObjectContext(restate.WithKey("user-123"))
	service := UserService{}

	// Pre-populate
	profile := UserProfile{
		UserID:       "user-123",
		PasswordHash: hashPassword("correctpass"),
		Verified:     true,
	}
	restate.Set(ctx, stateKeyProfile, profile)

	// Try to delete with wrong password
	err := service.DeleteAccount(ctx, "wrongpass")

	// Verify
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid password")
	assert.True(t, restate.IsTerminalError(err))

	// Verify state unchanged
	keys, _ := restate.Keys(ctx)
	assert.NotEmpty(t, keys, "State should not be deleted")

	stillThere, _ := restate.Get[UserProfile](ctx, stateKeyProfile)
	assert.True(t, stillThere.Verified, "Profile should still exist")
}

func TestUserService_DeleteAccount_NoUser(t *testing.T) {
	// Setup - no existing user
	ctx := restate.NewMockObjectContext(restate.WithKey("user-123"))
	service := UserService{}

	// Try to delete non-existent account
	err := service.DeleteAccount(ctx, "anypassword")

	// Verify
	require.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")
	assert.True(t, restate.IsTerminalError(err))
}
