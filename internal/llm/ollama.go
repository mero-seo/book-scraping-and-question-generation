package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// OllamaClient implements LLMClient using a local Ollama instance.
type OllamaClient struct {
	BaseURL    string
	Model      string
	HTTPClient *http.Client
}

// NewOllamaClient creates a new Ollama client from environment variables.
func NewOllamaClient() *OllamaClient {
	baseURL := os.Getenv("OLLAMA_URL")
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	model := os.Getenv("OLLAMA_MODEL")
	if model == "" {
		model = "llama3.2:3b"
	}
	return &OllamaClient{
		BaseURL:    baseURL,
		Model:      model,
		HTTPClient: &http.Client{},
	}
}

type ollamaChatRequest struct {
	Model    string           `json:"model"`
	Messages []ollamaMessage  `json:"messages"`
	Stream   bool             `json:"stream"`
	Options  *ollamaOptions   `json:"options,omitempty"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaOptions struct {
	Temperature float64 `json:"temperature"`
	NumPredict  int     `json:"num_predict,omitempty"`
}

type ollamaChatResponse struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
	Error string `json:"error,omitempty"`
}

func (c *OllamaClient) Name() string { return "ollama" }

func (c *OllamaClient) Available(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/api/tags", nil)
	if err != nil {
		return false
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (c *OllamaClient) Complete(ctx context.Context, req CompletionRequest) (string, error) {
	messages := []ollamaMessage{
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

	body := ollamaChatRequest{
		Model:    c.Model,
		Messages: messages,
		Stream:   false,
		Options: &ollamaOptions{
			Temperature: temp,
			NumPredict:  maxTokens,
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("failed to marshal Ollama request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/chat", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create Ollama request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("Ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read Ollama response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Ollama returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp ollamaChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("failed to parse Ollama response: %w", err)
	}

	if chatResp.Error != "" {
		return "", fmt.Errorf("Ollama error: %s", chatResp.Error)
	}

	return chatResp.Message.Content, nil
}

func (c *OllamaClient) CompleteJSON(ctx context.Context, req CompletionRequest, dst interface{}) error {
	raw, err := c.Complete(ctx, req)
	if err != nil {
		return err
	}
	parsed, err := ParseJSONResponseRaw(raw)
	if err != nil {
		return fmt.Errorf("failed to parse JSON from Ollama: %w", err)
	}
	return json.Unmarshal([]byte(parsed), dst)
}
