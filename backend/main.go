package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"nexhire/backend/auth"
	"nexhire/backend/middleware"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	cfg := auth.LoadConfigFromEnv()
	authService := auth.NewService(cfg)
	authHandler := auth.NewHandler(authService, cfg)

	mux := http.NewServeMux()

	// Register auth routes
	authHandler.RegisterRoutes(mux)

	// Protected test endpoint
	requireAuth := middleware.RequireAuth(authService)
	mux.Handle("GET /api/protected/profile", requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := middleware.GetUserClaims(r)
		if !ok {
			http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":    "Access granted to protected endpoint!",
			"user_id":    claims.UserID,
			"email":      claims.Email,
			"name":       claims.Name,
			"picture":    claims.Picture,
			"expires_at": claims.ExpiresAt,
		})
	})))

	// Serve static frontend files
	frontendDir := getFrontendDir()
	fs := http.FileServer(http.Dir(frontendDir))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// If requesting API routes, let multiplexer handle standard 404
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		// Handle OAuth callback URL path route if loaded via frontend SPA route like /auth/success
		requestedPath := filepath.Join(frontendDir, r.URL.Path)
		info, err := os.Stat(requestedPath)
		if err == nil && !info.IsDir() {
			fs.ServeHTTP(w, r)
			return
		}

		// Fallback to index.html for SPA routing
		http.ServeFile(w, r, filepath.Join(frontendDir, "index.html"))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	handler := corsMiddleware(mux)

	fmt.Printf("🚀 NexHire Backend Auth Server running on http://localhost:%s\n", port)
	fmt.Printf("📁 Serving frontend from: %s\n", frontendDir)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

func getFrontendDir() string {
	// Check relative to current working directory
	candidates := []string{
		"../frontend",
		"./frontend",
		"frontend",
	}

	for _, c := range candidates {
		if abs, err := filepath.Abs(c); err == nil {
			if info, err := os.Stat(abs); err == nil && info.IsDir() {
				return abs
			}
		}
	}

	// Default fallback to ../frontend
	abs, _ := filepath.Abs("../frontend")
	return abs
}
