package services

import (
	"strconv"

	"fmt"
	"strings"

	models "github.com/drama-generator/backend/domain/models"
	"github.com/drama-generator/backend/pkg/ai"
	"github.com/drama-generator/backend/pkg/config"
	"github.com/drama-generator/backend/pkg/logger"
	"github.com/drama-generator/backend/pkg/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type StoryboardService struct {
	db          *gorm.DB
	aiService   *AIService
	taskService *TaskService
	log         *logger.Logger
	config      *config.Config
	promptI18n  *PromptI18n
}

func NewStoryboardService(db *gorm.DB, cfg *config.Config, log *logger.Logger) *StoryboardService {
	return &StoryboardService{
		db:          db,
		aiService:   NewAIService(db, log),
		taskService: NewTaskService(db, log),
		log:         log,
		config:      cfg,
		promptI18n:  NewPromptI18n(cfg),
	}
}

type Storyboard struct {
	ShotNumber  int    `json:"shot_number"`
	Title       string `json:"title"`        // Shot title
	ShotType    string `json:"shot_type"`    // Shot size
	Angle       string `json:"angle"`        // Camera angle
	Time        string `json:"time"`         // Time
	Location    string `json:"location"`     // Location
	SceneID     *uint  `json:"scene_id"`     // Scene ID (returned directly by AI, can be null)
	Movement    string `json:"movement"`     // Camera movement
	Action      string `json:"action"`       // Action
	Dialogue    string `json:"dialogue"`     // Dialogue/monologue
	Result      string `json:"result"`       // Visual result
	Atmosphere  string `json:"atmosphere"`   // Environmental atmosphere
	Emotion     string `json:"emotion"`      // Emotion
	Duration    int    `json:"duration"`     // Duration (seconds)
	BgmPrompt   string `json:"bgm_prompt"`   // Background music prompt
	SoundEffect string `json:"sound_effect"` // Sound effect description
	Characters  []uint `json:"characters"`   // List of involved character IDs
	IsPrimary   bool   `json:"is_primary"`   // Whether this is a primary shot
}

type GenerateStoryboardResult struct {
	Storyboards []Storyboard `json:"storyboards"`
	Total       int          `json:"total"`
}

