// Package convominer provides conversation mining functionality.
package convominer

import (
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

	"github.com/kinwyb/mempalace-go/internal/normalize"
	"github.com/kinwyb/mempalace-go/pkg/vector"
)

// Conversation file extensions
var ConvoExtensions = map[string]bool{
	".txt": true, ".md": true, ".json": true, ".jsonl": true,
}

// Directories to skip
var SkipDirs = map[string]bool{
	".git": true, "node_modules": true, "__pycache__": true,
	".venv": true, "venv": true, "env": true, "dist": true,
	"build": true, ".next": true, ".mempalace": true,
	"tool-results": true, "memory": true,
}

// Topic keywords for room detection
var TopicKeywords = map[string][]string{
	"technical":    {"code", "python", "function", "bug", "error", "api", "database", "server", "deploy", "git", "test", "debug", "refactor"},
	"architecture": {"architecture", "design", "pattern", "structure", "schema", "interface", "module", "component", "service", "layer"},
	"planning":     {"plan", "roadmap", "milestone", "deadline", "priority", "sprint", "backlog", "scope", "requirement", "spec"},
	"decisions":    {"decided", "chose", "picked", "switched", "migrated", "replaced", "trade-off", "alternative", "option", "approach"},
	"problems":     {"problem", "issue", "broken", "failed", "crash", "stuck", "workaround", "fix", "solved", "resolved"},
}

const MinChunkSize = 30

// ConvoMiner handles conversation mining
type ConvoMiner struct {
	store       vector.Store
	dryRun      bool
	limit       int
	extractMode string
}

// New creates a new ConvoMiner
func New(store vector.Store) *ConvoMiner {
	return &ConvoMiner{store: store}
}

// SetDryRun sets dry run mode
func (m *ConvoMiner) SetDryRun(dryRun bool) *ConvoMiner {
	m.dryRun = dryRun
	return m
}

// SetLimit sets file limit
func (m *ConvoMiner) SetLimit(limit int) *ConvoMiner {
	m.limit = limit
	return m
}

// SetExtractMode sets extract mode (exchange or general)
func (m *ConvoMiner) SetExtractMode(mode string) *ConvoMiner {
	m.extractMode = mode
	return m
}

// Mine mines a directory of conversation files
func (m *ConvoMiner) Mine(ctx context.Context, convoDir, wingOverride string) error {
	convoPath, err := filepath.Abs(os.ExpandEnv(convoDir))
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	wing := wingOverride
	if wing == "" {
		wing = strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(filepath.Base(convoPath), " ", "_"), "-", "_"))
	}

	// Scan files
	files := m.scanConvos(convoPath)
	if m.limit > 0 && len(files) > m.limit {
		files = files[:m.limit]
	}

	slog.Info("Mining conversations",
		"wing", wing,
		"source", convoPath,
		"files", len(files),
		"mode", m.extractMode,
	)

	if m.dryRun {
		slog.Info("DRY RUN — nothing will be filed")
	}

	totalDrawers := 0
	for i, file := range files {
		drawers, err := m.processFile(ctx, file, wing)
		if err != nil {
			slog.Warn("failed to process file", "file", file, "error", err)
			continue
		}
		totalDrawers += drawers
		if !m.dryRun && drawers > 0 {
			slog.Info("Processed conversation",
				"progress", fmt.Sprintf("%d/%d", i+1, len(files)),
				"file", filepath.Base(file),
				"drawers", drawers,
			)
		}
	}

	slog.Info("Conversation mining complete",
		"files_processed", len(files),
		"total_drawers", totalDrawers,
	)

	return nil
}

// scanConvos scans a directory for conversation files
func (m *ConvoMiner) scanConvos(convoPath string) []string {
	var files []string
	filepath.WalkDir(convoPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			if SkipDirs[d.Name()] {
				return fs.SkipDir
			}
			return nil
		}

		// Skip meta files
		if strings.HasSuffix(d.Name(), ".meta.json") {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ConvoExtensions[ext] {
			files = append(files, path)
		}
		return nil
	})
	return files
}

