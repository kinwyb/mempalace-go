package searcher

import (
	"context"
	"testing"

	"github.com/kinwyb/mempalace-go/pkg/vector"
)

// mockStore implements vector.Store for testing
type mockStore struct {
	docs []vector.Document
}

func (m *mockStore) Initialize(ctx context.Context) error { return nil }
func (m *mockStore) Add(ctx context.Context, docs []vector.Document) error {
	m.docs = append(m.docs, docs...)
	return nil
}
func (m *mockStore) Delete(ctx context.Context, id string) error               { return nil }
func (m *mockStore) DeleteByWing(ctx context.Context, wing string) error       { return nil }
func (m *mockStore) DeleteByRoom(ctx context.Context, wing, room string) error { return nil }
func (m *mockStore) Close() error                                              { return nil }

func (m *mockStore) Search(ctx context.Context, query, wing, room string, limit int) ([]vector.SearchResult, error) {
	var results []vector.SearchResult
	for _, doc := range m.docs {
		if wing != "" && getTestMetadataString(doc.Metadata, "wing") != wing {
			continue
		}
		if room != "" && getTestMetadataString(doc.Metadata, "room") != room {
			continue
		}
		results = append(results, vector.SearchResult{
			ID:       doc.ID,
			Content:  doc.Content,
			Metadata: doc.Metadata,
			Score:    0.9,
		})
	}
	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func (m *mockStore) SearchByVector(ctx context.Context, vec []float32, wing, room string, limit int) ([]vector.SearchResult, error) {
	return m.Search(ctx, "", wing, room, limit)
}

func (m *mockStore) Get(ctx context.Context, id string) (*vector.Document, error) {
	for _, doc := range m.docs {
		if doc.ID == id {
			return &doc, nil
		}
	}
	return nil, nil
}

func (m *mockStore) Count(ctx context.Context) (int, error)                    { return len(m.docs), nil }
func (m *mockStore) CountByWing(ctx context.Context, wing string) (int, error) { return 0, nil }

func (m *mockStore) GetStats(ctx context.Context) (*vector.StoreStats, error) {
	stats := &vector.StoreStats{
		TotalDocuments: len(m.docs),
		WingRoomCounts: make(map[string]map[string]int),
	}
	for _, doc := range m.docs {
		wing := getTestMetadataString(doc.Metadata, "wing")
		room := getTestMetadataString(doc.Metadata, "room")
		if stats.WingRoomCounts[wing] == nil {
			stats.WingRoomCounts[wing] = make(map[string]int)
		}
		stats.WingRoomCounts[wing][room]++
	}
	return stats, nil
}

func getTestMetadataString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func TestNewSearcher(t *testing.T) {
	store := &mockStore{}
	s := New(store)

	if s == nil {
		t.Fatal("New returned nil")
	}

	if s.limit != 10 {
		t.Errorf("Expected default limit 10, got %d", s.limit)
	}
}

func TestSearch(t *testing.T) {
	store := &mockStore{
		docs: []vector.Document{
			{ID: "1", Content: "Golang content", Metadata: map[string]any{"wing": "tech", "room": "code"}},
			{ID: "2", Content: "Python content", Metadata: map[string]any{"wing": "tech", "room": "code"}},
			{ID: "3", Content: "Other content", Metadata: map[string]any{"wing": "other", "room": "misc"}},
		},
	}

	s := New(store)
	ctx := context.Background()

	resp := s.Search(ctx, "test", "", "", 10)

	if resp.Error != "" {
		t.Errorf("Search error: %s", resp.Error)
	}

	if len(resp.Results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(resp.Results))
	}
}

func TestSearchWithFilters(t *testing.T) {
	store := &mockStore{
		docs: []vector.Document{
			{ID: "1", Content: "Content 1", Metadata: map[string]any{"wing": "tech", "room": "code"}},
			{ID: "2", Content: "Content 2", Metadata: map[string]any{"wing": "tech", "room": "docs"}},
			{ID: "3", Content: "Content 3", Metadata: map[string]any{"wing": "other", "room": "code"}},
		},
	}

	s := New(store)
	ctx := context.Background()

	// Filter by wing
	resp := s.Search(ctx, "test", "tech", "", 10)
	if len(resp.Results) != 2 {
		t.Errorf("Expected 2 results for wing 'tech', got %d", len(resp.Results))
	}

	// Filter by room
	resp = s.Search(ctx, "test", "", "code", 10)
	if len(resp.Results) != 2 {
		t.Errorf("Expected 2 results for room 'code', got %d", len(resp.Results))
	}

	// Filter by both
	resp = s.Search(ctx, "test", "tech", "code", 10)
	if len(resp.Results) != 1 {
		t.Errorf("Expected 1 result for wing 'tech' and room 'code', got %d", len(resp.Results))
	}
}

func TestCalculateSimilarity(t *testing.T) {
	tests := []struct {
		a        string
		b        string
		minScore float64
	}{
		{"golang programming", "golang programming", 0.9},
		{"golang programming", "python programming", 0.3},
		{"", "", 0.0},
		{"short", "completely different text", 0.0},
	}

	for _, tt := range tests {
		score := calculateSimilarity(tt.a, tt.b)
		if score < tt.minScore-0.1 { // Allow some tolerance
			t.Errorf("Similarity(%q, %q) = %f, want >= %f", tt.a, tt.b, score, tt.minScore)
		}
	}
}

func TestTokenize(t *testing.T) {
	text := "Hello World! This is a test."
	tokens := tokenize(text)

	if len(tokens) < 3 {
		t.Errorf("Expected at least 3 tokens, got %d", len(tokens))
	}

	// Should not have punctuation
	for token := range tokens {
		if len(token) <= 2 {
			t.Errorf("Token too short: %s", token)
		}
	}
}

func TestFormatResults(t *testing.T) {
	results := []vector.SearchResult{
		{
			ID:      "test-1",
			Content: "This is test content",
			Score:   0.95,
			Metadata: map[string]any{
				"wing": "test",
				"room": "unit",
			},
		},
	}

	formatted := FormatResults(results)

	if formatted == "" {
		t.Error("FormatResults returned empty string")
	}

	// Should contain the content
	if len(formatted) < 20 {
		t.Errorf("Formatted result seems too short: %s", formatted)
	}
}

func TestSetLimit(t *testing.T) {
	store := &mockStore{}
	s := New(store)

	s.SetLimit(20)

	if s.limit != 20 {
		t.Errorf("Expected limit 20, got %d", s.limit)
	}
}

func TestSetHintEnabled(t *testing.T) {
	store := &mockStore{}
	s := New(store)

	if !s.hintEnabled {
		t.Error("Hints should be enabled by default")
	}

	s.SetHintEnabled(false)

	if s.hintEnabled {
		t.Error("Hints should be disabled")
	}
}
