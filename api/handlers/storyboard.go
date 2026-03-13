package handlers

import (
	"strconv"

	"github.com/drama-generator/backend/application/services"
	"github.com/drama-generator/backend/pkg/config"
	"github.com/drama-generator/backend/pkg/logger"
	"github.com/drama-generator/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type StoryboardHandler struct {
	storyboardService *services.StoryboardService
	taskService       *services.TaskService
	log               *logger.Logger
}

func NewStoryboardHandler(db *gorm.DB, cfg *config.Config, log *logger.Logger) *StoryboardHandler {
	return &StoryboardHandler{
		storyboardService: services.NewStoryboardService(db, cfg, log),
		taskService:       services.NewTaskService(db, log),
		log:               log,
	}
}

func (h *StoryboardHandler) GenerateStoryboard(c *gin.Context) {
	episodeID := c.Param("episode_id")

	var req struct {
		Model string `json:"model"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Model = ""
	}

	taskID, err := h.storyboardService.GenerateStoryboard(episodeID, req.Model)
	if err != nil {
		h.log.Errorw("Failed to generate storyboard", "error", err, "episode_id", episodeID)
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"task_id": taskID,
		"status":  "pending",
		"message": "Storyboard generation task created and processing in background...",
	})
}

func (h *StoryboardHandler) UpdateStoryboard(c *gin.Context) {
	storyboardID := c.Param("id")

	var req map[string]interface{}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	err := h.storyboardService.UpdateStoryboard(storyboardID, req)
	if err != nil {
		h.log.Errorw("Failed to update storyboard", "error", err)
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "Storyboard updated successfully"})
}

func (h *StoryboardHandler) CreateStoryboard(c *gin.Context) {
	var req services.CreateStoryboardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	sb, err := h.storyboardService.CreateStoryboard(&req)
	if err != nil {
		h.log.Errorw("Failed to create storyboard", "error", err)
		response.InternalError(c, err.Error())
		return
	}

	response.Created(c, sb)
}

func (h *StoryboardHandler) DeleteStoryboard(c *gin.Context) {
	storyboardIDStr := c.Param("id")
	storyboardID, err := strconv.ParseUint(storyboardIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}

	if err := h.storyboardService.DeleteStoryboard(uint(storyboardID)); err != nil {
		h.log.Errorw("Failed to delete storyboard", "error", err)
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}
