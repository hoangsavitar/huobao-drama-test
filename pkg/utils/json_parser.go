package utils

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// SafeParseAIJSON parses AI-returned JSON robustly, handling common format issues:
// 1. Strip Markdown code fences
// 2. Extract JSON object/array
// 3. Clean extra whitespace/newlines
// 4. Attempt to repair truncated JSON
// 5. Provide detailed error context
func SafeParseAIJSON(aiResponse string, v interface{}) error {
	if aiResponse == "" {
		return fmt.Errorf("AI returned empty content")
	}

	cleaned := strings.TrimSpace(aiResponse)
	cleaned = regexp.MustCompile("(?m)^```json\\s*").ReplaceAllString(cleaned, "")
	cleaned = regexp.MustCompile("(?m)^```\\s*").ReplaceAllString(cleaned, "")
	cleaned = regexp.MustCompile("(?m)```\\s*$").ReplaceAllString(cleaned, "")
	cleaned = strings.TrimSpace(cleaned)

	var jsonMatch string

	if strings.HasPrefix(cleaned, "{") {
		jsonRegex := regexp.MustCompile(`(?s)\{.*\}`)
		jsonMatch = jsonRegex.FindString(cleaned)
	}

	if jsonMatch == "" && strings.HasPrefix(cleaned, "[") {
		jsonRegex := regexp.MustCompile(`(?s)\[.*\]`)
		jsonMatch = jsonRegex.FindString(cleaned)
	}

	if jsonMatch == "" {
		objRegex := regexp.MustCompile(`(?s)\{.*\}`)
		jsonMatch = objRegex.FindString(cleaned)

		if jsonMatch == "" {
			arrRegex := regexp.MustCompile(`(?s)\[.*\]`)
			jsonMatch = arrRegex.FindString(cleaned)
		}
	}

	if jsonMatch == "" {
		return fmt.Errorf("No valid JSON object/array found in response. Raw response: %s", truncateString(aiResponse, 200))
	}

	err := json.Unmarshal([]byte(jsonMatch), v)
	if err == nil {
		return nil
	}

	fixedJSON := attemptJSONRepair(jsonMatch)
	if fixedJSON != jsonMatch {
		if err := json.Unmarshal([]byte(fixedJSON), v); err == nil {
			return nil
		}
	}

	if isTruncated(jsonMatch) {
		return fmt.Errorf(
			"AI response may be truncated and JSON is incomplete.\nTry:\n1. Increase maxTokens\n2. Simplify input\n3. Use a more capable model\n\nOriginal error: %s\nResponse length: %d\nResponse tail: %s",
			err.Error(),
			len(jsonMatch),
			truncateString(jsonMatch[maxInt(0, len(jsonMatch)-200):], 200),
		)
	}

	if jsonErr, ok := err.(*json.SyntaxError); ok {
		errorPos := int(jsonErr.Offset)
		start := maxInt(0, errorPos-100)
		end := minInt(len(jsonMatch), errorPos+100)

		context := jsonMatch[start:end]
		marker := strings.Repeat(" ", errorPos-start) + "^"

		return fmt.Errorf(
			"JSON parse failed: %s\nContext near error:\n%s\n%s",
			jsonErr.Error(),
			context,
			marker,
		)
	}

	return fmt.Errorf("JSON parse failed: %w\nRaw response: %s", err, truncateString(jsonMatch, 300))
}

// attemptJSONRepair attempts to repair common JSON issues.
func attemptJSONRepair(jsonStr string) string {
	trimmed := strings.TrimSpace(jsonStr)

	if strings.Count(trimmed, `"`)%2 != 0 {
		trimmed += `"`
	}

	openBraces := strings.Count(trimmed, "{")
	closeBraces := strings.Count(trimmed, "}")
	openBrackets := strings.Count(trimmed, "[")
	closeBrackets := strings.Count(trimmed, "]")

	for closeBrackets > openBrackets && len(trimmed) > 0 {
		lastIdx := strings.LastIndex(trimmed, "]")
		if lastIdx >= 0 {
			trimmed = trimmed[:lastIdx] + trimmed[lastIdx+1:]
			closeBrackets--
		} else {
			break
		}
	}

	for closeBraces > openBraces && len(trimmed) > 0 {
		lastIdx := strings.LastIndex(trimmed, "}")
		if lastIdx >= 0 {
			trimmed = trimmed[:lastIdx] + trimmed[lastIdx+1:]
			closeBraces--
		} else {
			break
		}
	}

	openBraces = strings.Count(trimmed, "{")
	closeBraces = strings.Count(trimmed, "}")
	openBrackets = strings.Count(trimmed, "[")
	closeBrackets = strings.Count(trimmed, "]")

	for i := 0; i < openBrackets-closeBrackets; i++ {
		trimmed += "]"
	}

	for i := 0; i < openBraces-closeBraces; i++ {
		trimmed += "}"
	}

	return trimmed
}

// ExtractJSONFromText extracts a JSON object/array from text.
func ExtractJSONFromText(text string) string {
	text = strings.TrimSpace(text)

	text = regexp.MustCompile("(?m)^```json\\s*").ReplaceAllString(text, "")
	text = regexp.MustCompile("(?m)^```\\s*").ReplaceAllString(text, "")
	text = strings.TrimSpace(text)

	if idx := strings.Index(text, "{"); idx != -1 {
		if lastIdx := strings.LastIndex(text, "}"); lastIdx != -1 && lastIdx > idx {
			return text[idx : lastIdx+1]
		}
	}

	if idx := strings.Index(text, "["); idx != -1 {
		if lastIdx := strings.LastIndex(text, "]"); lastIdx != -1 && lastIdx > idx {
			return text[idx : lastIdx+1]
		}
	}

	return text
}

// ValidateJSON validates whether a JSON string is valid.
func ValidateJSON(jsonStr string) error {
	var js json.RawMessage
	return json.Unmarshal([]byte(jsonStr), &js)
}

// isTruncated checks whether a JSON string might be truncated.
func isTruncated(jsonStr string) bool {
	trimmed := strings.TrimSpace(jsonStr)
	if len(trimmed) == 0 {
		return false
	}

	lastChar := trimmed[len(trimmed)-1]
	if lastChar != '}' && lastChar != ']' {
		return true
	}

	openBraces := strings.Count(trimmed, "{")
	closeBraces := strings.Count(trimmed, "}")
	openBrackets := strings.Count(trimmed, "[")
	closeBrackets := strings.Count(trimmed, "]")

	if openBraces != closeBraces || openBrackets != closeBrackets {
		return true
	}

	quoteCount := strings.Count(trimmed, `"`)
	if quoteCount%2 != 0 {
		return true
	}

	return false
}

// Helper functions
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
