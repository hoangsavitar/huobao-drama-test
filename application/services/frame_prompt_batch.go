package services

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"

	"github.com/drama-generator/backend/domain/models"
)

type BatchPromptResult struct {
	OriginalFilename string `json:"original_filename"`
	OptimizedPrompt  string `json:"optimized_prompt"`
	Prompt           string `json:"prompt"`
}

type BatchPromptResponse struct {
	Results []BatchPromptResult `json:"results"`
}

// BatchGenerateFirstFramePrompts batches first frame prompt generation
func (s *FramePromptService) BatchGenerateFirstFramePrompts(episodeID string, model string, chunkSize int) (string, error) {
	// Create task
	task, err := s.taskService.CreateTask("batch_first_frame_generation", episodeID)
	if err != nil {
		return "", fmt.Errorf("failed to create task: %w", err)
	}

	go s.processBatchPrompts(task.ID, episodeID, model, chunkSize, "first_frame")
	return task.ID, nil
}

// BatchGenerateLtxPrompts batches LtxPrompt generation
func (s *FramePromptService) BatchGenerateLtxPrompts(episodeID string, model string, chunkSize int) (string, error) {
	// Create task
	task, err := s.taskService.CreateTask("batch_ltx_prompt_generation", episodeID)
	if err != nil {
		return "", fmt.Errorf("failed to create task: %w", err)
	}

	go s.processBatchPrompts(task.ID, episodeID, model, chunkSize, "ltx")
	return task.ID, nil
}

func (s *FramePromptService) processBatchPrompts(taskID, episodeID, model string, chunkSize int, promptType string) {
	s.taskService.UpdateTaskStatus(taskID, "processing", 0, "Initializing batch generation...")

	// 1. Fetch storyboards for episode
	var episode models.Episode
	if err := s.db.Preload("Drama").First(&episode, episodeID).Error; err != nil {
		s.taskService.UpdateTaskStatus(taskID, "failed", 0, "Episode not found")
		return
	}

	var storyboards []models.Storyboard
	if err := s.db.Where("episode_id = ?", episodeID).Order("sort_order asc").Find(&storyboards).Error; err != nil {
		s.taskService.UpdateTaskStatus(taskID, "failed", 0, "Failed to fetch storyboards")
		return
	}

	if len(storyboards) == 0 {
		s.taskService.UpdateTaskStatus(taskID, "completed", 100, "No storyboards found")
		return
	}

	// 2. Chunking
	totalChunks := int(math.Ceil(float64(len(storyboards)) / float64(chunkSize)))
	successCount := 0

	for i := 0; i < totalChunks; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > len(storyboards) {
			end = len(storyboards)
		}
		chunk := storyboards[start:end]

		// Update progress
		progress := int(float64(i) / float64(totalChunks) * 100)
		s.taskService.UpdateTaskStatus(taskID, "processing", progress, fmt.Sprintf("Processing chunk %d/%d", i+1, totalChunks))

		// Process chunk
		s.processChunk(chunk, episode.Drama.Style, episode.Drama.AspectRatio, model, promptType)
		successCount += len(chunk)
	}

	s.taskService.UpdateTaskStatus(taskID, "completed", 100, fmt.Sprintf("Successfully processed %d shots", successCount))
}

func (s *FramePromptService) processChunk(chunk []models.Storyboard, style, aspectRatio, model string, promptType string) {
	// Format inputs
	inputText := s.promptI18n.FormatBatchPromptShots(chunk)

	var systemPrompt string
	if promptType == "ltx" {
		systemPrompt = s.promptI18n.GetLtxVideoSystemPrompt()
	} else {
		// First frame batch prompt wrapper
		baseFirstFrame := s.promptI18n.GetFirstFramePrompt(style, aspectRatio)
		systemPrompt = baseFirstFrame + `

OUTPUT RULE: You are outputting for a BATCH of files. You must ONLY output a valid JSON object matching the following structure:
{
  "results": [
    {
      "original_filename": "shot_01.txt",
      "prompt": "your generated first frame prompt here"
    }
  ]
}
No conversational text, no extra markdown formatting around the JSON. Just raw structural JSON.`
	}

	// Call AI
	var aiResponse string
	var err error
	if model != "" {
		client, getErr := s.aiService.GetAIClientForModel("text", model)
		if getErr == nil && client != nil {
			aiResponse, err = client.GenerateText(inputText, systemPrompt)
		} else {
			aiResponse, err = s.aiService.GenerateText(inputText, systemPrompt)
		}
	} else {
		aiResponse, err = s.aiService.GenerateText(inputText, systemPrompt)
	}

	if err != nil {
		s.log.Errorw("Batch AI generation failed", "error", err, "promptType", promptType)
		return
	}

	// Clean JSON
	jsonStr := cleanJSON(aiResponse)
	var resp BatchPromptResponse
	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		s.log.Errorw("Failed to parse batch JSON", "error", err, "response", aiResponse)
		return
	}

	// Save to DB
	for _, res := range resp.Results {
		// extract ID from "shot_XX.txt"
		re := regexp.MustCompile(`shot_(\d+)\.txt`)
		matches := re.FindStringSubmatch(res.OriginalFilename)
		if len(matches) < 2 {
			continue
		}
		shotID, parseErr := strconv.ParseUint(matches[1], 10, 32)
		if parseErr != nil {
			continue
		}

		finalStr := res.OptimizedPrompt
		if finalStr == "" {
			finalStr = res.Prompt
		}
		if finalStr == "" {
			continue
		}

		if promptType == "ltx" {
			s.db.Model(&models.Storyboard{}).Where("id = ?", shotID).Update("ltx_prompt", finalStr)
		} else {
			s.saveFramePrompt(fmt.Sprintf("%d", shotID), "first", finalStr, "")
		}
	}
}

func cleanJSON(s string) string {
	if start := strings.Index(s, "```json"); start != -1 {
		s = s[start+7:]
	} else if start := strings.Index(s, "```JSON"); start != -1 {
		s = s[start+7:]
	} else if start := strings.Index(s, "```"); start != -1 {
		s = s[start+3:]
	}
	if end := strings.LastIndex(s, "```"); end != -1 {
		s = s[:end]
	}
	return strings.TrimSpace(s)
}
