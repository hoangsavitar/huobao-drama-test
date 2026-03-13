package handlers

import (
"github.com/drama-generator/backend/application/services"
"github.com/drama-generator/backend/pkg/config"
"github.com/drama-generator/backend/pkg/logger"
"github.com/drama-generator/backend/pkg/response"
"github.com/gin-gonic/gin"
"gorm.io/gorm"
)

type ScriptGenerationHandler struct {
scriptService *services.ScriptGenerationService
taskService   *services.TaskService
log           *logger.Logger
}

func NewScriptGenerationHandler(db *gorm.DB, cfg *config.Config, log *logger.Logger) *ScriptGenerationHandler {
return &ScriptGenerationHandler{
scriptService: services.NewScriptGenerationService(db, cfg, log),
taskService:   services.NewTaskService(db, log),
log:           log,
}
}

func (h *ScriptGenerationHandler) GenerateCharacters(c *gin.Context) {
var req services.GenerateCharactersRequest
if err := c.ShouldBindJSON(&req); err != nil {
response.BadRequest(c, err.Error())
return
}

// Directly call the async service method, which creates a task and returns a task ID
taskID, err := h.scriptService.GenerateCharacters(&req)
if err != nil {
h.log.Errorw("Failed to generate characters", "error", err, "drama_id", req.DramaID)
response.InternalError(c, err.Error())
return
}

// Return task ID immediately
response.Success(c, gin.H{
"task_id": taskID,
"status":  "pending",
"message": "Character generation task created, processing in background...",
})
}
