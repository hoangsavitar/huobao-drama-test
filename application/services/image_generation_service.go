package services

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	models "github.com/drama-generator/backend/domain/models"
	"github.com/drama-generator/backend/infrastructure/storage"
	"github.com/drama-generator/backend/pkg/ai"
	"github.com/drama-generator/backend/pkg/config"
	"github.com/drama-generator/backend/pkg/image"
	"github.com/drama-generator/backend/pkg/logger"
	"github.com/drama-generator/backend/pkg/utils"
	"gorm.io/gorm"
)

type ImageGenerationService struct {
	db              *gorm.DB
	aiService       *AIService
	transferService *ResourceTransferService
	localStorage    *storage.LocalStorage
	log             *logger.Logger
	config          *config.Config
	promptI18n      *PromptI18n
	taskService     *TaskService
}

// truncateImageURL truncates image URLs to prevent base64-encoded URLs from flooding the logs
func truncateImageURL(url string) string {
	if url == "" {
		return ""
	}
	// If it's a data URI format (base64), only show the prefix
	if strings.HasPrefix(url, "data:") {
		if len(url) > 50 {
			return url[:50] + "...[base64 data]"
		}
	}
	// Truncate regular URLs if they are too long
	if len(url) > 100 {
		return url[:100] + "..."
	}
	return url
}

func NewImageGenerationService(db *gorm.DB, cfg *config.Config, transferService *ResourceTransferService, localStorage *storage.LocalStorage, log *logger.Logger) *ImageGenerationService {
	return &ImageGenerationService{
		db:              db,
		aiService:       NewAIService(db, log),
		transferService: transferService,
		localStorage:    localStorage,
		config:          cfg,
		promptI18n:      NewPromptI18n(cfg),
		log:             log,
		taskService:     NewTaskService(db, log),
	}
}

// GetDB returns the database connection
func (s *ImageGenerationService) GetDB() *gorm.DB {
	return s.db
}

type GenerateImageRequest struct {
	StoryboardID    *uint    `json:"storyboard_id"`
	DramaID         string   `json:"drama_id" binding:"required"`
	SceneID         *uint    `json:"scene_id"`
	CharacterID     *uint    `json:"character_id"`
	PropID          *uint    `json:"prop_id"`
	ImageType       string   `json:"image_type"` // character, scene, storyboard
	FrameType       *string  `json:"frame_type"` // first, key, last, panel, action
	Prompt          string   `json:"prompt" binding:"required,min=5,max=2000"`
	NegativePrompt  *string  `json:"negative_prompt"`
	Provider        string   `json:"provider"`
	Model           string   `json:"model"`
	Size            string   `json:"size"`
	Quality         string   `json:"quality"`
	Style           *string  `json:"style"`
	Steps           *int     `json:"steps"`
	CfgScale        *float64 `json:"cfg_scale"`
	Seed            *int64   `json:"seed"`
	Width           *int     `json:"width"`
	Height          *int     `json:"height"`
	ImageLocalPath  *string  `json:"image_local_path"` // Local image path, used for image-to-image generation
	ReferenceImages []string `json:"reference_images"` // List of reference image URLs
}

func (s *ImageGenerationService) GenerateImage(request *GenerateImageRequest) (*models.ImageGeneration, error) {
	var drama models.Drama
	if err := s.db.Where("id = ? ", request.DramaID).First(&drama).Error; err != nil {
		return nil, fmt.Errorf("drama not found")
	}
	// Note: SceneID may refer to the Scene or Storyboard table; the caller has already performed permission verification, so we skip it here

	provider := request.Provider
	if provider == "" {
		provider = "openai"
	}

	// Serialize reference images
	var referenceImagesJSON []byte
	if len(request.ReferenceImages) > 0 {
		referenceImagesJSON, _ = json.Marshal(request.ReferenceImages)
	}

	// Convert DramaID
	dramaIDParsed, err := strconv.ParseUint(request.DramaID, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid drama ID")
	}

	// Set default image type
	imageType := request.ImageType
	if imageType == "" {
		imageType = string(models.ImageTypeStoryboard)
	}

	imageGen := &models.ImageGeneration{
		StoryboardID:    request.StoryboardID,
		DramaID:         uint(dramaIDParsed),
		SceneID:         request.SceneID,
		CharacterID:     request.CharacterID,
		PropID:          request.PropID,
		ImageType:       imageType,
		FrameType:       request.FrameType,
		Provider:        provider,
		Prompt:          request.Prompt,
		NegPrompt:       request.NegativePrompt,
		Model:           request.Model,
		Size:            request.Size,
		ReferenceImages: referenceImagesJSON,
		Quality:         request.Quality,
		Style:           request.Style,
		Steps:           request.Steps,
		CfgScale:        request.CfgScale,
		Seed:            request.Seed,
		Width:           request.Width,
		Height:          request.Height,
		LocalPath:       request.ImageLocalPath,
		Status:          models.ImageStatusPending,
	}

	if err := s.db.Create(imageGen).Error; err != nil {
		return nil, fmt.Errorf("failed to create record: %w", err)
	}

	go s.ProcessImageGeneration(imageGen.ID)

	return imageGen, nil
}

