package handlers

import (
"github.com/drama-generator/backend/pkg/config"
"github.com/drama-generator/backend/pkg/logger"
"github.com/drama-generator/backend/pkg/response"
"github.com/gin-gonic/gin"
"github.com/spf13/viper"
)

type SettingsHandler struct {
config *config.Config
log    *logger.Logger
}

func NewSettingsHandler(cfg *config.Config, log *logger.Logger) *SettingsHandler {
return &SettingsHandler{
config: cfg,
log:    log,
}
}

// GetLanguage retrieves the current system language
func (h *SettingsHandler) GetLanguage(c *gin.Context) {
language := h.config.App.Language
if language == "" {
language = "zh" // Default: Chinese
}

response.Success(c, gin.H{
"language": language,
})
}

// UpdateLanguage updates the system language
func (h *SettingsHandler) UpdateLanguage(c *gin.Context) {
var req struct {
Language string `json:"language" binding:"required,oneof=zh en"`
}

if err := c.ShouldBindJSON(&req); err != nil {
response.BadRequest(c, "Invalid language parameter, only zh or en are supported")
return
}

// Update in-memory config
h.config.App.Language = req.Language

// Update config file
viper.Set("app.language", req.Language)
if err := viper.WriteConfig(); err != nil {
h.log.Warnw("Failed to write config file", "error", err)
// Even if writing to file fails, in-memory config is already updated and still usable
}

h.log.Infow("System language updated", "language", req.Language)

message := "Language switched to Chinese"
if req.Language == "en" {
message = "Language switched to English"
}

response.Success(c, gin.H{
"message":  message,
"language": req.Language,
})
}
