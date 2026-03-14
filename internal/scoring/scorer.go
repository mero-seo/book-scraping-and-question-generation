package scoring

import (
	"context"
	"fmt"
	"internal/llm"
	"internal/models"
	"strings"
)

// Weights defines the scoring weight distribution for a question type.
type Weights struct {
	Semantic     float64
	Completeness float64
	Keyword      float64
}

// ScoringWeights maps question types to their scoring weight distribution.
var ScoringWeights = map[string]Weights{
	"essay":        {Semantic: 0.50, Completeness: 0.30, Keyword: 0.20},
	"short_answer": {Semantic: 0.40, Completeness: 0.30, Keyword: 0.30},
}

// DefaultWeights are used when no specific weights are defined for a question type.
var DefaultWeights = Weights{Semantic: 0.50, Completeness: 0.30, Keyword: 0.20}

// ScoringResult holds the complete scoring output for an answer.
type ScoringResult struct {
	IsCorrect         *bool
	SemanticScore     *float64
	KeywordScore      *float64
	CompletenessScore *float64
	OverallScore      float64
	Feedback          string
}

// Scorer evaluates student answers against questions.
type Scorer struct {
	LLM llm.LLMClient
}

// NewScorer creates a new scorer with the given LLM client.
func NewScorer(client llm.LLMClient) *Scorer {
	return &Scorer{LLM: client}
}

// ScoreAnswer scores a student's answer based on the question type.
func (s *Scorer) ScoreAnswer(ctx context.Context, question *models.Question, answerText string) (ScoringResult, error) {
	switch question.QuestionType {
	case models.QuestionTypeMCQ, models.QuestionTypeTrueFalse, models.QuestionTypeFillBlank:
		return s.ScoreObjective(question, answerText), nil
	case models.QuestionTypeEssay, models.QuestionTypeShortAnswer:
		return s.ScoreSubjective(ctx, question, answerText)
	case models.QuestionTypeMatch:
		return s.ScoreObjective(question, answerText), nil
	case models.QuestionTypeAssertionReasoning:
		return s.ScoreObjective(question, answerText), nil
	default:
		return s.ScoreObjective(question, answerText), nil
	}
}

// ScoreObjective evaluates MCQ, true/false, fill-in-the-blank, and similar objective questions.
func (s *Scorer) ScoreObjective(question *models.Question, answerText string) ScoringResult {
	normalizedAnswer := strings.TrimSpace(strings.ToLower(answerText))
	normalizedCorrect := strings.TrimSpace(strings.ToLower(question.CorrectAnswer))

	isCorrect := normalizedAnswer == normalizedCorrect
	var score float64
	var feedback string

	if isCorrect {
		score = 100
		feedback = fmt.Sprintf("Correct! %s", question.Explanation)
	} else {
		score = 0
		feedback = fmt.Sprintf("The correct answer is %s. %s", question.CorrectAnswer, question.Explanation)
	}

	return ScoringResult{
		IsCorrect:    &isCorrect,
		OverallScore: score,
		Feedback:     feedback,
	}
}

// ScoreSubjective evaluates essay and short answer questions using three-dimension scoring.
func (s *Scorer) ScoreSubjective(ctx context.Context, question *models.Question, answerText string) (ScoringResult, error) {
	weights := ScoringWeights[question.QuestionType]
	if weights == (Weights{}) {
		weights = DefaultWeights
	}

	// Semantic scoring via LLM
	semantic := &SemanticScorer{Client: s.LLM}
	semResult, err := semantic.Score(ctx, question.QuestionText, question.ModelAnswer, answerText, question.BloomLevel, question.Topic, question.KeyPoints)
	if err != nil {
		return ScoringResult{}, fmt.Errorf("semantic scoring failed: %w", err)
	}

	// Keyword scoring
	kwResult := ScoreKeywords(answerText, question.KeyPoints)

	// Completeness scoring
	compResult := ScoreCompleteness(answerText, question.KeyPoints)

	// Calculate overall score
	overall := weights.Semantic*semResult.Score +
		weights.Completeness*compResult.Score +
		weights.Keyword*kwResult.Score

	// Build feedback
	feedback := buildSubjectiveFeedback(semResult, kwResult, compResult, overall)

	semScore := semResult.Score
	kwScore := kwResult.Score
	compScore := compResult.Score

	return ScoringResult{
		SemanticScore:     &semScore,
		KeywordScore:      &kwScore,
		CompletenessScore: &compScore,
		OverallScore:      overall,
		Feedback:          feedback,
	}, nil
}

func buildSubjectiveFeedback(sem SemanticResult, kw KeywordResult, comp CompletenessResult, overall float64) string {
	var b strings.Builder

	// Overall assessment
	label := scoreLabel(overall)
	fmt.Fprintf(&b, "Overall: %s (%.0f/100)\n\n", label, overall)

	// Semantic feedback from LLM
	if sem.Feedback != "" {
		fmt.Fprintf(&b, "%s\n\n", sem.Feedback)
	}

	// Completeness feedback
	totalPoints := len(comp.CoveredPoints) + len(comp.MissingPoints)
	if totalPoints > 0 {
		fmt.Fprintf(&b, "Key points covered: %d of %d.", len(comp.CoveredPoints), totalPoints)
		if len(comp.MissingPoints) > 0 {
			fmt.Fprintf(&b, " Missing: %s", strings.Join(comp.MissingPoints, ", "))
		}
		b.WriteString("\n")
	}

	// Keyword feedback
	if len(kw.MatchedKeywords) > 0 || len(kw.MissingKeywords) > 0 {
		if len(kw.MatchedKeywords) > 0 {
			fmt.Fprintf(&b, "Key terms used: %s.", strings.Join(kw.MatchedKeywords, ", "))
		}
		if len(kw.MissingKeywords) > 0 {
			fmt.Fprintf(&b, " Missing terms: %s.", strings.Join(kw.MissingKeywords, ", "))
		}
		b.WriteString("\n")
	}

	// Improvement suggestion for scores < 75
	if overall < 75 {
		b.WriteString("\n")
		minScore := sem.Score
		weakest := "semantic understanding"
		if kw.Score < minScore {
			minScore = kw.Score
			weakest = "use of technical terminology"
		}
		if comp.Score < minScore {
			weakest = "coverage of key points"
		}
		fmt.Fprintf(&b, "Suggestion: Focus on improving your %s.", weakest)
	}

	return b.String()
}

func scoreLabel(score float64) string {
	switch {
	case score >= 90:
		return "Excellent"
	case score >= 75:
		return "Good"
	case score >= 60:
		return "Satisfactory"
	case score >= 40:
		return "Needs Improvement"
	case score >= 20:
		return "Poor"
	default:
		return "Inadequate"
	}
}
