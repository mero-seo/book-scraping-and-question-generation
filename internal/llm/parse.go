package llm

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// ParseJSONResponse parses an LLM response that should contain JSON into the target type.
// It handles markdown fences, surrounding text, trailing commas, and other common LLM output issues.
func ParseJSONResponse[T any](raw string) (T, error) {
	var result T

	cleaned, err := ParseJSONResponseRaw(raw)
	if err != nil {
		return result, err
	}

	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return result, fmt.Errorf("JSON unmarshal failed: %w", err)
	}

	return result, nil
}

// ParseJSONResponseRaw extracts and cleans JSON from an LLM response string.
// Returns the cleaned JSON string ready for unmarshalling.
func ParseJSONResponseRaw(raw string) (string, error) {
	// Step 1: Strip markdown fences
	cleaned := strings.TrimSpace(raw)
	cleaned = stripMarkdownFences(cleaned)

	// Step 2: Find JSON boundaries
	cleaned = findJSONBoundaries(cleaned)
	if cleaned == "" {
		return "", fmt.Errorf("no JSON found in response")
	}

	// Step 3: Try standard parse
	if json.Valid([]byte(cleaned)) {
		return cleaned, nil
	}

	// Step 4: Repair common issues
	repaired := repairJSON(cleaned)
	if json.Valid([]byte(repaired)) {
		return repaired, nil
	}

	return "", fmt.Errorf("JSON parse failed after repair")
}

// stripMarkdownFences removes ```json ... ``` or ``` ... ``` wrappers.
func stripMarkdownFences(s string) string {
	s = strings.TrimSpace(s)

	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
	}

	if strings.HasSuffix(s, "```") {
		s = strings.TrimSuffix(s, "```")
	}

	return strings.TrimSpace(s)
}

// findJSONBoundaries locates the outermost JSON object or array in the string.
func findJSONBoundaries(s string) string {
	s = strings.TrimSpace(s)

	startArr := strings.Index(s, "[")
	startObj := strings.Index(s, "{")

	start := -1
	var openChar, closeChar byte

	if startArr >= 0 && (startObj < 0 || startArr < startObj) {
		start = startArr
		openChar = '['
		closeChar = ']'
	} else if startObj >= 0 {
		start = startObj
		openChar = '{'
		closeChar = '}'
	}

	if start < 0 {
		return ""
	}

	// Find the matching closing bracket, accounting for nesting and strings
	depth := 0
	inString := false
	escaped := false

	for i := start; i < len(s); i++ {
		c := s[i]

		if escaped {
			escaped = false
			continue
		}

		if c == '\\' && inString {
			escaped = true
			continue
		}

		if c == '"' {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		if c == openChar {
			depth++
		} else if c == closeChar {
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}

	// If we didn't find a matching close, return from start to end
	return s[start:]
}

// repairJSON attempts to fix common JSON issues from LLM output.
func repairJSON(s string) string {
	// Remove trailing commas before } or ]
	re := regexp.MustCompile(`,\s*([}\]])`)
	s = re.ReplaceAllString(s, "$1")

	// Remove control characters except \n \r \t
	var b strings.Builder
	b.Grow(len(s))
	inStr := false
	escaped := false
	for i := 0; i < len(s); i++ {
		c := s[i]

		if escaped {
			escaped = false
			b.WriteByte(c)
			continue
		}

		if c == '\\' && inStr {
			escaped = true
			b.WriteByte(c)
			continue
		}

		if c == '"' {
			inStr = !inStr
			b.WriteByte(c)
			continue
		}

		// Remove control chars inside strings
		if inStr && c < 0x20 && c != '\n' && c != '\r' && c != '\t' {
			continue
		}

		// Replace unescaped newlines inside strings with space
		if inStr && (c == '\n' || c == '\r') {
			b.WriteByte(' ')
			continue
		}

		b.WriteByte(c)
	}

	return b.String()
}

// ValidateQuestionType checks if a question type string is valid.
func ValidateQuestionType(qt string) bool {
	switch qt {
	case "mcq", "essay", "fill_blank", "true_false", "short_answer", "match", "assertion_reasoning":
		return true
	}
	return false
}

// ValidateBloomLevel checks if a Bloom's level string is valid.
func ValidateBloomLevel(bl string) bool {
	switch bl {
	case "remember", "understand", "apply", "analyze", "evaluate", "create":
		return true
	}
	return false
}

// ValidateDifficulty checks if a difficulty string is valid.
func ValidateDifficulty(d string) bool {
	switch d {
	case "easy", "medium", "hard":
		return true
	}
	return false
}
