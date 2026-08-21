//go:build integration

package ai

import (
	"context"
	"os"
	"testing"

	"google.golang.org/genai"
)

func TestGeminiProvider_LiveIntegration(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping live Gemini integration test: GEMINI_API_KEY is not set.")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: apiKey})
	if err != nil {
		t.Fatalf("Failed to create genai client: %v", err)
	}

	model := os.Getenv("GEMINI_MODEL")
	if model == "" {
		model = "gemini-3.6-flash"
	}

	provider := NewGeminiProvider(client, model)
	svc := NewService(provider)

	// 1. Initial Question
	initCtx := InterviewContext{
		Role:           "Senior Backend Engineer (Go)",
		Difficulty:     "hard",
		QuestionsAsked: 0,
		TimeRemaining:  900,
	}

	turn1, err := svc.GenerateInitialQuestion(ctx, initCtx)
	if err != nil {
		t.Fatalf("GenerateInitialQuestion failed: %v", err)
	}
	if turn1.NextQuestion == "" {
		t.Errorf("Expected non-empty question from Gemini")
	}
	t.Logf("Initial Question from Gemini: %s", turn1.NextQuestion)

	// 2. Answer Turn 1
	answerCtx := InterviewContext{
		Role:            "Senior Backend Engineer (Go)",
		Difficulty:      "hard",
		CurrentQuestion: turn1.NextQuestion,
		LastAnswer:      "In Go, I use context.Context for cancellation timeouts and channels with select for synchronization. Mutexes are used for shared state memory protection.",
		QuestionsAsked:  1,
		TimeRemaining:   600,
	}

	turn2, err := provider.ProcessAnswer(ctx, answerCtx)
	if err != nil {
		t.Fatalf("ProcessAnswer failed: %v", err)
	}
	if turn2.Score < 0 || turn2.Score > 10 {
		t.Errorf("Invalid score: %f", turn2.Score)
	}
	t.Logf("Score: %.1f, Feedback: %s", turn2.Score, turn2.Feedback)

	// 3. Final Report Synthesis
	repCtx := ReportContext{
		Answers: []QuestionAnswer{
			{
				Question:   turn1.NextQuestion,
				Answer:     answerCtx.LastAnswer,
				Score:      turn2.Score,
				Feedback:   turn2.Feedback,
				Strengths:  turn2.Strengths,
				Weaknesses: turn2.Weaknesses,
			},
		},
		OverallScore: turn2.Score * 10.0,
	}

	rep, err := svc.GenerateReport(ctx, repCtx)
	if err != nil {
		t.Fatalf("GenerateReport failed: %v", err)
	}
	if rep.OverallScore < 0 || rep.OverallScore > 100 {
		t.Errorf("Invalid overall score: %f", rep.OverallScore)
	}
	t.Logf("Overall Score: %.1f, Recommendation: %s", rep.OverallScore, rep.Recommendation)
}
