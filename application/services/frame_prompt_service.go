package services

import (
"fmt"
"strings"

"github.com/drama-generator/backend/domain/models"
"github.com/drama-generator/backend/pkg/config"
"github.com/drama-generator/backend/pkg/logger"
"gorm.io/gorm"
)

// FramePromptService handles frame prompt generation
type FramePromptService struct {
db          *gorm.DB
aiService   *AIService
log         *logger.Logger
config      *config.Config
promptI18n  *PromptI18n
taskService *TaskService
}

// NewFramePromptService creates a frame prompt service
func NewFramePromptService(db *gorm.DB, cfg *config.Config, log *logger.Logger) *FramePromptService {
return &FramePromptService{
db:          db,
aiService:   NewAIService(db, log),
log:         log,
config:      cfg,
promptI18n:  NewPromptI18n(cfg),
taskService: NewTaskService(db, log),
}
}

// FrameType represents a frame type
type FrameType string

const (
FrameTypeFirst  FrameType = "first"  // First frame
FrameTypeKey    FrameType = "key"    // Key frame
FrameTypeLast   FrameType = "last"   // Last frame
FrameTypePanel  FrameType = "panel"  // Panel board (3-grid combo)
FrameTypeAction FrameType = "action" // Action sequence (5-grid)
)

// GenerateFramePromptRequest represents a request to generate frame prompts
type GenerateFramePromptRequest struct {
StoryboardID string    `json:"storyboard_id"`
FrameType    FrameType `json:"frame_type"`
// Optional parameters
PanelCount int `json:"panel_count,omitempty"` // Panel grid count, default 3
}

// FramePromptResponse represents a frame prompt response
type FramePromptResponse struct {
FrameType   FrameType          `json:"frame_type"`
SingleFrame *SingleFramePrompt `json:"single_frame,omitempty"` // Single frame prompt
MultiFrame  *MultiFramePrompt  `json:"multi_frame,omitempty"`  // Multi frame prompt
}

// SingleFramePrompt represents a single frame prompt
type SingleFramePrompt struct {
Prompt string `json:"prompt"`
}

// MultiFramePrompt represents a multi-frame prompt
type MultiFramePrompt struct {
Layout string              `json:"layout"` // horizontal_3, grid_2x2 etc
Frames []SingleFramePrompt `json:"frames"`
}

// GenerateFramePrompt generates a frame prompt of the specified type and saves it to the frame_prompts table
func (s *FramePromptService) GenerateFramePrompt(req GenerateFramePromptRequest, model string) (string, error) {
// Query storyboard information
var storyboard models.Storyboard
if err := s.db.Preload("Characters").First(&storyboard, req.StoryboardID).Error; err != nil {
return "", fmt.Errorf("storyboard not found: %w", err)
}

// Create task
task, err := s.taskService.CreateTask("frame_prompt_generation", req.StoryboardID)
if err != nil {
s.log.Errorw("Failed to create frame prompt generation task", "error", err, "storyboard_id", req.StoryboardID)
return "", fmt.Errorf("failed to create task: %w", err)
}

// Asynchronously process frame prompt generation
go s.processFramePromptGeneration(task.ID, req, model)

s.log.Infow("Frame prompt generation task created", "task_id", task.ID, "storyboard_id", req.StoryboardID, "frame_type", req.FrameType)
return task.ID, nil
}

