package models

import "time"

type Interview struct {
	ID           string     `json:"id"`
	RecruiterID  string     `json:"recruiter_id"`
	Title        string     `json:"title"`
	Role         string     `json:"role"`
	Description  string     `json:"description"`
	Difficulty   Difficulty `json:"difficulty"`
	Duration     int        `json:"duration"`
	VoiceEnabled bool       `json:"voice_enabled"`
	Status       Status     `json:"status"`
	ShareToken   string     `json:"share_token"`
	CreatedAt    time.Time  `json:"created_at"`
}

type Difficulty string

const (
	Easy   Difficulty = "easy"
	Medium Difficulty = "medium"
	Hard   Difficulty = "hard"
)

type Status string

const (
	Draft  Status = "draft"
	Active Status = "active"
	Closed Status = "closed"
)

type CandidateSession struct {
	ID          string        `json:"id"`
	InterviewID string        `json:"interview_id"`
	Name        string        `json:"name"`
	Email       string        `json:"email"`
	TokenHash   string        `json:"-"`
	RawToken    string        `json:"session_token,omitempty"`
	Status      SessionStatus `json:"status"`
	StartedAt   time.Time     `json:"started_at"`
	ExpiresAt   time.Time     `json:"expires_at"`
	FinishedAt  *time.Time    `json:"finished_at"`
}

type SessionStatus string

const (
	SessionNotStarted SessionStatus = "not_started"
	SessionActive     SessionStatus = "active"
	SessionCompleted  SessionStatus = "completed"
)

type SessionProgress struct {
	QuestionsAsked int   `json:"questions_asked"`
	TimeRemaining  int64 `json:"time_remaining_seconds"`
}

type CandidateResponse struct {
	ID            string    `json:"id"`
	SessionID     string    `json:"session_id"`
	Question      string    `json:"question"`
	Answer        string    `json:"answer,omitempty"`
	Score         *float64  `json:"score,omitempty"`
	Feedback      string    `json:"feedback,omitempty"`
	Strengths     []string  `json:"strengths,omitempty"`
	Weaknesses    []string  `json:"weaknesses,omitempty"`
	QuestionType  string    `json:"question_type,omitempty"`
	QuestionOrder int       `json:"question_order"`
	CreatedAt     time.Time `json:"created_at"`
}

type CandidateReport struct {
	ID             string       `json:"id"`
	SessionID      string       `json:"session_id"`
	OverallScore   float64      `json:"overall_score"`
	Recommendation string       `json:"recommendation"`
	Summary        string       `json:"summary"`
	Strengths      []string     `json:"strengths"`
	Weaknesses     []string     `json:"weaknesses"`
	Skills         []SkillScore `json:"skills"`
	CreatedAt      time.Time    `json:"created_at"`
}

type SkillScore struct {
	Skill    string  `json:"skill"`
	Score    float64 `json:"score"`
	Feedback string  `json:"feedback"`
}
