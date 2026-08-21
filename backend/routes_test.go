package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"google.golang.org/genai"

	"nexhire/backend/ai"
	"nexhire/backend/auth"
	"nexhire/backend/candidate"
	"nexhire/backend/interview"
	"nexhire/backend/middleware"
	"nexhire/backend/models"
)

type mockAIProviderForRoutes struct{}

func (m *mockAIProviderForRoutes) GenerateInitialQuestion(ctx context.Context, input ai.InterviewContext) (*ai.InterviewTurn, error) {
	return &ai.InterviewTurn{
		NextQuestion:   "Tell me about your experience with Go and backend systems.",
		QuestionType:   "technical",
		ShouldContinue: true,
	}, nil
}

func (m *mockAIProviderForRoutes) ProcessAnswer(ctx context.Context, input ai.InterviewContext) (*ai.InterviewTurn, error) {
	return &ai.InterviewTurn{
		Score:          8.5,
		Feedback:       "Great explanation of Go concepts.",
		Strengths:      []string{"Go language", "Backend architecture"},
		NextQuestion:   "How do you approach error handling and concurrency in Go?",
		QuestionType:   "technical",
		ShouldContinue: true,
	}, nil
}

func (m *mockAIProviderForRoutes) GenerateReport(ctx context.Context, input ai.ReportContext) (*ai.InterviewReport, error) {
	return &ai.InterviewReport{
		OverallScore:   85.0,
		Recommendation: ai.Hire,
		Summary:        "Candidate demonstrated strong software engineering principles.",
		Strengths:      []string{"Backend Development", "Go"},
		Skills: []ai.SkillScore{
			{Skill: "Go", Score: 85.0, Feedback: "Good performance"},
		},
	}, nil
}

func setupTestServer(t *testing.T) (*httptest.Server, *sql.DB) {
	cfg := auth.Load()
	if cfg.JWTSecret == "" {
		cfg.JWTSecret = "super-secret-jwt-key-change-in-production"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgrespassword@localhost:5433/nexhire?sslmode=disable"
	}
	db, err := initDatabase(dbURL)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	interviewRepo := interview.NewPostgresRepository(db)
	candidateRepo := candidate.NewPostgresRepository(db)

	var aiService *ai.Service
	geminiAPIKey := os.Getenv("GEMINI_API_KEY")
	if geminiAPIKey != "" {
		ctx := context.Background()
		client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: geminiAPIKey})
		if err == nil {
			provider := ai.NewGeminiProvider(client, "gemini-3.6-flash")
			aiService = ai.NewService(provider)
		}
	}
	if aiService == nil {
		mockProv := &mockAIProviderForRoutes{}
		aiService = ai.NewService(mockProv)
	}

	authService := auth.NewServiceWithDB(cfg, db)
	authHandler := auth.NewHandler(authService, cfg)

	interviewService := interview.NewService(interviewRepo)
	interviewHandler := interview.NewHandler(interviewService)

	candidateService := candidate.NewService(candidateRepo, aiService)
	candidateHandler := candidate.NewHandler(candidateService)

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

	mux := http.NewServeMux()

	mux.Handle("/api/interviews", ginEngine)
	mux.Handle("/api/interviews/", ginEngine)
	mux.Handle("/api/candidates", ginEngine)
	mux.Handle("/api/candidates/", ginEngine)

	authHandler.RegisterRoutes(mux)

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

	frontendDir := getFrontendDir()
	fs := http.FileServer(http.Dir(frontendDir))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		requestedPath := filepath.Join(frontendDir, r.URL.Path)
		info, err := os.Stat(requestedPath)
		if err == nil && !info.IsDir() {
			fs.ServeHTTP(w, r)
			return
		}

		http.ServeFile(w, r, filepath.Join(frontendDir, "index.html"))
	})

	handler := corsMiddleware(mux)
	ts := httptest.NewServer(handler)
	return ts, db
}

