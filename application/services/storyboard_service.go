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
	Title       string `json:"title"`        // 镜头标题
	ShotType    string `json:"shot_type"`    // 景别
	Angle       string `json:"angle"`        // 镜头角度
	Time        string `json:"time"`         // 时间
	Location    string `json:"location"`     // 地点
	SceneID     *uint  `json:"scene_id"`     // 背景ID（AI直接返回，可为null）
	Movement    string `json:"movement"`     // 运镜
	Action      string `json:"action"`       // 动作
	Dialogue    string `json:"dialogue"`     // 对话/独白
	Result      string `json:"result"`       // 画面结果
	Atmosphere  string `json:"atmosphere"`   // 环境氛围
	Emotion     string `json:"emotion"`      // 情绪
	Duration    int    `json:"duration"`     // 时长（秒）
	BgmPrompt   string `json:"bgm_prompt"`   // 配乐提示词
	SoundEffect string `json:"sound_effect"` // 音效描述
	Characters  []uint `json:"characters"`   // 涉及的角色ID列表
	IsPrimary   bool   `json:"is_primary"`   // 是否主镜
}

type GenerateStoryboardResult struct {
	Storyboards []Storyboard `json:"storyboards"`
	Total       int          `json:"total"`
}

func (s *StoryboardService) GenerateStoryboard(episodeID string, model string) (string, error) {
	// 从数据库获取剧集信息
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

	// 获取剧本内容
	var scriptContent string
	if episode.ScriptContent != nil && *episode.ScriptContent != "" {
		scriptContent = *episode.ScriptContent
	} else if episode.Description != nil && *episode.Description != "" {
		scriptContent = *episode.Description
	} else {
		return "", fmt.Errorf("script content is empty, generate episode content first")
	}

	// 获取该剧本的所有角色
	var characters []models.Character
	if err := s.db.Where("drama_id = ?", episode.DramaID).Order("name ASC").Find(&characters).Error; err != nil {
		return "", fmt.Errorf("failed to get character list: %w", err)
	}

	// 构建角色列表字符串（包含ID和名称）
	characterList := "No characters"
	if len(characters) > 0 {
		var charInfoList []string
		for _, char := range characters {
			charInfoList = append(charInfoList, fmt.Sprintf(`{"id": %d, "name": "%s"}`, char.ID, char.Name))
		}
		characterList = fmt.Sprintf("[%s]", strings.Join(charInfoList, ", "))
	}

	// 获取该项目已提取的场景列表（项目级）
	var scenes []models.Scene
	if err := s.db.Where("drama_id = ?", episode.DramaID).Order("location ASC, time ASC").Find(&scenes).Error; err != nil {
		s.log.Warnw("Failed to get scenes", "error", err)
	}

	// 构建场景列表字符串（包含ID、地点、时间）
	sceneList := "No scenes"
	if len(scenes) > 0 {
		var sceneInfoList []string
		for _, bg := range scenes {
			sceneInfoList = append(sceneInfoList, fmt.Sprintf(`{"id": %d, "location": "%s", "time": "%s"}`, bg.ID, bg.Location, bg.Time))
		}
		sceneList = fmt.Sprintf("[%s]", strings.Join(sceneInfoList, ", "))
	}

	// 使用国际化提示词
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

%s
%s

%s
%s

%s
%s

%s
%s

Return JSON only with schema:
{
  "storyboards": [
    {
      "shot_number": 1,
      "title": "Short title",
      "shot_type": "wide|medium|closeup|etc",
      "angle": "camera angle",
      "time": "detailed time/lighting",
      "location": "detailed location",
      "scene_id": 1,
      "movement": "camera movement",
      "action": "detailed character action",
      "dialogue": "dialogue or empty string",
      "result": "visual outcome",
      "atmosphere": "mood and environment",
      "emotion": "audience emotion",
      "duration": 6,
      "bgm_prompt": "music cue",
      "sound_effect": "sfx cue",
      "characters": [1,2],
      "is_primary": true
    }
  ]
}

Rules:
- Fully cover all story content in sequence; do not skip scenes/dialogue.
- Use only IDs from provided character/background lists.
- duration must be 4-12 seconds per shot.
- Provide concrete, visual descriptions suitable for image/video generation.
`, systemPrompt, scriptLabel, scriptContent, taskLabel, taskInstruction, charListLabel, characterList, charConstraint, sceneListLabel, sceneList, sceneConstraint)

	// 创建异步任务
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

	// 启动后台goroutine处理AI调用和后续逻辑
	go s.processStoryboardGeneration(task.ID, episodeID, model, prompt)

	// 立即返回任务ID
	return task.ID, nil
}

// processStoryboardGeneration 后台处理故事板生成
func (s *StoryboardService) processStoryboardGeneration(taskID, episodeID, model, prompt string) {
	// 更新任务状态为处理中
	if err := s.taskService.UpdateTaskStatus(taskID, "processing", 10, "Generating storyboard..."); err != nil {
		s.log.Errorw("Failed to update task status", "error", err, "task_id", taskID)
		return
	}

	s.log.Infow("Processing storyboard generation", "task_id", taskID, "episode_id", episodeID)

	// 调用AI服务生成（如果指定了模型则使用指定的模型）
	// 设置较大的max_tokens以确保完整返回所有分镜的JSON
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

	// 更新任务进度
	if err := s.taskService.UpdateTaskStatus(taskID, "processing", 50, "Storyboard generated, parsing result..."); err != nil {
		s.log.Errorw("Failed to update task status", "error", err, "task_id", taskID)
		return
	}

	// 解析JSON结果
	// AI可能返回两种格式：
	// 1. 数组格式: [{...}, {...}]
	// 2. 对象格式: {"storyboards": [{...}, {...}]}
	var result GenerateStoryboardResult

	// 先尝试解析为数组格式
	var storyboards []Storyboard
	if err := utils.SafeParseAIJSON(text, &storyboards); err == nil {
		// 成功解析为数组，包装为对象
		result.Storyboards = storyboards
		result.Total = len(storyboards)
		s.log.Infow("Parsed storyboard as array format", "count", len(storyboards), "task_id", taskID)
	} else {
		// 尝试解析为对象格式
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

	// 计算总时长（所有分镜时长之和）
	totalDuration := 0
	for _, sb := range result.Storyboards {
		totalDuration += sb.Duration
	}

	s.log.Infow("Storyboard generated",
		"task_id", taskID,
		"episode_id", episodeID,
		"count", result.Total,
		"total_duration_seconds", totalDuration)

	// 更新任务进度
	if err := s.taskService.UpdateTaskStatus(taskID, "processing", 70, "Saving storyboards..."); err != nil {
		s.log.Errorw("Failed to update task status", "error", err, "task_id", taskID)
		return
	}

	// 保存分镜头到数据库
	if err := s.saveStoryboards(episodeID, result.Storyboards); err != nil {
		s.log.Errorw("Failed to save storyboards", "error", err, "task_id", taskID)
		if updateErr := s.taskService.UpdateTaskError(taskID, fmt.Errorf("failed to save storyboards: %w", err)); updateErr != nil {
			s.log.Errorw("Failed to update task error", "error", updateErr, "task_id", taskID)
		}
		return
	}

	// 更新任务进度
	if err := s.taskService.UpdateTaskStatus(taskID, "processing", 90, "Updating episode duration..."); err != nil {
		s.log.Errorw("Failed to update task status", "error", err, "task_id", taskID)
		return
	}

	// 更新剧集时长（秒转分钟，向上取整）
	durationMinutes := (totalDuration + 59) / 60
	if err := s.db.Model(&models.Episode{}).Where("id = ?", episodeID).Update("duration", durationMinutes).Error; err != nil {
		s.log.Errorw("Failed to update episode duration", "error", err, "task_id", taskID)
		// 不中断流程，只记录错误
	} else {
		s.log.Infow("Episode duration updated",
			"task_id", taskID,
			"episode_id", episodeID,
			"duration_seconds", totalDuration,
			"duration_minutes", durationMinutes)
	}

	// 更新任务结果
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

// generateImagePrompt 生成专门用于图片生成的提示词（首帧静态画面）
func (s *StoryboardService) generateImagePrompt(sb Storyboard) string {
	var parts []string

	// 1. 完整的场景背景描述
	if sb.Location != "" {
		locationDesc := sb.Location
		if sb.Time != "" {
			locationDesc += ", " + sb.Time
		}
		parts = append(parts, locationDesc)
	}

	// 2. 角色初始静态姿态（去除动作过程，只保留起始状态）
	if sb.Action != "" {
		initialPose := extractInitialPose(sb.Action)
		if initialPose != "" {
			parts = append(parts, initialPose)
		}
	}

	// 3. 情绪氛围
	if sb.Emotion != "" {
		parts = append(parts, sb.Emotion)
	}

	// 4. 动漫风格
	parts = append(parts, "anime style, first frame")

	if len(parts) > 0 {
		return strings.Join(parts, ", ")
	}
	return "anime scene"
}

// extractInitialPose 提取初始静态姿态（去除动作过程）
func extractInitialPose(action string) string {
	processWords := []string{
		"then", "next", "after that", "followed by",
		"down", "up", "forward", "backward", "left", "right",
		"start", "continue", "gradually", "slowly", "quickly", "suddenly",
	}

	result := action
	for _, word := range processWords {
		if idx := strings.Index(result, word); idx > 0 {
			result = result[:idx]
			break
		}
	}

	result = strings.TrimRight(result, ",. ")
	return strings.TrimSpace(result)
}

// extractSimpleLocation 提取简化的场景地点（去除详细描述）
func extractSimpleLocation(location string) string {
	if idx := strings.Index(location, "·"); idx > 0 {
		return strings.TrimSpace(location[:idx])
	}

	if idx := strings.Index(location, ","); idx > 0 {
		return strings.TrimSpace(location[:idx])
	}

	maxLen := 15
	if len(location) > maxLen {
		return strings.TrimSpace(location[:maxLen])
	}

	return strings.TrimSpace(location)
}

// extractSimplePose 提取简单的核心姿态关键词（不超过10个字）
func extractSimplePose(action string) string {
	runes := []rune(action)
	maxLen := 10
	if len(runes) > maxLen {
		truncated := runes[:maxLen]
		for i := maxLen - 1; i >= 0; i-- {
			if truncated[i] == ',' || truncated[i] == '.' {
				truncated = runes[:i]
				break
			}
		}
		return strings.TrimSpace(string(truncated))
	}
	return strings.TrimSpace(action)
}

// extractFirstFramePose 从动作描述中提取首帧静态姿态
func extractFirstFramePose(action string) string {
	processWords := []string{
		"then", "next", "down", "forward", "move to", "rush to", "turn",
		"start", "continue", "gradually", "slowly", "quickly", "suddenly",
	}

	pose := action
	for _, word := range processWords {
		if idx := strings.Index(pose, word); idx > 0 {
			pose = pose[:idx]
			break
		}
	}

	pose = strings.TrimRight(pose, ",.")
	return strings.TrimSpace(pose)
}

// extractCompositionType 从镜头类型中提取构图类型（去除运镜）
func extractCompositionType(shotType string) string {
	cameraMovements := []string{
		"shake", "push", "pull", "follow", "orbit",
		"camera move", "camera", "move", "rotate",
	}

	comp := shotType
	for _, movement := range cameraMovements {
		comp = strings.ReplaceAll(comp, movement, "")
	}

	// 清理多余的标点和空格
	comp = strings.ReplaceAll(comp, "··", "·")
	comp = strings.ReplaceAll(comp, "·", " ")
	comp = strings.TrimSpace(comp)

	return comp
}

// generateVideoPrompt 生成专门用于视频生成的提示词（包含运镜和动态元素）
func (s *StoryboardService) generateVideoPrompt(sb Storyboard) string {
	var parts []string
	videoRatio := "16:9"
	// 1. 人物动作
	if sb.Action != "" {
		parts = append(parts, fmt.Sprintf("Action: %s", sb.Action))
	}

	// 2. 对话
	if sb.Dialogue != "" {
		parts = append(parts, fmt.Sprintf("Dialogue: %s", sb.Dialogue))
	}

	// 3. 镜头运动（视频特有）
	if sb.Movement != "" {
		parts = append(parts, fmt.Sprintf("Camera movement: %s", sb.Movement))
	}

	// 4. 镜头类型和角度
	if sb.ShotType != "" {
		parts = append(parts, fmt.Sprintf("Shot type: %s", sb.ShotType))
	}
	if sb.Angle != "" {
		parts = append(parts, fmt.Sprintf("Camera angle: %s", sb.Angle))
	}

	// 5. 场景环境
	if sb.Location != "" {
		locationDesc := sb.Location
		if sb.Time != "" {
			locationDesc += ", " + sb.Time
		}
		parts = append(parts, fmt.Sprintf("Scene: %s", locationDesc))
	}

	// 6. 环境氛围
	if sb.Atmosphere != "" {
		parts = append(parts, fmt.Sprintf("Atmosphere: %s", sb.Atmosphere))
	}

	// 7. 情绪和结果
	if sb.Emotion != "" {
		parts = append(parts, fmt.Sprintf("Mood: %s", sb.Emotion))
	}
	if sb.Result != "" {
		parts = append(parts, fmt.Sprintf("Result: %s", sb.Result))
	}

	// 8. 音频元素
	if sb.BgmPrompt != "" {
		parts = append(parts, fmt.Sprintf("BGM: %s", sb.BgmPrompt))
	}
	if sb.SoundEffect != "" {
		parts = append(parts, fmt.Sprintf("Sound effects: %s", sb.SoundEffect))
	}

	// 9. 视频比例
	parts = append(parts, fmt.Sprintf("=VideoRatio: %s", videoRatio))
	if len(parts) > 0 {
		return strings.Join(parts, ". ")
	}
	return "Anime style video scene"
}

func (s *StoryboardService) saveStoryboards(episodeID string, storyboards []Storyboard) error {
	// 验证 episodeID
	epID, err := strconv.ParseUint(episodeID, 10, 32)
	if err != nil {
		s.log.Errorw("Invalid episode ID", "episode_id", episodeID, "error", err)
		return fmt.Errorf("invalid episode ID: %s", episodeID)
	}

	// 防御性检查：如果AI返回的分镜数量为0，不应该删除旧分镜
	if len(storyboards) == 0 {
		s.log.Errorw("AI returned zero storyboards, refusing to overwrite existing records", "episode_id", episodeID)
		return fmt.Errorf("AI storyboard generation failed: zero storyboards returned")
	}

	s.log.Infow("Start saving storyboards",
		"episode_id", episodeID,
		"episode_id_uint", uint(epID),
		"storyboard_count", len(storyboards))

	// 开启事务
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 验证该章节是否存在
		var episode models.Episode
		if err := tx.First(&episode, epID).Error; err != nil {
			s.log.Errorw("Episode not found", "episode_id", episodeID, "error", err)
			return fmt.Errorf("episode not found: %s", episodeID)
		}

		s.log.Infow("Loaded episode info",
			"episode_id", episode.ID,
			"episode_number", episode.EpisodeNum,
			"drama_id", episode.DramaID,
			"title", episode.Title)

		// 获取该剧集所有的分镜ID（使用 uint 类型）
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

		// 如果有分镜，先清理关联的image_generations的storyboard_id
		if len(storyboardIDs) > 0 {
			if err := tx.Model(&models.ImageGeneration{}).
				Where("storyboard_id IN ?", storyboardIDs).
				Update("storyboard_id", nil).Error; err != nil {
				return err
			}
			s.log.Infow("Cleared related image generation records", "count", len(storyboardIDs))
		}

		s.log.Warnw("Preparing to delete existing storyboard data",
			"episode_id_string", episodeID,
			"episode_id_uint", uint(epID),
			"episode_id_from_db", episode.ID,
			"will_delete_count", len(storyboardIDs))

		result := tx.Where("episode_id = ?", uint(epID)).Delete(&models.Storyboard{})
		if result.Error != nil {
			s.log.Errorw("Failed to delete old storyboards", "episode_id", uint(epID), "error", result.Error)
			return result.Error
		}

		s.log.Infow("Deleted old storyboards",
			"episode_id", uint(epID),
			"deleted_count", result.RowsAffected)

		// 注意：不删除背景，因为背景是在分镜拆解前就提取好的
		// AI会直接返回scene_id，不需要在这里做字符串匹配

		// 保存新的分镜头
		for _, sb := range storyboards {
			// 构建描述信息，包含对话
			description := fmt.Sprintf("[Shot Type] %s\n[Camera Movement] %s\n[Action] %s\n[Dialogue] %s\n[Result] %s\n[Emotion] %s",
				sb.ShotType, sb.Movement, sb.Action, sb.Dialogue, sb.Result, sb.Emotion)

			// 生成两种专用提示词
			imagePrompt := s.generateImagePrompt(sb) // 专用于图片生成
			videoPrompt := s.generateVideoPrompt(sb) // 专用于视频生成

			// 处理 dialogue 字段
			var dialoguePtr *string
			if sb.Dialogue != "" {
				dialoguePtr = &sb.Dialogue
			}

			// 使用AI直接返回的SceneID
			if sb.SceneID != nil {
				s.log.Infow("Background ID from AI",
					"shot_number", sb.ShotNumber,
					"scene_id", *sb.SceneID)
			}

			// 处理 title 字段
			var titlePtr *string
			if sb.Title != "" {
				titlePtr = &sb.Title
			}

			// 处理shot_type、angle、movement字段
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

			// 处理bgm_prompt、sound_effect字段
			var bgmPromptPtr, soundEffectPtr *string
			if sb.BgmPrompt != "" {
				bgmPromptPtr = &sb.BgmPrompt
			}
			if sb.SoundEffect != "" {
				soundEffectPtr = &sb.SoundEffect
			}

			// 处理result、atmosphere字段
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

			// 关联角色
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

// CreateStoryboardRequest 创建分镜请求
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

// CreateStoryboard 创建单个分镜
func (s *StoryboardService) CreateStoryboard(req *CreateStoryboardRequest) (*models.Storyboard, error) {
	// 构建Storyboard对象
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
		Emotion:     "", // 可以后续添加
		Duration:    req.Duration,
		BgmPrompt:   getString(req.BgmPrompt),
		SoundEffect: getString(req.SoundEffect),
		Characters:  req.Characters,
	}
	if req.Title != nil {
		sb.Title = *req.Title
	}

	// 生成提示词
	imagePrompt := s.generateImagePrompt(sb)
	videoPrompt := s.generateVideoPrompt(sb)

	// 构建 description
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

	// 关联角色
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

// DeleteStoryboard 删除分镜
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
