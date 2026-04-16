package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/drama-generator/backend/domain/models"
	"github.com/drama-generator/backend/pkg/config"
	"github.com/drama-generator/backend/pkg/logger"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type DramaService struct {
	db                    *gorm.DB
	log                   *logger.Logger
	baseURL               string
	narrativeFallbackStub bool
	narrativePkg          *NarrativePackageService
}

func NewDramaService(db *gorm.DB, cfg *config.Config, log *logger.Logger, ai *AIService) *DramaService {
	var nps *NarrativePackageService
	if ai != nil {
		nps = NewNarrativePackageService(ai, log, cfg.Narrative.FallbackStub)
	}
	return &DramaService{
		db:                    db,
		log:                   log,
		baseURL:               cfg.Storage.BaseURL,
		narrativeFallbackStub: cfg.Narrative.FallbackStub,
		narrativePkg:          nps,
	}
}

// NarrativeGenerateRequest body for POST /dramas/:id/narrative/generate
// user_idea optional — empty string uses drama title / idea inside GenerateNarrativeEpisodes.
type NarrativeGenerateRequest struct {
	UserIdea string `json:"user_idea"`
}

// NarrativeDramaPackage is the contract returned by narrative_AI /api/pipeline/drama-package or the built-in stub.
type NarrativeDramaPackage struct {
	StartNarrativeNodeID string                  `json:"start_narrative_node_id"`
	Episodes             []NarrativeEpisodeDraft `json:"episodes"`
}

// NarrativeEpisodeDraft one graph node → one huobao episode.
type NarrativeEpisodeDraft struct {
	NarrativeNodeID string                 `json:"narrative_node_id"`
	EpisodeNumber   int                    `json:"episode_number"`
	Title           string                 `json:"title"`
	ScriptContent   string                 `json:"script_content"`
	IsEntry         bool                   `json:"is_entry"`
	Choices         []NarrativeChoiceDraft `json:"choices"`
}

type NarrativeChoiceDraft struct {
	Label               string `json:"label"`
	NextNarrativeNodeID string `json:"next_narrative_node_id"`
}

type episodeChoiceJSON struct {
	Label               string `json:"label"`
	NextNarrativeNodeID string `json:"next_narrative_node_id,omitempty"`
	NextEpisodeID       uint   `json:"next_episode_id,omitempty"`
}

type CreateDramaRequest struct {
	Title       string `json:"title" binding:"required,min=1,max=100"`
	Description string `json:"description"`
	Genre       string `json:"genre"`
	Style       string `json:"style"`
	AspectRatio string `json:"aspect_ratio"`
	Tags        string `json:"tags"`
}

type UpdateDramaRequest struct {
	Title       string `json:"title" binding:"omitempty,min=1,max=100"`
	Description string `json:"description"`
	Genre       string `json:"genre"`
	Style       string `json:"style"`
	AspectRatio string `json:"aspect_ratio"`
	Tags        string `json:"tags"`
	Status      string `json:"status" binding:"omitempty,oneof=draft planning production completed archived"`
}

type DramaListQuery struct {
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=20"`
	Status   string `form:"status"`
	Genre    string `form:"genre"`
	Keyword  string `form:"keyword"`
}

func (s *DramaService) CreateDrama(req *CreateDramaRequest) (*models.Drama, error) {
	drama := &models.Drama{
		Title:       req.Title,
		Status:      "draft",
		Style:       "ghibli",
		AspectRatio: "16:9",
	}

	if req.Description != "" {
		drama.Description = &req.Description
	}
	if req.Genre != "" {
		drama.Genre = &req.Genre
	}
	if req.Style != "" {
		drama.Style = req.Style
	}
	if req.AspectRatio != "" {
		drama.AspectRatio = req.AspectRatio
	}

	if err := s.db.Create(drama).Error; err != nil {
		s.log.Errorw("Failed to create drama", "error", err)
		return nil, err
	}

	s.log.Infow("Drama created", "drama_id", drama.ID)
	return drama, nil
}

