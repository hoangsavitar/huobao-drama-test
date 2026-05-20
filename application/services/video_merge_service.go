package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	models "github.com/drama-generator/backend/domain/models"
	"github.com/drama-generator/backend/infrastructure/external/ffmpeg"
	"github.com/drama-generator/backend/pkg/logger"
	"github.com/drama-generator/backend/pkg/video"
	"gorm.io/gorm"
)

type VideoMergeService struct {
	db              *gorm.DB
	aiService       *AIService
	transferService *ResourceTransferService
	ffmpeg          *ffmpeg.FFmpeg
	storagePath     string
	baseURL         string
	log             *logger.Logger
}

func NewVideoMergeService(db *gorm.DB, transferService *ResourceTransferService, storagePath, baseURL string, log *logger.Logger) *VideoMergeService {
	return &VideoMergeService{
		db:              db,
		aiService:       NewAIService(db, log),
		transferService: transferService,
		ffmpeg:          ffmpeg.NewFFmpeg(log),
		storagePath:     storagePath,
		baseURL:         baseURL,
		log:             log,
	}
}

type MergeVideoRequest struct {
	EpisodeID string             `json:"episode_id" binding:"required"`
	DramaID   string             `json:"drama_id" binding:"required"`
	Title     string             `json:"title"`
	Scenes    []models.SceneClip `json:"scenes" binding:"required,min=1"`
	Provider  string             `json:"provider"`
	Model     string             `json:"model"`
}

func (s *VideoMergeService) MergeVideos(req *MergeVideoRequest) (*models.VideoMerge, error) {
	// Verify episode access
	var episode models.Episode
	if err := s.db.Preload("Drama").Where("id = ?", req.EpisodeID).First(&episode).Error; err != nil {
		return nil, fmt.Errorf("episode not found")
	}

	// Verify all scenes have videos
	for i, scene := range req.Scenes {
		if scene.VideoURL == "" {
			return nil, fmt.Errorf("scene %d has no video", i+1)
		}
	}

	provider := req.Provider
	if provider == "" {
		provider = "doubao"
	}

	// Serialize scene list
	scenesJSON, err := json.Marshal(req.Scenes)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize scenes: %w", err)
	}

	s.log.Infow("Serialized scenes to JSON",
		"scenes_count", len(req.Scenes),
		"scenes_json", string(scenesJSON))

	epID, _ := strconv.ParseUint(req.EpisodeID, 10, 32)
	dramaID, _ := strconv.ParseUint(req.DramaID, 10, 32)

	videoMerge := &models.VideoMerge{
		EpisodeID: uint(epID),
		DramaID:   uint(dramaID),
		Title:     req.Title,
		Provider:  provider,
		Model:     &req.Model,
		Scenes:    scenesJSON,
		Status:    models.VideoMergeStatusPending,
	}

	if err := s.db.Create(videoMerge).Error; err != nil {
		return nil, fmt.Errorf("failed to create merge record: %w", err)
	}

	go s.processMergeVideo(videoMerge.ID)

	return videoMerge, nil
}

func (s *VideoMergeService) processMergeVideo(mergeID uint) {
	var videoMerge models.VideoMerge
	if err := s.db.First(&videoMerge, mergeID).Error; err != nil {
		s.log.Errorw("Failed to load video merge", "error", err, "id", mergeID)
		return
	}

	s.db.Model(&videoMerge).Update("status", models.VideoMergeStatusProcessing)

	client, err := s.getVideoClient(videoMerge.Provider)
	if err != nil {
		s.updateMergeError(mergeID, err.Error())
		return
	}

	// Parse scene list
	var scenes []models.SceneClip
	if err := json.Unmarshal(videoMerge.Scenes, &scenes); err != nil {
		s.updateMergeError(mergeID, fmt.Sprintf("failed to parse scenes: %v", err))
		return
	}

	// Call video merge API
	result, err := s.mergeVideoClips(client, scenes)
	if err != nil {
		s.updateMergeError(mergeID, err.Error())
		return
	}

	if !result.Completed {
		s.db.Model(&videoMerge).Updates(map[string]interface{}{
			"status":  models.VideoMergeStatusProcessing,
			"task_id": result.TaskID,
		})
		go s.pollMergeStatus(mergeID, client, result.TaskID)
		return
	}

	s.completeMerge(mergeID, result)
}

