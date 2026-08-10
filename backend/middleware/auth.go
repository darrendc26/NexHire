package middleware

import "context"
import "net/http"
import "strings"

import "nexhire/backend/auth"

// RequireAuth creates a HTTP middleware that validates JWT tokens and injects user claims into request context
func RequireAuth(authService *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString := extractToken(r)
			if tokenString == "" {
				http.Error(w, `{"error": "Unauthorized: missing token"}`, http.StatusUnauthorized)
				return
			}

			claims, err := authService.ValidateJWT(tokenString)
			if err != nil {
				http.Error(w, `{"error": "Unauthorized: `+err.Error()+`"}`, http.StatusUnauthorized)
				return
			}

			// Add claims to context
			ctx := context.WithValue(r.Context(), auth.UserContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserClaims retrieves authenticated user JWT claims from the request context
func GetUserClaims(r *http.Request) (*auth.JWTClaims, bool) {
	claims, ok := r.Context().Value(auth.UserContextKey).(*auth.JWTClaims)
	return claims, ok && claims != nil
}

func extractToken(r *http.Request) string {
	// 1. Check HTTP-only cookie
	if cookie, err := r.Cookie("auth_token"); err == nil && cookie.Value != "" {
		return cookie.Value
	}

	// 2. Check Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return parts[1]
		}
	}

	return ""
}
