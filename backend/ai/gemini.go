package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"google.golang.org/genai"
)

type GeminiProvider struct {
	Client *genai.Client
	Model  string
}

func NewGeminiProvider(client *genai.Client, model string) *GeminiProvider {
	return &GeminiProvider{
		Client: client,
		Model:  model,
	}
}

func generateWithRetry(
	ctx context.Context,
	client *genai.Client,
	model string,
	prompt string,
	schema *genai.Schema,
) (*genai.GenerateContentResponse, error) {
	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		result, err := client.Models.GenerateContent(
			ctx,
			model,
			genai.Text(prompt),
			&genai.GenerateContentConfig{
				ResponseMIMEType: "application/json",
				ResponseSchema:   schema,
			},
		)
		if err == nil {
			return result, nil
		}
		lastErr = err
		errStr := err.Error()
		if strings.Contains(errStr, "429") || strings.Contains(errStr, "RESOURCE_EXHAUSTED") {
			time.Sleep(52 * time.Second)
		} else if strings.Contains(errStr, "503") || strings.Contains(errStr, "UNAVAILABLE") {
			time.Sleep(time.Duration(1<<attempt) * 2 * time.Second)
		} else {
			time.Sleep(2 * time.Second)
		}
	}
	return nil, lastErr
}

func extractResponseText(result *genai.GenerateContentResponse) (string, error) {
	if result == nil || len(result.Candidates) == 0 || result.Candidates[0].Content == nil || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from Gemini provider")
	}
	part := result.Candidates[0].Content.Parts[0]
	return part.Text, nil
}

func cleanJSONString(raw string) string {
	cleaned := strings.TrimSpace(raw)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```JSON")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	return strings.TrimSpace(cleaned)
}

func (g *GeminiProvider) GenerateInitialQuestion(
	ctx context.Context,
	input InterviewContext,
) (*InterviewTurn, error) {
	prompt := BuildInitialQuestionPrompt(input)

	result, err := generateWithRetry(ctx, g.Client, g.Model, prompt, interviewTurnSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to generate initial question: %w", err)
	}

	rawText, err := extractResponseText(result)
	if err != nil {
		return nil, err
	}

	var turn InterviewTurn
	if err := json.Unmarshal([]byte(cleanJSONString(rawText)), &turn); err != nil {
		return nil, fmt.Errorf("failed to unmarshal interview turn: %w (raw response: %s)", err, rawText)
	}

	return &turn, nil
}

func (g *GeminiProvider) ProcessAnswer(
	ctx context.Context,
	input InterviewContext,
) (*InterviewTurn, error) {
	prompt := BuildProcessAnswerPrompt(input)

	result, err := generateWithRetry(ctx, g.Client, g.Model, prompt, interviewTurnSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to process answer: %w", err)
	}

	rawText, err := extractResponseText(result)
	if err != nil {
		return nil, err
	}

	var turn InterviewTurn
	if err := json.Unmarshal([]byte(cleanJSONString(rawText)), &turn); err != nil {
		return nil, fmt.Errorf("failed to unmarshal interview turn: %w (raw response: %s)", err, rawText)
	}

	return &turn, nil
}

func (g *GeminiProvider) GenerateReport(
	ctx context.Context,
	input ReportContext,
) (*InterviewReport, error) {
	prompt := BuildReportPrompt(input)

	result, err := generateWithRetry(ctx, g.Client, g.Model, prompt, interviewReportSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to generate report: %w", err)
	}

	rawText, err := extractResponseText(result)
	if err != nil {
		return nil, err
	}

	var report InterviewReport
	if err := json.Unmarshal([]byte(cleanJSONString(rawText)), &report); err != nil {
		return nil, fmt.Errorf("failed to unmarshal interview report: %w (raw response: %s)", err, rawText)
	}

	return &report, nil
}