func (s *DramaService) GetDrama(dramaID string) (*models.Drama, error) {
	var drama models.Drama
	err := s.db.Where("id = ? ", dramaID).
		Preload("Characters").          // Load Drama-level characters
		Preload("Scenes").              // Load Drama-level scenes
		Preload("Props").               // Load Drama-level props
		Preload("Episodes.Characters"). // Load characters associated with each episode
		Preload("Episodes.Scenes").     // Load scenes associated with each episode
		Preload("Episodes.Storyboards", func(db *gorm.DB) *gorm.DB {
			return db.Order("storyboards.storyboard_number ASC")
		}).
		Preload("Episodes.Storyboards.Background"). // Load scene/background linked by scene_id
		Preload("Episodes.Storyboards.Props").      // Load props associated with storyboards
		Preload("Episodes.Storyboards.Characters"). // Load characters associated with storyboards
		First(&drama).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("drama not found")
		}
		s.log.Errorw("Failed to get drama", "error", err)
		return nil, err
	}

	// Calculate each episode's duration (based on sum of scene durations)
	for i := range drama.Episodes {
		totalDuration := 0
		for _, scene := range drama.Episodes[i].Storyboards {
			totalDuration += scene.Duration
		}
		// Update episode duration (seconds to minutes, rounded up)
		durationMinutes := (totalDuration + 59) / 60
		drama.Episodes[i].Duration = durationMinutes

		// If database duration differs from calculated, update database
		if drama.Episodes[i].Duration != durationMinutes {
			s.db.Model(&models.Episode{}).Where("id = ?", drama.Episodes[i].ID).Update("duration", durationMinutes)
		}

		// Query character image generation status
		for j := range drama.Episodes[i].Characters {
			var imageGen models.ImageGeneration
			// Query in-progress or failed task status
			err := s.db.Where("character_id = ? AND (status = ? OR status = ?)",
				drama.Episodes[i].Characters[j].ID, "pending", "processing").
				Order("created_at DESC").
				First(&imageGen).Error

			if err == nil {
				// Found in-progress record, set status
				statusStr := string(imageGen.Status)
				drama.Episodes[i].Characters[j].ImageGenerationStatus = &statusStr
				if imageGen.ErrorMsg != nil {
					drama.Episodes[i].Characters[j].ImageGenerationError = imageGen.ErrorMsg
				}
			} else if errors.Is(err, gorm.ErrRecordNotFound) {
				// Check for failed records
				err := s.db.Where("character_id = ? AND status = ?",
					drama.Episodes[i].Characters[j].ID, "failed").
					Order("created_at DESC").
					First(&imageGen).Error

				if err == nil {
					statusStr := string(imageGen.Status)
					drama.Episodes[i].Characters[j].ImageGenerationStatus = &statusStr
					if imageGen.ErrorMsg != nil {
						drama.Episodes[i].Characters[j].ImageGenerationError = imageGen.ErrorMsg
					}
				}
			}
		}

		// Query scene image generation status
		for j := range drama.Episodes[i].Scenes {
			var imageGen models.ImageGeneration
			// Query in-progress or failed task status
			err := s.db.Where("scene_id = ? AND (status = ? OR status = ?)",
				drama.Episodes[i].Scenes[j].ID, "pending", "processing").
				Order("created_at DESC").
				First(&imageGen).Error

			if err == nil {
				// Found in-progress record, set status
				statusStr := string(imageGen.Status)
				drama.Episodes[i].Scenes[j].ImageGenerationStatus = &statusStr
				if imageGen.ErrorMsg != nil {
					drama.Episodes[i].Scenes[j].ImageGenerationError = imageGen.ErrorMsg
				}
			} else if errors.Is(err, gorm.ErrRecordNotFound) {
				// Check for failed records
				err := s.db.Where("scene_id = ? AND status = ?",
					drama.Episodes[i].Scenes[j].ID, "failed").
					Order("created_at DESC").
					First(&imageGen).Error

				if err == nil {
					statusStr := string(imageGen.Status)
					drama.Episodes[i].Scenes[j].ImageGenerationStatus = &statusStr
					if imageGen.ErrorMsg != nil {
						drama.Episodes[i].Scenes[j].ImageGenerationError = imageGen.ErrorMsg
					}
				}
			}
		}
	}

	// Consolidate all episode scenes into Drama-level Scenes field
	sceneMap := make(map[uint]*models.Scene) // for deduplication
	for i := range drama.Episodes {
		for j := range drama.Episodes[i].Scenes {
			scene := &drama.Episodes[i].Scenes[j]
			sceneMap[scene.ID] = scene
		}
	}

	// Add consolidated scenes to drama.Scenes
	drama.Scenes = make([]models.Scene, 0, len(sceneMap))
	for _, scene := range sceneMap {
		drama.Scenes = append(drama.Scenes, *scene)
	}

	// Add base_url prefix to all scene local_paths
	// s.addBaseURLToScenes(&drama)

	return &drama, nil
}

