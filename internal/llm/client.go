package llm

import "context"

// LLMClient is the provider-agnostic interface for all LLM interactions.
type LLMClient interface {
	// Complete sends a prompt and returns the raw text completion.
	Complete(ctx context.Context, req CompletionRequest) (string, error)

	// CompleteJSON sends a prompt and parses the response as JSON into dst.
	CompleteJSON(ctx context.Context, req CompletionRequest, dst interface{}) error

	// Name returns the provider name for logging ("openrouter", "ollama").
	Name() string

	// Available returns true if the provider is reachable and ready.
	Available(ctx context.Context) bool
}

// CompletionRequest holds the parameters for an LLM completion call.
type CompletionRequest struct {
	SystemPrompt string
	UserPrompt   string
	Temperature  float64
	MaxTokens    int
}
