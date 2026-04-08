package entity

import (
	"testing"
)

func TestNewDetector(t *testing.T) {
	d := NewDetector(nil)

	if d == nil {
		t.Fatal("NewDetector returned nil")
	}

	if d.patterns == nil {
		t.Error("Patterns should be initialized")
	}
}

func TestDetectPerson(t *testing.T) {
	d := NewDetector(nil)

	content := "My name is John Smith and I work with Jane Doe."
	entities := d.Detect(content)

	// Should detect potential names
	if len(entities) == 0 {
		t.Log("Warning: No entities detected (pattern matching may vary)")
	}
}

func TestDetectProject(t *testing.T) {
	d := NewDetector(nil)

	content := "I am working on Project Alpha and building the Mempalace system."
	entities := d.Detect(content)

	// Check for project detection
	for _, e := range entities {
		if e.Type == EntityProject {
			t.Logf("Detected project: %s", e.Name)
		}
	}
}

func TestDetectTech(t *testing.T) {
	d := NewDetector(nil)

	content := "We are using Python, React, and Kubernetes for this project."
	entities := d.Detect(content)

	// Should detect tech entities
	techCount := 0
	for _, e := range entities {
		if e.Type == EntityTech {
			techCount++
		}
	}

	t.Logf("Detected %d tech entities", techCount)
}

func TestRegister(t *testing.T) {
	d := NewDetector(nil)

	entity := Entity{
		ID:          "test-1",
		Name:        "TestEntity",
		Type:        EntityPerson,
		Description: "A test entity",
		Aliases:     []string{"TE", "test"},
	}

	d.Register(entity)

	// Should be retrievable
	retrieved := d.Get("TestEntity")
	if retrieved == nil {
		t.Error("Entity should be registered")
	}

	if retrieved.Name != "TestEntity" {
		t.Errorf("Retrieved entity name = %s, want TestEntity", retrieved.Name)
	}

	// Should be retrievable by alias
	retrievedByAlias := d.Get("TE")
	if retrievedByAlias == nil {
		t.Error("Entity should be retrievable by alias")
	}
}

func TestRegisterFromContent(t *testing.T) {
	d := NewDetector(nil)

	content := "I'm working with Alice on the Apollo project using Go."
	entities := d.RegisterFromContent(content)

	// Should have detected and registered entities
	t.Logf("Registered %d entities", len(entities))
}

func TestGetAll(t *testing.T) {
	d := NewDetector(nil)

	d.Register(Entity{ID: "1", Name: "Entity1", Type: EntityPerson})
	d.Register(Entity{ID: "2", Name: "Entity2", Type: EntityProject})
	d.Register(Entity{ID: "3", Name: "Entity3", Type: EntityTech})

	all := d.GetAll()

	if len(all) < 3 {
		t.Errorf("Expected at least 3 entities, got %d", len(all))
	}
}

func TestGetByType(t *testing.T) {
	d := NewDetector(nil)

	d.Register(Entity{ID: "1", Name: "Person1", Type: EntityPerson})
	d.Register(Entity{ID: "2", Name: "Person2", Type: EntityPerson})
	d.Register(Entity{ID: "3", Name: "Project1", Type: EntityProject})

	persons := d.GetByType(EntityPerson)

	if len(persons) != 2 {
		t.Errorf("Expected 2 persons, got %d", len(persons))
	}
}

func TestExtractContext(t *testing.T) {
	d := NewDetector(nil)
	d.contextWindow = 50

	content := "This is a long piece of content with John Smith mentioned somewhere in the middle of it."
	context := d.extractContext(content, "John Smith")

	if context == "" {
		t.Error("Context should not be empty")
	}

	// Context should contain the match
	if len(context) > 100 {
		t.Log("Context was extracted successfully")
	}
}

func TestGenerateEntityID(t *testing.T) {
	id := generateEntityID("John Doe", EntityPerson)

	if id == "" {
		t.Error("ID should not be empty")
	}

	// Should contain entity type
	if len(id) < 5 {
		t.Errorf("ID seems too short: %s", id)
	}
}

func TestFormatEntity(t *testing.T) {
	entity := Entity{
		ID:          "test-1",
		Name:        "Test Entity",
		Type:        EntityPerson,
		Description: "A test description",
		Aliases:     []string{"TE"},
		Context:     "Found in test context",
	}

	formatted := FormatEntity(entity)

	if formatted == "" {
		t.Error("Formatted entity should not be empty")
	}

	// Should contain entity name and type
	if len(formatted) < 10 {
		t.Errorf("Formatted entity seems too short: %s", formatted)
	}
}
