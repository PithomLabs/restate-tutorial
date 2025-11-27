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
