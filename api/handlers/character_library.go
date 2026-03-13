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

// ListLibraryItems retrieves the character library list
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
response.InternalError(c, "Failed to retrieve character library")
return
}

response.SuccessWithPagination(c, items, total, query.Page, query.PageSize)
}

// CreateLibraryItem adds an item to the character library
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

// GetLibraryItem retrieves character library item details
func (h *CharacterLibraryHandler) GetLibraryItem(c *gin.Context) {

itemID := c.Param("id")

item, err := h.libraryService.GetLibraryItem(itemID)
if err != nil {
if err.Error() == "library item not found" {
response.NotFound(c, "Character library item not found")
return
}
h.log.Errorw("Failed to get library item", "error", err)
response.InternalError(c, "Failed to retrieve")
return
}

response.Success(c, item)
}

// DeleteLibraryItem deletes a character library item
func (h *CharacterLibraryHandler) DeleteLibraryItem(c *gin.Context) {

itemID := c.Param("id")

if err := h.libraryService.DeleteLibraryItem(itemID); err != nil {
if err.Error() == "library item not found" {
response.NotFound(c, "Character library item not found")
return
}
h.log.Errorw("Failed to delete library item", "error", err)
response.InternalError(c, "Delete failed")
return
}

response.Success(c, gin.H{"message": "Deleted successfully"})
}

// UploadCharacterImage uploads a character image
func (h *CharacterLibraryHandler) UploadCharacterImage(c *gin.Context) {

characterID := c.Param("id")

// TODO: Handle file upload
// File upload logic needs to be implemented here, saving to OSS or local storage
// Using a simple implementation for now
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
response.Forbidden(c, "Access denied")
return
}
h.log.Errorw("Failed to upload character image", "error", err)
response.InternalError(c, "Upload failed")
return
}

response.Success(c, gin.H{"message": "Upload successful"})
}

// ApplyLibraryItemToCharacter applies an appearance from the character library
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
response.NotFound(c, "Character library item not found")
return
}
if err.Error() == "character not found" {
response.NotFound(c, "Character not found")
return
}
if err.Error() == "unauthorized" {
response.Forbidden(c, "Access denied")
return
}
h.log.Errorw("Failed to apply library item", "error", err)
response.InternalError(c, "Apply failed")
return
}

response.Success(c, gin.H{"message": "Applied successfully"})
}

// AddCharacterToLibrary adds a character to the character library
func (h *CharacterLibraryHandler) AddCharacterToLibrary(c *gin.Context) {

characterID := c.Param("id")

var req struct {
Category *string `json:"category"`
}

if err := c.ShouldBindJSON(&req); err != nil {
// Allow empty body
req.Category = nil
}

item, err := h.libraryService.AddCharacterToLibrary(characterID, req.Category)
if err != nil {
if err.Error() == "character not found" {
response.NotFound(c, "Character not found")
return
}
if err.Error() == "unauthorized" {
response.Forbidden(c, "Access denied")
return
}
if err.Error() == "character has no image" {
response.BadRequest(c, "Character has no appearance image yet")
return
}
h.log.Errorw("Failed to add character to library", "error", err)
response.InternalError(c, "Failed to add")
return
}

response.Created(c, item)
}

// UpdateCharacter updates character information
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
response.Forbidden(c, "Access denied")
return
}
h.log.Errorw("Failed to update character", "error", err)
response.InternalError(c, "Update failed")
return
}

response.Success(c, gin.H{"message": "Updated successfully"})
}

// DeleteCharacter deletes a single character
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
response.Forbidden(c, "No permission to delete this character")
return
}
response.InternalError(c, "Delete failed")
return
}

response.Success(c, gin.H{"message": "Character deleted"})
}

// ExtractCharacters extracts characters from a script
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
