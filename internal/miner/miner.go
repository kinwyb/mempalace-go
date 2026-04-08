// Package miner provides project file mining functionality.
package miner

import (
	"bufio"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kinwyb/mempalace-go/internal/config"
	"github.com/kinwyb/mempalace-go/pkg/vector"
	"gopkg.in/yaml.v3"
)

// Readable file extensions
var ReadableExtensions = map[string]bool{
	".txt": true, ".md": true, ".py": true, ".js": true, ".ts": true,
	".jsx": true, ".tsx": true, ".json": true, ".yaml": true, ".yml": true,
	".html": true, ".css": true, ".java": true, ".go": true, ".rs": true,
	".rb": true, ".sh": true, ".csv": true, ".sql": true, ".toml": true,
}

// Directories to skip
var SkipDirs = map[string]bool{
	".git": true, "node_modules": true, "__pycache__": true,
	".venv": true, "venv": true, "env": true, "dist": true,
	"build": true, ".next": true, "coverage": true, ".mempalace": true,
	".ruff_cache": true, ".mypy_cache": true, ".pytest_cache": true,
	".cache": true, ".tox": true, ".nox": true, ".idea": true,
	".vscode": true, ".ipynb_checkpoints": true, ".eggs": true,
	"htmlcov": true, "target": true, "vendor": true,
}

// Skip filenames
var SkipFilenames = map[string]bool{
	"mempalace.yaml": true, "mempalace.yml": true,
	"mempal.yaml": true, "mempal.yml": true,
	".gitignore": true, "package-lock.json": true,
}

const (
	ChunkSize    = 800
	ChunkOverlap = 100
	MinChunkSize = 50
)

// ProjectConfig represents the mempalace.yaml configuration
type ProjectConfig struct {
	Wing  string `yaml:"wing"`
	Rooms []Room `yaml:"rooms"`
}

// Room represents a room configuration
type Room struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Keywords    []string `yaml:"keywords"`
}

// Miner handles project file mining
type Miner struct {
	config *config.Config
	store  vector.Store
	dryRun bool
	limit  int
}

// New creates a new Miner
func New(cfg *config.Config, store vector.Store) *Miner {
	return &Miner{config: cfg, store: store}
}

// SetDryRun sets dry run mode
func (m *Miner) SetDryRun(dryRun bool) *Miner {
	m.dryRun = dryRun
	return m
}

// SetLimit sets file limit
func (m *Miner) SetLimit(limit int) *Miner {
	m.limit = limit
	return m
}

// Mine mines a project directory
func (m *Miner) Mine(ctx context.Context, projectDir string, wingOverride string) error {
	projectPath, err := filepath.Abs(os.ExpandEnv(projectDir))
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Load project config
	projConfig, err := m.LoadProjectConfig(projectPath)
	if err != nil {
		return err
	}

	wing := wingOverride
	if wing == "" {
		wing = projConfig.Wing
	}
	if wing == "" {
		wing = filepath.Base(projectPath)
	}

	// Scan files
	files := m.ScanProject(projectPath)
	if m.limit > 0 && len(files) > m.limit {
		files = files[:m.limit]
	}

	slog.Info("Mining project",
		"wing", wing,
		"rooms", len(projConfig.Rooms),
		"files", len(files),
		"palace", m.config.PalacePath,
	)

	if m.dryRun {
		slog.Info("DRY RUN — nothing will be filed")
	}

	// Process files
	totalDrawers := 0
	for i, file := range files {
		drawers, err := m.processFile(ctx, file, projectPath, wing, projConfig.Rooms)
		if err != nil {
			slog.Warn("failed to process file", "file", file, "error", err)
			continue
		}
		totalDrawers += drawers
		if !m.dryRun && drawers > 0 {
			slog.Info("Processed file", "progress", fmt.Sprintf("%d/%d", i+1, len(files)), "file", filepath.Base(file), "drawers", drawers)
		}
	}

	slog.Info("Mining complete",
		"files_processed", len(files),
		"total_drawers", totalDrawers,
	)

	return nil
}

// LoadProjectConfig loads mempalace.yaml from project directory
func (m *Miner) LoadProjectConfig(projectPath string) (*ProjectConfig, error) {
	configPath := filepath.Join(projectPath, "mempalace.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Try legacy name
		configPath = filepath.Join(projectPath, "mempal.yaml")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			return &ProjectConfig{
				Wing:  filepath.Base(projectPath),
				Rooms: []Room{{Name: "general", Description: "All project files"}},
			}, nil
		}
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var projConfig ProjectConfig
	if err := yaml.Unmarshal(data, &projConfig); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &projConfig, nil
}

// ScanProject scans a project directory for readable files
func (m *Miner) ScanProject(projectPath string) []string {
	var files []string
	filepath.WalkDir(projectPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		// Skip directories
		if d.IsDir() {
			if SkipDirs[d.Name()] || strings.HasSuffix(d.Name(), ".egg-info") {
				return fs.SkipDir
			}
			return nil
		}

		// Skip specific filenames
		if SkipFilenames[d.Name()] {
			return nil
		}

		// Check extension
		ext := strings.ToLower(filepath.Ext(path))
		if !ReadableExtensions[ext] {
			return nil
		}

		files = append(files, path)
		return nil
	})
	return files
}

