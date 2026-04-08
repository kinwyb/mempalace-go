package miner

import (
	"testing"
)

func TestChunkText(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected int // minimum expected chunks
	}{
		{
			name:     "short content",
			content:  "This is short",
			expected: 0, // Less than MinChunkSize
		},
		{
			name:     "single chunk",
			content:  string(make([]byte, 500)), // 500 chars
			expected: 1,
		},
		{
			name:     "multiple chunks",
			content:  string(make([]byte, 2000)), // 2000 chars
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := ChunkText(tt.content, "test.txt")

			if len(chunks) < tt.expected {
				t.Errorf("Expected at least %d chunks, got %d", tt.expected, len(chunks))
			}
		})
	}
}

func TestChunkTextWithParagraphs(t *testing.T) {
	content := "First paragraph.\n\nSecond paragraph.\n\nThird paragraph with more text to make it longer than the minimum chunk size requirement for testing purposes."

	chunks := ChunkText(content, "test.txt")

	if len(chunks) == 0 {
		t.Error("Expected at least one chunk")
	}
}

func TestDetectRoom(t *testing.T) {
	rooms := []Room{
		{Name: "api", Keywords: []string{"endpoint", "handler", "route"}},
		{Name: "database", Keywords: []string{"sql", "query", "table"}},
		{Name: "tests", Keywords: []string{"test", "spec", "mock"}},
	}

	tests := []struct {
		filePath    string
		content     string
		projectPath string
		expected    string
	}{
		{
			filePath:    "/project/api/handlers.go",
			content:     "package handlers",
			projectPath: "/project",
			expected:    "api",
		},
		{
			filePath:    "/project/internal/db/models.go",
			content:     "sql query table database",
			projectPath: "/project",
			expected:    "database",
		},
		{
			filePath:    "/project/tests/main_test.go",
			content:     "func TestSomething",
			projectPath: "/project",
			expected:    "tests",
		},
	}

	for _, tt := range tests {
		t.Run(tt.filePath, func(t *testing.T) {
			room := DetectRoom(tt.filePath, tt.content, rooms, tt.projectPath)
			if room != tt.expected {
				t.Errorf("DetectRoom() = %s, want %s", room, tt.expected)
			}
		})
	}
}

func TestReadableExtensions(t *testing.T) {
	extensions := []string{".go", ".py", ".js", ".ts", ".md", ".txt", ".json"}

	for _, ext := range extensions {
		if !ReadableExtensions[ext] {
			t.Errorf("Extension %s should be readable", ext)
		}
	}

	nonReadable := []string{".bin", ".exe", ".png", ".jpg"}

	for _, ext := range nonReadable {
		if ReadableExtensions[ext] {
			t.Errorf("Extension %s should not be readable", ext)
		}
	}
}

func TestSkipDirs(t *testing.T) {
	dirs := []string{".git", "node_modules", "vendor", "dist", "build"}

	for _, dir := range dirs {
		if !SkipDirs[dir] {
			t.Errorf("Directory %s should be skipped", dir)
		}
	}

	keepDirs := []string{"src", "lib", "cmd"}

	for _, dir := range keepDirs {
		if SkipDirs[dir] {
			t.Errorf("Directory %s should not be skipped", dir)
		}
	}
}

func TestSkipFilenames(t *testing.T) {
	files := []string{".gitignore", "package-lock.json"}

	for _, file := range files {
		if !SkipFilenames[file] {
			t.Errorf("File %s should be skipped", file)
		}
	}
}
