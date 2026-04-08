// Package dialect implements the AAAK compression dialect.
package dialect

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/kinwyb/mempalace-go/pkg/vector"
)

// EntityType represents the type of entity in AAAK dialect
type EntityType string

const (
	EntityTypePerson   EntityType = "P"
	EntityTypeProject  EntityType = "p"
	EntityTypeTech     EntityType = "T"
	EntityTypeConcept  EntityType = "C"
	EntityTypeLocation EntityType = "L"
	EntityTypeDate     EntityType = "D"
	EntityTypeUnknown  EntityType = "?"
)

// CompressionLevel defines how aggressively to compress
type CompressionLevel int

const (
	CompressionNone       CompressionLevel = 0
	CompressionLight      CompressionLevel = 1
	CompressionMedium     CompressionLevel = 2
	CompressionAggressive CompressionLevel = 3
)

// Dialect handles AAAK compression
type Dialect struct {
	store            vector.Store
	compressionLevel CompressionLevel
	entityRegistry   map[string]EntityType
	entityAliases    map[string]string // alias -> canonical name
}

// New creates a new Dialect
func New(store vector.Store) *Dialect {
	return &Dialect{
		store:            store,
		compressionLevel: CompressionMedium,
		entityRegistry:   make(map[string]EntityType),
		entityAliases:    make(map[string]string),
	}
}

// SetCompressionLevel sets the compression level
func (d *Dialect) SetCompressionLevel(level CompressionLevel) *Dialect {
	d.compressionLevel = level
	return d
}

// RegisterEntity registers an entity with its type
func (d *Dialect) RegisterEntity(name string, entityType EntityType, aliases []string) {
	d.entityRegistry[name] = entityType
	for _, alias := range aliases {
		d.entityAliases[alias] = name
	}
}

// Compress compresses content using AAAK dialect
func (d *Dialect) Compress(ctx context.Context, content string) (string, error) {
	if d.compressionLevel == CompressionNone {
		return content, nil
	}

	// Detect and encode entities
	entities := d.detectEntities(content)
	encodedContent := content

	// Replace entities with AAAK encoding
	for _, entity := range entities {
		encoded := d.encodeEntity(entity)
		encodedContent = strings.ReplaceAll(encodedContent, entity.Name, encoded)
	}

	// Apply additional compression based on level
	switch d.compressionLevel {
	case CompressionLight:
		encodedContent = d.compressLight(encodedContent)
	case CompressionMedium:
		encodedContent = d.compressMedium(encodedContent)
	case CompressionAggressive:
		encodedContent = d.compressAggressive(encodedContent)
	}

	slog.Debug("Compressed content",
		"original_len", len(content),
		"compressed_len", len(encodedContent),
		"entities", len(entities),
	)

	return encodedContent, nil
}

// Decompress decompresses AAAK-encoded content
func (d *Dialect) Decompress(content string) (string, error) {
	// Find AAAK patterns and decode
	// aaakPattern := `[A-Z][a-z]([A-Z]|[0-9]+)` // Pattern like "PjD" or "p123"

	result := content
	// Find all AAAK patterns and decode them
	for name, entity := range d.entityRegistry {
		// Decode known entities
		_ = d.encodeEntity(Entity{Name: name, Type: entity}) // placeholder for decoding
		// Inverse mapping would be applied here
	}

	return result, nil
}

// Entity represents a detected entity
type Entity struct {
	Name string
	Type EntityType
}

// detectEntities detects entities in content
func (d *Dialect) detectEntities(content string) []Entity {
	var entities []Entity
	seen := make(map[string]bool)

	// Check registered entities
	for name, entityType := range d.entityRegistry {
		if strings.Contains(content, name) && !seen[name] {
			entities = append(entities, Entity{Name: name, Type: entityType})
			seen[name] = true
		}
	}

	// Check aliases
	for alias, canonical := range d.entityAliases {
		if strings.Contains(content, alias) && !seen[canonical] {
			entities = append(entities, Entity{Name: alias, Type: d.entityRegistry[canonical]})
			seen[canonical] = true
		}
	}

	// Auto-detect potential entities (capitalized words, patterns)
	if d.compressionLevel >= CompressionMedium {
		autoEntities := d.autoDetectEntities(content)
		for _, entity := range autoEntities {
			if !seen[entity.Name] {
				entities = append(entities, entity)
				seen[entity.Name] = true
			}
		}
	}

	return entities
}

// autoDetectEntities auto-detects entities from content patterns
func (d *Dialect) autoDetectEntities(content string) []Entity {
	var entities []Entity

	// Pattern: Capitalized names (likely people)
	words := strings.Fields(content)
	for _, word := range words {
		if len(word) >= 2 && word[0] >= 'A' && word[0] <= 'Z' {
			// Check if it's a potential name
			if isPotentialName(word) {
				entities = append(entities, Entity{Name: word, Type: EntityTypePerson})
			}
		}
	}

	return entities
}

