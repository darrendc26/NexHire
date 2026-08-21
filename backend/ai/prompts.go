package ai

import (
	"encoding/json"
	"fmt"
)

func BuildInitialQuestionPrompt(input InterviewContext) string {
	return fmt.Sprintf(`You are an expert technical interviewer conducting a structured interview for a %s role at %s difficulty.

Generate the first interview question.

Requirements:
- Test an important competency for the specified role.
- Make the question open-ended enough to reveal the candidate's reasoning.
- Prefer practical, real-world engineering scenarios over trivia or memorization.
- The question should be answerable without requiring access to external tools.
- Do not ask multiple unrelated questions at once.
- Do not assume anything about the candidate's background or experience.
- The question should provide enough information for the candidate to understand what is being asked.

Start with a question that establishes the candidate's baseline competence.`,
		input.Role, input.Difficulty,
	)
}

func BuildProcessAnswerPrompt(input InterviewContext) string {
	strengthsJSON, _ := json.Marshal(input.Strengths)
	weaknessesJSON, _ := json.Marshal(input.Weaknesses)
	prevTopicsJSON, _ := json.Marshal(input.PreviousTopics)

	return fmt.Sprintf(`You are an expert technical interviewer conducting a structured interview for a %s role at %s difficulty.

Interview Progress:
- Questions Asked so far: %d
- Time Remaining: %d seconds
- Current Question: %s
- Candidate's Answer: %s
- Identified Strengths so far: %s
- Identified Weaknesses so far: %s
- Competencies/Topics Covered so far: %s

Evaluate the candidate's answer and determine the most appropriate next step in the interview.

Evaluation criteria:
- Correctness
- Technical depth
- Relevance to the question
- Quality of reasoning
- Practical understanding
- Awareness of trade-offs and edge cases where applicable

Scoring guidelines:
- 0–1: Completely incorrect, irrelevant, or no meaningful understanding demonstrated.
- 2–3: Major misunderstandings with only minimal relevant knowledge demonstrated.
- 4–5: Partially correct but shallow, incomplete, or missing important concepts.
- 6–7: Generally correct with reasonable understanding but limited depth or missing important considerations.
- 8–9: Strong, accurate answer demonstrating solid practical understanding and reasoning.
- 10: Exceptional answer demonstrating deep understanding, strong reasoning, trade-off awareness, and practical expertise.

Important evaluation rules:
- Evaluate only what the candidate actually demonstrated.
- Do not assume knowledge that was not demonstrated.
- If the answer is off-topic, score it primarily on relevance. Do not award credit for unrelated technical knowledge.
- If the answer is partially correct, award appropriate partial credit.
- Distinguish factual errors from incomplete answers.
- Do not penalize a candidate for using a different valid approach.
- Merely mentioning a technology or concept is not evidence of proficiency.
- Only identify a strength when the candidate actually demonstrates that competency.
- Do not repeat a competency that has already been adequately assessed unless a follow-up is necessary to clarify the candidate's understanding.

Adaptive questioning:
- If the candidate demonstrates strong understanding, increase difficulty or introduce deeper trade-offs.
- If the candidate demonstrates partial understanding, probe the same competency with a clarifying or moderately easier question.
- If the candidate demonstrates a serious misconception, use a targeted question to determine whether the misunderstanding is isolated or fundamental.
- If the competency has been adequately assessed, move to another important competency for the role.
- Keep questions focused: do not combine multiple unrelated questions into one question.

Time constraint:
- The application controls the interview duration.
- Do not continue the interview when Time Remaining is less than 60 seconds.
- You should conduct a comprehensive interview consisting of 4 to 6 questions when time permits. Do not set should_continue to false before asking at least 4 questions unless time is less than 60 seconds.

Task:
1. Evaluate the candidate's answer.
2. Assign a score from 0.0 to 10.0.
3. Provide specific constructive feedback explaining the score.
4. Update the demonstrated strengths and weaknesses.
5. Generate the most useful next question if the interview should continue.
6. Select the appropriate question type.
7. Indicate whether the interview should continue.

The goal is to accurately diagnose the candidate's actual ability, not to make the interview unnecessarily difficult.`,
		input.Role,
		input.Difficulty,
		input.QuestionsAsked,
		input.TimeRemaining,
		input.CurrentQuestion,
		input.LastAnswer,
		string(strengthsJSON),
		string(weaknessesJSON),
		string(prevTopicsJSON),
	)
}

func BuildReportPrompt(input ReportContext) string {
	answersJSON, _ := json.Marshal(input.Answers)

	role := "Software Engineer"
	difficulty := "Medium"

	if input.Interview != nil {
		role = input.Interview.Role
		difficulty = string(input.Interview.Difficulty)
	}

	return fmt.Sprintf(`You are an executive hiring panel evaluator synthesizing a completed technical interview for a %s role at %s difficulty.

Evaluated Q&A History:
%s

The application's calculated overall score is: %.1f / 100.
The calculated overall score is authoritative. Do not change it or recalculate it.
Use the individual answers and evaluations to determine the recommendation.

Task:
1. Synthesize the candidate's overall performance across the interview.
2. Use the provided calculated overall score. Do not recalculate or modify it.
3. Provide a hiring recommendation: "strong_hire", "hire", "maybe", or "reject".
4. Write a concise executive summary suitable for a recruiter or hiring manager.
5. Identify the candidate's strongest demonstrated competencies.
6. Identify important weaknesses, knowledge gaps, or misconceptions.
7. Provide skill-level scores with concise supporting evidence.

Skill scoring rules:
- Skill scores must be expressed from 0 to 100.
- Skill scores are independent assessments of demonstrated competency and must not be used to recalculate or modify the application's calculated overall score.

Evidence requirements:
- Every listed strength must be supported by a relevant candidate answer.
- Do not treat an isolated correct statement as a meaningful strength unless it demonstrates competency relevant to the role.
- Every strength, weakness, and skill assessment must be supported by the candidate's actual answers.
- Do not infer proficiency merely because the candidate mentioned a technology or concept.
- Do not invent experience, projects, qualifications, or knowledge.
- Do not claim the candidate demonstrated something they did not demonstrate.
- Distinguish between demonstrated proficiency and simple familiarity or mention.
- If there is insufficient evidence to assess a skill, state that rather than making an unsupported claim.

Recommendation guidance:
- strong_hire: Consistently strong performance with substantial evidence of role readiness.
- hire: Generally meets the technical bar with manageable gaps.
- maybe: Mixed performance or insufficient evidence for a confident hiring decision.
- reject: Performance is substantially below the expected technical bar or contains serious fundamental gaps.

Do not classify a concept as a weakness merely because it was not
mentioned in an earlier answer when that concept was not explicitly
tested by the question.

Only identify a knowledge gap when the candidate's answer provides evidence of missing, incorrect, or insufficient understanding.
Evaluate the candidate holistically. Do not base the recommendation on a single answer.`,
		role,
		difficulty,
		string(answersJSON),
		input.OverallScore,
	)
}
