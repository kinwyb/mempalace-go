package mempalace

import (
	"os"
	"path/filepath"
)

// Config holds all configuration for MemPalace.
type Config struct {
	// Paths
	PalacePath string

	// Embedding configuration
	EmbeddingProvider string // "ollama" or "openai"
	EmbeddingModel    string
	EmbeddingAPIBase  string // For OpenAI-compatible APIs
	OpenAIAPIKey      string
	OllamaHost        string

	// Text processing
	ChunkSize      int
	ChunkOverlap   int
	MinChunkSize   int

	// Search defaults
	SearchLimit         int
	SimilarityThreshold float64

	// Logging
	LogLevel string
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	if homeDir == "" {
		homeDir = "/tmp"
	}

	return &Config{
		PalacePath:          filepath.Join(homeDir, ".mempalace", "palace"),
		EmbeddingProvider:   "ollama",
		EmbeddingModel:      "nomic-embed-text",
		EmbeddingAPIBase:    "",
		OpenAIAPIKey:        "",
		OllamaHost:          "http://localhost:11434",
		LogLevel:            "info",
		ChunkSize:           800,
		ChunkOverlap:        100,
		MinChunkSize:        50,
		SearchLimit:         10,
		SimilarityThreshold: 0.9,
	}
}

// GetConfigPath returns the default config file path.
func GetConfigPath() string {
	homeDir, _ := os.UserHomeDir()
	if homeDir == "" {
		return "mempalace.yaml"
	}
	return filepath.Join(homeDir, ".mempalace", "config.yaml")
}