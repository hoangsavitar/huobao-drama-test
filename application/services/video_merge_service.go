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
	var episode models.Episode
	if err := s.db.Preload("Drama").Where("id = ?", req.EpisodeID).First(&episode).Error; err != nil {
		return nil, fmt.Errorf("episode not found")
	}

	for i, scene := range req.Scenes {
		if scene.VideoURL == "" {
			return nil, fmt.Errorf("scene %d has no video", i+1)
		}
	}

	provider := req.Provider
	if provider == "" {
		provider = "doubao"
	}

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

	var scenes []models.SceneClip
	if err := json.Unmarshal(videoMerge.Scenes, &scenes); err != nil {
		s.updateMergeError(mergeID, fmt.Sprintf("failed to parse scenes: %v", err))
		return
	}

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

	sort.Slice(scenes, func(i, j int) bool {
		return scenes[i].Order < scenes[j].Order
	})

	s.log.Infow("Merging video clips with FFmpeg", "scene_count", len(scenes))

	var totalDuration float64
	for _, scene := range scenes {
		totalDuration += scene.Duration
	}

	clips := make([]ffmpeg.VideoClip, len(scenes))
	for i, scene := range scenes {
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

	videoDir := filepath.Join(s.storagePath, "videos", "merged")
	if err := os.MkdirAll(videoDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create video directory: %w", err)
	}

	fileName := fmt.Sprintf("merged_%d.mp4", time.Now().Unix())
	outputPath := filepath.Join(videoDir, fileName)

	mergedPath, err := s.ffmpeg.MergeVideos(&ffmpeg.MergeOptions{
		OutputPath: outputPath,
		Clips:      clips,
	})
	if err != nil {
		return nil, fmt.Errorf("ffmpeg merge failed: %w", err)
	}

	s.log.Infow("Video merged successfully", "path", mergedPath)

	relPath := filepath.Join("videos", "merged", fileName)

	result := &video.VideoResult{
		VideoURL:  relPath,
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

	var videoMerge models.VideoMerge
	if err := s.db.First(&videoMerge, mergeID).Error; err != nil {
		s.log.Errorw("Failed to load video merge for completion", "error", err, "id", mergeID)
		return
	}

	finalVideoURL := result.VideoURL

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

	model := ""
	if len(config.Model) > 0 {
		model = config.Model[0]
	}

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

type TimelineClip struct {
	AssetID      interface{}            `json:"asset_id"`
	StoryboardID string                 `json:"storyboard_id"`
	Order        int                    `json:"order"`
	StartTime    float64                `json:"start_time"`
	EndTime      float64                `json:"end_time"`
	Duration     float64                `json:"duration"`
	Transition   map[string]interface{} `json:"transition"`
}

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

type FinalizeEpisodeRequest struct {
	EpisodeID string         `json:"episode_id"`
	Clips     []TimelineClip `json:"clips"`
}

func (s *VideoMergeService) FinalizeEpisode(episodeID string, timelineData *FinalizeEpisodeRequest) (map[string]interface{}, error) {
	var episode models.Episode
	if err := s.db.Preload("Drama").Preload("Storyboards").Where("id = ?", episodeID).First(&episode).Error; err != nil {
		return nil, fmt.Errorf("episode not found")
	}

	sceneMap := make(map[string]models.Storyboard)
	for _, scene := range episode.Storyboards {
		sceneMap[fmt.Sprintf("%d", scene.ID)] = scene
	}

	var sceneClips []models.SceneClip
	var skippedScenes []int

	if timelineData != nil && len(timelineData.Clips) > 0 {
		s.log.Infow("Processing timeline data", "clips_count", len(timelineData.Clips))
		for i, clip := range timelineData.Clips {
			assetIDStr := getAssetIDString(clip.AssetID)
			s.log.Infow("Processing clip", "index", i, "storyboard_id", clip.StoryboardID, "asset_id", assetIDStr, "order", clip.Order)
			var videoURL string
			var sceneID uint

			if assetIDStr != "" {
				var asset models.Asset
				if err := s.db.Where("id = ? AND type = ?", assetIDStr, models.AssetTypeVideo).First(&asset).Error; err == nil {
					if asset.LocalPath != nil && *asset.LocalPath != "" {
						if filepath.IsAbs(*asset.LocalPath) || filepath.HasPrefix(*asset.LocalPath, s.storagePath) {
							videoURL = *asset.LocalPath
						} else {
							videoURL = filepath.Join(s.storagePath, *asset.LocalPath)
						}
						s.log.Infow("Using local video from asset library", "asset_id", assetIDStr, "local_path", videoURL)
					} else {
						videoURL = asset.URL
						s.log.Infow("Using remote video from asset library", "asset_id", assetIDStr, "video_url", videoURL)
					}
					if asset.StoryboardID != nil {
						sceneID = *asset.StoryboardID
					}
				} else {
					s.log.Warnw("Asset not found, will try storyboard video", "asset_id", assetIDStr, "error", err)
				}
			}

			if videoURL == "" && clip.StoryboardID != "" {
				scene, exists := sceneMap[clip.StoryboardID]
				if !exists {
					s.log.Warnw("Storyboard not found in episode, skipping", "storyboard_id", clip.StoryboardID)
					continue
				}

				var videoGen models.VideoGeneration
				if err := s.db.Where("storyboard_id = ? AND status = ?", scene.ID, "completed").Order("created_at DESC").First(&videoGen).Error; err == nil {
					if videoGen.LocalPath != nil && *videoGen.LocalPath != "" {
						if filepath.IsAbs(*videoGen.LocalPath) || filepath.HasPrefix(*videoGen.LocalPath, s.storagePath) {
							videoURL = *videoGen.LocalPath
						} else {
							videoURL = filepath.Join(s.storagePath, *videoGen.LocalPath)
						}
						sceneID = scene.ID
						s.log.Infow("Using local video from video_generation", "storyboard_id", clip.StoryboardID, "local_path", videoURL)
					} else if scene.VideoURL != nil && *scene.VideoURL != "" {
						videoURL = *scene.VideoURL
						sceneID = scene.ID
						s.log.Infow("Using remote video from storyboard", "storyboard_id", clip.StoryboardID, "video_url", videoURL)
					}
				} else if scene.VideoURL != nil && *scene.VideoURL != "" {
					videoURL = *scene.VideoURL
					sceneID = scene.ID
					s.log.Infow("Using video from storyboard (no video_generation found)", "storyboard_id", clip.StoryboardID, "video_url", videoURL)
				}
			}

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
		if len(episode.Storyboards) == 0 {
			return nil, fmt.Errorf("no scenes found for this episode")
		}

		order := 0
		for _, scene := range episode.Storyboards {
			var videoURL string
			var asset models.Asset
			if err := s.db.Where("storyboard_id = ? AND type = ? AND episode_id = ?",
				scene.ID, models.AssetTypeVideo, episode.ID).
				Order("created_at DESC").
				First(&asset).Error; err == nil {
				if asset.LocalPath != nil && *asset.LocalPath != "" {
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
				var videoGen models.VideoGeneration
				if err := s.db.Where("storyboard_id = ? AND status = ?", scene.ID, "completed").Order("created_at DESC").First(&videoGen).Error; err == nil {
					if videoGen.LocalPath != nil && *videoGen.LocalPath != "" {
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
					videoURL = *scene.VideoURL
					s.log.Infow("Using fallback video from storyboard",
						"storyboard_id", scene.ID,
						"video_url", videoURL)
				}
			}

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

	if len(sceneClips) == 0 {
		return nil, fmt.Errorf("no scenes with videos available for merging")
	}

	title := fmt.Sprintf("%s - Episode %d", episode.Drama.Title, episode.EpisodeNum)

	finalReq := &MergeVideoRequest{
		EpisodeID: episodeID,
		DramaID:   fmt.Sprintf("%d", episode.DramaID),
		Title:     title,
		Scenes:    sceneClips,
		Provider:  "doubao",
	}

	videoMerge, err := s.MergeVideos(finalReq)
	if err != nil {
		return nil, fmt.Errorf("failed to start video merge: %w", err)
	}

	s.db.Model(&episode).Updates(map[string]interface{}{
		"status": "processing",
	})

	result := map[string]interface{}{
		"message":      "Video merge task created and processing in background",
		"merge_id":     videoMerge.ID,
		"episode_id":   episodeID,
		"scenes_count": len(sceneClips),
	}

	if len(skippedScenes) > 0 {
		result["skipped_scenes"] = skippedScenes
		result["warning"] = fmt.Sprintf("Skipped %d scenes without generated videos (scene numbers: %v)", len(skippedScenes), skippedScenes)
	}

	return result, nil
}
