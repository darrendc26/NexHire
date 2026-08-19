package candidate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"nexhire/backend/models"
)

type Repository interface {
	CreateSession(ctx context.Context, session *models.CandidateSession) error
	GetSessionByID(ctx context.Context, id string) (*models.CandidateSession, error)
	GetSessionByTokenHash(ctx context.Context, tokenHash string) (*models.CandidateSession, error)
	GetSessionsByInterviewID(ctx context.Context, interviewID string) ([]models.CandidateSession, error)
	GetInterviewByShareToken(ctx context.Context, shareToken string) (*models.Interview, error)
	GetInterviewByID(ctx context.Context, id string) (*models.Interview, error)
}

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) Repository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateSession(ctx context.Context, session *models.CandidateSession) error {
	if session.StartedAt.IsZero() {
		session.StartedAt = time.Now()
	}

	query := `
		INSERT INTO candidate_sessions (id, interview_id, name, email, token_hash, status, started_at, finished_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.ExecContext(
		ctx,
		query,
		session.ID,
		session.InterviewID,
		session.Name,
		session.Email,
		session.TokenHash,
		session.Status,
		session.StartedAt,
		session.FinishedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to insert candidate session: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetSessionByID(ctx context.Context, id string) (*models.CandidateSession, error) {
	query := `
		SELECT id, interview_id, name, email, COALESCE(token_hash, ''), status, started_at, finished_at
		FROM candidate_sessions
		WHERE id = $1
	`
	var s models.CandidateSession
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&s.ID,
		&s.InterviewID,
		&s.Name,
		&s.Email,
		&s.TokenHash,
		&s.Status,
		&s.StartedAt,
		&s.FinishedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("session not found")
		}
		return nil, fmt.Errorf("failed to fetch session: %w", err)
	}

	return &s, nil
}

func (r *PostgresRepository) GetSessionByTokenHash(ctx context.Context, tokenHash string) (*models.CandidateSession, error) {
	query := `
		SELECT id, interview_id, name, email, COALESCE(token_hash, ''), status, started_at, finished_at
		FROM candidate_sessions
		WHERE token_hash = $1
	`
	var s models.CandidateSession
	err := r.db.QueryRowContext(ctx, query, tokenHash).Scan(
		&s.ID,
		&s.InterviewID,
		&s.Name,
		&s.Email,
		&s.TokenHash,
		&s.Status,
		&s.StartedAt,
		&s.FinishedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("session not found")
		}
		return nil, fmt.Errorf("failed to fetch session by token hash: %w", err)
	}

	return &s, nil
}

func (r *PostgresRepository) GetSessionsByInterviewID(ctx context.Context, interviewID string) ([]models.CandidateSession, error) {
	query := `
		SELECT id, interview_id, name, email, COALESCE(token_hash, ''), status, started_at, finished_at
		FROM candidate_sessions
		WHERE interview_id = $1
		ORDER BY started_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, interviewID)
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions: %w", err)
	}
	defer rows.Close()

	var sessions []models.CandidateSession
	for rows.Next() {
		var s models.CandidateSession
		if err := rows.Scan(
			&s.ID,
			&s.InterviewID,
			&s.Name,
			&s.Email,
			&s.TokenHash,
			&s.Status,
			&s.StartedAt,
			&s.FinishedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan session row: %w", err)
		}
		sessions = append(sessions, s)
	}

	return sessions, nil
}

func (r *PostgresRepository) GetInterviewByShareToken(ctx context.Context, shareToken string) (*models.Interview, error) {
	query := `
		SELECT id, recruiter_id, title, role, description, difficulty, duration, voice_enabled, status, share_token, created_at
		FROM interviews
		WHERE share_token = $1
	`
	var i models.Interview
	err := r.db.QueryRowContext(ctx, query, shareToken).Scan(
		&i.ID,
		&i.RecruiterID,
		&i.Title,
		&i.Role,
		&i.Description,
		&i.Difficulty,
		&i.Duration,
		&i.VoiceEnabled,
		&i.Status,
		&i.ShareToken,
		&i.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("interview not found")
		}
		return nil, fmt.Errorf("failed to fetch interview: %w", err)
	}

	return &i, nil
}

func (r *PostgresRepository) GetInterviewByID(ctx context.Context, id string) (*models.Interview, error) {
	query := `
		SELECT id, recruiter_id, title, role, description, difficulty, duration, voice_enabled, status, share_token, created_at
		FROM interviews
		WHERE id = $1
	`
	var i models.Interview
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&i.ID,
		&i.RecruiterID,
		&i.Title,
		&i.Role,
		&i.Description,
		&i.Difficulty,
		&i.Duration,
		&i.VoiceEnabled,
		&i.Status,
		&i.ShareToken,
		&i.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("interview not found")
		}
		return nil, fmt.Errorf("failed to fetch interview by id: %w", err)
	}

	return &i, nil
}
