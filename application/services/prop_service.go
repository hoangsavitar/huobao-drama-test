package services

import (
"fmt"
"time"

// Added missing import
models "github.com/drama-generator/backend/domain/models"
"github.com/drama-generator/backend/pkg/ai"
"github.com/drama-generator/backend/pkg/config"
"github.com/drama-generator/backend/pkg/logger"
"github.com/drama-generator/backend/pkg/utils"
"gorm.io/gorm"
)

type PropService struct {
db                     *gorm.DB
aiService              *AIService
taskService            *TaskService
imageGenerationService *ImageGenerationService
log                    *logger.Logger
config                 *config.Config
promptI18n             *PromptI18n
}

func NewPropService(db *gorm.DB, aiService *AIService, taskService *TaskService, imageGenerationService *ImageGenerationService, log *logger.Logger, cfg *config.Config) *PropService {
return &PropService{
db:                     db,
aiService:              aiService,
taskService:            taskService,
imageGenerationService: imageGenerationService,
log:                    log,
config:                 cfg,
promptI18n:             NewPromptI18n(cfg),
}
}

// ListProps retrieves the prop list for a drama
func (s *PropService) ListProps(dramaID uint) ([]models.Prop, error) {
var props []models.Prop
if err := s.db.Where("drama_id = ?", dramaID).Find(&props).Error; err != nil {
return nil, err
}
return props, nil
}

// CreateProp creates a prop
func (s *PropService) CreateProp(prop *models.Prop) error {
return s.db.Create(prop).Error
}

