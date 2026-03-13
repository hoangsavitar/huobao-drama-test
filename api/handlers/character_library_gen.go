package handlers

import (
	"github.com/drama-generator/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

func (h *CharacterLibraryHandler) GenerateCharacterImage(c *gin.Context) {

	characterID := c.Param("id")

	var req struct {
		Model string `json:"model"`
		Style string `json:"style"`
	}
	c.ShouldBindJSON(&req)

	imageGen, err := h.libraryService.GenerateCharacterImage(characterID, h.imageService, req.Model, req.Style)
	if err != nil {
		if err.Error() == "character not found" {
			response.NotFound(c, "Character not found")
			return
		}
		if err.Error() == "unauthorized" {
			response.Forbidden(c, "No permission")
			return
		}
		h.log.Errorw("Failed to generate character image", "error", err)
		response.InternalError(c, "Generation failed")
		return
	}

	response.Success(c, gin.H{
		"message":          "Character image generation started",
		"image_generation": imageGen,
	})
}
