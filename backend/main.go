package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"google.golang.org/genai"

	"nexhire/backend/ai"
	"nexhire/backend/auth"
	"nexhire/backend/candidate"
	"nexhire/backend/interview"
	"nexhire/backend/middleware"
	"nexhire/backend/speech"
	"nexhire/backend/utils"
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

func extractTokenFromGin(c *gin.Context) string {
	if cookie, err := c.Request.Cookie("auth_token"); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	authHeader := c.GetHeader("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	return ""
}

func initDatabase(dbURL string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping postgres database: %w", err)
	}

	// Try reading init.sql from relative paths
	initPaths := []string{"init.sql", "./init.sql", "../init.sql"}
	var initSQL []byte
	for _, p := range initPaths {
		if content, err := os.ReadFile(p); err == nil {
			initSQL = content
			break
		}
	}

	if len(initSQL) > 0 {
		if _, err := db.Exec(string(initSQL)); err != nil {
			log.Printf("Warning: schema init.sql execution error: %v", err)
		} else {
			log.Println("Database schema initialized from init.sql")
		}
	}

	return db, nil
}

func main() {
	cfg := auth.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgrespassword@localhost:5433/nexhire?sslmode=disable"
	}

	db, err := initDatabase(dbURL)
	if err != nil {
		log.Fatalf("Fatal: PostgreSQL connection failed: %v", err)
	}

	log.Println("Connected to PostgreSQL successfully!")
	interviewRepo := interview.NewPostgresRepository(db)
	candidateRepo := candidate.NewPostgresRepository(db)

	var aiService *ai.Service
	geminiAPIKey := os.Getenv("GEMINI_API_KEY")
	if geminiAPIKey != "" {
		ctx := context.Background()
		client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: geminiAPIKey})
		if err != nil {
			log.Printf("Warning: Failed to initialize Gemini API client: %v", err)
		} else {
			geminiModel := os.Getenv("GEMINI_MODEL")
			if geminiModel == "" {
				geminiModel = "gemini-3.5-flash-lite"
			}
			provider := ai.NewGeminiProvider(client, geminiModel)
			aiService = ai.NewService(provider)
			log.Printf("Gemini AI provider initialized with model %s!", geminiModel)
		}
	} else {
		log.Println("Notice: GEMINI_API_KEY not set in environment.")
	}

	redisClient := utils.NewRedisClient()
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Redis connection failed: %v", err)
	}

	log.Println("Redis connected")

	authService := auth.NewServiceWithDB(cfg, db)
	authHandler := auth.NewHandler(authService, cfg)

	interviewService := interview.NewService(interviewRepo)
	interviewHandler := interview.NewHandler(interviewService)

	candidateService := candidate.NewService(candidateRepo, aiService, redisClient)
	candidateHandler := candidate.NewHandler(candidateService)

	speechService := speech.NewService()
	speechHandler := speech.NewHandler(speechService, candidateService)

	ginEngine := gin.New()
	ginEngine.Use(gin.Recovery())

	api := ginEngine.Group("/api")

	ginAuthMiddleware := func(c *gin.Context) {
		tokenString := extractTokenFromGin(c)
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: missing token"})
			c.Abort()
			return
		}

		claims, err := authService.ValidateJWT(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: " + err.Error()})
			c.Abort()
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("claims", claims)
		c.Next()
	}

	interviewHandler.RegisterRoutes(api, ginAuthMiddleware)
	candidateHandler.RegisterRoutes(api)
	speechHandler.RegisterRoutes(api)

	mux := http.NewServeMux()

	// Register Gin routers on mux
	mux.Handle("/api/interviews", ginEngine)
	mux.Handle("/api/interviews/", ginEngine)
	mux.Handle("/api/candidates", ginEngine)
	mux.Handle("/api/candidates/", ginEngine)
	mux.Handle("/api/speech", ginEngine)
	mux.Handle("/api/speech/", ginEngine)

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

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	handler := corsMiddleware(mux)

	fmt.Printf("NexHire Backend Auth Server running on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}
