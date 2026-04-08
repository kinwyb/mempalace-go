// Package embedding provides embedding generation interfaces and implementations.
package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// Embedder is the interface for generating text embeddings.
type Embedder interface {
	// Embed generates an embedding for a single text.
	Embed(ctx context.Context, text string) ([]float32, error)

	// EmbedBatch generates embeddings for multiple texts.
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)

	// Dimension returns the embedding dimension.
	Dimension() int

	// Model returns the model name.
	Model() string
}

// OllamaEmbedder implements Embedder using Ollama.
type OllamaEmbedder struct {
	host      string
	model     string
	dimension int
	client    *http.Client
}

// OllamaRequest represents an Ollama API request.
type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// OllamaResponse represents an Ollama API response.
type OllamaResponse struct {
	Embedding []float32 `json:"embedding"`
}

// NewOllamaEmbedder creates a new Ollama embedder.
func NewOllamaEmbedder(host, model string) *OllamaEmbedder {
	return &OllamaEmbedder{
		host:      host,
		model:     model,
		dimension: 768, // Default for nomic-embed-text
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Embed generates an embedding for a single text.
func (e *OllamaEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	reqBody := OllamaRequest{
		Model:  e.model,
		Prompt: text,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := e.host + "/api/embeddings"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, body)
	}

	var ollamaResp OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(ollamaResp.Embedding) == 0 {
		return nil, fmt.Errorf("empty embedding returned")
	}

	slog.Debug("Generated embedding", "model", e.model, "dimension", len(ollamaResp.Embedding))
	return ollamaResp.Embedding, nil
}

// EmbedBatch generates embeddings for multiple texts.
func (e *OllamaEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))

	for i, text := range texts {
		emb, err := e.Embed(ctx, text)
		if err != nil {
			slog.Warn("failed to embed text in batch", "index", i, "error", err)
			continue
		}
		embeddings[i] = emb
	}

	return embeddings, nil
}

// Dimension returns the embedding dimension.
func (e *OllamaEmbedder) Dimension() int {
	return e.dimension
}

// Model returns the model name.
func (e *OllamaEmbedder) Model() string {
	return e.model
}

// MockEmbedder is a mock embedder for testing.
type MockEmbedder struct {
	dimension int
}

// NewMockEmbedder creates a new mock embedder.
func NewMockEmbedder(dimension int) *MockEmbedder {
	return &MockEmbedder{dimension: dimension}
}

// Embed generates a mock embedding.
func (e *MockEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	// Generate deterministic mock embedding based on text hash
	embedding := make([]float32, e.dimension)
	for i := range embedding {
		embedding[i] = 0.1 + float32(i%10)/100.0
	}
	return embedding, nil
}

// EmbedBatch generates mock embeddings.
func (e *MockEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i := range texts {
		emb, _ := e.Embed(ctx, texts[i])
		embeddings[i] = emb
	}
	return embeddings, nil
}

// Dimension returns the embedding dimension.
func (e *MockEmbedder) Dimension() int {
	return e.dimension
}

// Model returns the model name.
func (e *MockEmbedder) Model() string {
	return "mock"
}