func (s *ImageGenerationService) ProcessImageGeneration(imageGenID uint) {
	var imageGen models.ImageGeneration
	imageRatio := "16:9"
	if err := s.db.First(&imageGen, imageGenID).Error; err != nil {
		s.log.Errorw("Failed to load image generation", "error", err, "id", imageGenID)
		return
	}

	// Get the drama's style information
	var drama models.Drama
	if err := s.db.First(&drama, imageGen.DramaID).Error; err != nil {
		s.log.Warnw("Failed to load drama for style", "error", err, "drama_id", imageGen.DramaID)
	}

	s.db.Model(&imageGen).Update("status", models.ImageStatusProcessing)

	// If associated with a background, synchronously update background status to generating
	if imageGen.StoryboardID != nil {
		if err := s.db.Model(&models.Scene{}).Where("id = ?", *imageGen.StoryboardID).Update("status", "generating").Error; err != nil {
			s.log.Warnw("Failed to update background status to generating", "scene_id", *imageGen.StoryboardID, "error", err)
		} else {
			s.log.Infow("Background status updated to generating", "scene_id", *imageGen.StoryboardID)
		}
	}

	client, err := s.getImageClientWithModel(imageGen.Provider, imageGen.Model)
	if err != nil {
		s.log.Errorw("Failed to get image client", "error", err, "provider", imageGen.Provider, "model", imageGen.Model)
		s.updateImageGenError(imageGenID, err.Error())
		return
	}

	// Parse reference images
	var referenceImagePaths []string
	if len(imageGen.ReferenceImages) > 0 {
		if err := json.Unmarshal(imageGen.ReferenceImages, &referenceImagePaths); err == nil {
			s.log.Infow("Using reference images for generation",
				"id", imageGenID,
				"reference_count", len(referenceImagePaths),
				"references", referenceImagePaths)
		}
	}

	// If local_path exists, prepend it to the reference images list
	if imageGen.LocalPath != nil && *imageGen.LocalPath != "" {
		referenceImagePaths = append([]string{*imageGen.LocalPath}, referenceImagePaths...)
	}

	// Convert all reference image paths to base64 (if local paths) or keep as-is (if URLs)
	var referenceImages []string
	for _, imgPath := range referenceImagePaths {
		// Check if it's an HTTP/HTTPS URL
		if strings.HasPrefix(imgPath, "http://") || strings.HasPrefix(imgPath, "https://") {
			// Keep the URL as-is
			referenceImages = append(referenceImages, imgPath)
		} else {
			// Treat as local path, convert to base64
			base64Image, err := s.loadImageAsBase64(imgPath)
			if err != nil {
				s.log.Warnw("Failed to load local image as base64",
					"error", err,
					"id", imageGenID,
					"local_path", imgPath)
			} else {
				referenceImages = append(referenceImages, base64Image)
				s.log.Infow("Loaded local image for generation",
					"id", imageGenID,
					"local_path", imgPath)
			}
		}
	}

	s.log.Infow("Starting image generation", "id", imageGenID, "prompt", imageGen.Prompt, "provider", imageGen.Provider)

	var opts []image.ImageOption
	if imageGen.NegPrompt != nil && *imageGen.NegPrompt != "" {
		opts = append(opts, image.WithNegativePrompt(*imageGen.NegPrompt))
	}
	if imageGen.Size != "" {
		opts = append(opts, image.WithSize(imageGen.Size))
	}
	if imageGen.Quality != "" {
		opts = append(opts, image.WithQuality(imageGen.Quality))
	}
	if imageGen.Style != nil && *imageGen.Style != "" {
		opts = append(opts, image.WithStyle(*imageGen.Style))
	}
	if imageGen.Steps != nil {
		opts = append(opts, image.WithSteps(*imageGen.Steps))
	}
	if imageGen.CfgScale != nil {
		opts = append(opts, image.WithCfgScale(*imageGen.CfgScale))
	}
	if imageGen.Seed != nil {
		opts = append(opts, image.WithSeed(*imageGen.Seed))
	}
	if imageGen.Model != "" {
		opts = append(opts, image.WithModel(imageGen.Model))
	}
	if imageGen.Width != nil && imageGen.Height != nil {
		opts = append(opts, image.WithDimensions(*imageGen.Width, *imageGen.Height))
	}
	// Add reference images
	if len(referenceImages) > 0 {
		opts = append(opts, image.WithReferenceImages(referenceImages))
	}

	// Build the full prompt: style prompt + user prompt
	prompt := imageGen.Prompt

	// If the drama has a style setting, add the style prompt
	if drama.Style != "" && drama.Style != "realistic" {
		stylePrompt := s.promptI18n.GetStylePrompt(drama.Style)
		if stylePrompt != "" {
			// Prepend the style prompt as a system-level constraint
			prompt = stylePrompt + "\n\n" + prompt
			s.log.Infow("Added style prompt to image generation",
				"id", imageGenID,
				"style", drama.Style,
				"style_prompt_length", len(stylePrompt))
		}
	}

	prompt += ", imageRatio:" + imageRatio

	// If there are reference images, append a consistency instruction to the prompt
	if len(referenceImages) > 0 {
		prompt += "\n\nImportant: strictly follow reference image elements and keep scene/character consistency."
		s.log.Infow("Added reference image consistency instruction to prompt",
			"id", imageGenID,
			"reference_count", len(referenceImages))
	}
	result, err := client.GenerateImage(prompt, opts...)
	if err != nil {
		s.log.Errorw("Image generation API call failed", "error", err, "id", imageGenID, "prompt", imageGen.Prompt)
		s.updateImageGenError(imageGenID, err.Error())
		return
	}

	s.log.Infow("Image generation API call completed", "id", imageGenID, "completed", result.Completed, "has_url", result.ImageURL != "")

	if !result.Completed {
		s.db.Model(&imageGen).Updates(map[string]interface{}{
			"status":  models.ImageStatusProcessing,
			"task_id": result.TaskID,
		})
		go s.pollTaskStatus(imageGenID, client, result.TaskID)
		return
	}

	s.completeImageGeneration(imageGenID, result)
}

func (s *ImageGenerationService) pollTaskStatus(imageGenID uint, client image.ImageClient, taskID string) {
	maxAttempts := 60
	pollInterval := 5 * time.Second

	for i := 0; i < maxAttempts; i++ {
		time.Sleep(pollInterval)

		result, err := client.GetTaskStatus(taskID)
		if err != nil {
			s.log.Errorw("Failed to get task status", "error", err, "task_id", taskID)
			continue
		}

		if result.Completed {
			s.completeImageGeneration(imageGenID, result)
			return
		}

		if result.Error != "" {
			s.updateImageGenError(imageGenID, result.Error)
			return
		}
	}

	s.updateImageGenError(imageGenID, "timeout: image generation took too long")
}

