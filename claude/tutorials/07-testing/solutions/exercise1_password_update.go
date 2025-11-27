package main

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
)

// Exercise 1 Solution: Password Update

// Add to types.go
type UpdatePasswordRequest struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

// Add to user_service.go
func (UserService) UpdatePassword(
	ctx restate.ObjectContext,
	req UpdatePasswordRequest,
) error {
	userID := restate.Key(ctx)

	ctx.Log().Info("Updating password", "userId", userID)

	// Get user profile
	profile, err := restate.Get[UserProfile](ctx, stateKeyProfile)
	if err != nil {
		return err
	}

	// Verify old password matches
	if hashPassword(req.OldPassword) != profile.PasswordHash {
		return restate.TerminalError(
			fmt.Errorf("invalid old password"), 401)
	}

	// Update to new password
	profile.PasswordHash = hashPassword(req.NewPassword)
	restate.Set(ctx, stateKeyProfile, profile)

	ctx.Log().Info("Password updated successfully", "userId", userID)

	return nil
}