func (s *DramaService) ListDramas(query *DramaListQuery) ([]models.Drama, int64, error) {
	var dramas []models.Drama
	var total int64

	db := s.db.Model(&models.Drama{})

	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}

	if query.Genre != "" {
		db = db.Where("genre = ?", query.Genre)
	}

	if query.Keyword != "" {
		db = db.Where("title LIKE ? OR description LIKE ?", "%"+query.Keyword+"%", "%"+query.Keyword+"%")
	}

	if err := db.Count(&total).Error; err != nil {
		s.log.Errorw("Failed to count dramas", "error", err)
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PageSize
	err := db.Order("updated_at DESC").
		Offset(offset).
		Limit(query.PageSize).
		Preload("Episodes.Storyboards", func(db *gorm.DB) *gorm.DB {
			return db.Order("storyboards.storyboard_number ASC")
		}).
		Preload("Episodes.Storyboards.Background").
		Find(&dramas).Error

	if err != nil {
		s.log.Errorw("Failed to list dramas", "error", err)
		return nil, 0, err
	}

	// Calculate each episode's duration for each drama (based on sum of scene durations)
	for i := range dramas {
		for j := range dramas[i].Episodes {
			totalDuration := 0
			for _, scene := range dramas[i].Episodes[j].Storyboards {
				totalDuration += scene.Duration
			}
			// Update episode duration (seconds to minutes, rounded up)
			durationMinutes := (totalDuration + 59) / 60
			dramas[i].Episodes[j].Duration = durationMinutes
		}
	}

	return dramas, total, nil
}

func (s *DramaService) UpdateDrama(dramaID string, req *UpdateDramaRequest) (*models.Drama, error) {
	var drama models.Drama
	if err := s.db.Where("id = ? ", dramaID).First(&drama).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("drama not found")
		}
		return nil, err
	}

	updates := make(map[string]interface{})

	if req.Title != "" {
		updates["title"] = req.Title
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Genre != "" {
		updates["genre"] = req.Genre
	}
	if req.Style != "" {
		updates["style"] = req.Style
	}
	if req.AspectRatio != "" {
		updates["aspect_ratio"] = req.AspectRatio
	}
	if req.Tags != "" {
		updates["tags"] = req.Tags
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}

	updates["updated_at"] = time.Now()

	if err := s.db.Model(&drama).Updates(updates).Error; err != nil {
		s.log.Errorw("Failed to update drama", "error", err)
		return nil, err
	}

	s.log.Infow("Drama updated", "drama_id", dramaID)
	return &drama, nil
}

func (s *DramaService) DeleteDrama(dramaID string) error {
	result := s.db.Where("id = ? ", dramaID).Delete(&models.Drama{})

	if result.Error != nil {
		s.log.Errorw("Failed to delete drama", "error", result.Error)
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("drama not found")
	}

	s.log.Infow("Drama deleted", "drama_id", dramaID)
	return nil
}

func (s *DramaService) GetDramaStats() (map[string]interface{}, error) {
	var total int64
	var byStatus []struct {
		Status string
		Count  int64
	}

	if err := s.db.Model(&models.Drama{}).Count(&total).Error; err != nil {
		return nil, err
	}

	if err := s.db.Model(&models.Drama{}).
		Select("status, count(*) as count").
		Group("status").
		Scan(&byStatus).Error; err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"total":     total,
		"by_status": byStatus,
	}

	return stats, nil
}

type SaveOutlineRequest struct {
	Title   string   `json:"title" binding:"required"`
	Summary string   `json:"summary" binding:"required"`
	Genre   string   `json:"genre"`
	Tags    []string `json:"tags"`
}

type SaveCharactersRequest struct {
	Characters []models.Character `json:"characters" binding:"required"`
	EpisodeID  *uint              `json:"episode_id"` // Optional: associate with specified episode if provided
}

