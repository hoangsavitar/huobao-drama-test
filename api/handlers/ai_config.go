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

type AIConfigHandler struct {
aiService *services.AIService
log       *logger.Logger
}

func NewAIConfigHandler(db *gorm.DB, cfg *config.Config, log *logger.Logger) *AIConfigHandler {
return &AIConfigHandler{
aiService: services.NewAIService(db, log),
log:       log,
}
}

func (h *AIConfigHandler) CreateConfig(c *gin.Context) {
var req services.CreateAIConfigRequest
if err := c.ShouldBindJSON(&req); err != nil {
response.BadRequest(c, err.Error())
return
}

config, err := h.aiService.CreateConfig(&req)
if err != nil {
response.InternalError(c, "Creation failed")
return
}

response.Created(c, config)
}

func (h *AIConfigHandler) GetConfig(c *gin.Context) {

configID, err := strconv.ParseUint(c.Param("id"), 10, 32)
if err != nil {
response.BadRequest(c, "Invalid config ID")
return
}

config, err := h.aiService.GetConfig(uint(configID))
if err != nil {
if err.Error() == "config not found" {
response.NotFound(c, "Config not found")
return
}
response.InternalError(c, "Failed to retrieve")
return
}

response.Success(c, config)
}

func (h *AIConfigHandler) ListConfigs(c *gin.Context) {

serviceType := c.Query("service_type")

configs, err := h.aiService.ListConfigs(serviceType)
if err != nil {
response.InternalError(c, "Failed to retrieve list")
return
}

response.Success(c, configs)
}

func (h *AIConfigHandler) UpdateConfig(c *gin.Context) {

configID, err := strconv.ParseUint(c.Param("id"), 10, 32)
if err != nil {
response.BadRequest(c, "Invalid config ID")
return
}

var req services.UpdateAIConfigRequest
if err := c.ShouldBindJSON(&req); err != nil {
response.BadRequest(c, err.Error())
return
}

config, err := h.aiService.UpdateConfig(uint(configID), &req)
if err != nil {
if err.Error() == "config not found" {
response.NotFound(c, "Config not found")
return
}
response.InternalError(c, "Update failed")
return
}

response.Success(c, config)
}

func (h *AIConfigHandler) DeleteConfig(c *gin.Context) {

configID, err := strconv.ParseUint(c.Param("id"), 10, 32)
if err != nil {
response.BadRequest(c, "Invalid config ID")
return
}

if err := h.aiService.DeleteConfig(uint(configID)); err != nil {
if err.Error() == "config not found" {
response.NotFound(c, "Config not found")
return
}
response.InternalError(c, "Delete failed")
return
}

response.Success(c, gin.H{"message": "Deleted successfully"})
}

func (h *AIConfigHandler) TestConnection(c *gin.Context) {
var req services.TestConnectionRequest
if err := c.ShouldBindJSON(&req); err != nil {
response.BadRequest(c, err.Error())
return
}

if err := h.aiService.TestConnection(&req); err != nil {
response.BadRequest(c, "Connection test failed: "+err.Error())
return
}

response.Success(c, gin.H{"message": "Connection test successful"})
}
