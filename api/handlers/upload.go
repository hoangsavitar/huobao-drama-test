package handlers

import (
services2 "github.com/drama-generator/backend/application/services"
"github.com/drama-generator/backend/pkg/config"
"github.com/drama-generator/backend/pkg/logger"
"github.com/drama-generator/backend/pkg/response"
"github.com/gin-gonic/gin"
)

type UploadHandler struct {
uploadService           *services2.UploadService
characterLibraryService *services2.CharacterLibraryService
log                     *logger.Logger
}

func NewUploadHandler(cfg *config.Config, log *logger.Logger, characterLibraryService *services2.CharacterLibraryService) (*UploadHandler, error) {
uploadService, err := services2.NewUploadService(cfg, log)
if err != nil {
return nil, err
}

return &UploadHandler{
uploadService:           uploadService,
characterLibraryService: characterLibraryService,
log:                     log,
}, nil
}

// UploadImage uploads an image
func (h *UploadHandler) UploadImage(c *gin.Context) {
// Get the uploaded file
file, header, err := c.Request.FormFile("file")
if err != nil {
response.BadRequest(c, "Please select a file")
return
}
defer file.Close()

// Check file type
contentType := header.Header.Get("Content-Type")
if contentType == "" {
contentType = "application/octet-stream"
}

// Validate image type
allowedTypes := map[string]bool{
"image/jpeg": true,
"image/jpg":  true,
"image/png":  true,
"image/gif":  true,
"image/webp": true,
}

if !allowedTypes[contentType] {
response.BadRequest(c, "Only image formats are supported (jpg, png, gif, webp)")
return
}

// Check file size (10MB)
if header.Size > 10*1024*1024 {
response.BadRequest(c, "File size cannot exceed 10MB")
return
}

// Upload to local storage
result, err := h.uploadService.UploadCharacterImage(file, header.Filename, contentType)
if err != nil {
h.log.Errorw("Failed to upload image", "error", err)
response.InternalError(c, "Upload failed")
return
}

response.Success(c, gin.H{
"url":        result.URL,
"local_path": result.LocalPath,
"filename":   header.Filename,
"size":       header.Size,
})
}

// UploadCharacterImage uploads a character image (with character ID)
func (h *UploadHandler) UploadCharacterImage(c *gin.Context) {
characterID := c.Param("id")

// Get the uploaded file
file, header, err := c.Request.FormFile("file")
if err != nil {
response.BadRequest(c, "Please select a file")
return
}
defer file.Close()

// Check file type
contentType := header.Header.Get("Content-Type")
if contentType == "" {
contentType = "application/octet-stream"
}

// Validate image type
allowedTypes := map[string]bool{
"image/jpeg": true,
"image/jpg":  true,
"image/png":  true,
"image/gif":  true,
"image/webp": true,
}

if !allowedTypes[contentType] {
response.BadRequest(c, "Only image formats are supported (jpg, png, gif, webp)")
return
}

// Check file size (10MB)
if header.Size > 10*1024*1024 {
response.BadRequest(c, "File size cannot exceed 10MB")
return
}

// Upload to local storage
result, err := h.uploadService.UploadCharacterImage(file, header.Filename, contentType)
if err != nil {
h.log.Errorw("Failed to upload character image", "error", err)
response.InternalError(c, "Upload failed")
return
}

// Update the character image_url field in the database
err = h.characterLibraryService.UploadCharacterImage(characterID, result.URL)
if err != nil {
h.log.Errorw("Failed to update character image_url", "error", err, "character_id", characterID)
response.InternalError(c, "Failed to update character image")
return
}

h.log.Infow("Character image uploaded and saved", "character_id", characterID, "url", result.URL, "local_path", result.LocalPath)

response.Success(c, gin.H{
"url":        result.URL,
"local_path": result.LocalPath,
"filename":   header.Filename,
"size":       header.Size,
})
}
