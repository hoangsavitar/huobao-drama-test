package handlers

import (
"fmt"
"github.com/drama-generator/backend/domain/models"
"github.com/drama-generator/backend/pkg/logger"
"github.com/drama-generator/backend/pkg/response"
"github.com/gin-gonic/gin"
"gorm.io/gorm"
)

// GetStoryboardFramePrompts queries all frame prompts for a storyboard
// GET /api/v1/storyboards/:id/frame-prompts
func GetStoryboardFramePrompts(db *gorm.DB, log *logger.Logger) gin.HandlerFunc {
return func(c *gin.Context) {
storyboardID := c.Param("id")

var framePrompts []models.FramePrompt
if err := db.Where("storyboard_id = ?", storyboardID).
Order("created_at DESC").
Find(&framePrompts).Error; err != nil {
log.Errorw("Failed to query frame prompts", "error", err)
response.InternalError(c, err.Error())
return
}

response.Success(c, gin.H{
"frame_prompts": framePrompts,
})
}
}

// GetEpisodeFramePrompts returns all frame prompts for every storyboard in an episode,
// keyed by storyboard_id.
// GET /api/v1/episodes/:episode_id/frame-prompts
func GetEpisodeFramePrompts(db *gorm.DB, log *logger.Logger) gin.HandlerFunc {
return func(c *gin.Context) {
episodeID := c.Param("episode_id")

// Fetch all storyboard IDs for this episode
var storyboardIDs []uint
if err := db.Model(&models.Storyboard{}).
Where("episode_id = ?", episodeID).
Pluck("id", &storyboardIDs).Error; err != nil {
log.Errorw("Failed to query storyboard IDs", "error", err, "episode_id", episodeID)
response.InternalError(c, err.Error())
return
}

if len(storyboardIDs) == 0 {
response.Success(c, gin.H{"frame_prompts_by_storyboard": map[string]interface{}{}})
return
}

var framePrompts []models.FramePrompt
if err := db.Where("storyboard_id IN ?", storyboardIDs).
Order("created_at DESC").
Find(&framePrompts).Error; err != nil {
log.Errorw("Failed to query episode frame prompts", "error", err, "episode_id", episodeID)
response.InternalError(c, err.Error())
return
}

// Group by storyboard_id
grouped := make(map[uint][]models.FramePrompt)
for _, fp := range framePrompts {
grouped[fp.StoryboardID] = append(grouped[fp.StoryboardID], fp)
}

response.Success(c, gin.H{"frame_prompts_by_storyboard": grouped})
}
}

// UpdateFramePrompt saves or overwrites a frame prompt (user-edited text)
// PUT /api/v1/storyboards/:id/frame-prompt
func UpdateFramePrompt(db *gorm.DB, log *logger.Logger) gin.HandlerFunc {
return func(c *gin.Context) {
storyboardID := c.Param("id")

var req struct {
FrameType string `json:"frame_type" binding:"required"`
Prompt    string `json:"prompt"`
}
if err := c.ShouldBindJSON(&req); err != nil {
response.BadRequest(c, err.Error())
return
}

// Delete existing record(s) for this storyboard + frame_type
if err := db.Where("storyboard_id = ? AND frame_type = ?", storyboardID, req.FrameType).
Delete(&models.FramePrompt{}).Error; err != nil {
log.Errorw("Failed to delete old frame prompt", "error", err)
response.InternalError(c, err.Error())
return
}

// If prompt is empty, just delete and return (clear prompt)
if req.Prompt == "" {
response.Success(c, gin.H{"frame_prompt": nil})
return
}

var sbID uint
fmt.Sscanf(storyboardID, "%d", &sbID)
fp := models.FramePrompt{
StoryboardID: sbID,
FrameType:    req.FrameType,
Prompt:       req.Prompt,
}
if err := db.Create(&fp).Error; err != nil {
log.Errorw("Failed to save frame prompt", "error", err)
response.InternalError(c, err.Error())
return
}

response.Success(c, gin.H{"frame_prompt": fp})
}
}
