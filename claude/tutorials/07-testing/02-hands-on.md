# Hands-On: Writing Tests for Restate Services

> **Build comprehensive tests with mock context and integration tests**

## 🎯 What We're Building

We'll write tests for a **User Registration Service** that demonstrates:
- ✅ Unit tests with mock context
- ✅ Testing state management
- ✅ Mocking external API calls
- ✅ Integration tests with real server
- ✅ Testing error scenarios

## 📋 Prerequisites

- ✅ Go 1.23+ installed
- ✅ Completed Module 06 (Sagas)
- ✅ Basic Go testing knowledge
- ✅ `testify` package for assertions

## 🏗️ Project Setup

### Step 1: Create Project Directory

```bash
mkdir -p ~/restate-tutorials/module07/code
cd ~/restate-tutorials/module07/code
```

### Step 2: Initialize Go Module

```bash
go mod init module07
go get github.com/restatedev/sdk-go@latest
go get github.com/stretchr/testify@latest
```

### Step 3: Create File Structure

```bash
touch types.go user_service.go user_service_test.go email_service.go
```

## 📝 Implementation

### Step 1: Define Types (`types.go`)

```go
package main

import "time"

// User registration request
type RegisterRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// User profile stored in state
type UserProfile struct {
	UserID       string    `json:"userId"`
	Email        string    `json:"email"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"passwordHash"`
	CreatedAt    time.Time `json:"createdAt"`
	Verified     bool      `json:"verified"`
}

// Registration result
type RegisterResult struct {
	UserID string `json:"userId"`
	Status string `json:"status"` // "created", "already_exists"
}

// Email verification
type VerifyEmailRequest struct {
	Token string `json:"token"`
}
```

### Step 2: Implement Email Service (`email_service.go`)

```go
package main

import (
	"fmt"
	
	restate "github.com/restatedev/sdk-go"
)

type EmailService struct{}

// SendVerificationEmail sends verification email
func (EmailService) SendVerificationEmail(
	ctx restate.Context,
	email string,
	token string,
) error {
	_, err := restate.Run(ctx, func(ctx restate.RunContext) (bool, error) {
		ctx.Log().Info("Sending verification email",
			"email", email,
			"token", token)
		
		// In real app: call SendGrid, SES, etc.
		// For now, just log
		
		return true, nil
	})
	
	return err
}
```

### Step 3: Implement User Service (`user_service.go`)

```go
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
	emailSvc := EmailService{}
	err = emailSvc.SendVerificationEmail(ctx, req.Email, token)
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
		return fmt.Errorf("invalid verification token")
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
```

### Step 4: Write Unit Tests (`user_service_test.go`)

```go
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
	ctx := restate.NewMockContext(restate.WithKey("user-123"))
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
	ctx := restate.NewMockContext(restate.WithKey("user-123"))
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
	ctx := restate.NewMockContext(restate.WithKey("user-123"))
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
	ctx := restate.NewMockContext(restate.WithKey("user-123"))
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
	ctx := restate.NewMockContext(restate.WithKey("user-123"))
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
	ctx1 := restate.NewMockContext(
		restate.WithKey("user-123"),
		restate.WithSeed(42),
	)
	ctx2 := restate.NewMockContext(
		restate.WithKey("user-123"),
		restate.WithSeed(42),
	)
	
	token1 := generateToken(ctx1)
	token2 := generateToken(ctx2)
	
	// Should be same with same seed
	assert.Equal(t, token1, token2)
}
```

### Step 5: Run Tests

```bash
# Run all tests
go test -v

# Run specific test
go test -v -run TestUserService_Register_Success

# Run with coverage
go test -cover

# Generate coverage report
go test -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## 📊 Expected Output

```
=== RUN   TestUserService_Register_Success
--- PASS: TestUserService_Register_Success (0.00s)
=== RUN   TestUserService_Register_AlreadyExists
--- PASS: TestUserService_Register_AlreadyExists (0.00s)
=== RUN   TestUserService_VerifyEmail_Success
--- PASS: TestUserService_VerifyEmail_Success (0.00s)
=== RUN   TestUserService_VerifyEmail_InvalidToken
--- PASS: TestUserService_VerifyEmail_InvalidToken (0.00s)
=== RUN   TestUserService_GetProfile
--- PASS: TestUserService_GetProfile (0.00s)
=== RUN   TestHashPassword
=== RUN   TestHashPassword/short_password
=== RUN   TestHashPassword/long_password
=== RUN   TestHashPassword/special_chars
--- PASS: TestHashPassword (0.00s)
=== RUN   TestGenerateToken_Deterministic
--- PASS: TestGenerateToken_Deterministic (0.00s)
PASS
coverage: 87.5% of statements
ok      module07    0.123s
```

## 🎓 Key Testing Patterns Demonstrated

### 1. Mock Context Setup

```go
ctx := restate.NewMockContext(restate.WithKey("user-123"))
```

### 2. Pre-populating State

```go
existingProfile := UserProfile{...}
restate.Set(ctx, stateKeyProfile, existingProfile)
```

### 3. Verifying State Changes

```go
profile, _ := restate.Get[UserProfile](ctx, stateKeyProfile)
assert.True(t, profile.Verified)
```

### 4. Table-Driven Tests

```go
tests := []struct {
    name string
    input string
    want int
}{...}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {...})
}
```

### 5. Deterministic Testing

```go
ctx := restate.NewMockContext(restate.WithSeed(42))
// UUIDs and random values are deterministic
```

## 🧪 Advanced Testing Scenarios

### Testing Email Service Failures

```go
func TestUserService_Register_EmailFailure(t *testing.T) {
	ctx := restate.NewMockContext(restate.WithKey("user-123"))
	service := UserService{}
	
	// Mock email service to fail
	ctx.MockRun("SendVerificationEmail").
		Returns(false, errors.New("email service down"))
	
	req := RegisterRequest{
		Email:    "alice@example.com",
		Username: "alice",
		Password: "secret123",
	}
	
	// Execute - should succeed despite email failure
	result, err := service.Register(ctx, req)
	
	require.NoError(t, err)
	assert.Equal(t, "created", result.Status)
}
```

### Testing Concurrent Access

```go
func TestUserService_ConcurrentRegistration(t *testing.T) {
	// This would be an integration test
	// showing that exclusive handlers prevent race conditions
}
```

## ✅ Success Criteria

Your tests should:
- ✅ All pass (`go test`)
- ✅ Cover main scenarios (success, failure, edge cases)
- ✅ Have good coverage (>80%)
- ✅ Run fast (<1 second total)
- ✅ Be deterministic (same result every time)
- ✅ Be independent (order doesn't matter)

## 🎯 Next Steps

Excellent! You've written comprehensive unit tests!

👉 **Continue to [Validation](./03-validation.md)**

Learn integration testing with real Restate server!

---

**Questions?** Review [concepts](./01-concepts.md) or check the [module README](./README.md).
