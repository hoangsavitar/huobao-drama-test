package services

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	models "github.com/drama-generator/backend/domain/models"
	"github.com/drama-generator/backend/infrastructure/external/ffmpeg"
	"github.com/drama-generator/backend/infrastructure/storage"
	"github.com/drama-generator/backend/pkg/logger"
	"github.com/drama-generator/backend/pkg/utils"
	"github.com/drama-generator/backend/pkg/video"
	"gorm.io/gorm"
)

type VideoGenerationService struct {
	db              *gorm.DB
	transferService *ResourceTransferService
	log             *logger.Logger
	localStorage    *storage.LocalStorage
	aiService       *AIService
	ffmpeg          *ffmpeg.FFmpeg
	promptI18n      *PromptI18n
}

func NewVideoGenerationService(db *gorm.DB, transferService *ResourceTransferService, localStorage *storage.LocalStorage, aiService *AIService, log *logger.Logger, promptI18n *PromptI18n) *VideoGenerationService {
	service := &VideoGenerationService{
		db:              db,
		localStorage:    localStorage,
		transferService: transferService,
		aiService:       aiService,
		log:             log,
		ffmpeg:          ffmpeg.NewFFmpeg(log),
		promptI18n:      promptI18n,
	}

	go service.RecoverPendingTasks()

	return service
}

type GenerateVideoRequest struct {
	StoryboardID *uint  `json:"storyboard_id"`
	DramaID      string `json:"drama_id" binding:"required"`
	ImageGenID   *uint  `json:"image_gen_id"`

	// Reference image mode: single, first_last, multiple, none
	ReferenceMode string `json:"reference_mode"`

	// Single image mode
	ImageURL       string  `json:"image_url"`
	ImageLocalPath *string `json:"image_local_path"` // Single image mode local path

	// First-last frame mode
	FirstFrameURL       *string `json:"first_frame_url"`
	FirstFrameLocalPath *string `json:"first_frame_local_path"` // First frame local path
	LastFrameURL        *string `json:"last_frame_url"`
	LastFrameLocalPath  *string `json:"last_frame_local_path"` // Last frame local path

	// Multiple image mode
	ReferenceImageURLs []string `json:"reference_image_urls"`

	Prompt       string  `json:"prompt" binding:"required,min=5,max=2000"`
	Provider     string  `json:"provider"`
	Model        string  `json:"model"`
	Duration     *int    `json:"duration"`
	FPS          *int    `json:"fps"`
	AspectRatio  *string `json:"aspect_ratio"`
	Style        *string `json:"style"`
	MotionLevel  *int    `json:"motion_level"`
	CameraMotion *string `json:"camera_motion"`
	Seed         *int64  `json:"seed"`
}

