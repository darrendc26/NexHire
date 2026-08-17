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
	Status      SessionStatus `json:"status"`
	StartedAt   time.Time     `json:"started_at"`
	FinishedAt  *time.Time    `json:"finished_at"`
}

type SessionStatus string

const (
	SessionNotStarted SessionStatus = "not_started"
	SessionActive     SessionStatus = "active"
	SessionCompleted  SessionStatus = "completed"
)
