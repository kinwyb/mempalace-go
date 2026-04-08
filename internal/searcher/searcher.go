// Package searcher provides semantic search functionality.
package searcher

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/kinwyb/mempalace-go/pkg/vector"
)

// SearchResponse represents a search response
type SearchResponse struct {
	Query   string
	Filters map[string]string
	Results []vector.SearchResult
	Error   string
	Hint    string
}

// DuplicateCheckResponse represents a duplicate check response
type DuplicateCheckResponse struct {
	IsDuplicate bool
	Matches     []vector.SearchResult
}

// Searcher provides search functionality
type Searcher struct {
	store       vector.Store
	limit       int
	hintEnabled bool
}

// New creates a new Searcher
func New(store vector.Store) *Searcher {
	return &Searcher{
		store:       store,
		limit:       10,
		hintEnabled: true,
	}
}

// SetLimit sets the default search limit
func (s *Searcher) SetLimit(limit int) *Searcher {
	s.limit = limit
	return s
}

// SetHintEnabled enables/disables search hints
func (s *Searcher) SetHintEnabled(enabled bool) *Searcher {
	s.hintEnabled = enabled
	return s
}

// Search performs a semantic search
func (s *Searcher) Search(ctx context.Context, query, wing, room string, limit int) *SearchResponse {
	if limit == 0 {
		limit = s.limit
	}

	results, err := s.store.Search(ctx, query, wing, room, limit)
	if err != nil {
		slog.Warn("search failed", "query", query, "error", err)
		return &SearchResponse{
			Query: query,
			Error: err.Error(),
		}
	}

	resp := &SearchResponse{
		Query:   query,
		Results: results,
		Filters: map[string]string{},
	}

	if wing != "" {
		resp.Filters["wing"] = wing
	}
	if room != "" {
		resp.Filters["room"] = room
	}

	// Generate hint if enabled
	if s.hintEnabled && len(results) > 0 {
		resp.Hint = s.generateHint(query, results)
	}

	slog.Info("Search completed",
		"query", query,
		"results", len(results),
		"wing", wing,
		"room", room,
	)

	return resp
}

// CheckDuplicate checks if content is similar to existing content
func (s *Searcher) CheckDuplicate(ctx context.Context, content string, threshold float64) (*DuplicateCheckResponse, error) {
	// Search for similar content
	results, err := s.store.Search(ctx, content, "", "", 5)
	if err != nil {
		return nil, err
	}

	var matches []vector.SearchResult
	for _, result := range results {
		// Calculate similarity score
		similarity := calculateSimilarity(content, result.Content)
		if similarity >= threshold {
			matches = append(matches, result)
		}
	}

	return &DuplicateCheckResponse{
		IsDuplicate: len(matches) > 0,
		Matches:     matches,
	}, nil
}

// GetWings returns all unique wings in the store
func (s *Searcher) GetWings(ctx context.Context) ([]string, error) {
	stats, err := s.store.GetStats(ctx)
	if err != nil {
		return nil, err
	}

	var wings []string
	for wing := range stats.WingRoomCounts {
		wings = append(wings, wing)
	}
	return wings, nil
}

// GetRooms returns all rooms for a wing
func (s *Searcher) GetRooms(ctx context.Context, wing string) ([]string, error) {
	stats, err := s.store.GetStats(ctx)
	if err != nil {
		return nil, err
	}

	if stats.WingRoomCounts[wing] == nil {
		return nil, nil
	}

	var rooms []string
	for room := range stats.WingRoomCounts[wing] {
		rooms = append(rooms, room)
	}
	return rooms, nil
}

// generateHint generates a search hint based on results
func (s *Searcher) generateHint(query string, results []vector.SearchResult) string {
	if len(results) == 0 {
		return "No results found. Try different keywords or remove filters."
	}

	// Check if results are from different wings
	wings := make(map[string]int)
	for _, r := range results {
		if wing := getMetadataString(r.Metadata, "wing"); wing != "" {
			wings[wing]++
		}
	}

	if len(wings) > 1 {
		return "Results span multiple wings. Use --wing to filter."
	}

	// Check room diversity
	rooms := make(map[string]int)
	for _, r := range results {
		if room := getMetadataString(r.Metadata, "room"); room != "" {
			rooms[room]++
		}
	}

	if len(rooms) > 2 {
		return "Results span multiple rooms. Use --room to filter."
	}

	return ""
}

// calculateSimilarity calculates text similarity (simple Jaccard)
func calculateSimilarity(a, b string) float64 {
	wordsA := tokenize(strings.ToLower(a))
	wordsB := tokenize(strings.ToLower(b))

	intersection := 0
	for w := range wordsA {
		if wordsB[w] {
			intersection++
		}
	}

	union := len(wordsA) + len(wordsB) - intersection
	if union == 0 {
		return 0
	}

	return float64(intersection) / float64(union)
}

func tokenize(text string) map[string]bool {
	words := strings.Fields(text)
	result := make(map[string]bool)
	for _, w := range words {
		w = strings.TrimFunc(w, func(r rune) bool {
			return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'))
		})
		if len(w) > 2 {
			result[w] = true
		}
	}
	return result
}

func getMetadataString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// FormatResults formats search results for display
func FormatResults(results []vector.SearchResult) string {
	var builder strings.Builder

	for i, result := range results {
		builder.WriteString(fmt.Sprintf("\n--- Result %d (score: %.3f) ---\n", i+1, result.Score))

		wing := getMetadataString(result.Metadata, "wing")
		room := getMetadataString(result.Metadata, "room")
		source := getMetadataString(result.Metadata, "source_file")

		if wing != "" || room != "" {
			builder.WriteString(fmt.Sprintf("Location: %s/%s\n", wing, room))
		}
		if source != "" {
			builder.WriteString(fmt.Sprintf("Source: %s\n", source))
		}

		builder.WriteString("\n")
		// Truncate content if too long
		content := result.Content
		if len(content) > 500 {
			content = content[:500] + "..."
		}
		builder.WriteString(content)
		builder.WriteString("\n")
	}

	return builder.String()
}