func (s *VideoMergeService) mergeVideoClips(client video.VideoClient, scenes []models.SceneClip) (*video.VideoResult, error) {
	if len(scenes) == 0 {
		return nil, fmt.Errorf("no scenes to merge")
	}

	// Sort scenes by Order field
	sort.Slice(scenes, func(i, j int) bool {
		return scenes[i].Order < scenes[j].Order
	})

	s.log.Infow("Merging video clips with FFmpeg", "scene_count", len(scenes))

	// Calculate total duration
	var totalDuration float64
	for _, scene := range scenes {
		totalDuration += scene.Duration
	}

	// Prepare FFmpeg merge options
	clips := make([]ffmpeg.VideoClip, len(scenes))
	for i, scene := range scenes {
		// Use scene.VideoURL, which has been properly handled in the preceding code
		// If it's a local file, it already contains the full path (storagePath + LocalPath)
		// If it's an HTTP URL, use it directly
		videoPath := scene.VideoURL

		clips[i] = ffmpeg.VideoClip{
			URL:        videoPath,
			Duration:   scene.Duration,
			StartTime:  scene.StartTime,
			EndTime:    scene.EndTime,
			Transition: scene.Transition,
		}

		s.log.Infow("Clip added to merge queue",
			"order", scene.Order,
			"index", i,
			"video_path", videoPath,
			"duration", scene.Duration,
			"start_time", scene.StartTime,
			"end_time", scene.EndTime)
	}

	// Create video output directory
	videoDir := filepath.Join(s.storagePath, "videos", "merged")
	if err := os.MkdirAll(videoDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create video directory: %w", err)
	}

	// Generate output file name
	fileName := fmt.Sprintf("merged_%d.mp4", time.Now().Unix())
	outputPath := filepath.Join(videoDir, fileName)

	// Use FFmpeg to merge videos
	mergedPath, err := s.ffmpeg.MergeVideos(&ffmpeg.MergeOptions{
		OutputPath: outputPath,
		Clips:      clips,
	})
	if err != nil {
		return nil, fmt.Errorf("ffmpeg merge failed: %w", err)
	}

	s.log.Infow("Video merged successfully", "path", mergedPath)

	// Generate relative path (without protocol, IP, port)
	relPath := filepath.Join("videos", "merged", fileName)

	result := &video.VideoResult{
		VideoURL:  relPath, // Save only the relative path
		Duration:  int(totalDuration),
		Completed: true,
		Status:    "completed",
	}

	return result, nil
}

func (s *VideoMergeService) pollMergeStatus(mergeID uint, client video.VideoClient, taskID string) {
	maxAttempts := 240
	pollInterval := 5 * time.Second

	for i := 0; i < maxAttempts; i++ {
		time.Sleep(pollInterval)

		result, err := client.GetTaskStatus(taskID)
		if err != nil {
			s.log.Errorw("Failed to get merge task status", "error", err, "task_id", taskID)
			continue
		}

		if result.Completed {
			s.completeMerge(mergeID, result)
			return
		}

		if result.Error != "" {
			s.updateMergeError(mergeID, result.Error)
			return
		}
	}

	s.updateMergeError(mergeID, "timeout: video merge took too long")
}

func (s *VideoMergeService) completeMerge(mergeID uint, result *video.VideoResult) {
	now := time.Now()

	// Get merge record
	var videoMerge models.VideoMerge
	if err := s.db.First(&videoMerge, mergeID).Error; err != nil {
		s.log.Errorw("Failed to load video merge for completion", "error", err, "id", mergeID)
		return
	}

	finalVideoURL := result.VideoURL

	// Use local storage, no longer using MinIO
	s.log.Infow("Video merge completed, using local storage", "merge_id", mergeID, "local_path", result.VideoURL)

	updates := map[string]interface{}{
		"status":       models.VideoMergeStatusCompleted,
		"merged_url":   finalVideoURL,
		"completed_at": now,
	}

	if result.Duration > 0 {
		updates["duration"] = result.Duration
	}

	s.db.Model(&models.VideoMerge{}).Where("id = ?", mergeID).Updates(updates)

	// Update episode status and final video URL
	if videoMerge.EpisodeID != 0 {
		s.db.Model(&models.Episode{}).Where("id = ?", videoMerge.EpisodeID).Updates(map[string]interface{}{
			"status":    "completed",
			"video_url": finalVideoURL,
		})
		s.log.Infow("Episode finalized", "episode_id", videoMerge.EpisodeID, "video_url", finalVideoURL)
	}

	s.log.Infow("Video merge completed", "id", mergeID, "url", finalVideoURL)
}

