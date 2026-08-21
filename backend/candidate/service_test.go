package candidate

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"nexhire/backend/ai"
	"nexhire/backend/models"
)

type mockRepo struct {
	mu         sync.Mutex
	interviews map[string]*models.Interview
	sessions   map[string]*models.CandidateSession
	responses  map[string][]models.CandidateResponse
	reports    map[string]*models.CandidateReport
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		interviews: make(map[string]*models.Interview),
		sessions:   make(map[string]*models.CandidateSession),
		responses:  make(map[string][]models.CandidateResponse),
		reports:    make(map[string]*models.CandidateReport),
	}
}

func (m *mockRepo) CreateSession(ctx context.Context, session *models.CandidateSession) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[session.TokenHash] = session
	return nil
}

func (m *mockRepo) GetSessionByID(ctx context.Context, id string) (*models.CandidateSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.sessions {
		if s.ID == id {
			return s, nil
		}
	}
	return nil, errors.New("session not found")
}

func (m *mockRepo) GetSessionByTokenHash(ctx context.Context, tokenHash string) (*models.CandidateSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[tokenHash]
	if !ok {
		return nil, errors.New("session not found")
	}
	return s, nil
}

func (m *mockRepo) GetSessionsByInterviewID(ctx context.Context, interviewID string) ([]models.CandidateSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var list []models.CandidateSession
	for _, s := range m.sessions {
		if s.InterviewID == interviewID {
			list = append(list, *s)
		}
	}
	return list, nil
}

func (m *mockRepo) GetInterviewByShareToken(ctx context.Context, shareToken string) (*models.Interview, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, i := range m.interviews {
		if i.ShareToken == shareToken {
			return i, nil
		}
	}
	return nil, errors.New("interview not found")
}

func (m *mockRepo) GetInterviewByID(ctx context.Context, id string) (*models.Interview, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	i, ok := m.interviews[id]
	if !ok {
		return nil, errors.New("interview not found")
	}
	return i, nil
}

func (m *mockRepo) UpdateSessionStatus(ctx context.Context, sessionID string, status models.SessionStatus, finishedAt *time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.sessions {
		if s.ID == sessionID {
			s.Status = status
			s.FinishedAt = finishedAt
			return nil
		}
	}
	return errors.New("session not found")
}

func (m *mockRepo) CreateResponse(ctx context.Context, resp *models.CandidateResponse) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses[resp.SessionID] = append(m.responses[resp.SessionID], *resp)
	return nil
}

func (m *mockRepo) UpdateResponseEvaluation(
	ctx context.Context,
	id string,
	answer string,
	score float64,
	feedback string,
	strengths []string,
	weaknesses []string,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for sessionID, list := range m.responses {
		for i, r := range list {
			if r.ID == id {
				r.Answer = answer
				r.Score = &score
				r.Feedback = feedback
				r.Strengths = strengths
				r.Weaknesses = weaknesses
				m.responses[sessionID][i] = r
				return nil
			}
		}
	}
	return errors.New("response not found")
}

func (m *mockRepo) GetResponsesBySessionID(ctx context.Context, sessionID string) ([]models.CandidateResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.responses[sessionID], nil
}

func (m *mockRepo) GetPendingResponse(ctx context.Context, sessionID string) (*models.CandidateResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.responses[sessionID] {
		if r.Answer == "" {
			cp := r
			return &cp, nil
		}
	}
	return nil, errors.New("no pending question found")
}

func (m *mockRepo) CreateReport(ctx context.Context, report *models.CandidateReport) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reports[report.SessionID] = report
	return nil
}

func (m *mockRepo) GetReportBySessionID(ctx context.Context, sessionID string) (*models.CandidateReport, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rep, ok := m.reports[sessionID]
	if !ok {
		return nil, errors.New("report not found")
	}
	return rep, nil
}

type mockAIProvider struct {
	turnCount int
}

func (p *mockAIProvider) GenerateInitialQuestion(ctx context.Context, input ai.InterviewContext) (*ai.InterviewTurn, error) {
	return &ai.InterviewTurn{
		NextQuestion:   "What is Go interface?",
		QuestionType:   "technical",
		ShouldContinue: true,
	}, nil
}

func (p *mockAIProvider) ProcessAnswer(ctx context.Context, input ai.InterviewContext) (*ai.InterviewTurn, error) {
	p.turnCount++
	switch p.turnCount {
	case 1:
		return &ai.InterviewTurn{
			Score:          8.5,
			Feedback:       "Good explanation of interfaces.",
			Strengths:      []string{"Go interfaces"},
			NextQuestion:   "Explain goroutines and concurrency.",
			QuestionType:   "technical",
			ShouldContinue: true,
		}, nil
	case 2:
		return &ai.InterviewTurn{
			Score:          9.0,
			Feedback:       "Excellent description of concurrency.",
			Strengths:      []string{"Goroutines"},
			NextQuestion:   "How do channels synchronize goroutines?",
			QuestionType:   "technical",
			ShouldContinue: true,
		}, nil
	default:
		return &ai.InterviewTurn{
			Score:          8.0,
			Feedback:       "Solid grasp of channels.",
			Strengths:      []string{"Go channels"},
			NextQuestion:   "Thank you for completing the interview.",
			QuestionType:   "technical",
			ShouldContinue: false,
		}, nil
	}
}

