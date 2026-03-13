package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type OpenAIClient struct {
	BaseURL    string
	APIKey     string
	Model      string
	Endpoint   string
	HTTPClient *http.Client
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionRequest struct {
	Model               string        `json:"model"`
	Messages            []ChatMessage `json:"messages"`
	Temperature         float64       `json:"temperature,omitempty"`
	MaxTokens           *int          `json:"max_tokens,omitempty"`
	MaxCompletionTokens *int          `json:"max_completion_tokens,omitempty"`
	TopP                float64       `json:"top_p,omitempty"`
	Stream              bool          `json:"stream,omitempty"`
}

type ChatCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type ImageGenerationRequest struct {
	Model  string `json:"model,omitempty"`
	Prompt string `json:"prompt"`
	N      int    `json:"n,omitempty"`
	Size   string `json:"size,omitempty"`
}

type ImageGenerationResponse struct {
	Created int64 `json:"created"`
	Data    []struct {
		URL     string `json:"url"`
		B64JSON string `json:"b64_json"`
	} `json:"data"`
}

type ErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

func NewOpenAIClient(baseURL, apiKey, model, endpoint string) *OpenAIClient {
	if endpoint == "" {
		endpoint = "/v1/chat/completions"
	}

	return &OpenAIClient{
		BaseURL:  baseURL,
		APIKey:   apiKey,
		Model:    model,
		Endpoint: endpoint,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Minute,
		},
	}
}

func (c *OpenAIClient) ChatCompletion(messages []ChatMessage, options ...func(*ChatCompletionRequest)) (*ChatCompletionResponse, error) {
	req := &ChatCompletionRequest{
		Model:    c.Model,
		Messages: messages,
	}

	for _, option := range options {
		option(req)
	}

	return c.sendChatRequest(req)
}

func (c *OpenAIClient) sendChatRequest(req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	resp, err := c.doChatRequest(req)
	if err == nil {
		return resp, nil
	}

	if shouldRetryWithMaxCompletionTokens(err, req) {
		tokens := *req.MaxTokens
		retryReq := *req
		retryReq.MaxTokens = nil
		retryReq.MaxCompletionTokens = &tokens
		fmt.Printf("OpenAI: retrying with max_completion_tokens=%d\n", tokens)
		return c.doChatRequest(&retryReq)
	}

	return nil, err
}

func (c *OpenAIClient) doChatRequest(req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		fmt.Printf("OpenAI: Failed to marshal request: %v\n", err)
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.BaseURL + c.Endpoint

	// Request debug logs
	fmt.Printf("OpenAI: Sending request to: %s\n", url)
	fmt.Printf("OpenAI: BaseURL=%s, Endpoint=%s, Model=%s\n", c.BaseURL, c.Endpoint, c.Model)
	requestPreview := string(jsonData)
	if len(jsonData) > 300 {
		requestPreview = string(jsonData[:300]) + "..."
	}
	fmt.Printf("OpenAI: Request body: %s\n", requestPreview)

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("OpenAI: Failed to create request: %v\n", err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	fmt.Printf("OpenAI: Executing HTTP request...\n")
	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		fmt.Printf("OpenAI: HTTP request failed: %v\n", err)
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	fmt.Printf("OpenAI: Received response with status: %d\n", resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("OpenAI: Failed to read response body: %v\n", err)
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("OpenAI: API error (status %d): %s\n", resp.StatusCode, string(body))
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err != nil {
			return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
		}
		return nil, fmt.Errorf("API error: %s", errResp.Error.Message)
	}

	// Response debug logs
	bodyPreview := string(body)
	if len(body) > 500 {
		bodyPreview = string(body[:500]) + "..."
	}
	fmt.Printf("OpenAI: Response body: %s\n", bodyPreview)

	var chatResp ChatCompletionResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		errorPreview := string(body)
		if len(body) > 200 {
			errorPreview = string(body[:200])
		}
		fmt.Printf("OpenAI: Failed to parse response: %v\n", err)
		return nil, fmt.Errorf("failed to unmarshal response: %w, body preview: %s", err, errorPreview)
	}

	fmt.Printf("OpenAI: Successfully parsed response, choices count: %d\n", len(chatResp.Choices))

	if len(chatResp.Choices) == 0 {
		fmt.Printf("OpenAI: No choices in response\n")
		return nil, fmt.Errorf("no choices in response")
	}

	// Handle finish_reason such as content_filter.
	if len(chatResp.Choices) > 0 {
		finishReason := chatResp.Choices[0].FinishReason
		content := chatResp.Choices[0].Message.Content
		usage := chatResp.Usage

		fmt.Printf("OpenAI: finish_reason=%s, content_length=%d\n", finishReason, len(content))

		if finishReason == "content_filter" {
			return nil, fmt.Errorf("AI content was blocked by the safety filter. Possible reasons:\n1. The request triggered a safety policy\n2. The generated content was flagged as sensitive\n3. Suggestion: adjust the input or contact the API provider to tune the filter policy")
		}

		if usage.TotalTokens == 0 && finishReason != "stop" {
			return nil, fmt.Errorf("AI returned empty content (finish_reason: %s). Possible reasons:\n1. Content was filtered\n2. Token limit\n3. API error", finishReason)
		}
	}

	return &chatResp, nil
}

