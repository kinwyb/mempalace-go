// Package palace provides palace graph traversal and room detection.
package palace

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/kinwyb/mempalace-go/pkg/vector"
)

// Palace represents the memory palace structure
type Palace struct {
	store vector.Store
}

// Wing represents a wing in the palace (project/person context)
type Wing struct {
	Name         string
	Description  string
	Rooms        []Room
	TotalDrawers int
}

// Room represents a room in a wing (topic area)
type Room struct {
	Name         string
	Description  string
	KeyWords     []string
	TotalDrawers int
	Wing         string
}

// Drawer represents a piece of content stored in a room
type Drawer struct {
	ID         string
	Content    string
	Wing       string
	Room       string
	SourceFile string
	CreatedAt  string
}

// New creates a new Palace
func New(store vector.Store) *Palace {
	return &Palace{store: store}
}

// GetWings returns all wings in the palace
func (p *Palace) GetWings(ctx context.Context) ([]Wing, error) {
	stats, err := p.store.GetStats(ctx)
	if err != nil {
		return nil, err
	}

	var wings []Wing
	for wingName, rooms := range stats.WingRoomCounts {
		wing := Wing{
			Name:         wingName,
			Rooms:        make([]Room, 0),
			TotalDrawers: 0,
		}

		for roomName, count := range rooms {
			wing.Rooms = append(wing.Rooms, Room{
				Name:         roomName,
				Wing:         wingName,
				TotalDrawers: count,
			})
			wing.TotalDrawers += count
		}

		wings = append(wings, wing)
	}

	slog.Debug("Got wings", "count", len(wings))
	return wings, nil
}

// GetRooms returns all rooms for a wing
func (p *Palace) GetRooms(ctx context.Context, wing string) ([]Room, error) {
	stats, err := p.store.GetStats(ctx)
	if err != nil {
		return nil, err
	}

	if stats.WingRoomCounts[wing] == nil {
		return nil, fmt.Errorf("wing '%s' not found", wing)
	}

	var rooms []Room
	for roomName, count := range stats.WingRoomCounts[wing] {
		rooms = append(rooms, Room{
			Name:         roomName,
			Wing:         wing,
			TotalDrawers: count,
		})
	}

	return rooms, nil
}

// GetRoom returns a specific room
func (p *Palace) GetRoom(ctx context.Context, wing, room string) (*Room, error) {
	rooms, err := p.GetRooms(ctx, wing)
	if err != nil {
		return nil, err
	}

	for _, r := range rooms {
		if r.Name == room {
			return &r, nil
		}
	}

	return nil, fmt.Errorf("room '%s' not found in wing '%s'", room, wing)
}

// GetDrawers returns drawers for a wing/room
func (p *Palace) GetDrawers(ctx context.Context, wing, room string, limit int) ([]Drawer, error) {
	results, err := p.store.Search(ctx, "", wing, room, limit)
	if err != nil {
		return nil, err
	}

	var drawers []Drawer
	for _, result := range results {
		drawer := Drawer{
			ID:      result.ID,
			Content: result.Content,
			Wing:    wing,
			Room:    room,
		}

		if v, ok := result.Metadata["source_file"]; ok {
			drawer.SourceFile = v.(string)
		}
		if v, ok := result.Metadata["filed_at"]; ok {
			drawer.CreatedAt = v.(string)
		}

		drawers = append(drawers, drawer)
	}

	return drawers, nil
}

// DetectRoom detects the appropriate room for content
func (p *Palace) DetectRoom(ctx context.Context, content, wing string) (string, error) {
	// Get all rooms for the wing
	rooms, err := p.GetRooms(ctx, wing)
	if err != nil {
		return "general", nil
	}

	// Score each room based on keyword matches
	contentLower := strings.ToLower(content)
	scores := make(map[string]int)

	for _, room := range rooms {
		// Check room name
		if strings.Contains(contentLower, strings.ToLower(room.Name)) {
			scores[room.Name] += 10
		}

		// Check keywords
		for _, kw := range room.KeyWords {
			if strings.Contains(contentLower, strings.ToLower(kw)) {
				scores[room.Name] += 5
			}
		}
	}

	// Find best room
	bestRoom := "general"
	bestScore := 0

	for room, score := range scores {
		if score > bestScore {
			bestScore = score
			bestRoom = room
		}
	}

	return bestRoom, nil
}

