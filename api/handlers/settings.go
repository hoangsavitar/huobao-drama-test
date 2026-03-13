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

func (h *SettingsHandler) GetLanguage(c *gin.Context) {
	language := h.config.App.Language
	if language == "" {
		language = "en"
	}

	response.Success(c, gin.H{
		"language": language,
	})
}

func (h *SettingsHandler) UpdateLanguage(c *gin.Context) {
	var req struct {
		Language string `json:"language" binding:"required,oneof=zh en"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid language parameter, only zh or en is allowed")
		return
	}

	h.config.App.Language = req.Language

	viper.Set("app.language", req.Language)
	if err := viper.WriteConfig(); err != nil {
		h.log.Warnw("Failed to write config file", "error", err)
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
