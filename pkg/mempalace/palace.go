// Package mempalace provides a public Go API for MemPalace operations.
// It allows Go projects to integrate MemPalace functionality without using MCP.
package mempalace

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/kinwyb/mempalace-go/internal/config"
	"github.com/kinwyb/mempalace-go/internal/layers"
	"github.com/kinwyb/mempalace-go/internal/miner"
	"github.com/kinwyb/mempalace-go/internal/palace"
	"github.com/kinwyb/mempalace-go/internal/searcher"
	"github.com/kinwyb/mempalace-go/pkg/embedding"
	"github.com/kinwyb/mempalace-go/pkg/vector"
)

// Palace is the main facade for MemPalace operations.
// It provides a unified interface for storing, searching, and mining content.
type Palace struct {
	store    vector.Store
	embedder embedding.Embedder
	config   *Config
	searcher *searcher.Searcher
	miner    *miner.Miner
	layers   *layers.Layers
	palace   *palace.Palace
	closed   bool
}

// New creates a new Palace instance with the given options.
// Default configuration is used if no options are provided.
func New(ctx context.Context, opts ...Option) (*Palace, error) {
	// Start with default config
	pc := &palaceConfig{
		config: DefaultConfig(),
	}

	// Apply options
	for _, opt := range opts {
		if err := opt(pc); err != nil {
			return nil, err
		}
	}

	// Ensure palace path exists
	if err := os.MkdirAll(pc.config.PalacePath, 0755); err != nil {
		return nil, NewError(ErrStoreInit, "failed to create palace directory", err)
	}

	// Create embedder if not provided via options
	if pc.embedder == nil {
		switch pc.config.EmbeddingProvider {
		case "openai":
			pc.embedder = embedding.NewOpenAIEmbedder(
				pc.config.OpenAIAPIKey,
				pc.config.EmbeddingAPIBase,
				pc.config.EmbeddingModel,
			)
		default:
			pc.embedder = embedding.NewOllamaEmbedder(
				pc.config.OllamaHost,
				pc.config.EmbeddingModel,
			)
		}
	}

	// Create vector store
	dbPath := pc.config.PalacePath + "/palace.db"
	store, err := vector.NewSQLiteStore(dbPath, pc.embedder)
	if err != nil {
		return nil, NewError(ErrStoreInit, "failed to create store", err)
	}

	// Initialize store
	if err := store.Initialize(ctx); err != nil {
		return nil, NewError(ErrStoreInit, "failed to initialize store", err)
	}

	// Create internal components
	internalCfg := convertConfig(pc.config)
	srch := searcher.New(store)
	mnr := miner.New(internalCfg, store)
	lyrs := layers.New(store)
	pal := palace.New(store)

	slog.Debug("Palace initialized",
		"path", pc.config.PalacePath,
		"provider", pc.config.EmbeddingProvider,
		"model", pc.config.EmbeddingModel,
	)

	return &Palace{
		store:    store,
		embedder: pc.embedder,
		config:   pc.config,
		searcher: srch,
		miner:    mnr,
		layers:   lyrs,
		palace:   pal,
	}, nil
}

// NewFromFile creates a Palace from a configuration file.
func NewFromFile(ctx context.Context, configPath string) (*Palace, error) {
	return New(ctx, WithConfigFile(configPath))
}

// NewWithDefaults creates a Palace with default configuration.
func NewWithDefaults(ctx context.Context) (*Palace, error) {
	return New(ctx)
}

// Search performs a semantic search in the palace.
func (p *Palace) Search(ctx context.Context, query string, opts ...SearchOption) (*SearchResult, error) {
	if p.closed {
		return nil, NewError(ErrClosed, "palace is closed", nil)
	}

	start := time.Now()

	// Apply search options
	sc := &searchConfig{}
	for _, opt := range opts {
		opt(sc)
	}

	limit := sc.limit
	if limit == 0 {
		limit = p.config.SearchLimit
	}

	// Perform search
	results, err := p.store.Search(ctx, query, sc.wing, sc.room, limit)
	if err != nil {
		return nil, NewError(ErrSearch, "search failed", err)
	}

	// Convert results
	items := make([]ResultItem, len(results))
	for i, r := range results {
		items[i] = convertSearchResult(r)
	}

	duration := time.Since(start)

	slog.Debug("Search completed",
		"query", query,
		"results", len(results),
		"duration", duration,
	)

	return &SearchResult{
		Query:    query,
		Results:  items,
		Total:    len(results),
		Filters:  SearchFilters{Wing: sc.wing, Room: sc.room},
		Duration: duration,
	}, nil
}

