package services

import (
	"fmt"
	"strings"

	"github.com/drama-generator/backend/domain/models"
	"github.com/drama-generator/backend/pkg/config"
	"github.com/drama-generator/backend/pkg/logger"
	"gorm.io/gorm"
)

type FramePromptService struct {
	db          *gorm.DB
	aiService   *AIService
	log         *logger.Logger
	config      *config.Config
	promptI18n  *PromptI18n
	taskService *TaskService
}

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

type FrameType string

const (
	FrameTypeFirst  FrameType = "first"
	FrameTypeKey    FrameType = "key"
	FrameTypeLast   FrameType = "last"
	FrameTypePanel  FrameType = "panel"
	FrameTypeAction FrameType = "action"
)

type GenerateFramePromptRequest struct {
	StoryboardID string    `json:"storyboard_id"`
	FrameType    FrameType `json:"frame_type"`
	PanelCount int `json:"panel_count,omitempty"`
}

type FramePromptResponse struct {
	FrameType   FrameType          `json:"frame_type"`
	SingleFrame *SingleFramePrompt `json:"single_frame,omitempty"`
	MultiFrame  *MultiFramePrompt  `json:"multi_frame,omitempty"`
}

type SingleFramePrompt struct {
	Prompt      string `json:"prompt"`
	Description string `json:"description"`
}

type MultiFramePrompt struct {
	Layout string              `json:"layout"`
	Frames []SingleFramePrompt `json:"frames"`
}

func (s *FramePromptService) GenerateFramePrompt(req GenerateFramePromptRequest, model string) (string, error) {
	var storyboard models.Storyboard
	if err := s.db.Preload("Characters").First(&storyboard, req.StoryboardID).Error; err != nil {
		return "", fmt.Errorf("storyboard not found: %w", err)
	}

	task, err := s.taskService.CreateTask("frame_prompt_generation", req.StoryboardID)
	if err != nil {
		s.log.Errorw("Failed to create frame prompt generation task", "error", err, "storyboard_id", req.StoryboardID)
		return "", fmt.Errorf("failed to create task: %w", err)
	}

	go s.processFramePromptGeneration(task.ID, req, model)

	s.log.Infow("Frame prompt generation task created", "task_id", task.ID, "storyboard_id", req.StoryboardID, "frame_type", req.FrameType)
	return task.ID, nil
}

func (s *FramePromptService) processFramePromptGeneration(taskID string, req GenerateFramePromptRequest, model string) {
	s.taskService.UpdateTaskStatus(taskID, "processing", 0, "Generating frame prompts...")

	var storyboard models.Storyboard
	if err := s.db.Preload("Characters").First(&storyboard, req.StoryboardID).Error; err != nil {
		s.log.Errorw("Storyboard not found during frame prompt generation", "error", err, "storyboard_id", req.StoryboardID)
		s.taskService.UpdateTaskStatus(taskID, "failed", 0, "Storyboard info not found")
		return
	}

	var scene *models.Scene
	if storyboard.SceneID != nil {
		scene = &models.Scene{}
		if err := s.db.First(scene, *storyboard.SceneID).Error; err != nil {
			s.log.Warnw("Scene not found during frame prompt generation", "scene_id", *storyboard.SceneID, "task_id", taskID)
			scene = nil
		}
	}

	var episode models.Episode
	if err := s.db.Preload("Drama").First(&episode, storyboard.EpisodeID).Error; err != nil {
		s.log.Warnw("Failed to load episode and drama", "error", err, "episode_id", storyboard.EpisodeID)
	}
	dramaStyle := episode.Drama.Style

	response := &FramePromptResponse{
		FrameType: req.FrameType,
	}

	switch req.FrameType {
	case FrameTypeFirst:
		response.SingleFrame = s.generateFirstFrame(storyboard, scene, dramaStyle, model)
		s.saveFramePrompt(req.StoryboardID, string(req.FrameType), response.SingleFrame.Prompt, response.SingleFrame.Description, "")
	case FrameTypeKey:
		response.SingleFrame = s.generateKeyFrame(storyboard, scene, dramaStyle, model)
		s.saveFramePrompt(req.StoryboardID, string(req.FrameType), response.SingleFrame.Prompt, response.SingleFrame.Description, "")
	case FrameTypeLast:
		response.SingleFrame = s.generateLastFrame(storyboard, scene, dramaStyle, model)
		s.saveFramePrompt(req.StoryboardID, string(req.FrameType), response.SingleFrame.Prompt, response.SingleFrame.Description, "")
	case FrameTypePanel:
		count := req.PanelCount
		if count == 0 {
			count = 3
		}
		response.MultiFrame = s.generatePanelFrames(storyboard, scene, count, dramaStyle, model)
		var prompts []string
		for _, frame := range response.MultiFrame.Frames {
			prompts = append(prompts, frame.Prompt)
		}
		combinedPrompt := strings.Join(prompts, "\n---\n")
		s.saveFramePrompt(req.StoryboardID, string(req.FrameType), combinedPrompt, "Storyboard panel combined prompt", response.MultiFrame.Layout)
	case FrameTypeAction:
		response.MultiFrame = s.generateActionSequence(storyboard, scene, dramaStyle, model)
		var prompts []string
		for _, frame := range response.MultiFrame.Frames {
			prompts = append(prompts, frame.Prompt)
		}
		combinedPrompt := strings.Join(prompts, "\n---\n")
		s.saveFramePrompt(req.StoryboardID, string(req.FrameType), combinedPrompt, "Action sequence combined prompt", response.MultiFrame.Layout)
	default:
		s.log.Errorw("Unsupported frame type during frame prompt generation", "frame_type", req.FrameType, "task_id", taskID)
		s.taskService.UpdateTaskStatus(taskID, "failed", 0, "Unsupported frame type")
		return
	}

	s.taskService.UpdateTaskResult(taskID, map[string]interface{}{
		"response":      response,
		"storyboard_id": req.StoryboardID,
		"frame_type":    string(req.FrameType),
	})

	s.log.Infow("Frame prompt generation completed", "task_id", taskID, "storyboard_id", req.StoryboardID, "frame_type", req.FrameType)
}

