package candidate

import (
	"context"
	"database/sql"
	"encoding/json"
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
	UpdateSessionStatus(ctx context.Context, sessionID string, status models.SessionStatus, finishedAt *time.Time) error
	CreateResponse(ctx context.Context, resp *models.CandidateResponse) error
	UpdateResponseEvaluation(ctx context.Context, id string, answer string, score float64, feedback string, strengths []string, weaknesses []string) error
	GetResponsesBySessionID(ctx context.Context, sessionID string) ([]models.CandidateResponse, error)
	GetPendingResponse(ctx context.Context, sessionID string) (*models.CandidateResponse, error)
	CreateReport(ctx context.Context, report *models.CandidateReport) error
	GetReportBySessionID(ctx context.Context, sessionID string) (*models.CandidateReport, error)
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
		INSERT INTO candidate_sessions (id, interview_id, name, email, token_hash, status, started_at, expires_at, finished_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
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
		session.ExpiresAt,
		session.FinishedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to insert candidate session: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetSessionByID(ctx context.Context, id string) (*models.CandidateSession, error) {
	query := `
		SELECT id, interview_id, name, email, COALESCE(token_hash, ''), status, started_at, expires_at, finished_at
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
		&s.ExpiresAt,
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
		SELECT id, interview_id, name, email, COALESCE(token_hash, ''), status, started_at, expires_at, finished_at
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
		&s.ExpiresAt,
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
		SELECT id, interview_id, name, email, COALESCE(token_hash, ''), status, started_at, expires_at, finished_at
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
			&s.ExpiresAt,
			&s.FinishedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan session row: %w", err)
		}
		sessions = append(sessions, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during sessions row iteration: %w", err)
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

func (r *PostgresRepository) UpdateSessionStatus(ctx context.Context, sessionID string, status models.SessionStatus, finishedAt *time.Time) error {
	query := `
		UPDATE candidate_sessions
		SET status = $1, finished_at = $2
		WHERE id = $3
	`
	_, err := r.db.ExecContext(ctx, query, status, finishedAt, sessionID)
	if err != nil {
		return fmt.Errorf("failed to update candidate session status: %w", err)
	}
	return nil
}

func (r *PostgresRepository) CreateResponse(ctx context.Context, resp *models.CandidateResponse) error {
	if resp.CreatedAt.IsZero() {
		resp.CreatedAt = time.Now()
	}

	strengthsJSON, _ := json.Marshal(resp.Strengths)
	weaknessesJSON, _ := json.Marshal(resp.Weaknesses)

	query := `
		INSERT INTO candidate_responses (id, session_id, question, answer, score, feedback, strengths, weaknesses, question_type, question_order, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.db.ExecContext(
		ctx,
		query,
		resp.ID,
		resp.SessionID,
		resp.Question,
		resp.Answer,
		resp.Score,
		resp.Feedback,
		strengthsJSON,
		weaknessesJSON,
		resp.QuestionType,
		resp.QuestionOrder,
		resp.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create candidate response: %w", err)
	}
	return nil
}

func (r *PostgresRepository) UpdateResponseEvaluation(
	ctx context.Context,
	id string,
	answer string,
	score float64,
	feedback string,
	strengths []string,
	weaknesses []string,
) error {
	strengthsJSON, _ := json.Marshal(strengths)
	weaknessesJSON, _ := json.Marshal(weaknesses)

	query := `
		UPDATE candidate_responses
		SET answer = $1, score = $2, feedback = $3, strengths = $4, weaknesses = $5
		WHERE id = $6
	`
	_, err := r.db.ExecContext(ctx, query, answer, score, feedback, strengthsJSON, weaknessesJSON, id)
	if err != nil {
		return fmt.Errorf("failed to update candidate response evaluation: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetResponsesBySessionID(ctx context.Context, sessionID string) ([]models.CandidateResponse, error) {
	query := `
		SELECT id, session_id, question, COALESCE(answer, ''), score, COALESCE(feedback, ''), strengths, weaknesses, COALESCE(question_type, 'technical'), question_order, created_at
		FROM candidate_responses
		WHERE session_id = $1
		ORDER BY question_order ASC
	`
	rows, err := r.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query responses: %w", err)
	}
	defer rows.Close()

	var responses []models.CandidateResponse
	for rows.Next() {
		var resp models.CandidateResponse
		var strengthsRaw, weaknessesRaw []byte
		var scoreVal sql.NullFloat64

		if err := rows.Scan(
			&resp.ID,
			&resp.SessionID,
			&resp.Question,
			&resp.Answer,
			&scoreVal,
			&resp.Feedback,
			&strengthsRaw,
			&weaknessesRaw,
			&resp.QuestionType,
			&resp.QuestionOrder,
			&resp.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan response row: %w", err)
		}

		if scoreVal.Valid {
			s := scoreVal.Float64
			resp.Score = &s
		}

		if len(strengthsRaw) > 0 {
			json.Unmarshal(strengthsRaw, &resp.Strengths)
		}
		if len(weaknessesRaw) > 0 {
			json.Unmarshal(weaknessesRaw, &resp.Weaknesses)
		}

		responses = append(responses, resp)
	}

	return responses, nil
}

func (r *PostgresRepository) GetPendingResponse(ctx context.Context, sessionID string) (*models.CandidateResponse, error) {
	query := `
		SELECT id, session_id, question, COALESCE(answer, ''), score, COALESCE(feedback, ''), strengths, weaknesses, COALESCE(question_type, 'technical'), question_order, created_at
		FROM candidate_responses
		WHERE session_id = $1 AND (answer IS NULL OR answer = '')
		ORDER BY question_order ASC
		LIMIT 1
	`
	var resp models.CandidateResponse
	var strengthsRaw, weaknessesRaw []byte
	var scoreVal sql.NullFloat64

	err := r.db.QueryRowContext(ctx, query, sessionID).Scan(
		&resp.ID,
		&resp.SessionID,
		&resp.Question,
		&resp.Answer,
		&scoreVal,
		&resp.Feedback,
		&strengthsRaw,
		&weaknessesRaw,
		&resp.QuestionType,
		&resp.QuestionOrder,
		&resp.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("no pending question found")
		}
		return nil, fmt.Errorf("failed to fetch pending response: %w", err)
	}

	if scoreVal.Valid {
		s := scoreVal.Float64
		resp.Score = &s
	}

	if len(strengthsRaw) > 0 {
		json.Unmarshal(strengthsRaw, &resp.Strengths)
	}
	if len(weaknessesRaw) > 0 {
		json.Unmarshal(weaknessesRaw, &resp.Weaknesses)
	}

	return &resp, nil
}

func (r *PostgresRepository) CreateReport(ctx context.Context, report *models.CandidateReport) error {
	if report.CreatedAt.IsZero() {
		report.CreatedAt = time.Now()
	}

	strengthsJSON, _ := json.Marshal(report.Strengths)
	weaknessesJSON, _ := json.Marshal(report.Weaknesses)
	skillsJSON, _ := json.Marshal(report.Skills)

	query := `
		INSERT INTO candidate_reports (id, session_id, overall_score, recommendation, summary, strengths, weaknesses, skills, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (session_id) DO UPDATE SET
			overall_score = EXCLUDED.overall_score,
			recommendation = EXCLUDED.recommendation,
			summary = EXCLUDED.summary,
			strengths = EXCLUDED.strengths,
			weaknesses = EXCLUDED.weaknesses,
			skills = EXCLUDED.skills
	`
	_, err := r.db.ExecContext(
		ctx,
		query,
		report.ID,
		report.SessionID,
		report.OverallScore,
		report.Recommendation,
		report.Summary,
		strengthsJSON,
		weaknessesJSON,
		skillsJSON,
		report.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create candidate report: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetReportBySessionID(ctx context.Context, sessionID string) (*models.CandidateReport, error) {
	query := `
		SELECT id, session_id, overall_score, recommendation, summary, strengths, weaknesses, skills, created_at
		FROM candidate_reports
		WHERE session_id = $1
	`
	var rep models.CandidateReport
	var strengthsRaw, weaknessesRaw, skillsRaw []byte

	err := r.db.QueryRowContext(ctx, query, sessionID).Scan(
		&rep.ID,
		&rep.SessionID,
		&rep.OverallScore,
		&rep.Recommendation,
		&rep.Summary,
		&strengthsRaw,
		&weaknessesRaw,
		&skillsRaw,
		&rep.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("report not found")
		}
		return nil, fmt.Errorf("failed to fetch candidate report: %w", err)
	}

	if len(strengthsRaw) > 0 {
		json.Unmarshal(strengthsRaw, &rep.Strengths)
	}
	if len(weaknessesRaw) > 0 {
		json.Unmarshal(weaknessesRaw, &rep.Weaknesses)
	}
	if len(skillsRaw) > 0 {
		json.Unmarshal(skillsRaw, &rep.Skills)
	}

	return &rep, nil
}

