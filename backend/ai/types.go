package ai

import (
	"nexhire/backend/models"
)

type InterviewContext struct {
	Role            string   `json:"role"`
	Difficulty      string   `json:"difficulty"`
	CurrentQuestion string   `json:"current_question"`
	LastAnswer      string   `json:"last_answer"`
	PreviousTopics  []string `json:"previous_topics"`
	Strengths       []string `json:"strengths"`
	Weaknesses      []string `json:"weaknesses"`
	QuestionsAsked  int      `json:"questions_asked"`
	TimeRemaining   int64    `json:"time_remaining_seconds"`
}

type InterviewTurn struct {
	Score          float64  `json:"score"`
	Feedback       string   `json:"feedback"`
	Strengths      []string `json:"strengths"`
	Weaknesses     []string `json:"weaknesses"`
	NextQuestion   string   `json:"next_question"`
	QuestionType   string   `json:"question_type"`
	ShouldContinue bool     `json:"should_continue"`
}

type ReportContext struct {
	Interview    *models.Interview
	Candidate    *models.CandidateSession
	Answers      []QuestionAnswer
	OverallScore float64
}

type QuestionAnswer struct {
	Question   string   `json:"question"`
	Answer     string   `json:"answer"`
	Score      float64  `json:"score"`
	Feedback   string   `json:"feedback"`
	Strengths  []string `json:"strengths"`
	Weaknesses []string `json:"weaknesses"`
}

type InterviewReport struct {
	OverallScore   float64        `json:"overall_score"`
	Recommendation Recommendation `json:"recommendation"`
	Summary        string         `json:"summary"`
	Strengths      []string       `json:"strengths"`
	Weaknesses     []string       `json:"weaknesses"`
	Skills         []SkillScore   `json:"skills"`
}

type Recommendation string

const (
	StrongHire Recommendation = "strong_hire"
	Hire       Recommendation = "hire"
	Maybe      Recommendation = "maybe"
	Reject     Recommendation = "reject"
)

type SkillScore struct {
	Skill    string  `json:"skill"`
	Score    float64 `json:"score"`
	Feedback string  `json:"feedback"`
}
