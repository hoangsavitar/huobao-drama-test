package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/drama-generator/backend/infrastructure/database"
	"github.com/drama-generator/backend/domain/models"
	"github.com/drama-generator/backend/pkg/config"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	db, err := database.NewDatabase(cfg.Database)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	dramaID := uint(7)
	episodeNum := 5

	var drama models.Drama
	if err := db.Preload("Characters").First(&drama, dramaID).Error; err != nil {
		log.Fatalf("failed to found drama %d: %v", dramaID, err)
	}

	fmt.Printf("Drama: %s\n", drama.Title)
	fmt.Println("Characters:")
	for _, char := range drama.Characters {
		appearance := ""
		if char.Appearance != nil {
			appearance = *char.Appearance
		}
		imgURL := ""
		if char.ImageURL != nil {
			imgURL = *char.ImageURL
		}
		localPath := ""
		if char.LocalPath != nil {
			localPath = *char.LocalPath
		}
		fmt.Printf("- ID: %d, Name: %s, Appearance: %s, ImageURL: %s, LocalPath: %s\n", char.ID, char.Name, appearance, imgURL, localPath)
		if char.Name == "Viper" || char.Name == "Seo-yeon" {
			refImages, _ := json.MarshalIndent(char.ReferenceImages, "", "  ")
			fmt.Printf("  %s Reference Images: %s\n", char.Name, string(refImages))
		}
	}

	var episode models.Episode
	if err := db.Where("drama_id = ? AND episode_number = ?", dramaID, episodeNum).First(&episode).Error; err != nil {
		log.Fatalf("failed to find episode %d for drama %d: %v", episodeNum, dramaID, err)
	}

	fmt.Printf("\nEpisode: %s (ID: %d)\n", episode.Title, episode.ID)

	var storyboards []models.Storyboard
	if err := db.Preload("Characters").Preload("Background").
		Where("episode_id = ? AND storyboard_number IN (3, 4, 5)", episode.ID).
		Order("storyboard_number ASC").
		Find(&storyboards).Error; err != nil {
		log.Fatalf("failed to find storyboards: %v", err)
	}

	for _, sb := range storyboards {
		fmt.Printf("\n--- Shot %d ---\n", sb.StoryboardNumber)
		action := ""
		if sb.Action != nil {
			action = *sb.Action
		}
		imgPrompt := ""
		if sb.ImagePrompt != nil {
			imgPrompt = *sb.ImagePrompt
		}
		fmt.Printf("Action: %s\n", action)
		fmt.Printf("Image Prompt: %s\n", imgPrompt)
		if sb.Background != nil {
			fmt.Printf("Background Prompt: %s\n", sb.Background.Prompt)
			fmt.Printf("Background image URL: %v\n", sb.Background.ImageURL)
		}
		fmt.Println("Characters in shot:")
		for _, c := range sb.Characters {
			app := ""
			if c.Appearance != nil {
				app = *c.Appearance
			}
			fmt.Printf("  - %s (Appearance: %s)\n", c.Name, app)
			refStr, _ := json.Marshal(c.ReferenceImages)
			fmt.Printf("    Reference Images: %s\n", string(refStr))
		}
	}
}
