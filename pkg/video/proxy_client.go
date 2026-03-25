package video

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type ProxyClient struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

type ProxyRequest struct {
	Model       string `json:"model"`
	Prompt      string `json:"prompt"`
	ImageURL    string `json:"image_url"`
	Duration    int    `json:"duration,omitempty"`
	AspectRatio string `json:"aspect_ratio,omitempty"`
	Seed        int64  `json:"seed,omitempty"`
}

type ProxyResponse struct {
	TaskID   string `json:"task_id"`
	Status   string `json:"status"` // processing, completed, failed
	VideoURL string `json:"video_url,omitempty"`
	Error    string `json:"error,omitempty"`
}

func NewProxyClient(baseURL, apiKey string) *ProxyClient {
	return &ProxyClient{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 180 * time.Second,
		},
	}
}

func (c *ProxyClient) GenerateVideo(imageURL, prompt string, opts ...VideoOption) (*VideoResult, error) {
	options := &VideoOptions{
		Duration:    5,
		AspectRatio: "16:9",
	}

	for _, opt := range opts {
		opt(options)
	}

	reqBody := ProxyRequest{
		Model:       options.Model,
		Prompt:      prompt,
		ImageURL:    imageURL,
		Duration:    options.Duration,
		AspectRatio: options.AspectRatio,
		Seed:        options.Seed,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	endpoint := c.BaseURL + "/v1/video/generations"
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result ProxyResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if result.Error != "" {
		return nil, fmt.Errorf("proxy error: %s", result.Error)
	}

	videoResult := &VideoResult{
		TaskID:    result.TaskID,
		Status:    result.Status,
		Completed: result.Status == "completed",
		Error:     result.Error,
	}

	if result.VideoURL != "" {
		videoResult.VideoURL = result.VideoURL
		// If synchronous completion
		videoResult.Completed = true
	}

	return videoResult, nil
}

func (c *ProxyClient) GetTaskStatus(taskID string) (*VideoResult, error) {
	endpoint := c.BaseURL + "/v1/video/tasks/" + taskID
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var result ProxyResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	videoResult := &VideoResult{
		TaskID:    result.TaskID,
		Status:    result.Status,
		Completed: result.Status == "completed",
	}

	if result.Error != "" {
		videoResult.Error = result.Error
	}

	if result.VideoURL != "" {
		videoResult.VideoURL = result.VideoURL
	}

	return videoResult, nil
}
