package services

import (
	"errors"
	"fmt"
	"time"

	models "github.com/drama-generator/backend/domain/models"
	"github.com/drama-generator/backend/pkg/ai"
	"github.com/drama-generator/backend/pkg/config"
	"github.com/drama-generator/backend/pkg/logger"
	"github.com/drama-generator/backend/pkg/utils"
	"gorm.io/gorm"
)

type CharacterLibraryService struct {
	db          *gorm.DB
	log         *logger.Logger
	config      *config.Config
	aiService   *AIService
	taskService *TaskService
	promptI18n  *PromptI18n
}

func NewCharacterLibraryService(db *gorm.DB, log *logger.Logger, cfg *config.Config) *CharacterLibraryService {
	return &CharacterLibraryService{
		db:          db,
		log:         log,
		config:      cfg,
		aiService:   NewAIService(db, log),
		taskService: NewTaskService(db, log),
		promptI18n:  NewPromptI18n(cfg),
	}
}

type CreateLibraryItemRequest struct {
	Name        string  `json:"name" binding:"required,min=1,max=100"`
	Category    *string `json:"category"`
	ImageURL    string  `json:"image_url" binding:"required"`
	LocalPath   *string `json:"local_path"`
	Description *string `json:"description"`
	Tags        *string `json:"tags"`
	SourceType  string  `json:"source_type"`
}

type CharacterLibraryQuery struct {
	Page       int    `form:"page,default=1"`
	PageSize   int    `form:"page_size,default=20"`
	Category   string `form:"category"`
	SourceType string `form:"source_type"`
	Keyword    string `form:"keyword"`
}

// ListLibraryItems retrieves the user's character library list
func (s *CharacterLibraryService) ListLibraryItems(query *CharacterLibraryQuery) ([]models.CharacterLibrary, int64, error) {
	var items []models.CharacterLibrary
	var total int64

	db := s.db.Model(&models.CharacterLibrary{})

	// Filter conditions
	if query.Category != "" {
		db = db.Where("category = ?", query.Category)
	}

	if query.SourceType != "" {
		db = db.Where("source_type = ?", query.SourceType)
	}

	if query.Keyword != "" {
		db = db.Where("name LIKE ? OR description LIKE ?", "%"+query.Keyword+"%", "%"+query.Keyword+"%")
	}

	// Get total count
	if err := db.Count(&total).Error; err != nil {
		s.log.Errorw("Failed to count character library", "error", err)
		return nil, 0, err
	}

	// Paginated query
	offset := (query.Page - 1) * query.PageSize
	err := db.Order("created_at DESC").
		Offset(offset).
		Limit(query.PageSize).
		Find(&items).Error

	if err != nil {
		s.log.Errorw("Failed to list character library", "error", err)
		return nil, 0, err
	}

	return items, total, nil
}

// CreateLibraryItem adds an item to the character library
func (s *CharacterLibraryService) CreateLibraryItem(req *CreateLibraryItemRequest) (*models.CharacterLibrary, error) {
	sourceType := req.SourceType
	if sourceType == "" {
		sourceType = "generated"
	}

	item := &models.CharacterLibrary{
		Name:        req.Name,
		Category:    req.Category,
		ImageURL:    req.ImageURL,
		LocalPath:   req.LocalPath,
		Description: req.Description,
		Tags:        req.Tags,
		SourceType:  sourceType,
	}

	if err := s.db.Create(item).Error; err != nil {
		s.log.Errorw("Failed to create library item", "error", err)
		return nil, err
	}

	s.log.Infow("Library item created", "item_id", item.ID)
	return item, nil
}

// GetLibraryItem retrieves a character library item
func (s *CharacterLibraryService) GetLibraryItem(itemID string) (*models.CharacterLibrary, error) {
	var item models.CharacterLibrary
	err := s.db.Where("id = ? ", itemID).First(&item).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("library item not found")
		}
		s.log.Errorw("Failed to get library item", "error", err)
		return nil, err
	}

	return &item, nil
}

// DeleteLibraryItem deletes a character library item
func (s *CharacterLibraryService) DeleteLibraryItem(itemID string) error {
	result := s.db.Where("id = ? ", itemID).Delete(&models.CharacterLibrary{})

	if result.Error != nil {
		s.log.Errorw("Failed to delete library item", "error", result.Error)
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("library item not found")
	}

	s.log.Infow("Library item deleted", "item_id", itemID)
	return nil
}