// Get retrieves a document by its ID.
func (p *Palace) Get(ctx context.Context, id string) (*Document, error) {
	if p.closed {
		return nil, NewError(ErrClosed, "palace is closed", nil)
	}

	doc, err := p.store.Get(ctx, id)
	if err != nil {
		return nil, NewError(ErrNotFound, "document not found", err)
	}
	if doc == nil {
		return nil, NewError(ErrNotFound, "document not found", nil)
	}

	return convertDocument(doc), nil
}

// CheckDuplicate checks if similar content already exists.
func (p *Palace) CheckDuplicate(ctx context.Context, content string, threshold float64) (*DuplicateCheckResult, error) {
	if p.closed {
		return nil, NewError(ErrClosed, "palace is closed", nil)
	}

	results, err := p.store.Search(ctx, content, "", "", 5)
	if err != nil {
		return nil, NewError(ErrSearch, "duplicate check failed", err)
	}

	var matches []ResultItem
	maxScore := 0.0
	for _, r := range results {
		// Simple Jaccard similarity
		similarity := calculateSimilarity(content, r.Content)
		if similarity >= threshold {
			matches = append(matches, convertSearchResult(r))
			if similarity > maxScore {
				maxScore = similarity
			}
		}
	}

	return &DuplicateCheckResult{
		IsDuplicate: len(matches) > 0,
		Matches:     matches,
		Score:       maxScore,
	}, nil
}

// Add stores content in the palace with optional configuration.
func (p *Palace) Add(ctx context.Context, content string, opts ...AddOption) (*AddResult, error) {
	if p.closed {
		return nil, NewError(ErrClosed, "palace is closed", nil)
	}

	// Apply add options
	ac := &addConfig{
		wing: "default",
		room: "general",
	}
	for _, opt := range opts {
		opt(ac)
	}

	// Check duplicate if requested
	if ac.checkDup {
		dupResult, err := p.CheckDuplicate(ctx, content, ac.dupThreshold)
		if err != nil {
			return nil, err
		}
		if dupResult.IsDuplicate {
			return nil, NewError(ErrDuplicate, "similar content already exists", nil)
		}
	}

	// Generate ID
	id := generateID(ac.wing, ac.room, content)

	// Create document
	doc := vector.Document{
		ID:      id,
		Content: content,
		Metadata: map[string]any{
			"wing":      ac.wing,
			"room":      ac.room,
			"source":    ac.source,
			"added_by":  "mempalace-api",
			"filed_at":  time.Now().Format(time.RFC3339),
			"layer":     int(ac.layer),
		},
	}

	// Add metadata
	for k, v := range ac.metadata {
		doc.Metadata[k] = v
	}

	// Store document
	if err := p.store.Add(ctx, []vector.Document{doc}); err != nil {
		return nil, NewError(ErrAdd, "failed to add document", err)
	}

	slog.Debug("Document added",
		"id", id,
		"wing", ac.wing,
		"room", ac.room,
	)

	return &AddResult{
		ID:        id,
		Wing:      ac.wing,
		Room:      ac.room,
		CreatedAt: time.Now(),
	}, nil
}

// AddDocument stores a document directly.
func (p *Palace) AddDocument(ctx context.Context, doc Document) (*AddResult, error) {
	if p.closed {
		return nil, NewError(ErrClosed, "palace is closed", nil)
	}

	vDoc := vector.Document{
		ID:       doc.ID,
		Content:  doc.Content,
		Metadata: doc.Metadata,
	}

	if vDoc.Metadata == nil {
		vDoc.Metadata = map[string]any{}
	}

	if doc.Wing != "" {
		vDoc.Metadata["wing"] = doc.Wing
	}
	if doc.Room != "" {
		vDoc.Metadata["room"] = doc.Room
	}
	if doc.Source != "" {
		vDoc.Metadata["source"] = doc.Source
	}
	vDoc.Metadata["added_by"] = "mempalace-api"
	vDoc.Metadata["filed_at"] = time.Now().Format(time.RFC3339)

	if doc.ID == "" {
		vDoc.ID = generateID(doc.Wing, doc.Room, doc.Content)
	}

	if err := p.store.Add(ctx, []vector.Document{vDoc}); err != nil {
		return nil, NewError(ErrAdd, "failed to add document", err)
	}

	wing := doc.Wing
	if wing == "" {
		wing = "default"
	}
	room := doc.Room
	if room == "" {
		room = "general"
	}

	return &AddResult{
		ID:        vDoc.ID,
		Wing:      wing,
		Room:      room,
		CreatedAt: time.Now(),
	}, nil
}

