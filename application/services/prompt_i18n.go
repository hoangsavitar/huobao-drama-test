package services

import (
	"fmt"

	"github.com/drama-generator/backend/pkg/config"
)

type PromptI18n struct {
	config *config.Config
}

func NewPromptI18n(cfg *config.Config) *PromptI18n {
	return &PromptI18n{config: cfg}
}

func (p *PromptI18n) GetLanguage() string {
	lang := p.config.App.Language
	if lang == "" {
		return "en"
	}
	return lang
}

func (p *PromptI18n) IsEnglish() bool {
	return p.GetLanguage() == "en"
}

func (p *PromptI18n) GetStoryboardSystemPrompt() string {
	return `[Role] You are a senior storyboard artist.
[Task] Break script into shots by independent action units.
[Requirements]
- One primary action per shot.
- Keep visual details concrete and cinematic.
- Keep timing realistic (4-12s per shot).
- Return JSON only.`
}

func (p *PromptI18n) GetSceneExtractionPrompt(style string) string {
	return fmt.Sprintf(`[Task] Extract unique scene backgrounds from script.
[Requirements]
- English only.
- Pure background, no people or characters.
- Detailed location, time, atmosphere.
- Style requirement: %s
- Return JSON only.`, style)
}

func (p *PromptI18n) GetFirstFramePrompt(style string) string {
	return fmt.Sprintf(`Generate an image prompt for the first frame.
- Static initial state only.
- No motion description.
- Style requirement: %s
- Return JSON object with fields: prompt, description.`, style)
}

func (p *PromptI18n) GetKeyFramePrompt(style string) string {
	return fmt.Sprintf(`Generate an image prompt for the key frame.
- Peak action moment.
- Strong emotion and motion tension.
- Style requirement: %s
- Return JSON object with fields: prompt, description.`, style)
}

func (p *PromptI18n) GetActionSequenceFramePrompt(style string) string {
	return fmt.Sprintf(`Generate one prompt for a complete 3x3 action-sequence panel.
- Ensure character consistency across all 9 panels.
- Show continuous motion progression.
- Style requirement: %s
- Return JSON object with fields: prompt, description.`, style)
}

func (p *PromptI18n) GetLastFramePrompt(style string) string {
	return fmt.Sprintf(`Generate an image prompt for the last frame.
- Final static outcome after action.
- Emphasize result and emotional landing.
- Style requirement: %s
- Return JSON object with fields: prompt, description.`, style)
}

func (p *PromptI18n) GetOutlineGenerationPrompt() string {
	return `Generate a short-drama outline from the given topic and constraints.
Return JSON only.`
}

func (p *PromptI18n) GetCharacterExtractionPrompt(style string) string {
	return fmt.Sprintf(`Extract up to the requested number of major characters from script.
- Use English fields.
- Keep appearance/personality concise and production-ready.
- Match style tone: %s
- Return JSON only.`, style)
}

func (p *PromptI18n) GetPropExtractionPrompt(style string) string {
	return fmt.Sprintf(`Extract key props from script.
- Include name, type, description, image_prompt.
- Keep prompts visual and generation-ready.
- Match style tone: %s
- Return JSON only.`, style)
}

func (p *PromptI18n) GetEpisodeScriptPrompt() string {
	return `Expand outline into episode scripts.
- Keep episode continuity and pacing.
- Return JSON only.`
}

func (p *PromptI18n) FormatUserPrompt(key string, args ...interface{}) string {
	prompts := map[string]string{
		"outline_request":        "Create a short-drama outline for topic:\n\n%s",
		"genre_preference":       "\nGenre preference: %s",
		"style_requirement":      "\nStyle requirement: %s",
		"episode_count":          "\nEpisode count: %d",
		"episode_importance":     "\n\nImportant: episodes array must contain exactly %d complete episodes.",
		"character_request":      "Script content:\n%s\n\nExtract up to %d major characters with structured details.",
		"episode_script_request": "Outline:\n%s\n%s\nWrite detailed scripts for %d episodes.\nReturn JSON with exactly %d episodes.",
		"frame_info":             "Shot info:\n%s\n\nGenerate first-frame prompt only:",
		"key_frame_info":         "Shot info:\n%s\n\nGenerate key-frame prompt only:",
		"last_frame_info":        "Shot info:\n%s\n\nGenerate last-frame prompt only:",
		"script_content_label":   "[Script Content]",
		"storyboard_list_label":  "[Storyboard List]",
		"task_label":             "[Task]",
		"character_list_label":   "[Available Characters]",
		"scene_list_label":       "[Available Scene Backgrounds]",
		"task_instruction":       "Break script into storyboard shots by independent action units.",
		"character_constraint":   "Important: characters field must use IDs from the provided character list only.",
		"scene_constraint":       "Important: scene_id must be selected from provided scene backgrounds; use null if none fits.",
		"shot_description_label": "Shot Description: %s",
		"scene_label":            "Scene: %s, %s",
		"characters_label":       "Characters: %s",
		"action_label":           "Action: %s",
		"result_label":           "Result: %s",
		"dialogue_label":         "Dialogue: %s",
		"atmosphere_label":       "Atmosphere: %s",
		"shot_type_label":        "Shot Type: %s",
		"angle_label":            "Angle: %s",
		"movement_label":         "Movement: %s",
		"drama_info_template":    "Title: %s\nDescription: %s\nGenre: %s",
	}

	template, exists := prompts[key]
	if !exists {
		return key
	}
	return fmt.Sprintf(template, args...)
}

func (p *PromptI18n) GetStylePrompt(style string) string {
	stylePrompts := map[string]string{
		"ghibli":          "Studio Ghibli-inspired, warm cinematic light, rich hand-painted background details.",
		"oriental_fantasy": "Oriental fantasy aesthetic, elegant composition, luminous mystical accents.",
		"post_apocalyptic": "Post-apocalyptic mood, desaturated palette, dramatic side-light, weathered environment.",
		"retro_anime":      "Retro anime look, film grain texture, soft cel shading, nostalgic color palette.",
		"pixel_art":        "Pixel-art rendering, limited palette, crisp aliased edges, game-scene readability.",
		"voxel":            "Voxel 3D block style, global illumination feel, miniature-world look.",
		"webtoon":          "Modern webtoon style, clean linework, urban cool-tone atmosphere.",
		"chinese_style":    "Chinese-style cinematic portrait, refined details, elegant natural lighting.",
		"chibi_3d":         "Chibi 3D collectible figurine look, high-quality material rendering, soft studio lighting.",
	}
	if prompt, exists := stylePrompts[style]; exists {
		return prompt
	}
	return ""
}

func (p *PromptI18n) GetVideoConstraintPrompt(referenceMode string) string {
	switch referenceMode {
	case "single":
		return `Single-image mode: treat input as frame zero, extend action naturally from visible tension cues.`
	case "first_last":
		return `First-last mode: generate coherent temporal transition from first frame to last frame.`
	case "multiple":
		return `Multi-image mode: preserve identity and style consistency across all reference images.`
	case "action_sequence":
		return `Action-sequence mode: output one coherent multi-panel motion progression with strict character consistency.`
	default:
		return `Generate a coherent video prompt with consistent character identity and scene continuity.`
	}
}
