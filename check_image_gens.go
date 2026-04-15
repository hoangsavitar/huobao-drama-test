package main

import (
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

	var episodes []models.Episode
	db.Where("drama_id = ? AND episode_number = ?", dramaID, 5).Find(&episodes)
	
	if len(episodes) == 0 {
		fmt.Println("No episodes found")
		return
	}
	
	episodeID := episodes[0].ID

	var imageGens []models.ImageGeneration
	if err := db.Where("drama_id = ? AND storyboard_id IN (SELECT id FROM storyboards WHERE episode_id = ?)", dramaID, episodeID).
		Order("created_at DESC").
		Find(&imageGens).Error; err != nil {
		log.Fatalf("failed to find image_generations: %v", err)
	}

	fmt.Printf("Found %d image generation records for Drama 7 Ep 5\n", len(imageGens))
	for _, ig := range imageGens {
		fmt.Printf("\nID: %d, StoryboardID: %v, Status: %s, CreatedAt: %v\n", ig.ID, *ig.StoryboardID, ig.Status, ig.CreatedAt)
		fmt.Printf("Prompt: %s\n", ig.Prompt)
		fmt.Printf("ReferenceImages: %s\n", string(ig.ReferenceImages))
		if ig.LocalPath != nil {
			fmt.Printf("LocalPath: %s\n", *ig.LocalPath)
		}
	}
}
