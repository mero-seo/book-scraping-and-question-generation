package scoring

import (
	"strings"
)

// CompletenessResult holds the result of completeness scoring.
type CompletenessResult struct {
	Score         float64
	CoveredPoints []string
	MissingPoints []string
}

// ScoreCompleteness measures what fraction of key points the student's answer addresses.
// Uses substring matching and token overlap (threshold 0.6) for flexible matching.
func ScoreCompleteness(answer string, keyPoints []string) CompletenessResult {
	if len(keyPoints) == 0 {
		return CompletenessResult{Score: 100}
	}

	normalizedAnswer := strings.ToLower(strings.TrimSpace(answer))
	answerTokens := tokenize(normalizedAnswer)
	answerTokenSet := make(map[string]bool, len(answerTokens))
	for _, t := range answerTokens {
		answerTokenSet[t] = true
	}

	var covered, missing []string

	for _, kp := range keyPoints {
		normalizedKP := strings.ToLower(strings.TrimSpace(kp))

		// Check 1: Direct substring match
		if strings.Contains(normalizedAnswer, normalizedKP) {
			covered = append(covered, kp)
			continue
		}

		// Check 2: Token overlap ratio
		kpTokens := tokenize(normalizedKP)
		if len(kpTokens) == 0 {
			continue
		}

		matchCount := 0
		for _, t := range kpTokens {
			if answerTokenSet[t] {
				matchCount++
			}
		}

		overlapRatio := float64(matchCount) / float64(len(kpTokens))
		if overlapRatio >= 0.6 {
			covered = append(covered, kp)
		} else {
			missing = append(missing, kp)
		}
	}

	total := len(covered) + len(missing)
	if total == 0 {
		return CompletenessResult{Score: 100}
	}

	return CompletenessResult{
		Score:         float64(len(covered)) / float64(total) * 100,
		CoveredPoints: covered,
		MissingPoints: missing,
	}
}

// tokenize splits text into lowercase words.
func tokenize(text string) []string {
	return strings.Fields(strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == ' ' {
			return r
		}
		if r >= 'A' && r <= 'Z' {
			return r + 32 // toLower
		}
		return ' '
	}, text))
}