// processFramePromptGeneration asynchronously processes frame prompt generation
func (s *FramePromptService) processFramePromptGeneration(taskID string, req GenerateFramePromptRequest, model string) {
// Update task status to processing
s.taskService.UpdateTaskStatus(taskID, "processing", 0, "Generating frame prompts...")

// Query storyboard information
var storyboard models.Storyboard
if err := s.db.Preload("Characters").First(&storyboard, req.StoryboardID).Error; err != nil {
s.log.Errorw("Storyboard not found during frame prompt generation", "error", err, "storyboard_id", req.StoryboardID)
s.taskService.UpdateTaskStatus(taskID, "failed", 0, "Storyboard not found")
return
}

// Get scene information
var scene *models.Scene
if storyboard.SceneID != nil {
scene = &models.Scene{}
if err := s.db.First(scene, *storyboard.SceneID).Error; err != nil {
s.log.Warnw("Scene not found during frame prompt generation", "scene_id", *storyboard.SceneID, "task_id", taskID)
scene = nil
}
}

// Get drama style and aspect ratio
var episode models.Episode
if err := s.db.Preload("Drama").First(&episode, storyboard.EpisodeID).Error; err != nil {
s.log.Warnw("Failed to load episode and drama", "error", err, "episode_id", storyboard.EpisodeID)
}
dramaStyle := episode.Drama.Style
aspectRatio := episode.Drama.AspectRatio
if aspectRatio == "" {
aspectRatio = "16:9"
}

response := &FramePromptResponse{
FrameType: req.FrameType,
}

// Generate prompts
switch req.FrameType {
case FrameTypeFirst:
response.SingleFrame = s.generateFirstFrame(storyboard, scene, dramaStyle, aspectRatio, model)
s.saveFramePrompt(req.StoryboardID, string(req.FrameType), response.SingleFrame.Prompt, "")
case FrameTypeKey:
response.SingleFrame = s.generateKeyFrame(storyboard, scene, dramaStyle, aspectRatio, model)
s.saveFramePrompt(req.StoryboardID, string(req.FrameType), response.SingleFrame.Prompt, "")
case FrameTypeLast:
response.SingleFrame = s.generateLastFrame(storyboard, scene, dramaStyle, aspectRatio, model)
s.saveFramePrompt(req.StoryboardID, string(req.FrameType), response.SingleFrame.Prompt, "")
case FrameTypePanel:
count := req.PanelCount
if count == 0 {
count = 3
}
response.MultiFrame = s.generatePanelFrames(storyboard, scene, count, dramaStyle, aspectRatio, model)
var prompts []string
for _, frame := range response.MultiFrame.Frames {
prompts = append(prompts, frame.Prompt)
}
combinedPrompt := strings.Join(prompts, "\n---\n")
s.saveFramePrompt(req.StoryboardID, string(req.FrameType), combinedPrompt, response.MultiFrame.Layout)
case FrameTypeAction:
response.MultiFrame = s.generateActionSequence(storyboard, scene, dramaStyle, aspectRatio, model)
var prompts []string
for _, frame := range response.MultiFrame.Frames {
prompts = append(prompts, frame.Prompt)
}
combinedPrompt := strings.Join(prompts, "\n---\n")
s.saveFramePrompt(req.StoryboardID, string(req.FrameType), combinedPrompt, response.MultiFrame.Layout)
default:
s.log.Errorw("Unsupported frame type during frame prompt generation", "frame_type", req.FrameType, "task_id", taskID)
s.taskService.UpdateTaskStatus(taskID, "failed", 0, "Unsupported frame type")
return
}

// Update task status to completed
s.taskService.UpdateTaskResult(taskID, map[string]interface{}{
"response":      response,
"storyboard_id": req.StoryboardID,
"frame_type":    string(req.FrameType),
})

s.log.Infow("Frame prompt generation completed", "task_id", taskID, "storyboard_id", req.StoryboardID, "frame_type", req.FrameType)
}

// saveFramePrompt saves a frame prompt to the database
func (s *FramePromptService) saveFramePrompt(storyboardID, frameType, prompt, layout string) {
framePrompt := models.FramePrompt{
StoryboardID: uint(mustParseUint(storyboardID)),
FrameType:    frameType,
Prompt:       prompt,
}

if layout != "" {
framePrompt.Layout = &layout
}

// Delete old records of the same type (keep latest)
s.db.Where("storyboard_id = ? AND frame_type = ?", storyboardID, frameType).Delete(&models.FramePrompt{})

// Insert new record
if err := s.db.Create(&framePrompt).Error; err != nil {
s.log.Warnw("Failed to save frame prompt", "error", err, "storyboard_id", storyboardID, "frame_type", frameType)
}
}

