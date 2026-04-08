// Package normalize provides conversation format normalization.
package normalize

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// Format represents a conversation format type
type Format string

const (
	FormatClaude  Format = "claude"
	FormatChatGPT Format = "chatgpt"
	FormatSlack   Format = "slack"
	FormatGeneric Format = "generic"
)

// Normalizer is the interface for conversation normalization
type Normalizer interface {
	Normalize(input io.Reader) (string, error)
	Detect(content []byte) bool
}

// ClaudeNormalizer normalizes Claude conversation exports
type ClaudeNormalizer struct{}

// ChatGPTNormalizer normalizes ChatGPT conversation exports
type ChatGPTNormalizer struct{}

// SlackNormalizer normalizes Slack exports
type SlackNormalizer struct{}

// Normalize normalizes a file based on detected format
func Normalize(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	format := DetectFormat(content, filepath.Ext(filePath))
	slog.Debug("Detected format", "file", filePath, "format", format)

	var result string
	switch format {
	case FormatClaude:
		result, err = normalizeClaude(content)
	case FormatChatGPT:
		result, err = normalizeChatGPT(content)
	case FormatSlack:
		result, err = normalizeSlack(content)
	default:
		result, err = normalizeGeneric(content)
	}

	if err != nil {
		return "", err
	}

	return strings.TrimSpace(result), nil
}

// DetectFormat detects the conversation format from content
func DetectFormat(content []byte, ext string) Format {
	ext = strings.ToLower(ext)

	// Check JSON formats
	if ext == ".json" || ext == ".jsonl" {
		// Try to detect Claude format
		if isClaudeFormat(content) {
			return FormatClaude
		}
		// Try to detect ChatGPT format
		if isChatGPTFormat(content) {
			return FormatChatGPT
		}
		// Try to detect Slack format
		if isSlackFormat(content) {
			return FormatSlack
		}
	}

	return FormatGeneric
}

func isClaudeFormat(content []byte) bool {
	// Claude conversations have specific structure
	var data map[string]any
	if err := json.Unmarshal(content, &data); err == nil {
		// Check for Claude-specific fields
		if _, ok := data["conversations"]; ok {
			return true
		}
		if _, ok := data["messages"]; ok {
			if _, ok := data["model"]; ok {
				return true
			}
		}
	}
	return false
}

func isChatGPTFormat(content []byte) bool {
	var data map[string]any
	if err := json.Unmarshal(content, &data); err == nil {
		if _, ok := data["mapping"]; ok {
			return true
		}
	}
	return false
}

func isSlackFormat(content []byte) bool {
	var data map[string]any
	if err := json.Unmarshal(content, &data); err == nil {
		if _, ok := data["messages"]; ok {
			if msgs, ok := data["messages"].([]any); ok && len(msgs) > 0 {
				if firstMsg, ok := msgs[0].(map[string]any); ok {
					if _, ok := firstMsg["user"]; ok {
						return true
					}
				}
			}
		}
	}
	return false
}

func normalizeClaude(content []byte) (string, error) {
	var data map[string]any
	if err := json.Unmarshal(content, &data); err != nil {
		// Try JSONL format
		return normalizeClaudeJSONL(content)
	}

	var builder strings.Builder

	// Handle conversations array
	if convs, ok := data["conversations"].([]any); ok {
		for _, conv := range convs {
			if c, ok := conv.(map[string]any); ok {
				processClaudeConversation(c, &builder)
			}
		}
	}

	// Handle messages array
	if msgs, ok := data["messages"].([]any); ok {
		for _, msg := range msgs {
			if m, ok := msg.(map[string]any); ok {
				role, _ := m["role"].(string)
				content, _ := m["content"].(string)
				if role == "user" {
					builder.WriteString("> ")
					builder.WriteString(content)
					builder.WriteString("\n\n")
				} else if role == "assistant" {
					builder.WriteString(content)
					builder.WriteString("\n\n---\n\n")
				}
			}
		}
	}

	return builder.String(), nil
}

