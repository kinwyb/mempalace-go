package split

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewSplitter(t *testing.T) {
	s := NewSplitter()

	if s == nil {
		t.Fatal("NewSplitter returned nil")
	}

	if s.minSessions != 2 {
		t.Errorf("Expected default minSessions 2, got %d", s.minSessions)
	}
}

func TestSetMinSessions(t *testing.T) {
	s := NewSplitter()
	s.SetMinSessions(5)

	if s.minSessions != 5 {
		t.Errorf("Expected minSessions 5, got %d", s.minSessions)
	}
}

func TestSetOutputDir(t *testing.T) {
	s := NewSplitter()
	s.SetOutputDir("/custom/output")

	if s.outputDir != "/custom/output" {
		t.Errorf("Expected outputDir /custom/output, got %s", s.outputDir)
	}
}

func TestSetSessionMarker(t *testing.T) {
	s := NewSplitter()
	s.SetSessionMarker("===")

	if s.sessionMarker != "===" {
		t.Errorf("Expected sessionMarker '===', got %s", s.sessionMarker)
	}
}

func TestSplit(t *testing.T) {
	// Create temp file
	content := `First session content here.

---
Second session content here.

---
Third session content here.`

	tmpDir, err := os.MkdirTemp("", "mempalace-split-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	inputPath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(inputPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	s := NewSplitter()
	s.SetMinSessions(1)

	outputFiles, err := s.Split(inputPath)
	if err != nil {
		t.Fatalf("Split failed: %v", err)
	}

	if len(outputFiles) == 0 {
		t.Error("Expected at least one output file")
	}

	// Verify output files exist
	for _, f := range outputFiles {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			t.Errorf("Output file does not exist: %s", f)
		}
	}
}

func TestSplitNotEnoughSessions(t *testing.T) {
	// Create temp file with only one session
	content := `Only one session here.`

	tmpDir, err := os.MkdirTemp("", "mempalace-split-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	inputPath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(inputPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	s := NewSplitter()
	s.SetMinSessions(2)

	_, err = s.Split(inputPath)
	if err == nil {
		t.Error("Expected error for insufficient sessions")
	}
}

func TestIsDateBoundary(t *testing.T) {
	tests := []struct {
		line     string
		expected bool
	}{
		{"# January 15, 2024", true},
		{"## February Meeting", true},
		{"* March 2024", true},
		{"Regular content", false},
		{"---", false},
	}

	for _, tt := range tests {
		result := isDateBoundary(tt.line)
		if result != tt.expected {
			t.Errorf("isDateBoundary(%q) = %v, want %v", tt.line, result, tt.expected)
		}
	}
}

func TestDetectSessions(t *testing.T) {
	content := `First paragraph.

---
Second paragraph.

## Session 3
Third paragraph.`

	boundaries := DetectSessions(content)

	if len(boundaries) == 0 {
		t.Error("Expected to detect session boundaries")
	}

	t.Logf("Detected %d boundaries", len(boundaries))
}

func TestSplitIntoFiles(t *testing.T) {
	// Create large content
	content := string(make([]byte, 5000)) // 5000 bytes

	tmpDir, err := os.MkdirTemp("", "mempalace-split-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	inputPath := filepath.Join(tmpDir, "large.txt")
	if err := os.WriteFile(inputPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	outputFiles, err := SplitIntoFiles(inputPath, 1000)
	if err != nil {
		t.Fatalf("SplitIntoFiles failed: %v", err)
	}

	if len(outputFiles) < 2 {
		t.Errorf("Expected at least 2 files, got %d", len(outputFiles))
	}
}

func TestSplitIntoFilesTooSmall(t *testing.T) {
	// Create small content
	content := "Small content"

	tmpDir, err := os.MkdirTemp("", "mempalace-split-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	inputPath := filepath.Join(tmpDir, "small.txt")
	if err := os.WriteFile(inputPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	_, err = SplitIntoFiles(inputPath, 1000)
	if err == nil {
		t.Error("Expected error for file smaller than chunk size")
	}
}