// processFile processes a single conversation file
func (m *ConvoMiner) processFile(ctx context.Context, filePath, wing string) (int, error) {
	// Normalize format
	content, err := normalize.Normalize(filePath)
	if err != nil {
		return 0, err
	}

	if len(strings.TrimSpace(content)) < MinChunkSize {
		return 0, nil
	}

	// Chunk content
	var chunks []Chunk
	if m.extractMode == "general" {
		chunks = extractMemories(content)
	} else {
		chunks = chunkExchanges(content)
	}

	if len(chunks) == 0 {
		return 0, nil
	}

	if m.dryRun {
		slog.Info("DRY RUN", "file", filepath.Base(filePath), "drawers", len(chunks))
		return len(chunks), nil
	}

	// Store chunks
	drawersAdded := 0
	for _, chunk := range chunks {
		room := chunk.Room
		if room == "" {
			room = detectConvoRoom(chunk.Content)
		}

		doc := vector.Document{
			ID:      m.generateDrawerID(wing, room, filePath, chunk.Index),
			Content: chunk.Content,
			Metadata: map[string]any{
				"wing":         wing,
				"room":         room,
				"source_file":  filePath,
				"chunk_index":  chunk.Index,
				"added_by":     "mempalace",
				"filed_at":     time.Now().Format(time.RFC3339),
				"ingest_mode":  "convos",
				"extract_mode": m.extractMode,
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
func (m *ConvoMiner) generateDrawerID(wing, room, filePath string, chunkIndex int) string {
	hash := md5.Sum([]byte(filePath + string(rune(chunkIndex))))
	return fmt.Sprintf("drawer_%s_%s_%s", wing, room, hex.EncodeToString(hash[:])[:16])
}

// Chunk represents a conversation chunk
type Chunk struct {
	Content string
	Index   int
	Room    string
}

// chunkExchanges chunks by exchange pair (Q+A = one unit)
func chunkExchanges(content string) []Chunk {
	lines := strings.Split(content, "\n")
	quoteLines := 0
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), ">") {
			quoteLines++
		}
	}

	if quoteLines >= 3 {
		return chunkByExchange(lines)
	}
	return chunkByParagraph(content)
}

// chunkByExchange chunks by exchange pairs
func chunkByExchange(lines []string) []Chunk {
	var chunks []Chunk
	i := 0

	for i < len(lines) {
		line := lines[i]
		if strings.HasPrefix(strings.TrimSpace(line), ">") {
			userTurn := strings.TrimSpace(line)
			i++

			var aiLines []string
			for i < len(lines) {
				nextLine := lines[i]
				if strings.HasPrefix(strings.TrimSpace(nextLine), ">") || strings.HasPrefix(strings.TrimSpace(nextLine), "---") {
					break
				}
				if strings.TrimSpace(nextLine) != "" {
					aiLines = append(aiLines, strings.TrimSpace(nextLine))
				}
				i++
			}

			aiResponse := strings.Join(aiLines[:min(8, len(aiLines))], " ")
			content := userTurn
			if aiResponse != "" {
				content = userTurn + "\n" + aiResponse
			}

			if len(strings.TrimSpace(content)) > MinChunkSize {
				chunks = append(chunks, Chunk{
					Content: content,
					Index:   len(chunks),
				})
			}
		} else {
			i++
		}
	}

	return chunks
}

// chunkByParagraph chunks by paragraph breaks
func chunkByParagraph(content string) []Chunk {
	paragraphs := strings.Split(content, "\n\n")
	var chunks []Chunk

	// If no paragraph breaks, chunk by line groups
	if len(paragraphs) <= 1 && strings.Count(content, "\n") > 20 {
		lines := strings.Split(content, "\n")
		for i := 0; i < len(lines); i += 25 {
			end := i + 25
			if end > len(lines) {
				end = len(lines)
			}
			group := strings.TrimSpace(strings.Join(lines[i:end], "\n"))
			if len(group) > MinChunkSize {
				chunks = append(chunks, Chunk{Content: group, Index: len(chunks)})
			}
		}
		return chunks
	}

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if len(para) > MinChunkSize {
			chunks = append(chunks, Chunk{Content: para, Index: len(chunks)})
		}
	}

	return chunks
}

// detectConvoRoom detects room based on content keywords
func detectConvoRoom(content string) string {
	contentLower := strings.ToLower(content[:min(3000, len(content))])
	scores := make(map[string]int)

	for room, keywords := range TopicKeywords {
		for _, kw := range keywords {
			if strings.Contains(contentLower, kw) {
				scores[room]++
			}
		}
	}

	if len(scores) == 0 {
		return "general"
	}

	best := "general"
	maxScore := 0
	for room, score := range scores {
		if score > maxScore {
			maxScore = score
			best = room
		}
	}
	return best
}

// extractMemories extracts memories for general mode
func extractMemories(content string) []Chunk {
	paragraphs := strings.Split(content, "\n\n")
	var chunks []Chunk

	for i, para := range paragraphs {
		para = strings.TrimSpace(para)
		if len(para) < MinChunkSize {
			continue
		}

		room := "general"
		paraLower := strings.ToLower(para)
		if strings.Contains(paraLower, "decided") || strings.Contains(paraLower, "chose") {
			room = "decisions"
		} else if strings.Contains(paraLower, "prefer") || strings.Contains(paraLower, "like") {
			room = "preferences"
		} else if strings.Contains(paraLower, "completed") || strings.Contains(paraLower, "finished") {
			room = "milestones"
		} else if strings.Contains(paraLower, "problem") || strings.Contains(paraLower, "issue") {
			room = "problems"
		}

		chunks = append(chunks, Chunk{
			Content: para,
			Index:   i,
			Room:    room,
		})
	}

	return chunks
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
