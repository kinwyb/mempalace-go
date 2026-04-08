// Package layers implements the 4-layer memory stack (L0-L3).
package layers

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/kinwyb/mempalace-go/pkg/vector"
)

// Layer represents a memory layer
type Layer int

const (
	// L0 - Identity layer: core identity, essential story, critical preferences
	L0 Layer = 0
	// L1 - Essential layer: project context, current goals, recent decisions
	L1 Layer = 1
	// L2 - On-Demand layer: search-based retrieval when needed
	L2 Layer = 2
	// L3 - Deep Search layer: comprehensive search with full context
	L3 Layer = 3
)

// LayerConfig defines configuration for each layer
type LayerConfig struct {
	Name         string
	Description  string
	MaxTokens    int
	Priority     int
	QueryPattern string
	Rooms        []string
}

// Default layer configurations
var LayerConfigs = map[Layer]LayerConfig{
	L0: {
		Name:         "Identity",
		Description:  "Core identity, essential story, critical preferences",
		MaxTokens:    500,
		Priority:     100,
		QueryPattern: "identity about_me core essentials",
		Rooms:        []string{"identity", "about_me", "core", "preferences"},
	},
	L1: {
		Name:         "Essential Story",
		Description:  "Project context, current goals, recent decisions",
		MaxTokens:    1000,
		Priority:     80,
		QueryPattern: "current recent active project goal",
		Rooms:        []string{"project_context", "goals", "recent", "active"},
	},
	L2: {
		Name:         "On-Demand",
		Description:  "Search-based retrieval when needed",
		MaxTokens:    2000,
		Priority:     50,
		QueryPattern: "",
		Rooms:        []string{},
	},
	L3: {
		Name:         "Deep Search",
		Description:  "Comprehensive search with full context",
		MaxTokens:    5000,
		Priority:     20,
		QueryPattern: "",
		Rooms:        []string{},
	},
}

// Layers manages the 4-layer memory stack
type Layers struct {
	store   vector.Store
	configs map[Layer]LayerConfig
}

// New creates a new Layers manager
func New(store vector.Store) *Layers {
	return &Layers{
		store:   store,
		configs: LayerConfigs,
	}
}

// WakeUp generates the L0 + L1 wake-up context
func (l *Layers) WakeUp(ctx context.Context) (string, error) {
	var builder strings.Builder

	// L0: Identity
	builder.WriteString("## L0 - Identity (Always Active)\n")
	builder.WriteString("Your core identity, essential story, and critical preferences.\n\n")

	l0Content, err := l.getLayerContent(ctx, L0)
	if err != nil {
		slog.Warn("failed to get L0 content", "error", err)
	}
	if l0Content != "" {
		builder.WriteString(l0Content)
	} else {
		builder.WriteString("(No identity content stored)\n")
	}
	builder.WriteString("\n\n")

	// L1: Essential Story
	builder.WriteString("## L1 - Essential Story (Context Window)\n")
	builder.WriteString("Project context, current goals, and recent decisions.\n\n")

	l1Content, err := l.getLayerContent(ctx, L1)
	if err != nil {
		slog.Warn("failed to get L1 content", "error", err)
	}
	if l1Content != "" {
		builder.WriteString(l1Content)
	} else {
		builder.WriteString("(No essential content stored)\n")
	}
	builder.WriteString("\n")

	return builder.String(), nil
}

// getLayerContent retrieves content for a specific layer
func (l *Layers) getLayerContent(ctx context.Context, layer Layer) (string, error) {
	config := l.configs[layer]

	var results []vector.SearchResult
	var err error

	// For L0 and L1, search specific rooms
	if len(config.Rooms) > 0 {
		for _, room := range config.Rooms {
			roomResults, err := l.store.Search(ctx, "", "", room, 5)
			if err == nil && len(roomResults) > 0 {
				results = append(results, roomResults...)
			}
		}
	}

	// If no results from rooms, try query pattern
	if len(results) == 0 && config.QueryPattern != "" {
		results, err = l.store.Search(ctx, config.QueryPattern, "", "", 10)
		if err != nil {
			return "", err
		}
	}

	if len(results) == 0 {
		return "", nil
	}

	// Format results
	var builder strings.Builder
	for _, result := range results {
		builder.WriteString(result.Content)
		builder.WriteString("\n\n")
	}

	content := builder.String()

	// Truncate if exceeds max tokens
	if len(content) > config.MaxTokens*4 { // Approximate token count
		content = content[:config.MaxTokens*4] + "\n... (truncated)"
	}

	return content, nil
}

// Retrieve retrieves content from a specific layer based on query
func (l *Layers) Retrieve(ctx context.Context, layer Layer, query string) ([]vector.SearchResult, error) {
	config := l.configs[layer]

	// L2 and L3 are search-based
	if layer >= L2 {
		limit := 10
		if layer == L3 {
			limit = 50
		}

		return l.store.Search(ctx, query, "", "", limit)
	}

	// L0 and L1 use room-based retrieval
	var results []vector.SearchResult

	// First try room-based retrieval
	for _, room := range config.Rooms {
		roomResults, err := l.store.Search(ctx, query, "", room, 5)
		if err == nil {
			results = append(results, roomResults...)
		}
	}

	// If insufficient, fall back to search
	if len(results) < 3 && query != "" {
		searchResults, err := l.store.Search(ctx, query, "", "", config.MaxTokens/100)
		if err == nil {
			results = append(results, searchResults...)
		}
	}

	return results, nil
}

// Store stores content in a specific layer
func (l *Layers) Store(ctx context.Context, layer Layer, wing, room, content string) error {
	config := l.configs[layer]

	// Determine room based on layer
	if room == "" {
		switch layer {
		case L0:
			room = "identity"
		case L1:
			room = "current"
		default:
			room = "general"
		}
	}

	doc := vector.Document{
		ID:      fmt.Sprintf("layer%d_%s_%s", layer, wing, room),
		Content: content,
		Metadata: map[string]any{
			"wing":       wing,
			"room":       room,
			"layer":      int(layer),
			"added_by":   "layers",
			"priority":   config.Priority,
			"max_tokens": config.MaxTokens,
		},
	}

	return l.store.Add(ctx, []vector.Document{doc})
}

// GetLayerInfo returns information about all layers
func (l *Layers) GetLayerInfo() map[Layer]LayerConfig {
	return l.configs
}

// AutoClassify attempts to classify content into appropriate layer
func (l *Layers) AutoClassify(content string) Layer {
	contentLower := strings.ToLower(content)

	// L0 keywords: identity, about me, core, essential
	l0Keywords := []string{"i am", "my name", "about me", "my identity", "core belief", "essential"}
	for _, kw := range l0Keywords {
		if strings.Contains(contentLower, kw) {
			return L0
		}
	}

	// L1 keywords: current, recent, project, goal
	l1Keywords := []string{"currently", "recently", "working on", "my project", "my goal", "this week"}
	for _, kw := range l1Keywords {
		if strings.Contains(contentLower, kw) {
			return L1
		}
	}

	// Default to L2 for general content
	return L2
}