func (s *ImageGenerationService) completeImageGeneration(imageGenID uint, result *image.ImageResult) {
	now := time.Now()

	// Download image to local storage and save the relative path to the database
	var localPath *string
	if s.localStorage != nil && result.ImageURL != "" &&
		(strings.HasPrefix(result.ImageURL, "http://") || strings.HasPrefix(result.ImageURL, "https://")) {
		downloadResult, err := s.localStorage.DownloadFromURLWithPath(result.ImageURL, "images")
		if err != nil {
			errStr := err.Error()
			if len(errStr) > 200 {
				errStr = errStr[:200] + "..."
			}
			s.log.Warnw("Failed to download image to local storage",
				"error", errStr,
				"id", imageGenID,
				"original_url", truncateImageURL(result.ImageURL))
		} else {
			localPath = &downloadResult.RelativePath
			s.log.Infow("Image downloaded to local storage",
				"id", imageGenID,
				"original_url", truncateImageURL(result.ImageURL),
				"local_path", downloadResult.RelativePath)
		}
	}

	// Save the original URL and local path in the database
	updates := map[string]interface{}{
		"status":       models.ImageStatusCompleted,
		"image_url":    result.ImageURL,
		"local_path":   localPath,
		"completed_at": now,
	}

	if result.Width > 0 {
		updates["width"] = result.Width
	}
	if result.Height > 0 {
		updates["height"] = result.Height
	}

	// Update the image_generation record
	var imageGen models.ImageGeneration
	if err := s.db.Where("id = ?", imageGenID).First(&imageGen).Error; err != nil {
		s.log.Errorw("Failed to load image generation", "error", err, "id", imageGenID)
		return
	}

	// Use Updates to update basic fields
	if err := s.db.Model(&models.ImageGeneration{}).Where("id = ?", imageGenID).Updates(updates).Error; err != nil {
		s.log.Errorw("Failed to update image generation", "error", err, "id", imageGenID)
		return
	}

	// Update the local_path field separately (even if it's nil)
	if err := s.db.Model(&models.ImageGeneration{}).Where("id = ?", imageGenID).Update("local_path", localPath).Error; err != nil {
		s.log.Errorw("Failed to update local_path", "error", err, "id", imageGenID)
	}

	s.log.Infow("Image generation completed", "id", imageGenID)

	// If associated with a storyboard, synchronously update the storyboard's composed_image
	if imageGen.StoryboardID != nil {
		if err := s.db.Model(&models.Storyboard{}).Where("id = ?", *imageGen.StoryboardID).Update("composed_image", result.ImageURL).Error; err != nil {
			s.log.Errorw("Failed to update storyboard composed_image", "error", err, "storyboard_id", *imageGen.StoryboardID)
		} else {
			s.log.Infow("Storyboard updated with composed image",
				"storyboard_id", *imageGen.StoryboardID,
				"composed_image", truncateImageURL(result.ImageURL))
		}
	}

	// If associated with a scene, synchronously update the scene's image_url, local_path, and status (only when ImageType is scene)
	if imageGen.SceneID != nil && imageGen.ImageType == string(models.ImageTypeScene) {
		sceneUpdates := map[string]interface{}{
			"status":    "generated",
			"image_url": result.ImageURL,
		}
		if localPath != nil {
			sceneUpdates["local_path"] = localPath
		}
		if err := s.db.Model(&models.Scene{}).Where("id = ?", *imageGen.SceneID).Updates(sceneUpdates).Error; err != nil {
			s.log.Errorw("Failed to update scene", "error", err, "scene_id", *imageGen.SceneID)
		} else {
			s.log.Infow("Scene updated with generated image",
				"scene_id", *imageGen.SceneID,
				"image_url", truncateImageURL(result.ImageURL),
				"local_path", localPath)
		}
	}

	// If associated with a character, synchronously update the character's image_url and local_path
	if imageGen.CharacterID != nil {
		characterUpdates := map[string]interface{}{
			"image_url": result.ImageURL,
		}
		if localPath != nil {
			characterUpdates["local_path"] = localPath
		}
		if err := s.db.Model(&models.Character{}).Where("id = ?", *imageGen.CharacterID).Updates(characterUpdates).Error; err != nil {
			s.log.Errorw("Failed to update character", "error", err, "character_id", *imageGen.CharacterID)
		} else {
			s.log.Infow("Character updated with generated image",
				"character_id", *imageGen.CharacterID,
				"image_url", truncateImageURL(result.ImageURL),
				"local_path", localPath)
		}
	}

	// If associated with a prop, synchronously update the prop's image_url and local_path
	if imageGen.PropID != nil {
		propUpdates := map[string]interface{}{
			"image_url": result.ImageURL,
		}
		if localPath != nil {
			propUpdates["local_path"] = localPath
		}
		if err := s.db.Model(&models.Prop{}).Where("id = ?", *imageGen.PropID).Updates(propUpdates).Error; err != nil {
			s.log.Errorw("Failed to update prop", "error", err, "prop_id", *imageGen.PropID)
		} else {
			s.log.Infow("Prop updated with generated image",
				"prop_id", *imageGen.PropID,
				"image_url", truncateImageURL(result.ImageURL),
				"local_path", localPath)
		}
	}
}

func (s *ImageGenerationService) updateImageGenError(imageGenID uint, errorMsg string) {
	// First, load the image_generation record
	var imageGen models.ImageGeneration
	if err := s.db.Where("id = ?", imageGenID).First(&imageGen).Error; err != nil {
		s.log.Errorw("Failed to load image generation", "error", err, "id", imageGenID)
		return
	}

	// Update the image_generation status
	s.db.Model(&models.ImageGeneration{}).Where("id = ?", imageGenID).Updates(map[string]interface{}{
		"status":    models.ImageStatusFailed,
		"error_msg": errorMsg,
	})
	s.log.Errorw("Image generation failed", "id", imageGenID, "error", errorMsg)

	// If associated with a scene, synchronously update the scene to failed status
	if imageGen.SceneID != nil {
		s.db.Model(&models.Scene{}).Where("id = ?", *imageGen.SceneID).Update("status", "failed")
		s.log.Warnw("Scene marked as failed", "scene_id", *imageGen.SceneID)
	}
}

