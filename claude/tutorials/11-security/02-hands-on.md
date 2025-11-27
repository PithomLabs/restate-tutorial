# Hands-On: Secure User Management System

> **Build a secure Restate application with authentication and authorization**

## 🎯 What We're Building

A **secure user management system** with:
- 🔐 JWT authentication
- 👤 Role-based access control (RBAC)
- 🔒 Encrypted sensitive data
- ✅ Input validation
- 📝 Audit logging

## 🏗️ Project Setup

```bash
mkdir -p ~/restate-tutorials/11-security/code
cd ~/restate-tutorials/11-security/code
go mod init security
go get github.com/restatedev/sdk-go@latest
go get github.com/golang-jwt/jwt/v5
go get golang.org/x/crypto/bcrypt
```

## 📝 Implementation

### Step 1: Types and Auth (`types.go`)

```go
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // Never serialize
	Role         Role      `json:"role"`
	CreatedAt    time.Time `json:"createdAt"`
}

type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct{
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token  string `json:"token"`
	UserID string `json:"userId"`
}

type AuthRequest struct {
	Token string `json:"token"`
}

// JWT Claims
type Claims struct {
	UserID string `json:"userId"`
	Email  string `json:"email"`
	Role   Role   `json:"role"`
	jwt.RegisteredClaims
}

// Auth helpers
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func checkPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func generateJWT(userID, email string, role Role) (string, error) {
	secret := []byte(os.Getenv("JWT_SECRET"))
	if len(secret) == 0 {
		secret = []byte("default-secret-change-in-production")
	}

	claims := Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func validateJWT(tokenString string) (*Claims, error) {
	secret := []byte(os.Getenv("JWT_SECRET"))
	if len(secret) == 0 {
		secret = []byte("default-secret-change-in-production")
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{},
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return secret, nil
		})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// Input validation
func validateEmail(email string) error {
	if len(email) == 0 {
		return fmt.Errorf("email required")
	}
	if len(email) > 255 {
		return fmt.Errorf("email too long")
	}
	// Simple validation
	if len(email) < 3 || !contains(email, "@") {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	if len(password) > 100 {
		return fmt.Errorf("password too long")
	}
	return nil
}

func contains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
```

### Step 2: User Service (`user_service.go`)

```go
package main

import (
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
)

type UserService struct{}

// Register creates a new user account
func (UserService) Register(
	ctx restate.ObjectContext,
	req RegisterRequest,
) (User, error) {
	userID := restate.Key(ctx)

	ctx.Log().Info("User registration started",
		"userId", userID,
		"email", req.Email)

	// Check if already registered
	existing, _ := restate.Get[*User](ctx, "user")
	if existing != nil {
		ctx.Log().Warn("User already registered",
			"userId", userID)
		return *existing, nil
	}

	// Validate input
	if err := validateEmail(req.Email); err != nil {
		ctx.Log().Error("Invalid email",
			"userId", userID,
			"error", err)
		return User{}, restate.TerminalError(err, 400)
	}

	if err := validatePassword(req.Password); err != nil {
		ctx.Log().Error("Invalid password",
			"userId", userID,
			"error", err)
		return User{}, restate.TerminalError(err, 400)
	}

	// Hash password (journaled for determinism)
	passwordHash, err := restate.Run(ctx,
		func(ctx restate.RunContext) (string, error) {
			return hashPassword(req.Password)
		})
	if err != nil {
		return User{}, err
	}

	// Create user
	user := User{
		ID:           userID,
		Email:        req.Email,
		PasswordHash: passwordHash,
		Role:         RoleUser, // Default role
		CreatedAt:    time.Now(),
	}

	restate.Set(ctx, "user", user)

	ctx.Log().Info("User registered successfully",
		"userId", userID,
		"email", user.Email)

	return user, nil
}

// Login authenticates user and returns JWT
func (UserService) Login(
	ctx restate.ObjectSharedContext,
	req LoginRequest,
) (LoginResponse, error) {
	userID := restate.Key(ctx)

	ctx.Log().Info("Login attempt",
		"userId", userID,
		"email", req.Email)

	// Get user
	user, err := restate.Get[User](ctx, "user")
	if err != nil {
		ctx.Log().Warn("User not found",
			"userId", userID)
		return LoginResponse{}, restate.TerminalError(
			fmt.Errorf("invalid credentials"), 401)
	}

	// Verify password (non-deterministic, but read-only so OK in shared handler)
	if !checkPassword(req.Password, user.PasswordHash) {
		ctx.Log().Warn("Invalid password",
			"userId", userID)
		return LoginResponse{}, restate.TerminalError(
			fmt.Errorf("invalid credentials"), 401)
	}

	// Generate JWT
	token, err := generateJWT(user.ID, user.Email, user.Role)
	if err != nil {
		ctx.Log().Error("Failed to generate token",
			"userId", userID,
			"error", err)
		return LoginResponse{}, err
	}

	ctx.Log().Info("Login successful",
		"userId", userID)

	return LoginResponse{
		Token:  token,
		UserID: user.ID,
	}, nil
}

// GetProfile returns user profile (authentication required)
func (UserService) GetProfile(
	ctx restate.ObjectSharedContext,
	req AuthRequest,
) (User, error) {
	userID := restate.Key(ctx)

	// Validate JWT
	claims, err := validateJWT(req.Token)
	if err != nil {
		ctx.Log().Warn("Invalid token",
			"userId", userID,
			"error", err)
		return User{}, restate.TerminalError(
			fmt.Errorf("unauthorized"), 401)
	}

	// Check authorization: user can only access their own profile
	if claims.UserID != userID {
		ctx.Log().Warn("Unauthorized access attempt",
			"userId", userID,
			"attemptedBy", claims.UserID)
		return User{}, restate.TerminalError(
			fmt.Errorf("forbidden"), 403)
	}

	// Get profile
	user, err := restate.Get[User](ctx, "user")
	if err != nil {
		return User{}, restate.TerminalError(
			fmt.Errorf("user not found"), 404)
	}

	ctx.Log().Info("Profile accessed",
		"userId", userID)

	// Don't return password hash
	user.PasswordHash = ""
	return user, nil
}

// PromoteToAdmin grants admin role (admin-only operation)
func (UserService) PromoteToAdmin(
	ctx restate.ObjectContext,
	req AuthRequest,
) error {
	targetUserID := restate.Key(ctx)

	// Validate JWT
	claims, err := validateJWT(req.Token)
	if err != nil {
		return restate.TerminalError(fmt.Errorf("unauthorized"), 401)
	}

	// Check authorization: only admins can promote
	if claims.Role != RoleAdmin {
		ctx.Log().Warn("Unauthorized promotion attempt",
			"targetUserId", targetUserID,
			"attemptedBy", claims.UserID,
			"attemptedByRole", claims.Role)
		return restate.TerminalError(fmt.Errorf("forbidden"), 403)
	}

	// Get user
	user, err := restate.Get[User](ctx, "user")
	if err != nil {
		return restate.TerminalError(fmt.Errorf("user not found"), 404)
	}

	// Promote to admin
	user.Role = RoleAdmin
	restate.Set(ctx, "user", user)

	ctx.Log().Info("User promoted to admin",
		"userId", targetUserID,
		"promotedBy", claims.UserID)

	return nil
}
```

