package services

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/drama-generator/backend/domain/models"
	"github.com/drama-generator/backend/pkg/logger"

	"gorm.io/gorm"
)

type DataMigrationService struct {
	db          *gorm.DB
	log         *logger.Logger
	storageRoot string
	urlMapping  map[string]string // 原始URL -> 本地路径的映射
}

func NewDataMigrationService(db *gorm.DB, log *logger.Logger) *DataMigrationService {
	return &DataMigrationService{
		db:          db,
		log:         log,
		storageRoot: "data/storage",
		urlMapping:  make(map[string]string),
	}
}

// MigrateLocalPaths 迁移所有表中 local_path 为空的数据
func (s *DataMigrationService) MigrateLocalPaths() error {
	s.log.Info("Starting data cleanup: migrate records with empty local_path")
	startTime := time.Now()

	// 确保存储目录存在
	if err := s.ensureStorageDirectories(); err != nil {
		return fmt.Errorf("failed to create storage directories: %w", err)
	}

	// 迁移各个表的数据（按指定顺序）
	stats := &MigrationStats{}

	// 1. 迁移 assets 表
	if err := s.migrateAssets(stats); err != nil {
		s.log.Errorw("Failed to migrate assets", "error", err)
	}

	// 2. 迁移 character_libraries 表
	if err := s.migrateCharacterLibraries(stats); err != nil {
		s.log.Errorw("Failed to migrate character_libraries", "error", err)
	}

	// 3. 迁移 characters 表
	if err := s.migrateCharacters(stats); err != nil {
		s.log.Errorw("Failed to migrate characters", "error", err)
	}

	// 4. 迁移 image_generations 表
	if err := s.migrateImageGenerations(stats); err != nil {
		s.log.Errorw("Failed to migrate image_generations", "error", err)
	}

	// 5. 迁移 scenes 表
	if err := s.migrateScenes(stats); err != nil {
		s.log.Errorw("Failed to migrate scenes", "error", err)
	}

	// 6. 迁移 video_generations 表
	if err := s.migrateVideoGenerations(stats); err != nil {
		s.log.Errorw("Failed to migrate video_generations", "error", err)
	}

	duration := time.Since(startTime)
	s.log.Infow("Data cleanup completed",
		"duration", duration.String(),
		"url_mapping_cache_size", len(s.urlMapping),
		"assets_success", stats.AssetsSuccess,
		"assets_failed", stats.AssetsFailed,
		"character_libraries_success", stats.CharacterLibrariesSuccess,
		"character_libraries_failed", stats.CharacterLibrariesFailed,
		"characters_success", stats.CharactersSuccess,
		"characters_failed", stats.CharactersFailed,
		"image_generations_success", stats.ImageGenerationsSuccess,
		"image_generations_failed", stats.ImageGenerationsFailed,
		"scenes_success", stats.ScenesSuccess,
		"scenes_failed", stats.ScenesFailed,
		"videos_success", stats.VideosSuccess,
		"videos_failed", stats.VideosFailed,
	)

	return nil
}

// MigrationStats 迁移统计信息
type MigrationStats struct {
	AssetsSuccess               int
	AssetsFailed                int
	CharacterLibrariesSuccess   int
	CharacterLibrariesFailed    int
	CharactersSuccess           int
	CharactersFailed            int
	ImageGenerationsSuccess     int
	ImageGenerationsFailed      int
	ScenesSuccess               int
	ScenesFailed                int
	VideosSuccess               int
	VideosFailed                int
}