type SaveProgressRequest struct {
	CurrentStep string                 `json:"current_step" binding:"required"`
	StepData    map[string]interface{} `json:"step_data"`
}

type SaveEpisodesRequest struct {
	Episodes []models.Episode `json:"episodes" binding:"required"`
}

func (s *DramaService) SaveOutline(dramaID string, req *SaveOutlineRequest) error {
	var drama models.Drama
	if err := s.db.Where("id = ? ", dramaID).First(&drama).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("drama not found")
		}
		return err
	}

	updates := map[string]interface{}{
		"title":       req.Title,
		"description": req.Summary,
		"updated_at":  time.Now(),
	}

	if req.Genre != "" {
		updates["genre"] = req.Genre
	}

	if len(req.Tags) > 0 {
		tagsJSON, err := json.Marshal(req.Tags)
		if err != nil {
			s.log.Errorw("Failed to marshal tags", "error", err)
			return err
		}
		updates["tags"] = tagsJSON
	}

	if err := s.db.Model(&drama).Updates(updates).Error; err != nil {
		s.log.Errorw("Failed to save outline", "error", err)
		return err
	}

	s.log.Infow("Outline saved", "drama_id", dramaID)
	return nil
}

func (s *DramaService) GetCharacters(dramaID string, episodeID *string) ([]models.Character, error) {
	var drama models.Drama
	if err := s.db.Where("id = ? ", dramaID).First(&drama).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("drama not found")
		}
		return nil, err
	}

	var characters []models.Character

	// If episodeID is specified, only get characters associated with that episode
	if episodeID != nil {
		var episode models.Episode
		if err := s.db.Preload("Characters").Where("id = ? AND drama_id = ?", *episodeID, dramaID).First(&episode).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.New("episode not found")
			}
			return nil, err
		}
		characters = episode.Characters
	} else {
		// If episodeID is not specified, get all characters for the project
		if err := s.db.Where("drama_id = ?", dramaID).Find(&characters).Error; err != nil {
			s.log.Errorw("Failed to get characters", "error", err)
			return nil, err
		}
	}

	// Query each character's image generation task status
	for i := range characters {
		// Query the character's latest image generation task
		var imageGen models.ImageGeneration
		err := s.db.Where("character_id = ?", characters[i].ID).
			Order("created_at DESC").
			First(&imageGen).Error

		if err == nil {
			// If there's an in-progress task, populate status info
			if imageGen.Status == models.ImageStatusPending || imageGen.Status == models.ImageStatusProcessing {
				statusStr := string(imageGen.Status)
				characters[i].ImageGenerationStatus = &statusStr
			} else if imageGen.Status == models.ImageStatusFailed {
				statusStr := "failed"
				characters[i].ImageGenerationStatus = &statusStr
				if imageGen.ErrorMsg != nil {
					characters[i].ImageGenerationError = imageGen.ErrorMsg
				}
			}
		}
	}

	return characters, nil
}

