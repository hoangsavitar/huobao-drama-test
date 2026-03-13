package handlers

import (
"github.com/drama-generator/backend/pkg/response"
"github.com/gin-gonic/gin"
)

// GenerateCharacterImage generates character appearance with AI
func (h *CharacterLibraryHandler) GenerateCharacterImage(c *gin.Context) {

characterID := c.Param("id")

// Get model and style parameters from request body
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
response.Forbidden(c, "Access denied")
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