// ApplyLibraryItemToCharacter applies a library item's image to a character
func (s *CharacterLibraryService) ApplyLibraryItemToCharacter(characterID string, libraryItemID string) error {
	// Verify the library item exists and belongs to the user
	var libraryItem models.CharacterLibrary
	if err := s.db.Where("id = ? ", libraryItemID).First(&libraryItem).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("library item not found")
		}
		return err
	}

	// Find the character
	var character models.Character
	if err := s.db.Where("id = ?", characterID).First(&character).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("character not found")
		}
		return err
	}

	// Query Drama to verify permissions
	var drama models.Drama
	if err := s.db.Where("id = ? ", character.DramaID).First(&drama).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("unauthorized")
		}
		return err
	}

	// Update the character's local_path and image_url
	updates := map[string]interface{}{}
	if libraryItem.LocalPath != nil && *libraryItem.LocalPath != "" {
		updates["local_path"] = libraryItem.LocalPath
	}
	if libraryItem.ImageURL != "" {
		updates["image_url"] = libraryItem.ImageURL
	}
	if len(updates) > 0 {
		if err := s.db.Model(&character).Updates(updates).Error; err != nil {
			s.log.Errorw("Failed to update character image", "error", err)
			return err
		}
	}

	s.log.Infow("Library item applied to character", "character_id", characterID, "library_item_id", libraryItemID)
	return nil
}

// UploadCharacterImage uploads a character image
func (s *CharacterLibraryService) UploadCharacterImage(characterID string, imageURL string) error {
	// Find the character
	var character models.Character
	if err := s.db.Where("id = ?", characterID).First(&character).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("character not found")
		}
		return err
	}

	// Query Drama to verify permissions
	var drama models.Drama
	if err := s.db.Where("id = ? ", character.DramaID).First(&drama).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("unauthorized")
		}
		return err
	}

	// Update image URL
	if err := s.db.Model(&character).Update("image_url", imageURL).Error; err != nil {
		s.log.Errorw("Failed to update character image", "error", err)
		return err
	}

	s.log.Infow("Character image uploaded", "character_id", characterID)
	return nil
}

// AddCharacterToLibrary adds a character to the character library
func (s *CharacterLibraryService) AddCharacterToLibrary(characterID string, category *string) (*models.CharacterLibrary, error) {
	// Find the character
	var character models.Character
	if err := s.db.Where("id = ?", characterID).First(&character).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("character not found")
		}
		return nil, err
	}

	// Query Drama to verify permissions
	var drama models.Drama
	if err := s.db.Where("id = ? ", character.DramaID).First(&drama).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("unauthorized")
		}
		return nil, err
	}

	// Check if the character has an image
	if character.ImageURL == nil || *character.ImageURL == "" {
		return nil, fmt.Errorf("character does not have an image yet")
	}

	// Create character library item
	charLibrary := &models.CharacterLibrary{
		Name:        character.Name,
		ImageURL:    *character.ImageURL,
		LocalPath:   character.LocalPath,
		Description: character.Description,
		SourceType:  "character",
	}

	if err := s.db.Create(charLibrary).Error; err != nil {
		s.log.Errorw("Failed to add character to library", "error", err)
		return nil, err
	}

	s.log.Infow("Character added to library", "character_id", characterID, "library_item_id", charLibrary.ID)
	return charLibrary, nil
}

// DeleteCharacter deletes a single character
func (s *CharacterLibraryService) DeleteCharacter(characterID uint) error {
	// Find the character
	var character models.Character
	if err := s.db.Where("id = ?", characterID).First(&character).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("character not found")
		}
		return err
	}

	// Verify permissions: check if the drama the character belongs to is owned by the current user
	var drama models.Drama
	if err := s.db.Where("id = ? ", character.DramaID).First(&drama).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("unauthorized")
		}
		return err
	}

	// Delete the character
	if err := s.db.Delete(&character).Error; err != nil {
		s.log.Errorw("Failed to delete character", "error", err, "id", characterID)
		return err
	}

	s.log.Infow("Character deleted", "id", characterID)
	return nil
}

