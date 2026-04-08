// Package entity provides entity detection and registry functionality.
package entity

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/kinwyb/mempalace-go/pkg/vector"
)

// EntityType represents the type of detected entity
type EntityType string

const (
	EntityPerson   EntityType = "person"
	EntityProject  EntityType = "project"
	EntityTech     EntityType = "technology"
	EntityConcept  EntityType = "concept"
	EntityLocation EntityType = "location"
	EntityDate     EntityType = "date"
)

// Entity represents a detected entity
type Entity struct {
	ID          string
	Name        string
	Type        EntityType
	Description string
	Aliases     []string
	Confidence  float64
	Context     string // Context where entity was found
}

// Detector handles entity detection
type Detector struct {
	store         vector.Store
	patterns      map[EntityType][]*regexp.Regexp
	knownEntities map[string]Entity
	contextWindow int
}

// NewDetector creates a new entity detector
func NewDetector(store vector.Store) *Detector {
	d := &Detector{
		store:         store,
		knownEntities: make(map[string]Entity),
		contextWindow: 100,
	}

	d.initPatterns()
	return d
}

// initPatterns initializes detection patterns
func (d *Detector) initPatterns() {
	d.patterns = map[EntityType][]*regexp.Regexp{
		EntityPerson: {
			regexp.MustCompile(`(?i)(?:I am|my name is|I'm|call me)\s+([A-Z][a-z]+(?:\s+[A-Z][a-z]+)?)`),
			regexp.MustCompile(`(?i)(?:meet|talked with|spoke to)\s+([A-Z][a-z]+(?:\s+[A-Z][a-z]+)?)`),
			regexp.MustCompile(`(?i)(?:@([a-zA-Z0-9_]+))`),
		},
		EntityProject: {
			regexp.MustCompile(`(?i)(?:working on|building|developing|creating)\s+(?:a\s+)?([A-Z][a-zA-Z0-9]+(?:\s+[a-zA-Z0-9]+)?)`),
			regexp.MustCompile(`(?i)(?:project|repo|repository)\s+(?:called|named)?\s*([A-Z][a-zA-Z0-9-]+)`),
		},
		EntityTech: {
			regexp.MustCompile(`(?i)(using|with|in|via)\s+(Python|JavaScript|Go|TypeScript|React|Vue|Django|Flask|Node\.js|Kubernetes|Docker|AWS|GCP|Azure)`),
			regexp.MustCompile(`(?i)(API|SDK|CLI|REST|GraphQL|WebSocket|gRPC|HTTP|HTTPS|JSON|YAML|SQL)`),
		},
		EntityLocation: {
			regexp.MustCompile(`(?i)(?:in|at|from|to)\s+([A-Z][a-z]+(?:\s+[A-Z][a-z]+)?(?:\s+(?:City|State|Country|Office|HQ))?)`),
		},
		EntityDate: {
			regexp.MustCompile(`(?i)(?:on|by|before|after|at)\s+(?:Monday|Tuesday|Wednesday|Thursday|Friday|Saturday|Sunday)`),
			regexp.MustCompile(`(?i)(?:on|by|before|after)\s+(?:January|February|March|April|May|June|July|August|September|October|November|December)\s+\d{1,2}`),
			regexp.MustCompile(`\d{4}-\d{2}-\d{2}`),
			regexp.MustCompile(`\d{1,2}/\d{1,2}/\d{2,4}`),
		},
	}
}

// Detect detects entities in content
func (d *Detector) Detect(content string) []Entity {
	var entities []Entity
	seen := make(map[string]bool)

	// Check known entities
	for name, entity := range d.knownEntities {
		if strings.Contains(content, name) && !seen[name] {
			entities = append(entities, entity)
			seen[name] = true
		}
		for _, alias := range entity.Aliases {
			if strings.Contains(content, alias) && !seen[alias] {
				entities = append(entities, entity)
				seen[alias] = true
			}
		}
	}

	// Pattern-based detection
	for entityType, patterns := range d.patterns {
		for _, pattern := range patterns {
			matches := pattern.FindAllStringSubmatch(content, -1)
			for _, match := range matches {
				if len(match) > 1 {
					name := match[1]
					if !seen[name] {
						context := d.extractContext(content, match[0])
						entity := Entity{
							ID:         generateEntityID(name, entityType),
							Name:       name,
							Type:       entityType,
							Confidence: 0.8,
							Context:    context,
						}
						entities = append(entities, entity)
						seen[name] = true
					}
				}
			}
		}
	}

	slog.Debug("Detected entities", "count", len(entities))
	return entities
}

