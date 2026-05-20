package handlers

import (
	"strconv"

	"github.com/drama-generator/backend/application/services"
	"github.com/drama-generator/backend/pkg/logger"
	"github.com/drama-generator/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

type LtxVideoPromptHandler struct {
	ltxService *services.LtxVideoPromptBatchService
	log        *logger.Logger
}

func NewLtxVideoPromptHandler(ltxService *services.LtxVideoPromptBatchService, log *logger.Logger) *LtxVideoPromptHandler {
	return &LtxVideoPromptHandler{
		ltxService: ltxService,
		log:        log,
	}
}

func (h *LtxVideoPromptHandler) BatchGenerateLtxVideoPrompts(c *gin.Context) {
	episodeID := c.Param("episode_id")
	if episodeID == "" {
		response.BadRequest(c, "episode_id cannot be empty")
		return
	}

	var req struct {
		StoryboardIDs []any   `json:"storyboard_ids"`
		Model         *string `json:"model"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if len(req.StoryboardIDs) == 0 {
		response.BadRequest(c, "storyboard_ids cannot be empty")
		return
	}

	ids := make([]uint, 0, len(req.StoryboardIDs))
	for _, raw := range req.StoryboardIDs {
		switch v := raw.(type) {
		case float64:
			if v <= 0 {
				continue
			}
			ids = append(ids, uint(v))
		case string:
			u, err := strconv.ParseUint(v, 10, 32)
			if err != nil {
				continue
			}
			if u > 0 {
				ids = append(ids, uint(u))
			}
		default:
			// ignore unsupported types
		}
	}

	if len(ids) == 0 {
		response.BadRequest(c, "no valid storyboard_ids provided")
		return
	}

	taskID, err := h.ltxService.BatchGenerateLtxVideoPrompts(episodeID, ids, func() string {
		if req.Model == nil {
			return ""
		}
		return *req.Model
	}())
	if err != nil {
		h.log.Errorw("Failed to batch generate LTX video prompts", "error", err, "episode_id", episodeID)
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"task_id":  taskID,
		"status":   "pending",
		"message":  "LTX video prompt batch task submitted",
		"episode_id": episodeID,
	})
}

