package services

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/drama-generator/backend/infrastructure/external/ffmpeg"
	"github.com/drama-generator/backend/pkg/logger"
)

type AudioExtractionService struct {
	ffmpeg *ffmpeg.FFmpeg
	log    *logger.Logger
}

func NewAudioExtractionService(log *logger.Logger) *AudioExtractionService {
	return &AudioExtractionService{
		ffmpeg: ffmpeg.NewFFmpeg(log),
		log:    log,
	}
}

type ExtractAudioRequest struct {
	VideoURL string `json:"video_url" binding:"required"`
}

type ExtractAudioResponse struct {
	AudioURL string  `json:"audio_url"`
	Duration float64 `json:"duration"`
}

// ExtractAudio extracts audio from a video URL and returns the audio file URL
func (s *AudioExtractionService) ExtractAudio(videoURL string, dataDir string) (*ExtractAudioResponse, error) {
	s.log.Infow("Starting audio extraction", "video_url", videoURL)

	// Generate output filename
	timestamp := time.Now().Unix()
	audioFileName := fmt.Sprintf("audio_%d.aac", timestamp)
	audioOutputPath := filepath.Join(dataDir, "audios", audioFileName)

	// Extract audio
	extractedPath, err := s.ffmpeg.ExtractAudio(videoURL, audioOutputPath)
	if err != nil {
		s.log.Errorw("Failed to extract audio", "error", err, "video_url", videoURL)
		return nil, fmt.Errorf("failed to extract audio: %w", err)
	}

	// Get audio duration (using the extracted local file path)
	duration, err := s.ffmpeg.GetVideoDuration(extractedPath)
	if err != nil {
		s.log.Errorw("Failed to get audio duration", "error", err, "path", extractedPath)
		return nil, fmt.Errorf("failed to get audio duration: %w", err)
	}

	if duration <= 0 {
		s.log.Errorw("Invalid audio duration", "duration", duration, "path", extractedPath)
		return nil, fmt.Errorf("invalid audio duration: %.2f", duration)
	}

	// Build audio URL (relative to data directory)
	audioURL := fmt.Sprintf("/data/audios/%s", audioFileName)

	s.log.Infow("Audio extraction completed",
		"video_url", videoURL,
		"audio_url", audioURL,
		"duration", duration,
		"local_path", extractedPath)

	return &ExtractAudioResponse{
		AudioURL: audioURL,
		Duration: duration,
	}, nil
}

// BatchExtractAudio batch extracts audio
func (s *AudioExtractionService) BatchExtractAudio(videoURLs []string, dataDir string) ([]*ExtractAudioResponse, error) {
	s.log.Infow("Starting batch audio extraction", "count", len(videoURLs))

	results := make([]*ExtractAudioResponse, 0, len(videoURLs))

	for i, videoURL := range videoURLs {
		s.log.Infow("Extracting audio", "index", i+1, "total", len(videoURLs), "video_url", videoURL)

		result, err := s.ExtractAudio(videoURL, dataDir)
		if err != nil {
			s.log.Errorw("Failed to extract audio in batch", "index", i, "video_url", videoURL, "error", err)
			// Continue processing other videos, but log the error
			return nil, fmt.Errorf("failed to extract audio at index %d: %w", i, err)
		}

		results = append(results, result)
	}

	s.log.Infow("Batch audio extraction completed", "successful_count", len(results))
	return results, nil
}