func (s *VideoGenerationService) GenerateVideo(request *GenerateVideoRequest) (*models.VideoGeneration, error) {
	if request.StoryboardID != nil {
		var storyboard models.Storyboard
		if err := s.db.Preload("Episode").Where("id = ?", *request.StoryboardID).First(&storyboard).Error; err != nil {
			return nil, fmt.Errorf("storyboard not found")
		}
		if fmt.Sprintf("%d", storyboard.Episode.DramaID) != request.DramaID {
			return nil, fmt.Errorf("storyboard does not belong to drama")
		}
	}

	if request.ImageGenID != nil {
		var imageGen models.ImageGeneration
		if err := s.db.Where("id = ?", *request.ImageGenID).First(&imageGen).Error; err != nil {
			return nil, fmt.Errorf("image generation not found")
		}
	}

	provider := request.Provider
	if provider == "" {
		provider = "doubao"
	}

	dramaID, _ := strconv.ParseUint(request.DramaID, 10, 32)

	videoGen := &models.VideoGeneration{
		StoryboardID: request.StoryboardID,
		DramaID:      uint(dramaID),
		ImageGenID:   request.ImageGenID,
		Provider:     provider,
		Prompt:       request.Prompt,
		Model:        request.Model,
		Duration:     request.Duration,
		FPS:          request.FPS,
		AspectRatio:  request.AspectRatio,
		Style:        request.Style,
		MotionLevel:  request.MotionLevel,
		CameraMotion: request.CameraMotion,
		Seed:         request.Seed,
		Status:       models.VideoStatusPending,
	}

	// Handle different parameters based on reference image mode
	if request.ReferenceMode != "" {
		videoGen.ReferenceMode = &request.ReferenceMode
	}

	switch request.ReferenceMode {
	case "single":
		// Single image mode - prefer local_path
		if request.ImageLocalPath != nil && *request.ImageLocalPath != "" {
			videoGen.ImageURL = request.ImageLocalPath
		} else if request.ImageURL != "" {
			videoGen.ImageURL = &request.ImageURL
		}
	case "first_last":
		// First-last frame mode - prefer local_path
		if request.FirstFrameLocalPath != nil && *request.FirstFrameLocalPath != "" {
			videoGen.FirstFrameURL = request.FirstFrameLocalPath
		} else if request.FirstFrameURL != nil {
			videoGen.FirstFrameURL = request.FirstFrameURL
		}
		if request.LastFrameLocalPath != nil && *request.LastFrameLocalPath != "" {
			videoGen.LastFrameURL = request.LastFrameLocalPath
		} else if request.LastFrameURL != nil {
			videoGen.LastFrameURL = request.LastFrameURL
		}
	case "multiple":
		// Multiple image mode
		if len(request.ReferenceImageURLs) > 0 {
			referenceImagesJSON, err := json.Marshal(request.ReferenceImageURLs)
			if err == nil {
				referenceImagesStr := string(referenceImagesJSON)
				videoGen.ReferenceImageURLs = &referenceImagesStr
			}
		}
	case "none":
		// No reference image, pure text generation
	default:
		// Backward compatible: auto-detect mode from provided parameters
		if request.ImageURL != "" {
			videoGen.ImageURL = &request.ImageURL
			mode := "single"
			videoGen.ReferenceMode = &mode
		} else if request.FirstFrameURL != nil || request.LastFrameURL != nil {
			videoGen.FirstFrameURL = request.FirstFrameURL
			videoGen.LastFrameURL = request.LastFrameURL
			mode := "first_last"
			videoGen.ReferenceMode = &mode
		} else if len(request.ReferenceImageURLs) > 0 {
			referenceImagesJSON, err := json.Marshal(request.ReferenceImageURLs)
			if err == nil {
				referenceImagesStr := string(referenceImagesJSON)
				videoGen.ReferenceImageURLs = &referenceImagesStr
				mode := "multiple"
				videoGen.ReferenceMode = &mode
			}
		}
	}

	if err := s.db.Create(videoGen).Error; err != nil {
		return nil, fmt.Errorf("failed to create record: %w", err)
	}

	// Start background goroutine to process video generation asynchronously
	// This allows the API to return immediately while video generation happens in background
	// CRITICAL: The goroutine will handle all video generation logic including API calls and polling
	go s.ProcessVideoGeneration(videoGen.ID)

	return videoGen, nil
}

