package handlers

import (
	"encoding/json"

	"github.com/drama-generator/backend/application/services"
	"github.com/drama-generator/backend/domain/models"
	"github.com/drama-generator/backend/pkg/config"
	"github.com/drama-generator/backend/pkg/logger"
	"github.com/drama-generator/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DramaHandler struct {
	db                *gorm.DB
	dramaService      *services.DramaService
	videoMergeService *services.VideoMergeService
	log               *logger.Logger
}

func NewDramaHandler(db *gorm.DB, cfg *config.Config, log *logger.Logger, transferService *services.ResourceTransferService) *DramaHandler {
	return &DramaHandler{
		db:                db,
		dramaService:      services.NewDramaService(db, cfg, log),
		videoMergeService: services.NewVideoMergeService(db, transferService, cfg.Storage.LocalPath, cfg.Storage.BaseURL, log),
		log:               log,
	}
}

func (h *DramaHandler) CreateDrama(c *gin.Context) {

	var req services.CreateDramaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	drama, err := h.dramaService.CreateDrama(&req)
	if err != nil {
		response.InternalError(c, "Create failed")
		return
	}

	response.Created(c, drama)
}

func (h *DramaHandler) GetDrama(c *gin.Context) {

	dramaID := c.Param("id")

	drama, err := h.dramaService.GetDrama(dramaID)
	if err != nil {
		if err.Error() == "drama not found" {
			response.NotFound(c, "Drama not found")
			return
		}
		response.InternalError(c, "Get failed")
		return
	}

	response.Success(c, drama)
}

func (h *DramaHandler) ListDramas(c *gin.Context) {

	var query services.DramaListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 || query.PageSize > 100 {
		query.PageSize = 20
	}

	dramas, total, err := h.dramaService.ListDramas(&query)
	if err != nil {
		response.InternalError(c, "List failed")
		return
	}

	response.SuccessWithPagination(c, dramas, total, query.Page, query.PageSize)
}

func (h *DramaHandler) UpdateDrama(c *gin.Context) {

	dramaID := c.Param("id")

	var req services.UpdateDramaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	drama, err := h.dramaService.UpdateDrama(dramaID, &req)
	if err != nil {
		if err.Error() == "drama not found" {
			response.NotFound(c, "Drama not found")
			return
		}
		response.InternalError(c, "Update failed")
		return
	}

	response.Success(c, drama)
}

func (h *DramaHandler) DeleteDrama(c *gin.Context) {

	dramaID := c.Param("id")

	if err := h.dramaService.DeleteDrama(dramaID); err != nil {
		if err.Error() == "drama not found" {
			response.NotFound(c, "Drama not found")
			return
		}
		response.InternalError(c, "Delete failed")
		return
	}

	response.Success(c, gin.H{"message": "Deleted successfully"})
}

func (h *DramaHandler) GetDramaStats(c *gin.Context) {

	stats, err := h.dramaService.GetDramaStats()
	if err != nil {
		response.InternalError(c, "Failed to get stats")
		return
	}

	response.Success(c, stats)
}

func (h *DramaHandler) SaveOutline(c *gin.Context) {

	dramaID := c.Param("id")

	var req services.SaveOutlineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.dramaService.SaveOutline(dramaID, &req); err != nil {
		if err.Error() == "drama not found" {
			response.NotFound(c, "Drama not found")
			return
		}
		response.InternalError(c, "Save failed")
		return
	}

	response.Success(c, gin.H{"message": "Saved successfully"})
}

func (h *DramaHandler) GetCharacters(c *gin.Context) {

	dramaID := c.Param("id")
	episodeID := c.Query("episode_id")

	var episodeIDPtr *string
	if episodeID != "" {
		episodeIDPtr = &episodeID
	}

	characters, err := h.dramaService.GetCharacters(dramaID, episodeIDPtr)
	if err != nil {
		if err.Error() == "drama not found" {
			response.NotFound(c, "Drama not found")
			return
		}
		if err.Error() == "episode not found" {
			response.NotFound(c, "Episode not found")
			return
		}
		response.InternalError(c, "Failed to get characters")
		return
	}

	response.Success(c, characters)
}

