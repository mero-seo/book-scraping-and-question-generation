package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// FallbackClient wraps a primary and fallback LLMClient.
// If the primary fails or times out, it transparently retries with the fallback.
type FallbackClient struct {
	Primary  LLMClient
	Fallback LLMClient
	Timeout  time.Duration
}

// NewFallbackClient creates a fallback client with OpenRouter as primary and Ollama as fallback.
func NewFallbackClient() *FallbackClient {
	return &FallbackClient{
		Primary:  NewOpenRouterClient(),
		Fallback: NewOllamaClient(),
		Timeout:  10 * time.Second,
	}
}

func (f *FallbackClient) Name() string {
	return fmt.Sprintf("fallback(%s->%s)", f.Primary.Name(), f.Fallback.Name())
}

func (f *FallbackClient) Available(ctx context.Context) bool {
	return f.Primary.Available(ctx) || f.Fallback.Available(ctx)
}

func (f *FallbackClient) Complete(ctx context.Context, req CompletionRequest) (string, error) {
	primaryCtx, cancel := context.WithTimeout(ctx, f.Timeout)
	defer cancel()

	resp, err := f.Primary.Complete(primaryCtx, req)
	if err == nil {
		return resp, nil
	}

	log.Printf("LLM primary (%s) failed: %v, falling back to %s",
		f.Primary.Name(), err, f.Fallback.Name())

	return f.Fallback.Complete(ctx, req)
}

func (f *FallbackClient) CompleteJSON(ctx context.Context, req CompletionRequest, dst interface{}) error {
	raw, err := f.Complete(ctx, req)
	if err != nil {
		return err
	}
	parsed, err := ParseJSONResponseRaw(raw)
	if err != nil {
		return fmt.Errorf("failed to parse JSON from fallback chain: %w", err)
	}
	return json.Unmarshal([]byte(parsed), dst)
}