func (s *VideoGenerationService) ProcessVideoGeneration(videoGenID uint) {
	var videoGen models.VideoGeneration
	if err := s.db.First(&videoGen, videoGenID).Error; err != nil {
		s.log.Errorw("Failed to load video generation", "error", err, "id", videoGenID)
		return
	}

	// Get drama style information
	var drama models.Drama
	if err := s.db.First(&drama, videoGen.DramaID).Error; err != nil {
		s.log.Warnw("Failed to load drama for style", "error", err, "drama_id", videoGen.DramaID)
	}

	s.db.Model(&videoGen).Update("status", models.VideoStatusProcessing)

	client, err := s.getVideoClient(videoGen.Provider, videoGen.Model)
	if err != nil {
		s.log.Errorw("Failed to get video client", "error", err, "provider", videoGen.Provider, "model", videoGen.Model)
		s.updateVideoGenError(videoGenID, err.Error())
		return
	}

	s.log.Infow("Starting video generation", "id", videoGenID, "prompt", videoGen.Prompt, "provider", videoGen.Provider)

	var opts []video.VideoOption
	if videoGen.Model != "" {
		opts = append(opts, video.WithModel(videoGen.Model))
	}
	if videoGen.Duration != nil {
		opts = append(opts, video.WithDuration(*videoGen.Duration))
	}
	if videoGen.FPS != nil {
		opts = append(opts, video.WithFPS(*videoGen.FPS))
	}
	if videoGen.AspectRatio != nil {
		opts = append(opts, video.WithAspectRatio(*videoGen.AspectRatio))
	}
	if videoGen.Style != nil {
		opts = append(opts, video.WithStyle(*videoGen.Style))
	}
	if videoGen.MotionLevel != nil {
		opts = append(opts, video.WithMotionLevel(*videoGen.MotionLevel))
	}
	if videoGen.CameraMotion != nil {
		opts = append(opts, video.WithCameraMotion(*videoGen.CameraMotion))
	}
	if videoGen.Seed != nil {
		opts = append(opts, video.WithSeed(*videoGen.Seed))
	}

	// Add options based on reference image mode, converting local images to base64
	if videoGen.ReferenceMode != nil {
		switch *videoGen.ReferenceMode {
		case "first_last":
			// First-last frame mode - convert local images to base64
			if videoGen.FirstFrameURL != nil {
				firstFrameBase64, err := s.convertImageToBase64(*videoGen.FirstFrameURL)
				if err != nil {
					s.log.Warnw("Failed to convert first frame to base64, using original URL", "error", err)
					opts = append(opts, video.WithFirstFrame(*videoGen.FirstFrameURL))
				} else {
					opts = append(opts, video.WithFirstFrame(firstFrameBase64))
				}
			}
			if videoGen.LastFrameURL != nil {
				lastFrameBase64, err := s.convertImageToBase64(*videoGen.LastFrameURL)
				if err != nil {
					s.log.Warnw("Failed to convert last frame to base64, using original URL", "error", err)
					opts = append(opts, video.WithLastFrame(*videoGen.LastFrameURL))
				} else {
					opts = append(opts, video.WithLastFrame(lastFrameBase64))
				}
			}
		case "multiple":
			// Multiple image mode - convert local images to base64
			if videoGen.ReferenceImageURLs != nil {
				var imageURLs []string
				if err := json.Unmarshal([]byte(*videoGen.ReferenceImageURLs), &imageURLs); err == nil {
					var base64Images []string
					for _, imgURL := range imageURLs {
						base64Img, err := s.convertImageToBase64(imgURL)
						if err != nil {
							s.log.Warnw("Failed to convert reference image to base64, using original URL", "error", err, "url", imgURL)
							base64Images = append(base64Images, imgURL)
						} else {
							base64Images = append(base64Images, base64Img)
						}
					}
					opts = append(opts, video.WithReferenceImages(base64Images))
				}
			}
		}
	}

	// Build imageURL parameter (used for single image mode, empty for others)
	// If local image, convert to base64
	imageURL := ""
	if videoGen.ImageURL != nil {
		base64Image, err := s.convertImageToBase64(*videoGen.ImageURL)
		if err != nil {
			s.log.Warnw("Failed to convert image to base64, using original URL", "error", err)
			imageURL = *videoGen.ImageURL
		} else {
			imageURL = base64Image
		}
	}

	// Build complete prompt: style prompt + constraint prompt + user prompt
	prompt := videoGen.Prompt

	// 2. Add video constraint prompt
	// Select constraint prompt based on reference image mode
	referenceMode := "single" // Default single image mode
	if videoGen.ReferenceMode != nil {
		referenceMode = *videoGen.ReferenceMode
	}

	// For single image mode, check if the image is an action sequence
	if referenceMode == "single" && videoGen.ImageGenID != nil {
		var imageGen models.ImageGeneration
		if err := s.db.First(&imageGen, *videoGen.ImageGenID).Error; err == nil {
			// If image frame_type is action, use action sequence constraint prompt
			if imageGen.FrameType != nil && *imageGen.FrameType == "action" {
				referenceMode = "action_sequence"
				s.log.Infow("Detected action sequence image in single mode",
					"id", videoGenID,
					"image_gen_id", *videoGen.ImageGenID,
					"frame_type", *imageGen.FrameType)
			}
		}
	}

	constraintPrompt := s.promptI18n.GetVideoConstraintPrompt(referenceMode)
	if constraintPrompt != "" {
		prompt = constraintPrompt + "\n\n" + prompt
		s.log.Infow("Added constraint prompt to video generation",
			"id", videoGenID,
			"reference_mode", referenceMode,
			"constraint_prompt_length", len(constraintPrompt))
	}

	// Log complete prompt information
	s.log.Infow("Video generation prompts",
		"id", videoGenID,
		"user_prompt", videoGen.Prompt,
		"constraint_prompt", constraintPrompt,
		"final_prompt", prompt)

	result, err := client.GenerateVideo(imageURL, prompt, opts...)
	if err != nil {
		s.log.Errorw("Video generation API call failed", "error", err, "id", videoGenID)
		s.updateVideoGenError(videoGenID, err.Error())
		return
	}

	// CRITICAL FIX: Validate TaskID before starting polling goroutine
	// Empty TaskID would cause polling to fail silently or cause issues
	if result.TaskID != "" {
		s.db.Model(&videoGen).Updates(map[string]interface{}{
			"task_id": result.TaskID,
			"status":  models.VideoStatusProcessing,
		})
		// Start background goroutine to poll task status
		// This allows the API to return immediately while video generation continues asynchronously
		// The goroutine will poll until completion, failure, or timeout (max 300 attempts * 10s = 50 minutes)
		go s.pollTaskStatus(videoGenID, result.TaskID, videoGen.Provider, videoGen.Model)
		return
	}

	if result.VideoURL != "" {
		s.completeVideoGeneration(videoGenID, result.VideoURL, &result.Duration, &result.Width, &result.Height, nil)
		return
	}

	s.updateVideoGenError(videoGenID, "no task ID or video URL returned")
}

