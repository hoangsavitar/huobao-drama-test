package handlers

import (
	"encoding/json"
	"strings"

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
	ai := services.NewAIService(db, log)
	return &DramaHandler{
		db:                db,
		dramaService:      services.NewDramaService(db, cfg, log, ai),
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
		response.InternalError(c, "Creation failed")
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
		response.InternalError(c, "Failed to retrieve")
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
		response.InternalError(c, "Failed to retrieve list")
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
		response.InternalError(c, "Failed to retrieve statistics")
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
	episodeID := c.Query("episode_id") // Optional: if provided, only return characters for this episode

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
		response.InternalError(c, "Failed to retrieve characters")
		return
	}

	response.Success(c, characters)
}

func (h *DramaHandler) SaveCharacters(c *gin.Context) {
	dramaID := c.Param("id")

	var req services.SaveCharactersRequest

	// First try normal JSON binding
	if err := c.ShouldBindJSON(&req); err != nil {
		// If binding fails, check if characters field is a string instead of an array
		var rawReq map[string]interface{}
		if err := c.ShouldBindJSON(&rawReq); err != nil {
			// If even rawReq binding fails, return the error directly
			response.BadRequest(c, err.Error())
			return
		}

		// Check characters field type
		if charField, ok := rawReq["characters"]; ok {
			if charStr, ok := charField.(string); ok {
				// If characters is a string, try to parse it as a JSON array
				var characters []models.Character
				if err := json.Unmarshal([]byte(charStr), &characters); err != nil {
					// Parse failed, return error
					response.BadRequest(c, "Invalid characters field format, expected a JSON array or a string-formatted JSON array")
					return
				}

				// Manually construct the request object
				req.Characters = characters

				// Handle episode_id field
				if epID, ok := rawReq["episode_id"]; ok {
					if epIDStr, ok := epID.(float64); ok {
						epIDUint := uint(epIDStr)
						req.EpisodeID = &epIDUint
					}
				}
			} else {
				// If characters is not a string, return the original error
				response.BadRequest(c, err.Error())
				return
			}
		} else {
			// If there is no characters field, return the original error
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

// FinalizeEpisode completes episode production (triggers video merging)
func (h *DramaHandler) FinalizeEpisode(c *gin.Context) {

	episodeID := c.Param("episode_id")
	if episodeID == "" {
		response.BadRequest(c, "episode_id cannot be empty")
		return
	}

	// Try to read timeline data (optional)
	var timelineData *services.FinalizeEpisodeRequest
	if err := c.ShouldBindJSON(&timelineData); err != nil {
		// If no request body or parse fails, use nil (will use default scene order)
		h.log.Warnw("No timeline data provided, will use default scene order", "error", err)
		timelineData = nil
	} else if timelineData != nil {
		h.log.Infow("Received timeline data", "clips_count", len(timelineData.Clips), "episode_id", episodeID)
	}

	// Trigger video merge task
	result, err := h.videoMergeService.FinalizeEpisode(episodeID, timelineData)
	if err != nil {
		h.log.Errorw("Failed to finalize episode", "error", err, "episode_id", episodeID)
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, result)
}

// DownloadEpisodeVideo downloads episode video
func (h *DramaHandler) DownloadEpisodeVideo(c *gin.Context) {

	episodeID := c.Param("episode_id")
	if episodeID == "" {
		response.BadRequest(c, "episode_id cannot be empty")
		return
	}

	// Query episode
	var episode models.Episode
	if err := h.db.Preload("Drama").Where("id = ?", episodeID).First(&episode).Error; err != nil {
		response.NotFound(c, "Episode not found")
		return
	}

	// Check if video exists
	if episode.VideoURL == nil || *episode.VideoURL == "" {
		response.BadRequest(c, "This episode has no generated video yet")
		return
	}

	// Return video URL for frontend redirect download
	c.JSON(200, gin.H{
		"video_url":      *episode.VideoURL,
		"title":          episode.Title,
		"episode_number": episode.EpisodeNum,
	})
}

// GenerateNarrativeEpisodes calls narrative_AI (if configured) or built-in stub, then replaces drama episodes with branching graph.
func (h *DramaHandler) GenerateNarrativeEpisodes(c *gin.Context) {
	dramaID := c.Param("id")
	var req services.NarrativeGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.dramaService.GenerateNarrativeEpisodes(dramaID, req.UserIdea); err != nil {
		msg := err.Error()
		if msg == "drama not found" {
			response.NotFound(c, "Drama not found")
			return
		}
		if msg == "invalid drama ID" {
			response.BadRequest(c, msg)
			return
		}
		// Text AI / prompt / normalize failures — dependency unavailable or model error
		if strings.HasPrefix(msg, "narrative:") {
			h.log.Warnw("Narrative dependency failed", "error", err, "drama_id", dramaID)
			response.ServiceUnavailable(c, msg)
			return
		}
		h.log.Errorw("Narrative generate failed", "error", err, "drama_id", dramaID)
		response.InternalError(c, msg)
		return
	}
	response.Success(c, gin.H{"message": "Episodes generated from narrative"})
}