func normalizeClaudeJSONL(content []byte) (string, error) {
	lines := strings.Split(string(content), "\n")
	var builder strings.Builder

	for _, line := range lines {
		if line == "" {
			continue
		}

		var msg map[string]any
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}

		role, _ := msg["role"].(string)
		content, _ := msg["content"].(string)

		if role == "user" {
			builder.WriteString("> ")
			builder.WriteString(content)
			builder.WriteString("\n\n")
		} else if role == "assistant" {
			builder.WriteString(content)
			builder.WriteString("\n\n---\n\n")
		}
	}

	return builder.String(), nil
}

func processClaudeConversation(conv map[string]any, builder *strings.Builder) {
	if msgs, ok := conv["messages"].([]any); ok {
		for _, msg := range msgs {
			if m, ok := msg.(map[string]any); ok {
				role, _ := m["role"].(string)
				content := extractContent(m["content"])

				if role == "user" {
					builder.WriteString("> ")
					builder.WriteString(content)
					builder.WriteString("\n\n")
				} else if role == "assistant" {
					builder.WriteString(content)
					builder.WriteString("\n\n---\n\n")
				}
			}
		}
	}
}

func extractContent(content any) string {
	switch c := content.(type) {
	case string:
		return c
	case []any:
		var parts []string
		for _, item := range c {
			if itemMap, ok := item.(map[string]any); ok {
				if text, ok := itemMap["text"].(string); ok {
					parts = append(parts, text)
				}
			}
		}
		return strings.Join(parts, "\n")
	default:
		return ""
	}
}

func normalizeChatGPT(content []byte) (string, error) {
	var data map[string]any
	if err := json.Unmarshal(content, &data); err != nil {
		return "", err
	}

	var builder strings.Builder

	// Handle ChatGPT mapping structure
	if mapping, ok := data["mapping"].(map[string]any); ok {
		// Sort by creation time and process messages
		var messages []map[string]any
		for _, node := range mapping {
			if n, ok := node.(map[string]any); ok {
				if msg, ok := n["message"].(map[string]any); ok {
					messages = append(messages, msg)
				}
			}
		}

		for _, msg := range messages {
			role, _ := msg["role"].(string)
			content := extractChatGPTContent(msg["content"])

			if role == "user" {
				builder.WriteString("> ")
				builder.WriteString(content)
				builder.WriteString("\n\n")
			} else if role == "assistant" {
				builder.WriteString(content)
				builder.WriteString("\n\n---\n\n")
			}
		}
	}

	return builder.String(), nil
}

func extractChatGPTContent(content any) string {
	switch c := content.(type) {
	case string:
		return c
	case map[string]any:
		if parts, ok := c["parts"].([]any); ok {
			var texts []string
			for _, part := range parts {
				if text, ok := part.(string); ok {
					texts = append(texts, text)
				}
			}
			return strings.Join(texts, "\n")
		}
		return ""
	default:
		return ""
	}
}

func normalizeSlack(content []byte) (string, error) {
	var data map[string]any
	if err := json.Unmarshal(content, &data); err != nil {
		return "", err
	}

	var builder strings.Builder

	if msgs, ok := data["messages"].([]any); ok {
		for _, msg := range msgs {
			if m, ok := msg.(map[string]any); ok {
				user, _ := m["user"].(string)
				text, _ := m["text"].(string)
				ts, _ := m["ts"].(string)

				if text != "" {
					builder.WriteString("[")
					builder.WriteString(ts)
					builder.WriteString("] ")
					builder.WriteString(user)
					builder.WriteString(": ")
					builder.WriteString(text)
					builder.WriteString("\n\n")
				}
			}
		}
	}

	return builder.String(), nil
}

func normalizeGeneric(content []byte) (string, error) {
	text := string(content)

	// Clean up common formatting
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.TrimSpace(text)

	return text, nil
}