func (s *VideoGenerationService) pollTaskStatus(videoGenID uint, taskID string, provider string, model string) {
	// CRITICAL FIX: Validate taskID parameter to prevent invalid API calls
	// Empty taskID would cause unnecessary API calls and potential errors
	if taskID == "" {
		s.log.Errorw("Invalid empty taskID for polling", "video_gen_id", videoGenID)
		s.updateVideoGenError(videoGenID, "invalid task ID for polling")
		return
	}

	client, err := s.getVideoClient(provider, model)
	if err != nil {
		s.log.Errorw("Failed to get video client for polling", "error", err)
		s.updateVideoGenError(videoGenID, "failed to get video client")
		return
	}

	// Polling configuration: max 300 attempts with 10 second intervals
	// Total maximum polling time: 300 * 10s = 50 minutes
	// This prevents infinite polling if the task never completes
	maxAttempts := 300
	interval := 10 * time.Second

	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Sleep before each poll attempt to avoid overwhelming the API
		// First iteration sleeps before the first check (after 0 attempts)
		time.Sleep(interval)

		var videoGen models.VideoGeneration
		if err := s.db.First(&videoGen, videoGenID).Error; err != nil {
			s.log.Errorw("Failed to load video generation", "error", err, "id", videoGenID)
			return
		}

		// CRITICAL FIX: Check if status was manually changed (e.g., cancelled by user)
		// If status is no longer "processing", stop polling to avoid unnecessary API calls
		// This prevents polling when the task has been cancelled or failed externally
		if videoGen.Status != models.VideoStatusProcessing {
			s.log.Infow("Video generation status changed, stopping poll", "id", videoGenID, "status", videoGen.Status)
			return
		}

		// Poll the video generation API for task status
		// Continue polling on transient errors (network issues, temporary API failures)
		// Only stop on permanent errors or task completion
		result, err := client.GetTaskStatus(taskID)
		if err != nil {
			s.log.Errorw("Failed to get task status", "error", err, "task_id", taskID, "attempt", attempt+1)
			// Continue polling on error - might be transient network issue
			// Will eventually timeout after maxAttempts if error persists
			continue
		}

		// Check if task completed successfully
		// CRITICAL FIX: Validate that video URL exists when task is marked as completed
		// Some APIs may mark task as completed but fail to provide the video URL
		if result.Completed {
			if result.VideoURL != "" {
				// Successfully completed with video URL - download and update database
				s.completeVideoGeneration(videoGenID, result.VideoURL, &result.Duration, &result.Width, &result.Height, nil)
				return
			}
			// Task marked as completed but no video URL - this is an error condition
			s.updateVideoGenError(videoGenID, "task completed but no video URL")
			return
		}

		// Check if task failed with an error message
		if result.Error != "" {
			s.updateVideoGenError(videoGenID, result.Error)
			return
		}

		// Task still in progress - log and continue polling
		s.log.Infow("Video generation in progress", "id", videoGenID, "attempt", attempt+1, "max_attempts", maxAttempts)
	}

	// CRITICAL FIX: Handle polling timeout gracefully
	// After maxAttempts (50 minutes), mark task as failed if still not completed
	// This prevents indefinite polling and resource waste
	s.updateVideoGenError(videoGenID, fmt.Sprintf("polling timeout after %d attempts (%.1f minutes)", maxAttempts, float64(maxAttempts*int(interval))/60.0))
}