func (s *ImageGenerationService) getImageClient(provider string) (image.ImageClient, error) {
	config, err := s.aiService.GetDefaultConfig("image")
	if err != nil {
		return nil, fmt.Errorf("no image AI config found: %w", err)
	}

	// Use the first model
	model := ""
	if len(config.Model) > 0 {
		model = config.Model[0]
	}

	// Use the provider from config; if not set, use the provided provider
	actualProvider := config.Provider
	if actualProvider == "" {
		actualProvider = provider
	}

	// Automatically set the default endpoint based on provider
	var endpoint string
	var queryEndpoint string

	switch actualProvider {
	case "openai", "dalle":
		endpoint = "/images/generations"
		return image.NewOpenAIImageClient(config.BaseURL, config.APIKey, model, endpoint), nil
	case "chatfire":
		endpoint = "/images/generations"
		return image.NewOpenAIImageClient(config.BaseURL, config.APIKey, model, endpoint), nil
	case "volcengine", "volces", "doubao":
		endpoint = "/images/generations"
		queryEndpoint = ""
		return image.NewVolcEngineImageClient(config.BaseURL, config.APIKey, model, endpoint, queryEndpoint), nil
	case "gemini", "google":
		endpoint = "/v1beta/models/{model}:generateContent"
		return image.NewGeminiImageClient(config.BaseURL, config.APIKey, model, endpoint), nil
	default:
		endpoint = "/images/generations"
		return image.NewOpenAIImageClient(config.BaseURL, config.APIKey, model, endpoint), nil
	}
}

// getImageClientWithModel gets the image client based on the model name
func (s *ImageGenerationService) getImageClientWithModel(provider string, modelName string) (image.ImageClient, error) {
	var config *models.AIServiceConfig
	var err error

	// If a model is specified, try to get the corresponding config
	if modelName != "" {
		config, err = s.aiService.GetConfigForModel("image", modelName)
		if err != nil {
			s.log.Warnw("Failed to get config for model, using default", "model", modelName, "error", err)
			config, err = s.aiService.GetDefaultConfig("image")
			if err != nil {
				return nil, fmt.Errorf("no image AI config found: %w", err)
			}
		}
	} else {
		config, err = s.aiService.GetDefaultConfig("image")
		if err != nil {
			return nil, fmt.Errorf("no image AI config found: %w", err)
		}
	}

	// Use the specified model or the first model from config
	model := modelName
	if model == "" && len(config.Model) > 0 {
		model = config.Model[0]
	}

	// Use the provider from config; if not set, use the provided provider
	actualProvider := config.Provider
	if actualProvider == "" {
		actualProvider = provider
	}

	// Automatically set the default endpoint based on provider
	var endpoint string
	var queryEndpoint string

	switch actualProvider {
	case "openai", "dalle":
		endpoint = "/images/generations"
		return image.NewOpenAIImageClient(config.BaseURL, config.APIKey, model, endpoint), nil
	case "chatfire":
		endpoint = "/images/generations"
		return image.NewOpenAIImageClient(config.BaseURL, config.APIKey, model, endpoint), nil
	case "volcengine", "volces", "doubao":
		endpoint = "/images/generations"
		queryEndpoint = ""
		return image.NewVolcEngineImageClient(config.BaseURL, config.APIKey, model, endpoint, queryEndpoint), nil
	case "gemini", "google":
		endpoint = "/v1beta/models/{model}:generateContent"
		return image.NewGeminiImageClient(config.BaseURL, config.APIKey, model, endpoint), nil
	default:
		endpoint = "/images/generations"
		return image.NewOpenAIImageClient(config.BaseURL, config.APIKey, model, endpoint), nil
	}
}

func (s *ImageGenerationService) GetImageGeneration(imageGenID uint) (*models.ImageGeneration, error) {
	var imageGen models.ImageGeneration
	if err := s.db.Where("id = ? ", imageGenID).First(&imageGen).Error; err != nil {
		return nil, err
	}
	return &imageGen, nil
}

func (s *ImageGenerationService) ListImageGenerations(dramaID *uint, sceneID *uint, storyboardID *uint, frameType string, status string, page, pageSize int) ([]models.ImageGeneration, int64, error) {
	query := s.db.Model(&models.ImageGeneration{})

	if dramaID != nil {
		query = query.Where("drama_id = ?", *dramaID)
	}

	if sceneID != nil {
		query = query.Where("scene_id = ?", *sceneID)
	}

	if storyboardID != nil {
		query = query.Where("storyboard_id = ?", *storyboardID)
	}

	if frameType != "" {
		query = query.Where("frame_type = ?", frameType)
	}

	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var images []models.ImageGeneration
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&images).Error; err != nil {
		return nil, 0, err
	}

	return images, total, nil
}

