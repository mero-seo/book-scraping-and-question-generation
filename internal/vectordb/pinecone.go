package vectordb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// PineconeClient handles embedding generation and vector storage/search via Pinecone.
// Uses Pinecone's inference API for embeddings and their vector DB for storage.
type PineconeClient struct {
	APIKey    string
	IndexHost string // e.g. "https://book-chapters-abc123.svc.aped-1234.pinecone.io"
	Client    *http.Client
	Model     string // embedding model, default "multilingual-e5-large"
}

// NewPineconeClient creates a client from environment variables.
func NewPineconeClient() (*PineconeClient, error) {
	apiKey := os.Getenv("PINECONE_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("PINECONE_API_KEY is required")
	}

	indexHost := os.Getenv("PINECONE_INDEX_HOST")
	if indexHost == "" {
		return nil, fmt.Errorf("PINECONE_INDEX_HOST is required")
	}

	// Ensure index host has https:// prefix
	if !strings.HasPrefix(indexHost, "https://") {
		indexHost = "https://" + indexHost
	}

	return &PineconeClient{
		APIKey:    apiKey,
		IndexHost: strings.TrimRight(indexHost, "/"),
		Client:    &http.Client{},
		Model:     "multilingual-e5-large",
	}, nil
}

// --- Embedding Generation ---

type embedRequest struct {
	Model      string        `json:"model"`
	Inputs     []embedInput  `json:"inputs"`
	Parameters embedParams   `json:"parameters"`
}

type embedInput struct {
	Text string `json:"text"`
}

type embedParams struct {
	InputType  string `json:"input_type"`
	Truncate   string `json:"truncate"`
}

type embedResponse struct {
	Data  []embedData `json:"data"`
	Model string      `json:"model"`
}

type embedData struct {
	Values []float64 `json:"values"`
}

// Embed generates a vector embedding for the given text using Pinecone's inference API.
// Returns a 1024-dimensional vector (for multilingual-e5-large).
func (p *PineconeClient) Embed(ctx context.Context, text string) ([]float64, error) {
	return p.embedWithType(ctx, text, "passage")
}

// EmbedQuery generates an embedding optimized for search queries.
func (p *PineconeClient) EmbedQuery(ctx context.Context, text string) ([]float64, error) {
	return p.embedWithType(ctx, text, "query")
}

func (p *PineconeClient) embedWithType(ctx context.Context, text, inputType string) ([]float64, error) {
	body := embedRequest{
		Model:  p.Model,
		Inputs: []embedInput{{Text: text}},
		Parameters: embedParams{
			InputType: inputType,
			Truncate:  "END",
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embed request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.pinecone.io/embed", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create embed request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Api-Key", p.APIKey)
	req.Header.Set("X-Pinecone-API-Version", "2025-01")

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embed request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Pinecone embed API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode embed response: %w", err)
	}

	if len(result.Data) == 0 || len(result.Data[0].Values) == 0 {
		return nil, fmt.Errorf("Pinecone returned empty embedding")
	}

	return result.Data[0].Values, nil
}

// EmbedChapter embeds chapter content. For long content, it chunks and averages.
func (p *PineconeClient) EmbedChapter(ctx context.Context, content string) ([]float64, error) {
	// Pinecone's multilingual-e5-large handles up to 512 tokens well.
	// For longer content, chunk and average.
	chunks := chunkText(content, 1500)
	if len(chunks) == 0 {
		return nil, fmt.Errorf("empty content")
	}

	if len(chunks) == 1 {
		return p.Embed(ctx, chunks[0])
	}

	var allVecs [][]float64
	for i, chunk := range chunks {
		vec, err := p.Embed(ctx, chunk)
		if err != nil {
			return nil, fmt.Errorf("failed to embed chunk %d/%d: %w", i+1, len(chunks), err)
		}
		allVecs = append(allVecs, vec)
	}

	return meanPool(allVecs), nil
}

// --- Vector Storage ---

type upsertRequest struct {
	Vectors []vector `json:"vectors"`
}

