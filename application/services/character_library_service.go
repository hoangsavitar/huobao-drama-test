package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"strings"
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

	// Build generation prompt. Agent 1 stores the canonical base prompt here; image
	// generation stays manual and should use that prompt when available.
	prompt := ""

	if character.BaseImagePrompt != nil && strings.TrimSpace(*character.BaseImagePrompt) != "" {
		prompt = strings.TrimSpace(*character.BaseImagePrompt)
	} else if character.Appearance != nil && *character.Appearance != "" {
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
	Name            *string `json:"name"`
	Role            *string `json:"role"`
	Appearance      *string `json:"appearance"`
	Personality     *string `json:"personality"`
	Description     *string `json:"description"`
	BaseImagePrompt *string `json:"base_image_prompt"`
	ImageURL        *string `json:"image_url"`
	LocalPath       *string `json:"local_path"`
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
	if req.BaseImagePrompt != nil {
		updates["base_image_prompt"] = *req.BaseImagePrompt
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

type CreateOutfitRequest struct {
	Name   string `json:"name" binding:"required"`
	Prompt string `json:"prompt"`
}

type UpdateOutfitRequest struct {
	Name      *string `json:"name"`
	Prompt    *string `json:"prompt"`
	ImageURL  *string `json:"image_url"`
	LocalPath *string `json:"local_path"`
}

func (s *CharacterLibraryService) CreateCharacterOutfit(characterID string, req *CreateOutfitRequest) (*models.CharacterOutfit, error) {
	var character models.Character
	if err := s.db.Where("id = ?", characterID).First(&character).Error; err != nil {
		return nil, errors.New("character not found")
	}

	outfit := &models.CharacterOutfit{
		CharacterID: character.ID,
		Name:        req.Name,
		Prompt:      req.Prompt,
	}

	if err := s.db.Create(outfit).Error; err != nil {
		s.log.Errorw("Failed to create outfit", "error", err)
		return nil, err
	}

	return outfit, nil
}

func (s *CharacterLibraryService) UpdateCharacterOutfit(outfitID string, req *UpdateOutfitRequest) error {
	var outfit models.CharacterOutfit
	if err := s.db.Where("id = ?", outfitID).First(&outfit).Error; err != nil {
		return errors.New("outfit not found")
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Prompt != nil {
		updates["prompt"] = *req.Prompt
	}
	if req.ImageURL != nil {
		updates["image_url"] = *req.ImageURL
	}
	if req.LocalPath != nil {
		updates["local_path"] = *req.LocalPath
	}

	if len(updates) > 0 {
		if err := s.db.Model(&outfit).Updates(updates).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *CharacterLibraryService) DeleteCharacterOutfit(outfitID string) error {
	var outfit models.CharacterOutfit
	if err := s.db.Where("id = ?", outfitID).First(&outfit).Error; err != nil {
		return errors.New("outfit not found")
	}

	// Check if outfit is used in any storyboards
	var count int64
	s.db.Table("storyboard_characters").Where("outfit_id = ?", outfit.ID).Count(&count)
	if count > 0 {
		return fmt.Errorf("this outfit is being used in %d storyboard shots and cannot be deleted", count)
	}

	return s.db.Delete(&outfit).Error
}

func (s *CharacterLibraryService) GenerateOutfitImage(outfitID string, imageService *ImageGenerationService, modelName string) (*models.ImageGeneration, error) {
	var outfit models.CharacterOutfit
	if err := s.db.Where("id = ?", outfitID).First(&outfit).Error; err != nil {
		return nil, errors.New("outfit not found")
	}

	var character models.Character
	if err := s.db.Where("id = ?", outfit.CharacterID).First(&character).Error; err != nil {
		return nil, errors.New("character not found")
	}

	var drama models.Drama
	if err := s.db.Where("id = ?", character.DramaID).First(&drama).Error; err != nil {
		return nil, errors.New("drama not found")
	}

	// Reference image is the character's base image
	var refImages []string
	if character.ImageURL != nil && *character.ImageURL != "" {
		refImages = append(refImages, *character.ImageURL)
	} else if character.LocalPath != nil && *character.LocalPath != "" {
		refImages = append(refImages, *character.LocalPath)
	}

	if len(refImages) == 0 {
		return nil, errors.New("character must have an avatar image to generate outfit")
	}

	// Prompt logic
	prompt := character.Name + ", " + outfit.Prompt
	if drama.Style != "" && drama.Style != "realistic" {
		prompt += ", " + drama.Style
	}
	aspectRatio := drama.AspectRatio
	if aspectRatio == "" {
		aspectRatio = "16:9"
	}
	prompt += ", " + aspectRatio + " aspect ratio"

	dramaIDStr := fmt.Sprintf("%d", character.DramaID)
	req := &GenerateImageRequest{
		DramaID:         dramaIDStr,
		CharacterID:     &character.ID,
		ImageType:       "character_outfit", // New type or reuse character
		Prompt:          prompt,
		Provider:        "openai",
		Model:           modelName,
		Quality:         "standard",
		ReferenceImages: refImages,
	}

	imageGen, err := imageService.GenerateImage(req)
	if err != nil {
		return nil, err
	}

	go s.waitAndUpdateOutfitImage(outfit.ID, imageGen.ID)

	return imageGen, nil
}

func (s *CharacterLibraryService) waitAndUpdateOutfitImage(outfitID uint, imageGenID uint) {
	maxAttempts := 60
	pollInterval := 5 * time.Second

	for i := 0; i < maxAttempts; i++ {
		time.Sleep(pollInterval)
		var imageGen models.ImageGeneration
		if err := s.db.First(&imageGen, imageGenID).Error; err != nil {
			continue
		}

		if imageGen.Status == models.ImageStatusCompleted && imageGen.ImageURL != nil {
			updates := map[string]interface{}{
				"image_url": *imageGen.ImageURL,
			}
			if imageGen.LocalPath != nil {
				updates["local_path"] = *imageGen.LocalPath
			}
			s.db.Model(&models.CharacterOutfit{}).Where("id = ?", outfitID).Updates(updates)
			return
		} else if imageGen.Status == models.ImageStatusFailed {
			return
		}
	}
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

type MagicWandOutput struct {
	LinkedCharacterIDs []uint `json:"linked_character_ids"`
	NewCharacters      []struct {
		Name        string `json:"name"`
		Role        string `json:"role"`
		Appearance  string `json:"appearance"`
		Description string `json:"description"`
		Personality string `json:"personality"`
	} `json:"new_characters"`
}

func (s *CharacterLibraryService) processCharacterExtraction(taskID string, episode models.Episode) {
	s.taskService.UpdateTaskStatus(taskID, "processing", 0, "Analyzing script with Magic Wand...")

	script := ""
	if episode.ScriptContent != nil {
		script = *episode.ScriptContent
	}

	var drama models.Drama
	if err := s.db.First(&drama, episode.DramaID).Error; err != nil {
		s.log.Warnw("Failed to load drama", "error", err, "drama_id", episode.DramaID)
	}

	var existingCharacters []models.Character
	if err := s.db.Where("drama_id = ?", episode.DramaID).Find(&existingCharacters).Error; err != nil {
		s.taskService.UpdateTaskError(taskID, err)
		return
	}

	// Prepare data for prompt
	type existingCharData struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
	}
	var existingJSONList []existingCharData
	for _, c := range existingCharacters {
		existingJSONList = append(existingJSONList, existingCharData{ID: c.ID, Name: c.Name})
	}
	existingJSONBytes, _ := json.MarshalIndent(existingJSONList, "", "  ")

	system, err := narrativePromptFS.ReadFile("prompts/narrative/magic_wand_extract.md")
	if err != nil {
		s.taskService.UpdateTaskError(taskID, fmt.Errorf("read system prompt: %w", err))
		return
	}

	tpl, err := template.New("magic_wand").Parse(string(system))
	if err != nil {
		s.taskService.UpdateTaskError(taskID, fmt.Errorf("parse template: %w", err))
		return
	}

	var promptBuf bytes.Buffer
	_ = tpl.Execute(&promptBuf, struct {
		DramaTitle             string
		ExistingCharactersJSON string
		ScriptContent          string
	}{
		DramaTitle:             drama.Title,
		ExistingCharactersJSON: string(existingJSONBytes),
		ScriptContent:          script,
	})

	userBlock := "Please extract characters from the script using the magic wand rules and return the JSON."

	response, err := s.aiService.GenerateText(userBlock, promptBuf.String(), ai.WithMaxTokens(3000))
	if err != nil {
		s.taskService.UpdateTaskError(taskID, err)
		return
	}

	s.taskService.UpdateTaskStatus(taskID, "processing", 50, "Organizing extracted data...")

	var out MagicWandOutput
	if err := utils.SafeParseAIJSON(response, &out); err != nil {
		s.log.Errorw("Failed to parse AI response for characters", "error", err, "response", response)
		s.taskService.UpdateTaskError(taskID, fmt.Errorf("failed to parse AI response"))
		return
	}

	var savedCharacters []models.Character
	linkedIDSet := make(map[uint]bool)

	// Process linked characters
	for _, linkedID := range out.LinkedCharacterIDs {
		if linkedIDSet[linkedID] {
			continue
		}
		var existingCharacter *models.Character
		for i := range existingCharacters {
			if existingCharacters[i].ID == linkedID {
				existingCharacter = &existingCharacters[i]
				break
			}
		}
		if existingCharacter != nil {
			if err := s.db.Model(&episode).Association("Characters").Append(existingCharacter); err != nil {
				s.log.Warnw("Failed to associate existing character", "error", err)
			}
			savedCharacters = append(savedCharacters, *existingCharacter)
			linkedIDSet[existingCharacter.ID] = true
		}
	}

	// Process new characters
	for _, newChar := range out.NewCharacters {
		var matchedExisting *models.Character
		for i := range existingCharacters {
			if isLikelySameCharacterName(existingCharacters[i].Name, newChar.Name) {
				matchedExisting = &existingCharacters[i]
				break
			}
		}
		if matchedExisting != nil {
			if !linkedIDSet[matchedExisting.ID] {
				if err := s.db.Model(&episode).Association("Characters").Append(matchedExisting); err != nil {
					s.log.Warnw("Failed to associate fuzzy-matched character", "error", err)
				}
				savedCharacters = append(savedCharacters, *matchedExisting)
				linkedIDSet[matchedExisting.ID] = true
			}
			continue
		}

		roleCopy := newChar.Role
		descCopy := newChar.Description
		persCopy := newChar.Personality
		appCopy := newChar.Appearance

		character := models.Character{
			DramaID:     episode.DramaID,
			Name:        newChar.Name,
			Role:        &roleCopy,
			Description: &descCopy,
			Personality: &persCopy,
			Appearance:  &appCopy,
		}

		if err := s.db.Create(&character).Error; err != nil {
			s.log.Errorw("Failed to create extracted character", "error", err)
			continue
		}

		if err := s.db.Model(&episode).Association("Characters").Append(&character); err != nil {
			s.log.Warnw("Failed to associate new character", "error", err)
		}
		savedCharacters = append(savedCharacters, character)
		existingCharacters = append(existingCharacters, character)
		linkedIDSet[character.ID] = true
	}

	s.taskService.UpdateTaskResult(taskID, map[string]interface{}{
		"characters":            savedCharacters,
		"linked_character_ids":  out.LinkedCharacterIDs,
		"saved_character_count": len(savedCharacters),
	})
}
