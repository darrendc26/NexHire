package ai

import (
	"google.golang.org/genai"
)

var interviewTurnSchema = &genai.Schema{
	Type: genai.TypeObject,
	Properties: map[string]*genai.Schema{
		"score": {
			Type:        genai.TypeNumber,
			Description: "Score for the candidate's last answer from 0.0 to 10.0",
		},
		"feedback": {
			Type:        genai.TypeString,
			Description: "Detailed feedback on the candidate's last answer",
		},
		"strengths": {
			Type: genai.TypeArray,
			Items: &genai.Schema{
				Type: genai.TypeString,
			},
			Description: "Candidate strengths observed",
		},
		"weaknesses": {
			Type: genai.TypeArray,
			Items: &genai.Schema{
				Type: genai.TypeString,
			},
			Description: "Candidate weaknesses or areas for improvement observed",
		},
		"next_question": {
			Type:        genai.TypeString,
			Description: "The next interview question to ask the candidate",
		},
		"question_type": {
			Type: genai.TypeString,
			Enum: []string{
				"technical",
				"behavioral",
				"situational",
			},
			Description: "Type of the next question",
		},
		"should_continue": {
			Type:        genai.TypeBoolean,
			Description: "Whether the interview session should continue",
		},
	},
	Required: []string{
		"score",
		"feedback",
		"strengths",
		"weaknesses",
		"next_question",
		"question_type",
		"should_continue",
	},
}

var interviewReportSchema = &genai.Schema{
	Type: genai.TypeObject,
	Properties: map[string]*genai.Schema{
		"overall_score": {
			Type:        genai.TypeNumber,
			Description: "Overall score from 0.0 to 100.0",
		},
		"recommendation": {
			Type: genai.TypeString,
			Enum: []string{
				"strong_hire",
				"hire",
				"maybe",
				"reject",
			},
			Description: "Hiring recommendation",
		},
		"summary": {
			Type:        genai.TypeString,
			Description: "Executive summary of candidate performance",
		},
		"strengths": {
			Type: genai.TypeArray,
			Items: &genai.Schema{
				Type: genai.TypeString,
			},
			Description: "Key overall strengths",
		},
		"weaknesses": {
			Type: genai.TypeArray,
			Items: &genai.Schema{
				Type: genai.TypeString,
			},
			Description: "Key overall weaknesses",
		},
		"skills": {
			Type: genai.TypeArray,
			Items: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"skill": {
						Type: genai.TypeString,
					},
					"score": {
						Type: genai.TypeNumber,
					},
					"feedback": {
						Type: genai.TypeString,
					},
				},
				Required: []string{"skill", "score", "feedback"},
			},
			Description: "Evaluated skill breakdown",
		},
	},
	Required: []string{
		"overall_score",
		"recommendation",
		"summary",
		"strengths",
		"weaknesses",
		"skills",
	},
}
