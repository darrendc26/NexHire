package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"nexhire/backend/models"
)

// ContextKey is the context key type for auth context values
type ContextKey string

const (
	// UserContextKey is used to store/retrieve user claims in http.Request Context
	UserContextKey ContextKey = "user_claims"
)

// Handler handles HTTP requests for authentication
type Handler struct {
	service *Service
	cfg     *Config
}

// NewHandler creates a new Auth Handler instance
func NewHandler(service *Service, cfg *Config) *Handler {
	return &Handler{
		service: service,
		cfg:     cfg,
	}
}

// RegisterRoutes registers auth endpoints on the provided ServeMux
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/auth/google/login", h.HandleGoogleLogin)
	mux.HandleFunc("GET /api/auth/google/callback", h.HandleGoogleCallback)
	mux.HandleFunc("POST /api/auth/google/verify", h.HandleGoogleVerifyToken)
	mux.HandleFunc("POST /api/auth/logout", h.HandleLogout)
	mux.HandleFunc("GET /api/auth/me", h.HandleMe)
}

// HandleGoogleLogin initiates the Google OAuth redirect flow
func (h *Handler) HandleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	if h.cfg.GoogleClientID == "" || strings.Contains(h.cfg.GoogleClientID, "your-google-client-id") {
		http.Error(w, "Google Client ID is not configured. Please set GOOGLE_CLIENT_ID in backend/.env with your OAuth Client ID from Google Cloud Console.", http.StatusBadRequest)
		return
	}

	state, err := generateRandomState()
	if err != nil {
		http.Error(w, "Failed to generate security state", http.StatusInternalServerError)
		return
	}

	// Set state cookie for CSRF verification
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Expires:  time.Now().Add(10 * time.Minute),
		HttpOnly: true,
		Secure:   h.cfg.SecureCookie,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})

	authURL := h.service.GetGoogleAuthURL(state)
	fmt.Printf("🔑 Initiating Google OAuth Login:\n   Redirect URI sent to Google: %s\n", h.cfg.GoogleRedirectURL)
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// HandleGoogleCallback handles the OAuth2 redirect callback from Google
func (h *Handler) HandleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	// Verify state parameter to prevent CSRF
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil || stateCookie.Value == "" {
		http.Error(w, "State cookie missing or invalid", http.StatusBadRequest)
		return
	}

	// Clear state cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		HttpOnly: true,
	})

	queryState := r.URL.Query().Get("state")
	if queryState != stateCookie.Value {
		http.Error(w, "State verification failed (CSRF mismatch)", http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Authorization code missing", http.StatusBadRequest)
		return
	}

	// Exchange code for Google User Profile
	profile, err := h.service.ExchangeCodeAndGetProfile(r.Context(), code)
	if err != nil {
		http.Error(w, "Failed to fetch user profile from Google: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get or Create user in system
	user, err := h.service.GetOrCreateUser(r.Context(), profile)
	if err != nil {
		http.Error(w, "Failed to process user account", http.StatusInternalServerError)
		return
	}

	// Generate JWT session token
	tokenString, expiration, err := h.service.GenerateJWT(user)
	if err != nil {
		http.Error(w, "Failed to issue session token", http.StatusInternalServerError)
		return
	}

	// Set auth token HTTP-Only cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    tokenString,
		Expires:  expiration,
		HttpOnly: true,
		Secure:   h.cfg.SecureCookie,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})

	// Redirect back to frontend
	redirectURL := h.cfg.FrontendURL + "/dashboard"
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

// HandleGoogleVerifyToken handles Google ID Token verification from SPA / One-Tap frontend SDKs
func (h *Handler) HandleGoogleVerifyToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.GoogleVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	if req.IDToken == "" {
		http.Error(w, "id_token is required", http.StatusBadRequest)
		return
	}

	profile, err := h.service.VerifyGoogleIDToken(r.Context(), req.IDToken)
	if err != nil {
		http.Error(w, "Invalid Google ID token: "+err.Error(), http.StatusUnauthorized)
		return
	}

	user, err := h.service.GetOrCreateUser(r.Context(), profile)
	if err != nil {
		http.Error(w, "Failed to process user account", http.StatusInternalServerError)
		return
	}

	tokenString, expiration, err := h.service.GenerateJWT(user)
	if err != nil {
		http.Error(w, "Failed to issue session token", http.StatusInternalServerError)
		return
	}

	// Set auth token HTTP-Only cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    tokenString,
		Expires:  expiration,
		HttpOnly: true,
		Secure:   h.cfg.SecureCookie,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.AuthResponse{
		Token:     tokenString,
		User:      user,
		ExpiresIn: int64(time.Until(expiration).Seconds()),
	})
}

// HandleLogout clears user session cookies
func (h *Handler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cfg.SecureCookie,
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Logged out successfully",
	})
}

// HandleMe returns details of the currently authenticated user
func (h *Handler) HandleMe(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(UserContextKey).(*JWTClaims)
	if !ok || claims == nil {
		// Try parsing token directly from cookie or Authorization header if middleware wasn't run
		tokenStr := extractTokenFromRequest(r)
		if tokenStr == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		parsedClaims, err := h.service.ValidateJWT(tokenStr)
		if err != nil {
			http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
			return
		}
		claims = parsedClaims
	}

	user, ok := h.service.GetUserByID(claims.UserID)
	if !ok {
		// Return claims data if user struct not found in transient memory
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(models.User{
			ID:      claims.UserID,
			Email:   claims.Email,
			Name:    claims.Name,
			Picture: claims.Picture,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func generateRandomState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func extractTokenFromRequest(r *http.Request) string {
	// Check cookie
	if cookie, err := r.Cookie("auth_token"); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	// Check Authorization header
	authHeader := r.Header.Get("Authorization")
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		return authHeader[7:]
	}
	return ""
}
