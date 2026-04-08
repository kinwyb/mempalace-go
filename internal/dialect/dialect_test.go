package dialect

import (
	"testing"
)

func TestNewDialect(t *testing.T) {
	d := New(nil)

	if d == nil {
		t.Fatal("New returned nil")
	}

	if d.compressionLevel != CompressionMedium {
		t.Errorf("Expected default compression level %d, got %d", CompressionMedium, d.compressionLevel)
	}
}

func TestSetCompressionLevel(t *testing.T) {
	d := New(nil)

	d.SetCompressionLevel(CompressionAggressive)

	if d.compressionLevel != CompressionAggressive {
		t.Errorf("Expected compression level %d, got %d", CompressionAggressive, d.compressionLevel)
	}
}

func TestRegisterEntity(t *testing.T) {
	d := New(nil)

	d.RegisterEntity("John Doe", EntityTypePerson, []string{"John", "JD"})

	if d.entityRegistry["John Doe"] != EntityTypePerson {
		t.Error("Entity not registered correctly")
	}

	if d.entityAliases["John"] != "John Doe" {
		t.Error("Alias not registered correctly")
	}
}

func TestCompressNone(t *testing.T) {
	d := New(nil)
	d.SetCompressionLevel(CompressionNone)

	content := "This is test content."
	result, err := d.Compress(nil, content)

	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	if result != content {
		t.Error("CompressionNone should return original content")
	}
}

func TestCompressLight(t *testing.T) {
	d := New(nil)
	d.SetCompressionLevel(CompressionLight)

	content := "This  is   test    content."
	result, err := d.Compress(nil, content)

	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	// Light compression should normalize whitespace
	if result == content {
		t.Error("Content should be modified")
	}
}

func TestCompressMedium(t *testing.T) {
	d := New(nil)
	d.SetCompressionLevel(CompressionMedium)

	content := "This is very really quite important."
	result, err := d.Compress(nil, content)

	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	// Medium compression should remove filler words
	if result == content {
		t.Error("Content should be modified")
	}
}

func TestCompressAggressive(t *testing.T) {
	d := New(nil)
	d.SetCompressionLevel(CompressionAggressive)

	content := "In order to succeed, we need to work hard."
	result, err := d.Compress(nil, content)

	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	// Aggressive compression should shorten phrases
	if result == content {
		t.Error("Content should be modified")
	}
}

func TestEntityDetection(t *testing.T) {
	d := New(nil)
	d.RegisterEntity("Python", EntityTypeTech, nil)
	d.RegisterEntity("React", EntityTypeTech, []string{"reactjs"})

	content := "I'm using Python and React for my project."
	entities := d.detectEntities(content)

	if len(entities) < 2 {
		t.Errorf("Expected at least 2 entities, got %d", len(entities))
	}
}

func TestEncodeEntity(t *testing.T) {
	d := New(nil)

	tests := []struct {
		entity   Entity
		expected string
	}{
		{Entity{Name: "John Doe", Type: EntityTypePerson}, "PJD"},
		{Entity{Name: "Mempalace", Type: EntityTypeProject}, "pM"},
		{Entity{Name: "", Type: EntityTypePerson}, "P"},
	}

	for _, tt := range tests {
		result := d.encodeEntity(tt.entity)
		if result != tt.expected {
			t.Errorf("encodeEntity(%v) = %s, want %s", tt.entity, result, tt.expected)
		}
	}
}

func TestGetEntityRegistry(t *testing.T) {
	d := New(nil)
	d.RegisterEntity("Alice", EntityTypePerson, nil)

	registry := d.GetEntityRegistry()

	if registry["Alice"] != EntityTypePerson {
		t.Error("Registry should contain registered entity")
	}
}

func TestGetEntityAliases(t *testing.T) {
	d := New(nil)
	d.RegisterEntity("Bob", EntityTypePerson, []string{"Bobby"})

	aliases := d.GetEntityAliases()

	if aliases["Bobby"] != "Bob" {
		t.Error("Aliases should contain registered alias")
	}
}

func TestIsPotentialName(t *testing.T) {
	tests := []struct {
		word     string
		expected bool
	}{
		{"John", true},
		{"The", false},
		{"This", false},
		{"Alice", true},
		{"We", false},
	}

	for _, tt := range tests {
		result := isPotentialName(tt.word)
		if result != tt.expected {
			t.Errorf("isPotentialName(%s) = %v, want %v", tt.word, result, tt.expected)
		}
	}
}

func TestEntityTypeDescription(t *testing.T) {
	tests := []struct {
		t        EntityType
		expected string
	}{
		{EntityTypePerson, "Person"},
		{EntityTypeProject, "Project"},
		{EntityTypeTech, "Technology"},
		{EntityTypeConcept, "Concept"},
		{EntityTypeLocation, "Location"},
		{EntityTypeDate, "Date"},
		{EntityTypeUnknown, "Unknown"},
	}

	for _, tt := range tests {
		result := entityTypeDescription(tt.t)
		if result != tt.expected {
			t.Errorf("entityTypeDescription(%s) = %s, want %s", tt.t, result, tt.expected)
		}
	}
}