func (s *VideoGenerationService) completeVideoGeneration(videoGenID uint, videoURL string, duration *int, width *int, height *int, firstFrameURL *string) {
	var localVideoPath *string

	// Download video to local storage and save relative path to database
	if s.localStorage != nil && videoURL != "" {
		downloadResult, err := s.localStorage.DownloadFromURLWithPath(videoURL, "videos")
		if err != nil {
			s.log.Warnw("Failed to download video to local storage",
				"error", err,
				"id", videoGenID,
				"original_url", videoURL)
		} else {
			localVideoPath = &downloadResult.RelativePath
			s.log.Infow("Video downloaded to local storage",
				"id", videoGenID,
				"original_url", videoURL,
				"local_path", downloadResult.RelativePath)
		}
	}

	// If video downloaded locally, probe actual duration
	// Especially when AI service returns duration of 0 or nil, probing is required
	shouldProbe := localVideoPath != nil && s.ffmpeg != nil && (duration == nil || *duration == 0)
	if shouldProbe {
		absPath := s.localStorage.GetAbsolutePath(*localVideoPath)
		if probedDuration, err := s.ffmpeg.GetVideoDuration(absPath); err == nil {
			// Convert to integer seconds (round up)
			durationInt := int(probedDuration + 0.5)
			duration = &durationInt
			s.log.Infow("Probed video duration (was 0 or nil)",
				"id", videoGenID,
				"duration_seconds", durationInt,
				"duration_float", probedDuration)
		} else {
			s.log.Errorw("Failed to probe video duration, duration will be 0",
				"error", err,
				"id", videoGenID,
				"local_path", *localVideoPath)
		}
	} else if localVideoPath != nil && s.ffmpeg != nil && duration != nil && *duration > 0 {
		// Even with duration, validate it (optional)
		absPath := s.localStorage.GetAbsolutePath(*localVideoPath)
		if probedDuration, err := s.ffmpeg.GetVideoDuration(absPath); err == nil {
			durationInt := int(probedDuration + 0.5)
			if durationInt != *duration {
				s.log.Warnw("Probed duration differs from provided duration",
					"id", videoGenID,
					"provided", *duration,
					"probed", durationInt)
				// Use probed duration (more accurate)
				duration = &durationInt
			}
		}
	}

	// Download first frame to local storage (for caching only, no DB update)
	if firstFrameURL != nil && *firstFrameURL != "" && s.localStorage != nil {
		_, err := s.localStorage.DownloadFromURL(*firstFrameURL, "video_frames")
		if err != nil {
			s.log.Warnw("Failed to download first frame to local storage",
				"error", err,
				"id", videoGenID,
				"original_url", *firstFrameURL)
		} else {
			s.log.Infow("First frame downloaded to local storage for caching",
				"id", videoGenID,
				"original_url", *firstFrameURL)
		}
	}

	// Save original URL and local path in database
	updates := map[string]interface{}{
		"status":     models.VideoStatusCompleted,
		"video_url":  videoURL,
		"local_path": localVideoPath,
	}
	// Only save duration when > 0 to avoid invalid zero values
	if duration != nil && *duration > 0 {
		updates["duration"] = *duration
	}
	if width != nil {
		updates["width"] = *width
	}
	if height != nil {
		updates["height"] = *height
	}
	if firstFrameURL != nil {
		updates["first_frame_url"] = *firstFrameURL
	}

	if err := s.db.Model(&models.VideoGeneration{}).Where("id = ?", videoGenID).Updates(updates).Error; err != nil {
		s.log.Errorw("Failed to update video generation", "error", err, "id", videoGenID)
		return
	}

	var videoGen models.VideoGeneration
	if err := s.db.First(&videoGen, videoGenID).Error; err == nil {
		if videoGen.StoryboardID != nil {
			// Update Storyboard video_url and duration
			storyboardUpdates := map[string]interface{}{
				"video_url": videoURL,
			}
			// Only update duration when > 0 to avoid overwriting with invalid zero
			if duration != nil && *duration > 0 {
				storyboardUpdates["duration"] = *duration
			}
			if err := s.db.Model(&models.Storyboard{}).Where("id = ?", *videoGen.StoryboardID).Updates(storyboardUpdates).Error; err != nil {
				s.log.Warnw("Failed to update storyboard", "storyboard_id", *videoGen.StoryboardID, "error", err)
			} else {
				s.log.Infow("Updated storyboard with video info", "storyboard_id", *videoGen.StoryboardID, "duration", duration)
			}
		}
	}

	s.log.Infow("Video generation completed", "id", videoGenID, "url", videoURL, "duration", duration)
}

