package services

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	models "github.com/drama-generator/backend/domain/models"
	"github.com/drama-generator/backend/pkg/logger"
	"gorm.io/gorm"
)

type LtxVideoPromptBatchEntry struct {
	OriginalFilename     string `json:"original_filename"`
	VideoIdea            string `json:"video_idea"`
	FirstFrameReference  string `json:"first_frame_reference"`
	TargetAspectRatio    string `json:"target_aspect_ratio"`
}

type LtxVideoPromptBatchOutput struct {
	Results []struct {
		OriginalFilename string `json:"original_filename"`
		OptimizedPrompt  string `json:"optimized_prompt"`
	} `json:"results"`
}

type LtxVideoPromptBatchService struct {
	db          *gorm.DB
	log         *logger.Logger
	aiService   *AIService
	taskService *TaskService
}

func NewLtxVideoPromptBatchService(db *gorm.DB, log *logger.Logger, aiService *AIService) *LtxVideoPromptBatchService {
	return &LtxVideoPromptBatchService{
		db:          db,
		log:         log,
		aiService:   aiService,
		taskService: NewTaskService(db, log),
	}
}

func (s *LtxVideoPromptBatchService) BatchGenerateLtxVideoPrompts(episodeID string, storyboardIDs []uint, textModel string) (string, error) {
	if len(storyboardIDs) == 0 {
		return "", fmt.Errorf("no storyboard ids provided")
	}

	task, err := s.taskService.CreateTask("ltx_video_prompt_generation", episodeID)
	if err != nil {
		return "", fmt.Errorf("failed to create task: %w", err)
	}

	go s.processBatch(task.ID, episodeID, storyboardIDs, textModel)
	return task.ID, nil
}