func (s *StoryboardService) GenerateStoryboard(episodeID string, model string) (string, error) {
	// Get episode information from database
	var episode struct {
		ID            string
		ScriptContent *string
		Description   *string
		DramaID       string
	}

	err := s.db.Table("episodes").
		Select("episodes.id, episodes.script_content, episodes.description, episodes.drama_id").
		Joins("INNER JOIN dramas ON dramas.id = episodes.drama_id").
		Where("episodes.id = ?", episodeID).
		First(&episode).Error

	if err != nil {
		return "", fmt.Errorf("episode not found or access denied")
	}

	// Get script content
	var scriptContent string
	if episode.ScriptContent != nil && *episode.ScriptContent != "" {
		scriptContent = *episode.ScriptContent
	} else if episode.Description != nil && *episode.Description != "" {
		scriptContent = *episode.Description
	} else {
		return "", fmt.Errorf("script content is empty, generate episode content first")
	}

	// Get all characters for this drama
	var characters []models.Character
	if err := s.db.Where("drama_id = ?", episode.DramaID).Order("name ASC").Find(&characters).Error; err != nil {
		return "", fmt.Errorf("failed to get character list: %w", err)
	}

	// Build character list string (including ID and name)
	characterList := "No characters"
	if len(characters) > 0 {
		var charInfoList []string
		for _, char := range characters {
			charInfoList = append(charInfoList, fmt.Sprintf(`{"id": %d, "name": "%s"}`, char.ID, char.Name))
		}
		characterList = fmt.Sprintf("[%s]", strings.Join(charInfoList, ", "))
	}

	// Get extracted scene list for this project (project level)
	var scenes []models.Scene
	if err := s.db.Where("drama_id = ?", episode.DramaID).Order("location ASC, time ASC").Find(&scenes).Error; err != nil {
		s.log.Warnw("Failed to get scenes", "error", err)
	}

	// Build scene list string (including ID, location, time)
	sceneList := "No scenes"
	if len(scenes) > 0 {
		var sceneInfoList []string
		for _, bg := range scenes {
			sceneInfoList = append(sceneInfoList, fmt.Sprintf(`{"id": %d, "location": "%s", "time": "%s"}`, bg.ID, bg.Location, bg.Time))
		}
		sceneList = fmt.Sprintf("[%s]", strings.Join(sceneInfoList, ", "))
	}

	// Use internationalized prompts
	systemPrompt := s.promptI18n.GetStoryboardSystemPrompt()

	scriptLabel := s.promptI18n.FormatUserPrompt("script_content_label")
	taskLabel := s.promptI18n.FormatUserPrompt("task_label")
	taskInstruction := s.promptI18n.FormatUserPrompt("task_instruction")
	charListLabel := s.promptI18n.FormatUserPrompt("character_list_label")
	charConstraint := s.promptI18n.FormatUserPrompt("character_constraint")
	sceneListLabel := s.promptI18n.FormatUserPrompt("scene_list_label")
	sceneConstraint := s.promptI18n.FormatUserPrompt("scene_constraint")

	prompt := fmt.Sprintf(`%s

%s
%s

%s%s

%s
%s

%s

%s
%s

%s

【Original Script】
%s

【Storyboard Elements】Each shot focuses on a single action, with thorough and specific descriptions:
1. **Shot Title (title)**: Summarize the core content or emotion of the shot in 3-5 words
   - Examples: "Nightmare Awakening", "Silent Eye Contact", "Fleeing the Scene", "Unexpected Discovery"
2. **Time**: [Dawn/Afternoon/Late night/Specific time + detailed lighting description]
   - Example: "Late night 22:30 · Moonlight slants through broken windows into the room, creating a boundary between light and shadow"
3. **Location**: [Complete scene description + spatial layout + environmental details]
   - Example: "Abandoned dock warehouse · Rows of rusted shelves, standing water on the floor reflecting faint light, rotting wooden crates piled in the corner"
4. **Shot Design**:
   - **Shot Size (shot_type)**: [Extreme wide/Wide/Medium/Close-up/Extreme close-up]
   - **Camera Angle (angle)**: [Eye level/Low angle/High angle/Side view/Rear view]
   - **Camera Movement (movement)**: [Static/Push in/Pull out/Pan/Tracking/Dolly]
5. **Character Action**: **Detailed action description**, including [who + specific action + body details + facial expression]
   - Example: "Chen Zheng bends down using a crowbar to pry open the safe door, veins bulging on his arms, brows furrowed, sweat dripping down his cheeks"
6. **Dialogue/Monologue**: Extract the complete dialogue or monologue in the shot (empty string if no dialogue)
7. **Visual Result**: Immediate consequence of the action + visual details + atmosphere change
   - Example: "The safe door springs open with a metallic clang, dust rises and drifts in the flashlight beam, the safe is empty except for old newspapers, Chen Zheng's expression shifts from anticipation to disappointment"
8. **Environmental Atmosphere**: Lighting quality + color tone + sound environment + overall mood
   - Example: "Dim cold tones, only the flashlight beam swaying in the darkness, distant sound of waves crashing, oppressive and heavy"
9. **BGM Prompt (bgm_prompt)**: Describe the mood, rhythm, and emotion of the background music for this shot (empty string if no special requirement)
   - Example: "Deep, tense strings, slow rhythm, creating an oppressive atmosphere"
10. **Sound Effect Description (sound_effect)**: Describe the key sound effects for this shot (empty string if no special effects)
    - Example: "Metallic clang, footsteps, sound of waves crashing"
11. **Audience Emotion**: [Emotion type] ([Intensity: ↑↑↑/↑↑/↑/→/↓] + [Resolution: Suspended/Released/Reversed])

【Output Format】Please output in JSON format, each shot containing the following fields (**all descriptive fields must be detailed and complete**):
{
  "storyboards": [
    {
      "shot_number": 1,
      "title": "Nightmare Awakening",
      "shot_type": "Wide shot",
      "angle": "High angle 45 degrees",
      "time": "Late night 22:30 · Moonlight slants through broken windows into the warehouse, creating silver reflections on the standing water, dark corners barely visible",
      "location": "Abandoned dock warehouse · Rows of rusted shelves, standing water reflecting faint light, rotting wooden crates and fishing nets piled in the corner, damp musty smell permeating the air",
      "scene_id": 1,
      "movement": "Static shot",
      "action": "Chen Zheng bends down gripping the crowbar with both hands to pry the safe door open, veins bulging on his arms, brows tightly furrowed, sweat dripping from his forehead down his cheeks, breathing heavily",
      "dialogue": "(Monologue) After all these years, what secret is hidden inside?",
      "result": "The safe door suddenly springs open with a sharp metallic sound, dust rises and drifts in the flashlight beam, the safe is empty except for a few yellowed old newspapers, Chen Zheng's expression shifts from anticipation to shock and disappointment, pupils dilating",
      "atmosphere": "Dim cold tones · Predominantly gray-blue, only the flashlight beam swaying in the darkness, distant sound of waves crashing against the dock, overall oppressive and heavy atmosphere",
      "emotion": "Curiosity ↑↑ turning to disappointment ↓ (emotional reversal)",
      "duration": 9,
      "bgm_prompt": "Deep tense strings, slow rhythm, creating an oppressive suspenseful atmosphere",
      "sound_effect": "Metallic clang, dust drifting sound, waves crashing",
      "characters": [159],
      "is_primary": true
    },
    {
      "shot_number": 2,
      "title": "Silent Eye Contact",
      "shot_type": "Close-up",
      "angle": "Eye level",
      "time": "Late night 22:31 · Dim lighting inside the warehouse, only the flashlight illuminating the two faces from the side",
      "location": "Abandoned dock warehouse · Beside the safe, blurred shelf silhouettes in the background",
      "scene_id": 1,
      "movement": "Push in",
      "action": "Chen Zheng slowly turns around, locking eyes with Li Fang behind him, Li Fang holding the flashlight with its beam swaying between them, her eyes revealing doubt and vigilance",
      "dialogue": "Chen Zheng: \"We've been played, there's nothing here that we're looking for.\" Li Fang: \"What now? We're running out of time.\"",
      "result": "The two stand in the darkness lost in thought, the flashlight beam casting a circular spot on the floor, faint metallic scraping sounds in the background, tense and heavy atmosphere",
      "atmosphere": "Low-key lighting · Shadows cover 70%% of the frame, hard side lighting outlines the characters, strong warm-cool contrast, howling sea wind creating a sense of urgency",
      "emotion": "Tension ↑↑ · Vigilance ↑↑ (Suspended)",
      "duration": 7,
      "bgm_prompt": "Gradually escalating tension sounds, low-frequency sustained tone",
      "sound_effect": "Breathing sounds, metallic scraping, howling sea wind",
      "characters": [159, 160],
      "is_primary": true
    }
  ]
}

**Dialogue Field Instructions**:
- If there is dialogue, format as: Character Name: "Line content"
- Multiple character dialogue separated by spaces: Character A: "..." Character B: "..."
- Monologue format: (Monologue) Content
- Narration format: (Narration) Content
- Empty string when there is no dialogue: ""
- **Dialogue content must be extracted from the original script, preserving the original wording**

**Character and Scene Requirements**:
- The characters field must include all character IDs appearing in the shot (numeric array format)
- Only extract IDs of characters that actually appear; use empty array [] if no characters appear
- **Character IDs must strictly use the id field (numeric) from the 【Available Character List】, no other IDs or made-up characters allowed**
- Example: If Li Ming (id:159) and Wang Fang (id:160) appear in the shot, the characters field should be [159, 160]
- The scene_id field must select the best matching scene ID (numeric) from the 【Extracted Scene List】
- If no suitable scene exists in the list, set scene_id to null
- Example: If the shot takes place in "City apartment bedroom · Early morning", select the scene with id 1

**Duration Estimation Rules (seconds)**:
- **All shot durations must be within the 4-12 second range** to ensure smooth and reasonable pacing
- **Comprehensive estimation principle**: Duration is determined by dialogue content, action complexity, and emotional pacing combined

**Estimation Steps**:
1. **Base Duration** (determined by scene content):
   - Pure dialogue scene (no significant action): Base 4 seconds
   - Pure action scene (no dialogue): Base 5 seconds
   - Mixed dialogue + action scene: Base 6 seconds

2. **Dialogue Adjustment** (add time based on line word count):
   - No dialogue: +0 seconds
   - Short dialogue (1-20 characters): +1-2 seconds
   - Medium dialogue (21-50 characters): +2-4 seconds
   - Long dialogue (51+ characters): +4-6 seconds

3. **Action Adjustment** (add time based on action complexity):
   - No action/static: +0 seconds
   - Simple action (expression, turning, picking up object): +0-1 seconds
   - Normal action (walking, opening door, sitting down): +1-2 seconds
   - Complex action (fighting, chasing, large movements): +2-4 seconds
   - Environment showcase (panoramic scan, atmosphere building): +2-5 seconds

4. **Final Duration** = Base duration + Dialogue adjustment + Action adjustment, ensuring result is within 4-12 second range

**Examples**:
- "Chen Zheng turns and leaves" (simple action, no dialogue): 5 + 0 + 1 = 6 seconds
- "Li Fang: 'Where are you going?'" (short dialogue, no action): 4 + 2 + 0 = 6 seconds  
- "Chen Zheng pushes open the door, Li Fang: 'Finally found you, where have you been all these years?'" (normal action + medium dialogue): 6 + 3 + 2 = 11 seconds
- "The two fight intensely in the rain, Chen Zheng: 'Stop!'" (complex action + short dialogue): 6 + 2 + 4 = 12 seconds

**Important**: Accurately estimate each shot's duration; the sum of all storyboard durations will be used as the total episode duration

**Special Requirements**:
- **【Extremely Important】Must completely break down the entire script 100%%, no omitting, skipping, or compressing any plot content**
- **Convert every sentence and paragraph from the first word to the last word of the script into storyboard shots**
- **Every dialogue, every action, every scene transition must have a corresponding storyboard shot**
- The longer the script, the more storyboard shots (short scripts 15-30, medium scripts 30-60, long scripts 60-100 or more)
- **Better to have more shots than to miss any plot**: A long scene can be split into multiple consecutive shots
- Each shot describes only one main action
- Distinguish between primary shots (is_primary: true) and linking shots (is_primary: false)
- Ensure emotional pacing has variation
- **The duration field is crucial**: Accurately estimate each shot's duration, which will be used to calculate the total episode duration
- Strictly output in JSON format

**【Prohibited Actions】**:
- ❌ Do not summarize multiple scenes in one shot
- ❌ Do not skip any dialogue or monologue
- ❌ Do not omit plot development
- ❌ Do not merge shots that should be separate
- ✅ Correct approach: Break down the script into a corresponding number of shots so that viewers can fully understand the plot after watching all shots

**【Key】Scene Description Detail Requirements** (these descriptions will be directly used for video generation models):
1. **Time field**: Must contain ≥15 characters of detailed description
   - ✓ Good example: "Late night 22:30 · Moonlight slants through broken windows into the warehouse, creating silver reflections on the standing water, dark corners barely visible"
   - ✗ Bad example: "Late night"

2. **Location field**: Must contain ≥20 characters of detailed scene description
   - ✓ Good example: "Abandoned dock warehouse · Rows of rusted shelves, standing water reflecting faint light, rotting wooden crates and fishing nets in the corner, damp musty smell in the air"
   - ✗ Bad example: "Warehouse"

3. **Action field**: Must contain ≥25 characters of detailed action description, including body details and expressions
   - ✓ Good example: "Chen Zheng bends down gripping the crowbar with both hands to pry the safe door open, veins bulging on his arms, brows furrowed, sweat dripping from his forehead, breathing heavily"
   - ✗ Bad example: "Chen Zheng opens the safe"

4. **Result field**: Must contain ≥25 characters of detailed visual result description
   - ✓ Good example: "The safe door suddenly springs open with a sharp metallic sound, dust rises and drifts in the flashlight beam, the safe is empty except for a few yellowed old newspapers, Chen Zheng's expression shifts from anticipation to shock and disappointment, pupils dilating"
   - ✗ Bad example: "The door opened"

5. **Atmosphere field**: Must contain ≥20 characters of environmental atmosphere description, including lighting, color tone, and sound
   - ✓ Good example: "Dim cold tones · Predominantly gray-blue, only the flashlight beam swaying in the darkness, distant sound of waves crashing against the dock, overall oppressive and heavy atmosphere"
   - ✗ Bad example: "Dim"

**Description Principles**:
- All descriptive fields should be as detailed as if describing a scene to a blind person
- Include sensory details: visual, auditory, tactile, olfactory
- Describe lighting, color, texture, and movement
- Provide sufficient visual information for the video generation AI to construct the scene
- Avoid abstract words, use concrete visual descriptions`, systemPrompt, scriptLabel, scriptContent, taskLabel, taskInstruction, charListLabel, characterList, charConstraint, sceneListLabel, sceneList, sceneConstraint)

	// Create async task
	task, err := s.taskService.CreateTask("storyboard_generation", episodeID)
	if err != nil {
		s.log.Errorw("Failed to create task", "error", err)
		return "", fmt.Errorf("failed to create task: %w", err)
	}

	s.log.Infow("Generating storyboard asynchronously",
		"task_id", task.ID,
		"episode_id", episodeID,
		"drama_id", episode.DramaID,
		"script_length", len(scriptContent),
		"character_count", len(characters),
		"characters", characterList,
		"scene_count", len(scenes),
		"scenes", sceneList)

	// Start background goroutine for AI call and subsequent logic
	go s.processStoryboardGeneration(task.ID, episodeID, model, prompt)

	// Return task ID immediately
	return task.ID, nil
}

