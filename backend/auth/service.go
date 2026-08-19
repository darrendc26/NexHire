package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"nexhire/backend/models"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// JWTClaims represents custom JWT token payload claims
type JWTClaims struct {
	UserID  string `json:"user_id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
	jwt.RegisteredClaims
}

// Service provides methods for Google authentication and session token management
type Service struct {
	cfg          *Config
	oauthConfig  *oauth2.Config
	db           *sql.DB
	usersMutex   sync.RWMutex
	usersByID    map[string]*models.User
	usersByEmail map[string]*models.User
}

// NewService initializes a new Auth Service with config
func NewService(cfg *Config) *Service {
	return NewServiceWithDB(cfg, nil)
}

// NewServiceWithDB initializes a new Auth Service with optional database connection
func NewServiceWithDB(cfg *Config, db *sql.DB) *Service {
	oauthCfg := &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.GoogleRedirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.profile",
			"https://www.googleapis.com/auth/userinfo.email",
			"openid",
		},
		Endpoint: google.Endpoint,
	}

	return &Service{
		cfg:          cfg,
		oauthConfig:  oauthCfg,
		db:           db,
		usersByID:    make(map[string]*models.User),
		usersByEmail: make(map[string]*models.User),
	}
}

// SetDB attaches a database connection to the service
func (s *Service) SetDB(db *sql.DB) {
	s.db = db
}

// GetGoogleAuthURL generates the Google OAuth consent URL with a secure state string
func (s *Service) GetGoogleAuthURL(state string) string {
	return s.oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// ExchangeCodeAndGetProfile exchanges an authorization code for tokens and retrieves the Google user profile
func (s *Service) ExchangeCodeAndGetProfile(ctx context.Context, code string) (*models.GoogleUserInfo, error) {
	token, err := s.oauthConfig.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange auth code: %w", err)
	}

	client := s.oauthConfig.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("google userinfo returned status %d: %s", resp.StatusCode, string(body))
	}

	var profile models.GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, fmt.Errorf("failed to decode user profile: %w", err)
	}

	return &profile, nil
}

// VerifyGoogleIDToken verifies a Google ID Token (e.g. from Google One-Tap or Google Sign-In SDK)
func (s *Service) VerifyGoogleIDToken(ctx context.Context, idToken string) (*models.GoogleUserInfo, error) {
	if idToken == "" {
		return nil, errors.New("id token cannot be empty")
	}

	url := fmt.Sprintf("https://oauth2.googleapis.com/tokeninfo?id_token=%s", idToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to verify id token with google: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("invalid google id token (status %d): %s", resp.StatusCode, string(body))
	}

	var profile models.GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, fmt.Errorf("failed to parse google id token payload: %w", err)
	}

	if profile.Email == "" {
		return nil, errors.New("id token does not contain a valid email")
	}

	return &profile, nil
}

// GetOrCreateUser retrieves an existing user by email or creates a new one
func (s *Service) GetOrCreateUser(ctx context.Context, profile *models.GoogleUserInfo) (*models.User, error) {
	if s.db != nil {
		return s.getOrCreateUserDB(ctx, profile)
	}

	s.usersMutex.Lock()
	defer s.usersMutex.Unlock()

	// Check if user already exists
	if existing, ok := s.usersByEmail[profile.Email]; ok {
		existing.Name = profile.Name
		existing.Picture = profile.Picture
		existing.GoogleID = profile.Sub
		return existing, nil
	}

	// Create new user
	now := time.Now()
	userID := fmt.Sprintf("usr_%d", now.UnixNano())
	newUser := &models.User{
		ID:        userID,
		GoogleID:  profile.Sub,
		Email:     profile.Email,
		Name:      profile.Name,
		Picture:   profile.Picture,
		CreatedAt: now,
	}

	s.usersByID[userID] = newUser
	s.usersByEmail[profile.Email] = newUser

	return newUser, nil
}

func (s *Service) getOrCreateUserDB(ctx context.Context, profile *models.GoogleUserInfo) (*models.User, error) {
	querySelect := `
		SELECT id, google_id, email, name, picture, created_at
		FROM users
		WHERE email = $1
	`

	var user models.User
	err := s.db.QueryRowContext(ctx, querySelect, profile.Email).Scan(
		&user.ID,
		&user.GoogleID,
		&user.Email,
		&user.Name,
		&user.Picture,
		&user.CreatedAt,
	)

	if err == nil {
		// Update user profile info
		queryUpdate := `
			UPDATE users
			SET google_id = $1, name = $2, picture = $3, updated_at = NOW()
			WHERE id = $4
		`
		_, _ = s.db.ExecContext(ctx, queryUpdate, profile.Sub, profile.Name, profile.Picture, user.ID)
		user.GoogleID = profile.Sub
		user.Name = profile.Name
		user.Picture = profile.Picture
		return &user, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to query user from database: %w", err)
	}

	// User not found, create new user
	now := time.Now()
	userID := fmt.Sprintf("usr_%d", now.UnixNano())
	newUser := &models.User{
		ID:        userID,
		GoogleID:  profile.Sub,
		Email:     profile.Email,
		Name:      profile.Name,
		Picture:   profile.Picture,
		CreatedAt: now,
	}

	queryInsert := `
		INSERT INTO users (id, google_id, email, name, picture, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $6)
	`
	_, err = s.db.ExecContext(ctx, queryInsert, newUser.ID, newUser.GoogleID, newUser.Email, newUser.Name, newUser.Picture, newUser.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert new user into database: %w", err)
	}

	return newUser, nil
}

// GenerateJWT creates a signed application JWT token for an authenticated user
func (s *Service) GenerateJWT(user *models.User) (string, time.Time, error) {
	expirationTime := time.Now().Add(time.Duration(s.cfg.JWTExpirationHours) * time.Hour)

	claims := &JWTClaims{
		UserID:  user.ID,
		Email:   user.Email,
		Name:    user.Name,
		Picture: user.Picture,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   user.ID,
			Issuer:    "nexhire-auth",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, expirationTime, nil
}

// ValidateJWT verifies and parses an application JWT token string
func (s *Service) ValidateJWT(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.cfg.JWTSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}

// GetUserByID fetches a user from storage by ID
func (s *Service) GetUserByID(id string) (*models.User, bool) {
	if s.db != nil {
		query := `SELECT id, google_id, email, name, picture, created_at FROM users WHERE id = $1`
		var user models.User
		err := s.db.QueryRow(query, id).Scan(
			&user.ID,
			&user.GoogleID,
			&user.Email,
			&user.Name,
			&user.Picture,
			&user.CreatedAt,
		)
		if err == nil {
			return &user, true
		}
		return nil, false
	}

	s.usersMutex.RLock()
	defer s.usersMutex.RUnlock()

	user, ok := s.usersByID[id]
	return user, ok
}