func (s *ImageGenerationService) DeleteImageGeneration(imageGenID uint) error {
	result := s.db.Where("id = ? ", imageGenID).Delete(&models.ImageGeneration{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("image generation not found")
	}
	return nil
}

// UploadImageRequest represents an image upload request
type UploadImageRequest struct {
	StoryboardID uint   `json:"storyboard_id"`
	DramaID      uint   `json:"drama_id"`
	FrameType    string `json:"frame_type"`
	ImageURL     string `json:"image_url"`
	Prompt       string `json:"prompt"`
}

// CreateImageFromUpload creates an image generation record from an uploaded image URL
func (s *ImageGenerationService) CreateImageFromUpload(req *UploadImageRequest) (*models.ImageGeneration, error) {
	// Verify that the storyboard exists
	var storyboard models.Storyboard
	if err := s.db.First(&storyboard, req.StoryboardID).Error; err != nil {
		return nil, fmt.Errorf("storyboard not found")
	}

	// Verify that the drama exists
	var drama models.Drama
	if err := s.db.First(&drama, req.DramaID).Error; err != nil {
		return nil, fmt.Errorf("drama not found")
	}

	prompt := req.Prompt
	if prompt == "" {
		prompt = "User uploaded image"
	}

	now := time.Now()
	imageGen := &models.ImageGeneration{
		StoryboardID: &req.StoryboardID,
		DramaID:      req.DramaID,
		ImageType:    string(models.ImageTypeStoryboard),
		FrameType:    &req.FrameType,
		Provider:     "upload",
		Prompt:       prompt,
		Model:        "upload",
		ImageURL:     &req.ImageURL,
		Status:       models.ImageStatusCompleted,
		CompletedAt:  &now,
	}

	if err := s.db.Create(imageGen).Error; err != nil {
		return nil, fmt.Errorf("failed to create image record: %w", err)
	}

	s.log.Infow("Image created from upload",
		"id", imageGen.ID,
		"storyboard_id", req.StoryboardID,
		"frame_type", req.FrameType)

	return imageGen, nil
}

func (s *ImageGenerationService) GenerateImagesForScene(sceneID string) ([]*models.ImageGeneration, error) {
	// Convert sceneID
	sid, err := strconv.ParseUint(sceneID, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid scene ID")
	}
	sceneIDUint := uint(sid)

	var scene models.Scene
	if err := s.db.Where("id = ?", sceneIDUint).First(&scene).Error; err != nil {
		return nil, fmt.Errorf("scene not found")
	}

	// Build the scene image generation prompt
	prompt := scene.Prompt
	if prompt == "" {
		// If Prompt is empty, build one using Location and Time
		prompt = fmt.Sprintf("%s scene, %s", scene.Location, scene.Time)
	}

	req := &GenerateImageRequest{
		SceneID:   &sceneIDUint,
		DramaID:   fmt.Sprintf("%d", scene.DramaID),
		ImageType: string(models.ImageTypeScene),
		Prompt:    prompt,
	}

	imageGen, err := s.GenerateImage(req)
	if err != nil {
		return nil, err
	}

	return []*models.ImageGeneration{imageGen}, nil
}

// BackgroundInfo represents the background information structure
type BackgroundInfo struct {
	Location          string `json:"location"`
	Time              string `json:"time"`
	Atmosphere        string `json:"atmosphere"`
	Prompt            string `json:"prompt"`
	StoryboardNumbers []int  `json:"storyboard_numbers"`
	SceneIDs          []uint `json:"scene_ids"`
	StoryboardCount   int    `json:"scene_count"`
}

func (s *ImageGenerationService) BatchGenerateImagesForEpisode(episodeID string) ([]*models.ImageGeneration, error) {
	var ep models.Episode
	if err := s.db.Preload("Drama").Where("id = ?", episodeID).First(&ep).Error; err != nil {
		return nil, fmt.Errorf("episode not found")
	}
	// Read saved scenes from the database
	var scenes []models.Storyboard
	if err := s.db.Where("episode_id = ?", episodeID).Find(&scenes).Error; err != nil {
		return nil, fmt.Errorf("failed to get scenes: %w", err)
	}

	backgrounds := s.extractUniqueBackgrounds(scenes)
	s.log.Infow("Extracted unique backgrounds",
		"episode_id", episodeID,
		"background_count", len(backgrounds))

	// Generate images for each background
	var results []*models.ImageGeneration
	for _, bg := range scenes {
		if bg.ImagePrompt == nil || *bg.ImagePrompt == "" {
			s.log.Warnw("Background has no prompt, skipping", "scene_id", bg.ID)
			continue
		}

		// Update background status to processing
		s.db.Model(bg).Update("status", "generating")

		req := &GenerateImageRequest{
			StoryboardID: &bg.ID,
			DramaID:      fmt.Sprintf("%d", ep.DramaID),
			Prompt:       *bg.ImagePrompt,
		}

		imageGen, err := s.GenerateImage(req)
		if err != nil {
			s.log.Errorw("Failed to generate image for background",
				"scene_id", bg.ID,
				"location", bg.Location,
				"error", err)
			s.db.Model(bg).Update("status", "failed")
			continue
		}

		s.log.Infow("Background image generation started",
			"scene_id", bg.ID,
			"image_gen_id", imageGen.ID,
			"location", bg.Location,
			"time", bg.Time)

		results = append(results, imageGen)
	}

	return results, nil
}

// GetScencesForEpisode gets the scene list for the project (project-level)
func (s *ImageGenerationService) GetScencesForEpisode(episodeID string) ([]*models.Scene, error) {
	var episode models.Episode
	if err := s.db.Preload("Drama").Where("id = ?", episodeID).First(&episode).Error; err != nil {
		return nil, fmt.Errorf("episode not found")
	}

	// Scenes are project-level, queried by drama_id
	var scenes []*models.Scene
	if err := s.db.Where("drama_id = ?", episode.DramaID).Order("location ASC, time ASC").Find(&scenes).Error; err != nil {
		return nil, fmt.Errorf("failed to load scenes: %w", err)
	}

	return scenes, nil
}

// ExtractBackgroundsForEpisode extracts scenes from script content and saves them to the project-level database
func (s *ImageGenerationService) ExtractBackgroundsForEpisode(episodeID string, model string, style string) (string, error) {
	var episode models.Episode
	if err := s.db.Preload("Storyboards").First(&episode, episodeID).Error; err != nil {
		return "", fmt.Errorf("episode not found")
	}

	// Cannot extract scenes without script content
	if episode.ScriptContent == nil || *episode.ScriptContent == "" {
		return "", fmt.Errorf("episode has no script content")
	}

	// Create task
	task, err := s.taskService.CreateTask("background_extraction", episodeID)
	if err != nil {
		s.log.Errorw("Failed to create background extraction task", "error", err, "episode_id", episodeID)
		return "", fmt.Errorf("failed to create task: %w", err)
	}

	// Process scene extraction asynchronously
	go s.processBackgroundExtraction(task.ID, episodeID, model, style)

	s.log.Infow("Background extraction task created", "task_id", task.ID, "episode_id", episodeID)
	return task.ID, nil
}

// processBackgroundExtraction processes scene extraction asynchronously
func (s *ImageGenerationService) processBackgroundExtraction(taskID string, episodeID string, model string, style string) {
	// Update task status to processing
	s.taskService.UpdateTaskStatus(taskID, "processing", 0, "Extracting scene information...")

	var episode models.Episode
	if err := s.db.Preload("Storyboards").First(&episode, episodeID).Error; err != nil {
		s.log.Errorw("Episode not found during background extraction", "error", err, "episode_id", episodeID)
		s.taskService.UpdateTaskStatus(taskID, "failed", 0, "Episode info not found")
		return
	}

	if episode.ScriptContent == nil || *episode.ScriptContent == "" {
		s.log.Errorw("Episode has no script content during background extraction", "episode_id", episodeID)
		s.taskService.UpdateTaskStatus(taskID, "failed", 0, "Script content is empty")
		return
	}

	s.log.Infow("Extracting backgrounds from script", "episode_id", episodeID, "model", model, "task_id", taskID)
	dramaID := episode.DramaID

	// Use AI to extract scenes from the script content
	backgroundsInfo, err := s.extractBackgroundsFromScript(*episode.ScriptContent, dramaID, model, style)
	if err != nil {
		s.log.Errorw("Failed to extract backgrounds from script", "error", err, "task_id", taskID)
		s.taskService.UpdateTaskStatus(taskID, "failed", 0, "AI scene extraction failed: "+err.Error())
		return
	}

	// Save to database (no Storyboard association, as storyboards haven't been generated yet)
	var scenes []*models.Scene
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// First delete all scenes for this episode (to support re-extraction override)
		if err := tx.Where("episode_id = ?", episode.ID).Delete(&models.Scene{}).Error; err != nil {
			s.log.Errorw("Failed to delete old scenes", "error", err, "task_id", taskID)
			return err
		}
		s.log.Infow("Deleted old scenes for re-extraction", "episode_id", episode.ID, "task_id", taskID)

		// Create newly extracted scenes
		for _, bgInfo := range backgroundsInfo {
			// Save new scene to database (episode-level)
			episodeIDVal := episode.ID
			scene := &models.Scene{
				DramaID:         dramaID,
				EpisodeID:       &episodeIDVal,
				Location:        bgInfo.Location,
				Time:            bgInfo.Time,
				Prompt:          bgInfo.Prompt,
				StoryboardCount: 1, // Default is 1
				Status:          "pending",
			}
			if err := tx.Create(scene).Error; err != nil {
				return err
			}
			scenes = append(scenes, scene)

			s.log.Infow("Created new scene from script",
				"scene_id", scene.ID,
				"location", scene.Location,
				"time", scene.Time,
				"task_id", taskID)
		}

		return nil
	})

	if err != nil {
		s.log.Errorw("Failed to save scenes to database", "error", err, "task_id", taskID)
		s.taskService.UpdateTaskStatus(taskID, "failed", 0, "Failed to save scene information: "+err.Error())
		return
	}

	// Update task status to completed
	resultData := map[string]interface{}{
		"scenes":     scenes,
		"count":      len(scenes),
		"episode_id": episodeID,
		"drama_id":   dramaID,
	}
	s.taskService.UpdateTaskResult(taskID, resultData)

	s.log.Infow("Background extraction completed",
		"task_id", taskID,
		"episode_id", episodeID,
		"total_storyboards", len(episode.Storyboards),
		"unique_scenes", len(scenes))
}

