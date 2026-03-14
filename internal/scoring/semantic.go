package scoring

import (
	"context"
	"fmt"
	"internal/llm"
)

// SemanticResult holds the result of LLM-based semantic scoring.
type SemanticResult struct {
	Score           float64
	Feedback        string
	KeyPointsCovered []string
	KeyPointsMissed  []string
	FactualErrors    []string
}

// SemanticScorer uses an LLM to compare student answers against model answers.
type SemanticScorer struct {
	Client llm.LLMClient
}

type semanticResponse struct {
	SemanticScore    float64  `json:"semantic_score"`
	Feedback         string   `json:"feedback"`
	KeyPointsCovered []string `json:"key_points_covered"`
	KeyPointsMissed  []string `json:"key_points_missed"`
	FactualErrors    []string `json:"factual_errors"`
}

const semanticSystemPrompt = `You are a precise academic answer evaluator. You compare a student's answer against a model answer and score how well the student's response captures the intended meaning, regardless of exact wording.

Rules:
- Score from 0 to 100 based on semantic similarity to the model answer.
- 90-100: Excellent -- captures all key ideas, may use different words but meaning is equivalent.
- 70-89: Good -- captures most key ideas with minor omissions or imprecisions.
- 50-69: Partial -- captures some key ideas but misses important aspects.
- 30-49: Weak -- shows some understanding but has significant gaps.
- 0-29: Poor -- does not demonstrate understanding of the concept.
- Be lenient about wording differences. "force equals mass times acceleration" and "F=ma" should score equally.
- Be strict about factual accuracy. Incorrect facts must reduce the score significantly.
- Provide specific, actionable feedback telling the student exactly what they got right and what they missed.
- Respond with valid JSON only. No markdown fences, no explanation, no text outside the JSON object.`

// Score evaluates a student's answer against the model answer using the LLM.
func (s *SemanticScorer) Score(ctx context.Context, questionText, modelAnswer, studentAnswer, bloomLevel, topic string, keyPoints []string) (SemanticResult, error) {
	if studentAnswer == "" {
		return SemanticResult{Score: 0, Feedback: "No answer provided."}, nil
	}

	keyPointsList := ""
	for _, kp := range keyPoints {
		keyPointsList += "- " + kp + "\n"
	}

	userPrompt := fmt.Sprintf(`Evaluate the student's answer against the model answer for this question.

QUESTION: "%s"
BLOOM'S LEVEL: %s
TOPIC: %s

MODEL ANSWER:
"%s"

KEY POINTS EXPECTED:
%s
STUDENT'S ANSWER:
"%s"

Respond with this exact JSON structure:

{
  "semantic_score": <integer 0-100>,
  "feedback": "Specific feedback explaining what the student got right, what they missed, and how to improve. Reference specific key points by name.",
  "key_points_covered": ["list of key points the student addressed"],
  "key_points_missed": ["list of key points the student did not address"],
  "factual_errors": ["list of any factually incorrect statements in the student's answer, empty array if none"]
}

Return ONLY the JSON object. No other text.`, questionText, bloomLevel, topic, modelAnswer, keyPointsList, studentAnswer)

	var resp semanticResponse
	err := s.Client.CompleteJSON(ctx, llm.CompletionRequest{
		SystemPrompt: semanticSystemPrompt,
		UserPrompt:   userPrompt,
		Temperature:  0.2,
		MaxTokens:    1024,
	}, &resp)
	if err != nil {
		return SemanticResult{}, fmt.Errorf("semantic scoring LLM call failed: %w", err)
	}

	// Clamp score to 0-100
	if resp.SemanticScore < 0 {
		resp.SemanticScore = 0
	}
	if resp.SemanticScore > 100 {
		resp.SemanticScore = 100
	}

	return SemanticResult{
		Score:           resp.SemanticScore,
		Feedback:        resp.Feedback,
		KeyPointsCovered: resp.KeyPointsCovered,
		KeyPointsMissed:  resp.KeyPointsMissed,
		FactualErrors:    resp.FactualErrors,
	}, nil
}
