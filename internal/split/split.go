// Package split provides transcript file splitting functionality.
package split

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Splitter handles splitting large transcript files
type Splitter struct {
	minSessions   int
	outputDir     string
	sessionMarker string
}

// NewSplitter creates a new Splitter
func NewSplitter() *Splitter {
	return &Splitter{
		minSessions:   2,
		sessionMarker: "---",
	}
}

// SetMinSessions sets the minimum sessions threshold
func (s *Splitter) SetMinSessions(min int) *Splitter {
	s.minSessions = min
	return s
}

// SetOutputDir sets the output directory
func (s *Splitter) SetOutputDir(dir string) *Splitter {
	s.outputDir = dir
	return s
}

// SetSessionMarker sets the session marker pattern
func (s *Splitter) SetSessionMarker(marker string) *Splitter {
	s.sessionMarker = marker
	return s
}

// Split splits a file into multiple session files
func (s *Splitter) Split(filePath string) ([]string, error) {
	// Read the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Determine output directory
	outputDir := s.outputDir
	if outputDir == "" {
		outputDir = filepath.Dir(filePath)
	}

	// Create output directory if needed
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Parse sessions
	scanner := bufio.NewScanner(file)
	var sessions []Session
	var currentSession *Session
	sessionNum := 0

	for scanner.Scan() {
		line := scanner.Text()

		// Check for session marker
		if strings.HasPrefix(line, s.sessionMarker) || strings.HasPrefix(line, "## Session") {
			if currentSession != nil && currentSession.Content.Len() > 0 {
				sessions = append(sessions, *currentSession)
			}
			sessionNum++
			currentSession = &Session{
				Number:    sessionNum,
				StartTime: time.Now(),
				Content:   &strings.Builder{},
			}
			continue
		}

		// Also detect date-based session boundaries
		if isDateBoundary(line) {
			if currentSession != nil && currentSession.Content.Len() > 0 {
				sessions = append(sessions, *currentSession)
			}
			sessionNum++
			currentSession = &Session{
				Number:    sessionNum,
				StartTime: parseDateFromLine(line),
				Content:   &strings.Builder{},
			}
			continue
		}

		// Add line to current session
		if currentSession == nil {
			currentSession = &Session{
				Number:    1,
				StartTime: time.Now(),
				Content:   &strings.Builder{},
			}
		}

		currentSession.Content.WriteString(line)
		currentSession.Content.WriteString("\n")
	}

	// Add final session
	if currentSession != nil && currentSession.Content.Len() > 0 {
		sessions = append(sessions, *currentSession)
	}

	if len(sessions) < s.minSessions {
		return nil, fmt.Errorf("only %d sessions found, minimum is %d", len(sessions), s.minSessions)
	}

	// Write session files
	baseName := filepath.Base(filePath)
	ext := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)

	var outputFiles []string
	for _, session := range sessions {
		outputName := fmt.Sprintf("%s_session_%d%s", nameWithoutExt, session.Number, ext)
		outputPath := filepath.Join(outputDir, outputName)

		content := session.Content.String()
		if len(strings.TrimSpace(content)) == 0 {
			continue
		}

		if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
			slog.Warn("failed to write session file", "path", outputPath, "error", err)
			continue
		}

		outputFiles = append(outputFiles, outputPath)
		slog.Info("Created session file", "path", outputPath, "session", session.Number, "size", len(content))
	}

	slog.Info("Split complete",
		"input", filePath,
		"sessions", len(sessions),
		"output_files", len(outputFiles),
	)

	return outputFiles, nil
}

// Session represents a single session
type Session struct {
	Number    int
	StartTime time.Time
	Content   *strings.Builder
}

// isDateBoundary checks if a line is a date boundary
func isDateBoundary(line string) bool {
	// Check for common date patterns
	datePatterns := []string{
		"January", "February", "March", "April", "May", "June",
		"July", "August", "September", "October", "November", "December",
	}

	lineLower := strings.ToLower(line)
	for _, month := range datePatterns {
		if strings.Contains(lineLower, strings.ToLower(month)) {
			// Check if it looks like a date header
			if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "*") {
				return true
			}
		}
	}

	return false
}

// parseDateFromLine attempts to parse a date from a line
func parseDateFromLine(line string) time.Time {
	// Try various date formats
	formats := []string{
		"January %d, %Y",
		"%d January %Y",
		"%Y-%m-%d",
		"%m/%d/%Y",
	}

	for _, format := range formats {
		t, err := time.Parse(format, line)
		if err == nil {
			return t
		}
	}

	return time.Now()
}

// SplitIntoFiles splits a large file into multiple files
func SplitIntoFiles(filePath string, chunkSize int) ([]string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	text := string(content)
	if len(text) < chunkSize {
		return nil, fmt.Errorf("file is smaller than chunk size, no splitting needed")
	}

	// Determine output directory
	outputDir := filepath.Dir(filePath)
	baseName := filepath.Base(filePath)
	ext := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)

	var outputFiles []string
	chunkNum := 1
	start := 0

	for start < len(text) {
		end := start + chunkSize
		if end > len(text) {
			end = len(text)
		}

		// Try to break at paragraph boundary
		if end < len(text) {
			// Look backwards for newline
			newlineIdx := end
			for i := end - 1; i >= start+chunkSize/2; i-- {
				if text[i] == '\n' {
					newlineIdx = i
					break
				}
			}
			if newlineIdx < end {
				end = newlineIdx + 1
			}
		}

		chunk := strings.TrimSpace(text[start:end])
		if len(chunk) > 0 {
			outputName := fmt.Sprintf("%s_part_%d%s", nameWithoutExt, chunkNum, ext)
			outputPath := filepath.Join(outputDir, outputName)

			if err := os.WriteFile(outputPath, []byte(chunk), 0644); err != nil {
				return nil, err
			}

			outputFiles = append(outputFiles, outputPath)
			chunkNum++
		}

		start = end
	}

	slog.Info("Split into files complete",
		"input", filePath,
		"output_files", len(outputFiles),
		"chunk_size", chunkSize,
	)

	return outputFiles, nil
}

// DetectSessions detects potential session boundaries in content
func DetectSessions(content string) []int {
	var boundaries []int

	lines := strings.Split(content, "\n")
	pos := 0

	for _, line := range lines {
		// Check for common session markers
		if strings.HasPrefix(line, "---") ||
			strings.HasPrefix(line, "## Session") ||
			strings.HasPrefix(line, "# Session") ||
			isDateBoundary(line) {
			boundaries = append(boundaries, pos)
		}
		pos += len(line) + 1 // +1 for newline
	}

	return boundaries
}