// processStoryboardGeneration processes storyboard generation in background
func (s *StoryboardService) processStoryboardGeneration(taskID, episodeID, model, prompt string) {
	// Update task status to processing
	if err := s.taskService.UpdateTaskStatus(taskID, "processing", 10, "Generating storyboard..."); err != nil {
		s.log.Errorw("Failed to update task status", "error", err, "task_id", taskID)
		return
	}

	s.log.Infow("Processing storyboard generation", "task_id", taskID, "episode_id", episodeID)

	// Call AI service to generate (use specified model if provided)
	// Set large max_tokens to ensure complete JSON return of all storyboard shots
	var text string
	var err error
	if model != "" {
		s.log.Infow("Using specified model for storyboard generation", "model", model, "task_id", taskID)
		client, getErr := s.aiService.GetAIClientForModel("text", model)
		if getErr != nil {
			s.log.Warnw("Failed to get client for specified model, using default", "model", model, "error", getErr, "task_id", taskID)
			text, err = s.aiService.GenerateText(prompt, "", ai.WithMaxTokens(16000))
		} else {
			text, err = client.GenerateText(prompt, "", ai.WithMaxTokens(16000))
		}
	} else {
		text, err = s.aiService.GenerateText(prompt, "", ai.WithMaxTokens(16000))
	}

	if err != nil {
		s.log.Errorw("Failed to generate storyboard", "error", err, "task_id", taskID)
		if updateErr := s.taskService.UpdateTaskError(taskID, fmt.Errorf("failed to generate storyboard: %w", err)); updateErr != nil {
			s.log.Errorw("Failed to update task error", "error", updateErr, "task_id", taskID)
		}
		return
	}

	// Update task progress
	if err := s.taskService.UpdateTaskStatus(taskID, "processing", 50, "Storyboard generated, parsing result..."); err != nil {
		s.log.Errorw("Failed to update task status", "error", err, "task_id", taskID)
		return
	}

	// Parse JSON result
	// AI may return two formats:
	// 1. Array format: [{...}, {...}]
	// 2. Object format: {"storyboards": [{...}, {...}]}
	var result GenerateStoryboardResult

	// Try parsing as array format first
	var storyboards []Storyboard
	if err := utils.SafeParseAIJSON(text, &storyboards); err == nil {
		// Successfully parsed as array, wrap as object
		result.Storyboards = storyboards
		result.Total = len(storyboards)
		s.log.Infow("Parsed storyboard as array format", "count", len(storyboards), "task_id", taskID)
	} else {
		// Try parsing as object format
		if err := utils.SafeParseAIJSON(text, &result); err != nil {
			s.log.Errorw("Failed to parse storyboard JSON in both formats", "error", err, "response", text[:min(500, len(text))], "task_id", taskID)
			if updateErr := s.taskService.UpdateTaskError(taskID, fmt.Errorf("failed to parse storyboard result: %w", err)); updateErr != nil {
				s.log.Errorw("Failed to update task error", "error", updateErr, "task_id", taskID)
			}
			return
		}
		result.Total = len(result.Storyboards)
		s.log.Infow("Parsed storyboard as object format", "count", len(result.Storyboards), "task_id", taskID)
	}

	// Calculate total duration (sum of all storyboard durations)
	totalDuration := 0
	for _, sb := range result.Storyboards {
		totalDuration += sb.Duration
	}

	s.log.Infow("Storyboard generated",
		"task_id", taskID,
		"episode_id", episodeID,
		"count", result.Total,
		"total_duration_seconds", totalDuration)

	// Update task progress
	if err := s.taskService.UpdateTaskStatus(taskID, "processing", 70, "Saving storyboards..."); err != nil {
		s.log.Errorw("Failed to update task status", "error", err, "task_id", taskID)
		return
	}

	// Save storyboard shots to database
	if err := s.saveStoryboards(episodeID, result.Storyboards); err != nil {
		s.log.Errorw("Failed to save storyboards", "error", err, "task_id", taskID)
		if updateErr := s.taskService.UpdateTaskError(taskID, fmt.Errorf("failed to save storyboards: %w", err)); updateErr != nil {
			s.log.Errorw("Failed to update task error", "error", updateErr, "task_id", taskID)
		}
		return
	}

	// Update task progress
	if err := s.taskService.UpdateTaskStatus(taskID, "processing", 90, "Updating episode duration..."); err != nil {
		s.log.Errorw("Failed to update task status", "error", err, "task_id", taskID)
		return
	}

	// Update episode duration (seconds to minutes, round up)
	durationMinutes := (totalDuration + 59) / 60
	if err := s.db.Model(&models.Episode{}).Where("id = ?", episodeID).Update("duration", durationMinutes).Error; err != nil {
		s.log.Errorw("Failed to update episode duration", "error", err, "task_id", taskID)
		// Don't interrupt the flow, just log the error
	} else {
		s.log.Infow("Episode duration updated",
			"task_id", taskID,
			"episode_id", episodeID,
			"duration_seconds", totalDuration,
			"duration_minutes", durationMinutes)
	}

	// Update task result
	resultData := gin.H{
		"storyboards":      result.Storyboards,
		"total":            result.Total,
		"total_duration":   totalDuration,
		"duration_minutes": durationMinutes,
	}

	if err := s.taskService.UpdateTaskResult(taskID, resultData); err != nil {
		s.log.Errorw("Failed to update task result", "error", err, "task_id", taskID)
		return
	}

	s.log.Infow("Storyboard generation completed", "task_id", taskID, "episode_id", episodeID)
}