func (s *DramaService) SaveCharacters(dramaID string, req *SaveCharactersRequest) error {
	// Convert dramaID
	id, err := strconv.ParseUint(dramaID, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid drama ID")
	}
	dramaIDUint := uint(id)

	var drama models.Drama
	if err := s.db.Where("id = ? ", dramaIDUint).First(&drama).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("drama not found")
		}
		return err
	}

	// If EpisodeID is specified, verify episode existence
	if req.EpisodeID != nil {
		var episode models.Episode
		if err := s.db.Where("id = ? AND drama_id = ?", *req.EpisodeID, dramaIDUint).First(&episode).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("episode not found")
			}
			return err
		}
	}

	// Get all existing characters for the project
	var existingCharacters []models.Character
	if err := s.db.Where("drama_id = ?", dramaIDUint).Find(&existingCharacters).Error; err != nil {
		s.log.Errorw("Failed to get existing characters", "error", err)
		return err
	}

	// Create character name to character mapping
	existingCharMap := make(map[string]*models.Character)
	for i := range existingCharacters {
		existingCharMap[existingCharacters[i].Name] = &existingCharacters[i]
	}

	// Collect character IDs to associate with episode
	var characterIDs []uint

	// Create new characters or reuse/update existing ones
	for _, char := range req.Characters {
		// 1. If ID is provided, try to update existing character
		if char.ID > 0 {
			var existing models.Character
			if err := s.db.Where("id = ? AND drama_id = ?", char.ID, dramaIDUint).First(&existing).Error; err == nil {
				// Update character info
				updates := map[string]interface{}{
					"name":        char.Name,
					"role":        char.Role,
					"description": char.Description,
					"personality": char.Personality,
					"appearance":  char.Appearance,
					"image_url":   char.ImageURL,
				}
				if err := s.db.Model(&existing).Updates(updates).Error; err != nil {
					s.log.Errorw("Failed to update character", "error", err, "id", char.ID)
				}
				characterIDs = append(characterIDs, existing.ID)
				continue
			}
		}

		// 2. If no ID but name exists, reuse directly (optional: could also update)
		if existingChar, exists := existingCharMap[char.Name]; exists {
			s.log.Infow("Character already exists, reusing", "name", char.Name, "character_id", existingChar.ID)
			characterIDs = append(characterIDs, existingChar.ID)
			continue
		}

		// 3. Character does not exist, create new character
		character := models.Character{
			DramaID:     dramaIDUint,
			Name:        char.Name,
			Role:        char.Role,
			Description: char.Description,
			Personality: char.Personality,
			Appearance:  char.Appearance,
			ImageURL:    char.ImageURL,
		}

		if err := s.db.Create(&character).Error; err != nil {
			s.log.Errorw("Failed to create character", "error", err, "name", char.Name)
			continue
		}

		s.log.Infow("New character created", "character_id", character.ID, "name", char.Name)
		characterIDs = append(characterIDs, character.ID)
	}

	// If EpisodeID is specified, establish character-episode association
	if req.EpisodeID != nil && len(characterIDs) > 0 {
		var episode models.Episode
		if err := s.db.First(&episode, *req.EpisodeID).Error; err != nil {
			return err
		}

		// Get character objects
		var characters []models.Character
		if err := s.db.Where("id IN ?", characterIDs).Find(&characters).Error; err != nil {
			s.log.Errorw("Failed to get characters", "error", err)
			return err
		}

		// Use GORM Association API to establish many-to-many relationship (auto-deduplicates)
		if err := s.db.Model(&episode).Association("Characters").Append(&characters); err != nil {
			s.log.Errorw("Failed to associate characters with episode", "error", err)
			return err
		}

		s.log.Infow("Characters associated with episode", "episode_id", *req.EpisodeID, "character_count", len(characterIDs))
	}

	if err := s.db.Model(&drama).Update("updated_at", time.Now()).Error; err != nil {
		s.log.Errorw("Failed to update drama timestamp", "error", err)
	}

	s.log.Infow("Characters saved", "drama_id", dramaID, "count", len(req.Characters))
	return nil
}

func (s *DramaService) SaveEpisodes(dramaID string, req *SaveEpisodesRequest) error {
	return s.saveEpisodes(dramaID, req, false)
}

// deleteAllEpisodesForDrama removes every episode row (and bound scenes/storyboards) for a drama.
// Used before inserting a fresh narrative graph so UNIQUE(drama_id, narrative_node_id) never collides
// with stale rows (GORM Update(column, nil) may not clear SQLite reliably; update-in-place can still race unique).
func (s *DramaService) deleteAllEpisodesForDrama(dramaIDUint uint) error {
	var eps []models.Episode
	if err := s.db.Where("drama_id = ?", dramaIDUint).Find(&eps).Error; err != nil {
		return err
	}
	for _, existing := range eps {
		// many2many join table name from GORM default — ignore if missing
		_ = s.db.Exec("DELETE FROM episode_characters WHERE episode_id = ?", existing.ID)
		if err := s.db.Where("episode_id = ?", existing.ID).Delete(&models.Scene{}).Error; err != nil {
			return fmt.Errorf("delete scenes episode %d: %w", existing.ID, err)
		}
		if err := s.db.Where("episode_id = ?", existing.ID).Delete(&models.Storyboard{}).Error; err != nil {
			return fmt.Errorf("delete storyboards episode %d: %w", existing.ID, err)
		}
		if err := s.db.Delete(&existing).Error; err != nil {
			return fmt.Errorf("delete episode %d: %w", existing.ID, err)
		}
	}
	if len(eps) > 0 {
		s.log.Infow("Deleted previous episodes for narrative replace", "drama_id", dramaIDUint, "count", len(eps))
	}
	return nil
}

