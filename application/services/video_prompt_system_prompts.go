package services

import (
	_ "embed"
	"strings"
)

//go:embed system_prompt_video_LTX.md
var systemPromptVideoLtx string

// GetVideoPromptSystemPrompt returns the system prompt template for a given video model key.
// For now this repo hardcodes to LTX, but the lookup is structured to support more md files later.
func GetVideoPromptSystemPrompt(modelKey string) string {
	key := strings.ToLower(strings.TrimSpace(modelKey))
	if key == "" {
		return systemPromptVideoLtx
	}

	// Hardcode mapping (extend later with more models/md files).
	if strings.Contains(key, "ltx") {
		return systemPromptVideoLtx
	}

	// Fallback to LTX.
	return systemPromptVideoLtx
}

