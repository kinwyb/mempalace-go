package vector

import (
	"context"
	"testing"
)

func TestNewSQLiteStore(t *testing.T) {
	// Create mock embedder
	embedder := NewMockEmbedderForTest(768)

	store, err := NewSQLiteStore(":memory:", embedder)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	if store == nil {
		t.Fatal("Store is nil")
	}
}

func TestInitialize(t *testing.T) {
	embedder := NewMockEmbedderForTest(768)
	store, err := NewSQLiteStore(":memory:", embedder)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	err = store.Initialize(ctx)
	if err != nil {
		t.Fatalf("Failed to initialize store: %v", err)
	}
}

func TestAddAndCount(t *testing.T) {
	embedder := NewMockEmbedderForTest(768)
	store, err := NewSQLiteStore(":memory:", embedder)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	store.Initialize(ctx)

	// Add documents
	docs := []Document{
		{
			ID:      "test-1",
			Content: "This is test content",
			Metadata: map[string]any{
				"wing": "test-wing",
				"room": "test-room",
			},
		},
		{
			ID:      "test-2",
			Content: "Another test content",
			Metadata: map[string]any{
				"wing": "test-wing",
				"room": "test-room",
			},
		},
	}

	err = store.Add(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to add documents: %v", err)
	}

	// Count
	count, err := store.Count(ctx)
	if err != nil {
		t.Fatalf("Failed to count: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected count 2, got %d", count)
	}
}

func TestSearch(t *testing.T) {
	embedder := NewMockEmbedderForTest(768)
	store, err := NewSQLiteStore(":memory:", embedder)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	store.Initialize(ctx)

	// Add test document
	docs := []Document{
		{
			ID:      "test-search-1",
			Content: "Golang is a programming language",
			Metadata: map[string]any{
				"wing": "tech",
				"room": "languages",
			},
		},
	}
	store.Add(ctx, docs)

	// Search
	results, err := store.Search(ctx, "Golang", "", "", 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// Note: Without actual embeddings, search may not find results
	// This test verifies the search function doesn't error
	t.Logf("Search returned %d results", len(results))
}

func TestGet(t *testing.T) {
	embedder := NewMockEmbedderForTest(768)
	store, err := NewSQLiteStore(":memory:", embedder)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	store.Initialize(ctx)

	// Add document
	doc := Document{
		ID:      "test-get-1",
		Content: "Test content for get",
		Metadata: map[string]any{
			"wing": "test",
		},
	}
	store.Add(ctx, []Document{doc})

	// Get
	retrieved, err := store.Get(ctx, "test-get-1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Document not found")
	}

	if retrieved.Content != "Test content for get" {
		t.Errorf("Expected content 'Test content for get', got %s", retrieved.Content)
	}
}

func TestDelete(t *testing.T) {
	embedder := NewMockEmbedderForTest(768)
	store, err := NewSQLiteStore(":memory:", embedder)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	store.Initialize(ctx)

	// Add document
	doc := Document{
		ID:      "test-delete-1",
		Content: "To be deleted",
		Metadata: map[string]any{
			"wing": "test",
		},
	}
	store.Add(ctx, []Document{doc})

	// Verify exists
	count, _ := store.Count(ctx)
	if count != 1 {
		t.Fatalf("Expected count 1 before delete, got %d", count)
	}

	// Delete
	err = store.Delete(ctx, "test-delete-1")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deleted
	count, _ = store.Count(ctx)
	if count != 0 {
		t.Errorf("Expected count 0 after delete, got %d", count)
	}
}

func TestGetStats(t *testing.T) {
	embedder := NewMockEmbedderForTest(768)
	store, err := NewSQLiteStore(":memory:", embedder)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	store.Initialize(ctx)

	// Add some documents
	docs := []Document{
		{
			ID:       "stats-1",
			Content:  "Content 1",
			Metadata: map[string]any{"wing": "wing1", "room": "room1"},
		},
		{
			ID:       "stats-2",
			Content:  "Content 2",
			Metadata: map[string]any{"wing": "wing1", "room": "room2"},
		},
		{
			ID:       "stats-3",
			Content:  "Content 3",
			Metadata: map[string]any{"wing": "wing2", "room": "room1"},
		},
	}
	store.Add(ctx, docs)

	// Get stats
	stats, err := store.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	if stats.TotalDocuments != 3 {
		t.Errorf("Expected 3 documents, got %d", stats.TotalDocuments)
	}

	if stats.TotalWings != 2 {
		t.Errorf("Expected 2 wings, got %d", stats.TotalWings)
	}
}

// MockEmbedder for testing
type MockEmbedder struct {
	dimension int
}

func NewMockEmbedderForTest(dimension int) *MockEmbedder {
	return &MockEmbedder{dimension: dimension}
}

func (e *MockEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	embedding := make([]float32, e.dimension)

	// Simple hash-based embedding generation for variety
	if len(text) > 0 {
		hash := uint32(0)
		for _, c := range text {
			hash = hash*31 + uint32(c)
		}

		// Use hash to seed different values
		for i := range embedding {
			seed := hash + uint32(i)
			// Generate normalized values between -1 and 1
			val := float32((seed % 1000) - 500) / 500.0
			embedding[i] = val
		}

		// Normalize the embedding
		var norm float32
		for _, v := range embedding {
			norm += v * v
		}
		if norm > 0 {
			invNorm := 1.0 / float64(norm)
			for i := range embedding {
				embedding[i] = embedding[i] * float32(invNorm)
			}
		}
	} else {
		// Default embedding for empty text
		for i := range embedding {
			embedding[i] = 0.01
		}
	}

	return embedding, nil
}

func (e *MockEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i := range texts {
		emb, _ := e.Embed(ctx, texts[i])
		embeddings[i] = emb
	}
	return embeddings, nil
}

func (e *MockEmbedder) Dimension() int {
	return e.dimension
}

func (e *MockEmbedder) Model() string {
	return "mock"
}
