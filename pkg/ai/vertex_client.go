package ai

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/genai"
)

// VertexClient implements AIClient using the Vertex AI Go SDK
// (google.golang.org/genai) with Application Default Credentials.
// Uses Vertex AI backend with "global" location (configurable via env).
type VertexClient struct {
	client *genai.Client
	ctx    context.Context
	model  string
}

func NewVertexClient(model string) (*VertexClient, error) {
	ctx := context.Background()

	project := os.Getenv("GOOGLE_CLOUD_PROJECT")
	location := os.Getenv("GOOGLE_CLOUD_LOCATION")
	if location == "" {
		location = "global"
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  project,
		Location: location,
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		return nil, fmt.Errorf("vertex: create client: %w", err)
	}
	return &VertexClient{
		client: client,
		ctx:    ctx,
		model:  model,
	}, nil
}

func (c *VertexClient) GenerateText(prompt string, systemPrompt string, options ...func(*ChatCompletionRequest)) (string, error) {
	optReq := &ChatCompletionRequest{}
	for _, opt := range options {
		opt(optReq)
	}

	// Build generation config from optional params.
	var config *genai.GenerateContentConfig
	if optReq.MaxTokens != nil || optReq.Temperature != 0 || optReq.TopP != 0 {
		config = &genai.GenerateContentConfig{}
		if optReq.MaxTokens != nil && *optReq.MaxTokens > 0 {
			config.MaxOutputTokens = int32(*optReq.MaxTokens)
		}
		if optReq.Temperature != 0 {
			config.Temperature = genai.Ptr(float32(optReq.Temperature))
		}
		if optReq.TopP != 0 {
			config.TopP = genai.Ptr(float32(optReq.TopP))
		}
	}

	// Build contents with optional system instruction.
	if systemPrompt != "" {
		if config == nil {
			config = &genai.GenerateContentConfig{}
		}
		config.SystemInstruction = &genai.Content{
			Parts: []*genai.Part{
				{Text: systemPrompt},
			},
		}
	}

	result, err := c.client.Models.GenerateContent(c.ctx, c.model, genai.Text(prompt), config)
	if err != nil {
		return "", fmt.Errorf("vertex generate text: %w", err)
	}
	return result.Text(), nil
}

func (c *VertexClient) GenerateImage(prompt string, size string, n int) ([]string, error) {
	return nil, fmt.Errorf("GenerateImage not implemented — use image generation service")
}

func (c *VertexClient) TestConnection() error {
	_, err := c.GenerateText("Hello", "", WithMaxTokens(50))
	return err
}