// extractContext extracts context around a match
func (d *Detector) extractContext(content, match string) string {
	idx := strings.Index(content, match)
	if idx == -1 {
		return ""
	}

	start := idx - d.contextWindow/2
	if start < 0 {
		start = 0
	}

	end := idx + len(match) + d.contextWindow/2
	if end > len(content) {
		end = len(content)
	}

	return strings.TrimSpace(content[start:end])
}

// Register registers a known entity
func (d *Detector) Register(entity Entity) {
	d.knownEntities[entity.Name] = entity
	for _, alias := range entity.Aliases {
		d.knownEntities[alias] = entity
	}
}

// RegisterFromContent extracts and registers entities from content
func (d *Detector) RegisterFromContent(content string) []Entity {
	entities := d.Detect(content)

	for _, entity := range entities {
		if entity.Confidence >= 0.7 {
			d.Register(entity)
		}
	}

	return entities
}

// Store stores detected entities to the vector store
func (d *Detector) Store(ctx context.Context, wing string, entities []Entity) error {
	var docs []vector.Document

	for _, entity := range entities {
		doc := vector.Document{
			ID:      entity.ID,
			Content: fmt.Sprintf("%s: %s", entity.Name, entity.Description),
			Metadata: map[string]any{
				"wing":        wing,
				"room":        "entities",
				"entity_type": string(entity.Type),
				"entity_name": entity.Name,
				"aliases":     strings.Join(entity.Aliases, ","),
				"confidence":  entity.Confidence,
			},
		}
		docs = append(docs, doc)
	}

	if len(docs) == 0 {
		return nil
	}

	return d.store.Add(ctx, docs)
}

// Get retrieves an entity by name
func (d *Detector) Get(name string) *Entity {
	if entity, ok := d.knownEntities[name]; ok {
		return &entity
	}
	return nil
}

// GetAll returns all known entities
func (d *Detector) GetAll() []Entity {
	var entities []Entity
	seen := make(map[string]bool)

	for _, entity := range d.knownEntities {
		if !seen[entity.ID] {
			entities = append(entities, entity)
			seen[entity.ID] = true
		}
	}

	return entities
}

// GetByType returns entities of a specific type
func (d *Detector) GetByType(entityType EntityType) []Entity {
	var entities []Entity
	seen := make(map[string]bool)

	for _, entity := range d.knownEntities {
		if entity.Type == entityType && !seen[entity.ID] {
			entities = append(entities, entity)
			seen[entity.ID] = true
		}
	}

	return entities
}

// generateEntityID generates a unique ID for an entity
func generateEntityID(name string, entityType EntityType) string {
	name = strings.ToLower(strings.ReplaceAll(name, " ", "_"))
	return fmt.Sprintf("entity_%s_%s", entityType, name)
}

// DetectAndRegister combines detection and registration
func (d *Detector) DetectAndRegister(content, wing string) ([]Entity, error) {
	entities := d.RegisterFromContent(content)

	ctx := context.Background()
	if err := d.Store(ctx, wing, entities); err != nil {
		return nil, err
	}

	return entities, nil
}

// FormatEntity formats an entity for display
func FormatEntity(entity Entity) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("**%s** (%s)\n", entity.Name, entity.Type))
	if entity.Description != "" {
		builder.WriteString(fmt.Sprintf("  %s\n", entity.Description))
	}
	if len(entity.Aliases) > 0 {
		builder.WriteString(fmt.Sprintf("  Aliases: %s\n", strings.Join(entity.Aliases, ", ")))
	}
	if entity.Context != "" {
		builder.WriteString(fmt.Sprintf("  Context: \"%s\"\n", entity.Context))
	}

	return builder.String()
}