func getTestJWTWithDB(t *testing.T, db *sql.DB) (string, *models.User) {
	cfg := auth.Load()
	if cfg.JWTSecret == "" {
		cfg.JWTSecret = "super-secret-jwt-key-change-in-production"
	}

	authService := auth.NewServiceWithDB(cfg, db)

	profile := &models.GoogleUserInfo{
		Sub:     "google_test_123",
		Email:   "testuser@example.com",
		Name:    "Test Recruiter",
		Picture: "https://example.com/photo.jpg",
	}

	user, err := authService.GetOrCreateUser(context.Background(), profile)
	if err != nil {
		t.Fatalf("Failed to get or create test user in DB: %v", err)
	}

	token, _, err := authService.GenerateJWT(user)
	if err != nil {
		t.Fatalf("Failed to generate test JWT: %v", err)
	}
	return token, user
}

func TestAllRoutes(t *testing.T) {
	ts, db := setupTestServer(t)
	defer ts.Close()

	testServerURL := ts.URL

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	testJWT, testUser := getTestJWTWithDB(t, db)

	// 1. GET / (Frontend static file)
	t.Run("GET / (Frontend index.html)", func(t *testing.T) {
		resp, err := client.Get(testServerURL + "/")
		if err != nil {
			t.Fatalf("Failed GET /: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", resp.StatusCode)
		}
	})

	// 2. GET /api/auth/google/login (OAuth redirect)
	t.Run("GET /api/auth/google/login", func(t *testing.T) {
		resp, err := client.Get(testServerURL + "/api/auth/google/login")
		if err != nil {
			t.Fatalf("Failed GET /api/auth/google/login: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusTemporaryRedirect {
			t.Errorf("Expected 307 Temporary Redirect, got %d", resp.StatusCode)
		}

		cookies := resp.Cookies()
		hasStateCookie := false
		for _, c := range cookies {
			if c.Name == "oauth_state" {
				hasStateCookie = true
				break
			}
		}
		if !hasStateCookie {
			t.Errorf("Expected oauth_state cookie in redirect response")
		}
	})

	// 3. GET /api/auth/google/callback without state
	t.Run("GET /api/auth/google/callback (missing state)", func(t *testing.T) {
		resp, err := client.Get(testServerURL + "/api/auth/google/callback")
		if err != nil {
			t.Fatalf("Failed GET callback: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected 400 Bad Request, got %d", resp.StatusCode)
		}
	})

	// 4. POST /api/auth/google/verify (invalid token)
	t.Run("POST /api/auth/google/verify (invalid token)", func(t *testing.T) {
		body := map[string]string{"id_token": "invalid-token-sample"}
		jsonBody, _ := json.Marshal(body)
		resp, err := client.Post(testServerURL+"/api/auth/google/verify", "application/json", bytes.NewBuffer(jsonBody))
		if err != nil {
			t.Fatalf("Failed POST verify: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected 401 or 400, got %d", resp.StatusCode)
		}
	})

	// 5. GET /api/auth/me (unauthenticated)
	t.Run("GET /api/auth/me (unauthenticated)", func(t *testing.T) {
		resp, err := client.Get(testServerURL + "/api/auth/me")
		if err != nil {
			t.Fatalf("Failed GET /api/auth/me: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("Expected 401 Unauthorized, got %d", resp.StatusCode)
		}
	})

	// 6. GET /api/auth/me (authenticated)
	t.Run("GET /api/auth/me (authenticated)", func(t *testing.T) {
		req, _ := http.NewRequest("GET", testServerURL+"/api/auth/me", nil)
		req.Header.Set("Authorization", "Bearer "+testJWT)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed GET /api/auth/me: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", resp.StatusCode)
		}

		var user models.User
		if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
			t.Fatalf("Failed to decode user response: %v", err)
		}
		if user.ID != testUser.ID {
			t.Errorf("Expected UserID %s, got %s", testUser.ID, user.ID)
		}
	})

	// 7. GET /api/protected/profile (unauthenticated)
	t.Run("GET /api/protected/profile (unauthenticated)", func(t *testing.T) {
		resp, err := client.Get(testServerURL + "/api/protected/profile")
		if err != nil {
			t.Fatalf("Failed GET protected profile: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("Expected 401 Unauthorized, got %d", resp.StatusCode)
		}
	})

	// 8. GET /api/protected/profile (authenticated)
	t.Run("GET /api/protected/profile (authenticated)", func(t *testing.T) {
		req, _ := http.NewRequest("GET", testServerURL+"/api/protected/profile", nil)
		req.Header.Set("Authorization", "Bearer "+testJWT)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed GET protected profile: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", resp.StatusCode)
		}

		var res map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&res)
		if res["user_id"] != testUser.ID {
			t.Errorf("Expected user_id %s, got %v", testUser.ID, res["user_id"])
		}
	})

	// 9. POST /api/auth/logout
	t.Run("POST /api/auth/logout", func(t *testing.T) {
		resp, err := client.Post(testServerURL+"/api/auth/logout", "application/json", nil)
		if err != nil {
			t.Fatalf("Failed POST logout: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", resp.StatusCode)
		}
	})

	// 10. POST /api/interviews (unauthenticated)
	t.Run("POST /api/interviews (unauthenticated)", func(t *testing.T) {
		payload := map[string]interface{}{
			"title":       "Senior Go Developer",
			"role":        "Backend Engineer",
			"description": "Golang backend role",
			"difficulty":  "hard",
			"duration":    5,
		}
		body, _ := json.Marshal(payload)
		resp, err := client.Post(testServerURL+"/api/interviews", "application/json", bytes.NewBuffer(body))
		if err != nil {
			t.Fatalf("Failed POST /api/interviews: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("Expected 401 Unauthorized, got %d", resp.StatusCode)
		}
	})

	var createdInterviewID string
	// 11. POST /api/interviews (authenticated - 5 min duration for testing)
	t.Run("POST /api/interviews (authenticated - 5 min duration)", func(t *testing.T) {
		payload := map[string]interface{}{
			"title":         "Fullstack Engineer Interview",
			"role":          "Senior Developer",
			"description":   "React + Go Stack",
			"difficulty":    "medium",
			"duration":      5,
			"voice_enabled": true,
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", testServerURL+"/api/interviews", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+testJWT)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed POST /api/interviews: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			respBody, _ := io.ReadAll(resp.Body)
			t.Fatalf("Expected 201 Created, got %d: %s", resp.StatusCode, string(respBody))
		}

		var res struct {
			Interview models.Interview `json:"interview"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if res.Interview.ID == "" {
			t.Errorf("Expected interview ID to be generated")
		}
		if res.Interview.Duration != 5 {
			t.Errorf("Expected duration = 5, got %d", res.Interview.Duration)
		}

		createdInterviewID = res.Interview.ID
		t.Logf("Created 5-Minute Interview ID: %s", createdInterviewID)
	})

	// 12. GET /api/interviews (authenticated)
	t.Run("GET /api/interviews (authenticated)", func(t *testing.T) {
		req, _ := http.NewRequest("GET", testServerURL+"/api/interviews", nil)
		req.Header.Set("Authorization", "Bearer "+testJWT)

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed GET /api/interviews: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", resp.StatusCode)
		}

		var res struct {
			Interviews []models.Interview `json:"interviews"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			t.Fatalf("Failed to decode interviews list: %v", err)
		}

		if len(res.Interviews) == 0 {
			t.Errorf("Expected at least 1 interview")
		}
	})

	// 13. GET /api/interviews/:id (authenticated)
	t.Run("GET /api/interviews/:id", func(t *testing.T) {
		if createdInterviewID == "" {
			t.Skip("No created interview ID")
		}

		req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/interviews/%s", testServerURL, createdInterviewID), nil)
		req.Header.Set("Authorization", "Bearer "+testJWT)

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed GET /api/interviews/:id: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", resp.StatusCode)
		}
	})

	var createdSessionID string
	var rawSessionToken string
	// 14. POST /api/candidates (Start session)
	t.Run("POST /api/candidates", func(t *testing.T) {
		if createdInterviewID == "" {
			t.Skip("No created interview ID to start candidate session")
		}

		payload := map[string]string{
			"interview_id": createdInterviewID,
			"name":         "John Candidate",
			"email":        "john.candidate@example.com",
		}
		body, _ := json.Marshal(payload)
		resp, err := client.Post(testServerURL+"/api/candidates", "application/json", bytes.NewBuffer(body))
		if err != nil {
			t.Fatalf("Failed POST /api/candidates: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			t.Fatalf("Expected 200 OK, got %d: %s", resp.StatusCode, string(respBody))
		}

		var res struct {
			Session candidate.StartSessionResponse `json:"session"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			t.Fatalf("Failed to decode session: %v", err)
		}

		if res.Session.SessionID == "" {
			t.Errorf("Expected candidate session ID")
		}

		createdSessionID = res.Session.SessionID
		rawSessionToken = res.Session.SessionToken
		t.Logf("Created Candidate Session ID: %s, Token: %s", createdSessionID, rawSessionToken)
	})

	// 15. GET /api/candidates/:id
	t.Run("GET /api/candidates/:id", func(t *testing.T) {
		if createdSessionID == "" {
			t.Skip("No candidate session ID")
		}

		resp, err := client.Get(fmt.Sprintf("%s/api/candidates/%s", testServerURL, createdSessionID))
		if err != nil {
			t.Fatalf("Failed GET /api/candidates/:id: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", resp.StatusCode)
		}
	})

	// 16. GET /api/candidates/interview/:interviewID
	t.Run("GET /api/candidates/interview/:interviewID", func(t *testing.T) {
		if createdInterviewID == "" {
			t.Skip("No created interview ID")
		}

		resp, err := client.Get(fmt.Sprintf("%s/api/candidates/interview/%s", testServerURL, createdInterviewID))
		if err != nil {
			t.Fatalf("Failed GET /api/candidates/interview/:interviewID: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", resp.StatusCode)
		}
	})

	// 17. POST /api/candidates/sessions/:token/start (Start AI Interview Turn)
	t.Run("POST /api/candidates/sessions/:token/start", func(t *testing.T) {
		if rawSessionToken == "" {
			t.Skip("No candidate session token")
		}

		resp, err := client.Post(fmt.Sprintf("%s/api/candidates/sessions/%s/start", testServerURL, rawSessionToken), "application/json", nil)
		if err != nil {
			t.Fatalf("Failed POST /api/candidates/sessions/:token/start: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			t.Fatalf("Expected 200 OK, got %d: %s", resp.StatusCode, string(respBody))
		}

		var res candidate.StartQuestionResponse
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			t.Fatalf("Failed to decode start response: %v", err)
		}

		if res.Question.Text == "" {
			t.Errorf("Expected initial question text")
		}
		if res.Progress.TimeRemaining <= 0 {
			t.Errorf("Expected positive time remaining for 5-minute interview, got %d", res.Progress.TimeRemaining)
		}
		t.Logf("Initial Q: %s, Time Remaining: %ds", res.Question.Text, res.Progress.TimeRemaining)
	})

	// 18. POST /api/candidates/sessions/:token/answer (Submit AI Answer Turn)
	t.Run("POST /api/candidates/sessions/:token/answer", func(t *testing.T) {
		if rawSessionToken == "" {
			t.Skip("No candidate session token")
		}

		payload := map[string]string{
			"answer": "I have 4 years of experience building microservices with Go and Gin framework.",
		}
		body, _ := json.Marshal(payload)
		resp, err := client.Post(fmt.Sprintf("%s/api/candidates/sessions/%s/answer", testServerURL, rawSessionToken), "application/json", bytes.NewBuffer(body))
		if err != nil {
			t.Fatalf("Failed POST /api/candidates/sessions/:token/answer: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			t.Fatalf("Expected 200 OK, got %d: %s", resp.StatusCode, string(respBody))
		}

		var res candidate.SubmitAnswerResponse
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			t.Fatalf("Failed to decode answer response: %v", err)
		}

		if res.Evaluation == nil {
			t.Errorf("Expected evaluation in answer response")
		}
		t.Logf("Answer score: %.1f, Next Q: %+v", res.Evaluation.Score, res.NextQuestion)
	})

	// 19. GET /api/candidates/sessions/:token/report (Get Candidate AI Report)
	t.Run("GET /api/candidates/sessions/:token/report", func(t *testing.T) {
		if rawSessionToken == "" {
			t.Skip("No candidate session token")
		}

		resp, err := client.Get(fmt.Sprintf("%s/api/candidates/sessions/%s/report", testServerURL, rawSessionToken))
		if err != nil {
			t.Fatalf("Failed GET /api/candidates/sessions/:token/report: %v", err)
		}
		defer resp.Body.Close()

		// Report endpoint returns 200 OK if session report exists, or 404 if not yet generated
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected 200 or 404, got %d", resp.StatusCode)
		}
	})
}
