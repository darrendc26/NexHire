package candidate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"nexhire/backend/ai"
	"nexhire/backend/models"
	"nexhire/backend/utils"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	OutroCompleted = "Thank you for completing the interview. Your responses have been recorded and will be reviewed by the hiring team."
	OutroExpired   = "Thank you. Your interview time has ended. Your responses have been recorded and will be reviewed by the hiring team."
)

type Service struct {
	repo        Repository
	aiService   *ai.Service
	redisClient *redis.Client
}

func NewService(repo Repository, aiService *ai.Service, redisClient *redis.Client) *Service {
	return &Service{
		repo:        repo,
		aiService:   aiService,
		redisClient: redisClient,
	}
}

func (s *Service) SetAIService(aiService *ai.Service) {
	s.aiService = aiService
}

func HashToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}

type StartSessionRequest struct {
	InterviewID string `json:"interview_id"`
	ShareToken  string `json:"share_token"`
	Name        string `json:"name" binding:"required"`
	Email       string `json:"email" binding:"required"`
}

type QuestionDTO struct {
	ID    string `json:"id"`
	Text  string `json:"text"`
	Order int    `json:"order"`
}

type EvaluationDTO struct {
	Score      float64  `json:"score"`
	Feedback   string   `json:"feedback"`
	Strengths  []string `json:"strengths"`
	Weaknesses []string `json:"weaknesses"`
}

type StartQuestionResponse struct {
	Question QuestionDTO            `json:"question"`
	Progress models.SessionProgress `json:"progress"`
}

type SubmitAnswerResponse struct {
	Evaluation     *EvaluationDTO         `json:"evaluation,omitempty"`
	NextQuestion   *QuestionDTO           `json:"next_question,omitempty"`
	OutroMessage   string                 `json:"outro_message,omitempty"`
	ShouldContinue bool                   `json:"should_continue"`
	Progress       models.SessionProgress `json:"progress"`
}

func (s *Service) StartSession(
	ctx context.Context,
	shareToken string,
	req StartSessionRequest,
) (*models.CandidateSession, error) {

	if shareToken == "" {
		shareToken = req.ShareToken
	}

	if req.Name == "" || req.Email == "" {
		return nil, errors.New("name and email are required")
	}

	var (
		interview *models.Interview
		err       error
	)

	if shareToken != "" {
		interview, err = s.repo.GetInterviewByShareToken(ctx, shareToken)
	} else if req.InterviewID != "" {
		interview, err = s.repo.GetInterviewByID(ctx, req.InterviewID)
	} else {
		return nil, errors.New("share token or interview_id is required")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to find interview: %w", err)
	}

	if interview.Status == models.Closed {
		return nil, errors.New("interview is closed")
	}

	if err := s.requireVerifiedEmail(ctx, req.Email, interview.ID); err != nil {
		return nil, err
	}

	duration := interview.Duration
	if duration <= 0 {
		duration = 15 // Default 15 minutes
	}

	now := time.Now()
	rawToken := uuid.New().String()
	tokenHash := HashToken(rawToken)

	session := &models.CandidateSession{
		ID:          uuid.New().String(),
		InterviewID: interview.ID,
		Name:        req.Name,
		Email:       req.Email,
		TokenHash:   tokenHash,
		RawToken:    rawToken,
		Status:      models.SessionActive,
		StartedAt:   now,
		ExpiresAt:   now.Add(time.Duration(duration) * time.Minute),
		FinishedAt:  nil,
	}

	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create candidate session: %w", err)
	}

	s.consumeVerifiedEmail(ctx, req.Email, interview.ID)

	return session, nil
}

