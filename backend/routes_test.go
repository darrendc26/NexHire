package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	_ "github.com/lib/pq"

	"nexhire/backend/auth"
	"nexhire/backend/models"
)

const baseURL = "http://localhost:8080"

func getTestJWT(t *testing.T) (string, *models.User) {
	cfg := auth.Load()
	if cfg.JWTSecret == "" {
		cfg.JWTSecret = "super-secret-jwt-key-change-in-production"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgrespassword@localhost:5433/nexhire?sslmode=disable"
	}
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
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
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects automatically
		},
	}

	testJWT, testUser := getTestJWT(t)

	// 1. GET / (Frontend static file)
	t.Run("GET / (Frontend index.html)", func(t *testing.T) {
		resp, err := client.Get(baseURL + "/")
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
		resp, err := client.Get(baseURL + "/api/auth/google/login")
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
		resp, err := client.Get(baseURL + "/api/auth/google/callback")
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
		resp, err := client.Post(baseURL+"/api/auth/google/verify", "application/json", bytes.NewBuffer(jsonBody))
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
		resp, err := client.Get(baseURL + "/api/auth/me")
		if err != nil {
			t.Fatalf("Failed GET /api/auth/me: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("Expected 401 Unauthorized, got %d", resp.StatusCode)
		}
	})

	// 6. GET /api/auth/me (authenticated via Bearer header)
	t.Run("GET /api/auth/me (authenticated)", func(t *testing.T) {
		req, _ := http.NewRequest("GET", baseURL+"/api/auth/me", nil)
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
		resp, err := client.Get(baseURL + "/api/protected/profile")
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
		req, _ := http.NewRequest("GET", baseURL+"/api/protected/profile", nil)
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
		resp, err := client.Post(baseURL+"/api/auth/logout", "application/json", nil)
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
			"duration":    45,
		}
		body, _ := json.Marshal(payload)
		resp, err := client.Post(baseURL+"/api/interviews", "application/json", bytes.NewBuffer(body))
		if err != nil {
			t.Fatalf("Failed POST /api/interviews: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("Expected 401 Unauthorized, got %d", resp.StatusCode)
		}
	})

	var createdInterviewID string
	// 11. POST /api/interviews (authenticated)
	t.Run("POST /api/interviews (authenticated)", func(t *testing.T) {
		payload := map[string]interface{}{
			"title":         "Fullstack Engineer Interview",
			"role":          "Senior Developer",
			"description":   "React + Go Stack",
			"difficulty":    "medium",
			"duration":      30,
			"voice_enabled": true,
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", baseURL+"/api/interviews", bytes.NewBuffer(body))
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
		if res.Interview.RecruiterID != testUser.ID {
			t.Errorf("Expected RecruiterID %s, got %s", testUser.ID, res.Interview.RecruiterID)
		}

		createdInterviewID = res.Interview.ID
		t.Logf("Created Interview ID: %s", createdInterviewID)
	})

	// 12. GET /api/interviews (authenticated)
	t.Run("GET /api/interviews (authenticated)", func(t *testing.T) {
		req, _ := http.NewRequest("GET", baseURL+"/api/interviews", nil)
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

		req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/interviews/%s", baseURL, createdInterviewID), nil)
		req.Header.Set("Authorization", "Bearer "+testJWT)

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed GET /api/interviews/:id: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", resp.StatusCode)
		}

		var res struct {
			Interview models.Interview `json:"interview"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			t.Fatalf("Failed to decode interview: %v", err)
		}

		if res.Interview.ID != createdInterviewID {
			t.Errorf("Expected ID %s, got %s", createdInterviewID, res.Interview.ID)
		}
	})

	var createdSessionID string
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
		resp, err := client.Post(baseURL+"/api/candidates", "application/json", bytes.NewBuffer(body))
		if err != nil {
			t.Fatalf("Failed POST /api/candidates: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			t.Fatalf("Expected 200 OK, got %d: %s", resp.StatusCode, string(respBody))
		}

		var res struct {
			Session models.CandidateSession `json:"session"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			t.Fatalf("Failed to decode session: %v", err)
		}

		if res.Session.ID == "" {
			t.Errorf("Expected candidate session ID")
		}
		if res.Session.InterviewID != createdInterviewID {
			t.Errorf("Expected interview_id %s, got %s", createdInterviewID, res.Session.InterviewID)
		}

		createdSessionID = res.Session.ID
		t.Logf("Created Candidate Session ID: %s", createdSessionID)
	})

	// 15. GET /api/candidates/:id (Get candidate session by ID)
	t.Run("GET /api/candidates/:id", func(t *testing.T) {
		if createdSessionID == "" {
			t.Skip("No candidate session ID")
		}

		resp, err := client.Get(fmt.Sprintf("%s/api/candidates/%s", baseURL, createdSessionID))
		if err != nil {
			t.Fatalf("Failed GET /api/candidates/:id: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", resp.StatusCode)
		}

		var res struct {
			Session models.CandidateSession `json:"session"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			t.Fatalf("Failed to decode session: %v", err)
		}

		if res.Session.ID != createdSessionID {
			t.Errorf("Expected Session ID %s, got %s", createdSessionID, res.Session.ID)
		}
	})

	// 16. GET /api/candidates/interview/:interviewID (Get candidate sessions by Interview ID)
	t.Run("GET /api/candidates/interview/:interviewID", func(t *testing.T) {
		if createdInterviewID == "" {
			t.Skip("No created interview ID")
		}

		resp, err := client.Get(fmt.Sprintf("%s/api/candidates/interview/%s", baseURL, createdInterviewID))
		if err != nil {
			t.Fatalf("Failed GET /api/candidates/interview/:interviewID: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", resp.StatusCode)
		}

		var res struct {
			Sessions []models.CandidateSession `json:"sessions"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			t.Fatalf("Failed to decode candidate sessions list: %v", err)
		}

		if len(res.Sessions) == 0 {
			t.Errorf("Expected at least 1 candidate session for interview %s", createdInterviewID)
		}
	})
}
