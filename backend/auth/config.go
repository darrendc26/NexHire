package auth

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config holds Google OAuth and JWT authentication configuration settings
type Config struct {
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	JWTSecret          string
	JWTExpirationHours int
	FrontendURL        string
	SecureCookie       bool
}

// LoadConfigFromEnv loads configuration from environment variables with sensible defaults
func LoadConfigFromEnv() *Config {
	// Auto-load .env file if present
	loadEnvFiles()

	jwtExpHours := 24
	secureCookie := os.Getenv("ENV") == "production"

	clientID := getEnvOrDefault("GOOGLE_CLIENT_ID", "")
	redirectURL := getEnvOrDefault("GOOGLE_REDIRECT_URL", "http://localhost:8080/api/auth/google/callback")

	if clientID == "" || strings.Contains(clientID, "your-google-client-id") {
		fmt.Println("⚠️  WARNING: GOOGLE_CLIENT_ID is not configured in backend/.env!")
		fmt.Println("👉 Update backend/.env with your Google OAuth Client ID from https://console.cloud.google.com/")
	} else {
		fmt.Printf("🔑 Configured Client ID: %s...\n", clientID[:min(15, len(clientID))])
		fmt.Printf("🔗 Configured Redirect URL: %s\n", redirectURL)
	}

	return &Config{
		GoogleClientID:     clientID,
		GoogleClientSecret: getEnvOrDefault("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:  redirectURL,
		JWTSecret:          getEnvOrDefault("JWT_SECRET", "nexhire-dev-secret-key-change-in-prod"),
		JWTExpirationHours: jwtExpHours,
		FrontendURL:        getEnvOrDefault("FRONTEND_URL", "http://localhost:3000"),
		SecureCookie:       secureCookie,
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func loadEnvFiles() {
	candidates := []string{".env", "./.env", "../.env", "backend/.env"}
	for _, path := range candidates {
		if absPath, err := filepath.Abs(path); err == nil {
			if parseEnvFile(absPath) {
				fmt.Printf("📄 Loaded environment variables from %s\n", absPath)
				break
			}
		}
	}
}

func parseEnvFile(filename string) bool {
	file, err := os.Open(filename)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			val = strings.Trim(val, `"'`)
			os.Setenv(key, val)
		}
	}
	return true
}

func getEnvOrDefault(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

