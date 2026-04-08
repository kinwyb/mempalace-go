package mempalace

import (
	"context"
	"log/slog"
	"time"

	"github.com/kinwyb/mempalace-go/internal/convominer"
	"github.com/kinwyb/mempalace-go/internal/miner"
)

// MineResult represents the result of a mining operation.
type MineResult struct {
	FilesProcessed int
	TotalDrawers   int
	Wing           string
	Duration       time.Duration
	Errors         []MineError
}

// MineError represents an error during mining.
type MineError struct {
	File    string
	Message string
}

// MineOption is a functional option for mining operations.
type MineOption func(*mineConfig)

type mineConfig struct {
	wing        string
	dryRun      bool
	limit       int
	extractMode string
	rooms       []RoomConfig
}

// RoomConfig represents room configuration for mining.
type RoomConfig struct {
	Name        string
	Description string
	Keywords    []string
}

// WithWingOverride overrides the wing name for mining.
func WithWingOverride(wing string) MineOption {
	return func(mc *mineConfig) {
		mc.wing = wing
	}
}

// WithDryRun enables dry run mode (no actual storage).
func WithDryRun() MineOption {
	return func(mc *mineConfig) {
		mc.dryRun = true
	}
}

// WithMineLimit limits the number of files to process.
func WithMineLimit(limit int) MineOption {
	return func(mc *mineConfig) {
		mc.limit = limit
	}
}

// WithExtractMode sets the conversation extraction mode ("exchange" or "general").
func WithExtractMode(mode string) MineOption {
	return func(mc *mineConfig) {
		mc.extractMode = mode
	}
}

// WithRoomConfigs provides custom room configuration.
func WithRoomConfigs(rooms []RoomConfig) MineOption {
	return func(mc *mineConfig) {
		mc.rooms = rooms
	}
}

// Mine mines files from a project directory into the palace.
func (p *Palace) Mine(ctx context.Context, dir string, opts ...MineOption) (*MineResult, error) {
	if p.closed {
		return nil, NewError(ErrClosed, "palace is closed", nil)
	}

	start := time.Now()

	// Apply mine options
	mc := &mineConfig{}
	for _, opt := range opts {
		opt(mc)
	}

	// Configure miner
	m := miner.New(convertConfig(p.config), p.store)
	m.SetDryRun(mc.dryRun)
	m.SetLimit(mc.limit)

	// Convert room configs
	if len(mc.rooms) > 0 {
		rooms := make([]miner.Room, len(mc.rooms))
		for i, rc := range mc.rooms {
			rooms[i] = miner.Room{
				Name:        rc.Name,
				Description: rc.Description,
				Keywords:    rc.Keywords,
			}
		}
	}

	// Perform mining
	err := m.Mine(ctx, dir, mc.wing)
	if err != nil {
		return nil, NewError(ErrMine, "mining failed", err)
	}

	duration := time.Since(start)

	slog.Info("Mining completed",
		"dir", dir,
		"wing", mc.wing,
		"dryRun", mc.dryRun,
		"duration", duration,
	)

	return &MineResult{
		Wing:     mc.wing,
		Duration: duration,
	}, nil
}

// MineConversations mines conversation files into the palace.
func (p *Palace) MineConversations(ctx context.Context, dir string, opts ...MineOption) (*MineResult, error) {
	if p.closed {
		return nil, NewError(ErrClosed, "palace is closed", nil)
	}

	start := time.Now()

	// Apply mine options
	mc := &mineConfig{
		extractMode: "exchange",
	}
	for _, opt := range opts {
		opt(mc)
	}

	// Configure convo miner
	m := convominer.New(p.store)
	m.SetDryRun(mc.dryRun)
	m.SetLimit(mc.limit)
	m.SetExtractMode(mc.extractMode)

	// Perform mining
	err := m.Mine(ctx, dir, mc.wing)
	if err != nil {
		return nil, NewError(ErrMine, "conversation mining failed", err)
	}

	duration := time.Since(start)

	slog.Info("Conversation mining completed",
		"dir", dir,
		"wing", mc.wing,
		"dryRun", mc.dryRun,
		"duration", duration,
	)

	return &MineResult{
		Wing:     mc.wing,
		Duration: duration,
	}, nil
}

// ChunkText splits text into chunks for storage.
// This is a utility function available without creating a Palace.
func ChunkText(content string, chunkSize, overlap, minSize int) []TextChunk {
	if chunkSize == 0 {
		chunkSize = 800
	}
	if overlap == 0 {
		overlap = 100
	}
	if minSize == 0 {
		minSize = 50
	}

	content = trimSpace(content)
	if content == "" {
		return nil
	}

	var chunks []TextChunk
	start := 0
	index := 0

	for start < len(content) {
		end := start + chunkSize
		if end > len(content) {
			end = len(content)
		}

		// Try to break at paragraph boundary
		if end < len(content) {
			newlinePos := lastNewline(content[start:end])
			if newlinePos > chunkSize/2 {
				end = start + newlinePos
			}
		}

		chunk := trimSpace(content[start:end])
		if len(chunk) >= minSize {
			chunks = append(chunks, TextChunk{
				Content: chunk,
				Index:   index,
			})
			index++
		}

		if end >= len(content) {
			break
		}
		start = end - overlap
		if start < 0 {
			start = 0
		}
	}

	return chunks
}

// TextChunk represents a chunk of text.
type TextChunk struct {
	Content string
	Index   int
}

// trimSpace is a helper to avoid importing strings in this file
func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t' || s[0] == '\n' || s[0] == '\r') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t' || s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}

func lastNewline(s string) int {
	// Find last \n\n or \n
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '\n' {
			return i
		}
	}
	return -1
}