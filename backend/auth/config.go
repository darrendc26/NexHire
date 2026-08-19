package auth

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds Google OAuth and JWT authentication configuration settings
type Config struct {
	Port               string
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	JWTSecret          string
	JWTExpirationHours int
	FrontendURL        string
	SecureCookie       bool
}

func Load() *Config {
	_ = godotenv.Load(".env", "backend/.env", "../.env")

	cfg := &Config{
		Port:               getEnv("PORT", "8080"),
		FrontendURL:        getEnv("FRONTEND_URL", "http://localhost:3000"),
		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:  getEnv("GOOGLE_REDIRECT_URL", ""),
		JWTSecret:          getEnv("JWT_SECRET", ""),
		JWTExpirationHours: getEnvInt("JWT_EXPIRATION_HOURS", 24),
		SecureCookie:       getEnv("ENV", "development") == "production",
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)

	if value == "" {
		return fallback
	}

	result, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return result
}

func validate(cfg *Config) {
	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}

	if cfg.GoogleClientID == "" {
		log.Fatal("GOOGLE_CLIENT_ID is required")
	}

	if cfg.GoogleClientSecret == "" {
		log.Fatal("GOOGLE_CLIENT_SECRET is required")
	}
}
