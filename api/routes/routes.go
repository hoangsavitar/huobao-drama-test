package routes

import (
	handlers2 "github.com/drama-generator/backend/api/handlers"
	middlewares2 "github.com/drama-generator/backend/api/middlewares"
	services2 "github.com/drama-generator/backend/application/services"
	storage2 "github.com/drama-generator/backend/infrastructure/storage"
	"github.com/drama-generator/backend/pkg/config"
	"github.com/drama-generator/backend/pkg/logger"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRouter(cfg *config.Config, db *gorm.DB, log *logger.Logger, localStorage interface{}) *gin.Engine {
	r := gin.New()

	r.Use(gin.Recovery())
	r.Use(middlewares2.LoggerMiddleware(log))
	r.Use(middlewares2.CORSMiddleware(cfg.Server.CORSOrigins))

	// Static file serving (user uploaded files)
	r.Static("/static", cfg.Storage.LocalPath)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"app":     cfg.App.Name,
			"version": cfg.App.Version,
		})
	})

	aiService := services2.NewAIService(db, log)
	localStoragePtr := localStorage.(*storage2.LocalStorage)
	transferService := services2.NewResourceTransferService(db, log)
	promptI18n := services2.NewPromptI18n(cfg)
	dramaHandler := handlers2.NewDramaHandler(db, cfg, log, nil)
	aiConfigHandler := handlers2.NewAIConfigHandler(db, cfg, log)
	scriptGenHandler := handlers2.NewScriptGenerationHandler(db, cfg, log)
	imageGenService := services2.NewImageGenerationService(db, cfg, transferService, localStoragePtr, log)
	imageGenHandler := handlers2.NewImageGenerationHandler(db, cfg, log, transferService, localStoragePtr)
	videoGenHandler := handlers2.NewVideoGenerationHandler(db, transferService, localStoragePtr, aiService, log, promptI18n)
	videoMergeHandler := handlers2.NewVideoMergeHandler(db, nil, cfg.Storage.LocalPath, cfg.Storage.BaseURL, log)
	assetHandler := handlers2.NewAssetHandler(db, cfg, log)
	characterLibraryService := services2.NewCharacterLibraryService(db, log, cfg)
	characterLibraryHandler := handlers2.NewCharacterLibraryHandler(db, cfg, log, transferService, localStoragePtr)
	uploadHandler, err := handlers2.NewUploadHandler(cfg, log, characterLibraryService)
	if err != nil {
		log.Fatalw("Failed to create upload handler", "error", err)
	}
	storyboardHandler := handlers2.NewStoryboardHandler(db, cfg, log)
	sceneHandler := handlers2.NewSceneHandler(db, log, imageGenService)
	taskHandler := handlers2.NewTaskHandler(db, log)
	framePromptService := services2.NewFramePromptService(db, cfg, log)
	framePromptHandler := handlers2.NewFramePromptHandler(framePromptService, log)
	audioExtractionHandler := handlers2.NewAudioExtractionHandler(log, cfg.Storage.LocalPath)
	settingsHandler := handlers2.NewSettingsHandler(cfg, log)
	propHandler := handlers2.NewPropHandler(db, cfg, log, aiService, imageGenService)
	ltxVideoPromptService := services2.NewLtxVideoPromptBatchService(db, log, aiService)
	ltxVideoPromptHandler := handlers2.NewLtxVideoPromptHandler(ltxVideoPromptService, log)

	api := r.Group("/api/v1")
	{
		api.Use(middlewares2.RateLimitMiddleware())

		dramas := api.Group("/dramas")
		{
			dramas.GET("", dramaHandler.ListDramas)
			dramas.POST("", dramaHandler.CreateDrama)
			dramas.GET("/stats", dramaHandler.GetDramaStats) // Stats endpoint must be before /:id
			dramas.GET("/:id", dramaHandler.GetDrama)
			dramas.PUT("/:id", dramaHandler.UpdateDrama)
			dramas.DELETE("/:id", dramaHandler.DeleteDrama)

			dramas.PUT("/:id/outline", dramaHandler.SaveOutline)
			dramas.GET("/:id/characters", dramaHandler.GetCharacters)
			dramas.PUT("/:id/characters", dramaHandler.SaveCharacters)
			dramas.PUT("/:id/episodes", dramaHandler.SaveEpisodes)
			dramas.POST("/:id/narrative/generate", dramaHandler.GenerateNarrativeEpisodes)
			dramas.PUT("/:id/progress", dramaHandler.SaveProgress)
			dramas.GET("/:id/props", propHandler.ListProps) // Added prop list route
		}

		aiConfigs := api.Group("/ai-configs")
		{
			aiConfigs.GET("", aiConfigHandler.ListConfigs)
			aiConfigs.POST("", aiConfigHandler.CreateConfig)
			aiConfigs.POST("/test", aiConfigHandler.TestConnection)
			aiConfigs.GET("/:id", aiConfigHandler.GetConfig)
			aiConfigs.PUT("/:id", aiConfigHandler.UpdateConfig)
			aiConfigs.DELETE("/:id", aiConfigHandler.DeleteConfig)
		}

		generation := api.Group("/generation")
		{
			generation.POST("/characters", scriptGenHandler.GenerateCharacters)
		}

		// Character library routes
		characterLibrary := api.Group("/character-library")
		{
			characterLibrary.GET("", characterLibraryHandler.ListLibraryItems)
			characterLibrary.POST("", characterLibraryHandler.CreateLibraryItem)
			characterLibrary.GET("/:id", characterLibraryHandler.GetLibraryItem)
			characterLibrary.DELETE("/:id", characterLibraryHandler.DeleteLibraryItem)
		}

		// Character image related routes
		characters := api.Group("/characters")
		{
			characters.PUT("/:id", characterLibraryHandler.UpdateCharacter)
			characters.DELETE("/:id", characterLibraryHandler.DeleteCharacter)
			characters.POST("/batch-generate-images", characterLibraryHandler.BatchGenerateCharacterImages)
			characters.POST("/:id/generate-image", characterLibraryHandler.GenerateCharacterImage)
			characters.POST("/:id/upload-image", uploadHandler.UploadCharacterImage)
			characters.PUT("/:id/image", characterLibraryHandler.UploadCharacterImage)
			characters.PUT("/:id/image-from-library", characterLibraryHandler.ApplyLibraryItemToCharacter)
			characters.POST("/:id/add-to-library", characterLibraryHandler.AddCharacterToLibrary)
		}

		props := api.Group("/props")
		{
			props.POST("", propHandler.CreateProp)
			props.PUT("/:id", propHandler.UpdateProp)
			props.DELETE("/:id", propHandler.DeleteProp)
			props.POST("/:id/generate", propHandler.GenerateImage)
		}

		// File upload routes
		upload := api.Group("/upload")
		{
			upload.POST("/image", uploadHandler.UploadImage)
		}

		// Storyboard routes
		episodes := api.Group("/episodes")
		{
			// Storyboards
			episodes.POST("/:episode_id/storyboards", storyboardHandler.GenerateStoryboard)
			episodes.POST("/:episode_id/props/extract", propHandler.ExtractProps)
			episodes.POST("/:episode_id/characters/extract", characterLibraryHandler.ExtractCharacters)
			episodes.GET("/:episode_id/storyboards", sceneHandler.GetStoryboardsForEpisode)
			episodes.GET("/:episode_id/frame-prompts", handlers2.GetEpisodeFramePrompts(db, log))
			episodes.POST("/:episode_id/ltx-video-prompts/batch", ltxVideoPromptHandler.BatchGenerateLtxVideoPrompts)
			episodes.POST("/:episode_id/finalize", dramaHandler.FinalizeEpisode)
			episodes.GET("/:episode_id/download", dramaHandler.DownloadEpisodeVideo)
		}

		// Task routes
		tasks := api.Group("/tasks")
		{
			tasks.GET("/:task_id", taskHandler.GetTaskStatus)
			tasks.GET("", taskHandler.GetResourceTasks)
		}

		// Scene routes
		scenes := api.Group("/scenes")
		{
			scenes.PUT("/:scene_id", sceneHandler.UpdateScene)
			scenes.PUT("/:scene_id/prompt", sceneHandler.UpdateScenePrompt)
			scenes.DELETE("/:scene_id", sceneHandler.DeleteScene)

			scenes.POST("/generate-image", sceneHandler.GenerateSceneImage)
			scenes.POST("", sceneHandler.CreateScene)
		}

		images := api.Group("/images")
		{
			images.GET("", imageGenHandler.ListImageGenerations)
			images.POST("", imageGenHandler.GenerateImage)
			images.GET("/:id", imageGenHandler.GetImageGeneration)
			images.DELETE("/:id", imageGenHandler.DeleteImageGeneration)
			images.POST("/scene/:scene_id", imageGenHandler.GenerateImagesForScene)
			images.POST("/upload", imageGenHandler.UploadImage)
			images.GET("/episode/:episode_id/backgrounds", imageGenHandler.GetBackgroundsForEpisode)
			images.POST("/episode/:episode_id/backgrounds/extract", imageGenHandler.ExtractBackgroundsForEpisode)
			images.POST("/episode/:episode_id/batch", imageGenHandler.BatchGenerateForEpisode)
		}

		videos := api.Group("/videos")
		{
			videos.GET("", videoGenHandler.ListVideoGenerations)
			videos.POST("", videoGenHandler.GenerateVideo)
			videos.GET("/:id", videoGenHandler.GetVideoGeneration)
			videos.DELETE("/:id", videoGenHandler.DeleteVideoGeneration)
			videos.POST("/image/:image_gen_id", videoGenHandler.GenerateVideoFromImage)
			videos.POST("/episode/:episode_id/batch", videoGenHandler.BatchGenerateForEpisode)
		}

		videoMerges := api.Group("/video-merges")
		{
			videoMerges.GET("", videoMergeHandler.ListMerges)
			videoMerges.POST("", videoMergeHandler.MergeVideos)
			videoMerges.GET("/:merge_id", videoMergeHandler.GetMerge)
			videoMerges.DELETE("/:merge_id", videoMergeHandler.DeleteMerge)
		}

		assets := api.Group("/assets")
		{
			assets.GET("", assetHandler.ListAssets)
			assets.POST("", assetHandler.CreateAsset)
			assets.GET("/:id", assetHandler.GetAsset)
			assets.PUT("/:id", assetHandler.UpdateAsset)
			assets.DELETE("/:id", assetHandler.DeleteAsset)
			assets.POST("/import/image/:image_gen_id", assetHandler.ImportFromImageGen)
			assets.POST("/import/video/:video_gen_id", assetHandler.ImportFromVideoGen)
		}

		storyboards := api.Group("/storyboards")
		{
			storyboards.GET("/episode/:episode_id/generate", storyboardHandler.GenerateStoryboard)
			storyboards.POST("", storyboardHandler.CreateStoryboard)
			storyboards.PUT("/:id", storyboardHandler.UpdateStoryboard)
			storyboards.DELETE("/:id", storyboardHandler.DeleteStoryboard)
			storyboards.POST("/:id/props", propHandler.AssociateProps)
			storyboards.POST("/:id/frame-prompt", framePromptHandler.GenerateFramePrompt)
			storyboards.PUT("/:id/frame-prompt", handlers2.UpdateFramePrompt(db, log))
			storyboards.GET("/:id/frame-prompts", handlers2.GetStoryboardFramePrompts(db, log))
		}

		audio := api.Group("/audio")
		{
			audio.POST("/extract", audioExtractionHandler.ExtractAudio)
			audio.POST("/extract/batch", audioExtractionHandler.BatchExtractAudio)
		}

		settings := api.Group("/settings")
		{
			settings.GET("/language", settingsHandler.GetLanguage)
			settings.PUT("/language", settingsHandler.UpdateLanguage)
		}
	}

	// Frontend static file serving (placed after API routes to avoid conflicts)
	// Serve frontend build artifacts
	r.Static("/assets", "./web/dist/assets")
	r.StaticFile("/favicon.ico", "./web/dist/favicon.ico")

	// NoRoute handler: for all unmatched routes
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// If it is an API path, return 404
		if len(path) >= 4 && path[:4] == "/api" {
			c.JSON(404, gin.H{"error": "API endpoint not found"})
			return
		}

		// SPA fallback - return index.html
		c.File("./web/dist/index.html")
	})

	return r
}
