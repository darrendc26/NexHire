package ai

import (
	"context"
)

type AIProvider interface {
	GenerateInitialQuestion(
		ctx context.Context,
		input InterviewContext,
	) (*InterviewTurn, error)

	ProcessAnswer(
		ctx context.Context,
		input InterviewContext,
	) (*InterviewTurn, error)

	GenerateReport(
		ctx context.Context,
		input ReportContext,
	) (*InterviewReport, error)
}