// mustParseUint helper function
func mustParseUint(s string) uint64 {
var result uint64
fmt.Sscanf(s, "%d", &result)
return result
}

// generateFirstFrame generates first frame prompt
func (s *FramePromptService) generateFirstFrame(sb models.Storyboard, scene *models.Scene, dramaStyle, aspectRatio, model string) *SingleFramePrompt {
// Build context information
contextInfo := s.buildStoryboardContext(sb, scene)

// Use i18n prompts
systemPrompt := s.promptI18n.GetFirstFramePrompt(dramaStyle, aspectRatio)
userPrompt := s.promptI18n.FormatUserPrompt("frame_info", contextInfo)

// Call AI generation (use specified model if provided)
var aiResponse string
var err error
if model != "" {
client, getErr := s.aiService.GetAIClientForModel("text", model)
if getErr != nil {
s.log.Warnw("Failed to get client for specified model, using default", "model", model, "error", getErr)
aiResponse, err = s.aiService.GenerateText(userPrompt, systemPrompt)
} else {
aiResponse, err = client.GenerateText(userPrompt, systemPrompt)
}
} else {
aiResponse, err = s.aiService.GenerateText(userPrompt, systemPrompt)
}
if err != nil {
s.log.Warnw("AI generation failed, using fallback", "error", err)
// Fallback: use simple concatenation
fallbackPrompt := s.buildFallbackPrompt(sb, scene, "first frame, static shot")
return &SingleFramePrompt{Prompt: fallbackPrompt}
}

// Parse AI-returned JSON
result := s.parseFramePromptJSON(aiResponse)
if result == nil {
// JSON parsing failed, use fallback
s.log.Warnw("Failed to parse AI JSON response, using fallback", "storyboard_id", sb.ID, "response", aiResponse)
fallbackPrompt := s.buildFallbackPrompt(sb, scene, "first frame, static shot")
return &SingleFramePrompt{Prompt: fallbackPrompt}
}

return result
}

// generateKeyFrame generates key frame prompt
func (s *FramePromptService) generateKeyFrame(sb models.Storyboard, scene *models.Scene, dramaStyle, aspectRatio, model string) *SingleFramePrompt {
// Build context information
contextInfo := s.buildStoryboardContext(sb, scene)

// Use i18n prompts
systemPrompt := s.promptI18n.GetKeyFramePrompt(dramaStyle, aspectRatio)
userPrompt := s.promptI18n.FormatUserPrompt("key_frame_info", contextInfo)

// Call AI generation (use specified model if provided)
var aiResponse string
var err error
if model != "" {
client, getErr := s.aiService.GetAIClientForModel("text", model)
if getErr != nil {
s.log.Warnw("Failed to get client for specified model, using default", "model", model, "error", getErr)
aiResponse, err = s.aiService.GenerateText(userPrompt, systemPrompt)
} else {
aiResponse, err = client.GenerateText(userPrompt, systemPrompt)
}
} else {
aiResponse, err = s.aiService.GenerateText(userPrompt, systemPrompt)
}
if err != nil {
s.log.Warnw("AI generation failed, using fallback", "error", err)
fallbackPrompt := s.buildFallbackPrompt(sb, scene, "key frame, dynamic action")
return &SingleFramePrompt{Prompt: fallbackPrompt}
}

// Parse AI-returned JSON
result := s.parseFramePromptJSON(aiResponse)
if result == nil {
// JSON parsing failed, use fallback
s.log.Warnw("Failed to parse AI JSON response, using fallback", "storyboard_id", sb.ID, "response", aiResponse)
fallbackPrompt := s.buildFallbackPrompt(sb, scene, "key frame, dynamic action")
return &SingleFramePrompt{Prompt: fallbackPrompt}
}

return result
}