func WithTemperature(temp float64) func(*ChatCompletionRequest) {
	return func(req *ChatCompletionRequest) {
		req.Temperature = temp
	}
}

func WithMaxTokens(tokens int) func(*ChatCompletionRequest) {
	return func(req *ChatCompletionRequest) {
		req.MaxTokens = &tokens
	}
}

func WithTopP(topP float64) func(*ChatCompletionRequest) {
	return func(req *ChatCompletionRequest) {
		req.TopP = topP
	}
}

func (c *OpenAIClient) GenerateText(prompt string, systemPrompt string, options ...func(*ChatCompletionRequest)) (string, error) {
	messages := []ChatMessage{}

	if systemPrompt != "" {
		messages = append(messages, ChatMessage{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	messages = append(messages, ChatMessage{
		Role:    "user",
		Content: prompt,
	})

	resp, err := c.ChatCompletion(messages, options...)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	return resp.Choices[0].Message.Content, nil
}

func (c *OpenAIClient) GenerateImage(prompt string, size string, n int) ([]string, error) {
	// Image generation endpoint is usually /v1/images/generations.
	imageEndpoint := "/v1/images/generations"

	url := c.BaseURL + imageEndpoint

	reqBody := ImageGenerationRequest{
		Prompt: prompt,
		N:      n,
		Size:   size,
		Model:  c.Model,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
			return nil, fmt.Errorf("API error: %s", errResp.Error.Message)
		}
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var imgResp ImageGenerationResponse
	if err := json.Unmarshal(body, &imgResp); err != nil {
		return nil, err
	}

	var urls []string
	for _, data := range imgResp.Data {
		if data.URL != "" {
			urls = append(urls, data.URL)
		} else if data.B64JSON != "" {
			urls = append(urls, "data:image/png;base64,"+data.B64JSON)
		}
	}

	return urls, nil
}

func (c *OpenAIClient) TestConnection() error {
	fmt.Printf("OpenAI: TestConnection called with BaseURL=%s, Endpoint=%s, Model=%s\n", c.BaseURL, c.Endpoint, c.Model)

	messages := []ChatMessage{
		{
			Role:    "user",
			Content: "Hello",
		},
	}

	_, err := c.ChatCompletion(messages, WithMaxTokens(50))
	if err != nil {
		fmt.Printf("OpenAI: TestConnection failed: %v\n", err)
	} else {
		fmt.Printf("OpenAI: TestConnection succeeded\n")
	}
	return err
}

func shouldRetryWithMaxCompletionTokens(err error, req *ChatCompletionRequest) bool {
	if err == nil || req == nil || req.MaxTokens == nil || req.MaxCompletionTokens != nil {
		return false
	}

	msg := err.Error()
	if strings.Contains(msg, "Unsupported parameter: 'max_tokens'") {
		return true
	}
	if strings.Contains(msg, "max_tokens is not supported") {
		return true
	}
	if strings.Contains(msg, "max_completion_tokens") {
		return true
	}
	return false
}
