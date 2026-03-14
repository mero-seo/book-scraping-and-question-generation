package scoring

import (
	"strings"
	"unicode"
)

// KeywordResult holds the result of keyword scoring.
type KeywordResult struct {
	Score           float64
	MatchedKeywords []string
	MissingKeywords []string
}

// ScoreKeywords checks if key terms from keyPoints appear in the answer text.
// Uses basic Porter stemming for morphological matching.
func ScoreKeywords(answer string, keyPoints []string) KeywordResult {
	if len(keyPoints) == 0 {
		return KeywordResult{Score: 100}
	}

	answerTokens := tokenizeAndStem(answer)
	answerSet := make(map[string]bool, len(answerTokens))
	for _, t := range answerTokens {
		answerSet[t] = true
	}

	var matched, missing []string
	for _, kp := range keyPoints {
		kpTokens := tokenizeAndStem(kp)
		if len(kpTokens) == 0 {
			continue
		}
		allFound := true
		for _, t := range kpTokens {
			if !answerSet[t] {
				allFound = false
				break
			}
		}
		if allFound {
			matched = append(matched, kp)
		} else {
			missing = append(missing, kp)
		}
	}

	total := len(matched) + len(missing)
	if total == 0 {
		return KeywordResult{Score: 100}
	}

	return KeywordResult{
		Score:           float64(len(matched)) / float64(total) * 100,
		MatchedKeywords: matched,
		MissingKeywords: missing,
	}
}

// tokenizeAndStem splits text into lowercase tokens, strips punctuation, and applies basic stemming.
func tokenizeAndStem(text string) []string {
	words := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	stemmed := make([]string, 0, len(words))
	for _, w := range words {
		if isStopWord(w) {
			continue
		}
		stemmed = append(stemmed, basicStem(w))
	}
	return stemmed
}

// basicStem applies a simplified Porter stemmer for common English suffixes.
func basicStem(word string) string {
	if len(word) <= 3 {
		return word
	}

	suffixes := []string{
		"ational", "tional", "ization", "iveness", "fulness", "ouseness",
		"ement", "ation", "ating", "ating", "ness", "ment", "ence", "ance",
		"ible", "able", "tion", "sion", "ally", "ical", "ious", "eous",
		"ized", "ised", "ling", "ting", "ing", "ies", "ive", "ous", "ful",
		"ist", "ism", "ity", "ent", "ant", "ion", "ary",
		"ly", "ed", "er", "es", "al", "ic",
		"s",
	}

	for _, suffix := range suffixes {
		if strings.HasSuffix(word, suffix) && len(word)-len(suffix) >= 3 {
			return word[:len(word)-len(suffix)]
		}
	}
	return word
}

var stopWords = map[string]bool{
	"a": true, "an": true, "the": true, "is": true, "are": true, "was": true,
	"were": true, "be": true, "been": true, "being": true, "have": true,
	"has": true, "had": true, "do": true, "does": true, "did": true,
	"will": true, "would": true, "could": true, "should": true, "may": true,
	"might": true, "shall": true, "can": true, "to": true, "of": true,
	"in": true, "for": true, "on": true, "with": true, "at": true, "by": true,
	"from": true, "as": true, "into": true, "through": true, "during": true,
	"before": true, "after": true, "above": true, "below": true, "between": true,
	"and": true, "but": true, "or": true, "nor": true, "not": true, "so": true,
	"yet": true, "both": true, "either": true, "neither": true, "each": true,
	"every": true, "all": true, "any": true, "few": true, "more": true,
	"most": true, "other": true, "some": true, "such": true, "no": true,
	"only": true, "own": true, "same": true, "than": true, "too": true,
	"very": true, "just": true, "because": true, "if": true, "when": true,
	"where": true, "how": true, "what": true, "which": true, "who": true,
	"whom": true, "this": true, "that": true, "these": true, "those": true,
	"it": true, "its": true, "i": true, "me": true, "my": true, "we": true,
	"our": true, "you": true, "your": true, "he": true, "him": true,
	"his": true, "she": true, "her": true, "they": true, "them": true, "their": true,
}

func isStopWord(w string) bool {
	return stopWords[w]
}
