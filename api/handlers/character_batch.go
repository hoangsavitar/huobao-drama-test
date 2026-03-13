package handlers

import (
	"github.com/drama-generator/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

func (h *CharacterLibraryHandler) BatchGenerateCharacterImages(c *gin.Context) {

	var req struct {
		CharacterIDs []string `json:"character_ids" binding:"required,min=1"`
		Model        string   `json:"model"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if len(req.CharacterIDs) > 10 {
		response.BadRequest(c, "Up to 10 characters per batch")
		return
	}

	go h.libraryService.BatchGenerateCharacterImages(req.CharacterIDs, h.imageService, req.Model)

	response.Success(c, gin.H{
		"message": "Batch generation task submitted",
		"count":   len(req.CharacterIDs),
	})
}
