package ai

import (
	"context"
	"nexhire/backend/models"
	"time"
)

type Service struct {
	provider AIProvider
}

func NewService(provider AIProvider) *Service {
	return &Service{provider: provider}
}

func (s *Service) GenerateInitialQuestion(ctx context.Context, input InterviewContext) (*InterviewTurn, error) {
	return s.provider.GenerateInitialQuestion(ctx, input)
}

const (
	OutroCompleted = "Thank you for completing the interview. Your responses have been recorded and will be reviewed by the hiring team."
	OutroExpired   = "Thank you. Your interview time has ended. Your responses have been recorded and will be reviewed by the hiring team."
)

func (s *Service) ProcessAnswer(
	ctx context.Context,
	session *models.CandidateSession,
	input InterviewContext,
) (*InterviewTurn, error) {

	// Hard deadline check.
	// If the interview has already expired, don't make an LLM call.
	if session != nil && !session.ExpiresAt.IsZero() {
		if time.Now().After(session.ExpiresAt) {
			return &InterviewTurn{
				Score:          0,
				Feedback:       "Your interview time has expired.",
				NextQuestion:   OutroExpired,
				ShouldContinue: false,
			}, nil
		}
	}

	turn, err := s.provider.ProcessAnswer(ctx, input)
	if err != nil {
		return nil, err
	}

	// If we're now within the final minute, don't start another question.
	if session != nil && !session.ExpiresAt.IsZero() {
		if time.Until(session.ExpiresAt) < time.Minute {
			turn.ShouldContinue = false
			turn.NextQuestion = OutroCompleted
			turn.Feedback = "You have less than one minute remaining. The interview is concluding."
		}
	}

	return turn, nil
}

func (s *Service) GenerateReport(ctx context.Context, input ReportContext) (*InterviewReport, error) {
	return s.provider.GenerateReport(ctx, input)
}
