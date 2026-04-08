// Package config provides configuration management for MemPalace.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"os/user"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds the MemPalace configuration.
type Config struct {
	// PalacePath is the directory where the palace (vector store) lives.
	PalacePath string `yaml:"palace_path"`

	// EmbeddingProvider is the embedding provider (ollama or openai).
	EmbeddingProvider string `yaml:"embedding_provider"`

	// EmbeddingModel is the model used for generating embeddings.
	EmbeddingModel string `yaml:"embedding_model"`

	// EmbeddingAPIBase is the base URL for the embedding API.
	EmbeddingAPIBase string `yaml:"embedding_api_base"`

	// OpenAIAPIKey is the OpenAI API key.
	OpenAIAPIKey string `yaml:"openai_api_key"`

	// OllamaHost is the Ollama server host.
	OllamaHost string `yaml:"ollama_host"`

	// LogLevel is the logging level (debug, info, warn, error).
	LogLevel string `yaml:"log_level"`

	// ChunkSize is the default chunk size for text splitting.
	ChunkSize int `yaml:"chunk_size"`

	// ChunkOverlap is the overlap between chunks.
	ChunkOverlap int `yaml:"chunk_overlap"`

	// MinChunkSize is the minimum chunk size to store.
	MinChunkSize int `yaml:"min_chunk_size"`

	// SearchLimit is the default number of search results.
	SearchLimit int `yaml:"search_limit"`

	// SimilarityThreshold is the threshold for duplicate detection.
	SimilarityThreshold float64 `yaml:"similarity_threshold"`
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

// Load loads configuration from file, environment, and defaults.
// Priority: file > environment > defaults.
func Load(configPath string) (*Config, error) {
	cfg := DefaultConfig()

	// Load from file if exists
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			data, err := os.ReadFile(configPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read config file: %w", err)
			}
			if err := yaml.Unmarshal(data, cfg); err != nil {
				return nil, fmt.Errorf("failed to parse config file: %w", err)
			}
			slog.Debug("Loaded config from file", "path", configPath)
		}
	}

	// Override with environment variables
	applyEnvOverrides(cfg)

	// Expand home directory in paths
	cfg.PalacePath = expandHome(cfg.PalacePath)

	return cfg, nil
}

// applyEnvOverrides applies environment variable overrides (only if config value is empty).
func applyEnvOverrides(cfg *Config) {
	if cfg.PalacePath == "" {
		if v := os.Getenv("MEMPALACE_PALACE_PATH"); v != "" {
			cfg.PalacePath = v
		}
	}
	if cfg.EmbeddingProvider == "" {
		if v := os.Getenv("MEMPALACE_EMBEDDING_PROVIDER"); v != "" {
			cfg.EmbeddingProvider = v
		}
	}
	if cfg.EmbeddingModel == "" {
		if v := os.Getenv("MEMPALACE_EMBEDDING_MODEL"); v != "" {
			cfg.EmbeddingModel = v
		}
	}
	if cfg.EmbeddingAPIBase == "" {
		if v := os.Getenv("MEMPALACE_EMBEDDING_API_BASE"); v != "" {
			cfg.EmbeddingAPIBase = v
		}
	}
	if cfg.OpenAIAPIKey == "" {
		if v := os.Getenv("MEMPALACE_OPENAI_API_KEY"); v != "" {
			cfg.OpenAIAPIKey = v
		}
		if v := os.Getenv("OPENAI_API_KEY"); v != "" && cfg.OpenAIAPIKey == "" {
			cfg.OpenAIAPIKey = v
		}
	}
	if cfg.OllamaHost == "" {
		if v := os.Getenv("MEMPALACE_OLLAMA_HOST"); v != "" {
			cfg.OllamaHost = v
		}
	}
	if cfg.LogLevel == "" {
		if v := os.Getenv("MEMPALACE_LOG_LEVEL"); v != "" {
			cfg.LogLevel = v
		}
	}
	if cfg.ChunkSize == 0 {
		if v := os.Getenv("MEMPALACE_CHUNK_SIZE"); v != "" {
			if size, err := parseInt(v); err == nil {
				cfg.ChunkSize = size
			}
		}
	}
	if cfg.SearchLimit == 0 {
		if v := os.Getenv("MEMPALACE_SEARCH_LIMIT"); v != "" {
			if limit, err := parseInt(v); err == nil {
				cfg.SearchLimit = limit
			}
		}
	}
}

// expandHome expands ~ to the user's home directory.
func expandHome(path string) string {
	if path == "" {
		return path
	}
	if path[0] == '~' {
		usr, err := user.Current()
		if err != nil {
			home := os.Getenv("HOME")
			if home == "" {
				return path
			}
			return filepath.Join(home, path[1:])
		}
		return filepath.Join(usr.HomeDir, path[1:])
	}
	return path
}

// parseInt parses an integer from a string.
func parseInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}

// Save saves the configuration to a file.
func (c *Config) Save(configPath string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	slog.Info("Saved config to file", "path", configPath)
	return nil
}

// EnsurePalacePath ensures the palace directory exists.
func (c *Config) EnsurePalacePath() error {
	if err := os.MkdirAll(c.PalacePath, 0755); err != nil {
		return fmt.Errorf("failed to create palace directory: %w", err)
	}
	return nil
}

// GetConfigPath returns the default config file path.
func GetConfigPath() string {
	homeDir, _ := os.UserHomeDir()
	if homeDir == "" {
		return "mempalace.yaml"
	}
	return filepath.Join(homeDir, ".mempalace", "config.yaml")
}