func (s *Service) StartSessionQuestion(ctx context.Context, rawToken string) (*StartQuestionResponse, error) {
	session, err := s.GetSessionByToken(ctx, rawToken)
	if err != nil {
		return nil, err
	}

	if session.Status == models.SessionCompleted {
		return nil, errors.New("candidate session has already been completed")
	}

	if !session.ExpiresAt.IsZero() && time.Now().After(session.ExpiresAt) {
		return nil, errors.New("candidate session time has expired")
	}

	timeRemaining := timeRemainingSeconds(session)

	responses, err := s.repo.GetResponsesBySessionID(ctx, session.ID)
	if err == nil && len(responses) > 0 {
		pending, pendingErr := s.repo.GetPendingResponse(ctx, session.ID)
		if pendingErr == nil && pending != nil {
			return &StartQuestionResponse{
				Question: QuestionDTO{
					ID:    pending.ID,
					Text:  pending.Question,
					Order: pending.QuestionOrder,
				},
				Progress: models.SessionProgress{
					QuestionsAsked: s.questionsAnsweredCount(responses),
					TimeRemaining:  timeRemaining,
				},
			}, nil
		}
	}

	interview, err := s.repo.GetInterviewByID(ctx, session.InterviewID)
	if err != nil {
		return nil, fmt.Errorf("failed to load interview: %w", err)
	}

	if s.aiService == nil {
		return nil, errors.New("AI service not configured")
	}

	aiCtx := ai.InterviewContext{
		Role:           interview.Role,
		Difficulty:     string(interview.Difficulty),
		QuestionsAsked: 0,
		TimeRemaining:  timeRemaining,
	}

	qCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	turn, err := s.aiService.GenerateInitialQuestion(qCtx, aiCtx)
	if err != nil || turn == nil || strings.TrimSpace(turn.NextQuestion) == "" {
		role := interview.Role
		if strings.TrimSpace(role) == "" {
			role = "this"
		}
		turn = &ai.InterviewTurn{
			NextQuestion: fmt.Sprintf(
				"Let's begin. For the %s role, tell me about a recent project you owned — what problem you solved, and the decisions you made along the way.",
				role,
			),
			QuestionType:   "opening",
			ShouldContinue: true,
		}
	}

	qResp := &models.CandidateResponse{
		ID:            uuid.New().String(),
		SessionID:     session.ID,
		Question:      turn.NextQuestion,
		QuestionType:  turn.QuestionType,
		QuestionOrder: 1,
		CreatedAt:     time.Now(),
	}

	if err := s.repo.CreateResponse(ctx, qResp); err != nil {
		return nil, fmt.Errorf("failed to persist initial question: %w", err)
	}

	return &StartQuestionResponse{
		Question: QuestionDTO{
			ID:    qResp.ID,
			Text:  qResp.Question,
			Order: 1,
		},
		Progress: models.SessionProgress{
			QuestionsAsked: 0,
			TimeRemaining:  timeRemaining,
		},
	}, nil
}

