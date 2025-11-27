package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
)

type UserService struct{}

const (
	stateKeyProfile = "profile"
	stateKeyToken   = "verification_token"
)

// Register creates a new user account
func (UserService) Register(
	ctx restate.ObjectContext,
	req RegisterRequest,
) (RegisterResult, error) {
	userID := restate.Key(ctx)

	ctx.Log().Info("Registering user", "userId", userID, "email", req.Email)

	// Check if user already exists
	existingProfile, err := restate.Get[*UserProfile](ctx, stateKeyProfile)
	if err != nil {
		return RegisterResult{}, err
	}

	if existingProfile != nil {
		ctx.Log().Warn("User already exists", "userId", userID)
		return RegisterResult{
			UserID: userID,
			Status: "already_exists",
		}, nil
	}

	// Hash password
	passwordHash := hashPassword(req.Password)

	// Create user profile
	profile := UserProfile{
		UserID:       userID,
		Email:        req.Email,
		Username:     req.Username,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now(),
		Verified:     false,
	}

	// Save to state
	restate.Set(ctx, stateKeyProfile, profile)

	// Generate verification token
	token := generateToken(ctx)
	restate.Set(ctx, stateKeyToken, token)

	// Send verification email
	err = restate.Service[error](ctx, "EmailService", "SendVerificationEmail").
		Request(struct {
			Email string `json:"email"`
			Token string `json:"token"`
		}{
			Email: req.Email,
			Token: token,
		})
	if err != nil {
		ctx.Log().Error("Failed to send verification email", "error", err)
		// Don't fail registration if email fails
	}

	ctx.Log().Info("User registered successfully", "userId", userID)

	return RegisterResult{
		UserID: userID,
		Status: "created",
	}, nil
}

// VerifyEmail verifies user's email with token
func (UserService) VerifyEmail(
	ctx restate.ObjectContext,
	req VerifyEmailRequest,
) error {
	userID := restate.Key(ctx)

	ctx.Log().Info("Verifying email", "userId", userID)

	// Get stored token
	storedToken, err := restate.Get[string](ctx, stateKeyToken)
	if err != nil {
		return err
	}

	// Validate token
	if storedToken != req.Token {
		return restate.TerminalError(fmt.Errorf("invalid verification token"), 400)
	}

	// Update profile
	profile, err := restate.Get[UserProfile](ctx, stateKeyProfile)
	if err != nil {
		return err
	}

	profile.Verified = true
	restate.Set(ctx, stateKeyProfile, profile)

	ctx.Log().Info("Email verified", "userId", userID)

	return nil
}

// GetProfile returns user profile
func (UserService) GetProfile(
	ctx restate.ObjectSharedContext,
	_ restate.Void,
) (UserProfile, error) {
	userID := restate.Key(ctx)

	profile, err := restate.Get[UserProfile](ctx, stateKeyProfile)
	if err != nil {
		return UserProfile{}, err
	}

	ctx.Log().Info("Retrieved profile", "userId", userID)

	return profile, nil
}

// Helper functions
func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

func generateToken(ctx restate.ObjectContext) string {
	return restate.UUID(ctx).String()
}
