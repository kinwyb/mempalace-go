// Package vector provides SQLite-based vector storage.
package vector

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"

	_ "modernc.org/sqlite"
)

// SQLiteStore implements Store using SQLite with FTS5.
type SQLiteStore struct {
	db       *sql.DB
	dbPath   string
	embedder Embedder
	mu       sync.RWMutex
}

// Embedder is the interface for generating embeddings.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
}

// NewSQLiteStore creates a new SQLite vector store.
func NewSQLiteStore(dbPath string, embedder Embedder) (*SQLiteStore, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for better performance
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		slog.Warn("failed to enable WAL mode", "error", err)
	}

	store := &SQLiteStore{
		db:       db,
		dbPath:   dbPath,
		embedder: embedder,
	}

	return store, nil
}

// Initialize creates the necessary tables.
func (s *SQLiteStore) Initialize(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create documents table
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS documents (
			id TEXT PRIMARY KEY,
			content TEXT NOT NULL,
			wing TEXT NOT NULL,
			room TEXT NOT NULL,
			source_file TEXT,
			chunk_index INTEGER,
			added_by TEXT,
			filed_at TEXT,
			ingest_mode TEXT,
			extract_mode TEXT,
			metadata TEXT
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create documents table: %w", err)
	}

	// Create FTS5 virtual table for full-text search
	_, err = s.db.Exec(`
		CREATE VIRTUAL TABLE IF NOT EXISTS documents_fts USING fts5(
			id,
			content,
			wing,
			room,
			content='documents',
			content_rowid='rowid'
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create FTS5 table: %w", err)
	}

	// Create embeddings table (for semantic search)
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS embeddings (
			document_id TEXT PRIMARY KEY,
			embedding BLOB NOT NULL,
			model TEXT,
			created_at TEXT,
			FOREIGN KEY (document_id) REFERENCES documents(id)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create embeddings table: %w", err)
	}

	// Create indexes
	_, err = s.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_documents_wing ON documents(wing);
		CREATE INDEX IF NOT EXISTS idx_documents_room ON documents(room);
		CREATE INDEX IF NOT EXISTS idx_documents_wing_room ON documents(wing, room);
		CREATE INDEX IF NOT EXISTS idx_documents_source ON documents(source_file);
	`)
	if err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	slog.Info("Initialized SQLite store", "path", s.dbPath)
	return nil
}