// processFile processes a single file
func (m *Miner) processFile(ctx context.Context, filePath, projectPath, wing string, rooms []Room) (int, error) {
	// Read file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return 0, err
	}

	text := strings.TrimSpace(string(content))
	if len(text) < MinChunkSize {
		return 0, nil
	}

	// Detect room
	room := DetectRoom(filePath, text, rooms, projectPath)

	// Chunk content
	chunks := ChunkText(text, filePath)

	if m.dryRun {
		slog.Info("DRY RUN", "file", filepath.Base(filePath), "room", room, "drawers", len(chunks))
		return len(chunks), nil
	}

	// Store chunks
	drawersAdded := 0
	for _, chunk := range chunks {
		doc := vector.Document{
			ID:      m.generateDrawerID(wing, room, filePath, chunk.Index),
			Content: chunk.Content,
			Metadata: map[string]any{
				"wing":        wing,
				"room":        room,
				"source_file": filePath,
				"chunk_index": chunk.Index,
				"added_by":    "mempalace",
				"filed_at":    time.Now().Format(time.RFC3339),
			},
		}

		if err := m.store.Add(ctx, []vector.Document{doc}); err != nil {
			slog.Warn("failed to add drawer", "error", err)
			continue
		}
		drawersAdded++
	}

	return drawersAdded, nil
}

// generateDrawerID generates a unique ID for a drawer
func (m *Miner) generateDrawerID(wing, room, filePath string, chunkIndex int) string {
	hash := md5.Sum([]byte(filePath + string(rune(chunkIndex))))
	return fmt.Sprintf("drawer_%s_%s_%s", wing, room, hex.EncodeToString(hash[:])[:16])
}

// Chunk represents a text chunk
type Chunk struct {
	Content string
	Index   int
}

// ChunkText splits text into chunks
func ChunkText(content, sourceFile string) []Chunk {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil
	}

	var chunks []Chunk
	start := 0
	index := 0

	for start < len(content) {
		end := start + ChunkSize
		if end > len(content) {
			end = len(content)
		}

		// Try to break at paragraph boundary
		if end < len(content) {
			if newlinePos := strings.LastIndex(content[start:end], "\n\n"); newlinePos > ChunkSize/2 {
				end = start + newlinePos
			} else if newlinePos := strings.LastIndex(content[start:end], "\n"); newlinePos > ChunkSize/2 {
				end = start + newlinePos
			}
		}

		chunk := strings.TrimSpace(content[start:end])
		if len(chunk) >= MinChunkSize {
			chunks = append(chunks, Chunk{Content: chunk, Index: index})
			index++
		}

		if end >= len(content) {
			break
		}
		start = end - ChunkOverlap
		if start < 0 {
			start = 0
		}
	}

	return chunks
}

// DetectRoom determines which room a file belongs to
func DetectRoom(filePath, content string, rooms []Room, projectPath string) string {
	relative, _ := filepath.Rel(projectPath, filePath)
	relative = strings.ToLower(relative)
	filename := strings.ToLower(filepath.Base(filePath))
	contentLower := strings.ToLower(content[:min(2000, len(content))])

	// Priority 1: folder path matches room name
	pathParts := strings.Split(strings.ReplaceAll(relative, "\\", "/"), "/")
	for _, part := range pathParts[:len(pathParts)-1] {
		for _, room := range rooms {
			candidates := append([]string{strings.ToLower(room.Name)}, room.Keywords...)
			for _, c := range candidates {
				if strings.Contains(part, c) || strings.Contains(c, part) {
					return room.Name
				}
			}
		}
	}

	// Priority 2: filename matches room name
	for _, room := range rooms {
		if strings.Contains(filename, strings.ToLower(room.Name)) {
			return room.Name
		}
	}

	// Priority 3: keyword scoring
	scores := make(map[string]int)
	for _, room := range rooms {
		keywords := append(room.Keywords, room.Name)
		for _, kw := range keywords {
			scores[room.Name] += strings.Count(contentLower, strings.ToLower(kw))
		}
	}

	if len(scores) > 0 {
		best := ""
		maxScore := 0
		for room, score := range scores {
			if score > maxScore {
				maxScore = score
				best = room
			}
		}
		if maxScore > 0 {
			return best
		}
	}

	return "general"
}

// Status shows palace status
func Status(palacePath string) error {
	fmt.Printf("\nPalace status for: %s\n", palacePath)
	fmt.Println("Use 'mempalace status' command for full status")
	return nil
}

// SplitMegaFile splits a large file by session markers
func SplitMegaFile(filePath string, minSessions int) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var sessions []string
	var currentSession strings.Builder
	sessionCount := 0

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "## Session") {
			if currentSession.Len() > 0 {
				sessions = append(sessions, currentSession.String())
				currentSession.Reset()
				sessionCount++
			}
			continue
		}
		currentSession.WriteString(line)
		currentSession.WriteString("\n")
	}

	if currentSession.Len() > 0 {
		sessions = append(sessions, currentSession.String())
	}

	if sessionCount < minSessions {
		return nil, fmt.Errorf("only %d sessions found, minimum is %d", sessionCount, minSessions)
	}

	return sessions, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
