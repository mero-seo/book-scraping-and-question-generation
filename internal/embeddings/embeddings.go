package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"strings"
)

const (
	defaultOllamaURL = "http://localhost:11434"
	defaultModel     = "nomic-embed-text"
)

// Embedder generates vector embeddings via the Ollama API.
type Embedder struct {
	OllamaURL string
	Model     string
}

// NewEmbedder creates an Embedder, reading OLLAMA_URL from the environment
// if set, otherwise falling back to the default localhost URL.
func NewEmbedder() *Embedder {
	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = defaultOllamaURL
	}
	return &Embedder{
		OllamaURL: ollamaURL,
		Model:     defaultModel,
	}
}

// embeddingRequest is the JSON body sent to POST /api/embeddings.
type embeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// embeddingResponse is the JSON response from POST /api/embeddings.
type embeddingResponse struct {
	Embedding []float64 `json:"embedding"`
}

// Embed sends text to the Ollama embeddings API and returns the resulting
// vector (768 dimensions for nomic-embed-text).
func (e *Embedder) Embed(ctx context.Context, text string) ([]float64, error) {
	reqBody := embeddingRequest{
		Model:  e.Model,
		Prompt: text,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embedding request: %w", err)
	}

	url := strings.TrimRight(e.OllamaURL, "/") + "/api/embeddings"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Ollama embeddings API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ollama embeddings API returned status %d", resp.StatusCode)
	}

	var result embeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode embedding response: %w", err)
	}

	if len(result.Embedding) == 0 {
		return nil, fmt.Errorf("Ollama returned empty embedding")
	}

	return result.Embedding, nil
}

// EmbedChapter embeds a full chapter by chunking the content (2000 tokens
// with 200 token overlap, sentence-boundary aware), embedding each chunk,
// and averaging the resulting vectors via mean pooling.
//
// If the content fits within a single chunk, it is embedded directly.
func (e *Embedder) EmbedChapter(ctx context.Context, content string) ([]float64, error) {
	chunks := ChunkText(content, 2000, 200)
	if len(chunks) == 0 {
		return nil, fmt.Errorf("failed to chunk chapter: no content")
	}

	if len(chunks) == 1 {
		return e.Embed(ctx, chunks[0])
	}

	var allVectors [][]float64
	for i, chunk := range chunks {
		vec, err := e.Embed(ctx, chunk)
		if err != nil {
			return nil, fmt.Errorf("failed to embed chunk %d/%d: %w", i+1, len(chunks), err)
		}
		allVectors = append(allVectors, vec)
	}

	return meanPool(allVectors), nil
}

// meanPool averages a set of vectors element-wise.
func meanPool(vectors [][]float64) []float64 {
	if len(vectors) == 0 {
		return nil
	}

	dim := len(vectors[0])
	result := make([]float64, dim)
	for _, vec := range vectors {
		for i := 0; i < dim && i < len(vec); i++ {
			result[i] += vec[i]
		}
	}

	n := float64(len(vectors))
	for i := range result {
		result[i] /= n
	}

	return result
}

// estimateTokens returns a rough token count for text.
// Approximation: word_count * 1.3.
func estimateTokens(text string) int {
	words := len(strings.Fields(text))
	return int(float64(words) * 1.3)
}

// ChunkText splits text into chunks of approximately maxTokens tokens each,
// with overlapTokens of overlap between consecutive chunks. Splitting is done
// at sentence boundaries (". " or ".\n") via greedy accumulation.
//
// If the entire text fits in one chunk, it is returned as a single element.
func ChunkText(text string, maxTokens, overlapTokens int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	// If text fits in one chunk, return it directly.
	if estimateTokens(text) <= maxTokens {
		return []string{text}
	}

	sentences := splitSentences(text)
	if len(sentences) == 0 {
		return nil
	}

	var chunks []string
	start := 0

	for start < len(sentences) {
		// Greedily accumulate sentences until we exceed maxTokens.
		tokenCount := 0
		end := start

		for end < len(sentences) {
			sentTokens := estimateTokens(sentences[end])
			if tokenCount+sentTokens > maxTokens && end > start {
				break
			}
			tokenCount += sentTokens
			end++
		}

		// Build the chunk from sentences [start, end).
		chunk := strings.TrimSpace(strings.Join(sentences[start:end], ""))
		if chunk != "" {
			chunks = append(chunks, chunk)
		}

		if end >= len(sentences) {
			break
		}

		// Backtrack by overlapTokens for the next chunk's start.
		overlapCount := 0
		nextStart := end
		for nextStart > start {
			sentTokens := estimateTokens(sentences[nextStart-1])
			if overlapCount+sentTokens > overlapTokens {
				break
			}
			overlapCount += sentTokens
			nextStart--
		}

		// Ensure forward progress: if backtracking didn't move past
		// the current end, start the next chunk at end.
		if nextStart <= start {
			nextStart = end
		}
		start = nextStart
	}

	return chunks
}

// splitSentences splits text into sentences. Each sentence retains its
// trailing delimiter (". " or ".\n") so that chunks can be reassembled
// without losing punctuation.
func splitSentences(text string) []string {
	var sentences []string
	remaining := text

	for len(remaining) > 0 {
		// Find the earliest sentence boundary.
		idx := -1

		dotSpace := strings.Index(remaining, ". ")
		dotNewline := strings.Index(remaining, ".\n")

		switch {
		case dotSpace >= 0 && dotNewline >= 0:
			if dotSpace <= dotNewline {
				idx = dotSpace
			} else {
				idx = dotNewline
			}
		case dotSpace >= 0:
			idx = dotSpace
		case dotNewline >= 0:
			idx = dotNewline
		}

		if idx < 0 {
			// No more sentence boundaries; take the rest.
			sentences = append(sentences, remaining)
			break
		}

		// Include the period and the delimiter character (space or newline).
		sentence := remaining[:idx+2]
		sentences = append(sentences, sentence)
		remaining = remaining[idx+2:]
	}

	return sentences
}

// CosineSimilarity computes the cosine similarity between two vectors.
// Returns 0 if the vectors have different lengths or if either is a zero vector.
func CosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

// Available checks whether the Ollama server is reachable by hitting
// GET /api/tags and returning true if it responds with HTTP 200.
func (e *Embedder) Available(ctx context.Context) bool {
	url := strings.TrimRight(e.OllamaURL, "/") + "/api/tags"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}
