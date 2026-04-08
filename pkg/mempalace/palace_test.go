package mempalace

import (
	"context"
	"testing"
	"time"

	"github.com/kinwyb/mempalace-go/pkg/embedding"
)

func TestNewWithDefaults(t *testing.T) {
	ctx := context.Background()

	palace, err := New(ctx,
		WithPalacePath(t.TempDir()),
		WithEmbedder(embedding.NewMockEmbedder(768)),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if palace == nil {
		t.Fatal("palace is nil")
	}

	if !palace.IsReady() {
		t.Error("palace should be ready")
	}

	if err := palace.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if palace.IsReady() {
		t.Error("palace should not be ready after close")
	}
}

func TestAddAndSearch(t *testing.T) {
	ctx := context.Background()

	palace := newTestPalace(t)
	defer palace.Close()

	// Add content
	addResult, err := palace.Add(ctx, "Test content about databases and PostgreSQL",
		WithWingForAdd("test"),
		WithRoomForAdd("tech"),
		WithMetadata(map[string]any{"priority": "high"}),
	)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	if addResult.ID == "" {
		t.Error("AddResult should have ID")
	}
	if addResult.Wing != "test" {
		t.Errorf("expected wing 'test', got '%s'", addResult.Wing)
	}
	if addResult.Room != "tech" {
		t.Errorf("expected room 'tech', got '%s'", addResult.Room)
	}

	// Search
	searchResult, err := palace.Search(ctx, "databases",
		WithWing("test"),
		WithLimit(5),
	)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if searchResult.Total == 0 {
		t.Error("Search should return results")
	}

	if len(searchResult.Results) != searchResult.Total {
		t.Errorf("results count mismatch: %d vs %d", len(searchResult.Results), searchResult.Total)
	}
}

func TestAddDocument(t *testing.T) {
	ctx := context.Background()

	palace := newTestPalace(t)
	defer palace.Close()

	doc := Document{
		ID:      "test-doc-1",
		Content: "Important decision: Use Go for backend development",
		Wing:    "testwing",
		Room:    "decisions",
		Source:  "test.go",
		Metadata: map[string]any{
			"date": "2024-01-15",
		},
	}

	result, err := palace.AddDocument(ctx, doc)
	if err != nil {
		t.Fatalf("AddDocument failed: %v", err)
	}

	if result.ID != doc.ID {
		t.Errorf("expected ID '%s', got '%s'", doc.ID, result.ID)
	}

	// Retrieve the document
	retrieved, err := palace.Get(ctx, doc.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.Content != doc.Content {
		t.Errorf("content mismatch")
	}
	if retrieved.Wing != doc.Wing {
		t.Errorf("wing mismatch")
	}
}

func TestAddBatch(t *testing.T) {
	ctx := context.Background()

	palace := newTestPalace(t)
	defer palace.Close()

	docs := []Document{
		{Content: "First document about APIs", Wing: "batch", Room: "api"},
		{Content: "Second document about databases", Wing: "batch", Room: "db"},
		{Content: "Third document about testing", Wing: "batch", Room: "test"},
	}

	results, err := palace.AddBatch(ctx, docs)
	if err != nil {
		t.Fatalf("AddBatch failed: %v", err)
	}

	if len(results) != len(docs) {
		t.Errorf("expected %d results, got %d", len(docs), len(results))
	}

	// Verify all were added
	for i, result := range results {
		if result.ID == "" {
			t.Errorf("result %d has empty ID", i)
		}
	}

	// Check stats
	stats, err := palace.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	if stats.TotalDocuments < len(docs) {
		t.Errorf("expected at least %d documents, got %d", len(docs), stats.TotalDocuments)
	}
}

func TestDuplicateCheck(t *testing.T) {
	ctx := context.Background()

	palace := newTestPalace(t)
	defer palace.Close()

	// Add original content
	_, err := palace.Add(ctx, "Original content about machine learning algorithms",
		WithWingForAdd("dup"),
		WithRoomForAdd("ml"),
	)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Check duplicate with similar content
	dupResult, err := palace.CheckDuplicate(ctx, "Content about machine learning", 0.3)
	if err != nil {
		t.Fatalf("CheckDuplicate failed: %v", err)
	}

	if !dupResult.IsDuplicate {
		t.Error("should detect duplicate")
	}

	if len(dupResult.Matches) == 0 {
		t.Error("should have matches")
	}
}

func TestLayerOperations(t *testing.T) {
	ctx := context.Background()

	palace := newTestPalace(t)
	defer palace.Close()

	// Get layer info
	layerInfo := palace.GetLayerInfo()
	if len(layerInfo) != 4 {
		t.Errorf("expected 4 layers, got %d", len(layerInfo))
	}

	// Store in L0
	err := palace.StoreInLayer(ctx, L0, "I am a Go developer",
		WithWingForLayer("identity"),
	)
	if err != nil {
		t.Fatalf("StoreInLayer failed: %v", err)
	}

	// Wake up
	wakeUp, err := palace.WakeUp(ctx)
	if err != nil {
		t.Fatalf("WakeUp failed: %v", err)
	}

	if wakeUp == "" {
		t.Error("WakeUp should return content")
	}

	// Auto classify - test various content
	layer0 := palace.AutoClassify("I am a backend developer")
	if layer0 != L0 {
		t.Errorf("expected L0 for identity content, got %v", layer0)
	}

	layer1 := palace.AutoClassify("Currently working on the auth module")
	if layer1 != L1 {
		t.Errorf("expected L1 for current work content, got %v", layer1)
	}

	layer2 := palace.AutoClassify("General documentation about APIs")
	if layer2 != L2 {
		t.Errorf("expected L2 for general content, got %v", layer2)
	}
}

func TestDelete(t *testing.T) {
	ctx := context.Background()

	palace := newTestPalace(t)
	defer palace.Close()

	// Add content
	result, err := palace.Add(ctx, "Content to be deleted",
		WithWingForAdd("delete"),
		WithRoomForAdd("temp"),
	)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Delete
	err = palace.Delete(ctx, result.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deleted
	_, err = palace.Get(ctx, result.ID)
	if err == nil {
		t.Error("document should be deleted")
	}
}

func TestGetWingsAndRooms(t *testing.T) {
	ctx := context.Background()

	palace := newTestPalace(t)
	defer palace.Close()

	// Add content to multiple wings/rooms
	_, err := palace.Add(ctx, "API documentation", WithWingForAdd("wing1"), WithRoomForAdd("api"))
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	_, err = palace.Add(ctx, "Database schema", WithWingForAdd("wing1"), WithRoomForAdd("db"))
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	_, err = palace.Add(ctx, "Test cases", WithWingForAdd("wing2"), WithRoomForAdd("test"))
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Get wings
	wings, err := palace.GetWings(ctx)
	if err != nil {
		t.Fatalf("GetWings failed: %v", err)
	}

	if len(wings) < 2 {
		t.Errorf("expected at least 2 wings, got %d", len(wings))
	}

	// Get rooms for a wing
	rooms, err := palace.GetRooms(ctx, "wing1")
	if err != nil {
		t.Fatalf("GetRooms failed: %v", err)
	}

	if len(rooms) < 2 {
		t.Errorf("expected at least 2 rooms, got %d", len(rooms))
	}
}

func TestOptions(t *testing.T) {
	ctx := context.Background()

	tempPath := t.TempDir()

	// Test WithPalacePath
	palace, err := New(ctx,
		WithPalacePath(tempPath),
		WithEmbedder(embedding.NewMockEmbedder(768)),
	)
	if err != nil {
		t.Fatalf("New with WithPalacePath failed: %v", err)
	}

	cfg := palace.GetConfig()
	if cfg.PalacePath != tempPath {
		t.Errorf("expected palace path '%s', got '%s'", tempPath, cfg.PalacePath)
	}

	palace.Close()

	// Test WithChunkSize
	palace, err = New(ctx,
		WithPalacePath(t.TempDir()),
		WithEmbedder(embedding.NewMockEmbedder(768)),
		WithChunkSize(1000, 200, 100),
	)
	if err != nil {
		t.Fatalf("New with WithChunkSize failed: %v", err)
	}

	cfg = palace.GetConfig()
	if cfg.ChunkSize != 1000 {
		t.Errorf("expected chunk size 1000, got %d", cfg.ChunkSize)
	}
	if cfg.ChunkOverlap != 200 {
		t.Errorf("expected chunk overlap 200, got %d", cfg.ChunkOverlap)
	}
	if cfg.MinChunkSize != 100 {
		t.Errorf("expected min chunk size 100, got %d", cfg.MinChunkSize)
	}

	palace.Close()
}

func TestErrors(t *testing.T) {
	ctx := context.Background()

	palace := newTestPalace(t)
	palace.Close() // Close immediately to test closed errors

	// Test closed palace
	_, err := palace.Search(ctx, "test")
	if err == nil {
		t.Error("Search on closed palace should fail")
	}
	if !Is(err, ErrClosed) {
		t.Errorf("expected ErrClosed, got %v", err)
	}

	_, err = palace.Add(ctx, "test")
	if err == nil {
		t.Error("Add on closed palace should fail")
	}
	if !Is(err, ErrClosed) {
		t.Errorf("expected ErrClosed, got %v", err)
	}
}

func TestChunkText(t *testing.T) {
	content := "This is a test paragraph.\n\nThis is another paragraph with more content.\n\nAnd a third one."

	chunks := ChunkText(content, 50, 10, 20)
	if len(chunks) == 0 {
		t.Error("ChunkText should return chunks")
	}

	for i, chunk := range chunks {
		if chunk.Index != i {
			t.Errorf("chunk index mismatch: expected %d, got %d", i, chunk.Index)
		}
		if len(chunk.Content) < 20 {
			t.Errorf("chunk too short: %d", len(chunk.Content))
		}
	}
}

func TestLayerConstants(t *testing.T) {
	tests := []struct {
		layer    Layer
		expected string
	}{
		{L0, "L0-Identity"},
		{L1, "L1-Essential"},
		{L2, "L2-OnDemand"},
		{L3, "L3-DeepSearch"},
	}

	for _, test := range tests {
		if test.layer.String() != test.expected {
			t.Errorf("layer %d: expected '%s', got '%s'", test.layer, test.expected, test.layer.String())
		}
	}
}

func TestSearchOptions(t *testing.T) {
	ctx := context.Background()

	palace := newTestPalace(t)
	defer palace.Close()

	// Add content to specific wing/room
	_, err := palace.Add(ctx, "API endpoint for user authentication",
		WithWingForAdd("myproject"),
		WithRoomForAdd("api"),
	)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Search with wing filter
	result, err := palace.Search(ctx, "authentication",
		WithWing("myproject"),
		WithLimit(5),
	)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if result.Filters.Wing != "myproject" {
		t.Errorf("expected wing filter 'myproject', got '%s'", result.Filters.Wing)
	}

	// Search with wing and room filter
	result, err = palace.Search(ctx, "authentication",
		WithWing("myproject"),
		WithRoom("api"),
		WithLimit(5),
	)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if result.Filters.Room != "api" {
		t.Errorf("expected room filter 'api', got '%s'", result.Filters.Room)
	}
}

// Helper function
func newTestPalace(t *testing.T) *Palace {
	ctx := context.Background()

	palace, err := New(ctx,
		WithPalacePath(t.TempDir()),
		WithEmbedder(embedding.NewMockEmbedder(768)),
		WithSearchDefaults(10, 0.9),
	)
	if err != nil {
		t.Fatalf("Failed to create test palace: %v", err)
	}

	// Give time for initialization
	time.Sleep(100 * time.Millisecond)

	return palace
}