package image

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"google.golang.org/genai"
)

// VertexImageClient implements ImageClient using the Vertex AI Go SDK
// (google.golang.org/genai) with Application Default Credentials.
// Uses Vertex AI backend with "global" location (configurable via env).
type VertexImageClient struct {
	client *genai.Client
	ctx    context.Context
	model  string
}

func NewVertexImageClient(model string) (*VertexImageClient, error) {
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
		return nil, fmt.Errorf("vertex image: create client: %w", err)
	}
	return &VertexImageClient{
		client: client,
		ctx:    ctx,
		model:  model,
	}, nil
}

func (c *VertexImageClient) GenerateImage(prompt string, opts ...ImageOption) (*ImageResult, error) {
	options := &ImageOptions{
		Size: "1K",
	}
	for _, opt := range opts {
		opt(options)
	}

	modelName := c.model
	if options.Model != "" {
		modelName = options.Model
	}

	// Build parts: optional reference images + text prompt.
	var parts []*genai.Part

	for _, refImg := range options.ReferenceImages {
		var base64Data string
		var mimeType string
		var err error

		if strings.HasPrefix(refImg, "http://") || strings.HasPrefix(refImg, "https://") {
			base64Data, mimeType, err = downloadImageToBase64(refImg)
			if err != nil {
				continue
			}
		} else if strings.HasPrefix(refImg, "data:") {
			mimeType = "image/jpeg"
			data := []byte(refImg)
			for i := 0; i < len(data); i++ {
				if data[i] == ',' {
					base64Data = refImg[i+1:]
					if i > 11 {
						mimeTypeEnd := i
						for j := 5; j < i; j++ {
							if data[j] == ';' {
								mimeTypeEnd = j
								break
							}
						}
						mimeType = refImg[5:mimeTypeEnd]
					}
					break
				}
			}
		} else {
			base64Data = refImg
			mimeType = "image/jpeg"
		}

		if base64Data != "" {
			rawBytes, err := base64.StdEncoding.DecodeString(base64Data)
			if err != nil {
				continue
			}
			parts = append(parts, &genai.Part{
				InlineData: &genai.Blob{
					Data:     rawBytes,
					MIMEType: mimeType,
				},
			})
		}
	}

	// Add the text prompt last.
	parts = append(parts, &genai.Part{Text: prompt})

	contents := []*genai.Content{
		{
			Parts: parts,
			Role:  genai.RoleUser,
		},
	}

	resp, err := c.client.Models.GenerateContent(c.ctx, modelName, contents,
		&genai.GenerateContentConfig{
			ResponseModalities: []string{
				string(genai.ModalityText),
				string(genai.ModalityImage),
			},
		})
	if err != nil {
		return nil, fmt.Errorf("vertex image gen: %w", err)
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return nil, fmt.Errorf("no candidates in response")
	}

	for _, part := range resp.Candidates[0].Content.Parts {
		if part.InlineData != nil {
			b64Data := base64.StdEncoding.EncodeToString(part.InlineData.Data)
			dataURI := fmt.Sprintf("data:%s;base64,%s", part.InlineData.MIMEType, b64Data)
			return &ImageResult{
				Status:    "completed",
				ImageURL:  dataURI,
				Completed: true,
				Width:     1024,
				Height:    1024,
			}, nil
		}
	}

	return nil, fmt.Errorf("no image data in response")
}

func (c *VertexImageClient) GetTaskStatus(taskID string) (*ImageResult, error) {
	return nil, fmt.Errorf("not supported for Vertex (synchronous generation)")
}
