// Package vector provides vector storage interfaces and implementations.
package vector

import (
	"context"
)

// Document represents a document to be stored in the vector store.
type Document struct {
	ID       string
	Content  string
	Metadata map[string]any
}

// SearchResult represents a search result from the vector store.
type SearchResult struct {
	ID       string
	Content  string
	Metadata map[string]any
	Score    float64
}

// Store is the interface for vector storage operations.
type Store interface {
	// Initialize creates the necessary tables/indexes.
	Initialize(ctx context.Context) error

	// Add adds documents to the store.
	Add(ctx context.Context, docs []Document) error

	// Search performs a semantic search.
	Search(ctx context.Context, query string, wing, room string, limit int) ([]SearchResult, error)

	// SearchByVector performs a search using an embedding vector.
	SearchByVector(ctx context.Context, vector []float32, wing, room string, limit int) ([]SearchResult, error)

	// Get retrieves a document by ID.
	Get(ctx context.Context, id string) (*Document, error)

	// Delete removes a document by ID.
	Delete(ctx context.Context, id string) error

	// DeleteByWing removes all documents in a wing.
	DeleteByWing(ctx context.Context, wing string) error

	// DeleteByRoom removes all documents in a wing/room.
	DeleteByRoom(ctx context.Context, wing, room string) error

	// Count returns the total number of documents.
	Count(ctx context.Context) (int, error)

	// CountByWing returns the number of documents in a wing.
	CountByWing(ctx context.Context, wing string) (int, error)

	// Close closes the store connection.
	Close() error

	// GetStats returns statistics about the store.
	GetStats(ctx context.Context) (*StoreStats, error)
}

// StoreStats holds statistics about the vector store.
type StoreStats struct {
	TotalDocuments int
	TotalWings     int
	TotalRooms     int
	WingRoomCounts map[string]map[string]int // wing -> room -> count
	StorageSize    int64
	LastUpdated    string
}