// generateLastFrame generates last frame prompt
func (s *FramePromptService) generateLastFrame(sb models.Storyboard, scene *models.Scene, dramaStyle, aspectRatio, model string) *SingleFramePrompt {
// Build context information
contextInfo := s.buildStoryboardContext(sb, scene)

// Use i18n prompts
systemPrompt := s.promptI18n.GetLastFramePrompt(dramaStyle, aspectRatio)
userPrompt := s.promptI18n.FormatUserPrompt("last_frame_info", contextInfo)

// Call AI generation (use specified model if provided)
var aiResponse string
var err error
if model != "" {
client, getErr := s.aiService.GetAIClientForModel("text", model)
if getErr != nil {
s.log.Warnw("Failed to get client for specified model, using default", "model", model, "error", getErr)
aiResponse, err = s.aiService.GenerateText(userPrompt, systemPrompt)
} else {
aiResponse, err = client.GenerateText(userPrompt, systemPrompt)
}
} else {
aiResponse, err = s.aiService.GenerateText(userPrompt, systemPrompt)
}
if err != nil {
s.log.Warnw("AI generation failed, using fallback", "error", err)
fallbackPrompt := s.buildFallbackPrompt(sb, scene, "last frame, final state")
return &SingleFramePrompt{Prompt: fallbackPrompt}
}

// Parse AI-returned JSON
result := s.parseFramePromptJSON(aiResponse)
if result == nil {
// JSON parsing failed, use fallback
s.log.Warnw("Failed to parse AI JSON response, using fallback", "storyboard_id", sb.ID, "response", aiResponse)
fallbackPrompt := s.buildFallbackPrompt(sb, scene, "last frame, final state")
return &SingleFramePrompt{Prompt: fallbackPrompt}
}

return result
}

// generatePanelFrames generates panel board prompts (multi-grid combo)
func (s *FramePromptService) generatePanelFrames(sb models.Storyboard, scene *models.Scene, count int, dramaStyle, aspectRatio, model string) *MultiFramePrompt {
layout := fmt.Sprintf("horizontal_%d", count)

frames := make([]SingleFramePrompt, count)

// Fixed generation: first frame -> key frame -> last frame
if count == 3 {
frames[0] = *s.generateFirstFrame(sb, scene, dramaStyle, aspectRatio, model)
frames[1] = *s.generateKeyFrame(sb, scene, dramaStyle, aspectRatio, model)
frames[2] = *s.generateLastFrame(sb, scene, dramaStyle, aspectRatio, model)
} else if count == 4 {
// 4 grids: first frame -> middle frame 1 -> middle frame 2 -> last frame
frames[0] = *s.generateFirstFrame(sb, scene, dramaStyle, aspectRatio, model)
frames[1] = *s.generateKeyFrame(sb, scene, dramaStyle, aspectRatio, model)
frames[2] = *s.generateKeyFrame(sb, scene, dramaStyle, aspectRatio, model)
frames[3] = *s.generateLastFrame(sb, scene, dramaStyle, aspectRatio, model)
}

return &MultiFramePrompt{
Layout: layout,
Frames: frames,
}
}