// generateImagePrompt generates a prompt specifically for image generation (first frame static image)
func (s *StoryboardService) generateImagePrompt(sb Storyboard) string {
	var parts []string

	// 1. Complete scene background description
	if sb.Location != "" {
		locationDesc := sb.Location
		if sb.Time != "" {
			locationDesc += ", " + sb.Time
		}
		parts = append(parts, locationDesc)
	}

	// 2. Character initial static pose (remove action process, keep starting state only)
	if sb.Action != "" {
		initialPose := extractInitialPose(sb.Action)
		if initialPose != "" {
			parts = append(parts, initialPose)
		}
	}

	// 3. Emotional atmosphere
	if sb.Emotion != "" {
		parts = append(parts, sb.Emotion)
	}

	// 4. Anime style
	parts = append(parts, "anime style, first frame")

	if len(parts) > 0 {
		return strings.Join(parts, ", ")
	}
	return "anime scene"
}

// extractInitialPose extracts the initial static pose (removes action process)
func extractInitialPose(action string) string {
	// Remove action process keywords, keep initial state description
	processWords := []string{
		"then", "next", "afterwards", "subsequently", "immediately after",
		"downward", "upward", "forward", "backward", "leftward", "rightward",
		"begin", "continue", "gradually", "slowly", "quickly", "suddenly", "abruptly",
	}

	result := action
	for _, word := range processWords {
		if idx := strings.Index(result, word); idx > 0 {
			// Truncate before the action process word
			result = result[:idx]
			break
		}
	}

	// Clean trailing punctuation
	return strings.TrimSpace(result)
}

