package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

// Default free models available on OpenRouter.
var DefaultModels = []string{
	"meta-llama/llama-3.1-8b-instruct:free",
	"mistralai/mistral-7b-instruct:free",
	"google/gemma-2-9b-it:free",
	"qwen/qwen-2.5-7b-instruct:free",
}

// OpenRouterClient implements LLMClient using the OpenRouter API.
type OpenRouterClient struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
	Models     []string
	Semaphore  chan struct{}
	BatchDelay time.Duration
}

// NewOpenRouterClient creates a new OpenRouter client from environment variables.
func NewOpenRouterClient() *OpenRouterClient {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	baseURL := os.Getenv("OPENROUTER_BASE_URL")
	if baseURL == "" {
		baseURL = "https://openrouter.ai/api/v1"
	}

	defaultModel := os.Getenv("OPENROUTER_DEFAULT_MODEL")
	models := make([]string, len(DefaultModels))
	copy(models, DefaultModels)
	if defaultModel != "" {
		models[0] = defaultModel
	}

	batchDelay := 2 * time.Second
	if d := os.Getenv("OPENROUTER_BATCH_DELAY"); d != "" {
		if parsed, err := time.ParseDuration(d); err == nil {
			batchDelay = parsed
		}
	}

	return &OpenRouterClient{
		APIKey:     apiKey,
		BaseURL:    baseURL,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		Models:     models,
		Semaphore:  make(chan struct{}, 2),
		BatchDelay: batchDelay,
	}
}

type openRouterRequest struct {
	Model       string              `json:"model"`
	Messages    []openRouterMessage `json:"messages"`
	Temperature float64             `json:"temperature"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
}

type openRouterMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openRouterResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error,omitempty"`
}

func (c *OpenRouterClient) Name() string { return "openrouter" }

func (c *OpenRouterClient) Available(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/models", nil)
	if err != nil {
		return false
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (c *OpenRouterClient) Complete(ctx context.Context, req CompletionRequest) (string, error) {
	// Acquire semaphore slot
	select {
	case c.Semaphore <- struct{}{}:
		defer func() { <-c.Semaphore }()
	case <-ctx.Done():
		return "", ctx.Err()
	}

	// Try each model in rotation
	var lastErr error
	for _, model := range c.Models {
		resp, err := c.callModel(ctx, model, req)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		log.Printf("OpenRouter model %s failed: %v, rotating...", model, err)
	}
	return "", fmt.Errorf("all OpenRouter models failed: %w", lastErr)
}

func (c *OpenRouterClient) CompleteJSON(ctx context.Context, req CompletionRequest, dst interface{}) error {
	raw, err := c.Complete(ctx, req)
	if err != nil {
		return err
	}
	parsed, err := ParseJSONResponseRaw(raw)
	if err != nil {
		return fmt.Errorf("failed to parse JSON from OpenRouter: %w", err)
	}
	return json.Unmarshal([]byte(parsed), dst)
}

func (c *OpenRouterClient) callModel(ctx context.Context, model string, req CompletionRequest) (string, error) {
	messages := []openRouterMessage{
		{Role: "system", Content: req.SystemPrompt},
		{Role: "user", Content: req.UserPrompt},
	}

	temp := req.Temperature
	if temp == 0 {
		temp = 0.3
	}
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	body := openRouterRequest{
		Model:       model,
		Messages:    messages,
		Temperature: temp,
		MaxTokens:   maxTokens,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	httpReq.Header.Set("HTTP-Referer", "https://github.com/ByteJar/book-scraping-and-question-generation")
	httpReq.Header.Set("X-Title", "BookQGen")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Handle rate limiting
	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := resp.Header.Get("Retry-After")
		waitSec := 5
		if retryAfter != "" {
			if parsed, err := strconv.Atoi(retryAfter); err == nil && parsed <= 60 {
				waitSec = parsed
			}
		}

		select {
		case <-time.After(time.Duration(waitSec) * time.Second):
		case <-ctx.Done():
			return "", ctx.Err()
		}

		// Retry once
		retryResp, err := c.HTTPClient.Do(httpReq.Clone(ctx))
		if err != nil {
			return "", fmt.Errorf("retry after rate limit failed: %w", err)
		}
		defer retryResp.Body.Close()

		if retryResp.StatusCode == http.StatusTooManyRequests {
			return "", fmt.Errorf("rate limited on model %s after retry", model)
		}
		resp = retryResp
	}

	if resp.StatusCode >= 500 {
		return "", fmt.Errorf("server error from OpenRouter: %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var orResp openRouterResponse
	if err := json.Unmarshal(respBody, &orResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if orResp.Error != nil {
		return "", fmt.Errorf("OpenRouter error: %s (code %d)", orResp.Error.Message, orResp.Error.Code)
	}

	if len(orResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in OpenRouter response")
	}

	return orResp.Choices[0].Message.Content, nil
}