// DetectWingFromPath detects wing from a file path
func DetectWingFromPath(path string) string {
	// Get the project root directory name
	absPath := path
	dir := filepath.Dir(absPath)

	// Walk up to find project root
	for {
		base := filepath.Base(dir)
		// Skip common non-project directories
		if base == "src" || base == "internal" || base == "pkg" || base == "cmd" {
			dir = filepath.Dir(dir)
			continue
		}

		// Found a likely project root
		if base != "." && base != "/" {
			return sanitizeWingName(base)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "general"
}

// DetectRoomFromPath detects room from a file path
func DetectRoomFromPath(path string) string {
	path = filepath.ToSlash(path)

	// Check for common directory patterns
	patterns := map[string]string{
		"src/":        "source",
		"internal/":   "internal",
		"pkg/":        "package",
		"cmd/":        "commands",
		"api/":        "api",
		"test/":       "tests",
		"tests/":      "tests",
		"docs/":       "docs",
		"doc/":        "docs",
		"config/":     "config",
		"util/":       "utils",
		"utils/":      "utils",
		"lib/":        "library",
		"scripts/":    "scripts",
		"db/":         "database",
		"models/":     "models",
		"views/":      "views",
		"handlers/":   "handlers",
		"services/":   "services",
		"components/": "components",
	}

	for pattern, room := range patterns {
		if strings.Contains(path, pattern) {
			return room
		}
	}

	// Use the immediate directory name as room
	dir := filepath.Dir(filepath.Base(path))
	if dir != "" && dir != "." {
		return sanitizeRoomName(dir)
	}

	return "general"
}

// sanitizeWingName sanitizes a wing name
func sanitizeWingName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, ".", "")
	return name
}

// sanitizeRoomName sanitizes a room name
func sanitizeRoomName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "-", "_")
	return name
}

// Traverse traverses the palace structure
func (p *Palace) Traverse(ctx context.Context, wing string, visitor func(wing Wing, room Room, drawer Drawer) bool) error {
	rooms, err := p.GetRooms(ctx, wing)
	if err != nil {
		return err
	}

	w := Wing{Name: wing, Rooms: rooms}

	for _, room := range rooms {
		drawers, err := p.GetDrawers(ctx, wing, room.Name, 100)
		if err != nil {
			slog.Warn("failed to get drawers", "wing", wing, "room", room.Name, "error", err)
			continue
		}

		for _, drawer := range drawers {
			if !visitor(w, room, drawer) {
				return nil
			}
		}
	}

	return nil
}

// GetStats returns palace statistics
func (p *Palace) GetStats(ctx context.Context) (*PalaceStats, error) {
	stats, err := p.store.GetStats(ctx)
	if err != nil {
		return nil, err
	}

	palaceStats := &PalaceStats{
		TotalDrawers: stats.TotalDocuments,
		TotalWings:   stats.TotalWings,
		TotalRooms:   stats.TotalRooms,
		StorageSize:  stats.StorageSize,
		LastUpdated:  stats.LastUpdated,
		Wings:        make(map[string]WingStats),
	}

	for wingName, rooms := range stats.WingRoomCounts {
		wingStats := WingStats{
			Name:  wingName,
			Rooms: make(map[string]int),
			Total: 0,
		}

		for roomName, count := range rooms {
			wingStats.Rooms[roomName] = count
			wingStats.Total += count
		}

		palaceStats.Wings[wingName] = wingStats
	}

	return palaceStats, nil
}

// PalaceStats represents palace statistics
type PalaceStats struct {
	TotalDrawers int
	TotalWings   int
	TotalRooms   int
	StorageSize  int64
	LastUpdated  string
	Wings        map[string]WingStats
}

// WingStats represents wing statistics
type WingStats struct {
	Name  string
	Rooms map[string]int
	Total int
}

// FormatStats formats palace stats for display
func FormatStats(stats *PalaceStats) string {
	var builder strings.Builder

	builder.WriteString("\n=== Palace Status ===\n\n")
	builder.WriteString(fmt.Sprintf("Total Drawers: %d\n", stats.TotalDrawers))
	builder.WriteString(fmt.Sprintf("Total Wings: %d\n", stats.TotalWings))
	builder.WriteString(fmt.Sprintf("Total Rooms: %d\n", stats.TotalRooms))
	builder.WriteString(fmt.Sprintf("Storage Size: %.2f MB\n", float64(stats.StorageSize)/1024/1024))
	builder.WriteString("\n")

	for wingName, wingStats := range stats.Wings {
		builder.WriteString(fmt.Sprintf("📁 Wing: %s (%d drawers)\n", wingName, wingStats.Total))
		for roomName, count := range wingStats.Rooms {
			builder.WriteString(fmt.Sprintf("  📂 Room: %s (%d drawers)\n", roomName, count))
		}
		builder.WriteString("\n")
	}

	return builder.String()
}