// saveEpisodes persists episode rows. If replaceAllEpisodesForNarrative is true, deletes all current
// episodes for the drama first, then inserts the payload as new rows (narrative regenerate only).
func (s *DramaService) saveEpisodes(dramaID string, req *SaveEpisodesRequest, replaceAllEpisodesForNarrative bool) error {
	// Convert dramaID
	id, err := strconv.ParseUint(dramaID, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid drama ID")
	}
	dramaIDUint := uint(id)

	var drama models.Drama
	if err := s.db.Where("id = ? ", dramaIDUint).First(&drama).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("drama not found")
		}
		return err
	}

	if replaceAllEpisodesForNarrative {
		if err := s.deleteAllEpisodesForDrama(dramaIDUint); err != nil {
			return err
		}
	}

	// Get existing episodes for mapping
	var existingEpisodes []models.Episode
	if err := s.db.Where("drama_id = ?", dramaIDUint).Find(&existingEpisodes).Error; err != nil {
		s.log.Errorw("Failed to fetch existing episodes", "error", err)
		return err
	}

	existingMap := make(map[int]models.Episode)
	for _, ep := range existingEpisodes {
		existingMap[ep.EpisodeNum] = ep
	}

	incomingMap := make(map[int]bool)

	// Create or update episodes
	for _, ep := range req.Episodes {
		incomingMap[ep.EpisodeNum] = true

		if existing, exists := existingMap[ep.EpisodeNum]; exists {
			// Update existing episode
			updates := map[string]interface{}{
				"title":          ep.Title,
				"description":    ep.Description,
				"script_content": ep.ScriptContent,
				"duration":       ep.Duration,
			}
			if ep.Status != "" {
				updates["status"] = ep.Status
			}
			// Interactive / narrative fields: only touch when client sends branching metadata
			// so normal workflow saves (script only) do not wipe narrative_node_id / choices / is_entry.
			narrativeTouch := ep.NarrativeNodeID != nil || len(ep.Choices) > 0 || ep.IsEntry
			if narrativeTouch {
				if ep.NarrativeNodeID != nil {
					updates["narrative_node_id"] = *ep.NarrativeNodeID
				}
				if len(ep.Choices) > 0 {
					updates["choices"] = ep.Choices
				}
				updates["is_entry"] = ep.IsEntry
			}

			if err := s.db.Model(&existing).Updates(updates).Error; err != nil {
				s.log.Errorw("Failed to update episode", "error", err, "episode", ep.EpisodeNum)
				return fmt.Errorf("update episode %d: %w", ep.EpisodeNum, err)
			}
		} else {
			// Create new episode
			status := ep.Status
			if status == "" {
				status = "draft"
			}

			episode := models.Episode{
				DramaID:         dramaIDUint,
				EpisodeNum:      ep.EpisodeNum,
				Title:           ep.Title,
				NarrativeNodeID: ep.NarrativeNodeID,
				Choices:         ep.Choices,
				IsEntry:         ep.IsEntry,
				Description:     ep.Description,
				ScriptContent:   ep.ScriptContent,
				Duration:        ep.Duration,
				Status:          status,
			}

			if err := s.db.Create(&episode).Error; err != nil {
				s.log.Errorw("Failed to create episode", "error", err, "episode", ep.EpisodeNum)
				return fmt.Errorf("create episode %d: %w", ep.EpisodeNum, err)
			}
		}
	}

	// Delete episodes that are no longer in the request
	for epNum, existing := range existingMap {
		if !incomingMap[epNum] {
			// Soft delete related scenes, storyboards, and other entities
			if err := s.db.Where("episode_id = ?", existing.ID).Delete(&models.Scene{}).Error; err != nil {
				s.log.Errorw("Failed to cascade delete scenes", "error", err, "episode", epNum)
				return fmt.Errorf("delete scenes for episode %d: %w", epNum, err)
			}
			if err := s.db.Where("episode_id = ?", existing.ID).Delete(&models.Storyboard{}).Error; err != nil {
				s.log.Errorw("Failed to cascade delete storyboards", "error", err, "episode", epNum)
				return fmt.Errorf("delete storyboards for episode %d: %w", epNum, err)
			}
			// (Note: StoryboardCharacters and Props might not have soft deletes or episode bindings depending on schema, so we stick to scenes and storyboards which are episode-bound and soft-deleted)

			if err := s.db.Delete(&existing).Error; err != nil {
				s.log.Errorw("Failed to delete episode", "error", err, "episode", epNum)
				return fmt.Errorf("delete episode %d: %w", epNum, err)
			}
		}
	}

	if err := s.db.Model(&drama).Update("updated_at", time.Now()).Error; err != nil {
		s.log.Errorw("Failed to update drama timestamp", "error", err)
	}

	s.log.Infow("Episodes saved", "drama_id", dramaID, "count", len(req.Episodes))
	return nil
}