func (s *FramePromptService) saveFramePrompt(storyboardID, frameType, prompt, description, layout string) {
	framePrompt := models.FramePrompt{
		StoryboardID: uint(mustParseUint(storyboardID)),
		FrameType:    frameType,
		Prompt:       prompt,
	}

	if description != "" {
		framePrompt.Description = &description
	}
	if layout != "" {
		framePrompt.Layout = &layout
	}

	s.db.Where("storyboard_id = ? AND frame_type = ?", storyboardID, frameType).Delete(&models.FramePrompt{})

	if err := s.db.Create(&framePrompt).Error; err != nil {
		s.log.Warnw("Failed to save frame prompt", "error", err, "storyboard_id", storyboardID, "frame_type", frameType)
	}
}

func mustParseUint(s string) uint64 {
	var result uint64
	fmt.Sscanf(s, "%d", &result)
	return result
}

func (s *FramePromptService) generateFirstFrame(sb models.Storyboard, scene *models.Scene, dramaStyle string, model string) *SingleFramePrompt {
	contextInfo := s.buildStoryboardContext(sb, scene)

	systemPrompt := s.promptI18n.GetFirstFramePrompt(dramaStyle)
	userPrompt := s.promptI18n.FormatUserPrompt("frame_info", contextInfo)

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
		fallbackPrompt := s.buildFallbackPrompt(sb, scene, "first frame, static shot")
		return &SingleFramePrompt{
			Prompt:      fallbackPrompt,
			Description: "Static opening frame showing the initial state",
		}
	}

	result := s.parseFramePromptJSON(aiResponse)
	if result == nil {
		s.log.Warnw("Failed to parse AI JSON response, using fallback", "storyboard_id", sb.ID, "response", aiResponse)
		fallbackPrompt := s.buildFallbackPrompt(sb, scene, "first frame, static shot")
		return &SingleFramePrompt{
			Prompt:      fallbackPrompt,
			Description: "Static opening frame showing the initial state",
		}
	}

	return result
}

func (s *FramePromptService) generateKeyFrame(sb models.Storyboard, scene *models.Scene, dramaStyle string, model string) *SingleFramePrompt {
	contextInfo := s.buildStoryboardContext(sb, scene)

	systemPrompt := s.promptI18n.GetKeyFramePrompt(dramaStyle)
	userPrompt := s.promptI18n.FormatUserPrompt("key_frame_info", contextInfo)

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
		return &SingleFramePrompt{
			Prompt:      fallbackPrompt,
			Description: "Peak action moment showing the key movement",
		}
	}

	result := s.parseFramePromptJSON(aiResponse)
	if result == nil {
		s.log.Warnw("Failed to parse AI JSON response, using fallback", "storyboard_id", sb.ID, "response", aiResponse)
		fallbackPrompt := s.buildFallbackPrompt(sb, scene, "key frame, dynamic action")
		return &SingleFramePrompt{
			Prompt:      fallbackPrompt,
			Description: "Peak action moment showing the key movement",
		}
	}

	return result
}

func (s *FramePromptService) generateLastFrame(sb models.Storyboard, scene *models.Scene, dramaStyle string, model string) *SingleFramePrompt {
	contextInfo := s.buildStoryboardContext(sb, scene)

	systemPrompt := s.promptI18n.GetLastFramePrompt(dramaStyle)
	userPrompt := s.promptI18n.FormatUserPrompt("last_frame_info", contextInfo)

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
		return &SingleFramePrompt{
			Prompt:      fallbackPrompt,
			Description: "Ending frame showing final state and outcome",
		}
	}

	result := s.parseFramePromptJSON(aiResponse)
	if result == nil {
		s.log.Warnw("Failed to parse AI JSON response, using fallback", "storyboard_id", sb.ID, "response", aiResponse)
		fallbackPrompt := s.buildFallbackPrompt(sb, scene, "last frame, final state")
		return &SingleFramePrompt{
			Prompt:      fallbackPrompt,
			Description: "Ending frame showing final state and outcome",
		}
	}

	return result
}