// generateActionSequence generates action sequence prompts (3x3 grid)
func (s *FramePromptService) generateActionSequence(sb models.Storyboard, scene *models.Scene, dramaStyle, aspectRatio, model string) *MultiFramePrompt {
// Build context information
contextInfo := s.buildStoryboardContext(sb, scene)

// Use i18n prompts - specifically designed for action sequences
systemPrompt := s.promptI18n.GetActionSequenceFramePrompt(dramaStyle, aspectRatio)
userPrompt := s.promptI18n.FormatUserPrompt("frame_info", contextInfo)

// Call AI generation (use specified model if provided)
var aiResponse string
var err error
if model != "" {
client, getErr := s.aiService.GetAIClientForModel("text", model)
if getErr != nil {
s.log.Warnw("Failed to get client for specified model, using default", "model", model, "error", getErr)
aiResponse, err = s.aiService.GenerateText(userPrompt, systemPrompt)
} else {
aiResponse, err = client.GenerateText(userPrompt, systemPrompt)
}
} else {
aiResponse, err = s.aiService.GenerateText(userPrompt, systemPrompt)
}

if err != nil {
s.log.Warnw("AI generation failed for action sequence, using fallback", "error", err)
// Fallback: use simple concatenation
fallbackPrompt := s.buildFallbackPrompt(sb, scene, "3x3 storyboard grid action sequence, character consistency, continuous movement progression")
return &MultiFramePrompt{
Layout: "grid_3x3",
Frames: []SingleFramePrompt{{Prompt: fallbackPrompt}},
}
}

// Parse AI-returned JSON
result := s.parseFramePromptJSON(aiResponse)
if result == nil {
// JSON parsing failed, use fallback
s.log.Warnw("Failed to parse AI JSON response for action sequence, using fallback", "storyboard_id", sb.ID, "response", aiResponse)
fallbackPrompt := s.buildFallbackPrompt(sb, scene, "3x3 storyboard grid action sequence, character consistency, continuous movement progression")
return &MultiFramePrompt{
Layout: "grid_3x3",
Frames: []SingleFramePrompt{{Prompt: fallbackPrompt}},
}
}

// Action sequence is a single 3x3 grid image, so only one prompt is returned
return &MultiFramePrompt{
Layout: "grid_3x3",
Frames: []SingleFramePrompt{*result},
}
}

// buildStoryboardContext builds shot context information
func (s *FramePromptService) buildStoryboardContext(sb models.Storyboard, scene *models.Scene) string {
var parts []string

// Shot description (most important)
if sb.Description != nil && *sb.Description != "" {
parts = append(parts, s.promptI18n.FormatUserPrompt("shot_description_label", *sb.Description))
}

// Scene information
if scene != nil {
parts = append(parts, s.promptI18n.FormatUserPrompt("scene_label", scene.Location, scene.Time))
} else if sb.Location != nil && sb.Time != nil {
parts = append(parts, s.promptI18n.FormatUserPrompt("scene_label", *sb.Location, *sb.Time))
}

// Characters
if len(sb.Characters) > 0 {
var charNames []string
for _, char := range sb.Characters {
charNames = append(charNames, char.Name)
}
parts = append(parts, s.promptI18n.FormatUserPrompt("characters_label", strings.Join(charNames, ", ")))
}

// Action
if sb.Action != nil && *sb.Action != "" {
parts = append(parts, s.promptI18n.FormatUserPrompt("action_label", *sb.Action))
}

// Result
if sb.Result != nil && *sb.Result != "" {
parts = append(parts, s.promptI18n.FormatUserPrompt("result_label", *sb.Result))
}

// Dialogue
if sb.Dialogue != nil && *sb.Dialogue != "" {
parts = append(parts, s.promptI18n.FormatUserPrompt("dialogue_label", *sb.Dialogue))
}

// Atmosphere
if sb.Atmosphere != nil && *sb.Atmosphere != "" {
parts = append(parts, s.promptI18n.FormatUserPrompt("atmosphere_label", *sb.Atmosphere))
}

// Shot parameters
if sb.ShotType != nil {
parts = append(parts, s.promptI18n.FormatUserPrompt("shot_type_label", *sb.ShotType))
}
if sb.Angle != nil {
parts = append(parts, s.promptI18n.FormatUserPrompt("angle_label", *sb.Angle))
}
if sb.Movement != nil {
parts = append(parts, s.promptI18n.FormatUserPrompt("movement_label", *sb.Movement))
}

return strings.Join(parts, "\n")
}

// buildFallbackPrompt builds fallback prompt (used when AI fails)
func (s *FramePromptService) buildFallbackPrompt(sb models.Storyboard, scene *models.Scene, suffix string) string {
var parts []string

// Scene
if scene != nil {
parts = append(parts, fmt.Sprintf("%s, %s", scene.Location, scene.Time))
}

// Characters
if len(sb.Characters) > 0 {
for _, char := range sb.Characters {
parts = append(parts, char.Name)
}
}

// Atmosphere
if sb.Atmosphere != nil {
parts = append(parts, *sb.Atmosphere)
}

parts = append(parts, "anime style", suffix)
return strings.Join(parts, ", ")
}
