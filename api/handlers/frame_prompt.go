package handlers

import (
	"github.com/drama-generator/backend/application/services"
	"github.com/drama-generator/backend/pkg/logger"
	"github.com/drama-generator/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

type FramePromptHandler struct {
	framePromptService *services.FramePromptService
	log                *logger.Logger
}

func NewFramePromptHandler(framePromptService *services.FramePromptService, log *logger.Logger) *FramePromptHandler {
	return &FramePromptHandler{
		framePromptService: framePromptService,
		log:                log,
	}
}

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

	taskID, err := h.framePromptService.GenerateFramePrompt(serviceReq, req.Model)
	if err != nil {
		h.log.Errorw("Failed to generate frame prompt", "error", err)
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"task_id": taskID,
		"status":  "pending",
		"message": "Frame prompt generation task created and processing in background...",
	})
}
