package main

import "time"

// User represents a marketplace user account with seller and admin status flags.
type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	FullName  *string   `json:"full_name,omitempty"`
	AvatarURL *string   `json:"avatar_url,omitempty"`
	IsSeller  bool      `json:"is_seller"`
	IsAdmin   bool      `json:"is_admin"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// RegisterRequest is the payload for user account creation.
type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	IsSeller bool   `json:"is_seller"`
}

// LoginRequest is the payload for user authentication.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Remember bool   `json:"remember"` // If true, extend refresh token lifetime to 30 days
}

// RefreshRequest is the payload to refresh an expired access token.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// AuthResponse contains the JWT tokens and user info after successful login or registration.
type AuthResponse struct {
	AccessToken  string `json:"access_token"`  // Short-lived token for API requests (15 min)
	RefreshToken string `json:"refresh_token"` // Long-lived token to refresh access (24h or 30d)
	User         User   `json:"user"`
}

// ErrorResponse is the standard error response format.
type ErrorResponse struct {
	Message string `json:"message"`
}
