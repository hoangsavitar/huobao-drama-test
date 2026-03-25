package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/drama-generator/backend/application/services"
	"github.com/drama-generator/backend/domain/models"
	"github.com/drama-generator/backend/pkg/config"
	"github.com/drama-generator/backend/infrastructure/database"
)

func buildStoryboardContext(sb models.Storyboard, scene *models.Scene, promptI18n *services.PromptI18n) string {
	var parts []string

	if sb.Description != nil && *sb.Description != "" {
		parts = append(parts, promptI18n.FormatUserPrompt("shot_description_label", *sb.Description))
	}

	if scene != nil {
		parts = append(parts, promptI18n.FormatUserPrompt("scene_label", scene.Location, scene.Time))
	} else if sb.Location != nil && sb.Time != nil {
		parts = append(parts, promptI18n.FormatUserPrompt("scene_label", *sb.Location, *sb.Time))
	}

	if len(sb.Characters) > 0 {
		var charNames []string
		for _, char := range sb.Characters {
			charNames = append(charNames, char.Name)
		}
		parts = append(parts, promptI18n.FormatUserPrompt("characters_label", strings.Join(charNames, ", ")))
	}

	if sb.Action != nil && *sb.Action != "" {
		parts = append(parts, promptI18n.FormatUserPrompt("action_label", *sb.Action))
	}

	if sb.Result != nil && *sb.Result != "" {
		parts = append(parts, promptI18n.FormatUserPrompt("result_label", *sb.Result))
	}

	if sb.Dialogue != nil && *sb.Dialogue != "" {
		parts = append(parts, promptI18n.FormatUserPrompt("dialogue_label", *sb.Dialogue))
	}

	if sb.Atmosphere != nil && *sb.Atmosphere != "" {
		parts = append(parts, promptI18n.FormatUserPrompt("atmosphere_label", *sb.Atmosphere))
	}

	if sb.ShotType != nil {
		parts = append(parts, promptI18n.FormatUserPrompt("shot_type_label", *sb.ShotType))
	}
	if sb.Angle != nil {
		parts = append(parts, promptI18n.FormatUserPrompt("angle_label", *sb.Angle))
	}
	if sb.Movement != nil {
		parts = append(parts, promptI18n.FormatUserPrompt("movement_label", *sb.Movement))
	}

	return strings.Join(parts, "\n")
}

