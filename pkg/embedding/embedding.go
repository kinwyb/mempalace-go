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

// OpenAIEmbedder implements Embedder using OpenAI-compatible API.
type OpenAIEmbedder struct {
	apiKey    string
	baseURL   string
	model     string
	dimension int
	client    *http.Client
}

// OpenAIRequest represents an OpenAI API request.
type OpenAIRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// OpenAIResponse represents an OpenAI API response.
type OpenAIResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Model string `json:"model"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// OpenAI model dimensions
var openAIModelDimensions = map[string]int{
	"text-embedding-ada-002":     1536,
	"text-embedding-3-small":     1536,
	"text-embedding-3-large":     3072,
	"text-embedding-3-small-512": 512, // Custom dimension via dimensions parameter
}

// NewOpenAIEmbedder creates a new OpenAI-compatible embedder.
// baseURL can be empty (defaults to https://api.openai.com/v1) or a custom endpoint.
func NewOpenAIEmbedder(apiKey, baseURL, model string) *OpenAIEmbedder {
	dimension := 1536 // Default
	if d, ok := openAIModelDimensions[model]; ok {
		dimension = d
	}

	// Default base URL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	return &OpenAIEmbedder{
		apiKey:    apiKey,
		baseURL:   baseURL,
		model:     model,
		dimension: dimension,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Embed generates an embedding for a single text.
func (e *OpenAIEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := e.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 || len(embeddings[0]) == 0 {
		return nil, fmt.Errorf("empty embedding returned")
	}
	return embeddings[0], nil
}

// EmbedBatch generates embeddings for multiple texts.
func (e *OpenAIEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if e.apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key not configured")
	}

	reqBody := OpenAIRequest{
		Model: e.model,
		Input: texts,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", e.baseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var openAIResp OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if openAIResp.Error != nil {
		return nil, fmt.Errorf("OpenAI API error: %s", openAIResp.Error.Message)
	}

	if len(openAIResp.Data) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	embeddings := make([][][]float32, len(texts))
	for _, d := range openAIResp.Data {
		if d.Index < len(texts) {
			embeddings[d.Index] = append(embeddings[d.Index], d.Embedding)
		}
	}

	// Flatten to correct order
	result := make([][]float32, len(texts))
	for i := range texts {
		if len(embeddings[i]) > 0 {
			result[i] = embeddings[i][0]
		}
	}

	slog.Debug("Generated embeddings", "model", e.model, "count", len(result))
	return result, nil
}

// Dimension returns the embedding dimension.
func (e *OpenAIEmbedder) Dimension() int {
	return e.dimension
}

// Model returns the model name.
func (e *OpenAIEmbedder) Model() string {
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

// Embed generates a mock embedding based on text content.
func (e *MockEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	// Generate deterministic mock embedding based on text hash
	embedding := make([]float32, e.dimension)

	// Simple hash-based embedding generation for variety
	if len(text) > 0 {
		hash := uint32(0)
		for _, c := range text {
			hash = hash*31 + uint32(c)
		}

		// Use hash to seed different values
		for i := range embedding {
			seed := hash + uint32(i)
			// Generate normalized values between -1 and 1
			val := float32((seed % 1000) - 500) / 500.0
			embedding[i] = val
		}

		// Normalize the embedding
		var norm float32
		for _, v := range embedding {
			norm += v * v
		}
		if norm > 0 {
			norm = float32(1.0 / sqrt(float64(norm)))
			for i := range embedding {
				embedding[i] *= norm
			}
		}
	} else {
		// Default embedding for empty text
		for i := range embedding {
			embedding[i] = 0.01
		}
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

// sqrt computes square root using Newton's method
func sqrt(x float64) float64 {
	if x < 0 {
		return 0
	}
	z := x
	for i := 0; i < 10; i++ {
		z = (z + x/z) / 2
	}
	return z
}