type vector struct {
	ID       string            `json:"id"`
	Values   []float64         `json:"values"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// UpsertChapter stores a chapter's embedding in Pinecone.
func (p *PineconeClient) UpsertChapter(ctx context.Context, chapterID, bookID, chapterTitle string, embedding []float64) error {
	body := upsertRequest{
		Vectors: []vector{
			{
				ID:     chapterID,
				Values: embedding,
				Metadata: map[string]string{
					"book_id":       bookID,
					"chapter_title": chapterTitle,
				},
			},
		},
	}

	return p.doIndexRequest(ctx, http.MethodPost, "/vectors/upsert", body, nil)
}

// --- Vector Search ---

type queryRequest struct {
	Vector          []float64         `json:"vector"`
	TopK            int               `json:"topK"`
	Filter          map[string]interface{} `json:"filter,omitempty"`
	IncludeMetadata bool              `json:"includeMetadata"`
}

type queryResponse struct {
	Matches []matchResult `json:"matches"`
}

// MatchResult holds a single search result from Pinecone.
type MatchResult struct {
	ID        string            `json:"id"`
	Score     float64           `json:"score"`
	Metadata  map[string]string `json:"metadata"`
}

type matchResult struct {
	ID       string            `json:"id"`
	Score    float64           `json:"score"`
	Metadata map[string]string `json:"metadata"`
}

// SearchSimilar finds the most similar chapters to a query vector.
func (p *PineconeClient) SearchSimilar(ctx context.Context, queryVec []float64, bookID string, topK int) ([]MatchResult, error) {
	if topK <= 0 {
		topK = 5
	}

	body := queryRequest{
		Vector:          queryVec,
		TopK:            topK,
		IncludeMetadata: true,
	}

	if bookID != "" {
		body.Filter = map[string]interface{}{
			"book_id": map[string]string{"$eq": bookID},
		}
	}

	var result queryResponse
	if err := p.doIndexRequest(ctx, http.MethodPost, "/query", body, &result); err != nil {
		return nil, err
	}

	matches := make([]MatchResult, len(result.Matches))
	for i, m := range result.Matches {
		matches[i] = MatchResult{
			ID:       m.ID,
			Score:    m.Score,
			Metadata: m.Metadata,
		}
	}

	return matches, nil
}

// --- Deletion ---

type deleteRequest struct {
	Filter map[string]interface{} `json:"filter,omitempty"`
	IDs    []string               `json:"ids,omitempty"`
}

// DeleteByBook removes all vectors for a given book.
func (p *PineconeClient) DeleteByBook(ctx context.Context, bookID string) error {
	body := deleteRequest{
		Filter: map[string]interface{}{
			"book_id": map[string]string{"$eq": bookID},
		},
	}
	return p.doIndexRequest(ctx, http.MethodPost, "/vectors/delete", body, nil)
}

// DeleteChapter removes a single chapter's vector.
func (p *PineconeClient) DeleteChapter(ctx context.Context, chapterID string) error {
	body := deleteRequest{
		IDs: []string{chapterID},
	}
	return p.doIndexRequest(ctx, http.MethodPost, "/vectors/delete", body, nil)
}

// --- Health ---

// Available checks if the Pinecone index is reachable.
func (p *PineconeClient) Available(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.IndexHost+"/describe_index_stats", nil)
	if err != nil {
		return false
	}
	req.Header.Set("Api-Key", p.APIKey)

	resp, err := p.Client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// --- Helpers ---

func (p *PineconeClient) doIndexRequest(ctx context.Context, method, path string, body interface{}, dst interface{}) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, p.IndexHost+path, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Api-Key", p.APIKey)

	resp, err := p.Client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Pinecone returned %d: %s", resp.StatusCode, string(respBody))
	}

	if dst != nil {
		if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// chunkText splits text into chunks of approximately maxTokens.
func chunkText(text string, maxTokens int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	maxChars := maxTokens * 4 // ~4 chars per token
	if len(text) <= maxChars {
		return []string{text}
	}

	var chunks []string
	for len(text) > 0 {
		end := maxChars
		if end > len(text) {
			end = len(text)
		}

		// Try to break at sentence boundary
		if end < len(text) {
			lastPeriod := strings.LastIndex(text[:end], ". ")
			if lastPeriod > end/2 {
				end = lastPeriod + 1
			}
		}

		chunks = append(chunks, strings.TrimSpace(text[:end]))
		text = strings.TrimSpace(text[end:])
	}

	return chunks
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
