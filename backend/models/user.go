package models

import "time"

// User represents an authenticated application user
type User struct {
	ID        string    `json:"id"`
	GoogleID  string    `json:"google_id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Picture   string    `json:"picture"`
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt time.Time `json:"updated_at"`
}

// GoogleUserInfo represents the profile details returned from Google APIs / ID Token
type GoogleUserInfo struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
	// EmailVerified bool   `json:"email_verified"`
	Name string `json:"name"`
	// GivenName     string `json:"given_name"`
	// FamilyName    string `json:"family_name"`
	Picture string `json:"picture"`
	// Locale        string `json:"locale"`
}

// AuthResponse represents the JSON payload returned on successful authentication
type AuthResponse struct {
	Token     string `json:"token"`
	User      *User  `json:"user"`
	ExpiresIn int64  `json:"expires_in"` // expiration in seconds
}

// GoogleVerifyRequest represents the request body for ID Token verification (frontend One-Tap / GSI)
type GoogleVerifyRequest struct {
	IDToken string `json:"id_token"`
}