// isPotentialName checks if a word is a potential name
func isPotentialName(word string) bool {
	// Simple heuristic: capitalized, not a common word
	commonWords := []string{"The", "This", "That", "It", "We", "They", "I", "You"}
	for _, cw := range commonWords {
		if word == cw {
			return false
		}
	}
	return true
}

// encodeEntity encodes an entity using AAAK format
func (d *Dialect) encodeEntity(entity Entity) string {
	// AAAK format: EntityType + initials + optional suffix
	// Examples: "PjD" = Person "John Doe", "pM" = project "Mempalace"

	if entity.Name == "" {
		return string(entity.Type)
	}

	// Extract initials
	parts := strings.Fields(entity.Name)
	initials := ""
	for _, part := range parts {
		if len(part) > 0 {
			initials += string(part[0])
		}
	}

	return string(entity.Type) + initials
}

// compressLight applies light compression
func (d *Dialect) compressLight(content string) string {
	// Remove redundant whitespace
	content = strings.Join(strings.Fields(content), " ")
	return content
}

// compressMedium applies medium compression
func (d *Dialect) compressMedium(content string) string {
	// Light compression + remove filler words
	content = d.compressLight(content)

	fillerWords := []string{"very", "really", "quite", "somewhat", "basically", "actually"}
	for _, filler := range fillerWords {
		content = strings.ReplaceAll(content, " "+filler+" ", " ")
	}

	return content
}

// compressAggressive applies aggressive compression
func (d *Dialect) compressAggressive(content string) string {
	// Medium compression + shorten common phrases
	content = d.compressMedium(content)

	phraseMap := map[string]string{
		"in order to":        "to",
		"as a result":        "so",
		"due to the fact":    "because",
		"at the present":     "now",
		"in the near future": "soon",
	}

	for phrase, replacement := range phraseMap {
		content = strings.ReplaceAll(content, phrase, replacement)
	}

	return content
}

// CompressDrawers compresses all drawers in a wing/room
func (d *Dialect) CompressDrawers(ctx context.Context, wing, room string, level CompressionLevel) (int, error) {
	d.compressionLevel = level

	// Get all drawers in wing/room
	results, err := d.store.Search(ctx, "", wing, room, 1000)
	if err != nil {
		return 0, err
	}

	compressedCount := 0
	for _, result := range results {
		compressed, err := d.Compress(ctx, result.Content)
		if err != nil {
			slog.Warn("failed to compress drawer", "id", result.ID, "error", err)
			continue
		}

		// Update the document with compressed content
		doc := vector.Document{
			ID:      result.ID,
			Content: compressed,
			Metadata: map[string]any{
				"wing":           wing,
				"room":           room,
				"compressed":     true,
				"original_len":   len(result.Content),
				"compressed_len": len(compressed),
			},
		}

		if err := d.store.Add(ctx, []vector.Document{doc}); err != nil {
			slog.Warn("failed to update compressed drawer", "id", result.ID, "error", err)
			continue
		}
		compressedCount++
	}

	slog.Info("Compressed drawers",
		"wing", wing,
		"room", room,
		"count", compressedCount,
		"level", level,
	)

	return compressedCount, nil
}

// GetEntityRegistry returns the current entity registry
func (d *Dialect) GetEntityRegistry() map[string]EntityType {
	return d.entityRegistry
}

// GetEntityAliases returns the entity aliases
func (d *Dialect) GetEntityAliases() map[string]string {
	return d.entityAliases
}

// ExportRegistry exports the entity registry as a readable format
func (d *Dialect) ExportRegistry() string {
	var builder strings.Builder

	builder.WriteString("# AAAK Entity Registry\n\n")

	// Group by type
	byType := make(map[EntityType][]string)
	for name, entityType := range d.entityRegistry {
		byType[entityType] = append(byType[entityType], name)
	}

	// Sort types
	var types []EntityType
	for t := range byType {
		types = append(types, t)
	}
	sort.Slice(types, func(i, j int) bool {
		return types[i] < types[j]
	})

	for _, t := range types {
		builder.WriteString(fmt.Sprintf("## %s (%s)\n", entityTypeDescription(t), t))
		for _, name := range byType[t] {
			builder.WriteString(fmt.Sprintf("- %s\n", name))
			// Show aliases
			for alias, canonical := range d.entityAliases {
				if canonical == name {
					builder.WriteString(fmt.Sprintf("  - alias: %s\n", alias))
				}
			}
		}
		builder.WriteString("\n")
	}

	return builder.String()
}

func entityTypeDescription(t EntityType) string {
	switch t {
	case EntityTypePerson:
		return "Person"
	case EntityTypeProject:
		return "Project"
	case EntityTypeTech:
		return "Technology"
	case EntityTypeConcept:
		return "Concept"
	case EntityTypeLocation:
		return "Location"
	case EntityTypeDate:
		return "Date"
	default:
		return "Unknown"
	}
}
