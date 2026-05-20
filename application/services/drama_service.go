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
	taskService           *TaskService
	imageGeneration       *ImageGenerationService
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
		taskService:           NewTaskService(db, log),
	}
}

func (s *DramaService) SetImageGenerationService(imageGeneration *ImageGenerationService) {
	s.imageGeneration = imageGeneration
}

// NarrativeGenerateRequest body for POST /dramas/:id/narrative/generate
// user_idea optional — empty string uses drama title / idea inside GenerateNarrativeEpisodes.
type NarrativeGenerateRequest struct {
	UserIdea  string `json:"user_idea"`
	AgentStep int    `json:"agent_step"`
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
	Title         string `json:"title" binding:"required,min=1,max=100"`
	Description   string `json:"description"`
	NarrativeIdea string `json:"narrative_idea"`
	Genre         string `json:"genre"`
	Style         string `json:"style"`
	AspectRatio   string `json:"aspect_ratio"`
	Tags          string `json:"tags"`
}

type UpdateDramaRequest struct {
	Title         string  `json:"title" binding:"omitempty,min=1,max=100"`
	Description   string  `json:"description"`
	NarrativeIdea *string `json:"narrative_idea"`
	Genre         string  `json:"genre"`
	Style         string  `json:"style"`
	AspectRatio   string  `json:"aspect_ratio"`
	Tags          string  `json:"tags"`
	Status        string  `json:"status" binding:"omitempty,oneof=draft planning production completed archived"`
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
	if req.NarrativeIdea != "" {
		drama.NarrativeIdea = &req.NarrativeIdea
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
		Preload("Characters").                  // Load Drama-level characters
		Preload("Characters.Outfits").          // Load Outfits for each character
		Preload("Scenes").                      // Load Drama-level scenes
		Preload("Props").                       // Load Drama-level props
		Preload("Episodes.Characters").         // Load characters associated with each episode
		Preload("Episodes.Characters.Outfits"). // Load outfits for episode characters
		Preload("Episodes.Scenes").             // Load scenes associated with each episode
		Preload("Episodes.Storyboards", func(db *gorm.DB) *gorm.DB {
			return db.Order("storyboards.storyboard_number ASC")
		}).
		Preload("Episodes.Storyboards.Background").         // Load scene/background linked by scene_id
		Preload("Episodes.Storyboards.Props").              // Load props associated with storyboards
		Preload("Episodes.Storyboards.Characters").         // Load characters associated with storyboards
		Preload("Episodes.Storyboards.Characters.Outfits"). // Load outfits for storyboard characters
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
	if req.NarrativeIdea != nil {
		updates["narrative_idea"] = strings.TrimSpace(*req.NarrativeIdea)
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

	if merged, mergeErr := mergeDuplicateCharactersByIdentity(s.db, s.log, dramaIDUint); mergeErr != nil {
		s.log.Warnw("Failed to pre-merge duplicate characters", "drama_id", dramaIDUint, "error", mergeErr)
	} else if merged > 0 {
		s.log.Infow("Pre-merged duplicate characters before save", "drama_id", dramaIDUint, "merged_count", merged)
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

	// Create normalized-name to character mapping
	existingCharMap := make(map[string]*models.Character)
	for i := range existingCharacters {
		key := normalizeCharacterName(existingCharacters[i].Name)
		if key != "" {
			existingCharMap[key] = &existingCharacters[i]
		}
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
					"name":              char.Name,
					"role":              char.Role,
					"description":       char.Description,
					"personality":       char.Personality,
					"appearance":        char.Appearance,
					"base_image_prompt": char.BaseImagePrompt,
					"image_url":         char.ImageURL,
				}
				if err := s.db.Model(&existing).Updates(updates).Error; err != nil {
					s.log.Errorw("Failed to update character", "error", err, "id", char.ID)
				}
				characterIDs = append(characterIDs, existing.ID)
				continue
			}
		}

		// 2. If no ID but same identity exists, reuse canonical character.
		reused := false
		incomingKey := normalizeCharacterName(char.Name)
		if incomingKey != "" {
			if existingChar, exists := existingCharMap[incomingKey]; exists {
				s.log.Infow("Character deduplicated by normalized name", "incoming_name", char.Name, "canonical_name", existingChar.Name, "character_id", existingChar.ID)
				characterIDs = append(characterIDs, existingChar.ID)
				reused = true
			}
		}
		if reused {
			continue
		}
		if incomingKey != "" {
			for i := range existingCharacters {
				existingChar := &existingCharacters[i]
				if isLikelySameCharacterName(existingChar.Name, char.Name) {
					s.log.Infow("Character deduplicated by fuzzy identity", "incoming_name", char.Name, "canonical_name", existingChar.Name, "character_id", existingChar.ID)
					characterIDs = append(characterIDs, existingChar.ID)
					if _, ok := existingCharMap[incomingKey]; !ok {
						existingCharMap[incomingKey] = existingChar
					}
					reused = true
					break
				}
			}
		}
		if reused {
			continue
		}

		// 3. Character does not exist, create new character
		character := models.Character{
			DramaID:         dramaIDUint,
			Name:            char.Name,
			Role:            char.Role,
			Description:     char.Description,
			Personality:     char.Personality,
			Appearance:      char.Appearance,
			BaseImagePrompt: char.BaseImagePrompt,
			ImageURL:        char.ImageURL,
		}

		if err := s.db.Create(&character).Error; err != nil {
			s.log.Errorw("Failed to create character", "error", err, "name", char.Name)
			continue
		}

		s.log.Infow("New character created", "character_id", character.ID, "name", char.Name)
		characterIDs = append(characterIDs, character.ID)
		existingCharacters = append(existingCharacters, character)
		if incomingKey := normalizeCharacterName(character.Name); incomingKey != "" {
			existingCharMap[incomingKey] = &character
		}
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
	if err := s.db.Unscoped().Where("drama_id = ?", dramaIDUint).Find(&eps).Error; err != nil {
		return err
	}
	for _, existing := range eps {
		// many2many join table name from GORM default — ignore if missing
		_ = s.db.Exec("DELETE FROM episode_characters WHERE episode_id = ?", existing.ID)
		if err := s.db.Unscoped().Where("episode_id = ?", existing.ID).Delete(&models.Scene{}).Error; err != nil {
			return fmt.Errorf("delete scenes episode %d: %w", existing.ID, err)
		}
		if err := s.db.Unscoped().Where("episode_id = ?", existing.ID).Delete(&models.Storyboard{}).Error; err != nil {
			return fmt.Errorf("delete storyboards episode %d: %w", existing.ID, err)
		}
		if err := s.db.Unscoped().Delete(&existing).Error; err != nil {
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
	if err := s.db.Unscoped().Where("drama_id = ?", dramaIDUint).Find(&existingEpisodes).Error; err != nil {
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
			narrativeTouch := ep.NarrativeNodeID != nil || ep.ParentNodeID != nil || len(ep.Choices) > 0 || len(ep.StateSnapshot) > 0 || ep.IsEntry
			if narrativeTouch {
				if ep.NarrativeNodeID != nil {
					updates["narrative_node_id"] = *ep.NarrativeNodeID
				}
				if ep.ParentNodeID != nil {
					updates["parent_node_id"] = *ep.ParentNodeID
				}
				if len(ep.Choices) > 0 {
					updates["choices"] = ep.Choices
				}
				if len(ep.StateSnapshot) > 0 {
					updates["state_snapshot"] = ep.StateSnapshot
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
				ParentNodeID:    ep.ParentNodeID,
				Choices:         ep.Choices,
				StateSnapshot:   ep.StateSnapshot,
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
			// Hard delete related scenes, storyboards, and other entities
			if err := s.db.Unscoped().Where("episode_id = ?", existing.ID).Delete(&models.Scene{}).Error; err != nil {
				s.log.Errorw("Failed to cascade delete scenes", "error", err, "episode", epNum)
				return fmt.Errorf("delete scenes for episode %d: %w", epNum, err)
			}
			if err := s.db.Unscoped().Where("episode_id = ?", existing.ID).Delete(&models.Storyboard{}).Error; err != nil {
				s.log.Errorw("Failed to cascade delete storyboards", "error", err, "episode", epNum)
				return fmt.Errorf("delete storyboards for episode %d: %w", epNum, err)
			}
			// (Note: StoryboardCharacters and Props might not have soft deletes or episode bindings depending on schema, so we stick to scenes and storyboards which are episode-bound and soft-deleted)

			if err := s.db.Unscoped().Delete(&existing).Error; err != nil {
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

// GenerateNarrativeEpisodes creates an async task to build branching episodes.
func (s *DramaService) GenerateNarrativeEpisodes(dramaID string, req NarrativeGenerateRequest) (string, error) {
	if req.AgentStep < 0 || req.AgentStep > 3 {
		return "", fmt.Errorf("invalid agent step: %d", req.AgentStep)
	}
	id, err := strconv.ParseUint(dramaID, 10, 32)
	if err != nil {
		return "", fmt.Errorf("invalid drama ID")
	}
	var drama models.Drama
	if err := s.db.Where("id = ?", uint(id)).First(&drama).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", errors.New("drama not found")
		}
		return "", err
	}

	taskType := "generate_narrative_full"
	if req.AgentStep > 0 {
		taskType = fmt.Sprintf("generate_narrative_step_%d", req.AgentStep)
	}
	task, err := s.taskService.CreateTask(taskType, dramaID)
	if err != nil {
		return "", fmt.Errorf("failed to create task: %w", err)
	}

	go s.processMultiAgentNarrative(task.ID, dramaID, req, drama)

	return task.ID, nil
}

func (s *DramaService) processMultiAgentNarrative(taskID string, dramaID string, req NarrativeGenerateRequest, drama models.Drama) {
	label := "full pipeline"
	if req.AgentStep > 0 {
		label = fmt.Sprintf("Agent %d", req.AgentStep)
	}
	s.taskService.UpdateTaskStatus(taskID, "processing", 5, fmt.Sprintf("Initializing %s...", label))

	if s.narrativePkg == nil || s.narrativeFallbackStub {
		s.taskService.UpdateTaskError(taskID, fmt.Errorf("multi-agent pipeline requires AI config"))
		return
	}
	if strings.TrimSpace(req.UserIdea) == "" && drama.NarrativeIdea != nil {
		req.UserIdea = *drama.NarrativeIdea
	}
	if strings.TrimSpace(req.UserIdea) != "" {
		if err := s.db.Model(&models.Drama{}).Where("id = ?", drama.ID).Updates(map[string]interface{}{
			"narrative_idea": strings.TrimSpace(req.UserIdea),
			"updated_at":     time.Now(),
		}).Error; err != nil {
			s.log.Warnw("Failed to save narrative idea before generation", "error", err, "drama_id", drama.ID)
		}
	}

	switch req.AgentStep {
	case 0:
		if err := s.runNarrativeAgent1(taskID, dramaID, req, drama, 5, 35); err != nil {
			s.taskService.UpdateTaskError(taskID, err)
			return
		}
		if err := s.runNarrativeAgent2(taskID, drama, 35, 70); err != nil {
			s.taskService.UpdateTaskError(taskID, err)
			return
		}
		if err := s.runNarrativeAgent3(taskID, drama, 70, 100); err != nil {
			s.taskService.UpdateTaskError(taskID, err)
			return
		}
		s.taskService.UpdateTaskStatus(taskID, "completed", 100, "Narrative full pipeline completed")
	case 1:
		if err := s.runNarrativeAgent1(taskID, dramaID, req, drama, 5, 100); err != nil {
			s.taskService.UpdateTaskError(taskID, err)
			return
		}
		s.taskService.UpdateTaskStatus(taskID, "completed", 100, "Agent 1 completed")
	case 2:
		if err := s.runNarrativeAgent2(taskID, drama, 5, 100); err != nil {
			s.taskService.UpdateTaskError(taskID, err)
			return
		}
		s.taskService.UpdateTaskStatus(taskID, "completed", 100, "Agent 2 completed")
	case 3:
		if err := s.runNarrativeAgent3(taskID, drama, 5, 100); err != nil {
			s.taskService.UpdateTaskError(taskID, err)
			return
		}
		s.taskService.UpdateTaskStatus(taskID, "completed", 100, "Agent 3 completed")
	default:
		s.taskService.UpdateTaskError(taskID, fmt.Errorf("invalid agent step: %d", req.AgentStep))
	}
}

func (s *DramaService) taskProgress(start, end, index, total int) int {
	if total <= 0 {
		return end
	}
	if index < 0 {
		index = 0
	}
	return start + ((end-start)*index)/total
}

func (s *DramaService) runNarrativeAgent1(taskID string, dramaID string, req NarrativeGenerateRequest, drama models.Drama, start, end int) error {
	s.taskService.UpdateTaskStatus(taskID, "processing", start, "Agent 1: Architecting global graph and characters...")
	out1, err := s.narrativePkg.RunAgent1Architect(req.UserIdea, drama)
	if err != nil {
		return fmt.Errorf("agent 1 failed: %w", err)
	}

	s.taskService.UpdateTaskStatus(taskID, "processing", s.taskProgress(start, end, 1, 3), "Agent 1: Saving characters...")
	if _, err := s.saveAgent1Characters(drama.ID, out1.Characters); err != nil {
		return fmt.Errorf("failed to save characters: %w", err)
	}

	parentByNode := make(map[string][]string)
	for _, node := range out1.GraphSkeleton {
		for _, choice := range node.Choices {
			parentByNode[choice.NextNarrativeNodeID] = append(parentByNode[choice.NextNarrativeNodeID], node.NarrativeNodeID)
		}
	}

	episodes := make([]models.Episode, 0, len(out1.GraphSkeleton))
	for i, node := range out1.GraphSkeleton {
		nodeCopy := node.NarrativeNodeID
		choiceRows := make([]episodeChoiceJSON, 0, len(node.Choices))
		for _, c := range node.Choices {
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
		var parentNodeID *string
		if parents := parentByNode[node.NarrativeNodeID]; len(parents) > 0 {
			parent := strings.Join(parents, ",")
			parentNodeID = &parent
		}
		summary := strings.TrimSpace(node.PlotSummary)
		stateJSON := datatypes.JSON("null")
		if summary != "" {
			stateMap := map[string]interface{}{
				"plot_summary": summary,
			}
			raw, _ := json.Marshal(stateMap)
			stateJSON = raw
		}
		episodes = append(episodes, models.Episode{
			EpisodeNum:      i + 1,
			Title:           node.Title,
			NarrativeNodeID: &nodeCopy,
			ParentNodeID:    parentNodeID,
			Choices:         choicesJSON,
			StateSnapshot:   stateJSON,
			IsEntry:         node.IsEntry || node.NarrativeNodeID == out1.StartNarrativeNodeID,
			Description:     &summary,
			Status:          "draft",
		})
	}

	s.taskService.UpdateTaskStatus(taskID, "processing", s.taskProgress(start, end, 2, 3), "Agent 1: Saving graph skeleton...")
	if err := s.saveEpisodes(dramaID, &SaveEpisodesRequest{Episodes: episodes}, true); err != nil {
		return fmt.Errorf("failed to save skeleton episodes: %w", err)
	}
	if err := s.resolveEpisodeChoiceNarrativeIDs(drama.ID); err != nil {
		return fmt.Errorf("failed to resolve graph choices: %w", err)
	}
	if err := s.db.Model(&models.Drama{}).Where("id = ?", drama.ID).Updates(map[string]interface{}{
		"total_episodes": len(episodes),
		"description":    out1.GlobalStoryline,
		"updated_at":     time.Now(),
	}).Error; err != nil {
		return err
	}
	s.taskService.UpdateTaskStatus(taskID, "processing", end, "Agent 1: Graph, characters, and base prompts saved")
	return nil
}

func (s *DramaService) runNarrativeAgent2(taskID string, drama models.Drama, start, end int) error {
	s.taskService.UpdateTaskStatus(taskID, "processing", start, "Agent 2: Loading graph and character data...")
	dbChars, agent1Chars, err := s.loadAgent1Characters(drama.ID)
	if err != nil {
		return err
	}
	dbEps, skeleton, err := s.loadGraphSkeleton(drama.ID)
	if err != nil {
		return err
	}
	order := s.graphExecutionOrder(skeleton)
	parentByNode := s.parentNodesByChoice(skeleton)
	episodeByNode := make(map[string]models.Episode, len(dbEps))
	for _, ep := range dbEps {
		if ep.NarrativeNodeID != nil {
			episodeByNode[*ep.NarrativeNodeID] = ep
		}
	}

	stateByNode := make(map[string]Agent2StateSnapshot)
	priorSummaries := make([]string, 0, len(order))
	for idx, node := range order {
		progress := s.taskProgress(start, end, idx, len(order)+1)
		s.taskService.UpdateTaskStatus(taskID, "processing", progress, fmt.Sprintf("Agent 2: Building %s...", node.NarrativeNodeID))
		incomingStates := make([]Agent2StateSnapshot, 0, len(parentByNode[node.NarrativeNodeID]))
		for _, parentID := range parentByNode[node.NarrativeNodeID] {
			if state, ok := stateByNode[parentID]; ok {
				incomingStates = append(incomingStates, state)
			}
		}
		epData, err := s.narrativePkg.RunAgent2BuilderEpisode(drama.Title, agent1Chars, skeleton, node, incomingStates, priorSummaries)
		if err != nil {
			return fmt.Errorf("agent 2 failed for %s: %w", node.NarrativeNodeID, err)
		}
		savedEp, ok := episodeByNode[epData.NarrativeNodeID]
		if !ok {
			return fmt.Errorf("agent 2 returned unknown node %s", epData.NarrativeNodeID)
		}
		if err := s.saveAgent2EpisodeData(drama.ID, savedEp, dbChars, *epData); err != nil {
			return fmt.Errorf("failed to save agent 2 data for %s: %w", epData.NarrativeNodeID, err)
		}
		stateByNode[epData.NarrativeNodeID] = epData.StateSnapshotT
		priorSummaries = append(priorSummaries, fmt.Sprintf("%s: %s", epData.NarrativeNodeID, strings.Join(epData.MicroBeats, " / ")))
		
		// Giãn cách nhẹ để tránh quá tải API rate limit
		time.Sleep(300 * time.Millisecond)
	}
	s.taskService.UpdateTaskStatus(taskID, "processing", end, "Agent 2: Beats, states, outfits, and scenes saved")
	return nil
}

func (s *DramaService) runNarrativeAgent3(taskID string, drama models.Drama, start, end int) error {
	s.taskService.UpdateTaskStatus(taskID, "processing", start, "Agent 3: Loading Agent 2 data...")
	_, agent1Chars, err := s.loadAgent1Characters(drama.ID)
	if err != nil {
		return err
	}
	dbEps, skeleton, err := s.loadGraphSkeleton(drama.ID)
	if err != nil {
		return err
	}
	order := s.graphExecutionOrder(skeleton)
	episodeByNode := make(map[string]models.Episode, len(dbEps))
	for _, ep := range dbEps {
		if ep.NarrativeNodeID != nil {
			episodeByNode[*ep.NarrativeNodeID] = ep
		}
	}
	for idx, node := range order {
		progress := s.taskProgress(start, end, idx, len(order)+1)
		s.taskService.UpdateTaskStatus(taskID, "processing", progress, fmt.Sprintf("Agent 3: Writing %s...", node.NarrativeNodeID))
		ep, ok := episodeByNode[node.NarrativeNodeID]
		if !ok {
			return fmt.Errorf("agent 3 missing DB episode for %s", node.NarrativeNodeID)
		}
		epData := s.agent2DataFromEpisode(ep)
		scriptData, err := s.narrativePkg.RunAgent3DesignerEpisode(drama.Title, agent1Chars, node, epData)
		if err != nil {
			return fmt.Errorf("agent 3 failed for %s: %w", node.NarrativeNodeID, err)
		}
		if err := s.db.Model(&models.Episode{}).
			Where("drama_id = ? AND narrative_node_id = ?", drama.ID, scriptData.NarrativeNodeID).
			Updates(map[string]interface{}{"script_content": scriptData.ScriptContent, "status": "draft"}).Error; err != nil {
			return err
		}
		
		// Giãn cách nhẹ để tránh quá tải API rate limit
		time.Sleep(300 * time.Millisecond)
	}
	s.taskService.UpdateTaskStatus(taskID, "processing", end, "Agent 3: Markdown scripts saved")
	return nil
}

func (s *DramaService) saveAgent1Characters(dramaIDUint uint, chars []Agent1Character) ([]models.Character, error) {
	// First, let's delete old characters and their outfits to avoid duplicates and orphan data
	var oldChars []models.Character
	if err := s.db.Unscoped().Where("drama_id = ?", dramaIDUint).Find(&oldChars).Error; err == nil {
		for _, c := range oldChars {
			_ = s.db.Unscoped().Where("character_id = ?", c.ID).Delete(&models.CharacterOutfit{})
			_ = s.db.Unscoped().Delete(&c)
		}
	}

	created := make([]models.Character, 0, len(chars))
	for _, c := range chars {
		roleCopy := c.Role
		descCopy := c.Description
		persCopy := c.Personality
		appCopy := c.Appearance
		baseImgCopy := c.BaseImagePrompt

		character := models.Character{
			DramaID:         dramaIDUint,
			Name:            c.Name,
			Role:            &roleCopy,
			Description:     &descCopy,
			Personality:     &persCopy,
			Appearance:      &appCopy,
			BaseImagePrompt: &baseImgCopy,
		}
		if err := s.db.Create(&character).Error; err != nil {
			s.log.Errorw("Failed to create agent 1 character", "error", err, "name", c.Name)
			return created, err
		}
		created = append(created, character)
	}
	return created, nil
}

func (s *DramaService) triggerBaseCharacterImages(drama models.Drama, characters []models.Character) {
	if s.imageGeneration == nil {
		s.log.Warnw("Skipping base character images; image generation service is not wired", "drama_id", drama.ID)
		return
	}
	for _, character := range characters {
		if character.BaseImagePrompt == nil || strings.TrimSpace(*character.BaseImagePrompt) == "" {
			continue
		}
		characterID := character.ID
		style := drama.Style
		if _, err := s.imageGeneration.GenerateImage(&GenerateImageRequest{
			DramaID:     fmt.Sprintf("%d", drama.ID),
			CharacterID: &characterID,
			ImageType:   string(models.ImageTypeCharacter),
			Prompt:      strings.TrimSpace(*character.BaseImagePrompt),
			Provider:    "openai",
			Style:       &style,
		}); err != nil {
			s.log.Warnw("Failed to queue base character image", "drama_id", drama.ID, "character_id", character.ID, "error", err)
		}
	}
}

func (s *DramaService) loadAgent1Characters(dramaID uint) ([]models.Character, []Agent1Character, error) {
	var dbChars []models.Character
	if err := s.db.Where("drama_id = ?", dramaID).Order("name ASC").Find(&dbChars).Error; err != nil {
		return nil, nil, err
	}
	if len(dbChars) == 0 {
		return nil, nil, fmt.Errorf("no characters found. Please run Agent 1 first")
	}
	agent1Chars := make([]Agent1Character, 0, len(dbChars))
	for _, c := range dbChars {
		role, desc, pers, app, base := "", "", "", "", ""
		if c.Role != nil {
			role = *c.Role
		}
		if c.Description != nil {
			desc = *c.Description
		}
		if c.Personality != nil {
			pers = *c.Personality
		}
		if c.Appearance != nil {
			app = *c.Appearance
		}
		if c.BaseImagePrompt != nil {
			base = *c.BaseImagePrompt
		}
		agent1Chars = append(agent1Chars, Agent1Character{
			Name:            c.Name,
			Role:            role,
			Description:     desc,
			Personality:     pers,
			Appearance:      app,
			BaseImagePrompt: base,
		})
	}
	return dbChars, agent1Chars, nil
}

func (s *DramaService) loadGraphSkeleton(dramaID uint) ([]models.Episode, []Agent1Node, error) {
	var dbEps []models.Episode
	if err := s.db.Where("drama_id = ?", dramaID).Order("episode_number ASC").Find(&dbEps).Error; err != nil {
		return nil, nil, err
	}
	if len(dbEps) == 0 {
		return nil, nil, fmt.Errorf("no episodes found. Please run Agent 1 first")
	}
	skeleton := make([]Agent1Node, 0, len(dbEps))
	for _, ep := range dbEps {
		if ep.NarrativeNodeID == nil || *ep.NarrativeNodeID == "" {
			return nil, nil, fmt.Errorf("episode %d has no narrative node id. Please run Agent 1 first", ep.EpisodeNum)
		}
		var choices []NarrativeChoiceDraft
		if len(ep.Choices) > 0 {
			var choiceRows []episodeChoiceJSON
			if err := json.Unmarshal(ep.Choices, &choiceRows); err != nil {
				return nil, nil, err
			}
			for _, choice := range choiceRows {
				choices = append(choices, NarrativeChoiceDraft{
					Label:               choice.Label,
					NextNarrativeNodeID: choice.NextNarrativeNodeID,
				})
			}
		}
		plotSummary := ""
		if ep.Description != nil {
			plotSummary = *ep.Description
		}
		skeleton = append(skeleton, Agent1Node{
			NarrativeNodeID: *ep.NarrativeNodeID,
			Title:           ep.Title,
			PlotSummary:     plotSummary,
			IsEntry:         ep.IsEntry,
			Choices:         choices,
		})
	}
	return dbEps, skeleton, nil
}

func (s *DramaService) parentNodesByChoice(nodes []Agent1Node) map[string][]string {
	parentByNode := make(map[string][]string)
	for _, node := range nodes {
		for _, choice := range node.Choices {
			parentByNode[choice.NextNarrativeNodeID] = append(parentByNode[choice.NextNarrativeNodeID], node.NarrativeNodeID)
		}
	}
	return parentByNode
}

func (s *DramaService) graphExecutionOrder(nodes []Agent1Node) []Agent1Node {
	byID := make(map[string]Agent1Node, len(nodes))
	for _, node := range nodes {
		byID[node.NarrativeNodeID] = node
	}
	startID := ""
	for _, node := range nodes {
		if node.IsEntry {
			startID = node.NarrativeNodeID
			break
		}
	}
	if startID == "" && len(nodes) > 0 {
		startID = nodes[0].NarrativeNodeID
	}

	seen := make(map[string]bool, len(nodes))
	order := make([]Agent1Node, 0, len(nodes))
	queue := []string{startID}
	for len(queue) > 0 {
		currentID := queue[0]
		queue = queue[1:]
		if seen[currentID] {
			continue
		}
		node, ok := byID[currentID]
		if !ok {
			continue
		}
		seen[currentID] = true
		order = append(order, node)
		for _, choice := range node.Choices {
			if !seen[choice.NextNarrativeNodeID] {
				queue = append(queue, choice.NextNarrativeNodeID)
			}
		}
	}
	for _, node := range nodes {
		if !seen[node.NarrativeNodeID] {
			order = append(order, node)
		}
	}
	return order
}

func (s *DramaService) saveAgent2EpisodeData(dramaID uint, episode models.Episode, dbChars []models.Character, epData Agent2EpisodeData) error {
	charMap := make(map[string]models.Character, len(dbChars))
	for _, c := range dbChars {
		charMap[c.Name] = c
	}

	var originalPlotSummary string
	if episode.Description != nil {
		originalPlotSummary = *episode.Description
	}
	if len(episode.StateSnapshot) > 0 {
		var existingMap map[string]interface{}
		_ = json.Unmarshal(episode.StateSnapshot, &existingMap)
		if existingMap != nil {
			if ps, ok := existingMap["plot_summary"].(string); ok {
				originalPlotSummary = ps
			}
		}
	}

	stateMap := map[string]interface{}{
		"timeline":            epData.StateSnapshotT.Timeline,
		"character_statuses":  epData.StateSnapshotT.CharacterStatuses,
		"key_items_locations": epData.StateSnapshotT.KeyItemsLocations,
	}
	if originalPlotSummary != "" {
		stateMap["plot_summary"] = originalPlotSummary
	}

	currentStateJSONBytes, err := json.Marshal(stateMap)
	if err != nil {
		return err
	}
	desc := strings.Join(epData.MicroBeats, "\n- ")
	if len(epData.MicroBeats) > 0 {
		desc = "- " + desc
	}
	if err := s.db.Model(&episode).Updates(map[string]interface{}{
		"description":    desc,
		"state_snapshot": datatypes.JSON(currentStateJSONBytes),
	}).Error; err != nil {
		return err
	}

	if err := s.db.Unscoped().Where("episode_id = ?", episode.ID).Delete(&models.Scene{}).Error; err != nil {
		return err
	}

	var epCharacters []models.Character
	charSet := make(map[uint]bool)
	nodeID := ""
	if episode.NarrativeNodeID != nil {
		nodeID = *episode.NarrativeNodeID
	}
	for _, outfit := range epData.EpisodeOutfits {
		dbChar, exists := charMap[outfit.CharacterName]
		if !exists {
			for _, candidate := range dbChars {
				if isLikelySameCharacterName(candidate.Name, outfit.CharacterName) {
					dbChar = candidate
					exists = true
					break
				}
			}
		}
		if !exists {
			s.log.Warnw("Agent 2 outfit references unknown character", "character_name", outfit.CharacterName, "episode_id", episode.ID)
			continue
		}
		if !charSet[dbChar.ID] {
			charSet[dbChar.ID] = true
			epCharacters = append(epCharacters, models.Character{ID: dbChar.ID})
		}
		if _, err := s.findOrCreateSemanticOutfit(dbChar.ID, outfit, nodeID); err != nil {
			return err
		}
	}
	if len(epCharacters) > 0 {
		if err := s.db.Model(&episode).Association("Characters").Replace(epCharacters); err != nil {
			return err
		}
	}

	defaultTime := "Unspecified"
	for _, scene := range epData.EpisodeScenes {
		location := strings.TrimSpace(scene.LocationName)
		if location == "" {
			location = "Unspecified location"
		}
		prompt := strings.TrimSpace(scene.ScenePrompt)
		if prompt == "" {
			prompt = location
		}
		prompt = ensureBackgroundOnlyPrompt(prompt)
		sceneRecord := models.Scene{
			DramaID:   dramaID,
			EpisodeID: &episode.ID,
			Location:  location,
			Time:      defaultTime,
			Prompt:    prompt,
		}
		if err := s.db.Create(&sceneRecord).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *DramaService) findOrCreateSemanticOutfit(characterID uint, outfit Agent2Outfit, nodeID string) (*models.CharacterOutfit, error) {
	name := cleanSemanticOutfitName(outfit.OutfitName, outfit.OutfitPrompt)
	prompt := strings.TrimSpace(outfit.OutfitPrompt)
	if prompt == "" {
		prompt = name
	}

	var existing []models.CharacterOutfit
	if err := s.db.Where("character_id = ?", characterID).Find(&existing).Error; err != nil {
		return nil, err
	}
	nameKey := normalizeComparableText(name)
	promptKey := normalizeComparableText(prompt)
	for i := range existing {
		existingNameKey := normalizeComparableText(existing[i].Name)
		existingPromptKey := normalizeComparableText(existing[i].Prompt)
		if existingNameKey == nameKey ||
			(existingPromptKey != "" && promptKey != "" && (existingPromptKey == promptKey || strings.Contains(existingPromptKey, promptKey) || strings.Contains(promptKey, existingPromptKey))) {
			
			// Cập nhật trường Appearances nếu chưa tồn tại nodeID
			if nodeID != "" {
				hasNode := false
				parts := strings.Split(existing[i].Appearances, ",")
				for _, p := range parts {
					if strings.TrimSpace(p) == nodeID {
						hasNode = true
						break
					}
				}
				if !hasNode {
					newApps := nodeID
					if strings.TrimSpace(existing[i].Appearances) != "" {
						newApps = existing[i].Appearances + "," + nodeID
					}
					if err := s.db.Model(&existing[i]).Update("appearances", newApps).Error; err != nil {
						return nil, err
					}
					existing[i].Appearances = newApps
				}
			}

			if isEpisodeOutfitName(existing[i].Name) && name != "" && name != existing[i].Name {
				if err := s.db.Model(&existing[i]).Update("name", name).Error; err != nil {
					return nil, err
				}
				existing[i].Name = name
			}
			return &existing[i], nil
		}
	}

	record := models.CharacterOutfit{
		CharacterID: characterID,
		Name:        name,
		Prompt:      prompt,
		Appearances: nodeID,
	}
	if err := s.db.Create(&record).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

func cleanSemanticOutfitName(name, prompt string) string {
	cleaned := strings.TrimSpace(name)
	if cleaned == "" || isEpisodeOutfitName(cleaned) {
		cleaned = deriveOutfitNameFromPrompt(prompt)
	}
	if cleaned == "" {
		return "Story Outfit"
	}
	return cleaned
}

func deriveOutfitNameFromPrompt(prompt string) string {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return ""
	}
	parts := strings.FieldsFunc(prompt, func(r rune) bool {
		return r == ',' || r == ';' || r == '.' || r == '\n'
	})
	if len(parts) == 0 {
		return ""
	}
	name := strings.TrimSpace(parts[0])
	words := strings.Fields(name)
	if len(words) > 5 {
		words = words[:5]
	}
	return titleCaseWords(strings.Join(words, " "))
}

func isEpisodeOutfitName(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	return strings.Contains(lower, "ep ") || strings.Contains(lower, "episode")
}

func titleCaseWords(s string) string {
	words := strings.Fields(strings.ToLower(s))
	for i, word := range words {
		if len(word) == 0 {
			continue
		}
		words[i] = strings.ToUpper(word[:1]) + word[1:]
	}
	return strings.Join(words, " ")
}

func normalizeComparableText(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	lastSpace := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastSpace = false
			continue
		}
		if !lastSpace {
			b.WriteRune(' ')
			lastSpace = true
		}
	}
	return strings.TrimSpace(b.String())
}

func ensureBackgroundOnlyPrompt(prompt string) string {
	cleaned := strings.TrimSpace(prompt)
	if cleaned == "" {
		cleaned = "an empty cinematic environment"
	}
	lower := strings.ToLower(cleaned)
	if !strings.Contains(lower, "pure background") && !strings.Contains(lower, "background only") {
		cleaned = fmt.Sprintf("A cinematic pure background scene depicting %s. The scene shows environment details, architecture, objects, and lighting only.", cleaned)
	}
	missing := []string{}
	for _, rule := range []string{"no people", "no characters", "no faces", "no bodies", "no hands", "no silhouettes", "no crowds"} {
		if !strings.Contains(lower, rule) {
			missing = append(missing, rule)
		}
	}
	if len(missing) > 0 {
		cleaned += ", " + strings.Join(missing, ", ")
	}
	return cleaned
}

func (s *DramaService) agent2DataFromEpisode(ep models.Episode) Agent2EpisodeData {
	var microBeats []string
	if ep.Description != nil && strings.TrimSpace(*ep.Description) != "" {
		microBeats = strings.Split(strings.TrimPrefix(strings.TrimSpace(*ep.Description), "- "), "\n- ")
	}
	var stateSnapshot Agent2StateSnapshot
	if len(ep.StateSnapshot) > 0 {
		_ = json.Unmarshal(ep.StateSnapshot, &stateSnapshot)
	}
	nodeID := ""
	if ep.NarrativeNodeID != nil {
		nodeID = *ep.NarrativeNodeID
	}
	return Agent2EpisodeData{
		NarrativeNodeID: nodeID,
		MicroBeats:      microBeats,
		StateSnapshotT:  stateSnapshot,
	}
}

// GenerateNarrativeEpisodesSync builds branching episodes via Text AI + embedded prompts (NarrativePackageService), optional template DAG if fallback_stub.
// Replaces the drama's episode list with the generated package (same semantics as SaveEpisodes).
func (s *DramaService) GenerateNarrativeEpisodesSync(dramaID string, userIdea string) error {
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