// GenerateCharacterImage generates a character image using AI
func (s *CharacterLibraryService) GenerateCharacterImage(characterID string, imageService *ImageGenerationService, modelName string, style string) (*models.ImageGeneration, error) {
	// Find the character
	var character models.Character
	if err := s.db.Where("id = ?", characterID).First(&character).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("character not found")
		}
		return nil, err
	}

	// Query Drama to verify permissions
	var drama models.Drama
	if err := s.db.Where("id = ? ", character.DramaID).First(&drama).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("unauthorized")
		}
		return nil, err
	}

	// Build generation prompt - use detailed appearance description, add clean background requirement
	prompt := ""

	// Prefer the appearance field as it contains the most detailed appearance description
	if character.Appearance != nil && *character.Appearance != "" {
		prompt = *character.Appearance
	} else if character.Description != nil && *character.Description != "" {
		prompt = *character.Description
	} else {
		prompt = character.Name
	}

	// Append style and aspect ratio so the image API uses the correct framing
	if drama.Style != "" && drama.Style != "realistic" {
		prompt += ", " + drama.Style
	}
	aspectRatio := drama.AspectRatio
	if aspectRatio == "" {
		aspectRatio = "16:9"
	}
	prompt += ", " + aspectRatio + " aspect ratio"

	// Call image generation service
	dramaIDStr := fmt.Sprintf("%d", character.DramaID)
	imageType := "character"
	req := &GenerateImageRequest{
		DramaID:     dramaIDStr,
		CharacterID: &character.ID,
		ImageType:   imageType,
		Prompt:      prompt,
		Provider:    "openai",
		Model:       modelName,
		Quality:     "standard",
	}

	imageGen, err := imageService.GenerateImage(req)
	if err != nil {
		s.log.Errorw("Failed to generate character image", "error", err)
		return nil, fmt.Errorf("image generation failed: %w", err)
	}

	// Async processing: listen in background for image generation completion, then update character image_url
	go s.waitAndUpdateCharacterImage(character.ID, imageGen.ID)

	// Return ImageGeneration object immediately so frontend can poll for status
	s.log.Infow("Character image generation started", "character_id", characterID, "image_gen_id", imageGen.ID)
	return imageGen, nil
}

// waitAndUpdateCharacterImage asynchronously waits in background for image generation to complete and updates character image_url
func (s *CharacterLibraryService) waitAndUpdateCharacterImage(characterID uint, imageGenID uint) {
	maxAttempts := 60
	pollInterval := 5 * time.Second

	for i := 0; i < maxAttempts; i++ {
		time.Sleep(pollInterval)

		// Query image generation status
		var imageGen models.ImageGeneration
		if err := s.db.First(&imageGen, imageGenID).Error; err != nil {
			s.log.Errorw("Failed to query image generation status", "error", err, "image_gen_id", imageGenID)
			continue
		}

		// Check if completed
		if imageGen.Status == models.ImageStatusCompleted && imageGen.ImageURL != nil && *imageGen.ImageURL != "" {
			// Update the character's image_url
			if err := s.db.Model(&models.Character{}).Where("id = ?", characterID).Update("image_url", *imageGen.ImageURL).Error; err != nil {
				s.log.Errorw("Failed to update character image_url", "error", err, "character_id", characterID)
				return
			}
			s.log.Infow("Character image updated successfully", "character_id", characterID, "image_url", *imageGen.ImageURL)
			return
		}

		// Check if failed
		if imageGen.Status == models.ImageStatusFailed {
			s.log.Errorw("Character image generation failed", "character_id", characterID, "image_gen_id", imageGenID, "error", imageGen.ErrorMsg)
			return
		}
	}

	s.log.Warnw("Character image generation timeout", "character_id", characterID, "image_gen_id", imageGenID)
}

type UpdateCharacterRequest struct {
	Name        *string `json:"name"`
	Role        *string `json:"role"`
	Appearance  *string `json:"appearance"`
	Personality *string `json:"personality"`
	Description *string `json:"description"`
	ImageURL    *string `json:"image_url"`
	LocalPath   *string `json:"local_path"`
}

// UpdateCharacter updates character information
func (s *CharacterLibraryService) UpdateCharacter(characterID string, req *UpdateCharacterRequest) error {
	// Find the character
	var character models.Character
	if err := s.db.Where("id = ?", characterID).First(&character).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("character not found")
		}
		return err
	}

	// Verify permissions: check if the drama the character belongs to is owned by the user
	var drama models.Drama
	if err := s.db.Where("id = ? ", character.DramaID).First(&drama).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("unauthorized")
		}
		return err
	}

	// Build update data
	updates := make(map[string]interface{})

	if req.Name != nil && *req.Name != "" {
		updates["name"] = *req.Name
	}
	if req.Role != nil {
		updates["role"] = *req.Role
	}
	if req.Appearance != nil {
		updates["appearance"] = *req.Appearance
	}
	if req.Personality != nil {
		updates["personality"] = *req.Personality
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.ImageURL != nil {
		updates["image_url"] = *req.ImageURL
	}
	if req.LocalPath != nil {
		updates["local_path"] = *req.LocalPath
	}

	if len(updates) == 0 {
		return errors.New("no fields to update")
	}

	// Update character information
	if err := s.db.Model(&character).Updates(updates).Error; err != nil {
		s.log.Errorw("Failed to update character", "error", err, "character_id", characterID)
		return err
	}

	s.log.Infow("Character updated", "character_id", characterID, "updates", updates)
	return nil
}