// Add adds documents to the store.
func (s *SQLiteStore) Add(ctx context.Context, docs []Document) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, doc := range docs {
		wing := getMetadataString(doc.Metadata, "wing")
		room := getMetadataString(doc.Metadata, "room")
		sourceFile := getMetadataString(doc.Metadata, "source_file")
		chunkIndex := getMetadataInt(doc.Metadata, "chunk_index")
		addedBy := getMetadataString(doc.Metadata, "added_by")
		filedAt := getMetadataString(doc.Metadata, "filed_at")
		ingestMode := getMetadataString(doc.Metadata, "ingest_mode")
		extractMode := getMetadataString(doc.Metadata, "extract_mode")
		metadataJSON := metadataToJSON(doc.Metadata)

		// Insert document
		_, err := s.db.Exec(`
			INSERT OR REPLACE INTO documents
			(id, content, wing, room, source_file, chunk_index, added_by, filed_at, ingest_mode, extract_mode, metadata)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, doc.ID, doc.Content, wing, room, sourceFile, chunkIndex, addedBy, filedAt, ingestMode, extractMode, metadataJSON)
		if err != nil {
			slog.Warn("failed to insert document", "id", doc.ID, "error", err)
			continue
		}

		// Update FTS5 index
		_, err = s.db.Exec(`
			INSERT INTO documents_fts (id, content, wing, room)
			VALUES (?, ?, ?, ?)
		`, doc.ID, doc.Content, wing, room)
		if err != nil {
			slog.Warn("failed to update FTS5 index", "id", doc.ID, "error", err)
		}

		// Generate and store embedding if embedder is available
		if s.embedder != nil {
			embedding, err := s.embedder.Embed(ctx, doc.Content)
			if err != nil {
				slog.Warn("failed to generate embedding", "id", doc.ID, "error", err)
				continue
			}

			embeddingBlob := floatsToBlob(embedding)
			_, err = s.db.Exec(`
				INSERT OR REPLACE INTO embeddings (document_id, embedding, model, created_at)
				VALUES (?, ?, ?, datetime('now'))
			`, doc.ID, embeddingBlob, "nomic-embed-text")
			if err != nil {
				slog.Warn("failed to store embedding", "id", doc.ID, "error", err)
			}
		}
	}

	return nil
}

// Search performs a search using FTS5 and optionally semantic similarity.
func (s *SQLiteStore) Search(ctx context.Context, query string, wing, room string, limit int) ([]SearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []SearchResult

	// Build FTS5 query with filters
	ftsQuery := s.buildFTSQuery(query, wing, room)

	// Execute with query parameter and limit
	searchQuery := query
	if searchQuery == "" {
		searchQuery = "*" // Match all if no query
	}
	rows, err := s.db.Query(ftsQuery, searchQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var result SearchResult
		var metadataJSON string

		err := rows.Scan(&result.ID, &result.Content, &result.Score, &metadataJSON)
		if err != nil {
			slog.Warn("failed to scan search result", "error", err)
			continue
		}

		// BM25 returns negative scores (lower is better), convert to positive
		if result.Score < 0 {
			result.Score = -result.Score
		}

		result.Metadata = jsonToMetadata(metadataJSON)
		results = append(results, result)
	}

	// If we have an embedder and want semantic search, enhance results
	if s.embedder != nil && len(query) > 0 {
		semanticResults, err := s.semanticSearch(ctx, query, wing, room, limit)
		if err == nil && len(semanticResults) > 0 {
			// Merge and rank results
			results = s.mergeResults(results, semanticResults)
			if len(results) > limit {
				results = results[:limit]
			}
		}
	}

	return results, nil
}

// SearchByVector performs a search using an embedding vector.
func (s *SQLiteStore) SearchByVector(ctx context.Context, vector []float32, wing, room string, limit int) ([]SearchResult, error) {
	return s.semanticSearchByVector(vector, wing, room, limit)
}

// semanticSearch performs semantic similarity search.
func (s *SQLiteStore) semanticSearch(ctx context.Context, query string, wing, room string, limit int) ([]SearchResult, error) {
	embedding, err := s.embedder.Embed(ctx, query)
	if err != nil {
		return nil, err
	}
	return s.semanticSearchByVector(embedding, wing, room, limit)
}

// semanticSearchByVector performs semantic search using a precomputed vector.
func (s *SQLiteStore) semanticSearchByVector(vector []float32, wing, room string, limit int) ([]SearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
		SELECT d.id, d.content, e.embedding, d.metadata
		FROM documents d
		JOIN embeddings e ON d.id = e.document_id
		WHERE 1=1
	`
	args := []any{}

	if wing != "" {
		query += " AND d.wing = ?"
		args = append(args, wing)
	}
	if room != "" {
		query += " AND d.room = ?"
		args = append(args, room)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candidates []SearchResult
	for rows.Next() {
		var result SearchResult
		var embeddingBlob []byte
		var metadataJSON string

		err := rows.Scan(&result.ID, &result.Content, &embeddingBlob, &metadataJSON)
		if err != nil {
			continue
		}

		docEmbedding := blobToFloats(embeddingBlob)
		result.Score = cosineSimilarity(vector, docEmbedding)
		result.Metadata = jsonToMetadata(metadataJSON)
		candidates = append(candidates, result)
	}

	// Sort by score descending
	sortByScore(candidates)

	if len(candidates) > limit {
		candidates = candidates[:limit]
	}

	return candidates, nil
}

// Get retrieves a document by ID.
func (s *SQLiteStore) Get(ctx context.Context, id string) (*Document, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var doc Document
	var metadataJSON string

	err := s.db.QueryRow(`
		SELECT id, content, metadata FROM documents WHERE id = ?
	`, id).Scan(&doc.ID, &doc.Content, &metadataJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	doc.Metadata = jsonToMetadata(metadataJSON)
	return &doc, nil
}

// Delete removes a document by ID.
func (s *SQLiteStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec("DELETE FROM documents WHERE id = ?", id)
	if err != nil {
		return err
	}

	_, err = s.db.Exec("DELETE FROM embeddings WHERE document_id = ?", id)
	if err != nil {
		return err
	}

	_, err = s.db.Exec("DELETE FROM documents_fts WHERE id = ?", id)
	return err
}

// DeleteByWing removes all documents in a wing.
func (s *SQLiteStore) DeleteByWing(ctx context.Context, wing string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get IDs first for FTS5 deletion
	rows, err := s.db.Query("SELECT id FROM documents WHERE wing = ?", wing)
	if err != nil {
		return err
	}
	var ids []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		ids = append(ids, id)
	}
	rows.Close()

	_, err = s.db.Exec("DELETE FROM documents WHERE wing = ?", wing)
	if err != nil {
		return err
	}

	_, err = s.db.Exec("DELETE FROM embeddings WHERE document_id NOT IN (SELECT id FROM documents)")
	if err != nil {
		return err
	}

	// Delete from FTS5 index
	for _, id := range ids {
		s.db.Exec("DELETE FROM documents_fts WHERE id = ?", id)
	}
	return nil
}

// DeleteByRoom removes all documents in a wing/room.
func (s *SQLiteStore) DeleteByRoom(ctx context.Context, wing, room string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get IDs first for FTS5 deletion
	rows, err := s.db.Query("SELECT id FROM documents WHERE wing = ? AND room = ?", wing, room)
	if err != nil {
		return err
	}
	var ids []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		ids = append(ids, id)
	}
	rows.Close()

	_, err = s.db.Exec("DELETE FROM documents WHERE wing = ? AND room = ?", wing, room)
	if err != nil {
		return err
	}

	_, err = s.db.Exec("DELETE FROM embeddings WHERE document_id NOT IN (SELECT id FROM documents)")
	if err != nil {
		return err
	}

	// Delete from FTS5 index
	for _, id := range ids {
		s.db.Exec("DELETE FROM documents_fts WHERE id = ?", id)
	}
	return nil
}

// Count returns the total number of documents.
func (s *SQLiteStore) Count(ctx context.Context) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM documents").Scan(&count)
	return count, err
}

