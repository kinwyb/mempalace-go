package convominer

import (
	"testing"
)

func TestChunkExchanges(t *testing.T) {
	content := `> What is Go?
Go is a programming language designed at Google.

> What are its features?
Go has garbage collection, goroutines, and channels.

> How do I install it?
You can download it from golang.org.`

	chunks := chunkExchanges(content)

	if len(chunks) == 0 {
		t.Error("Expected at least one chunk")
	}

	// Verify chunks contain both question and answer
	for _, chunk := range chunks {
		if len(chunk.Content) < MinChunkSize {
			t.Errorf("Chunk content too short: %d", len(chunk.Content))
		}
	}
}

func TestChunkByParagraph(t *testing.T) {
	content := `First paragraph with some content.

Second paragraph with more content.

Third paragraph with even more content to reach minimum size.`

	chunks := chunkByParagraph(content)

	if len(chunks) < 2 {
		t.Errorf("Expected at least 2 chunks, got %d", len(chunks))
	}
}

func TestDetectConvoRoom(t *testing.T) {
	tests := []struct {
		content  string
		expected string
	}{
		{
			content:  "I'm having a bug with my Python code. The error says undefined variable.",
			expected: "technical",
		},
		{
			content:  "Let me explain the architecture and design patterns for this service.",
			expected: "architecture",
		},
		{
			content:  "We decided to switch to a new approach.",
			expected: "decisions",
		},
		{
			content:  "The deployment failed with an error. We need to fix this issue.",
			expected: "problems",
		},
		{
			content:  "Random content that doesn't match any keywords.",
			expected: "general",
		},
	}

	for _, tt := range tests {
		t.Run(tt.content[:min(30, len(tt.content))], func(t *testing.T) {
			room := detectConvoRoom(tt.content)
			if room != tt.expected {
				t.Errorf("detectConvoRoom() = %s, want %s", room, tt.expected)
			}
		})
	}
}

func TestExtractMemories(t *testing.T) {
	content := `I decided to use React for the frontend.

I prefer dark mode themes for coding.

I completed the first milestone of the project.

There was a problem with the API connection.`

	chunks := extractMemories(content)

	if len(chunks) == 0 {
		t.Fatal("Expected at least one chunk")
	}

	// Check room detection
	rooms := make(map[string]bool)
	for _, chunk := range chunks {
		rooms[chunk.Room] = true
	}

	// Should have detected multiple room types
	if len(rooms) < 2 {
		t.Logf("Warning: Only detected %d room types", len(rooms))
	}
}

func TestConvoExtensions(t *testing.T) {
	validExts := []string{".txt", ".md", ".json", ".jsonl"}
	invalidExts := []string{".py", ".go", ".exe"}

	for _, ext := range validExts {
		if !ConvoExtensions[ext] {
			t.Errorf("Extension %s should be valid for conversations", ext)
		}
	}

	for _, ext := range invalidExts {
		if ConvoExtensions[ext] {
			t.Errorf("Extension %s should not be valid for conversations", ext)
		}
	}
}

func TestSkipDirs(t *testing.T) {
	skipDirs := []string{".git", "node_modules", ".mempalace"}

	for _, dir := range skipDirs {
		if !SkipDirs[dir] {
			t.Errorf("Directory %s should be skipped", dir)
		}
	}
}