// AddBatch stores multiple documents at once.
func (p *Palace) AddBatch(ctx context.Context, docs []Document) ([]AddResult, error) {
	if p.closed {
		return nil, NewError(ErrClosed, "palace is closed", nil)
	}

	vDocs := make([]vector.Document, len(docs))
	results := make([]AddResult, len(docs))

	for i, doc := range docs {
		vDocs[i] = vector.Document{
			ID:       doc.ID,
			Content:  doc.Content,
			Metadata: doc.Metadata,
		}
		if vDocs[i].Metadata == nil {
			vDocs[i].Metadata = map[string]any{}
		}
		if doc.Wing != "" {
			vDocs[i].Metadata["wing"] = doc.Wing
		}
		if doc.Room != "" {
			vDocs[i].Metadata["room"] = doc.Room
		}
		if doc.Source != "" {
			vDocs[i].Metadata["source"] = doc.Source
		}
		vDocs[i].Metadata["added_by"] = "mempalace-api"
		vDocs[i].Metadata["filed_at"] = time.Now().Format(time.RFC3339)

		if doc.ID == "" {
			vDocs[i].ID = generateID(doc.Wing, doc.Room, doc.Content)
		}

		wing := doc.Wing
		if wing == "" {
			wing = "default"
		}
		room := doc.Room
		if room == "" {
			room = "general"
		}

		results[i] = AddResult{
			ID:        vDocs[i].ID,
			Wing:      wing,
			Room:      room,
			CreatedAt: time.Now(),
		}
	}

	if err := p.store.Add(ctx, vDocs); err != nil {
		return nil, NewError(ErrAdd, "failed to add batch", err)
	}

	return results, nil
}

// Delete removes a document by ID.
func (p *Palace) Delete(ctx context.Context, id string) error {
	if p.closed {
		return NewError(ErrClosed, "palace is closed", nil)
	}

	if err := p.store.Delete(ctx, id); err != nil {
		return NewError(ErrNotFound, "failed to delete document", err)
	}

	return nil
}

// DeleteByWing removes all documents in a wing.
func (p *Palace) DeleteByWing(ctx context.Context, wing string) error {
	if p.closed {
		return NewError(ErrClosed, "palace is closed", nil)
	}

	return p.store.DeleteByWing(ctx, wing)
}

// DeleteByRoom removes all documents in a wing/room.
func (p *Palace) DeleteByRoom(ctx context.Context, wing, room string) error {
	if p.closed {
		return NewError(ErrClosed, "palace is closed", nil)
	}

	return p.store.DeleteByRoom(ctx, wing, room)
}

// WakeUp generates the L0 + L1 wake-up context.
func (p *Palace) WakeUp(ctx context.Context) (string, error) {
	if p.closed {
		return "", NewError(ErrClosed, "palace is closed", nil)
	}

	content, err := p.layers.WakeUp(ctx)
	if err != nil {
		return "", NewError(ErrSearch, "wake-up failed", err)
	}

	return content, nil
}

// StoreInLayer stores content in a specific layer.
func (p *Palace) StoreInLayer(ctx context.Context, layer Layer, content string, opts ...LayerOption) error {
	if p.closed {
		return NewError(ErrClosed, "palace is closed", nil)
	}

	lc := &layerConfig{
		wing: "identity",
	}
	for _, opt := range opts {
		opt(lc)
	}

	// Determine room based on layer
	room := lc.room
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

	return p.layers.Store(ctx, layers.Layer(layer), lc.wing, room, content)
}

// RetrieveFromLayer retrieves content from a specific layer.
func (p *Palace) RetrieveFromLayer(ctx context.Context, layer Layer, query string) (*SearchResult, error) {
	if p.closed {
		return nil, NewError(ErrClosed, "palace is closed", nil)
	}

	results, err := p.layers.Retrieve(ctx, layers.Layer(layer), query)
	if err != nil {
		return nil, NewError(ErrSearch, "layer retrieval failed", err)
	}

	items := make([]ResultItem, len(results))
	for i, r := range results {
		items[i] = convertSearchResult(r)
	}

	return &SearchResult{
		Query:   query,
		Results: items,
		Total:   len(results),
	}, nil
}

// AutoClassify automatically determines the appropriate layer for content.
func (p *Palace) AutoClassify(content string) Layer {
	return Layer(p.layers.AutoClassify(content))
}

// GetLayerInfo returns information about all layers.
func (p *Palace) GetLayerInfo() map[Layer]LayerInfo {
	internalInfo := p.layers.GetLayerInfo()
	info := make(map[Layer]LayerInfo)
	for l, cfg := range internalInfo {
		info[Layer(l)] = LayerInfo{
			Name:        cfg.Name,
			Description: cfg.Description,
			MaxTokens:   cfg.MaxTokens,
			Priority:    cfg.Priority,
			Rooms:       cfg.Rooms,
		}
	}
	return info
}

