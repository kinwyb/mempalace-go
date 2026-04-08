package mempalace

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kinwyb/mempalace-go/pkg/embedding"
	"gopkg.in/yaml.v3"
)

// Option is the functional options type for New().
type Option func(*palaceConfig) error

// palaceConfig holds internal configuration for palace initialization.
type palaceConfig struct {
	config     *Config
	embedder   embedding.Embedder
	configFile string
}

// WithPalacePath sets a custom palace storage path.
func WithPalacePath(path string) Option {
	return func(pc *palaceConfig) error {
		if path == "" {
			return NewError(ErrInvalidInput, "palace path cannot be empty", nil)
		}
		// Expand home directory
		if path[0] == '~' {
			home, _ := os.UserHomeDir()
			if home != "" {
				path = filepath.Join(home, path[1:])
			}
		}
		pc.config.PalacePath = path
		return nil
	}
}

// WithOllama configures Ollama as the embedding provider.
func WithOllama(host, model string) Option {
	return func(pc *palaceConfig) error {
		if model == "" {
			return NewError(ErrInvalidInput, "model cannot be empty", nil)
		}
		if host == "" {
			host = "http://localhost:11434"
		}
		pc.config.EmbeddingProvider = "ollama"
		pc.config.OllamaHost = host
		pc.config.EmbeddingModel = model
		pc.embedder = embedding.NewOllamaEmbedder(host, model)
		return nil
	}
}

// WithOpenAI configures OpenAI as the embedding provider.
func WithOpenAI(apiKey, baseURL, model string) Option {
	return func(pc *palaceConfig) error {
		if apiKey == "" {
			return NewError(ErrInvalidInput, "API key cannot be empty", nil)
		}
		if model == "" {
			model = "text-embedding-3-small"
		}
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1"
		}
		pc.config.EmbeddingProvider = "openai"
		pc.config.OpenAIAPIKey = apiKey
		pc.config.EmbeddingAPIBase = baseURL
		pc.config.EmbeddingModel = model
		pc.embedder = embedding.NewOpenAIEmbedder(apiKey, baseURL, model)
		return nil
	}
}

// WithEmbedder allows providing a custom embedder implementation.
func WithEmbedder(emb embedding.Embedder) Option {
	return func(pc *palaceConfig) error {
		if emb == nil {
			return NewError(ErrInvalidInput, "embedder cannot be nil", nil)
		}
		pc.embedder = emb
		return nil
	}
}

// WithChunkSize sets text chunking parameters.
func WithChunkSize(size, overlap, minSize int) Option {
	return func(pc *palaceConfig) error {
		if size <= 0 {
			return NewError(ErrInvalidInput, "chunk size must be positive", nil)
		}
		if overlap < 0 {
			return NewError(ErrInvalidInput, "chunk overlap cannot be negative", nil)
		}
		if minSize < 0 {
			return NewError(ErrInvalidInput, "min chunk size cannot be negative", nil)
		}
		pc.config.ChunkSize = size
		pc.config.ChunkOverlap = overlap
		pc.config.MinChunkSize = minSize
		return nil
	}
}

// WithSearchDefaults sets default search parameters.
func WithSearchDefaults(limit int, threshold float64) Option {
	return func(pc *palaceConfig) error {
		if limit <= 0 {
			return NewError(ErrInvalidInput, "search limit must be positive", nil)
		}
		if threshold < 0 || threshold > 1 {
			return NewError(ErrInvalidInput, "similarity threshold must be between 0 and 1", nil)
		}
		pc.config.SearchLimit = limit
		pc.config.SimilarityThreshold = threshold
		return nil
	}
}

// WithConfigFile loads configuration from a YAML file.
func WithConfigFile(path string) Option {
	return func(pc *palaceConfig) error {
		if path == "" {
			return NewError(ErrInvalidInput, "config file path cannot be empty", nil)
		}

		// Expand home directory
		if path[0] == '~' {
			home, _ := os.UserHomeDir()
			if home != "" {
				path = filepath.Join(home, path[1:])
			}
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			return NewError(ErrConfigLoad, fmt.Sprintf("config file not found: %s", path), err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return NewError(ErrConfigLoad, "failed to read config file", err)
		}

		// Parse YAML into the existing config (preserves defaults for missing fields)
		if err := yaml.Unmarshal(data, pc.config); err != nil {
			return NewError(ErrConfigLoad, "failed to parse config file", err)
		}

		// Expand home in palace path
		if pc.config.PalacePath != "" && pc.config.PalacePath[0] == '~' {
			home, _ := os.UserHomeDir()
			if home != "" {
				pc.config.PalacePath = filepath.Join(home, pc.config.PalacePath[1:])
			}
		}

		pc.configFile = path
		return nil
	}
}

// WithLogLevel sets the logging level.
func WithLogLevel(level string) Option {
	return func(pc *palaceConfig) error {
		pc.config.LogLevel = level
		return nil
	}
}