func (s *VideoGenerationService) updateVideoGenError(videoGenID uint, errorMsg string) {
	if err := s.db.Model(&models.VideoGeneration{}).Where("id = ?", videoGenID).Updates(map[string]interface{}{
		"status":    models.VideoStatusFailed,
		"error_msg": errorMsg,
	}).Error; err != nil {
		s.log.Errorw("Failed to update video generation error", "error", err, "id", videoGenID)
	}
}

func (s *VideoGenerationService) getVideoClient(provider string, modelName string) (video.VideoClient, error) {
	// Get AI config by model name
	var config *models.AIServiceConfig
	var err error

	if modelName != "" {
		config, err = s.aiService.GetConfigForModel("video", modelName)
		if err != nil {
			s.log.Warnw("Failed to get config for model, using default", "model", modelName, "error", err)
			config, err = s.aiService.GetDefaultConfig("video")
			if err != nil {
				return nil, fmt.Errorf("no video AI config found: %w", err)
			}
		}
	} else {
		config, err = s.aiService.GetDefaultConfig("video")
		if err != nil {
			return nil, fmt.Errorf("no video AI config found: %w", err)
		}
	}

	// Create client using config information
	baseURL := config.BaseURL
	apiKey := config.APIKey
	model := modelName
	if model == "" && len(config.Model) > 0 {
		model = config.Model[0]
	}

	// Create corresponding client based on config provider
	var endpoint string
	var queryEndpoint string

	switch config.Provider {
	case "chatfire":
		endpoint = "/video/generations"
		queryEndpoint = "/video/task/{taskId}"
		return video.NewChatfireClient(baseURL, apiKey, model, endpoint, queryEndpoint), nil
	case "doubao", "volcengine", "volces":
		endpoint = "/contents/generations/tasks"
		queryEndpoint = "/contents/generations/tasks/{taskId}"
		return video.NewVolcesArkClient(baseURL, apiKey, model, endpoint, queryEndpoint), nil
	case "openai":
		// OpenAI Sora uses /v1/videos endpoint
		return video.NewOpenAISoraClient(baseURL, apiKey, model), nil
	case "runway":
		return video.NewRunwayClient(baseURL, apiKey, model), nil
	case "pika":
		return video.NewPikaClient(baseURL, apiKey, model), nil
	case "minimax":
		return video.NewMinimaxClient(baseURL, apiKey, model), nil
	case "ltx_proxy", "proxy":
		return video.NewProxyClient(baseURL, apiKey), nil
	default:
		return nil, fmt.Errorf("unsupported video provider: %s", config.Provider)
	}
}

