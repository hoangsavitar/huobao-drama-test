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
urlMapping  map[string]string // original URL -> local path mapping
}

func NewDataMigrationService(db *gorm.DB, log *logger.Logger) *DataMigrationService {
return &DataMigrationService{
db:          db,
log:         log,
storageRoot: "data/storage",
urlMapping:  make(map[string]string),
}
}

// MigrateLocalPaths migrates data with empty local_path across all tables
func (s *DataMigrationService) MigrateLocalPaths() error {
s.log.Info("Starting data cleanup: migrating data with empty local_path")
startTime := time.Now()

// Ensure storage directories exist
if err := s.ensureStorageDirectories(); err != nil {
return fmt.Errorf("failed to create storage directories: %w", err)
}

// Migrate data from each table (in specified order)
stats := &MigrationStats{}

// 1. Migrate assets table
if err := s.migrateAssets(stats); err != nil {
s.log.Errorw("failed to migrate assets data", "error", err)
}

// 2. Migrate character_libraries table
if err := s.migrateCharacterLibraries(stats); err != nil {
s.log.Errorw("failed to migrate character_libraries data", "error", err)
}

// 3. Migrate characters table
if err := s.migrateCharacters(stats); err != nil {
s.log.Errorw("failed to migrate characters data", "error", err)
}

// 4. Migrate image_generations table
if err := s.migrateImageGenerations(stats); err != nil {
s.log.Errorw("failed to migrate image_generations data", "error", err)
}

// 5. Migrate scenes table
if err := s.migrateScenes(stats); err != nil {
s.log.Errorw("failed to migrate scenes data", "error", err)
}

// 6. Migrate video_generations table
if err := s.migrateVideoGenerations(stats); err != nil {
s.log.Errorw("failed to migrate video_generations data", "error", err)
}

duration := time.Since(startTime)
s.log.Infow("Data cleanup completed",
"total_duration", duration.String(),
"url_mapping_cache_count", len(s.urlMapping),
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

// MigrationStats migration statistics
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

// ensureStorageDirectories ensures storage directories exist
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

s.log.Infow("Storage directories created successfully", "root", s.storageRoot)
return nil
}

// migrateAssets migrates assets table data
func (s *DataMigrationService) migrateAssets(stats *MigrationStats) error {
s.log.Info("Starting assets data migration...")

var assets []models.Asset
// Query assets with empty local_path but non-empty url
if err := s.db.Where("(local_path IS NULL OR local_path = '') AND url IS NOT NULL AND url != ''").Find(&assets).Error; err != nil {
return fmt.Errorf("failed to query assets data: %w", err)
}

s.log.Infow("Found assets to migrate", "count", len(assets))

for _, asset := range assets {
s.log.Infow("Processing asset", "id", asset.ID, "name", asset.Name, "type", asset.Type, "url", asset.URL)

// Select storage directory based on type
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

// Update local_path
if err := s.db.Model(&asset).Update("local_path", localPath).Error; err != nil {
s.log.Errorw("Failed to update asset local_path", "asset_id", asset.ID, "error", err)
stats.AssetsFailed++
continue
}

s.log.Infow("Asset migration succeeded", "asset_id", asset.ID, "local_path", localPath)
stats.AssetsSuccess++
}

return nil
}

// migrateCharacterLibraries migrates character_libraries table data
func (s *DataMigrationService) migrateCharacterLibraries(stats *MigrationStats) error {
s.log.Info("Starting character_libraries data migration...")

var charLibs []models.CharacterLibrary
// Query character libraries with empty local_path but non-empty image_url
if err := s.db.Where("(local_path IS NULL OR local_path = '') AND image_url IS NOT NULL AND image_url != ''").Find(&charLibs).Error; err != nil {
return fmt.Errorf("failed to query character_libraries data: %w", err)
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

// Update local_path
if err := s.db.Model(&charLib).Update("local_path", localPath).Error; err != nil {
s.log.Errorw("Failed to update character_library local_path", "charlib_id", charLib.ID, "error", err)
stats.CharacterLibrariesFailed++
continue
}

s.log.Infow("Character_library migration succeeded", "charlib_id", charLib.ID, "local_path", localPath)
stats.CharacterLibrariesSuccess++
}

return nil
}

// migrateImageGenerations migrates image_generations table data
func (s *DataMigrationService) migrateImageGenerations(stats *MigrationStats) error {
s.log.Info("Starting image_generations data migration...")

var imageGens []models.ImageGeneration
// Query image generations with empty local_path but non-empty image_url
if err := s.db.Where("(local_path IS NULL OR local_path = '') AND image_url IS NOT NULL AND image_url != ''").Find(&imageGens).Error; err != nil {
return fmt.Errorf("failed to query image_generations data: %w", err)
}

s.log.Infow("Found image_generations to migrate", "count", len(imageGens))

for _, imageGen := range imageGens {
if imageGen.ImageURL == nil {
continue
}

imageTypeStr := string(imageGen.ImageType)
s.log.Infow("Processing image_generation", "id", imageGen.ID, "image_type", imageTypeStr, "image_url", *imageGen.ImageURL)

// Select storage directory based on image type
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

// Update local_path
if err := s.db.Model(&imageGen).Update("local_path", localPath).Error; err != nil {
s.log.Errorw("Failed to update image_generation local_path", "imggen_id", imageGen.ID, "error", err)
stats.ImageGenerationsFailed++
continue
}

s.log.Infow("Image_generation migration succeeded", "imggen_id", imageGen.ID, "local_path", localPath)
stats.ImageGenerationsSuccess++
}

return nil
}

// migrateScenes migrates scene data
func (s *DataMigrationService) migrateScenes(stats *MigrationStats) error {
s.log.Info("Starting scene data migration...")

var scenes []models.Scene
// Query scenes with empty local_path but non-empty image_url
if err := s.db.Where("(local_path IS NULL OR local_path = '') AND image_url IS NOT NULL AND image_url != ''").Find(&scenes).Error; err != nil {
return fmt.Errorf("failed to query scene data: %w", err)
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

// Update local_path
if err := s.db.Model(&scene).Update("local_path", localPath).Error; err != nil {
s.log.Errorw("Failed to update scene local_path", "scene_id", scene.ID, "error", err)
stats.ScenesFailed++
continue
}

s.log.Infow("Scene migration succeeded", "scene_id", scene.ID, "local_path", localPath)
stats.ScenesSuccess++
}

return nil
}

// migrateCharacters migrates character data
func (s *DataMigrationService) migrateCharacters(stats *MigrationStats) error {
s.log.Info("Starting character data migration...")

var characters []models.Character
// Query characters with empty local_path but non-empty image_url
if err := s.db.Where("(local_path IS NULL OR local_path = '') AND image_url IS NOT NULL AND image_url != ''").Find(&characters).Error; err != nil {
return fmt.Errorf("failed to query character data: %w", err)
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

// Update local_path
if err := s.db.Model(&character).Update("local_path", localPath).Error; err != nil {
s.log.Errorw("Failed to update character local_path", "character_id", character.ID, "error", err)
stats.CharactersFailed++
continue
}

s.log.Infow("Character migration succeeded", "character_id", character.ID, "local_path", localPath)
stats.CharactersSuccess++
}

return nil
}

// migrateVideoGenerations migrates video generation data
func (s *DataMigrationService) migrateVideoGenerations(stats *MigrationStats) error {
s.log.Info("Starting video generation data migration...")

var videoGens []models.VideoGeneration
// Query videos with empty local_path but non-empty video_url
if err := s.db.Where("(local_path IS NULL OR local_path = '') AND video_url IS NOT NULL AND video_url != ''").Find(&videoGens).Error; err != nil {
return fmt.Errorf("failed to query video generation data: %w", err)
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

// Update local_path
if err := s.db.Model(&videoGen).Update("local_path", localPath).Error; err != nil {
s.log.Errorw("Failed to update video local_path", "video_gen_id", videoGen.ID, "error", err)
stats.VideosFailed++
continue
}

s.log.Infow("Video migration succeeded", "video_gen_id", videoGen.ID, "local_path", localPath)
stats.VideosSuccess++
}

return nil
}

// downloadOrGetCached downloads a file or retrieves local path from cache
func (s *DataMigrationService) downloadOrGetCached(url, subDir, prefix string) (string, error) {
// 1. Check URL mapping cache
if localPath, exists := s.urlMapping[url]; exists {
s.log.Infow("Using cached local path", "url", url, "local_path", localPath)
return localPath, nil
}

// 2. If not in cache, download the file
var localPath string
var err error

// Determine if image or video based on subdirectory
if subDir == "videos" {
localPath, err = s.downloadAndSaveVideo(url, subDir, prefix)
} else {
localPath, err = s.downloadAndSaveImage(url, subDir, prefix)
}

if err != nil {
return "", err
}

// 3. Store URL to local path mapping in cache
s.urlMapping[url] = localPath
s.log.Infow("URL mapping cached", "url", url, "local_path", localPath)

return localPath, nil
}

// downloadAndSaveImage downloads and saves an image
func (s *DataMigrationService) downloadAndSaveImage(imageURL, subDir, prefix string) (string, error) {
if imageURL == "" {
return "", fmt.Errorf("image URL is empty")
}

// If already a local path, return directly
if strings.HasPrefix(imageURL, "/static/") || strings.HasPrefix(imageURL, "data/") {
return imageURL, nil
}

// Extract file extension from URL (removing query parameters)
ext := s.extractFileExtension(imageURL)

// Generate filename
timestamp := time.Now().Unix()
filename := fmt.Sprintf("%s_%d%s", prefix, timestamp, ext)
relativePath := filepath.Join(subDir, filename)
fullPath := filepath.Join(s.storageRoot, relativePath)

// Download file
if err := s.downloadFile(imageURL, fullPath); err != nil {
return "", fmt.Errorf("failed to download file: %w", err)
}

// Return relative path (for database storage)
return relativePath, nil
}

// downloadAndSaveVideo downloads and saves a video
func (s *DataMigrationService) downloadAndSaveVideo(videoURL, subDir, prefix string) (string, error) {
if videoURL == "" {
return "", fmt.Errorf("video URL is empty")
}

// If already a local path, return directly
if strings.HasPrefix(videoURL, "/static/") || strings.HasPrefix(videoURL, "data/") {
return videoURL, nil
}

// Extract file extension from URL (removing query parameters)
ext := s.extractFileExtension(videoURL)
if ext == "" || ext == ".jpeg" || ext == ".jpg" || ext == ".png" {
ext = ".mp4" // default video extension
}

// Generate filename
timestamp := time.Now().Unix()
filename := fmt.Sprintf("%s_%d%s", prefix, timestamp, ext)
relativePath := filepath.Join(subDir, filename)
fullPath := filepath.Join(s.storageRoot, relativePath)

// Download file
if err := s.downloadFile(videoURL, fullPath); err != nil {
return "", fmt.Errorf("failed to download file: %w", err)
}

// Return relative path (for database storage)
return relativePath, nil
}

// extractFileExtension extracts file extension from URL (removing query parameters)
func (s *DataMigrationService) extractFileExtension(url string) string {
// Remove query parameters
if idx := strings.Index(url, "?"); idx != -1 {
url = url[:idx]
}

// Remove fragment
if idx := strings.Index(url, "#"); idx != -1 {
url = url[:idx]
}

// Get file extension
ext := filepath.Ext(url)
if ext == "" {
// If no extension, default to .jpg
return ".jpg"
}

// Convert to lowercase
ext = strings.ToLower(ext)

// Validate extension is reasonable (limit length)
if len(ext) > 10 {
return ".jpg"
}

return ext
}

// downloadFile downloads a file to the specified path
func (s *DataMigrationService) downloadFile(url, filepath string) error {
s.log.Infow("Starting file download", "url", url, "filepath", filepath)

// Create HTTP request
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

// Create file
out, err := os.Create(filepath)
if err != nil {
return fmt.Errorf("failed to create file: %w", err)
}
defer out.Close()

// Copy content
written, err := io.Copy(out, resp.Body)
if err != nil {
return fmt.Errorf("failed to write file: %w", err)
}

s.log.Infow("File download succeeded", "filepath", filepath, "size", written)
return nil
}
