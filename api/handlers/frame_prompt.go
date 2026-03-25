package handlers

import (
"github.com/drama-generator/backend/application/services"
"github.com/drama-generator/backend/pkg/logger"
"github.com/drama-generator/backend/pkg/response"
"github.com/gin-gonic/gin"
)

// FramePromptHandler handles frame prompt generation requests
type FramePromptHandler struct {
framePromptService *services.FramePromptService
log                *logger.Logger
}

// NewFramePromptHandler creates a frame prompt handler
func NewFramePromptHandler(framePromptService *services.FramePromptService, log *logger.Logger) *FramePromptHandler {
return &FramePromptHandler{
framePromptService: framePromptService,
log:                log,
}
}

// GenerateFramePrompt generates frame prompts of the specified type
// POST /api/v1/storyboards/:id/frame-prompt
func (h *FramePromptHandler) GenerateFramePrompt(c *gin.Context) {
storyboardID := c.Param("id")

var req struct {
FrameType  string `json:"frame_type"`
PanelCount int    `json:"panel_count"`
Model      string `json:"model"`
}
if err := c.ShouldBindJSON(&req); err != nil {
response.BadRequest(c, err.Error())
return
}

serviceReq := services.GenerateFramePromptRequest{
StoryboardID: storyboardID,
FrameType:    services.FrameType(req.FrameType),
PanelCount:   req.PanelCount,
}

// Directly call the async service method, which creates a task and returns a task ID
taskID, err := h.framePromptService.GenerateFramePrompt(serviceReq, req.Model)
	storyboardID := c.Param("id")

	var req struct {
		FrameType  string `json:"frame_type"`
		PanelCount int    `json:"panel_count"`
		Model      string `json:"model"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	serviceReq := services.GenerateFramePromptRequest{
		StoryboardID: storyboardID,
		FrameType:    services.FrameType(req.FrameType),
		PanelCount:   req.PanelCount,
	}

	// Directly call the async service method, which creates a task and returns a task ID
	taskID, err := h.framePromptService.GenerateFramePrompt(serviceReq, req.Model)
	if err != nil {
		h.log.Errorw("Failed to generate frame prompt", "error", err)
		response.InternalError(c, err.Error())
		return
	}

	// Return task ID immediately
	response.Success(c, gin.H{
		"task_id": taskID,
		"status":  "pending",
		"message": "Frame prompt generation task created, processing in background...",
	})
}

// BatchGenerateFirstFramePrompts handles batch generation of first frame prompts
// POST /api/v1/episodes/:episode_id/batch-first-frames
func (h *FramePromptHandler) BatchGenerateFirstFramePrompts(c *gin.Context) {
	episodeID := c.Param("episode_id")

	var req struct {
		ChunkSize int    `json:"chunk_size"`
		Model     string `json:"model"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if req.ChunkSize <= 0 {
		req.ChunkSize = 5 // Default chunk size
	}

	taskID, err := h.framePromptService.BatchGenerateFirstFramePrompts(episodeID, req.Model, req.ChunkSize)
	if err != nil {
		h.log.Errorw("Failed to batch generate first frame prompts", "error", err)
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"task_id": taskID,
		"status":  "pending",
		"message": "Batch first frame prompt generation task created, processing in background...",
	})
}

// BatchGenerateLtxPrompts handles batch generation of LTX video prompts
// POST /api/v1/episodes/:episode_id/batch-ltx-prompts
func (h *FramePromptHandler) BatchGenerateLtxPrompts(c *gin.Context) {
	episodeID := c.Param("episode_id")

	var req struct {
		ChunkSize int    `json:"chunk_size"`
		Model     string `json:"model"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if req.ChunkSize <= 0 {
		req.ChunkSize = 5 // Default chunk size
	}

	taskID, err := h.framePromptService.BatchGenerateLtxPrompts(episodeID, req.Model, req.ChunkSize)
	if err != nil {
		h.log.Errorw("Failed to batch generate LTX prompts", "error", err)
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"task_id": taskID,
		"status":  "pending",
		"message": "Batch LTX prompt generation task created, processing in background...",
	})
}
