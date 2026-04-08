package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	if cfg.PalacePath == "" {
		t.Error("Default palace path should not be empty")
	}

	if cfg.EmbeddingModel == "" {
		t.Error("Default embedding model should not be empty")
	}

	if cfg.ChunkSize <= 0 {
		t.Error("Default chunk size should be positive")
	}

	if cfg.SearchLimit <= 0 {
		t.Error("Default search limit should be positive")
	}
}

func TestLoad(t *testing.T) {
	// Test loading non-existent config
	cfg, err := Load("/non/existent/path")
	if err != nil {
		t.Errorf("Load should succeed for non-existent config: %v", err)
	}

	if cfg == nil {
		t.Fatal("Load returned nil config")
	}

	// Should return defaults
	if cfg.EmbeddingModel != "nomic-embed-text" {
		t.Errorf("Expected default embedding model, got %s", cfg.EmbeddingModel)
	}
}

func TestSaveAndLoad(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "mempalace-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create config
	cfg := DefaultConfig()
	cfg.PalacePath = "/custom/palace"
	cfg.EmbeddingModel = "custom-model"
	cfg.ChunkSize = 1000

	// Save
	err = cfg.Save(configPath)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Load
	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if loaded.PalacePath != "/custom/palace" {
		t.Errorf("Expected palace path /custom/palace, got %s", loaded.PalacePath)
	}

	if loaded.EmbeddingModel != "custom-model" {
		t.Errorf("Expected embedding model custom-model, got %s", loaded.EmbeddingModel)
	}

	if loaded.ChunkSize != 1000 {
		t.Errorf("Expected chunk size 1000, got %d", loaded.ChunkSize)
	}
}

func TestEnvironmentOverride(t *testing.T) {
	// Set environment variable
	os.Setenv("MEMPALACE_EMBEDDING_MODEL", "env-model")
	defer os.Unsetenv("MEMPALACE_EMBEDDING_MODEL")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.EmbeddingModel != "env-model" {
		t.Errorf("Expected embedding model from env, got %s", cfg.EmbeddingModel)
	}
}

func TestEnsurePalacePath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mempalace-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := DefaultConfig()
	cfg.PalacePath = filepath.Join(tmpDir, "test-palace")

	err = cfg.EnsurePalacePath()
	if err != nil {
		t.Fatalf("EnsurePalacePath failed: %v", err)
	}

	// Check directory exists
	if _, err := os.Stat(cfg.PalacePath); os.IsNotExist(err) {
		t.Error("Palace path was not created")
	}
}

func TestExpandHome(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"/absolute/path", "/absolute/path"},
	}

	for _, tt := range tests {
		result := expandHome(tt.input)
		// Skip home expansion test as it depends on environment
		if tt.input == "" || tt.input[0] != '~' {
			if result != tt.expected {
				t.Errorf("expandHome(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		}
	}
}