// BatchGenerateCharacterImages batch generates character images (executed concurrently)
func (s *CharacterLibraryService) BatchGenerateCharacterImages(characterIDs []string, imageService *ImageGenerationService, modelName string) {
	s.log.Infow("Starting batch character image generation",
		"count", len(characterIDs),
		"model", modelName)

	// Use goroutines to concurrently generate all character images
	for _, characterID := range characterIDs {
		// Start a separate goroutine for each character
		go func(charID string) {
			imageGen, err := s.GenerateCharacterImage(charID, imageService, modelName, "") // Batch generation does not support custom styles yet, using default
			if err != nil {
				s.log.Errorw("Failed to generate character image in batch",
					"character_id", charID,
					"error", err)
				return
			}

			s.log.Infow("Character image generated in batch",
				"character_id", charID,
				"image_gen_id", imageGen.ID)
		}(characterID)
	}

	s.log.Infow("Batch character image generation tasks submitted",
		"total", len(characterIDs))
}

// ExtractCharactersFromScript extracts characters from an episode script
func (s *CharacterLibraryService) ExtractCharactersFromScript(episodeID uint) (string, error) {
	var episode models.Episode
	if err := s.db.First(&episode, episodeID).Error; err != nil {
		return "", fmt.Errorf("episode not found")
	}

	if episode.ScriptContent == nil || *episode.ScriptContent == "" {
		return "", fmt.Errorf("script content is empty")
	}

	task, err := s.taskService.CreateTask("character_extraction", fmt.Sprintf("%d", episode.DramaID))
	if err != nil {
		return "", fmt.Errorf("failed to create task: %w", err)
	}

	go s.processCharacterExtraction(task.ID, episode)

	return task.ID, nil
}

func (s *CharacterLibraryService) processCharacterExtraction(taskID string, episode models.Episode) {
	s.taskService.UpdateTaskStatus(taskID, "processing", 0, "Analyzing script...")

	script := ""
	if episode.ScriptContent != nil {
		script = *episode.ScriptContent
	}

	// Get drama's style information
	var drama models.Drama
	if err := s.db.First(&drama, episode.DramaID).Error; err != nil {
		s.log.Warnw("Failed to load drama", "error", err, "drama_id", episode.DramaID)
	}

	prompt := s.promptI18n.GetCharacterExtractionPrompt(drama.Style, drama.AspectRatio)
	userPrompt := fmt.Sprintf("Script Content:\n%s", script)

	response, err := s.aiService.GenerateText(userPrompt, prompt, ai.WithMaxTokens(3000))
	if err != nil {
		s.taskService.UpdateTaskError(taskID, err)
		return
	}

	s.taskService.UpdateTaskStatus(taskID, "processing", 50, "Organizing character data...")

	var extractedCharacters []struct {
		Name        string `json:"name"`
		Role        string `json:"role"`
		Appearance  string `json:"appearance"`
		Personality string `json:"personality"`
		Description string `json:"description"`
	}

	if err := utils.SafeParseAIJSON(response, &extractedCharacters); err != nil {
		s.log.Errorw("Failed to parse AI response for characters", "error", err, "response", response)
		s.taskService.UpdateTaskError(taskID, fmt.Errorf("failed to parse AI response"))
		return
	}

	var savedCharacters []models.Character
	for _, charData := range extractedCharacters {
		// Check if a character with the same name already exists
		var existingCharacter models.Character
		err := s.db.Where("drama_id = ? AND name = ?", episode.DramaID, charData.Name).First(&existingCharacter).Error

		if err == nil {
			// If exists, only associate without updating (could update, but skipping for now)
			if err := s.db.Model(&episode).Association("Characters").Append(&existingCharacter); err != nil {
				s.log.Warnw("Failed to associate existing character", "error", err)
			}
			savedCharacters = append(savedCharacters, existingCharacter)
		} else {
			// Create new character
			newCharacter := models.Character{
				DramaID:     episode.DramaID,
				Name:        charData.Name,
				Role:        &charData.Role,
				Appearance:  &charData.Appearance,
				Personality: &charData.Personality,
				Description: &charData.Description,
			}
			if err := s.db.Create(&newCharacter).Error; err != nil {
				s.log.Errorw("Failed to create extracted character", "error", err)
				continue
			}

			// Associate with episode
			if err := s.db.Model(&episode).Association("Characters").Append(&newCharacter); err != nil {
				s.log.Warnw("Failed to associate new character", "error", err)
			}
			savedCharacters = append(savedCharacters, newCharacter)
		}
	}

	s.taskService.UpdateTaskResult(taskID, map[string]interface{}{
		"characters": savedCharacters,
		"count":      len(savedCharacters),
	})
}