func (s *FramePromptService) generatePanelFrames(sb models.Storyboard, scene *models.Scene, count int, dramaStyle string, model string) *MultiFramePrompt {
	layout := fmt.Sprintf("horizontal_%d", count)

	frames := make([]SingleFramePrompt, count)

	if count == 3 {
		frames[0] = *s.generateFirstFrame(sb, scene, dramaStyle, model)
		frames[0].Description = "Panel 1: initial state"

		frames[1] = *s.generateKeyFrame(sb, scene, dramaStyle, model)
		frames[1].Description = "Panel 2: action peak"

		frames[2] = *s.generateLastFrame(sb, scene, dramaStyle, model)
		frames[2].Description = "Panel 3: final state"
	} else if count == 4 {
		frames[0] = *s.generateFirstFrame(sb, scene, dramaStyle, model)
		frames[1] = *s.generateKeyFrame(sb, scene, dramaStyle, model)
		frames[2] = *s.generateKeyFrame(sb, scene, dramaStyle, model)
		frames[3] = *s.generateLastFrame(sb, scene, dramaStyle, model)
	}

	return &MultiFramePrompt{
		Layout: layout,
		Frames: frames,
	}
}

func (s *FramePromptService) generateActionSequence(sb models.Storyboard, scene *models.Scene, dramaStyle string, model string) *MultiFramePrompt {
	contextInfo := s.buildStoryboardContext(sb, scene)

	systemPrompt := s.promptI18n.GetActionSequenceFramePrompt(dramaStyle)
	userPrompt := s.promptI18n.FormatUserPrompt("frame_info", contextInfo)

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
		fallbackPrompt := s.buildFallbackPrompt(sb, scene, "3x3 storyboard grid action sequence, character consistency, continuous movement progression")
		return &MultiFramePrompt{
			Layout: "grid_3x3",
			Frames: []SingleFramePrompt{
				{
					Prompt:      fallbackPrompt,
					Description: "3x3 action sequence showing continuous motion progression",
				},
			},
		}
	}

	result := s.parseFramePromptJSON(aiResponse)
	if result == nil {
		s.log.Warnw("Failed to parse AI JSON response for action sequence, using fallback", "storyboard_id", sb.ID, "response", aiResponse)
		fallbackPrompt := s.buildFallbackPrompt(sb, scene, "3x3 storyboard grid action sequence, character consistency, continuous movement progression")
		return &MultiFramePrompt{
			Layout: "grid_3x3",
			Frames: []SingleFramePrompt{
				{
					Prompt:      fallbackPrompt,
					Description: "3x3 action sequence showing continuous motion progression",
				},
			},
		}
	}

	return &MultiFramePrompt{
		Layout: "grid_3x3",
		Frames: []SingleFramePrompt{*result},
	}
}

func (s *FramePromptService) buildStoryboardContext(sb models.Storyboard, scene *models.Scene) string {
	var parts []string

	if sb.Description != nil && *sb.Description != "" {
		parts = append(parts, s.promptI18n.FormatUserPrompt("shot_description_label", *sb.Description))
	}

	if scene != nil {
		parts = append(parts, s.promptI18n.FormatUserPrompt("scene_label", scene.Location, scene.Time))
	} else if sb.Location != nil && sb.Time != nil {
		parts = append(parts, s.promptI18n.FormatUserPrompt("scene_label", *sb.Location, *sb.Time))
	}

	if len(sb.Characters) > 0 {
		var charNames []string
		for _, char := range sb.Characters {
			charNames = append(charNames, char.Name)
		}
		parts = append(parts, s.promptI18n.FormatUserPrompt("characters_label", strings.Join(charNames, ", ")))
	}

	if sb.Action != nil && *sb.Action != "" {
		parts = append(parts, s.promptI18n.FormatUserPrompt("action_label", *sb.Action))
	}

	if sb.Result != nil && *sb.Result != "" {
		parts = append(parts, s.promptI18n.FormatUserPrompt("result_label", *sb.Result))
	}

	if sb.Dialogue != nil && *sb.Dialogue != "" {
		parts = append(parts, s.promptI18n.FormatUserPrompt("dialogue_label", *sb.Dialogue))
	}

	if sb.Atmosphere != nil && *sb.Atmosphere != "" {
		parts = append(parts, s.promptI18n.FormatUserPrompt("atmosphere_label", *sb.Atmosphere))
	}

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

func (s *FramePromptService) buildFallbackPrompt(sb models.Storyboard, scene *models.Scene, suffix string) string {
	var parts []string

	if scene != nil {
		parts = append(parts, fmt.Sprintf("%s, %s", scene.Location, scene.Time))
	}

	if len(sb.Characters) > 0 {
		for _, char := range sb.Characters {
			parts = append(parts, char.Name)
		}
	}

	if sb.Atmosphere != nil {
		parts = append(parts, *sb.Atmosphere)
	}

	parts = append(parts, "anime style", suffix)
	return strings.Join(parts, ", ")
}