func (p *mockAIProvider) GenerateReport(ctx context.Context, input ai.ReportContext) (*ai.InterviewReport, error) {
	return &ai.InterviewReport{
		OverallScore:   88.0,
		Recommendation: ai.Hire,
		Summary:        "Strong backend engineer with solid Go knowledge.",
		Strengths:      []string{"Go interfaces", "Concurrency", "Channels"},
		Weaknesses:     []string{"None major"},
		Skills: []ai.SkillScore{
			{Skill: "Go Language", Score: 88.0, Feedback: "Demonstrated strong understanding."},
		},
	}, nil
}

func TestCandidateService_FullInterviewTurnCycle(t *testing.T) {
	repo := newMockRepo()
	aiProv := &mockAIProvider{}
	aiSvc := ai.NewService(aiProv)
	svc := NewService(repo, aiSvc)

	ctx := context.Background()

	// 1. Setup Interview
	interview := &models.Interview{
		ID:          "int_123",
		RecruiterID: "rec_1",
		Title:       "Backend Engineer",
		Role:        "Go Developer",
		Difficulty:  models.Medium,
		Duration:    15,
		Status:      models.Active,
		ShareToken:  "token_123",
	}
	repo.interviews[interview.ID] = interview

	// 2. Start Candidate Session
	session, err := svc.StartSession(ctx, "token_123", StartSessionRequest{
		Name:  "Jane Candidate",
		Email: "jane@example.com",
	})
	if err != nil {
		t.Fatalf("StartSession failed: %v", err)
	}

	token := session.RawToken

	// 3. Start Session Question (Turn 1 Q1)
	startRes, err := svc.StartSessionQuestion(ctx, token)
	if err != nil {
		t.Fatalf("StartSessionQuestion failed: %v", err)
	}

	if startRes.Question.Text != "What is Go interface?" {
		t.Errorf("Expected initial question 'What is Go interface?', got %q", startRes.Question.Text)
	}

	// 4. Submit Answer 1
	ans1Res, err := svc.SubmitAnswer(ctx, token, "Interfaces define behavior by specifying method signatures.")
	if err != nil {
		t.Fatalf("SubmitAnswer 1 failed: %v", err)
	}
	if !ans1Res.ShouldContinue {
		t.Errorf("Expected shouldContinue = true after Q1")
	}
	if ans1Res.NextQuestion == nil || ans1Res.NextQuestion.Text != "Explain goroutines and concurrency." {
		t.Errorf("Expected Q2 'Explain goroutines...', got %+v", ans1Res.NextQuestion)
	}
	if ans1Res.Progress.QuestionsAsked != 1 || ans1Res.Progress.TimeRemaining <= 0 {
		t.Errorf("Expected 1 asked with positive time remaining, got %+v", ans1Res.Progress)
	}

	// 5. Submit Answer 2
	ans2Res, err := svc.SubmitAnswer(ctx, token, "Goroutines are lightweight threads managed by the Go runtime.")
	if err != nil {
		t.Fatalf("SubmitAnswer 2 failed: %v", err)
	}
	if !ans2Res.ShouldContinue {
		t.Errorf("Expected shouldContinue = true after Q2")
	}
	if ans2Res.NextQuestion == nil || ans2Res.NextQuestion.Text != "How do channels synchronize goroutines?" {
		t.Errorf("Expected Q3 'How do channels...', got %+v", ans2Res.NextQuestion)
	}
	if ans2Res.Progress.QuestionsAsked != 2 || ans2Res.Progress.TimeRemaining <= 0 {
		t.Errorf("Expected 2 asked with positive time remaining, got %+v", ans2Res.Progress)
	}

	// 6. Submit Answer 3 (Final Answer)
	ans3Res, err := svc.SubmitAnswer(ctx, token, "Channels allow goroutines to communicate and synchronize execution.")
	if err != nil {
		t.Fatalf("SubmitAnswer 3 failed: %v", err)
	}
	if ans3Res.ShouldContinue {
		t.Errorf("Expected shouldContinue = false after Q3")
	}
	if ans3Res.Progress.QuestionsAsked != 3 {
		t.Errorf("Expected 3 asked, got %+v", ans3Res.Progress)
	}

	// 7. Get Report
	rep, err := svc.GetReport(ctx, token)
	if err != nil {
		t.Fatalf("GetReport failed: %v", err)
	}
	if rep.OverallScore != 85.0 {
		t.Errorf("Expected overall score 85.0, got %f", rep.OverallScore)
	}
	if rep.Recommendation != "hire" {
		t.Errorf("Expected recommendation 'hire', got %s", rep.Recommendation)
	}
}
