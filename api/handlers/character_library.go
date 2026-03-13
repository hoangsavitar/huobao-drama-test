package handlers

import (
	"strconv"

	services2 "github.com/drama-generator/backend/application/services"
	"github.com/drama-generator/backend/infrastructure/storage"
	"github.com/drama-generator/backend/pkg/config"
	"github.com/drama-generator/backend/pkg/logger"
	"github.com/drama-generator/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CharacterLibraryHandler struct {
	libraryService *services2.CharacterLibraryService
	imageService   *services2.ImageGenerationService
	log            *logger.Logger
}

func NewCharacterLibraryHandler(db *gorm.DB, cfg *config.Config, log *logger.Logger, transferService *services2.ResourceTransferService, localStorage *storage.LocalStorage) *CharacterLibraryHandler {
	return &CharacterLibraryHandler{
		libraryService: services2.NewCharacterLibraryService(db, log, cfg),
		imageService:   services2.NewImageGenerationService(db, cfg, transferService, localStorage, log),
		log:            log,
	}
}

func (h *CharacterLibraryHandler) ListLibraryItems(c *gin.Context) {

	var query services2.CharacterLibraryQuery
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

	items, total, err := h.libraryService.ListLibraryItems(&query)
	if err != nil {
		h.log.Errorw("Failed to list library items", "error", err)
		response.InternalError(c, "Failed to get character library")
		return
	}

	response.SuccessWithPagination(c, items, total, query.Page, query.PageSize)
}

func (h *CharacterLibraryHandler) CreateLibraryItem(c *gin.Context) {

	var req services2.CreateLibraryItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	item, err := h.libraryService.CreateLibraryItem(&req)
	if err != nil {
		h.log.Errorw("Failed to create library item", "error", err)
		response.InternalError(c, "Failed to add to character library")
		return
	}

	response.Created(c, item)
}

func (h *CharacterLibraryHandler) GetLibraryItem(c *gin.Context) {

	itemID := c.Param("id")

	item, err := h.libraryService.GetLibraryItem(itemID)
	if err != nil {
		if err.Error() == "library item not found" {
			response.NotFound(c, "Library item not found")
			return
		}
		h.log.Errorw("Failed to get library item", "error", err)
		response.InternalError(c, "Failed to get item")
		return
	}

	response.Success(c, item)
}

func (h *CharacterLibraryHandler) DeleteLibraryItem(c *gin.Context) {

	itemID := c.Param("id")

	if err := h.libraryService.DeleteLibraryItem(itemID); err != nil {
		if err.Error() == "library item not found" {
			response.NotFound(c, "Library item not found")
			return
		}
		h.log.Errorw("Failed to delete library item", "error", err)
		response.InternalError(c, "Delete failed")
		return
	}

	response.Success(c, gin.H{"message": "Deleted successfully"})
}

func (h *CharacterLibraryHandler) UploadCharacterImage(c *gin.Context) {

	characterID := c.Param("id")

	var req struct {
		ImageURL string `json:"image_url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.libraryService.UploadCharacterImage(characterID, req.ImageURL); err != nil {
		if err.Error() == "character not found" {
			response.NotFound(c, "Character not found")
			return
		}
		if err.Error() == "unauthorized" {
			response.Forbidden(c, "No permission")
			return
		}
		h.log.Errorw("Failed to upload character image", "error", err)
		response.InternalError(c, "Upload failed")
		return
	}

	response.Success(c, gin.H{"message": "Uploaded successfully"})
}

func (h *CharacterLibraryHandler) ApplyLibraryItemToCharacter(c *gin.Context) {

	characterID := c.Param("id")

	var req struct {
		LibraryItemID string `json:"library_item_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.libraryService.ApplyLibraryItemToCharacter(characterID, req.LibraryItemID); err != nil {
		if err.Error() == "library item not found" {
			response.NotFound(c, "Library item not found")
			return
		}
		if err.Error() == "character not found" {
			response.NotFound(c, "Character not found")
			return
		}
		if err.Error() == "unauthorized" {
			response.Forbidden(c, "No permission")
			return
		}
		h.log.Errorw("Failed to apply library item", "error", err)
		response.InternalError(c, "Apply failed")
		return
	}

	response.Success(c, gin.H{"message": "Applied successfully"})
}

func (h *CharacterLibraryHandler) AddCharacterToLibrary(c *gin.Context) {

	characterID := c.Param("id")

	var req struct {
		Category *string `json:"category"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		req.Category = nil
	}

	item, err := h.libraryService.AddCharacterToLibrary(characterID, req.Category)
	if err != nil {
		if err.Error() == "character not found" {
			response.NotFound(c, "Character not found")
			return
		}
		if err.Error() == "unauthorized" {
			response.Forbidden(c, "No permission")
			return
		}
		if err.Error() == "character has no image" {
			response.BadRequest(c, "Character does not have an image yet")
			return
		}
		h.log.Errorw("Failed to add character to library", "error", err)
		response.InternalError(c, "Add failed")
		return
	}

	response.Created(c, item)
}

func (h *CharacterLibraryHandler) UpdateCharacter(c *gin.Context) {

	characterID := c.Param("id")

	var req services2.UpdateCharacterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.libraryService.UpdateCharacter(characterID, &req); err != nil {
		if err.Error() == "character not found" {
			response.NotFound(c, "Character not found")
			return
		}
		if err.Error() == "unauthorized" {
			response.Forbidden(c, "No permission")
			return
		}
		h.log.Errorw("Failed to update character", "error", err)
		response.InternalError(c, "Update failed")
		return
	}

	response.Success(c, gin.H{"message": "Updated successfully"})
}

func (h *CharacterLibraryHandler) DeleteCharacter(c *gin.Context) {

	characterIDStr := c.Param("id")
	characterID, err := strconv.ParseUint(characterIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "Invalid character ID")
		return
	}

	if err := h.libraryService.DeleteCharacter(uint(characterID)); err != nil {
		h.log.Errorw("Failed to delete character", "error", err, "id", characterID)
		if err.Error() == "character not found" {
			response.NotFound(c, "Character not found")
			return
		}
		if err.Error() == "unauthorized" {
			response.Forbidden(c, "Not authorized to delete this character")
			return
		}
		response.InternalError(c, "Delete failed")
		return
	}

	response.Success(c, gin.H{"message": "Character deleted"})
}

func (h *CharacterLibraryHandler) ExtractCharacters(c *gin.Context) {
	episodeIDStr := c.Param("episode_id")
	episodeID, err := strconv.ParseUint(episodeIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "Invalid episode_id")
		return
	}

	taskID, err := h.libraryService.ExtractCharactersFromScript(uint(episodeID))
	if err != nil {
		h.log.Errorw("Failed to extract characters", "error", err)
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"task_id": taskID, "message": "Character extraction task submitted"})
}