// extractBackgroundsFromScript uses AI to extract scene information from script content
func (s *ImageGenerationService) extractBackgroundsFromScript(scriptContent string, dramaID uint, model string, style string) ([]BackgroundInfo, error) {
	if scriptContent == "" {
		return []BackgroundInfo{}, nil
	}

	// Get the AI client (use the specified model if one is provided)
	var client ai.AIClient
	var err error
	if model != "" {
		s.log.Infow("Using specified model for background extraction", "model", model)
		client, err = s.aiService.GetAIClientForModel("text", model)
		if err != nil {
			s.log.Warnw("Failed to get client for specified model, using default", "model", model, "error", err)
			client, err = s.aiService.GetAIClient("text")
		}
	} else {
		client, err = s.aiService.GetAIClient("text")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get AI client: %w", err)
	}

	// Use internationalized prompts
	systemPrompt := s.promptI18n.GetSceneExtractionPrompt(style)
	contentLabel := s.promptI18n.FormatUserPrompt("script_content_label")

	// Build different format instructions based on language
	var formatInstructions string
	if s.promptI18n.IsEnglish() {
		formatInstructions = `[Output JSON Format]
{
  "backgrounds": [
    {
      "location": "Location name (English)",
      "time": "Time description (English)",
      "atmosphere": "Atmosphere description (English)",
      "prompt": "A cinematic anime-style pure background scene depicting [location description] at [time]. The scene shows [environment details, architecture, objects, lighting, no characters]. Style: rich details, high quality, atmospheric lighting. Mood: [environment mood description]."
    }
  ]
}

[Example]
Correct example (note: no characters):
{
  "backgrounds": [
    {
      "location": "Repair Shop Interior",
      "time": "Late Night",
      "atmosphere": "Dim, lonely, industrial",
      "prompt": "A cinematic anime-style pure background scene depicting a messy repair shop interior at late night. Under dim fluorescent lights, the workbench is scattered with various wrenches, screwdrivers and mechanical parts, oil-stained tool boards and faded posters hang on walls, oil stains on the floor, used tires piled in corners. Style: rich details, high quality, dim atmosphere. Mood: lonely, industrial."
    },
    {
      "location": "City Street",
      "time": "Dusk",
      "atmosphere": "Warm, busy, lively",
      "prompt": "A cinematic anime-style pure background scene depicting a bustling city street at dusk. Sunset afterglow shines on the asphalt road, neon lights of shops on both sides begin to light up, bicycle racks and bus stops on the street, high-rise buildings in the distance, sky showing orange-red gradient. Style: rich details, high quality, warm atmosphere. Mood: lively, busy."
    }
  ]
}

[Wrong Examples (containing characters, forbidden)]:
❌ "Depicting protagonist standing on the street" - contains character
❌ "People hurrying by" - contains characters
❌ "Character moving in the room" - contains character

Please strictly follow the JSON format and ensure all fields use English.`
	} else {
		formatInstructions = `[Output JSON Format]
{
  "backgrounds": [
    {
      "location": "Location name",
      "time": "Time description",
      "atmosphere": "Atmosphere description",
      "prompt": "A cinematic anime-style pure background scene depicting [location description] at [time]. The scene shows [environment details, architecture, objects, lighting, no characters]. Style: rich details, high quality, atmospheric lighting. Mood: [environment mood description]."
    }
  ]
}

[Example]
Correct example (note: no characters):
{
  "backgrounds": [
    {
      "location": "Repair Shop Interior",
      "time": "Late Night",
      "atmosphere": "Dim, lonely, industrial",
      "prompt": "A cinematic anime-style pure background scene depicting a messy repair shop interior at late night. Under dim fluorescent lights, the workbench is scattered with various wrenches, screwdrivers and mechanical parts, oil-stained tool boards and faded posters hang on walls, oil stains on the floor, used tires piled in corners. Style: rich details, high quality, dim atmosphere. Mood: lonely, industrial."
    },
    {
      "location": "City Street",
      "time": "Dusk",
      "atmosphere": "Warm, busy, lively",
      "prompt": "A cinematic anime-style pure background scene depicting a bustling city street at dusk. Sunset afterglow shines on the asphalt road, neon lights of shops on both sides begin to light up, bicycle racks and bus stops on the street, high-rise buildings in the distance, sky showing orange-red gradient. Style: rich details, high quality, warm atmosphere. Mood: lively, busy."
    }
  ]
}

[Wrong Examples (containing characters, forbidden)]:
❌ "Depicting protagonist standing on the street" - contains character
❌ "People hurrying by" - contains characters
❌ "Character moving in the room" - contains character

Please strictly follow the JSON format and ensure all fields use English.`
	}

	prompt := fmt.Sprintf(`%s

%s
%s

%s`, systemPrompt, contentLabel, scriptContent, formatInstructions)

	// Print the full prompt for debugging
	s.log.Infow("=== AI Prompt for Background Extraction (extractBackgroundsFromScript) ===",
		"language", s.promptI18n.GetLanguage(),
		"prompt_length", len(prompt),
		"full_prompt", prompt)

	response, err := client.GenerateText(prompt, "", ai.WithTemperature(0.7))
	if err != nil {
		s.log.Errorw("Failed to extract backgrounds with AI", "error", err)
		return nil, fmt.Errorf("AI scene extraction failed: %w", err)
	}

	// Print the raw AI response
	s.log.Infow("=== AI Response for Background Extraction (extractBackgroundsFromScript) ===",
		"response_length", len(response),
		"raw_response", response)

	// Parse the JSON returned by AI
	var backgrounds []BackgroundInfo

	// First try to parse as array format
	if err := utils.SafeParseAIJSON(response, &backgrounds); err == nil {
		s.log.Infow("Parsed backgrounds as array format", "count", len(backgrounds))
	} else {
		// Try to parse as object format
		var result struct {
			Backgrounds []BackgroundInfo `json:"backgrounds"`
		}
		if err := utils.SafeParseAIJSON(response, &result); err != nil {
			s.log.Errorw("Failed to parse AI response in both formats", "error", err, "response", response[:min(len(response), 500)])
			return nil, fmt.Errorf("failed to parse AI response: %w", err)
		}
		backgrounds = result.Backgrounds
		s.log.Infow("Parsed backgrounds as object format", "count", len(backgrounds))
	}

	s.log.Infow("Extracted backgrounds from script",
		"drama_id", dramaID,
		"backgrounds_count", len(backgrounds))

	return backgrounds, nil
}