func (s *Service) SubmitAnswer(ctx context.Context, rawToken string, answerText string) (*SubmitAnswerResponse, error) {
	session, err := s.GetSessionByToken(ctx, rawToken)
	if err != nil {
		return nil, err
	}

	if session.Status == models.SessionCompleted {
		return nil, errors.New("candidate session has already been completed")
	}

	if !session.ExpiresAt.IsZero() && time.Now().After(session.ExpiresAt) {
		return s.completeExpiredSession(ctx, session)
	}

	pendingResp, err := s.repo.GetPendingResponse(ctx, session.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to find pending question: %w", err)
	}

	allResponses, err := s.repo.GetResponsesBySessionID(ctx, session.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch session responses: %w", err)
	}

	var answeredResponses []models.CandidateResponse
	var pastStrengths []string
	var pastWeaknesses []string
	var pastTopics []string

	for _, r := range allResponses {
		if r.Answer != "" && r.ID != pendingResp.ID {
			answeredResponses = append(answeredResponses, r)
			pastStrengths = append(pastStrengths, r.Strengths...)
			pastWeaknesses = append(pastWeaknesses, r.Weaknesses...)
			pastTopics = append(pastTopics, r.Question)
		}
	}

	questionsAsked := len(answeredResponses) + 1

	interview, err := s.repo.GetInterviewByID(ctx, session.InterviewID)
	if err != nil {
		return nil, fmt.Errorf("failed to load interview: %w", err)
	}

	if s.aiService == nil {
		return nil, errors.New("AI service not configured")
	}

	timeRemaining := timeRemainingSeconds(session)

	aiCtx := ai.InterviewContext{
		Role:            interview.Role,
		Difficulty:      string(interview.Difficulty),
		CurrentQuestion: pendingResp.Question,
		LastAnswer:      answerText,
		PreviousTopics:  pastTopics,
		Strengths:       pastStrengths,
		Weaknesses:      pastWeaknesses,
		QuestionsAsked:  questionsAsked,
		TimeRemaining:   timeRemaining,
	}

	turn, err := s.aiService.ProcessAnswer(ctx, session, aiCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to process answer via AI: %w", err)
	}

	if err := s.repo.UpdateResponseEvaluation(
		ctx,
		pendingResp.ID,
		answerText,
		turn.Score,
		turn.Feedback,
		turn.Strengths,
		turn.Weaknesses,
	); err != nil {
		return nil, fmt.Errorf("failed to save evaluation: %w", err)
	}

	eval := &EvaluationDTO{
		Score:      turn.Score,
		Feedback:   turn.Feedback,
		Strengths:  turn.Strengths,
		Weaknesses: turn.Weaknesses,
	}

	remAfter := timeRemainingSeconds(session)
	shouldContinue := turn.ShouldContinue && (remAfter > 60) && (turn.NextQuestion != "")

	if shouldContinue {
		nextOrder := questionsAsked + 1
		nextQResp := &models.CandidateResponse{
			ID:            uuid.New().String(),
			SessionID:     session.ID,
			Question:      turn.NextQuestion,
			QuestionType:  turn.QuestionType,
			QuestionOrder: nextOrder,
			CreatedAt:     time.Now(),
		}

		if err := s.repo.CreateResponse(ctx, nextQResp); err != nil {
			return nil, fmt.Errorf("failed to persist next question: %w", err)
		}

		return &SubmitAnswerResponse{
			Evaluation: eval,
			NextQuestion: &QuestionDTO{
				ID:    nextQResp.ID,
				Text:  nextQResp.Question,
				Order: nextOrder,
			},
			ShouldContinue: true,
			Progress: models.SessionProgress{
				QuestionsAsked: questionsAsked,
				TimeRemaining:  remAfter,
			},
		}, nil
	}

	return s.completeSession(ctx, session, interview, eval, turn, questionsAsked, turn.NextQuestion)
}

func (s *Service) completeSession(
	ctx context.Context,
	session *models.CandidateSession,
	interview *models.Interview,
	eval *EvaluationDTO,
	turn *ai.InterviewTurn,
	questionsAsked int,
	outro string,
) (*SubmitAnswerResponse, error) {

	now := time.Now()
	_ = s.repo.UpdateSessionStatus(ctx, session.ID, models.SessionCompleted, &now)

	_ = s.generateAndSaveReport(ctx, session, interview)

	if outro == "" {
		if turn != nil && turn.NextQuestion != "" {
			outro = turn.NextQuestion
		} else {
			outro = OutroCompleted
		}
	}

	return &SubmitAnswerResponse{
		Evaluation:     eval,
		OutroMessage:   outro,
		ShouldContinue: false,
		Progress: models.SessionProgress{
			QuestionsAsked: questionsAsked,
			TimeRemaining:  timeRemainingSeconds(session),
		},
	}, nil
}

