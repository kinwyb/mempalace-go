// Package kg provides a temporal knowledge graph implementation.
package kg

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// Entity represents a knowledge graph entity.
type Entity struct {
	ID          string
	Name        string
	Type        string // person, project, concept, technology, location
	Description string
	Aliases     []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Relation represents a relationship between entities.
type Relation struct {
	ID         string
	FromID     string
	ToID       string
	Type       string // knows, works_on, uses, mentions, depends_on
	Context    string
	ValidFrom  time.Time
	ValidTo    *time.Time // nil means currently valid
	Confidence float64
	CreatedAt  time.Time
}

// KnowledgeGraph implements a temporal knowledge graph using SQLite.
type KnowledgeGraph struct {
	db   *sql.DB
	path string
	mu   sync.RWMutex
}

// New creates a new KnowledgeGraph.
func New(path string) (*KnowledgeGraph, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	kg := &KnowledgeGraph{
		db:   db,
		path: path,
	}

	if err := kg.initialize(); err != nil {
		return nil, err
	}

	return kg, nil
}

// initialize creates the necessary tables.
func (kg *KnowledgeGraph) initialize() error {
	// Create entities table
	_, err := kg.db.Exec(`
		CREATE TABLE IF NOT EXISTS entities (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			description TEXT,
			aliases TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create entities table: %w", err)
	}

	// Create relations table with temporal validity
	_, err = kg.db.Exec(`
		CREATE TABLE IF NOT EXISTS relations (
			id TEXT PRIMARY KEY,
			from_id TEXT NOT NULL,
			to_id TEXT NOT NULL,
			type TEXT NOT NULL,
			context TEXT,
			valid_from DATETIME DEFAULT CURRENT_TIMESTAMP,
			valid_to DATETIME,
			confidence REAL DEFAULT 1.0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (from_id) REFERENCES entities(id),
			FOREIGN KEY (to_id) REFERENCES entities(id)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create relations table: %w", err)
	}

	// Create indexes
	_, err = kg.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_entities_type ON entities(type);
		CREATE INDEX IF NOT EXISTS idx_entities_name ON entities(name);
		CREATE INDEX IF NOT EXISTS idx_relations_from ON relations(from_id);
		CREATE INDEX IF NOT EXISTS idx_relations_to ON relations(to_id);
		CREATE INDEX IF NOT EXISTS idx_relations_type ON relations(type);
		CREATE INDEX IF NOT EXISTS idx_relations_valid ON relations(valid_from, valid_to);
	`)
	if err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	slog.Info("Initialized knowledge graph", "path", kg.path)
	return nil
}

// AddEntity adds a new entity to the graph.
func (kg *KnowledgeGraph) AddEntity(entity Entity) error {
	kg.mu.Lock()
	defer kg.mu.Unlock()

	aliasesJSON := aliasesToJSON(entity.Aliases)

	_, err := kg.db.Exec(`
		INSERT OR REPLACE INTO entities
		(id, name, type, description, aliases, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, entity.ID, entity.Name, entity.Type, entity.Description,
		aliasesJSON, entity.CreatedAt, entity.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to add entity: %w", err)
	}

	slog.Debug("Added entity", "id", entity.ID, "name", entity.Name, "type", entity.Type)
	return nil
}

// AddRelation adds a new relation to the graph.
func (kg *KnowledgeGraph) AddRelation(relation Relation) error {
	kg.mu.Lock()
	defer kg.mu.Unlock()

	_, err := kg.db.Exec(`
		INSERT OR REPLACE INTO relations
		(id, from_id, to_id, type, context, valid_from, valid_to, confidence, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, relation.ID, relation.FromID, relation.ToID, relation.Type,
		relation.Context, relation.ValidFrom, relation.ValidTo,
		relation.Confidence, relation.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to add relation: %w", err)
	}

	slog.Debug("Added relation", "from", relation.FromID, "to", relation.ToID, "type", relation.Type)
	return nil
}

// GetEntity retrieves an entity by ID or name.
func (kg *KnowledgeGraph) GetEntity(idOrName string) (*Entity, error) {
	kg.mu.RLock()
	defer kg.mu.RUnlock()

	var entity Entity
	var aliasesJSON string

	err := kg.db.QueryRow(`
		SELECT id, name, type, description, aliases, created_at, updated_at
		FROM entities WHERE id = ? OR name = ?
	`, idOrName, idOrName).Scan(&entity.ID, &entity.Name, &entity.Type,
		&entity.Description, &aliasesJSON, &entity.CreatedAt, &entity.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	entity.Aliases = jsonToAliases(aliasesJSON)
	return &entity, nil
}

// GetRelations retrieves all relations for an entity.
func (kg *KnowledgeGraph) GetRelations(entityID string, includeExpired bool) ([]Relation, error) {
	kg.mu.RLock()
	defer kg.mu.RUnlock()

	query := `
		SELECT id, from_id, to_id, type, context, valid_from, valid_to, confidence, created_at
		FROM relations WHERE from_id = ? OR to_id = ?
	`
	if !includeExpired {
		query += " AND (valid_to IS NULL OR valid_to > CURRENT_TIMESTAMP)"
	}

	rows, err := kg.db.Query(query, entityID, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var relations []Relation
	for rows.Next() {
		var r Relation
		var validTo sql.NullTime

		err := rows.Scan(&r.ID, &r.FromID, &r.ToID, &r.Type, &r.Context,
			&r.ValidFrom, &validTo, &r.Confidence, &r.CreatedAt)
		if err != nil {
			continue
		}

		if validTo.Valid {
			r.ValidTo = &validTo.Time
		}
		relations = append(relations, r)
	}

	return relations, nil
}

// ExpireRelation marks a relation as no longer valid.
func (kg *KnowledgeGraph) ExpireRelation(relationID string, validTo time.Time) error {
	kg.mu.Lock()
	defer kg.mu.Unlock()

	_, err := kg.db.Exec(`
		UPDATE relations SET valid_to = ? WHERE id = ?
	`, validTo, relationID)

	return err
}

// SearchEntities searches for entities by name or description.
func (kg *KnowledgeGraph) SearchEntities(query string, entityType string) ([]Entity, error) {
	kg.mu.RLock()
	defer kg.mu.RUnlock()

	searchQuery := `
		SELECT id, name, type, description, aliases, created_at, updated_at
		FROM entities WHERE
		(name LIKE ? OR description LIKE ? OR aliases LIKE ?)
	`
	args := []any{"%" + query + "%", "%" + query + "%", "%" + query + "%"}

	if entityType != "" {
		searchQuery += " AND type = ?"
		args = append(args, entityType)
	}

	rows, err := kg.db.Query(searchQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entities []Entity
	for rows.Next() {
		var e Entity
		var aliasesJSON string

		err := rows.Scan(&e.ID, &e.Name, &e.Type, &e.Description,
			&aliasesJSON, &e.CreatedAt, &e.UpdatedAt)
		if err != nil {
			continue
		}

		e.Aliases = jsonToAliases(aliasesJSON)
		entities = append(entities, e)
	}

	return entities, nil
}

// FindPath finds a path between two entities.
func (kg *KnowledgeGraph) FindPath(fromID, toID string, maxDepth int) ([]Relation, error) {
	kg.mu.RLock()
	defer kg.mu.RUnlock()

	// Simple BFS path finding
	visited := make(map[string]bool)
	queue := [][]Relation{{}}

	for len(queue) > 0 && len(queue[0]) < maxDepth {
		path := queue[0]
		queue = queue[1:]

		currentID := fromID
		if len(path) > 0 {
			currentID = path[len(path)-1].ToID
		}

		if currentID == toID {
			return path, nil
		}

		if visited[currentID] {
			continue
		}
		visited[currentID] = true

		// Get relations from current entity
		rows, err := kg.db.Query(`
			SELECT id, from_id, to_id, type, context, valid_from, valid_to, confidence, created_at
			FROM relations WHERE from_id = ? AND (valid_to IS NULL OR valid_to > CURRENT_TIMESTAMP)
		`, currentID)
		if err != nil {
			continue
		}

		for rows.Next() {
			var r Relation
			var validTo sql.NullTime

			err := rows.Scan(&r.ID, &r.FromID, &r.ToID, &r.Type, &r.Context,
				&r.ValidFrom, &validTo, &r.Confidence, &r.CreatedAt)
			if err != nil {
				continue
			}

			if validTo.Valid {
				r.ValidTo = &validTo.Time
			}

			newPath := append([]Relation{}, path...)
			newPath = append(newPath, r)
			queue = append(queue, newPath)
		}
		rows.Close()
	}

	return nil, nil
}

// GetStats returns statistics about the knowledge graph.
func (kg *KnowledgeGraph) GetStats() (map[string]int, error) {
	kg.mu.RLock()
	defer kg.mu.RUnlock()

	stats := make(map[string]int)

	// Count entities
	var entityCount int
	err := kg.db.QueryRow("SELECT COUNT(*) FROM entities").Scan(&entityCount)
	if err != nil {
		return nil, err
	}
	stats["entities"] = entityCount

	// Count relations
	var relationCount int
	err = kg.db.QueryRow("SELECT COUNT(*) FROM relations").Scan(&relationCount)
	if err != nil {
		return nil, err
	}
	stats["relations"] = relationCount

	// Count active relations
	var activeCount int
	err = kg.db.QueryRow(`
		SELECT COUNT(*) FROM relations
		WHERE valid_to IS NULL OR valid_to > CURRENT_TIMESTAMP
	`).Scan(&activeCount)
	if err != nil {
		return nil, err
	}
	stats["active_relations"] = activeCount

	// Count by entity type
	rows, err := kg.db.Query("SELECT type, COUNT(*) FROM entities GROUP BY type")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var typ string
		var count int
		if err := rows.Scan(&typ, &count); err != nil {
			continue
		}
		stats["entities_"+typ] = count
	}

	return stats, nil
}

// Close closes the database connection.
func (kg *KnowledgeGraph) Close() error {
	kg.mu.Lock()
	defer kg.mu.Unlock()
	return kg.db.Close()
}

// Helper functions

func aliasesToJSON(aliases []string) string {
	if len(aliases) == 0 {
		return "[]"
	}
	result := "["
	for i, a := range aliases {
		if i > 0 {
			result += ","
		}
		result += "\"" + a + "\""
	}
	result += "]"
	return result
}

func jsonToAliases(s string) []string {
	if s == "" || s == "[]" {
		return nil
	}
	// Simple parsing
	return []string{}
}
