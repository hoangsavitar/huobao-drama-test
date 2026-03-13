package services

import (
"fmt"
"io"
"os"
"path/filepath"
"time"

"github.com/drama-generator/backend/pkg/config"
"github.com/drama-generator/backend/pkg/logger"
"github.com/google/uuid"
)

type UploadService struct {
storagePath string
baseURL     string
log         *logger.Logger
}

func NewUploadService(cfg *config.Config, log *logger.Logger) (*UploadService, error) {
// Ensure storage directory exists
if err := os.MkdirAll(cfg.Storage.LocalPath, 0755); err != nil {
return nil, fmt.Errorf("failed to create storage directory: %w", err)
}

return &UploadService{
storagePath: cfg.Storage.LocalPath,
baseURL:     cfg.Storage.BaseURL,
log:         log,
}, nil
}

// UploadResult represents an upload result
type UploadResult struct {
URL       string // Full access URL
LocalPath string // Relative path (relative to storage root)
}

// UploadFile uploads a file to local storage
func (s *UploadService) UploadFile(file io.Reader, fileName, contentType string, category string) (*UploadResult, error) {
// Create category directory
categoryPath := filepath.Join(s.storagePath, category)
if err := os.MkdirAll(categoryPath, 0755); err != nil {
return nil, fmt.Errorf("failed to create category directory: %w", err)
}

// Generate unique file name
ext := filepath.Ext(fileName)
uniqueID := uuid.New().String()
timestamp := time.Now().Format("20060102_150405")
newFileName := fmt.Sprintf("%s_%s%s", timestamp, uniqueID, ext)
filePath := filepath.Join(categoryPath, newFileName)

// Create file
dst, err := os.Create(filePath)
if err != nil {
s.log.Errorw("Failed to create file", "error", err, "path", filePath)
return nil, fmt.Errorf("failed to create file: %w", err)
}
defer dst.Close()

// Write file
if _, err := io.Copy(dst, file); err != nil {
s.log.Errorw("Failed to write file", "error", err, "path", filePath)
return nil, fmt.Errorf("failed to write file: %w", err)
}

// Build access URL and relative path
fileURL := fmt.Sprintf("%s/%s/%s", s.baseURL, category, newFileName)
localPath := fmt.Sprintf("%s/%s", category, newFileName)

s.log.Infow("File uploaded successfully", "path", filePath, "url", fileURL, "local_path", localPath)
return &UploadResult{
URL:       fileURL,
LocalPath: localPath,
}, nil
}

// UploadCharacterImage uploads a character image
func (s *UploadService) UploadCharacterImage(file io.Reader, fileName, contentType string) (*UploadResult, error) {
return s.UploadFile(file, fileName, contentType, "characters")
}

// DeleteFile deletes a local file
func (s *UploadService) DeleteFile(fileURL string) error {
// Extract relative path from URL
// URL format: http://localhost:8080/static/characters/20060102_150405_uuid.jpg
relPath := s.extractRelativePathFromURL(fileURL)
if relPath == "" {
return fmt.Errorf("invalid file URL")
}

filePath := filepath.Join(s.storagePath, relPath)
err := os.Remove(filePath)
if err != nil {
s.log.Errorw("Failed to delete file", "error", err, "path", filePath)
return fmt.Errorf("failed to delete file: %w", err)
}

s.log.Infow("File deleted successfully", "path", filePath)
return nil
}

// extractRelativePathFromURL extracts relative path from URL
func (s *UploadService) extractRelativePathFromURL(fileURL string) string {
// Extract path after baseURL
// e.g.: http://localhost:8080/static/characters/xxx.jpg -> characters/xxx.jpg
if len(fileURL) <= len(s.baseURL) {
return ""
}
return fileURL[len(s.baseURL)+1:] // +1 for the '/'
}

// GetPresignedURL returns original URL directly (local storage needs no presigning)
func (s *UploadService) GetPresignedURL(objectName string, expiry time.Duration) (string, error) {
// Local storage accessed directly via static file server, no presigning needed
return fmt.Sprintf("%s/%s", s.baseURL, objectName), nil
}