func (s *Service) completeExpiredSession(
	ctx context.Context,
	session *models.CandidateSession,
) (*SubmitAnswerResponse, error) {

	interview, err := s.repo.GetInterviewByID(ctx, session.InterviewID)
	if err != nil {
		return nil, fmt.Errorf("failed to load interview: %w", err)
	}

	now := time.Now()
	_ = s.repo.UpdateSessionStatus(ctx, session.ID, models.SessionCompleted, &now)

	_ = s.generateAndSaveReport(ctx, session, interview)

	responses, _ := s.repo.GetResponsesBySessionID(ctx, session.ID)

	return &SubmitAnswerResponse{
		OutroMessage:   OutroExpired,
		ShouldContinue: false,
		Progress: models.SessionProgress{
			QuestionsAsked: s.questionsAnsweredCount(responses),
			TimeRemaining:  0,
		},
	}, nil
}

func (s *Service) generateAndSaveReport(
	ctx context.Context,
	session *models.CandidateSession,
	interview *models.Interview,
) error {

	if s.aiService == nil {
		return errors.New("AI service not configured")
	}

	responses, err := s.repo.GetResponsesBySessionID(ctx, session.ID)
	if err != nil {
		return fmt.Errorf("failed to fetch responses for report: %w", err)
	}

	var qaList []ai.QuestionAnswer
	for _, r := range responses {
		if r.Answer != "" {
			score := 0.0
			if r.Score != nil {
				score = *r.Score
			}
			qaList = append(qaList, ai.QuestionAnswer{
				Question:   r.Question,
				Answer:     r.Answer,
				Score:      score,
				Feedback:   r.Feedback,
				Strengths:  r.Strengths,
				Weaknesses: r.Weaknesses,
			})
		}
	}

	var totalScore float64
	for _, qa := range qaList {
		totalScore += qa.Score
	}
	var overallScore float64
	if len(qaList) > 0 {
		overallScore = (totalScore / float64(len(qaList))) * 10.0
	}

	reportCtx := ai.ReportContext{
		Interview:    interview,
		Candidate:    session,
		Answers:      qaList,
		OverallScore: overallScore,
	}

	report, err := s.aiService.GenerateReport(ctx, reportCtx)
	if err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	if report == nil {
		return errors.New("AI returned empty report")
	}

	report.OverallScore = overallScore

	var skills []models.SkillScore
	for _, sk := range report.Skills {
		skills = append(skills, models.SkillScore{
			Skill:    sk.Skill,
			Score:    sk.Score,
			Feedback: sk.Feedback,
		})
	}

	candReport := &models.CandidateReport{
		ID:             uuid.New().String(),
		SessionID:      session.ID,
		OverallScore:   report.OverallScore,
		Recommendation: string(report.Recommendation),
		Summary:        report.Summary,
		Strengths:      report.Strengths,
		Weaknesses:     report.Weaknesses,
		Skills:         skills,
		CreatedAt:      time.Now(),
	}

	return s.repo.CreateReport(ctx, candReport)
}

