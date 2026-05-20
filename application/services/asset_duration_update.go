package services

import (
	"fmt"

	models "github.com/drama-generator/backend/domain/models"
	"github.com/drama-generator/backend/infrastructure/storage"
)

// UpdateAssetDurationFromFile probes and updates video asset duration from a local file
func (s *AssetService) UpdateAssetDurationFromFile(assetID uint, localFilePath string) error {
	var asset models.Asset
	if err := s.db.Where("id = ?", assetID).First(&asset).Error; err != nil {
		return fmt.Errorf("asset not found")
	}

	if asset.Type != models.AssetTypeVideo {
		return fmt.Errorf("asset is not a video")
	}

	if s.ffmpeg == nil {
		return fmt.Errorf("ffmpeg not available")
	}

	duration, err := s.ffmpeg.GetVideoDuration(localFilePath)
	if err != nil {
		return fmt.Errorf("failed to probe video duration: %w", err)
	}

	durationInt := int(duration + 0.5)
	if err := s.db.Model(&asset).Update("duration", durationInt).Error; err != nil {
		return fmt.Errorf("failed to update duration: %w", err)
	}

	s.log.Infow("Updated asset duration from file",
		"asset_id", assetID,
		"duration", durationInt,
		"file", localFilePath)

	return nil
}

// UpdateAssetDurationFromURL downloads video and probes its duration
func (s *AssetService) UpdateAssetDurationFromURL(assetID uint, localStorage *storage.LocalStorage) error {
	var asset models.Asset
	if err := s.db.Where("id = ?", assetID).First(&asset).Error; err != nil {
		return fmt.Errorf("asset not found")
	}

	if asset.Type != models.AssetTypeVideo {
		return fmt.Errorf("asset is not a video")
	}

	if localStorage == nil {
		return fmt.Errorf("local storage not available")
	}

	// Download video to local storage
	localPath, err := localStorage.DownloadFromURL(asset.URL, "videos")
	if err != nil {
		return fmt.Errorf("failed to download video: %w", err)
	}

	// Probe duration
	return s.UpdateAssetDurationFromFile(assetID, localPath)
}
