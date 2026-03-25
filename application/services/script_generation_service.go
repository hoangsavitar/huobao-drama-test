package services

import (
"fmt"
"strconv"

"github.com/drama-generator/backend/domain/models"
"github.com/drama-generator/backend/pkg/ai"
"github.com/drama-generator/backend/pkg/config"
"github.com/drama-generator/backend/pkg/logger"
"github.com/drama-generator/backend/pkg/utils"
"gorm.io/gorm"
)

type ScriptGenerationService struct {
db          *gorm.DB
aiService   *AIService
log         *logger.Logger
config      *config.Config
promptI18n  *PromptI18n
taskService *TaskService
}

func NewScriptGenerationService(db *gorm.DB, cfg *config.Config, log *logger.Logger) *ScriptGenerationService {
return &ScriptGenerationService{
db:          db,
aiService:   NewAIService(db, log),
log:         log,
config:      cfg,
promptI18n:  NewPromptI18n(cfg),
taskService: NewTaskService(db, log),
}
}

type GenerateCharactersRequest struct {
DramaID     string  `json:"drama_id" binding:"required"`
EpisodeID   uint    `json:"episode_id"`
Outline     string  `json:"outline"`
Count       int     `json:"count"`
Temperature float64 `json:"temperature"`
Model       string  `json:"model"` // specify the text model to use
}

func (s *ScriptGenerationService) GenerateCharacters(req *GenerateCharactersRequest) (string, error) {
var drama models.Drama
if err := s.db.Where("id = ? ", req.DramaID).First(&drama).Error; err != nil {
return "", fmt.Errorf("drama not found")
}

// Create task
task, err := s.taskService.CreateTask("character_generation", req.DramaID)
if err != nil {
s.log.Errorw("Failed to create character generation task", "error", err)
return "", fmt.Errorf("failed to create task: %w", err)
}

// Process character generation asynchronously
go s.processCharacterGeneration(task.ID, req)

s.log.Infow("Character generation task created", "task_id", task.ID, "drama_id", req.DramaID)
return task.ID, nil
}

// processCharacterGeneration processes character generation asynchronously
func (s *ScriptGenerationService) processCharacterGeneration(taskID string, req *GenerateCharactersRequest) {
// Update task status to processing
s.taskService.UpdateTaskStatus(taskID, "processing", 0, "Generating characters...")

count := req.Count
if count == 0 {
count = 5
}

// Get drama's style information
var drama models.Drama
if err := s.db.Where("id = ? ", req.DramaID).First(&drama).Error; err != nil {
s.log.Errorw("Drama not found during character generation", "error", err, "drama_id", req.DramaID)
s.taskService.UpdateTaskStatus(taskID, "failed", 0, "drama information not found")
return
}

systemPrompt := s.promptI18n.GetCharacterExtractionPrompt(drama.Style, drama.AspectRatio)

outlineText := req.Outline
if outlineText == "" {
outlineText = s.promptI18n.FormatUserPrompt("drama_info_template", drama.Title, drama.Description, drama.Genre)
}

userPrompt := s.promptI18n.FormatUserPrompt("character_request", outlineText, count)

temperature := req.Temperature
if temperature == 0 {
temperature = 0.7
}

// If model is specified, use it; otherwise use default config
var text string
var err error
if req.Model != "" {
s.log.Infow("Using specified model for character generation", "model", req.Model, "task_id", taskID)
client, getErr := s.aiService.GetAIClientForModel("text", req.Model)
if getErr != nil {
s.log.Warnw("Failed to get client for specified model, using default", "model", req.Model, "error", getErr, "task_id", taskID)
text, err = s.aiService.GenerateText(userPrompt, systemPrompt, ai.WithTemperature(temperature))
} else {
text, err = client.GenerateText(userPrompt, systemPrompt, ai.WithTemperature(temperature))
}
} else {
text, err = s.aiService.GenerateText(userPrompt, systemPrompt, ai.WithTemperature(temperature))
}

if err != nil {
s.log.Errorw("Failed to generate characters", "error", err, "task_id", taskID)
s.taskService.UpdateTaskStatus(taskID, "failed", 0, "AI generation failed: "+err.Error())
return
}

s.log.Infow("AI response received for character generation", "length", len(text), "preview", text[:minInt(200, len(text))], "task_id", taskID)

// AI directly returns array format
var result []struct {
Name        string `json:"name"`
Role        string `json:"role"`
Description string `json:"description"`
Personality string `json:"personality"`
Appearance  string `json:"appearance"`
VoiceStyle  string `json:"voice_style"`
}

if err := utils.SafeParseAIJSON(text, &result); err != nil {
s.log.Errorw("Failed to parse characters JSON", "error", err, "raw_response", text[:minInt(500, len(text))], "task_id", taskID)
s.taskService.UpdateTaskStatus(taskID, "failed", 0, "failed to parse AI response")
return
}

var characters []models.Character
for _, char := range result {
// Check if character already exists
var existingChar models.Character
err := s.db.Where("drama_id = ? AND name = ?", req.DramaID, char.Name).First(&existingChar).Error
if err == nil {
// Character already exists, use existing one without overwriting
s.log.Infow("Character already exists, skipping", "drama_id", req.DramaID, "name", char.Name, "task_id", taskID)
characters = append(characters, existingChar)
continue
}

// Character does not exist, create new character
dramaID, _ := strconv.ParseUint(req.DramaID, 10, 32)
character := models.Character{
DramaID:     uint(dramaID),
Name:        char.Name,
Role:        &char.Role,
Description: &char.Description,
Personality: &char.Personality,
Appearance:  &char.Appearance,
VoiceStyle:  &char.VoiceStyle,
}

if err := s.db.Create(&character).Error; err != nil {
s.log.Errorw("Failed to create character", "error", err, "task_id", taskID)
continue
}

characters = append(characters, character)
}

// If EpisodeID is provided, establish episode_characters association
if req.EpisodeID > 0 {
var episode models.Episode
if err := s.db.First(&episode, req.EpisodeID).Error; err == nil {
// Use GORM Association to establish many-to-many association
if err := s.db.Model(&episode).Association("Characters").Append(characters); err != nil {
s.log.Errorw("Failed to associate characters with episode", "error", err, "episode_id", req.EpisodeID, "task_id", taskID)
} else {
s.log.Infow("Characters associated with episode", "episode_id", req.EpisodeID, "character_count", len(characters), "task_id", taskID)
}
} else {
s.log.Errorw("Episode not found for association", "episode_id", req.EpisodeID, "error", err, "task_id", taskID)
}
}

// Update task status to completed
resultData := map[string]interface{}{
"characters": characters,
"count":      len(characters),
}
s.taskService.UpdateTaskResult(taskID, resultData)

s.log.Infow("Character generation completed", "task_id", taskID, "drama_id", req.DramaID, "character_count", len(characters))
}

// GenerateScenesForEpisode is deprecated, use StoryboardService.GenerateStoryboard instead
// ParseScript is deprecated, use GenerateCharacters instead

// minInt returns the smaller of two integers
func minInt(a, b int) int {
if a < b {
return a
}
return b
}