func (s *Service) generateAndSaveFallbackReport(
	ctx context.Context,
	session *models.CandidateSession,
	interview *models.Interview,
) error {
	responses, err := s.repo.GetResponsesBySessionID(ctx, session.ID)
	if err != nil || len(responses) == 0 {
		return fmt.Errorf("no responses found for fallback report: %w", err)
	}

	var totalScore float64
	var count int
	var strengths []string
	var weaknesses []string

	for _, r := range responses {
		if r.Answer != "" {
			count++
			if r.Score != nil {
				totalScore += *r.Score
			}
			if len(r.Strengths) > 0 {
				strengths = append(strengths, r.Strengths...)
			}
			if len(r.Weaknesses) > 0 {
				weaknesses = append(weaknesses, r.Weaknesses...)
			}
		}
	}

	var overallScore float64
	if count > 0 {
		overallScore = (totalScore / float64(count)) * 10.0
	} else {
		overallScore = 50.0
	}

	recommendation := "maybe"
	if overallScore >= 75.0 {
		recommendation = "strong_hire"
	} else if overallScore >= 60.0 {
		recommendation = "hire"
	} else if overallScore < 40.0 {
		recommendation = "reject"
	}

	if len(strengths) == 0 {
		strengths = []string{"Demonstrated technical engagement during evaluation."}
	}
	if len(weaknesses) == 0 {
		weaknesses = []string{"Further technical probing recommended for complex scenarios."}
	}

	roleName := "Software Engineer"
	if interview != nil && interview.Role != "" {
		roleName = interview.Role
	}

	skills := []models.SkillScore{
		{Skill: "Technical Proficiency", Score: overallScore, Feedback: "Evaluated based on candidate answer completeness."},
		{Skill: "Problem Solving", Score: overallScore, Feedback: "Evaluated based on response reasoning."},
	}

	summary := fmt.Sprintf(
		"Candidate %s completed %d question(s) for the %s role. Overall technical evaluation score is %.1f/100.",
		session.Name, count, roleName, overallScore,
	)

	candReport := &models.CandidateReport{
		ID:             uuid.New().String(),
		SessionID:      session.ID,
		OverallScore:   overallScore,
		Recommendation: recommendation,
		Summary:        summary,
		Strengths:      strengths,
		Weaknesses:     weaknesses,
		Skills:         skills,
		CreatedAt:      time.Now(),
	}

	return s.repo.CreateReport(ctx, candReport)
}

func (s *Service) GetReport(ctx context.Context, rawToken string) (*models.CandidateReport, error) {
	session, err := s.GetSessionByToken(ctx, rawToken)
	if err != nil {
		return nil, err
	}

	return s.GetReportBySessionID(ctx, session.ID)
}

func (s *Service) GetReportBySessionID(ctx context.Context, sessionID string) (*models.CandidateReport, error) {
	report, err := s.repo.GetReportBySessionID(ctx, sessionID)
	if err == nil && report != nil {
		return report, nil
	}

	// Attempt on-demand report generation if missing
	session, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	interview, err := s.repo.GetInterviewByID(ctx, session.InterviewID)
	if err != nil {
		return nil, err
	}

	if genErr := s.generateAndSaveReport(ctx, session, interview); genErr != nil {
		if fbErr := s.generateAndSaveFallbackReport(ctx, session, interview); fbErr != nil {
			return nil, fmt.Errorf("failed to generate report: %w", fbErr)
		}
	}

	return s.repo.GetReportBySessionID(ctx, session.ID)
}

func (s *Service) GetSessionByID(ctx context.Context, id string) (*models.CandidateSession, error) {
	return s.repo.GetSessionByID(ctx, id)
}

func (s *Service) GetSessionByToken(ctx context.Context, rawToken string) (*models.CandidateSession, error) {
	if rawToken == "" {
		return nil, errors.New("session token is required")
	}

	tokenHash := HashToken(rawToken)
	return s.repo.GetSessionByTokenHash(ctx, tokenHash)
}

func (s *Service) ValidateActiveSession(ctx context.Context, rawToken string) (*models.CandidateSession, *models.Interview, error) {
	session, err := s.GetSessionByToken(ctx, rawToken)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid session token: %w", err)
	}

	if session.Status != models.SessionActive {
		return nil, nil, errors.New("candidate session is not active")
	}

	if !session.ExpiresAt.IsZero() && time.Now().After(session.ExpiresAt) {
		return nil, nil, errors.New("candidate session has expired")
	}

	interview, err := s.repo.GetInterviewByID(ctx, session.InterviewID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load interview: %w", err)
	}

	if interview.Status == models.Closed {
		return nil, nil, errors.New("interview is closed")
	}

	return session, interview, nil
}

func (s *Service) GetSessionsByInterviewID(ctx context.Context, interviewID string) ([]models.CandidateSession, error) {
	return s.repo.GetSessionsByInterviewID(ctx, interviewID)
}

