package mempalace

import (
	"time"
)

// SearchResult represents the result of a search operation.
type SearchResult struct {
	Query    string
	Results  []ResultItem
	Total    int
	Hint     string
	Filters  SearchFilters
	Duration time.Duration
}

// ResultItem represents a single search result.
type ResultItem struct {
	ID       string
	Content  string
	Score    float64
	Wing     string
	Room     string
	Source   string
	Metadata map[string]string
}

// SearchFilters represents applied search filters.
type SearchFilters struct {
	Wing string
	Room string
}

// SearchOption is a functional option for search operations.
type SearchOption func(*searchConfig)

type searchConfig struct {
	wing           string
	room           string
	limit          int
	includeContent bool
}

// WithWing filters search to a specific wing.
func WithWing(wing string) SearchOption {
	return func(sc *searchConfig) {
		sc.wing = wing
	}
}

// WithRoom filters search to a specific room.
func WithRoom(room string) SearchOption {
	return func(sc *searchConfig) {
		sc.room = room
	}
}

// WithLimit limits the number of results.
func WithLimit(limit int) SearchOption {
	return func(sc *searchConfig) {
		sc.limit = limit
	}
}

// WithFullContent includes full content in results (no truncation).
func WithFullContent() SearchOption {
	return func(sc *searchConfig) {
		sc.includeContent = true
	}
}

// DuplicateCheckResult represents a duplicate check result.
type DuplicateCheckResult struct {
	IsDuplicate bool
	Matches     []ResultItem
	Score       float64
}