func (s *VideoGenerationService) RecoverPendingTasks() {
	var pendingVideos []models.VideoGeneration
	// Query for pending tasks with non-empty task_id
	// Note: Using IS NOT NULL and != '' to ensure we only get valid task IDs
	if err := s.db.Where("status = ? AND task_id IS NOT NULL AND task_id != ''", models.VideoStatusProcessing).Find(&pendingVideos).Error; err != nil {
		s.log.Errorw("Failed to load pending video tasks", "error", err)
		return
	}

	s.log.Infow("Recovering pending video generation tasks", "count", len(pendingVideos))

	for _, videoGen := range pendingVideos {
		// CRITICAL FIX: Check for nil TaskID before dereferencing to prevent panic
		// Even though we filter for non-empty task_id, GORM might still return nil pointers
		// This nil check prevents a potential runtime panic
		if videoGen.TaskID == nil || *videoGen.TaskID == "" {
			s.log.Warnw("Skipping video generation with nil or empty TaskID", "id", videoGen.ID)
			continue
		}

		// Start goroutine to poll task status for each pending video
		// Each goroutine will poll independently until completion or timeout
		go s.pollTaskStatus(videoGen.ID, *videoGen.TaskID, videoGen.Provider, videoGen.Model)
	}
}

func (s *VideoGenerationService) GetVideoGeneration(id uint) (*models.VideoGeneration, error) {
	var videoGen models.VideoGeneration
	if err := s.db.First(&videoGen, id).Error; err != nil {
		return nil, err
	}
	return &videoGen, nil
}

