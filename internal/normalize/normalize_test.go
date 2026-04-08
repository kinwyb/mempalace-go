package normalize

import (
	"testing"
)

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name     string
		content  []byte
		ext      string
		expected Format
	}{
		{
			name:     "generic text",
			content:  []byte("This is plain text content."),
			ext:      ".txt",
			expected: FormatGeneric,
		},
		{
			name:     "markdown file",
			content:  []byte("# Heading\n\nContent here"),
			ext:      ".md",
			expected: FormatGeneric,
		},
		{
			name:     "claude format",
			content:  []byte(`{"conversations": [{"role": "user", "content": "Hello"}]}`),
			ext:      ".json",
			expected: FormatClaude,
		},
		{
			name:     "chatgpt format",
			content:  []byte(`{"mapping": {"node1": {"message": {"role": "user"}}}}`),
			ext:      ".json",
			expected: FormatChatGPT,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format := DetectFormat(tt.content, tt.ext)
			if format != tt.expected {
				t.Errorf("DetectFormat() = %s, want %s", format, tt.expected)
			}
		})
	}
}

func TestNormalize(t *testing.T) {
	// Test with nil/empty scenarios
	format := DetectFormat([]byte{}, ".txt")
	if format != FormatGeneric {
		t.Errorf("Empty content should be generic, got %s", format)
	}
}

func TestNormalizeGeneric(t *testing.T) {
	content := []byte("Line 1\r\nLine 2\r\nLine 3")

	result, err := normalizeGeneric(content)
	if err != nil {
		t.Fatalf("normalizeGeneric failed: %v", err)
	}

	// Should convert \r\n to \n
	if result != "Line 1\nLine 2\nLine 3" {
		t.Errorf("Unexpected result: %s", result)
	}
}

func TestIsClaudeFormat(t *testing.T) {
	tests := []struct {
		content  string
		expected bool
	}{
		{
			content:  `{"conversations": []}`,
			expected: true,
		},
		{
			content:  `{"messages": [], "model": "claude"}`,
			expected: true,
		},
		{
			content:  `{"other": "data"}`,
			expected: false,
		},
		{
			content:  `invalid json`,
			expected: false,
		},
	}

	for i, tt := range tests {
		result := isClaudeFormat([]byte(tt.content))
		if result != tt.expected {
			t.Errorf("Test %d: isClaudeFormat() = %v, want %v", i, result, tt.expected)
		}
	}
}

func TestIsChatGPTFormat(t *testing.T) {
	tests := []struct {
		content  string
		expected bool
	}{
		{
			content:  `{"mapping": {"node1": {}}}`,
			expected: true,
		},
		{
			content:  `{"other": "data"}`,
			expected: false,
		},
	}

	for i, tt := range tests {
		result := isChatGPTFormat([]byte(tt.content))
		if result != tt.expected {
			t.Errorf("Test %d: isChatGPTFormat() = %v, want %v", i, result, tt.expected)
		}
	}
}

func TestExtractContent(t *testing.T) {
	tests := []struct {
		content  any
		expected string
	}{
		{
			content:  "simple string",
			expected: "simple string",
		},
		{
			content:  []any{map[string]any{"text": "first"}, map[string]any{"text": "second"}},
			expected: "first\nsecond",
		},
		{
			content:  123,
			expected: "",
		},
	}

	for i, tt := range tests {
		result := extractContent(tt.content)
		if result != tt.expected {
			t.Errorf("Test %d: extractContent() = %s, want %s", i, result, tt.expected)
		}
	}
}