func (h *DramaHandler) SaveCharacters(c *gin.Context) {
	dramaID := c.Param("id")

	var req services.SaveCharactersRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		var rawReq map[string]interface{}
		if err := c.ShouldBindJSON(&rawReq); err != nil {
			response.BadRequest(c, err.Error())
			return
		}

		if charField, ok := rawReq["characters"]; ok {
			if charStr, ok := charField.(string); ok {
				var characters []models.Character
				if err := json.Unmarshal([]byte(charStr), &characters); err != nil {
					response.BadRequest(c, "Invalid characters field: expected JSON array or a JSON-array string")
					return
				}

				req.Characters = characters

				if epID, ok := rawReq["episode_id"]; ok {
					if epIDStr, ok := epID.(float64); ok {
						epIDUint := uint(epIDStr)
						req.EpisodeID = &epIDUint
					}
				}
			} else {
				response.BadRequest(c, err.Error())
				return
			}
		} else {
			response.BadRequest(c, err.Error())
			return
		}
	}

	if err := h.dramaService.SaveCharacters(dramaID, &req); err != nil {
		if err.Error() == "drama not found" {
			response.NotFound(c, "Drama not found")
			return
		}
		response.InternalError(c, "Save failed")
		return
	}

	response.Success(c, gin.H{"message": "Saved successfully"})
}

func (h *DramaHandler) SaveEpisodes(c *gin.Context) {

	dramaID := c.Param("id")

	var req services.SaveEpisodesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.dramaService.SaveEpisodes(dramaID, &req); err != nil {
		if err.Error() == "drama not found" {
			response.NotFound(c, "Drama not found")
			return
		}
		response.InternalError(c, "Save failed")
		return
	}

	response.Success(c, gin.H{"message": "Saved successfully"})
}

func (h *DramaHandler) SaveProgress(c *gin.Context) {

	dramaID := c.Param("id")

	var req services.SaveProgressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.dramaService.SaveProgress(dramaID, &req); err != nil {
		if err.Error() == "drama not found" {
			response.NotFound(c, "Drama not found")
			return
		}
		response.InternalError(c, "Save failed")
		return
	}

	response.Success(c, gin.H{"message": "Saved successfully"})
}

func (h *DramaHandler) FinalizeEpisode(c *gin.Context) {

	episodeID := c.Param("episode_id")
	if episodeID == "" {
		response.BadRequest(c, "episode_id is required")
		return
	}

	var timelineData *services.FinalizeEpisodeRequest
	if err := c.ShouldBindJSON(&timelineData); err != nil {
		h.log.Warnw("No timeline data provided, will use default scene order", "error", err)
		timelineData = nil
	} else if timelineData != nil {
		h.log.Infow("Received timeline data", "clips_count", len(timelineData.Clips), "episode_id", episodeID)
	}

	result, err := h.videoMergeService.FinalizeEpisode(episodeID, timelineData)
	if err != nil {
		h.log.Errorw("Failed to finalize episode", "error", err, "episode_id", episodeID)
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, result)
}

func (h *DramaHandler) DownloadEpisodeVideo(c *gin.Context) {

	episodeID := c.Param("episode_id")
	if episodeID == "" {
		response.BadRequest(c, "episode_id is required")
		return
	}

	var episode models.Episode
	if err := h.db.Preload("Drama").Where("id = ?", episodeID).First(&episode).Error; err != nil {
		response.NotFound(c, "Episode not found")
		return
	}

	if episode.VideoURL == nil || *episode.VideoURL == "" {
		response.BadRequest(c, "This episode does not have a generated video yet")
		return
	}

	c.JSON(200, gin.H{
		"video_url":      *episode.VideoURL,
		"title":          episode.Title,
		"episode_number": episode.EpisodeNum,
	})
}