func (s *DramaService) SaveProgress(dramaID string, req *SaveProgressRequest) error {
	var drama models.Drama
	if err := s.db.Where("id = ? ", dramaID).First(&drama).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("drama not found")
		}
		return err
	}

	// Build metadata object
	metadata := make(map[string]interface{})

	// Preserve existing metadata
	if drama.Metadata != nil {
		if err := json.Unmarshal(drama.Metadata, &metadata); err != nil {
			s.log.Warnw("Failed to unmarshal existing metadata", "error", err)
		}
	}

	// Update progress info
	metadata["current_step"] = req.CurrentStep
	if req.StepData != nil {
		metadata["step_data"] = req.StepData
	}

	// Serialize metadata
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		s.log.Errorw("Failed to marshal metadata", "error", err)
		return err
	}

	updates := map[string]interface{}{
		"metadata":   metadataJSON,
		"updated_at": time.Now(),
	}

	if err := s.db.Model(&drama).Updates(updates).Error; err != nil {
		s.log.Errorw("Failed to save progress", "error", err)
		return err
	}

	s.log.Infow("Progress saved", "drama_id", dramaID, "step", req.CurrentStep)
	return nil
}

// GenerateNarrativeEpisodes builds branching episodes via Text AI + embedded prompts (NarrativePackageService), optional template DAG if fallback_stub.
// Replaces the drama's episode list with the generated package (same semantics as SaveEpisodes).
func (s *DramaService) GenerateNarrativeEpisodes(dramaID string, userIdea string) error {
	id, err := strconv.ParseUint(dramaID, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid drama ID")
	}
	var drama models.Drama
	if err := s.db.Where("id = ?", uint(id)).First(&drama).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("drama not found")
		}
		return err
	}

	var pkg *NarrativeDramaPackage
	if s.narrativePkg != nil {
		s.log.Infow("Narrative generate: Huobao text AI + embedded prompts")
		pkg, err = s.narrativePkg.BuildPackage(userIdea, drama)
		if err != nil {
			return err
		}
	} else {
		if !s.narrativeFallbackStub {
			return fmt.Errorf("narrative: AI service not available; configure Text AI (Settings) or narrative.fallback_stub=true for offline template")
		}
		s.log.Infow("Narrative generate: no NarrativePackageService, using stub (fallback_stub)")
		pkg = BuildStubNarrativeDramaPackage(strings.TrimSpace(userIdea), drama.Title)
	}

	if len(pkg.Episodes) == 0 {
		return fmt.Errorf("narrative package has no episodes")
	}

	episodes := make([]models.Episode, 0, len(pkg.Episodes))
	for _, d := range pkg.Episodes {
		nodeCopy := d.NarrativeNodeID
		choiceRows := make([]episodeChoiceJSON, 0, len(d.Choices))
		for _, c := range d.Choices {
			choiceRows = append(choiceRows, episodeChoiceJSON{
				Label:               c.Label,
				NextNarrativeNodeID: c.NextNarrativeNodeID,
			})
		}
		choicesJSON := datatypes.JSON("[]")
		if len(choiceRows) > 0 {
			raw, mErr := json.Marshal(choiceRows)
			if mErr != nil {
				return mErr
			}
			choicesJSON = raw
		}
		script := stripUIUXBlock(d.ScriptContent)
		episodes = append(episodes, models.Episode{
			EpisodeNum:      d.EpisodeNumber,
			Title:           d.Title,
			NarrativeNodeID: &nodeCopy,
			Choices:         choicesJSON,
			IsEntry:         d.IsEntry,
			ScriptContent:   &script,
			Status:          "draft",
		})
	}

	// Exactly one entry node when package provides start id
	if pkg.StartNarrativeNodeID != "" {
		for i := range episodes {
			nid := *episodes[i].NarrativeNodeID
			if nid == pkg.StartNarrativeNodeID {
				episodes[i].IsEntry = true
			} else if len(pkg.Episodes) > 1 {
				episodes[i].IsEntry = false
			}
		}
	}

	if err := s.saveEpisodes(dramaID, &SaveEpisodesRequest{Episodes: episodes}, true); err != nil {
		return err
	}
	if err := s.resolveEpisodeChoiceNarrativeIDs(uint(id)); err != nil {
		return err
	}
	return s.db.Model(&models.Drama{}).Where("id = ?", uint(id)).Updates(map[string]interface{}{
		"total_episodes": len(pkg.Episodes),
		"updated_at":     time.Now(),
	}).Error
}