// GetStats returns palace statistics.
func (p *Palace) GetStats(ctx context.Context) (*Stats, error) {
	if p.closed {
		return nil, NewError(ErrClosed, "palace is closed", nil)
	}

	stats, err := p.palace.GetStats(ctx)
	if err != nil {
		return nil, NewError(ErrSearch, "failed to get stats", err)
	}

	return convertStats(stats), nil
}

// GetWings returns all wings in the palace.
func (p *Palace) GetWings(ctx context.Context) ([]string, error) {
	if p.closed {
		return nil, NewError(ErrClosed, "palace is closed", nil)
	}

	return p.searcher.GetWings(ctx)
}

// GetRooms returns all rooms for a wing.
func (p *Palace) GetRooms(ctx context.Context, wing string) ([]string, error) {
	if p.closed {
		return nil, NewError(ErrClosed, "palace is closed", nil)
	}

	return p.searcher.GetRooms(ctx, wing)
}

// Close closes the palace and releases resources.
func (p *Palace) Close() error {
	if p.closed {
		return nil
	}

	p.closed = true
	return p.store.Close()
}

// IsReady checks if the palace is properly initialized.
func (p *Palace) IsReady() bool {
	return !p.closed
}

// GetConfig returns the current configuration.
func (p *Palace) GetConfig() *Config {
	return p.config
}

// Helper functions

func convertConfig(c *Config) *config.Config {
	return &config.Config{
		PalacePath:          c.PalacePath,
		EmbeddingProvider:   c.EmbeddingProvider,
		EmbeddingModel:      c.EmbeddingModel,
		EmbeddingAPIBase:    c.EmbeddingAPIBase,
		OpenAIAPIKey:        c.OpenAIAPIKey,
		OllamaHost:          c.OllamaHost,
		LogLevel:            c.LogLevel,
		ChunkSize:           c.ChunkSize,
		ChunkOverlap:        c.ChunkOverlap,
		MinChunkSize:        c.MinChunkSize,
		SearchLimit:         c.SearchLimit,
		SimilarityThreshold: c.SimilarityThreshold,
	}
}

func convertSearchResult(r vector.SearchResult) ResultItem {
	wing := ""
	room := ""
	source := ""
	meta := make(map[string]string)

	if r.Metadata != nil {
		if v, ok := r.Metadata["wing"]; ok {
			if s, ok := v.(string); ok {
				wing = s
			}
		}
		if v, ok := r.Metadata["room"]; ok {
			if s, ok := v.(string); ok {
				room = s
			}
		}
		if v, ok := r.Metadata["source_file"]; ok {
			if s, ok := v.(string); ok {
				source = s
			}
		}
		if v, ok := r.Metadata["source"]; ok {
			if s, ok := v.(string); ok {
				source = s
			}
		}
		for k, v := range r.Metadata {
			if s, ok := v.(string); ok {
				meta[k] = s
			}
		}
	}

	return ResultItem{
		ID:       r.ID,
		Content:  r.Content,
		Score:    r.Score,
		Wing:     wing,
		Room:     room,
		Source:   source,
		Metadata: meta,
	}
}

func convertDocument(d *vector.Document) *Document {
	if d == nil {
		return nil
	}
	doc := &Document{
		ID:      d.ID,
		Content: d.Content,
	}

	if d.Metadata != nil {
		if v, ok := d.Metadata["wing"]; ok {
			if s, ok := v.(string); ok {
				doc.Wing = s
			}
		}
		if v, ok := d.Metadata["room"]; ok {
			if s, ok := v.(string); ok {
				doc.Room = s
			}
		}
		if v, ok := d.Metadata["source_file"]; ok {
			if s, ok := v.(string); ok {
				doc.Source = s
			}
		}
		if v, ok := d.Metadata["source"]; ok {
			if s, ok := v.(string); ok {
				doc.Source = s
			}
		}
		doc.Metadata = d.Metadata
	}

	return doc
}

func convertStats(s *palace.PalaceStats) *Stats {
	wings := make(map[string]WingStats)
	for wing, ws := range s.Wings {
		wingStats := WingStats{
			Name:      wing,
			Rooms:     ws.Rooms,
			Total:     ws.Total,
			RoomCount: len(ws.Rooms),
		}
		wings[wing] = wingStats
	}

	return &Stats{
		TotalDocuments: s.TotalDrawers,
		TotalWings:     s.TotalWings,
		TotalRooms:     s.TotalRooms,
		StorageSize:    s.StorageSize,
		Wings:          wings,
	}
}

func generateID(wing, room, content string) string {
	hash := uint32(0)
	for _, c := range content {
		hash = hash*31 + uint32(c)
	}
	return fmt.Sprintf("doc_%s_%s_%d", wing, room, hash%1000000)
}

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