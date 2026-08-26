package speech

import (
	"context"
	"errors"
	"fmt"

	"nexhire/backend/candidate"
	"nexhire/backend/models"
)

// VoiceAgentConfig holds configuration for initializing a Deepgram Voice Agent session.
type VoiceAgentConfig struct {
	SessionToken string `json:"session_token"`
	Role         string `json:"role"`
	Difficulty   string `json:"difficulty"`
	Instructions string `json:"instructions"`
}

// ToolCallPayload represents a tool execution request coming from the Deepgram Voice Agent.
type ToolCallPayload struct {
	ToolName     string `json:"tool_name"`
	SessionToken string `json:"session_token"`
	AnswerText   string `json:"answer_text,omitempty"`
}

// ToolCallResult represents the output sent back to the Deepgram Voice Agent after executing a tool.
type ToolCallResult struct {
	Success        bool                   `json:"success"`
	ShouldContinue bool                   `json:"should_continue"`
	NextQuestion   string                 `json:"next_question,omitempty"`
	OutroMessage   string                 `json:"outro_message,omitempty"`
	Evaluation     *candidate.EvaluationDTO `json:"evaluation,omitempty"`
	Error          string                 `json:"error,omitempty"`
}

// VoiceAgentService manages integrations with the Deepgram Voice Agent API
// and bridges Voice Agent tool calls with NexHire candidate session management.
type VoiceAgentService struct {
	candidateService *candidate.Service
}

// NewVoiceAgentService creates a new VoiceAgentService.
func NewVoiceAgentService(candidateService *candidate.Service) *VoiceAgentService {
	return &VoiceAgentService{
		candidateService: candidateService,
	}
}

// GetVoiceAgentConfig retrieves session and interview parameters required to spin up
// the Deepgram Voice Agent for a given candidate session.
func (v *VoiceAgentService) GetVoiceAgentConfig(ctx context.Context, sessionToken string) (*VoiceAgentConfig, *models.CandidateSession, error) {
	if v.candidateService == nil {
		return nil, nil, errors.New("candidate service is not initialized")
	}

	session, interview, err := v.candidateService.ValidateActiveSession(ctx, sessionToken)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to validate active session for voice agent: %w", err)
	}

	instructions := fmt.Sprintf(
		"You are an AI interviewer evaluating %s for the role of %s (difficulty: %s). Ask one question at a time, listen carefully to answers, and submit answers via tool call.",
		session.Name, interview.Role, interview.Difficulty,
	)

	cfg := &VoiceAgentConfig{
		SessionToken: sessionToken,
		Role:         interview.Role,
		Difficulty:   string(interview.Difficulty),
		Instructions: instructions,
	}

	return cfg, session, nil
}

// ProcessVoiceAgentToolCall handles tool calls triggered by the Deepgram Voice Agent
// (e.g. submitting candidate answers directly into candidateService.SubmitAnswer).
func (v *VoiceAgentService) ProcessVoiceAgentToolCall(ctx context.Context, payload ToolCallPayload) (*ToolCallResult, error) {
	if v.candidateService == nil {
		return nil, errors.New("candidate service is not initialized")
	}

	switch payload.ToolName {
	case "submit_answer":
		if payload.SessionToken == "" {
			return &ToolCallResult{Success: false, Error: "session_token is required"}, nil
		}
		if payload.AnswerText == "" {
			return &ToolCallResult{Success: false, Error: "answer_text is required"}, nil
		}

		res, err := v.candidateService.SubmitAnswer(ctx, payload.SessionToken, payload.AnswerText)
		if err != nil {
			return &ToolCallResult{
				Success: false,
				Error:   fmt.Sprintf("failed to submit answer: %v", err),
			}, nil
		}

		nextQ := ""
		if res.NextQuestion != nil {
			nextQ = res.NextQuestion.Text
		}

		return &ToolCallResult{
			Success:        true,
			ShouldContinue: res.ShouldContinue,
			NextQuestion:   nextQ,
			OutroMessage:   res.OutroMessage,
			Evaluation:     res.Evaluation,
		}, nil

	default:
		return &ToolCallResult{
			Success: false,
			Error:   fmt.Sprintf("unknown tool name: %s", payload.ToolName),
		}, nil
	}
}