// extractSimpleLocation extracts simplified scene location (removes detailed description)
func extractSimpleLocation(location string) string {
	// Truncate at "·" symbol, keep only the main scene name
	if idx := strings.Index(location, "·"); idx > 0 {
		return strings.TrimSpace(location[:idx])
	}

	// If there's a comma, keep only the first part
	if idx := strings.Index(location, "，"); idx > 0 {
		return strings.TrimSpace(location[:idx])
	}
	if idx := strings.Index(location, ","); idx > 0 {
		return strings.TrimSpace(location[:idx])
	}

	// Limit length to no more than 15 characters
	maxLen := 15
	if len(location) > maxLen {
		return strings.TrimSpace(location[:maxLen])
	}

	return strings.TrimSpace(location)
}

// extractSimplePose extracts simple core pose keywords (no more than 10 characters)
func extractSimplePose(action string) string {
	// Extract only the first 10 characters as core pose
	runes := []rune(action)
	maxLen := 10
	if len(runes) > maxLen {
		// Truncate at punctuation
		truncated := runes[:maxLen]
		for i := maxLen - 1; i >= 0; i-- {
			if truncated[i] == '，' || truncated[i] == '。' || truncated[i] == ',' || truncated[i] == '.' {
				truncated = runes[:i]
				break
			}
		}
		return strings.TrimSpace(string(truncated))
	}
	return strings.TrimSpace(action)
}

