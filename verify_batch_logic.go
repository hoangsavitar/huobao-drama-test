package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/drama-generator/backend/application/services"
	models "github.com/drama-generator/backend/domain/models"
	"github.com/drama-generator/backend/infrastructure/persistence"
	"github.com/drama-generator/backend/pkg/logger"
)

func main() {
	// Initialize DB
	dbPath := "data/drama.db"
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		log.Fatalf("Database not found at %s", dbPath)
	}

	db, err := persistence.NewSQLiteDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	l := logger.NewLogger()

	// Initialize Service
	// Note: We don't need all dependencies just to test the data gathering logic in BatchGenerate
	imageGen := services.NewImageGenerationService(db, nil, nil, nil, nil, l)

	episodeID := "7" // Drama 7, Episode 5 might have a different ID in the DB, let's find it
	var ep models.Episode
	if err := db.Where("id = ?", 5).First(&ep).Error; err != nil {
		fmt.Printf("Episode 5 not found, searching by drama_id 7\n")
		db.Where("drama_id = ?", 7).Order("id ASC").Offset(4).First(&ep)
	}

	fmt.Printf("Testing Batch Generation Logic for Episode ID: %d\n", ep.ID)

	// We will manually run the gathering logic part
	var scenes []models.Storyboard
	if err := db.Preload("Characters").Where("episode_id = ?", ep.ID).Order("storyboard_number ASC").Find(&scenes).Error; err != nil {
		log.Fatalf("Failed to load scenes: %v", err)
	}

	fmt.Printf("Found %d storyboards\n", len(scenes))

	for _, sb := range scenes {
		fmt.Printf("\n--- Shot %d ---\n", sb.StoryboardNumber)
		fmt.Printf("Prompt Preview: %s\n", *sb.ImagePrompt)
		
		var referenceImages []string
		// 1. Background
		if sb.SceneID != nil {
			var sc models.Scene
			if err := db.First(&sc, *sb.SceneID).Error; err == nil {
				if sc.ImageURL != nil {
					referenceImages = append(referenceImages, *sc.ImageURL)
					fmt.Printf("- Background Reference: %s\n", *sc.ImageURL)
				}
			}
		}

		// 2. Characters
		fmt.Printf("- Characters count: %d\n", len(sb.Characters))
		for _, char := range sb.Characters {
			if char.ImageURL != nil {
				referenceImages = append(referenceImages, *char.ImageURL)
				fmt.Printf("  * Character %s Reference: %s\n", char.Name, *char.ImageURL)
			}
		}

		fmt.Printf("TOTAL REFERENCES: %d\n", len(referenceImages))
	}
}