// ensureStorageDirectories 确保存储目录存在
func (s *DataMigrationService) ensureStorageDirectories() error {
	dirs := []string{
		filepath.Join(s.storageRoot, "images"),
		filepath.Join(s.storageRoot, "characters"),
		filepath.Join(s.storageRoot, "videos"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	s.log.Infow("Storage directories created", "root", s.storageRoot)
	return nil
}

// migrateAssets 迁移 assets 表数据
func (s *DataMigrationService) migrateAssets(stats *MigrationStats) error {
	s.log.Info("Start migrating assets...")

	var assets []models.Asset
	// 查询 local_path 为空但 url 不为空的资源
	if err := s.db.Where("(local_path IS NULL OR local_path = '') AND url IS NOT NULL AND url != ''").Find(&assets).Error; err != nil {
		return fmt.Errorf("failed to query assets: %w", err)
	}

	s.log.Infow("Found assets to migrate", "count", len(assets))

	for _, asset := range assets {
		s.log.Infow("Processing asset", "id", asset.ID, "name", asset.Name, "type", asset.Type, "url", asset.URL)

		// 根据类型选择存储目录
		subDir := "images"
		if asset.Type == models.AssetTypeVideo {
			subDir = "videos"
		}

		localPath, err := s.downloadOrGetCached(asset.URL, subDir, fmt.Sprintf("asset_%d", asset.ID))
		if err != nil {
			s.log.Errorw("Failed to download asset", "asset_id", asset.ID, "error", err)
			stats.AssetsFailed++
			continue
		}

		// 更新 local_path
		if err := s.db.Model(&asset).Update("local_path", localPath).Error; err != nil {
			s.log.Errorw("Failed to update asset local_path", "asset_id", asset.ID, "error", err)
			stats.AssetsFailed++
			continue
		}

		s.log.Infow("Asset migrated", "asset_id", asset.ID, "local_path", localPath)
		stats.AssetsSuccess++
	}

	return nil
}

// migrateCharacterLibraries 迁移 character_libraries 表数据
func (s *DataMigrationService) migrateCharacterLibraries(stats *MigrationStats) error {
	s.log.Info("Start migrating character_libraries...")

	var charLibs []models.CharacterLibrary
	// 查询 local_path 为空但 image_url 不为空的角色库
	if err := s.db.Where("(local_path IS NULL OR local_path = '') AND image_url IS NOT NULL AND image_url != ''").Find(&charLibs).Error; err != nil {
		return fmt.Errorf("failed to query character_libraries: %w", err)
	}

	s.log.Infow("Found character_libraries to migrate", "count", len(charLibs))

	for _, charLib := range charLibs {
		s.log.Infow("Processing character_library", "id", charLib.ID, "name", charLib.Name, "image_url", charLib.ImageURL)

		localPath, err := s.downloadOrGetCached(charLib.ImageURL, "characters", fmt.Sprintf("charlib_%d", charLib.ID))
		if err != nil {
			s.log.Errorw("Failed to download character_library image", "charlib_id", charLib.ID, "error", err)
			stats.CharacterLibrariesFailed++
			continue
		}

		// 更新 local_path
		if err := s.db.Model(&charLib).Update("local_path", localPath).Error; err != nil {
			s.log.Errorw("Failed to update character_library local_path", "charlib_id", charLib.ID, "error", err)
			stats.CharacterLibrariesFailed++
			continue
		}

		s.log.Infow("character_library migrated", "charlib_id", charLib.ID, "local_path", localPath)
		stats.CharacterLibrariesSuccess++
	}

	return nil
}

// migrateImageGenerations 迁移 image_generations 表数据
func (s *DataMigrationService) migrateImageGenerations(stats *MigrationStats) error {
	s.log.Info("Start migrating image_generations...")

	var imageGens []models.ImageGeneration
	// 查询 local_path 为空但 image_url 不为空的图片生成记录
	if err := s.db.Where("(local_path IS NULL OR local_path = '') AND image_url IS NOT NULL AND image_url != ''").Find(&imageGens).Error; err != nil {
		return fmt.Errorf("failed to query image_generations: %w", err)
	}

	s.log.Infow("Found image_generations to migrate", "count", len(imageGens))

	for _, imageGen := range imageGens {
		if imageGen.ImageURL == nil {
			continue
		}

		imageTypeStr := string(imageGen.ImageType)
		s.log.Infow("Processing image_generation", "id", imageGen.ID, "image_type", imageTypeStr, "image_url", *imageGen.ImageURL)

		// 根据图片类型选择存储目录
		subDir := "images"
		if imageGen.ImageType == "character" {
			subDir = "characters"
		}

		localPath, err := s.downloadOrGetCached(*imageGen.ImageURL, subDir, fmt.Sprintf("imggen_%d", imageGen.ID))
		if err != nil {
			s.log.Errorw("Failed to download image_generation image", "imggen_id", imageGen.ID, "error", err)
			stats.ImageGenerationsFailed++
			continue
		}

		// 更新 local_path
		if err := s.db.Model(&imageGen).Update("local_path", localPath).Error; err != nil {
			s.log.Errorw("Failed to update image_generation local_path", "imggen_id", imageGen.ID, "error", err)
			stats.ImageGenerationsFailed++
			continue
		}

		s.log.Infow("image_generation migrated", "imggen_id", imageGen.ID, "local_path", localPath)
		stats.ImageGenerationsSuccess++
	}

	return nil
}

// migrateScenes 迁移场景数据
func (s *DataMigrationService) migrateScenes(stats *MigrationStats) error {
	s.log.Info("Start migrating scenes...")

	var scenes []models.Scene
	// 查询 local_path 为空但 image_url 不为空的场景
	if err := s.db.Where("(local_path IS NULL OR local_path = '') AND image_url IS NOT NULL AND image_url != ''").Find(&scenes).Error; err != nil {
		return fmt.Errorf("failed to query scenes: %w", err)
	}

	s.log.Infow("Found scenes to migrate", "count", len(scenes))

	for _, scene := range scenes {
		if scene.ImageURL == nil {
			continue
		}
		s.log.Infow("Processing scene", "id", scene.ID, "location", scene.Location, "image_url", *scene.ImageURL)

		localPath, err := s.downloadOrGetCached(*scene.ImageURL, "images", fmt.Sprintf("scene_%d", scene.ID))
		if err != nil {
			s.log.Errorw("Failed to download scene image", "scene_id", scene.ID, "error", err)
			stats.ScenesFailed++
			continue
		}

		// 更新 local_path
		if err := s.db.Model(&scene).Update("local_path", localPath).Error; err != nil {
			s.log.Errorw("Failed to update scene local_path", "scene_id", scene.ID, "error", err)
			stats.ScenesFailed++
			continue
		}

		s.log.Infow("Scene migrated", "scene_id", scene.ID, "local_path", localPath)
		stats.ScenesSuccess++
	}

	return nil
}

// migrateCharacters 迁移角色数据
func (s *DataMigrationService) migrateCharacters(stats *MigrationStats) error {
	s.log.Info("Start migrating characters...")

	var characters []models.Character
	// 查询 local_path 为空但 image_url 不为空的角色
	if err := s.db.Where("(local_path IS NULL OR local_path = '') AND image_url IS NOT NULL AND image_url != ''").Find(&characters).Error; err != nil {
		return fmt.Errorf("failed to query characters: %w", err)
	}

	s.log.Infow("Found characters to migrate", "count", len(characters))

	for _, character := range characters {
		if character.ImageURL == nil {
			continue
		}
		s.log.Infow("Processing character", "id", character.ID, "name", character.Name, "image_url", *character.ImageURL)

		localPath, err := s.downloadOrGetCached(*character.ImageURL, "characters", fmt.Sprintf("character_%d", character.ID))
		if err != nil {
			s.log.Errorw("Failed to download character image", "character_id", character.ID, "error", err)
			stats.CharactersFailed++
			continue
		}

		// 更新 local_path
		if err := s.db.Model(&character).Update("local_path", localPath).Error; err != nil {
			s.log.Errorw("Failed to update character local_path", "character_id", character.ID, "error", err)
			stats.CharactersFailed++
			continue
		}

		s.log.Infow("Character migrated", "character_id", character.ID, "local_path", localPath)
		stats.CharactersSuccess++
	}

	return nil
}

// migrateVideoGenerations 迁移视频生成数据
func (s *DataMigrationService) migrateVideoGenerations(stats *MigrationStats) error {
	s.log.Info("Start migrating video generations...")

	var videoGens []models.VideoGeneration
	// 查询 local_path 为空但 video_url 不为空的视频
	if err := s.db.Where("(local_path IS NULL OR local_path = '') AND video_url IS NOT NULL AND video_url != ''").Find(&videoGens).Error; err != nil {
		return fmt.Errorf("failed to query video generations: %w", err)
	}

	s.log.Infow("Found videos to migrate", "count", len(videoGens))

	for _, videoGen := range videoGens {
		if videoGen.VideoURL == nil {
			continue
		}
		s.log.Infow("Processing video", "id", videoGen.ID, "video_url", *videoGen.VideoURL)

		localPath, err := s.downloadOrGetCached(*videoGen.VideoURL, "videos", fmt.Sprintf("video_%d", videoGen.ID))
		if err != nil {
			s.log.Errorw("Failed to download video", "video_gen_id", videoGen.ID, "error", err)
			stats.VideosFailed++
			continue
		}

		// 更新 local_path
		if err := s.db.Model(&videoGen).Update("local_path", localPath).Error; err != nil {
			s.log.Errorw("Failed to update video local_path", "video_gen_id", videoGen.ID, "error", err)
			stats.VideosFailed++
			continue
		}

		s.log.Infow("Video migrated", "video_gen_id", videoGen.ID, "local_path", localPath)
		stats.VideosSuccess++
	}

	return nil
}

// downloadOrGetCached 下载文件或从缓存获取本地路径
func (s *DataMigrationService) downloadOrGetCached(url, subDir, prefix string) (string, error) {
	// 1. 检查 URL 映射缓存
	if localPath, exists := s.urlMapping[url]; exists {
		s.log.Infow("Using cached local path", "url", url, "local_path", localPath)
		return localPath, nil
	}

	// 2. 如果缓存中没有，则下载文件
	var localPath string
	var err error

	// 根据子目录判断是图片还是视频
	if subDir == "videos" {
		localPath, err = s.downloadAndSaveVideo(url, subDir, prefix)
	} else {
		localPath, err = s.downloadAndSaveImage(url, subDir, prefix)
	}

	if err != nil {
		return "", err
	}

	// 3. 将 URL 和本地路径的映射关系存入缓存
	s.urlMapping[url] = localPath
	s.log.Infow("Cached URL mapping", "url", url, "local_path", localPath)

	return localPath, nil
}

// downloadAndSaveImage 下载并保存图片
func (s *DataMigrationService) downloadAndSaveImage(imageURL, subDir, prefix string) (string, error) {
	if imageURL == "" {
		return "", fmt.Errorf("image URL is empty")
	}

	// 如果已经是本地路径，直接返回
	if strings.HasPrefix(imageURL, "/static/") || strings.HasPrefix(imageURL, "data/") {
		return imageURL, nil
	}

	// 从 URL 中提取文件扩展名（去掉查询参数）
	ext := s.extractFileExtension(imageURL)

	// 生成文件名
	timestamp := time.Now().Unix()
	filename := fmt.Sprintf("%s_%d%s", prefix, timestamp, ext)
	relativePath := filepath.Join(subDir, filename)
	fullPath := filepath.Join(s.storageRoot, relativePath)

	// 下载文件
	if err := s.downloadFile(imageURL, fullPath); err != nil {
		return "", fmt.Errorf("failed to download file: %w", err)
	}

	// 返回相对路径（用于存储到数据库）
	return relativePath, nil
}

// downloadAndSaveVideo 下载并保存视频
func (s *DataMigrationService) downloadAndSaveVideo(videoURL, subDir, prefix string) (string, error) {
	if videoURL == "" {
		return "", fmt.Errorf("video URL is empty")
	}

	// 如果已经是本地路径，直接返回
	if strings.HasPrefix(videoURL, "/static/") || strings.HasPrefix(videoURL, "data/") {
		return videoURL, nil
	}

	// 从 URL 中提取文件扩展名（去掉查询参数）
	ext := s.extractFileExtension(videoURL)
	if ext == "" || ext == ".jpeg" || ext == ".jpg" || ext == ".png" {
		ext = ".mp4" // 视频默认扩展名
	}

	// 生成文件名
	timestamp := time.Now().Unix()
	filename := fmt.Sprintf("%s_%d%s", prefix, timestamp, ext)
	relativePath := filepath.Join(subDir, filename)
	fullPath := filepath.Join(s.storageRoot, relativePath)

	// 下载文件
	if err := s.downloadFile(videoURL, fullPath); err != nil {
		return "", fmt.Errorf("failed to download file: %w", err)
	}

	// 返回相对路径（用于存储到数据库）
	return relativePath, nil
}

// extractFileExtension 从 URL 中提取文件扩展名（去掉查询参数）
func (s *DataMigrationService) extractFileExtension(url string) string {
	// 去掉查询参数
	if idx := strings.Index(url, "?"); idx != -1 {
		url = url[:idx]
	}
	
	// 去掉 fragment
	if idx := strings.Index(url, "#"); idx != -1 {
		url = url[:idx]
	}
	
	// 获取文件扩展名
	ext := filepath.Ext(url)
	if ext == "" {
		// 如果没有扩展名，默认返回 .jpg
		return ".jpg"
	}
	
	// 转换为小写
	ext = strings.ToLower(ext)
	
	// 验证扩展名是否合理（限制长度）
	if len(ext) > 10 {
		return ".jpg"
	}
	
	return ext
}

// downloadFile 下载文件到指定路径
func (s *DataMigrationService) downloadFile(url, filepath string) error {
	s.log.Infow("Start downloading file", "url", url, "filepath", filepath)

	// 创建 HTTP 请求
	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP status code error: %d", resp.StatusCode)
	}

	// 创建文件
	out, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	// 复制内容
	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	s.log.Infow("File downloaded successfully", "filepath", filepath, "size", written)
	return nil
}