// extractFirstFramePose extracts the first frame static pose from action description
func extractFirstFramePose(action string) string {
	// Remove action process keywords, keep initial state
	processWords := []string{
		"then", "next", "downward", "forward", "walk toward", "rush toward", "turn around",
		"begin", "continue", "gradually", "slowly", "quickly", "suddenly",
	}

	pose := action
	for _, word := range processWords {
		// Simple handling: truncate before these words
		if idx := strings.Index(pose, word); idx > 0 {
			pose = pose[:idx]
			break
		}
	}

	// Clean trailing punctuation
	return strings.TrimSpace(pose)
}

// extractCompositionType extracts composition type from shot type (removes camera movement)
func extractCompositionType(shotType string) string {
	// Remove camera movement related descriptions
	cameraMovements := []string{
		"shake", "sway", "push in", "pull out", "follow", "orbit",
		"camera movement", "cinematography", "move", "rotate",
	}

	comp := shotType
	for _, movement := range cameraMovements {
		comp = strings.ReplaceAll(comp, movement, "")
	}

	// Clean extra punctuation and spaces
	comp = strings.ReplaceAll(comp, "··", "·")
	comp = strings.ReplaceAll(comp, "·", " ")
	comp = strings.TrimSpace(comp)

	return comp
}

// generateVideoPrompt generates a prompt specifically for video generation (includes camera movement and dynamic elements)
func (s *StoryboardService) generateVideoPrompt(sb Storyboard) string {
	var parts []string
	videoRatio := "16:9"
	// 1. Character action
	if sb.Action != "" {
		parts = append(parts, fmt.Sprintf("Action: %s", sb.Action))
	}

	// 2. Dialogue
	if sb.Dialogue != "" {
		parts = append(parts, fmt.Sprintf("Dialogue: %s", sb.Dialogue))
	}

	// 3. Camera movement (video-specific)
	if sb.Movement != "" {
		parts = append(parts, fmt.Sprintf("Camera movement: %s", sb.Movement))
	}

	// 4. Shot type and angle
	if sb.ShotType != "" {
		parts = append(parts, fmt.Sprintf("Shot type: %s", sb.ShotType))
	}
	if sb.Angle != "" {
		parts = append(parts, fmt.Sprintf("Camera angle: %s", sb.Angle))
	}

	// 5. Scene environment
	if sb.Location != "" {
		locationDesc := sb.Location
		if sb.Time != "" {
			locationDesc += ", " + sb.Time
		}
		parts = append(parts, fmt.Sprintf("Scene: %s", locationDesc))
	}

	// 6. Environmental atmosphere
	if sb.Atmosphere != "" {
		parts = append(parts, fmt.Sprintf("Atmosphere: %s", sb.Atmosphere))
	}

	// 7. Emotion and result
	if sb.Emotion != "" {
		parts = append(parts, fmt.Sprintf("Mood: %s", sb.Emotion))
	}
	if sb.Result != "" {
		parts = append(parts, fmt.Sprintf("Result: %s", sb.Result))
	}

	// 8. Audio elements
	if sb.BgmPrompt != "" {
		parts = append(parts, fmt.Sprintf("BGM: %s", sb.BgmPrompt))
	}
	if sb.SoundEffect != "" {
		parts = append(parts, fmt.Sprintf("Sound effects: %s", sb.SoundEffect))
	}

	// 9. Video ratio
	parts = append(parts, fmt.Sprintf("=VideoRatio: %s", videoRatio))
	if len(parts) > 0 {
		return strings.Join(parts, ". ")
	}
	return "Anime style video scene"
}