func main() {
	var dramaID uint
	var outputDir string

	flag.UintVar(&dramaID, "id", 0, "Drama (Project) ID to extract")
	flag.StringVar(&outputDir, "out", "exports", "Output directory for the extracted data")
	flag.Parse()

	if dramaID == 0 {
		fmt.Println("Please provide a valid Drama ID using -id=X")
		os.Exit(1)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := database.NewDatabase(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	promptI18n := services.NewPromptI18n(cfg)

	var drama models.Drama
	if err := db.Preload("Characters").Preload("Episodes").Preload("Scenes").First(&drama, dramaID).Error; err != nil {
		log.Fatalf("Failed to fetch drama ID %d: %v", dramaID, err)
	}

	os.MkdirAll(outputDir, os.ModePerm)
	
	// Create main markdown report
	fileName := filepath.Join(outputDir, fmt.Sprintf("drama_%d_prompts.md", drama.ID))
	f, err := os.Create(fileName)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer f.Close()

	// Create Image Only Prompt File
	imageOnlyFileName := filepath.Join(outputDir, fmt.Sprintf("drama_%d_image_prompts.txt", drama.ID))
	fImg, err := os.Create(imageOnlyFileName)
	if err != nil {
		log.Fatalf("Failed to create image prompt file: %v", err)
	}
	defer fImg.Close()

	// Create Video Only Prompt File
	videoOnlyFileName := filepath.Join(outputDir, fmt.Sprintf("drama_%d_video_prompts.txt", drama.ID))
	fVid, err := os.Create(videoOnlyFileName)
	if err != nil {
		log.Fatalf("Failed to create video prompt file: %v", err)
	}
	defer fVid.Close()

	writeLine := func(data string) {
		f.WriteString(data + "\n")
	}
	
	writeImgLine := func(data string) {
		fImg.WriteString(data + "\n")
	}

	writeVidLine := func(data string) {
		fVid.WriteString(data + "\n")
	}

	writeLine(fmt.Sprintf("# Project Report: %s (ID: %d)", drama.Title, drama.ID))
	writeLine(fmt.Sprintf("**Style**: %s", drama.Style))
	if drama.Description != nil {
		writeLine(fmt.Sprintf("**Description**: %s", *drama.Description))
	}

	writeLine("\n## Characters Reference summary")
	for _, char := range drama.Characters {
		writeLine(fmt.Sprintf("### %s", char.Name))
		if char.Appearance != nil {
			writeLine(fmt.Sprintf("- **Appearance**: %s", *char.Appearance))
		}
		
		if char.ImageURL != nil && *char.ImageURL != "" {
			writeLine(fmt.Sprintf("- **Avatar**: ![Avatar](%s)", *char.ImageURL))
		}

		var refs []string
		if char.ReferenceImages != nil {
			json.Unmarshal(char.ReferenceImages, &refs)
			if len(refs) > 0 {
				writeLine("- **Reference Images Uploaded**:")
				for i, ref := range refs {
					writeLine(fmt.Sprintf("  - ![Ref %d](%s)", i+1, ref))
				}
			}
		}
	}

	writeLine("\n## Scenes Summary")
	for _, scene := range drama.Scenes {
		writeLine(fmt.Sprintf("### %s (%s)", scene.Location, scene.Time))
		writeLine(fmt.Sprintf("- **Prompt**: %s", scene.Prompt))
		if scene.ImageURL != nil && *scene.ImageURL != "" {
			writeLine(fmt.Sprintf("- **Image**: ![Scene](%s)", *scene.ImageURL))
		}
	}

	writeLine("\n## Storyboards & Frames Extraction")
	
	systemPrompt := promptI18n.GetFirstFramePrompt(drama.Style, drama.AspectRatio)
	systemPrompt += "\n\nImportant Character and Scene Consistency Instructions:"
	systemPrompt += "\n6. Character Consistency: You MUST strictly adhere to the character appearances described in the 'Characters Reference summary' section. Ensure their clothing, facial features, and style perfectly match the provided descriptions."
	systemPrompt += "\n7. Scene Accuracy: Always respect the 'Scenes Summary' when a specific location is mentioned. Keep the backgrounds consistent with the established scene parameters."
	systemPrompt += "\n8. Do not hallucinate or invent new character aesthetics that conflict with their reference."

	writeLine("### Global System Prompt for First Frame Generation")
	writeLine("Use this System Prompt for all shots below:")
	writeLine("```text")
	writeLine(systemPrompt)
	writeLine("```\n")

	for _, ep := range drama.Episodes {
		writeImgLine(fmt.Sprintf("=== Episode %d: %s ===", ep.EpisodeNum, ep.Title))
		writeVidLine(fmt.Sprintf("=== Episode %d: %s ===", ep.EpisodeNum, ep.Title))

		var storyboards []models.Storyboard
		if err := db.Preload("Background").Preload("Characters").Where("episode_id = ?", ep.ID).Order("storyboard_number asc").Find(&storyboards).Error; err != nil {
			continue
		}

		for _, sb := range storyboards {
			shotHeader := fmt.Sprintf("Shot %d", sb.StoryboardNumber)
			writeLine(fmt.Sprintf("\n#### %s (Storyboard ID: %d)", shotHeader, sb.ID))
			writeImgLine("\n" + shotHeader)
			writeVidLine("\n" + shotHeader)
			
			if sb.Action != nil {
				writeLine(fmt.Sprintf("- **Action**: %s", *sb.Action))
			}
			if sb.Atmosphere != nil {
				writeLine(fmt.Sprintf("- **Atmosphere**: %s", *sb.Atmosphere))
			}
			if sb.ComposedImage != nil && *sb.ComposedImage != "" {
				writeLine(fmt.Sprintf("- **Shot Image (Composed)**: ![Shot](%s)", *sb.ComposedImage))
			}

			var assets []models.Asset
			if err := db.Where("storyboard_id = ? AND type IN ('image', 'video')", sb.ID).Find(&assets).Error; err == nil && len(assets) > 0 {
				for _, asset := range assets {
					if asset.Type == "image" {
						writeLine(fmt.Sprintf("- **Asset Image**: ![Shot](%s)", asset.URL))
					} else if asset.Type == "video" {
						writeLine(fmt.Sprintf("- **Asset Video**: %s", asset.URL))
					}
				}
			}

			var frames []models.FramePrompt
			if err := db.Where("storyboard_id = ?", sb.ID).Find(&frames).Error; err == nil && len(frames) > 0 {
				writeLine("\n**Extracted Image Prompts (Copy for Image Generation)**:")
				for _, fp := range frames {
					writeLine(fmt.Sprintf("- **[%s]**: `%s`", fp.FrameType, fp.Prompt))
				}
			} else {
				writeLine("\n*No frame prompts extracted yet in Database.*")
			}

			// Lấy nội dung context chuẩn bị gửi qua AI cho khung hình First Frame
			contextInfo := buildStoryboardContext(sb, sb.Background, promptI18n)
			userPrompt := promptI18n.FormatUserPrompt("frame_info", contextInfo)

			writeLine("\n> *(API User Prompt to generate First Frame for this shot)*")
			writeLine(fmt.Sprintf("> **User Prompt**:\n> ```text\n> %s\n> ```\n", strings.ReplaceAll(userPrompt, "\n", "\n> ")))

			// Write User Prompt exclusively to the image prompts file
			writeImgLine("User Prompt:\n")
			writeImgLine(userPrompt)
			writeImgLine("\n")

			if sb.VideoPrompt != nil && *sb.VideoPrompt != "" {
				writeLine("\n**Video Generation Prompt**:")
				writeLine(fmt.Sprintf("`%s`", *sb.VideoPrompt))
				writeVidLine(*sb.VideoPrompt)
			} else {
				writeVidLine("*No video prompt available.*")
			}
		}
	}

	fmt.Printf("\nExtraction completed!\n- Markdown Report: %s\n- Image Prompts: %s\n- Video Prompts: %s\n", fileName, imageOnlyFileName, videoOnlyFileName)
}
