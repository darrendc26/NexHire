package candidate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"nexhire/backend/models"

	"github.com/google/uuid"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
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

	var interview *models.Interview
	var err error

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
		StartedAt:   time.Now(),
	}

	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create candidate session: %w", err)
	}

	return session, nil
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

func (s *Service) GetSessionsByInterviewID(ctx context.Context, interviewID string) ([]models.CandidateSession, error) {
	return s.repo.GetSessionsByInterviewID(ctx, interviewID)
}

func (s *Service) GetInterviewByShareToken(ctx context.Context, shareToken string) (*models.Interview, error) {
	return s.repo.GetInterviewByShareToken(ctx, shareToken)
}
