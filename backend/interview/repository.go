package interview

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"nexhire/backend/models"

	"github.com/google/uuid"
)

// PostgresRepository implements Repository using PostgreSQL database
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository initializes a new PostgreSQL backed repository
func NewPostgresRepository(db *sql.DB) Repository {
	return &PostgresRepository{
		db: db,
	}
}

// NewRepository returns a PostgreSQL repository instance
func NewRepository(db *sql.DB) Repository {
	return NewPostgresRepository(db)
}

func (r *PostgresRepository) Create(ctx context.Context, interview *models.Interview) error {
	if interview.CreatedAt.IsZero() {
		interview.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO interviews (id, recruiter_id, title, role, description, difficulty, duration, voice_enabled, status, share_token, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		interview.ID,
		interview.RecruiterID,
		interview.Title,
		interview.Role,
		interview.Description,
		interview.Difficulty,
		interview.Duration,
		interview.VoiceEnabled,
		interview.Status,
		interview.ShareToken,
		interview.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create interview in postgres: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id string) (*models.Interview, error) {
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
		return nil, fmt.Errorf("failed to fetch interview: %w", err)
	}

	return &i, nil
}

func (r *PostgresRepository) GetByRecruiterID(ctx context.Context, recruiterID string) ([]models.Interview, error) {
	query := `
		SELECT id, recruiter_id, title, role, description, difficulty, duration, voice_enabled, status, share_token, created_at
		FROM interviews
		WHERE recruiter_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, recruiterID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recruiter interviews: %w", err)
	}
	defer rows.Close()

	var interviews []models.Interview
	for rows.Next() {
		var i models.Interview
		if err := rows.Scan(
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
		); err != nil {
			return nil, fmt.Errorf("failed to scan interview row: %w", err)
		}
		interviews = append(interviews, i)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating interview rows: %w", err)
	}

	return interviews, nil
}

func (r *PostgresRepository) Update(ctx context.Context, interview *models.Interview) error {
	query := `
		UPDATE interviews
		SET title = $1, role = $2, description = $3, difficulty = $4, duration = $5, voice_enabled = $6, status = $7, updated_at = NOW()
		WHERE id = $8 AND recruiter_id = $9
	`

	res, err := r.db.ExecContext(
		ctx,
		query,
		interview.Title,
		interview.Role,
		interview.Description,
		interview.Difficulty,
		interview.Duration,
		interview.VoiceEnabled,
		interview.Status,
		interview.ID,
		interview.RecruiterID,
	)
	if err != nil {
		return fmt.Errorf("failed to update interview: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("interview not found or unauthorized")
	}

	return nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM interviews WHERE id = $1`

	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete interview: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("interview not found")
	}

	return nil
}

func (r *PostgresRepository) AddCandidateEmail(
	ctx context.Context,
	interviewID string,
	email string,
	candidateName string,
) error {
	email = strings.ToLower(strings.TrimSpace(email))

	if email == "" {
		return errors.New("email is required")
	}

	name := strings.TrimSpace(candidateName)
	id := uuid.New().String()

	query := `
		INSERT INTO interview_candidates (
			id,
			interview_id,
			email,
			name
		)
		VALUES ($1, $2, $3, $4)
			ON CONFLICT (interview_id, email) DO NOTHING

	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		id,
		interviewID,
		email,
		name,
	)
	if err != nil {
		return fmt.Errorf("failed to add candidate email: %w", err)
	}

	return nil
}