func (s *VideoMergeService) updateMergeError(mergeID uint, errorMsg string) {
	s.db.Model(&models.VideoMerge{}).Where("id = ?", mergeID).Updates(map[string]interface{}{
		"status":    models.VideoMergeStatusFailed,
		"error_msg": errorMsg,
	})
	s.log.Errorw("Video merge failed", "id", mergeID, "error", errorMsg)
}

func (s *VideoMergeService) getVideoClient(provider string) (video.VideoClient, error) {
	config, err := s.aiService.GetDefaultConfig("video")
	if err != nil {
		return nil, fmt.Errorf("failed to get video config: %w", err)
	}

	// Use the first model
	model := ""
	if len(config.Model) > 0 {
		model = config.Model[0]
	}

	// Create the corresponding client based on the provider in config
	var endpoint string
	var queryEndpoint string

	switch config.Provider {
	case "runway":
		return video.NewRunwayClient(config.BaseURL, config.APIKey, model), nil
	case "pika":
		return video.NewPikaClient(config.BaseURL, config.APIKey, model), nil
	case "openai", "sora":
		return video.NewOpenAISoraClient(config.BaseURL, config.APIKey, model), nil
	case "minimax":
		return video.NewMinimaxClient(config.BaseURL, config.APIKey, model), nil
	case "chatfire":
		endpoint = "/video/generations"
		queryEndpoint = "/video/task/{taskId}"
		return video.NewChatfireClient(config.BaseURL, config.APIKey, model, endpoint, queryEndpoint), nil
	case "doubao", "volces", "ark":
		endpoint = "/contents/generations/tasks"
		queryEndpoint = "/generations/tasks/{taskId}"
		return video.NewVolcesArkClient(config.BaseURL, config.APIKey, model, endpoint, queryEndpoint), nil
	default:
		endpoint = "/contents/generations/tasks"
		queryEndpoint = "/generations/tasks/{taskId}"
		return video.NewVolcesArkClient(config.BaseURL, config.APIKey, model, endpoint, queryEndpoint), nil
	}
}

func (s *VideoMergeService) GetMerge(mergeID uint) (*models.VideoMerge, error) {
	var merge models.VideoMerge
	if err := s.db.Where("id = ? ", mergeID).First(&merge).Error; err != nil {
		return nil, err
	}
	return &merge, nil
}

func (s *VideoMergeService) ListMerges(episodeID *string, status string, page, pageSize int) ([]models.VideoMerge, int64, error) {
	query := s.db.Model(&models.VideoMerge{})

	if episodeID != nil && *episodeID != "" {
		query = query.Where("episode_id = ?", *episodeID)
	}

	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var merges []models.VideoMerge
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&merges).Error; err != nil {
		return nil, 0, err
	}

	return merges, total, nil
}

