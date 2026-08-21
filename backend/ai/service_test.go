package ai

import (
	"context"
	"nexhire/backend/models"
	"testing"
	"time"
)

type mockAIProvider struct {
	called bool
}

func (m *mockAIProvider) GenerateInitialQuestion(ctx context.Context, input InterviewContext) (*InterviewTurn, error) {
	return &InterviewTurn{
		NextQuestion:   "Initial question?",
		QuestionType:   "technical",
		ShouldContinue: true,
	}, nil
}

func (m *mockAIProvider) ProcessAnswer(ctx context.Context, input InterviewContext) (*InterviewTurn, error) {
	m.called = true
	return &InterviewTurn{
		Score:          8.5,
		Feedback:       "Good answer",
		NextQuestion:   "Follow up question?",
		QuestionType:   "technical",
		ShouldContinue: true,
	}, nil
}

func (m *mockAIProvider) GenerateReport(ctx context.Context, input ReportContext) (*InterviewReport, error) {
	return &InterviewReport{
		OverallScore:   85.0,
		Recommendation: Hire,
	}, nil
}

func TestProcessAnswer_PreChecksAvoidLLMCall(t *testing.T) {
	mock := &mockAIProvider{}
	svc := NewService(mock)
	ctx := context.Background()

	// Test 1: Expired session (remaining <= 0)
	expiredSession := &models.CandidateSession{
		ExpiresAt: time.Now().Add(-5 * time.Minute),
	}
	turn, err := svc.ProcessAnswer(ctx, expiredSession, InterviewContext{
		TimeRemaining:  0,
		QuestionsAsked: 2,
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if mock.called {
		t.Errorf("Expected LLM provider NOT to be called when session is expired")
	}
	if turn.ShouldContinue {
		t.Errorf("Expected ShouldContinue = false for expired session")
	}

	// Test 2: Session with < 1 min remaining -> LLM evaluates last answer, but ShouldContinue = false and NextQuestion set to OutroExpired
	nearEndSession := &models.CandidateSession{
		ExpiresAt: time.Now().Add(30 * time.Second),
	}
	mock.called = false
	turn, err = svc.ProcessAnswer(ctx, nearEndSession, InterviewContext{
		TimeRemaining:  30,
		QuestionsAsked: 2,
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !mock.called {
		t.Errorf("Expected LLM provider to be called to evaluate last answer when remaining time < 1 minute")
	}
	if turn.ShouldContinue {
		t.Errorf("Expected ShouldContinue = false when remaining time < 1 minute")
	}
	if turn.NextQuestion != OutroCompleted {
		t.Errorf("Expected NextQuestion = %q when remaining time < 1 minute, got %q", OutroCompleted, turn.NextQuestion)
	}

	// Test 3: Normal session with plenty of time -> should invoke LLM provider
	mock.called = false
	normalSession := &models.CandidateSession{
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	turn, err = svc.ProcessAnswer(ctx, normalSession, InterviewContext{
		TimeRemaining:  600,
		QuestionsAsked: 2,
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !mock.called {
		t.Errorf("Expected LLM provider to be called for normal session")
	}
	if !turn.ShouldContinue {
		t.Errorf("Expected ShouldContinue = true for normal session with questions remaining")
	}
}