func (s *StoryboardService) saveStoryboards(episodeID string, storyboards []Storyboard) error {
	// Validate episodeID
	epID, err := strconv.ParseUint(episodeID, 10, 32)
	if err != nil {
		s.log.Errorw("Invalid episode ID", "episode_id", episodeID, "error", err)
		return fmt.Errorf("invalid episode ID: %s", episodeID)
	}

	// Defensive check: if AI returned 0 storyboard shots, don't delete old ones
	if len(storyboards) == 0 {
		s.log.Errorw("AI returned 0 storyboard shots, refusing to save to avoid deleting existing storyboards", "episode_id", episodeID)
		return fmt.Errorf("AI storyboard generation failed: returned 0 storyboard shots")
	}

	s.log.Infow("Starting to save storyboard shots",
		"episode_id", episodeID,
		"episode_id_uint", uint(epID),
		"storyboard_count", len(storyboards))

	// Begin transaction
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Validate that the episode exists
		var episode models.Episode
		if err := tx.First(&episode, epID).Error; err != nil {
			s.log.Errorw("Episode not found", "episode_id", episodeID, "error", err)
			return fmt.Errorf("episode not found: %s", episodeID)
		}

		s.log.Infow("Found episode information",
			"episode_id", episode.ID,
			"episode_number", episode.EpisodeNum,
			"drama_id", episode.DramaID,
			"title", episode.Title)

		// Get all storyboard IDs for this episode (using uint type)
		var storyboardIDs []uint
		if err := tx.Model(&models.Storyboard{}).
			Where("episode_id = ?", uint(epID)).
			Pluck("id", &storyboardIDs).Error; err != nil {
			return err
		}

		s.log.Infow("Found existing storyboards",
			"episode_id_string", episodeID,
			"episode_id_uint", uint(epID),
			"existing_storyboard_count", len(storyboardIDs),
			"storyboard_ids", storyboardIDs)

		// If there are storyboards, first clear the storyboard_id in associated image_generations
		if len(storyboardIDs) > 0 {
			if err := tx.Model(&models.ImageGeneration{}).
				Where("storyboard_id IN ?", storyboardIDs).
				Update("storyboard_id", nil).Error; err != nil {
				return err
			}
			s.log.Infow("Cleared associated image generation records", "count", len(storyboardIDs))
		}

		// Delete existing storyboard shots for this episode (using uint type to ensure type match)
		s.log.Warnw("Preparing to delete storyboard data",
			"episode_id_string", episodeID,
			"episode_id_uint", uint(epID),
			"episode_id_from_db", episode.ID,
			"will_delete_count", len(storyboardIDs))

		result := tx.Where("episode_id = ?", uint(epID)).Delete(&models.Storyboard{})
		if result.Error != nil {
			s.log.Errorw("Failed to delete old storyboards", "episode_id", uint(epID), "error", result.Error)
			return result.Error
		}

		s.log.Infow("Deleted old storyboard shots",
			"episode_id", uint(epID),
			"deleted_count", result.RowsAffected)

		// Note: Do not delete scenes, as they are extracted before storyboard breakdown
		// AI returns scene_id directly, no string matching needed here

		// Save new storyboard shots
		for _, sb := range storyboards {
			// Build description info, including dialogue
			description := fmt.Sprintf("【Shot Type】%s\n【Movement】%s\n【Action】%s\n【Dialogue】%s\n【Result】%s\n【Emotion】%s",
				sb.ShotType, sb.Movement, sb.Action, sb.Dialogue, sb.Result, sb.Emotion)

			// Generate two specialized prompts
			imagePrompt := s.generateImagePrompt(sb) // For image generation
			videoPrompt := s.generateVideoPrompt(sb) // For video generation

			// Handle dialogue field
			var dialoguePtr *string
			if sb.Dialogue != "" {
				dialoguePtr = &sb.Dialogue
			}

			// Use SceneID returned directly by AI
			if sb.SceneID != nil {
				s.log.Infow("Background ID from AI",
					"shot_number", sb.ShotNumber,
					"scene_id", *sb.SceneID)
			}

			// Handle title field
			var titlePtr *string
			if sb.Title != "" {
				titlePtr = &sb.Title
			}

			// Handle shot_type, angle, movement fields
			var shotTypePtr, anglePtr, movementPtr *string
			if sb.ShotType != "" {
				shotTypePtr = &sb.ShotType
			}
			if sb.Angle != "" {
				anglePtr = &sb.Angle
			}
			if sb.Movement != "" {
				movementPtr = &sb.Movement
			}

			// Handle bgm_prompt, sound_effect fields
			var bgmPromptPtr, soundEffectPtr *string
			if sb.BgmPrompt != "" {
				bgmPromptPtr = &sb.BgmPrompt
			}
			if sb.SoundEffect != "" {
				soundEffectPtr = &sb.SoundEffect
			}

			// Handle result, atmosphere fields
			var resultPtr, atmospherePtr *string
			if sb.Result != "" {
				resultPtr = &sb.Result
			}
			if sb.Atmosphere != "" {
				atmospherePtr = &sb.Atmosphere
			}

			scene := models.Storyboard{
				EpisodeID:        uint(epID),
				SceneID:          sb.SceneID,
				StoryboardNumber: sb.ShotNumber,
				Title:            titlePtr,
				Location:         &sb.Location,
				Time:             &sb.Time,
				ShotType:         shotTypePtr,
				Angle:            anglePtr,
				Movement:         movementPtr,
				Description:      &description,
				Action:           &sb.Action,
				Result:           resultPtr,
				Atmosphere:       atmospherePtr,
				Dialogue:         dialoguePtr,
				ImagePrompt:      &imagePrompt,
				VideoPrompt:      &videoPrompt,
				BgmPrompt:        bgmPromptPtr,
				SoundEffect:      soundEffectPtr,
				Duration:         sb.Duration,
			}

			if err := tx.Create(&scene).Error; err != nil {
				s.log.Errorw("Failed to create scene", "error", err, "shot_number", sb.ShotNumber)
				return err
			}

			// Associate characters
			if len(sb.Characters) > 0 {
				var characters []models.Character
				if err := tx.Where("id IN ?", sb.Characters).Find(&characters).Error; err != nil {
					s.log.Warnw("Failed to load characters for association", "error", err, "character_ids", sb.Characters)
				} else if len(characters) > 0 {
					if err := tx.Model(&scene).Association("Characters").Append(characters); err != nil {
						s.log.Warnw("Failed to associate characters", "error", err, "shot_number", sb.ShotNumber)
					} else {
						s.log.Infow("Characters associated successfully",
							"shot_number", sb.ShotNumber,
							"character_ids", sb.Characters,
							"count", len(characters))
					}
				}
			}
		}

		s.log.Infow("Storyboards saved successfully", "episode_id", episodeID, "count", len(storyboards))
		return nil
	})
}

