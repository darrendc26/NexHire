package auth

import (
	"nexhire/backend/models"
	"testing"
	"time"
)

func TestJWTGenerationAndValidation(t *testing.T) {
	cfg := &Config{
		JWTSecret:          "test-secret-key-12345",
		JWTExpirationHours: 1,
	}

	service := NewService(cfg)

	user := &models.User{
		ID:        "usr_123",
		GoogleID:  "google_999",
		Email:     "test@example.com",
		Name:      "Test User",
		Picture:   "https://example.com/avatar.png",
		CreatedAt: time.Now(),
		// UpdatedAt: time.Now(),
	}

	tokenStr, exp, err := service.GenerateJWT(user)
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}

	if tokenStr == "" {
		t.Fatal("Expected non-empty token string")
	}

	if exp.Before(time.Now()) {
		t.Fatal("Expected expiration to be in the future")
	}

	// Validate valid token
	claims, err := service.ValidateJWT(tokenStr)
	if err != nil {
		t.Fatalf("Failed to validate JWT token: %v", err)
	}

	if claims.UserID != user.ID {
		t.Errorf("Expected UserID %s, got %s", user.ID, claims.UserID)
	}
	if claims.Email != user.Email {
		t.Errorf("Expected Email %s, got %s", user.Email, claims.Email)
	}
}

func TestGoogleAuthURL(t *testing.T) {
	cfg := &Config{
		GoogleClientID:    "client-id-xyz",
		GoogleRedirectURL: "http://localhost:8080/api/auth/google/callback",
	}

	service := NewService(cfg)
	url := service.GetGoogleAuthURL("random-state-123")

	if url == "" {
		t.Fatal("Expected non-empty Auth URL")
	}
}