func (s *Service) GetInterviewByShareToken(ctx context.Context, shareToken string) (*models.Interview, error) {
	return s.repo.GetInterviewByShareToken(ctx, shareToken)
}

func timeRemainingSeconds(session *models.CandidateSession) int64 {
	if session == nil || session.ExpiresAt.IsZero() {
		return 0
	}
	remaining := int64(time.Until(session.ExpiresAt).Seconds())
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (s *Service) questionsAnsweredCount(responses []models.CandidateResponse) int {
	count := 0
	for _, r := range responses {
		if r.Answer != "" {
			count++
		}
	}
	return count
}

var (
	ErrOTPExpired      = errors.New("OTP has expired or is invalid")
	ErrOTPInvalid      = errors.New("invalid OTP")
	ErrEmailUnverified = errors.New("email is not verified")
)

func emailIdentityHash(email string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(email))))
	return hex.EncodeToString(sum[:])
}

func otpRedisKey(interviewID, email string) string {
	return "nexhire:otp:" + interviewID + ":" + emailIdentityHash(email)
}

func verifiedRedisKey(interviewID, email string) string {
	return "nexhire:verified:" + interviewID + ":" + emailIdentityHash(email)
}

func (s *Service) SendEmailOtp(
	ctx context.Context,
	email string,
	interviewID string,
) error {
	if s.redisClient == nil {
		return errors.New("email verification is unavailable")
	}

	otp := rand.Intn(900000) + 100000
	otpHash := sha256.Sum256([]byte(strconv.Itoa(otp)))

	err := s.redisClient.Set(
		ctx,
		otpRedisKey(interviewID, email),
		hex.EncodeToString(otpHash[:]),
		5*time.Minute,
	).Err()
	if err != nil {
		return fmt.Errorf("failed to store OTP in Redis: %w", err)
	}

	if err := utils.NewService().SendOTP(ctx, email, strconv.Itoa(otp)); err != nil {
		_ = s.redisClient.Del(ctx, otpRedisKey(interviewID, email)).Err()
		return fmt.Errorf("failed to send verification email: %w", err)
	}

	return nil
}

func (s *Service) VerifyEmailOTP(
	ctx context.Context,
	email string,
	interviewID string,
	otp string,
) error {
	if s.redisClient == nil {
		return errors.New("email verification is unavailable")
	}

	otp = strings.TrimSpace(otp)
	if len(otp) != 6 {
		return ErrOTPInvalid
	}

	key := otpRedisKey(interviewID, email)
	storedHash, err := s.redisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		return ErrOTPExpired
	}
	if err != nil {
		return fmt.Errorf("failed to retrieve OTP from Redis: %w", err)
	}

	otpHash := sha256.Sum256([]byte(otp))
	if storedHash != hex.EncodeToString(otpHash[:]) {
		return ErrOTPInvalid
	}

	if err := s.redisClient.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete OTP from Redis: %w", err)
	}

	if err := s.redisClient.Set(ctx, verifiedRedisKey(interviewID, email), "1", 30*time.Minute).Err(); err != nil {
		return fmt.Errorf("failed to mark email as verified: %w", err)
	}

	return nil
}

func (s *Service) requireVerifiedEmail(ctx context.Context, email, interviewID string) error {
	if s.redisClient == nil {
		return errors.New("email verification is unavailable")
	}

	_, err := s.redisClient.Get(ctx, verifiedRedisKey(interviewID, email)).Result()
	if err == redis.Nil {
		return ErrEmailUnverified
	}
	if err != nil {
		return fmt.Errorf("failed to check email verification: %w", err)
	}
	return nil
}

func (s *Service) consumeVerifiedEmail(ctx context.Context, email, interviewID string) {
	if s.redisClient == nil {
		return
	}
	_ = s.redisClient.Del(ctx, verifiedRedisKey(interviewID, email)).Err()
}