// CreateStoryboardRequest represents a request to create a storyboard
type CreateStoryboardRequest struct {
	EpisodeID        uint    `json:"episode_id"`
	SceneID          *uint   `json:"scene_id"`
	StoryboardNumber int     `json:"storyboard_number"`
	Title            *string `json:"title"`
	Location         *string `json:"location"`
	Time             *string `json:"time"`
	ShotType         *string `json:"shot_type"`
	Angle            *string `json:"angle"`
	Movement         *string `json:"movement"`
	Description      *string `json:"description"`
	Action           *string `json:"action"`
	Result           *string `json:"result"`
	Atmosphere       *string `json:"atmosphere"`
	Dialogue         *string `json:"dialogue"`
	BgmPrompt        *string `json:"bgm_prompt"`
	SoundEffect      *string `json:"sound_effect"`
	Duration         int     `json:"duration"`
	Characters       []uint  `json:"characters"`
}

// CreateStoryboard creates a single storyboard shot
func (s *StoryboardService) CreateStoryboard(req *CreateStoryboardRequest) (*models.Storyboard, error) {
	// Build Storyboard object
	sb := Storyboard{
		ShotNumber:  req.StoryboardNumber,
		ShotType:    getString(req.ShotType),
		Angle:       getString(req.Angle),
		Time:        getString(req.Time),
		Location:    getString(req.Location),
		SceneID:     req.SceneID,
		Movement:    getString(req.Movement),
		Action:      getString(req.Action),
		Dialogue:    getString(req.Dialogue),
		Result:      getString(req.Result),
		Atmosphere:  getString(req.Atmosphere),
		Emotion:     "", // Can be added later
		Duration:    req.Duration,
		BgmPrompt:   getString(req.BgmPrompt),
		SoundEffect: getString(req.SoundEffect),
		Characters:  req.Characters,
	}
	if req.Title != nil {
		sb.Title = *req.Title
	}

	// Generate prompts
	imagePrompt := s.generateImagePrompt(sb)
	videoPrompt := s.generateVideoPrompt(sb)

	// Build description
	desc := ""
	if req.Description != nil {
		desc = *req.Description
	}

	modelSB := &models.Storyboard{
		EpisodeID:        req.EpisodeID,
		SceneID:          req.SceneID,
		StoryboardNumber: req.StoryboardNumber,
		Title:            req.Title,
		Location:         req.Location,
		Time:             req.Time,
		ShotType:         req.ShotType,
		Angle:            req.Angle,
		Movement:         req.Movement,
		Description:      &desc,
		Action:           req.Action,
		Result:           req.Result,
		Atmosphere:       req.Atmosphere,
		Dialogue:         req.Dialogue,
		ImagePrompt:      &imagePrompt,
		VideoPrompt:      &videoPrompt,
		BgmPrompt:        req.BgmPrompt,
		SoundEffect:      req.SoundEffect,
		Duration:         req.Duration,
	}

	if err := s.db.Create(modelSB).Error; err != nil {
		return nil, fmt.Errorf("failed to create storyboard: %w", err)
	}

	// Associate characters
	if len(req.Characters) > 0 {
		var characters []models.Character
		if err := s.db.Where("id IN ?", req.Characters).Find(&characters).Error; err != nil {
			s.log.Warnw("Failed to find characters for new storyboard", "error", err)
		} else if len(characters) > 0 {
			s.db.Model(modelSB).Association("Characters").Append(characters)
		}
	}

	s.log.Infow("Storyboard created", "id", modelSB.ID, "episode_id", req.EpisodeID)
	return modelSB, nil
}

// DeleteStoryboard deletes a storyboard shot
func (s *StoryboardService) DeleteStoryboard(storyboardID uint) error {
	result := s.db.Where("id = ? ", storyboardID).Delete(&models.Storyboard{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("storyboard not found")
	}
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func getString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
