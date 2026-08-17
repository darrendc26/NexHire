package interview

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"nexhire/backend/models"

	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, interview *models.Interview) error
	GetByID(ctx context.Context, id string) (*models.Interview, error)
	GetByRecruiterID(ctx context.Context, recruiterID string) ([]models.Interview, error)
	Update(ctx context.Context, interview *models.Interview) error
	Delete(ctx context.Context, id string) error
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}

type CreateInterviewRequest struct {
	Title        string            `json:"title" binding:"required"`
	Role         string            `json:"role" binding:"required"`
	Description  string            `json:"description"`
	Difficulty   models.Difficulty `json:"difficulty" binding:"required"`
	Duration     int               `json:"duration" binding:"required"`
	VoiceEnabled bool              `json:"voice_enabled"`
}

func (s *Service) Create(
	ctx context.Context,
	recruiterID string,
	req CreateInterviewRequest,
) (*models.Interview, error) {
	if recruiterID == "" {
		return nil, errors.New("recruiterID is required")
	}
	if req.Title == "" {
		return nil, errors.New("title is required")
	}

	if !validDifficulty(req.Difficulty) {
		return nil, errors.New("invalid difficulty")
	}

	if req.Difficulty == "" {
		return nil, errors.New("difficulty is required")
	}
	if req.Duration < 0 {
		return nil, errors.New("duration is required")
	}

	shareToken, err := generateShareToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate share token: %w", err)
	}

	newInterview := &models.Interview{
		ID:           generateID(),
		RecruiterID:  recruiterID,
		Title:        req.Title,
		Role:         req.Role,
		Description:  req.Description,
		Difficulty:   req.Difficulty,
		Duration:     req.Duration,
		VoiceEnabled: req.VoiceEnabled,
		Status:       models.Draft,
		ShareToken:   shareToken,
	}
	if err := s.repo.Create(ctx, newInterview); err != nil {
		return nil, err
	}
	return newInterview, nil
}

func generateShareToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func generateID() string {
	return uuid.New().String()
}

func validDifficulty(d models.Difficulty) bool {
	switch d {
	case models.Easy, models.Medium, models.Hard:
		return true
	default:
		return false
	}
}

func (s *Service) GetByID(ctx context.Context, id string, recruiterID string) (*models.Interview, error) {
	interview, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if interview.RecruiterID != recruiterID {
		return nil, errors.New("you are not authorized to access this interview")
	}

	return interview, nil
}

func (s *Service) GetByRecruiterID(ctx context.Context, recruiterID string) ([]models.Interview, error) {
	interviews, err := s.repo.GetByRecruiterID(ctx, recruiterID)
	if err != nil {
		return nil, err
	}
	return interviews, nil
}
