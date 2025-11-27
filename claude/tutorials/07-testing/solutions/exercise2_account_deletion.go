package main

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
)

// Exercise 2 Solution: Account Deletion

// Add to user_service.go
func (UserService) DeleteAccount(
	ctx restate.ObjectContext,
	password string,
) error {
	userID := restate.Key(ctx)

	ctx.Log().Info("Deleting account", "userId", userID)

	// Get user profile to verify password
	profile, err := restate.Get[UserProfile](ctx, stateKeyProfile)
	if err != nil {
		return restate.TerminalError(
			fmt.Errorf("user not found"), 404)
	}

	// Verify password
	if hashPassword(password) != profile.PasswordHash {
		return restate.TerminalError(
			fmt.Errorf("invalid password"), 403)
	}

	// Clear all state - account deleted
	restate.ClearAll(ctx)

	ctx.Log().Info("Account deleted successfully", "userId", userID)

	return nil
}