// extractBackgroundsWithAI uses AI to intelligently analyze scenes and extract unique backgrounds
func (s *ImageGenerationService) extractBackgroundsWithAI(storyboards []models.Storyboard, style string) ([]BackgroundInfo, error) {
	if len(storyboards) == 0 {
		return []BackgroundInfo{}, nil
	}

	// Build scene list text, using SceneNumber instead of index
	var scenesText string
	for _, storyboard := range storyboards {
		location := ""
		if storyboard.Location != nil {
			location = *storyboard.Location
		}
		time := ""
		if storyboard.Time != nil {
			time = *storyboard.Time
		}
		action := ""
		if storyboard.Action != nil {
			action = *storyboard.Action
		}
		description := ""
		if storyboard.Description != nil {
			description = *storyboard.Description
		}

		scenesText += fmt.Sprintf("Shot %d:\nLocation: %s\nTime: %s\nAction: %s\nDescription: %s\n\n",
			storyboard.StoryboardNumber, location, time, action, description)
	}

	// Use internationalized prompts
	systemPrompt := s.promptI18n.GetSceneExtractionPrompt(style)
	storyboardLabel := s.promptI18n.FormatUserPrompt("storyboard_list_label")

	// Build different prompts based on language
	var formatInstructions string
	if s.promptI18n.IsEnglish() {
		formatInstructions = `[Output JSON Format]
{
  "backgrounds": [
    {
      "location": "Location name (English)",
      "time": "Time description (English)",
      "prompt": "A cinematic anime-style background depicting [location description] at [time]. The scene shows [detail description]. Style: rich details, high quality, atmospheric lighting. Mood: [mood description].",
      "scene_numbers": [1, 2, 3]
    }
  ]
}

[Example]
Correct example:
{
  "backgrounds": [
    {
      "location": "Repair Shop",
      "time": "Late Night",
      "prompt": "A cinematic anime-style background depicting a messy repair shop interior at late night. Under dim lighting, the workbench is scattered with various tools and parts, with greasy posters hanging on the walls. Style: rich details, high quality, dim atmosphere. Mood: lonely, industrial.",
      "scene_numbers": [1, 5, 6, 10, 15]
    },
    {
      "location": "City Panorama",
      "time": "Late Night with Acid Rain",
      "prompt": "A cinematic anime-style background depicting a coastal city panorama in late night acid rain. Neon lights blur in the rain, skyscrapers shrouded in gray-green rain curtain, streets reflecting colorful lights. Style: rich details, high quality, cyberpunk atmosphere. Mood: oppressive, sci-fi, apocalyptic.",
      "scene_numbers": [2, 7]
    }
  ]
}

Please strictly follow the JSON format and ensure:
1. prompt field uses English
2. scene_numbers includes all scene numbers using this background
3. All scenes are assigned to a background`
	} else {
		formatInstructions = `[Output JSON Format]
{
  "backgrounds": [
    {
      "location": "Location name",
      "time": "Time description",
      "prompt": "A cinematic anime-style background depicting [location description] at [time]. The scene shows [detail description]. Style: rich details, high quality, atmospheric lighting. Mood: [mood description].",
      "scene_numbers": [1, 2, 3]
    }
  ]
}

[Example]
Correct example:
{
  "backgrounds": [
    {
      "location": "Repair Shop",
      "time": "Late Night",
      "prompt": "A cinematic anime-style background depicting a messy repair shop interior at late night. Under dim lighting, the workbench is scattered with various tools and parts, with greasy posters hanging on the walls. Style: rich details, high quality, dim atmosphere. Mood: lonely, industrial.",
      "scene_numbers": [1, 5, 6, 10, 15]
    },
    {
      "location": "City Panorama",
      "time": "Late Night with Acid Rain",
      "prompt": "A cinematic anime-style background depicting a coastal city panorama in late night acid rain. Neon lights blur in the rain, skyscrapers shrouded in gray-green rain curtain, streets reflecting colorful lights. Style: rich details, high quality, cyberpunk atmosphere. Mood: oppressive, sci-fi, apocalyptic.",
      "scene_numbers": [2, 7]
    }
  ]
}

Please strictly follow the JSON format and ensure:
1. prompt field uses English
2. scene_numbers includes all scene numbers using this background
3. All scenes are assigned to a background`
	}

	prompt := fmt.Sprintf(`%s

%s
%s

%s`, systemPrompt, storyboardLabel, scenesText, formatInstructions)

	// Print the full prompt for debugging
	s.log.Infow("=== AI Prompt for Background Extraction (extractBackgroundsWithAI) ===",
		"language", s.promptI18n.GetLanguage(),
		"prompt_length", len(prompt),
		"full_prompt", prompt)

	// Call the AI service
	text, err := s.aiService.GenerateText(prompt, "")
	if err != nil {
		return nil, fmt.Errorf("AI analysis failed: %w", err)
	}

	// Print the raw AI response
	s.log.Infow("=== AI Response for Background Extraction ===",
		"response_length", len(text),
		"raw_response", text)

	// Parse the JSON returned by AI
	var result struct {
		Scenes []struct {
			Location         string `json:"location"`
			Time             string `json:"time"`
			Prompt           string `json:"prompt"`
			StoryboardNumber []int  `json:"storyboard_number"`
		} `json:"backgrounds"`
	}

	if err := utils.SafeParseAIJSON(text, &result); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	// Build a mapping from scene numbers to scene IDs
	storyboardNumberToID := make(map[int]uint)
	for _, scene := range storyboards {
		storyboardNumberToID[scene.StoryboardNumber] = scene.ID
	}

	// Convert to BackgroundInfo
	var backgrounds []BackgroundInfo
	for _, bg := range result.Scenes {
		// Convert scene numbers to scene IDs
		var sceneIDs []uint
		for _, storyboardNum := range bg.StoryboardNumber {
			if storyboardID, ok := storyboardNumberToID[storyboardNum]; ok {
				sceneIDs = append(sceneIDs, storyboardID)
			}
		}

		backgrounds = append(backgrounds, BackgroundInfo{
			Location:          bg.Location,
			Time:              bg.Time,
			Prompt:            bg.Prompt,
			StoryboardNumbers: bg.StoryboardNumber,
			SceneIDs:          sceneIDs,
			StoryboardCount:   len(sceneIDs),
		})
	}

	s.log.Infow("AI extracted backgrounds",
		"total_scenes", len(storyboards),
		"extracted_backgrounds", len(backgrounds))

	return backgrounds, nil
}