### Step 3: Main Server (`main.go`)

```go
package main

import (
	"context"
	"fmt"
	"log"

	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/server"
)

func main() {
	restateServer := server.NewRestate()

	if err := restateServer.Bind(restate.Reflect(UserService{})); err != nil {
		log.Fatal("Failed to bind service:", err)
	}

	fmt.Println("🔐 Secure User Management System starting on :9090")
	fmt.Println("")
	fmt.Println("📝 Services:")
	fmt.Println("  👤 UserService - Register, Login, GetProfile, PromoteToAdmin")
	fmt.Println("")
	fmt.Println("🔒 Security Features:")
	fmt.Println("  ✅ JWT authentication")
	fmt.Println("  ✅ Password hashing (bcrypt)")
	fmt.Println("  ✅ Role-based access control")
	fmt.Println("  ✅ Input validation")
	fmt.Println("  ✅ Audit logging")
	fmt.Println("")

	if err := restateServer.Start(context.Background(), ":9090"); err != nil {
		log.Fatal("Server error:", err)
	}
}
```

### Step 4: Go Module (`go.mod`)

```go
module security

go 1.23

require (
	github.com/golang-jwt/jwt/v5 v5.2.0
	github.com/restatedev/sdk-go v0.13.1
	golang.org/x/crypto v0.18.0
)
```

## 🚀 Running the System

### 1. Start Restate

```bash
docker run --name restate_dev --rm \
  -p 8080:8080 -p 9070:9070 -p 9091:9091 \
  --add-host=host.docker.internal:host-gateway \
  docker.io/restatedev/restate:latest
```

### 2. Start Service

```bash
# Set JWT secret
export JWT_SECRET="your-secret-key-change-in-production"

go mod tidy
go run .
```

### 3. Register Service

```bash
curl -X POST http://localhost:9070/deployments \
  -H 'Content-Type: application/json' \
  -d '{"uri": "http://host.docker.internal:9090"}'
```

## 🧪 Testing Security

### Register User

```bash
curl -X POST http://localhost:8080/UserService/user-001/Register \
  -H 'Content-Type: application/json' \
  -d '{
    "email": "alice@example.com",
    "password": "SecurePassword123"
  }'
```

### Login

```bash
curl -X POST http://localhost:8080/UserService/user-001/Login \
  -H 'Content-Type: application/json' \
  -d '{
    "email": "alice@example.com",
    "password": "SecurePassword123"
  }'

# Save the token from response
TOKEN="eyJhbGc..."
```

### Get Profile (Authenticated)

```bash
curl -X POST http://localhost:8080/UserService/user-001/GetProfile \
  -H 'Content-Type: application/json' \
  -d "{\"token\": \"$TOKEN\"}"
```

### Try Unauthorized Access

```bash
# Try to access another user's profile (should fail with 403)
curl -X POST http://localhost:8080/UserService/user-002/GetProfile \
  -H 'Content-Type: application/json' \
  -d "{\"token\": \"$TOKEN\"}"
```

## 🎓 What You Learned

1. **JWT Authentication** - Token-based auth
2. **Password Hashing** - bcrypt for security
3. **Authorization** - Role-based access control
4. **Input Validation** - Prevent bad data
5. **Audit Logging** - Track security events

## 🚀 Next Steps

👉 **Continue to [Validation](./03-validation.md)**