// UpdateProp updates a prop
func (s *PropService) UpdateProp(id uint, updates map[string]interface{}) error {
return s.db.Model(&models.Prop{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteProp deletes a prop
func (s *PropService) DeleteProp(id uint) error {
return s.db.Delete(&models.Prop{}, id).Error
}

// ExtractPropsFromScript extracts props from script (async)
func (s *PropService) ExtractPropsFromScript(episodeID uint) (string, error) {
var episode models.Episode
if err := s.db.First(&episode, episodeID).Error; err != nil {
return "", fmt.Errorf("episode not found: %w", err)
}

task, err := s.taskService.CreateTask("prop_extraction", fmt.Sprintf("%d", episodeID))
if err != nil {
return "", err
}

go s.processPropExtraction(task.ID, episode)

return task.ID, nil
}

func (s *PropService) processPropExtraction(taskID string, episode models.Episode) {
s.taskService.UpdateTaskStatus(taskID, "processing", 0, "Analyzing script...")

script := ""
if episode.ScriptContent != nil {
script = *episode.ScriptContent
}

// Get drama's style information
var drama models.Drama
if err := s.db.First(&drama, episode.DramaID).Error; err != nil {
s.log.Warnw("Failed to load drama", "error", err, "drama_id", episode.DramaID)
}

promptTemplate := s.promptI18n.GetPropExtractionPrompt(drama.Style, drama.AspectRatio)
prompt := fmt.Sprintf(promptTemplate, script)

response, err := s.aiService.GenerateText(prompt, "", ai.WithMaxTokens(2000))
if err != nil {
s.taskService.UpdateTaskError(taskID, err)
return
}

var extractedProps []struct {
Name        string `json:"name"`
Type        string `json:"type"`
Description string `json:"description"`
ImagePrompt string `json:"image_prompt"`
}

if err := utils.SafeParseAIJSON(response, &extractedProps); err != nil {
s.taskService.UpdateTaskError(taskID, fmt.Errorf("failed to parse AI result: %w", err))
return
}

s.taskService.UpdateTaskStatus(taskID, "processing", 50, "Saving props...")

var createdProps []models.Prop
for _, p := range extractedProps {
prop := models.Prop{
DramaID:     episode.DramaID,
Name:        p.Name,
Type:        &p.Type,
Description: &p.Description,
Prompt:      &p.ImagePrompt,
}
// Check if prop with same name already exists (avoid duplicates)
var count int64
s.db.Model(&models.Prop{}).Where("drama_id = ? AND name = ?", episode.DramaID, p.Name).Count(&count)
if count == 0 {
if err := s.db.Create(&prop).Error; err == nil {
createdProps = append(createdProps, prop)
}
}
}

s.taskService.UpdateTaskResult(taskID, createdProps)
}

// GeneratePropImage generates a prop image
// Could reuse ImageGenerationService or call AI Service directly.
// For simplicity, call ImageGenerationService if possible, or AI Service.
// For architectural consistency, should create an ImageGeneration record and reuse the existing image generation flow.
// But for quick implementation, write a dedicated method first, or better yet:
// Create an ImageGeneration record with type "prop", then reuse ImageGenerationService logic.
// However, ImageGenerationService is currently bound to Storyboard/Scene IDs.
// So here we implement a simplified direct generation logic, or extend ImageGenerationService.
// Given time constraints, implementing a simplified method that directly generates and saves the image.

func (s *PropService) GeneratePropImage(propID uint) (string, error) {
// 1. Get prop information
var prop models.Prop
if err := s.db.First(&prop, propID).Error; err != nil {
return "", err
}

if prop.Prompt == nil || *prop.Prompt == "" {
return "", fmt.Errorf("prop has no image prompt")
}

// 2. Create task
task, err := s.taskService.CreateTask("prop_image_generation", fmt.Sprintf("%d", propID))
if err != nil {
return "", err
}

go s.processPropImageGeneration(task.ID, prop)
return task.ID, nil
}

func (s *PropService) processPropImageGeneration(taskID string, prop models.Prop) {
s.taskService.UpdateTaskStatus(taskID, "processing", 0, "Generating image...")

// Prepare generation parameters
imageStyle := "Modern Japanese anime style"
imageSize := "1024x1024"

// Create generation request
req := &GenerateImageRequest{
DramaID:   fmt.Sprintf("%d", prop.DramaID),
PropID:    &prop.ID,
ImageType: string(models.ImageTypeProp),
Prompt:    *prop.Prompt,
Size:      imageSize,
Style:     &imageStyle,
Provider:  s.config.AI.DefaultImageProvider, // use default config
}

// Call ImageGenerationService
imageGen, err := s.imageGenerationService.GenerateImage(req)
if err != nil {
s.taskService.UpdateTaskError(taskID, err)
return
}

// Poll ImageGeneration status until completed
maxAttempts := 60
pollInterval := 2 * time.Second

for i := 0; i < maxAttempts; i++ {
time.Sleep(pollInterval)

// Reload imageGen
var currentImageGen models.ImageGeneration
if err := s.db.First(&currentImageGen, imageGen.ID).Error; err != nil {
s.log.Errorw("Failed to poll image generation", "error", err, "id", imageGen.ID)
continue
}

if currentImageGen.Status == models.ImageStatusCompleted {
if currentImageGen.ImageURL != nil {
// Task succeeded
// ImageGenerationService already updated Prop.ImageURL, only need to update TaskService here
s.taskService.UpdateTaskResult(taskID, map[string]string{"image_url": *currentImageGen.ImageURL})
return
}
} else if currentImageGen.Status == models.ImageStatusFailed {
errMsg := "image generation failed"
if currentImageGen.ErrorMsg != nil {
errMsg = *currentImageGen.ErrorMsg
}
s.taskService.UpdateTaskError(taskID, fmt.Errorf(errMsg))
return
}

// Update progress (optional)
s.taskService.UpdateTaskStatus(taskID, "processing", 10+i, "Generating image...")
}

s.taskService.UpdateTaskError(taskID, fmt.Errorf("generation timeout"))
}

// AssociatePropsWithStoryboard associates props with a storyboard
func (s *PropService) AssociatePropsWithStoryboard(storyboardID uint, propIDs []uint) error {
var storyboard models.Storyboard
if err := s.db.First(&storyboard, storyboardID).Error; err != nil {
return err
}

var props []models.Prop
if len(propIDs) > 0 {
if err := s.db.Where("id IN ?", propIDs).Find(&props).Error; err != nil {
return err
}
}

return s.db.Model(&storyboard).Association("Props").Replace(props)
}
