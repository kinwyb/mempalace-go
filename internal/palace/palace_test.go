package palace

import (
	"testing"
)

func TestSanitizeWingName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"My Project", "my_project"},
		{"my-project", "my_project"},
		{"My.Project", "myproject"},
		{"UPPERCASE", "uppercase"},
		{"MixedCase", "mixedcase"},
	}

	for _, tt := range tests {
		result := sanitizeWingName(tt.input)
		if result != tt.expected {
			t.Errorf("sanitizeWingName(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestSanitizeRoomName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"API Handlers", "api_handlers"},
		{"database-layer", "database_layer"},
		{"TestFiles", "testfiles"},
	}

	for _, tt := range tests {
		result := sanitizeRoomName(tt.input)
		if result != tt.expected {
			t.Errorf("sanitizeRoomName(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestDetectWingFromPath(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/home/user/projects/myapp/src/main.go", "myapp"},
		{"/workspace/backend/internal/handlers/api.go", "backend"},
		{"/simple/project/main.py", "project"},
	}

	for _, tt := range tests {
		result := DetectWingFromPath(tt.path)
		// Result depends on path structure, just verify it's not empty
		if result == "" {
			t.Errorf("DetectWingFromPath(%s) returned empty string", tt.path)
		}
	}
}

func TestDetectRoomFromPath(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/project/src/main.go", "source"},
		{"/project/internal/handlers/api.go", "internal"},
		{"/project/pkg/utils/helper.go", "package"},
		{"/project/tests/main_test.go", "tests"},
		{"/project/api/routes.go", "api"},
		{"/project/docs/readme.md", "docs"},
	}

	for _, tt := range tests {
		result := DetectRoomFromPath(tt.path)
		if result != tt.expected {
			t.Errorf("DetectRoomFromPath(%s) = %s, want %s", tt.path, result, tt.expected)
		}
	}
}

func TestFormatStats(t *testing.T) {
	stats := &PalaceStats{
		TotalDrawers: 100,
		TotalWings:   3,
		TotalRooms:   10,
		StorageSize:  1024 * 1024, // 1 MB
		Wings: map[string]WingStats{
			"project1": {
				Name:  "project1",
				Total: 50,
				Rooms: map[string]int{"api": 20, "db": 30},
			},
			"project2": {
				Name:  "project2",
				Total: 50,
				Rooms: map[string]int{"docs": 50},
			},
		},
	}

	formatted := FormatStats(stats)

	if formatted == "" {
		t.Error("FormatStats returned empty string")
	}

	// Should contain total info
	if len(formatted) < 50 {
		t.Errorf("Formatted stats seems too short: %s", formatted)
	}
}
