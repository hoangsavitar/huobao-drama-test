package services

import (
"encoding/json"
"regexp"
"strings"
)

// parseFramePromptJSON parses AI-returned JSON format prompt
func (s *FramePromptService) parseFramePromptJSON(aiResponse string) *SingleFramePrompt {
// Clean possible markdown code block markers
cleaned := strings.TrimSpace(aiResponse)

// Remove ```json and ``` markers
re := regexp.MustCompile("(?s)```json\\s*(.+?)\\s*```")
if matches := re.FindStringSubmatch(cleaned); len(matches) > 1 {
cleaned = strings.TrimSpace(matches[1])
} else {
// Remove standalone ``` markers
cleaned = strings.Trim(cleaned, "`")
cleaned = strings.TrimSpace(cleaned)
}

// Try to parse JSON
var result SingleFramePrompt
if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
s.log.Warnw("Failed to parse JSON", "error", err, "cleaned_response", cleaned)
return nil
}

// Validate required fields
if result.Prompt == "" {
s.log.Warnw("Parsed JSON missing prompt field", "response", cleaned)
return nil
}

return &result
}