func (s *LtxVideoPromptBatchService) processBatch(taskID string, episodeID string, storyboardIDs []uint, textModel string) {
	s.taskService.UpdateTaskStatus(taskID, "processing", 0, "Generating LTX video prompts...")

	episodeIDUint, err := strconv.ParseUint(episodeID, 10, 32)
	if err != nil {
		s.taskService.UpdateTaskError(taskID, fmt.Errorf("invalid episode_id: %s", episodeID))
		return
	}

	var episode models.Episode
	if err := s.db.Preload("Drama").Where("id = ?", episodeIDUint).First(&episode).Error; err != nil {
		s.taskService.UpdateTaskError(taskID, fmt.Errorf("episode not found: %w", err))
		return
	}

	aspectRatio := episode.Drama.AspectRatio
	if strings.TrimSpace(aspectRatio) == "" {
		aspectRatio = "16:9"
	}

	// Load storyboards
	var storyboards []models.Storyboard
	if err := s.db.
		Where("episode_id = ? AND id IN ?", episodeIDUint, storyboardIDs).
		Find(&storyboards).Error; err != nil {
		s.taskService.UpdateTaskError(taskID, fmt.Errorf("failed to load storyboards: %w", err))
		return
	}

	storyboardByID := make(map[uint]*models.Storyboard, len(storyboards))
	for i := range storyboards {
		sb := &storyboards[i]
		storyboardByID[sb.ID] = sb
	}

	// Only storyboards that already have video_prompt can be optimized
	var validIDs []uint
	for _, id := range storyboardIDs {
		if sb, ok := storyboardByID[id]; ok && sb.VideoPrompt != nil && strings.TrimSpace(*sb.VideoPrompt) != "" {
			validIDs = append(validIDs, id)
		}
	}

	if len(validIDs) == 0 {
		s.taskService.UpdateTaskResult(taskID, map[string]any{
			"done":   0,
			"failed": len(storyboardIDs),
			"reason": "no video_prompt found for selected storyboards",
		})
		return
	}

	// Load first-frame prompts
	var firstFramePrompts []models.FramePrompt
	if err := s.db.
		Where("storyboard_id IN ? AND frame_type = ?", validIDs, models.FrameTypeFirst).
		Find(&firstFramePrompts).Error; err != nil {
		s.taskService.UpdateTaskError(taskID, fmt.Errorf("failed to load first frame prompts: %w", err))
		return
	}

	firstFrameByStoryboard := make(map[uint]string, len(firstFramePrompts))
	for _, fp := range firstFramePrompts {
		if strings.TrimSpace(fp.Prompt) != "" {
			firstFrameByStoryboard[fp.StoryboardID] = fp.Prompt
		}
	}

	// Build batch entries for LTX system prompt.
	entries := make([]LtxVideoPromptBatchEntry, 0, len(validIDs))
	originalFilenameToStoryboardID := make(map[string]uint, len(validIDs))
	for _, id := range validIDs {
		sb := storyboardByID[id]
		if sb == nil || sb.VideoPrompt == nil {
			continue
		}
		originalFilename := fmt.Sprintf("storyboard_%d", sb.ID)
		originalFilenameToStoryboardID[originalFilename] = sb.ID
		entries = append(entries, LtxVideoPromptBatchEntry{
			OriginalFilename:    originalFilename,
			VideoIdea:           *sb.VideoPrompt,
			FirstFrameReference: firstFrameByStoryboard[sb.ID],
			TargetAspectRatio:   aspectRatio,
		})
	}

	entryJSON, _ := json.Marshal(entries)

	systemPrompt := GetVideoPromptSystemPrompt("LTX")
	userPrompt := fmt.Sprintf(
		"Optimize LTX video prompts for the following batch inputs.\n\nBatch entries JSON (keep original_filename unchanged):\n%s\n\nUse `first_frame_reference` ONLY for anchoring identity and camera continuity (do not rewrite subject appearance). Respect `target_aspect_ratio`. Return ONLY the JSON object described in the system prompt (results[].original_filename and results[].optimized_prompt).",
		string(entryJSON),
	)

	var aiResponse string
	var genErr error
	if strings.TrimSpace(textModel) != "" {
		client, getErr := s.aiService.GetAIClientForModel("text", textModel)
		if getErr != nil {
			s.log.Warnw("Failed to get text client for selected model, using default", "model", textModel, "error", getErr)
			aiResponse, genErr = s.aiService.GenerateText(userPrompt, systemPrompt)
		} else {
			aiResponse, genErr = client.GenerateText(userPrompt, systemPrompt)
		}
	} else {
		aiResponse, genErr = s.aiService.GenerateText(userPrompt, systemPrompt)
	}

	if genErr != nil {
		s.taskService.UpdateTaskError(taskID, genErr)
		return
	}

	parsed := parseLtxVideoPromptBatchJSON(s.log, aiResponse)
	if parsed == nil {
		s.taskService.UpdateTaskError(taskID, fmt.Errorf("failed to parse LTX batch JSON output"))
		return
	}

	done := 0
	failed := len(storyboardIDs)

	// Update each storyboard with optimized prompt when present.
	for _, r := range parsed.Results {
		sbID, ok := originalFilenameToStoryboardID[r.OriginalFilename]
		if !ok {
			continue
		}
		p := strings.TrimSpace(r.OptimizedPrompt)
		if p == "" {
			continue
		}

		if err := s.db.Model(&models.Storyboard{}).Where("id = ?", sbID).Update("ltx_video_prompt", p).Error; err != nil {
			s.log.Warnw("Failed to update ltx_video_prompt", "storyboard_id", sbID, "error", err)
			continue
		}
		done++
	}

	failed = len(storyboardIDs) - done

	// Store summary in task.result (optional; frontend just reloads)
	s.taskService.UpdateTaskResult(taskID, map[string]any{
		"done":   done,
		"failed": failed,
	})
}

func parseLtxVideoPromptBatchJSON(log *logger.Logger, aiResponse string) *LtxVideoPromptBatchOutput {
	cleaned := strings.TrimSpace(aiResponse)

	// Remove ```json ... ``` code fences
	re := regexp.MustCompile("(?s)```json\\s*(.+?)\\s*```")
	if matches := re.FindStringSubmatch(cleaned); len(matches) > 1 {
		cleaned = strings.TrimSpace(matches[1])
	}

	// Strip remaining backticks
	cleaned = strings.Trim(cleaned, "`")

	// Try direct unmarshal first
	var out LtxVideoPromptBatchOutput
	if err := json.Unmarshal([]byte(cleaned), &out); err == nil {
		if len(out.Results) == 0 {
			return nil
		}
		return &out
	}

	// Fallback: extract the first {...} block
	start := strings.Index(cleaned, "{")
	end := strings.LastIndex(cleaned, "}")
	if start < 0 || end <= start {
		return nil
	}

	sub := cleaned[start : end+1]
	if err := json.Unmarshal([]byte(sub), &out); err != nil {
		log.Warnw("Failed to parse LTX batch JSON", "error", err)
		return nil
	}
	if len(out.Results) == 0 {
		return nil
	}
	return &out
}