func (s *VideoMergeService) DeleteMerge(mergeID uint) error {
	result := s.db.Where("id = ? ", mergeID).Delete(&models.VideoMerge{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("merge not found")
	}
	return nil
}

// TimelineClip represents timeline clip data
type TimelineClip struct {
	AssetID      interface{}            `json:"asset_id"`      // Asset library video ID (preferred, can be number or string)
	StoryboardID string                 `json:"storyboard_id"` // Storyboard ID (fallback)
	Order        int                    `json:"order"`
	StartTime    float64                `json:"start_time"`
	EndTime      float64                `json:"end_time"`
	Duration     float64                `json:"duration"`
	Transition   map[string]interface{} `json:"transition"`
}

// getAssetIDString converts AssetID to string
func getAssetIDString(assetID interface{}) string {
	if assetID == nil {
		return ""
	}
	switch v := assetID.(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%.0f", v)
	case int:
		return fmt.Sprintf("%d", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// FinalizeEpisodeRequest represents a request to finalize episode production
type FinalizeEpisodeRequest struct {
	EpisodeID string         `json:"episode_id"`
	Clips     []TimelineClip `json:"clips"`
}

// FinalizeEpisode completes episode production by merging the final video based on timeline scene order
func (s *VideoMergeService) FinalizeEpisode(episodeID string, timelineData *FinalizeEpisodeRequest) (map[string]interface{}, error) {
	// Verify episode exists and belongs to the user
	var episode models.Episode
	if err := s.db.Preload("Drama").Preload("Storyboards").Where("id = ?", episodeID).First(&episode).Error; err != nil {
		return nil, fmt.Errorf("episode not found")
	}

	// Build storyboard ID mapping
	sceneMap := make(map[string]models.Storyboard)
	for _, scene := range episode.Storyboards {
		sceneMap[fmt.Sprintf("%d", scene.ID)] = scene
	}

	// Build scene clips based on timeline data
	var sceneClips []models.SceneClip
	var skippedScenes []int

	if timelineData != nil && len(timelineData.Clips) > 0 {
		s.log.Infow("Processing timeline data", "clips_count", len(timelineData.Clips))
		// Use timeline data provided by the frontend
		for i, clip := range timelineData.Clips {
			assetIDStr := getAssetIDString(clip.AssetID)
			s.log.Infow("Processing clip", "index", i, "storyboard_id", clip.StoryboardID, "asset_id", assetIDStr, "order", clip.Order)
			// Prefer videos from the asset library (via AssetID)
			var videoURL string
			var sceneID uint

			if assetIDStr != "" {
				// Get video from asset library, prefer local_path
				var asset models.Asset
				if err := s.db.Where("id = ? AND type = ?", assetIDStr, models.AssetTypeVideo).First(&asset).Error; err == nil {
					// Prefer using local_path
					if asset.LocalPath != nil && *asset.LocalPath != "" {
						// Check if already a full path
						if filepath.IsAbs(*asset.LocalPath) || filepath.HasPrefix(*asset.LocalPath, s.storagePath) {
							videoURL = *asset.LocalPath
						} else {
							videoURL = filepath.Join(s.storagePath, *asset.LocalPath)
						}
						s.log.Infow("Using local video from asset library", "asset_id", assetIDStr, "local_path", videoURL)
					} else {
						// Fall back to remote URL
						videoURL = asset.URL
						s.log.Infow("Using remote video from asset library", "asset_id", assetIDStr, "video_url", videoURL)
					}
					// If the asset is linked to a storyboard, use the linked storyboard_id
					if asset.StoryboardID != nil {
						sceneID = *asset.StoryboardID
					}
				} else {
					s.log.Warnw("Asset not found, will try storyboard video", "asset_id", assetIDStr, "error", err)
				}
			}

			// If no video was obtained from the asset library, try getting it from storyboard
			if videoURL == "" && clip.StoryboardID != "" {
				scene, exists := sceneMap[clip.StoryboardID]
				if !exists {
					s.log.Warnw("Storyboard not found in episode, skipping", "storyboard_id", clip.StoryboardID)
					continue
				}

				// Find the associated video_generation record to get local_path
				var videoGen models.VideoGeneration
				if err := s.db.Where("storyboard_id = ? AND status = ?", scene.ID, "completed").Order("created_at DESC").First(&videoGen).Error; err == nil {
					if videoGen.LocalPath != nil && *videoGen.LocalPath != "" {
						// Check if already a full path
						if filepath.IsAbs(*videoGen.LocalPath) || filepath.HasPrefix(*videoGen.LocalPath, s.storagePath) {
							videoURL = *videoGen.LocalPath
						} else {
							videoURL = filepath.Join(s.storagePath, *videoGen.LocalPath)
						}
						sceneID = scene.ID
						s.log.Infow("Using local video from video_generation", "storyboard_id", clip.StoryboardID, "local_path", videoURL)
					} else if scene.VideoURL != nil && *scene.VideoURL != "" {
						// Fall back to remote URL
						videoURL = *scene.VideoURL
						sceneID = scene.ID
						s.log.Infow("Using remote video from storyboard", "storyboard_id", clip.StoryboardID, "video_url", videoURL)
					}
				} else if scene.VideoURL != nil && *scene.VideoURL != "" {
					// If no video_generation found, use storyboard video_url directly
					videoURL = *scene.VideoURL
					sceneID = scene.ID
					s.log.Infow("Using video from storyboard (no video_generation found)", "storyboard_id", clip.StoryboardID, "video_url", videoURL)
				}
			}

			// If there is still no video URL, skip this clip
			if videoURL == "" {
				s.log.Warnw("No video available for clip, skipping", "clip", clip)
				if clip.StoryboardID != "" {
					if scene, exists := sceneMap[clip.StoryboardID]; exists {
						skippedScenes = append(skippedScenes, scene.StoryboardNumber)
					}
				}
				continue
			}

			sceneClip := models.SceneClip{
				SceneID:    sceneID,
				VideoURL:   videoURL,
				Duration:   clip.Duration,
				Order:      clip.Order,
				StartTime:  clip.StartTime,
				EndTime:    clip.EndTime,
				Transition: clip.Transition,
			}
			s.log.Infow("Adding scene clip with transition",
				"scene_id", sceneID,
				"order", clip.Order,
				"video_url", videoURL,
				"transition", clip.Transition)
			sceneClips = append(sceneClips, sceneClip)
			s.log.Infow("Scene clip added", "total_clips", len(sceneClips))
		}
	} else {
		// No timeline data, use default scene order
		if len(episode.Storyboards) == 0 {
			return nil, fmt.Errorf("no scenes found for this episode")
		}

		order := 0
		for _, scene := range episode.Storyboards {
			// First look for videos associated with this storyboard in the asset library
			var videoURL string
			var asset models.Asset
			if err := s.db.Where("storyboard_id = ? AND type = ? AND episode_id = ?",
				scene.ID, models.AssetTypeVideo, episode.ID).
				Order("created_at DESC").
				First(&asset).Error; err == nil {
				// Prefer local_path
				if asset.LocalPath != nil && *asset.LocalPath != "" {
					// Check if it's already a full path
					if filepath.IsAbs(*asset.LocalPath) || filepath.HasPrefix(*asset.LocalPath, s.storagePath) {
						videoURL = *asset.LocalPath
					} else {
						videoURL = filepath.Join(s.storagePath, *asset.LocalPath)
					}
					s.log.Infow("Using local video from asset library for storyboard",
						"storyboard_id", scene.ID,
						"asset_id", asset.ID,
						"local_path", videoURL)
				} else {
					videoURL = asset.URL
					s.log.Infow("Using remote video from asset library for storyboard",
						"storyboard_id", scene.ID,
						"asset_id", asset.ID,
						"video_url", videoURL)
				}
			} else {
				// If not in the asset library, look for video_generation records
				var videoGen models.VideoGeneration
				if err := s.db.Where("storyboard_id = ? AND status = ?", scene.ID, "completed").Order("created_at DESC").First(&videoGen).Error; err == nil {
					if videoGen.LocalPath != nil && *videoGen.LocalPath != "" {
						// Check if already a full path
						if filepath.IsAbs(*videoGen.LocalPath) || filepath.HasPrefix(*videoGen.LocalPath, s.storagePath) {
							videoURL = *videoGen.LocalPath
						} else {
							videoURL = filepath.Join(s.storagePath, *videoGen.LocalPath)
						}
						s.log.Infow("Using local video from video_generation for storyboard",
							"storyboard_id", scene.ID,
							"local_path", videoURL)
					} else if scene.VideoURL != nil && *scene.VideoURL != "" {
						videoURL = *scene.VideoURL
						s.log.Infow("Using remote video from storyboard",
							"storyboard_id", scene.ID,
							"video_url", videoURL)
					}
				} else if scene.VideoURL != nil && *scene.VideoURL != "" {
					// Last fallback to storyboard video_url
					videoURL = *scene.VideoURL
					s.log.Infow("Using fallback video from storyboard",
						"storyboard_id", scene.ID,
						"video_url", videoURL)
				}
			}

			// Skip scenes without videos
			if videoURL == "" {
				s.log.Warnw("Scene has no video, skipping", "storyboard_number", scene.StoryboardNumber)
				skippedScenes = append(skippedScenes, scene.StoryboardNumber)
				continue
			}

			clip := models.SceneClip{
				SceneID:  scene.ID,
				VideoURL: videoURL,
				Duration: float64(scene.Duration),
				Order:    order,
			}
			sceneClips = append(sceneClips, clip)
			order++
		}
	}

	// Check if there is at least one scene available for merging
	if len(sceneClips) == 0 {
		return nil, fmt.Errorf("no scenes with videos available for merging")
	}

	// Create video merge task
	title := fmt.Sprintf("%s - Episode %d", episode.Drama.Title, episode.EpisodeNum)

	finalReq := &MergeVideoRequest{
		EpisodeID: episodeID,
		DramaID:   fmt.Sprintf("%d", episode.DramaID),
		Title:     title,
		Scenes:    sceneClips,
		Provider:  "doubao", // Default to doubao
	}

	// Execute video merge
	videoMerge, err := s.MergeVideos(finalReq)
	if err != nil {
		return nil, fmt.Errorf("failed to start video merge: %w", err)
	}

	// Update episode status to processing
	s.db.Model(&episode).Updates(map[string]interface{}{
		"status": "processing",
	})

	result := map[string]interface{}{
		"message":      "Video merge task created and processing in background",
		"merge_id":     videoMerge.ID,
		"episode_id":   episodeID,
		"scenes_count": len(sceneClips),
	}

	// If there are skipped scenes, add warning information
	if len(skippedScenes) > 0 {
		result["skipped_scenes"] = skippedScenes
		result["warning"] = fmt.Sprintf("Skipped %d scenes without generated videos (scene numbers: %v)", len(skippedScenes), skippedScenes)
	}

	return result, nil
}
