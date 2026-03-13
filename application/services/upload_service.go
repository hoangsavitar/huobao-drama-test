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
	if err := os.MkdirAll(cfg.Storage.LocalPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &UploadService{
		storagePath: cfg.Storage.LocalPath,
		baseURL:     cfg.Storage.BaseURL,
		log:         log,
	}, nil
}

type UploadResult struct {
	URL       string
	LocalPath string
}

func (s *UploadService) UploadFile(file io.Reader, fileName, contentType string, category string) (*UploadResult, error) {
	categoryPath := filepath.Join(s.storagePath, category)
	if err := os.MkdirAll(categoryPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create category directory: %w", err)
	}

	ext := filepath.Ext(fileName)
	uniqueID := uuid.New().String()
	timestamp := time.Now().Format("20060102_150405")
	newFileName := fmt.Sprintf("%s_%s%s", timestamp, uniqueID, ext)
	filePath := filepath.Join(categoryPath, newFileName)

	dst, err := os.Create(filePath)
	if err != nil {
		s.log.Errorw("Failed to create file", "error", err, "path", filePath)
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		s.log.Errorw("Failed to write file", "error", err, "path", filePath)
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	fileURL := fmt.Sprintf("%s/%s/%s", s.baseURL, category, newFileName)
	localPath := fmt.Sprintf("%s/%s", category, newFileName)

	s.log.Infow("File uploaded successfully", "path", filePath, "url", fileURL, "local_path", localPath)
	return &UploadResult{
		URL:       fileURL,
		LocalPath: localPath,
	}, nil
}

func (s *UploadService) UploadCharacterImage(file io.Reader, fileName, contentType string) (*UploadResult, error) {
	return s.UploadFile(file, fileName, contentType, "characters")
}

func (s *UploadService) DeleteFile(fileURL string) error {
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

func (s *UploadService) extractRelativePathFromURL(fileURL string) string {
	if len(fileURL) <= len(s.baseURL) {
		return ""
	}
	return fileURL[len(s.baseURL)+1:]
}

func (s *UploadService) GetPresignedURL(objectName string, expiry time.Duration) (string, error) {
	return fmt.Sprintf("%s/%s", s.baseURL, objectName), nil
}