// extractUniqueBackgrounds extracts unique backgrounds from storyboards (code logic, as a backup for AI extraction)
func (s *ImageGenerationService) extractUniqueBackgrounds(scenes []models.Storyboard) []BackgroundInfo {
	backgroundMap := make(map[string]*BackgroundInfo)

	for _, scene := range scenes {
		if scene.Location == nil || scene.Time == nil {
			continue
		}

		// Use location + time as the unique identifier
		key := *scene.Location + "|" + *scene.Time

		if bg, exists := backgroundMap[key]; exists {
			// Background already exists, add the scene ID
			bg.SceneIDs = append(bg.SceneIDs, scene.ID)
			bg.StoryboardCount++
		} else {
			// New background - build background prompt using ImagePrompt
			prompt := ""
			if scene.ImagePrompt != nil {
				prompt = *scene.ImagePrompt
			}
			backgroundMap[key] = &BackgroundInfo{
				Location:        *scene.Location,
				Time:            *scene.Time,
				Prompt:          prompt,
				SceneIDs:        []uint{scene.ID},
				StoryboardCount: 1,
			}
		}
	}

	// Convert to slice
	var backgrounds []BackgroundInfo
	for _, bg := range backgroundMap {
		backgrounds = append(backgrounds, *bg)
	}

	return backgrounds
}

// loadImageAsBase64 reads a local image file and converts it to a base64-encoded data URI
func (s *ImageGenerationService) loadImageAsBase64(localPath string) (string, error) {
	// Build the full file path
	var fullPath string
	if filepath.IsAbs(localPath) {
		fullPath = localPath
	} else {
		// If it's a relative path, join with the storage root directory
		if s.localStorage != nil {
			fullPath = s.localStorage.GetAbsolutePath(localPath)
		} else {
			fullPath = filepath.Join(s.config.Storage.LocalPath, localPath)
		}
	}

	// Read the file
	fileData, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read image file: %w", err)
	}

	// Determine the MIME type based on file extension
	ext := strings.ToLower(filepath.Ext(fullPath))
	mimeType := "image/jpeg" // Default
	switch ext {
	case ".png":
		mimeType = "image/png"
	case ".jpg", ".jpeg":
		mimeType = "image/jpeg"
	case ".gif":
		mimeType = "image/gif"
	case ".webp":
		mimeType = "image/webp"
	}

	// Convert to base64
	base64Data := base64.StdEncoding.EncodeToString(fileData)

	// Build the data URI
	dataURI := fmt.Sprintf("data:%s;base64,%s", mimeType, base64Data)

	return dataURI, nil
}