// CountByWing returns the number of documents in a wing.
func (s *SQLiteStore) CountByWing(ctx context.Context, wing string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM documents WHERE wing = ?", wing).Scan(&count)
	return count, err
}

// Close closes the database connection.
func (s *SQLiteStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db.Close()
}

// GetStats returns statistics about the store.
func (s *SQLiteStore) GetStats(ctx context.Context) (*StoreStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &StoreStats{
		WingRoomCounts: make(map[string]map[string]int),
	}

	// Total documents
	err := s.db.QueryRow("SELECT COUNT(*) FROM documents").Scan(&stats.TotalDocuments)
	if err != nil {
		return nil, err
	}

	// Wing and room counts
	rows, err := s.db.Query(`
		SELECT wing, room, COUNT(*) as count
		FROM documents
		GROUP BY wing, room
		ORDER BY wing, room
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	wings := make(map[string]bool)
	for rows.Next() {
		var wing, room string
		var count int
		if err := rows.Scan(&wing, &room, &count); err != nil {
			continue
		}

		wings[wing] = true
		if stats.WingRoomCounts[wing] == nil {
			stats.WingRoomCounts[wing] = make(map[string]int)
		}
		stats.WingRoomCounts[wing][room] = count
	}

	stats.TotalWings = len(wings)
	stats.TotalRooms = 0
	for _, rooms := range stats.WingRoomCounts {
		stats.TotalRooms += len(rooms)
	}

	// Storage size
	var size int64
	info, err := os.Stat(s.dbPath)
	if err == nil {
		size = info.Size()
	}
	stats.StorageSize = size

	return stats, nil
}

// buildFTSQuery builds a FTS5 search query.
func (s *SQLiteStore) buildFTSQuery(query, wing, room string) string {
	baseQuery := `
		SELECT d.id, d.content, bm25(documents_fts) as score, d.metadata
		FROM documents d
		JOIN documents_fts fts ON d.id = fts.id
		WHERE documents_fts MATCH ?
	`

	if wing != "" {
		baseQuery += " AND d.wing = '" + wing + "'"
	}
	if room != "" {
		baseQuery += " AND d.room = '" + room + "'"
	}

	baseQuery += " ORDER BY score LIMIT ?"

	return baseQuery
}

// mergeResults merges FTS and semantic results.
func (s *SQLiteStore) mergeResults(ftsResults, semanticResults []SearchResult) []SearchResult {
	seen := make(map[string]bool)
	var merged []SearchResult

	// Add semantic results first (higher priority)
	for _, r := range semanticResults {
		if !seen[r.ID] {
			merged = append(merged, r)
			seen[r.ID] = true
		}
	}

	// Add FTS results
	for _, r := range ftsResults {
		if !seen[r.ID] {
			merged = append(merged, r)
			seen[r.ID] = true
		}
	}

	return merged
}

// Helper functions

func getMetadataString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getMetadataInt(m map[string]any, key string) int {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case int:
			return n
		case float64:
			return int(n)
		}
	}
	return 0
}

func metadataToJSON(m map[string]any) string {
	if m == nil {
		return "{}"
	}
	// Simple JSON encoding
	var parts []string
	for k, v := range m {
		switch val := v.(type) {
		case string:
			parts = append(parts, fmt.Sprintf("\"%s\":\"%s\"", k, val))
		case int:
			parts = append(parts, fmt.Sprintf("\"%s\":%d", k, val))
		case float64:
			parts = append(parts, fmt.Sprintf("\"%s\":%f", k, val))
		default:
			parts = append(parts, fmt.Sprintf("\"%s\":\"%v\"", k, val))
		}
	}
	return "{" + strings.Join(parts, ",") + "}"
}

func jsonToMetadata(s string) map[string]any {
	if s == "" || s == "{}" {
		return make(map[string]any)
	}

	result := make(map[string]any)
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		slog.Warn("failed to parse metadata JSON", "error", err, "json", s)
		return make(map[string]any)
	}
	return result
}

func floatsToBlob(f []float32) []byte {
	blob := make([]byte, len(f)*4)
	for i, v := range f {
		bits := math.Float32bits(v)
		blob[i*4] = byte(bits >> 24)
		blob[i*4+1] = byte(bits >> 16)
		blob[i*4+2] = byte(bits >> 8)
		blob[i*4+3] = byte(bits)
	}
	return blob
}

func blobToFloats(b []byte) []float32 {
	count := len(b) / 4
	f := make([]float32, count)
	for i := 0; i < count; i++ {
		bits := uint32(b[i*4])<<24 | uint32(b[i*4+1])<<16 | uint32(b[i*4+2])<<8 | uint32(b[i*4+3])
		f[i] = math.Float32frombits(bits)
	}
	return f
}

func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (sqrt(normA) * sqrt(normB))
}

func sqrt(x float64) float64 {
	if x < 0 {
		return 0
	}
	// Newton's method
	z := x
	for i := 0; i < 10; i++ {
		z = (z + x/z) / 2
	}
	return z
}

func sortByScore(results []SearchResult) {
	// Simple sort implementation
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
}