// stripUIUXBlock removes Agent 3 "## UI/UX ..." tail so huobao Split Shots does not treat button copy as shots.
func stripUIUXBlock(script string) string {
	script = strings.TrimSpace(script)
	if script == "" {
		return script
	}
	low := strings.ToLower(script)
	markers := []string{"\r\n## ui/ux states", "\n## ui/ux states", "\r\n## ui/ux", "\n## ui/ux"}
	cut := -1
	for _, m := range markers {
		if idx := strings.Index(low, m); idx >= 0 {
			if cut < 0 || idx < cut {
				cut = idx
			}
		}
	}
	if cut >= 0 {
		return strings.TrimSpace(script[:cut])
	}
	return script
}

func (s *DramaService) resolveEpisodeChoiceNarrativeIDs(dramaID uint) error {
	var eps []models.Episode
	if err := s.db.Where("drama_id = ?", dramaID).Find(&eps).Error; err != nil {
		return err
	}
	nodeToID := make(map[string]uint)
	for _, e := range eps {
		if e.NarrativeNodeID != nil && *e.NarrativeNodeID != "" {
			nodeToID[*e.NarrativeNodeID] = e.ID
		}
	}
	for _, e := range eps {
		if len(e.Choices) == 0 {
			continue
		}
		var choices []episodeChoiceJSON
		if err := json.Unmarshal(e.Choices, &choices); err != nil {
			s.log.Warnw("episode choices json", "episode_id", e.ID, "error", err)
			continue
		}
		changed := false
		for i := range choices {
			if choices[i].NextNarrativeNodeID == "" || choices[i].NextEpisodeID != 0 {
				continue
			}
			if tid, ok := nodeToID[choices[i].NextNarrativeNodeID]; ok {
				choices[i].NextEpisodeID = tid
				changed = true
			}
		}
		if !changed {
			continue
		}
		out, err := json.Marshal(choices)
		if err != nil {
			return err
		}
		if err := s.db.Model(&models.Episode{}).Where("id = ?", e.ID).Update("choices", datatypes.JSON(out)).Error; err != nil {
			return err
		}
	}
	return nil
}

// addBaseURLToScenes adds base_url prefix to local_path for all scenes in the drama
func (s *DramaService) addBaseURLToScenes(drama *models.Drama) {
	// Process drama.Scenes
	for i := range drama.Scenes {
		if drama.Scenes[i].LocalPath != nil && *drama.Scenes[i].LocalPath != "" {
			fullPath := fmt.Sprintf("%s/%s", s.baseURL, *drama.Scenes[i].LocalPath)
			drama.Scenes[i].LocalPath = &fullPath
		}
	}

	// Process drama.Episodes[].Scenes
	for i := range drama.Episodes {
		for j := range drama.Episodes[i].Scenes {
			if drama.Episodes[i].Scenes[j].LocalPath != nil && *drama.Episodes[i].Scenes[j].LocalPath != "" {
				fullPath := fmt.Sprintf("%s/%s", s.baseURL, *drama.Episodes[i].Scenes[j].LocalPath)
				drama.Episodes[i].Scenes[j].LocalPath = &fullPath
			}
		}
	}
}