func (s *VideoGenerationService) ListVideoGenerations(dramaID *uint, storyboardID *uint, status string, limit int, offset int) ([]*models.VideoGeneration, int64, error) {
	var videos []*models.VideoGeneration
	var total int64

	query := s.db.Model(&models.VideoGeneration{})

	if dramaID != nil {
		query = query.Where("drama_id = ?", *dramaID)
	}
	if storyboardID != nil {
		query = query.Where("storyboard_id = ?", *storyboardID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&videos).Error; err != nil {
		return nil, 0, err
	}

	return videos, total, nil
}

func (s *VideoGenerationService) GenerateVideoFromImage(imageGenID uint) (*models.VideoGeneration, error) {
	var imageGen models.ImageGeneration
	if err := s.db.First(&imageGen, imageGenID).Error; err != nil {
		return nil, fmt.Errorf("image generation not found")
	}

	if imageGen.Status != models.ImageStatusCompleted || imageGen.ImageURL == nil {
		return nil, fmt.Errorf("image is not ready")
	}

	// Get associated Storyboard for duration
	var duration *int
	if imageGen.StoryboardID != nil {
		var storyboard models.Storyboard
		if err := s.db.Where("id = ?", *imageGen.StoryboardID).First(&storyboard).Error; err == nil {
			duration = &storyboard.Duration
			s.log.Infow("Using storyboard duration for video generation",
				"storyboard_id", *imageGen.StoryboardID,
				"duration", storyboard.Duration)
		}
	}

	req := &GenerateVideoRequest{
		DramaID:      fmt.Sprintf("%d", imageGen.DramaID),
		StoryboardID: imageGen.StoryboardID,
		ImageGenID:   &imageGenID,
		ImageURL:     *imageGen.ImageURL,
		Prompt:       imageGen.Prompt,
		Provider:     "doubao",
		Duration:     duration,
	}

	return s.GenerateVideo(req)
}

func (s *VideoGenerationService) BatchGenerateVideosForEpisode(episodeID string) ([]*models.VideoGeneration, error) {
	var episode models.Episode
	if err := s.db.Preload("Storyboards").Where("id = ?", episodeID).First(&episode).Error; err != nil {
		return nil, fmt.Errorf("episode not found")
	}

	var results []*models.VideoGeneration
	for _, storyboard := range episode.Storyboards {
		if storyboard.ImagePrompt == nil {
			continue
		}

		var imageGen models.ImageGeneration
		if err := s.db.Where("storyboard_id = ? AND status = ?", storyboard.ID, models.ImageStatusCompleted).
			Order("created_at DESC").First(&imageGen).Error; err != nil {
			s.log.Warnw("No completed image for storyboard", "storyboard_id", storyboard.ID)
			continue
		}

		videoGen, err := s.GenerateVideoFromImage(imageGen.ID)
		if err != nil {
			s.log.Errorw("Failed to generate video", "storyboard_id", storyboard.ID, "error", err)
			continue
		}

		results = append(results, videoGen)
	}

	return results, nil
}

func (s *VideoGenerationService) DeleteVideoGeneration(id uint) error {
	return s.db.Delete(&models.VideoGeneration{}, id).Error
}

// convertImageToBase64 converts an image to base64 format
// Prefers locally stored images, falls back to URL
func (s *VideoGenerationService) convertImageToBase64(imageURL string) (string, error) {
	// If already base64 format, return directly
	if strings.HasPrefix(imageURL, "data:") {
		return imageURL, nil
	}

	// Try reading from local storage
	if s.localStorage != nil {
		var relativePath string

		// 1. Check if it is a local URL (contains /static/)
		if strings.Contains(imageURL, "/static/") {
			parts := strings.Split(imageURL, "/static/")
			if len(parts) == 2 {
				relativePath = parts[1]
			}
		} else if !strings.HasPrefix(imageURL, "http://") && !strings.HasPrefix(imageURL, "https://") {
			// 2. If not HTTP/HTTPS URL, treat as relative path (e.g. "images/xxx.jpg")
			relativePath = imageURL
		}

		// If relative path identified, try reading local file
		if relativePath != "" {
			absPath := s.localStorage.GetAbsolutePath(relativePath)

			// Use utility function to convert to base64
			base64Str, err := utils.ImageToBase64(absPath)
			if err == nil {
				s.log.Infow("Converted local image to base64", "path", relativePath)
				return base64Str, nil
			}
			s.log.Warnw("Failed to convert local image to base64, will try URL", "error", err, "path", absPath)
		}
	}

	// If local read failed or not a local path, try downloading from URL and converting
	base64Str, err := utils.ImageToBase64(imageURL)
	if err != nil {
		return "", fmt.Errorf("failed to convert image to base64: %w", err)
	}

	urlLen := len(imageURL)
	if urlLen > 50 {
		urlLen = 50
	}
	s.log.Infow("Converted remote image to base64", "url", imageURL[:urlLen])
	return base64Str, nil
}
