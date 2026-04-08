// Package mcp implements the MCP (Model Context Protocol) server.
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/kinwyb/mempalace-go/internal/config"
	"github.com/kinwyb/mempalace-go/internal/layers"
	"github.com/kinwyb/mempalace-go/internal/searcher"
	"github.com/kinwyb/mempalace-go/pkg/vector"
)

// Server implements an MCP server for MemPalace
type Server struct {
	config  *config.Config
	store   vector.Store
	search  *searcher.Searcher
	layers  *layers.Layers
	running bool
	mu      sync.RWMutex
}

// NewServer creates a new MCP server
func NewServer(cfg *config.Config, store vector.Store) *Server {
	return &Server{
		config: cfg,
		store:  store,
		search: searcher.New(store),
		layers: layers.New(store),
	}
}

// Start starts the MCP server
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	s.running = true
	s.mu.Unlock()

	slog.Info("Starting MCP server", "palace", s.config.PalacePath)

	// Use stdio transport for MCP
	scanner := bufio.NewScanner(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var request MCPRequest
		if err := json.Unmarshal([]byte(line), &request); err != nil {
			slog.Warn("failed to parse request", "error", err)
			s.sendError(encoder, "parse_error", err.Error())
			continue
		}

		// Handle the request
		response := s.handleRequest(ctx, request)

		// Send response
		if err := encoder.Encode(response); err != nil {
			slog.Warn("failed to send response", "error", err)
		}
	}

	return scanner.Err()
}

// Stop stops the MCP server
func (s *Server) Stop() {
	s.mu.Lock()
	s.running = false
	s.mu.Unlock()

	slog.Info("MCP server stopped")
}

// handleRequest handles an MCP request
func (s *Server) handleRequest(ctx context.Context, request MCPRequest) MCPResponse {
	switch request.Method {
	case "initialize":
		return s.handleInitialize(request)
	case "tools/list":
		return s.handleToolsList(request)
	case "tools/call":
		return s.handleToolsCall(ctx, request)
	case "resources/list":
		return s.handleResourcesList(request)
	case "resources/read":
		return s.handleResourcesRead(ctx, request)
	default:
		return MCPResponse{
			Error: &MCPError{
				Code:    -32601,
				Message: fmt.Sprintf("Method not found: %s", request.Method),
			},
		}
	}
}

// handleInitialize handles the initialize request
func (s *Server) handleInitialize(request MCPRequest) MCPResponse {
	return MCPResponse{
		Result: map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]any{
				"tools": map[string]any{
					"listChanged": false,
				},
				"resources": map[string]any{
					"list": true,
					"read": true,
				},
			},
			"serverInfo": map[string]any{
				"name":    "mempalace",
				"version": "1.0.0",
			},
		},
	}
}

// handleToolsList handles the tools/list request
func (s *Server) handleToolsList(request MCPRequest) MCPResponse {
	tools := s.getToolDefinitions()
	return MCPResponse{
		Result: map[string]any{
			"tools": tools,
		},
	}
}

// handleToolsCall handles the tools/call request
func (s *Server) handleToolsCall(ctx context.Context, request MCPRequest) MCPResponse {
	params, ok := request.Params.(map[string]any)
	if !ok {
		return MCPResponse{
			Error: &MCPError{
				Code:    -32602,
				Message: "Invalid params",
			},
		}
	}

	toolName, ok := params["name"].(string)
	if !ok {
		return MCPResponse{
			Error: &MCPError{
				Code:    -32602,
				Message: "Tool name required",
			},
		}
	}

	toolArgs, _ := params["arguments"].(map[string]any)

	result := s.executeTool(ctx, toolName, toolArgs)

	return MCPResponse{
		Result: map[string]any{
			"content": []map[string]any{
				{
					"type": "text",
					"text": result,
				},
			},
		},
	}
}

// handleResourcesList handles the resources/list request
func (s *Server) handleResourcesList(request MCPRequest) MCPResponse {
	stats, err := s.store.GetStats(context.Background())
	if err != nil {
		return MCPResponse{
			Error: &MCPError{
				Code:    -32603,
				Message: err.Error(),
			},
		}
	}

	var resources []map[string]any
	for wing := range stats.WingRoomCounts {
		resources = append(resources, map[string]any{
			"uri":  fmt.Sprintf("mempalace://wing/%s", wing),
			"name": wing,
			"type": "wing",
		})
	}

	return MCPResponse{
		Result: map[string]any{
			"resources": resources,
		},
	}
}

// handleResourcesRead handles the resources/read request
func (s *Server) handleResourcesRead(ctx context.Context, request MCPRequest) MCPResponse {
	params, ok := request.Params.(map[string]any)
	if !ok {
		return MCPResponse{
			Error: &MCPError{
				Code:    -32602,
				Message: "Invalid params",
			},
		}
	}

	uri, ok := params["uri"].(string)
	if !ok {
		return MCPResponse{
			Error: &MCPError{
				Code:    -32602,
				Message: "URI required",
			},
		}
	}

	content := s.readResource(ctx, uri)

	return MCPResponse{
		Result: map[string]any{
			"contents": []map[string]any{
				{
					"uri":      uri,
					"mimeType": "text/plain",
					"text":     content,
				},
			},
		},
	}
}

// getToolDefinitions returns all tool definitions
func (s *Server) getToolDefinitions() []map[string]any {
	return []map[string]any{
		{
			"name":        "search",
			"description": "Search the memory palace for relevant content",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{"type": "string", "description": "Search query"},
					"wing":  map[string]any{"type": "string", "description": "Optional wing filter"},
					"room":  map[string]any{"type": "string", "description": "Optional room filter"},
					"limit": map[string]any{"type": "integer", "description": "Max results (default 10)"},
				},
				"required": []string{"query"},
			},
		},
		{
			"name":        "wake_up",
			"description": "Get L0 + L1 wake-up context for current session",
			"inputSchema": map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			"name":        "add_drawer",
			"description": "Add a new drawer to the memory palace",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"wing":        map[string]any{"type": "string", "description": "Wing name"},
					"room":        map[string]any{"type": "string", "description": "Room name"},
					"content":     map[string]any{"type": "string", "description": "Content to store"},
					"source_file": map[string]any{"type": "string", "description": "Optional source file"},
					"added_by":    map[string]any{"type": "string", "description": "Who added it"},
				},
				"required": []string{"wing", "room", "content"},
			},
		},
		{
			"name":        "get_status",
			"description": "Get current palace status and statistics",
			"inputSchema": map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			"name":        "list_wings",
			"description": "List all wings in the palace",
			"inputSchema": map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			"name":        "list_rooms",
			"description": "List all rooms in a wing",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"wing": map[string]any{"type": "string", "description": "Wing name"},
				},
				"required": []string{"wing"},
			},
		},
		{
			"name":        "check_duplicate",
			"description": "Check if content is similar to existing content",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"content":   map[string]any{"type": "string", "description": "Content to check"},
					"threshold": map[string]any{"type": "number", "description": "Similarity threshold (0-1)"},
				},
				"required": []string{"content"},
			},
		},
		{
			"name":        "get_layer_content",
			"description": "Get content from a specific memory layer (L0-L3)",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"layer": map[string]any{"type": "integer", "description": "Layer number (0-3)"},
					"query": map[string]any{"type": "string", "description": "Optional search query"},
				},
				"required": []string{"layer"},
			},
		},
		{
			"name":        "delete_drawer",
			"description": "Delete a drawer by ID",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{"type": "string", "description": "Drawer ID"},
				},
				"required": []string{"id"},
			},
		},
		{
			"name":        "compress_drawer",
			"description": "Compress a drawer using AAAK dialect",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":    map[string]any{"type": "string", "description": "Drawer ID"},
					"level": map[string]any{"type": "integer", "description": "Compression level (0-3)"},
				},
				"required": []string{"id"},
			},
		},
		{
			"name":        "register_entity",
			"description": "Register a new entity (person, project, etc.)",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name":        map[string]any{"type": "string", "description": "Entity name"},
					"type":        map[string]any{"type": "string", "description": "Entity type (person, project, etc.)"},
					"description": map[string]any{"type": "string", "description": "Optional description"},
					"aliases":     map[string]any{"type": "array", "description": "Optional aliases"},
				},
				"required": []string{"name", "type"},
			},
		},
		{
			"name":        "detect_entities",
			"description": "Detect entities in provided content",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"content": map[string]any{"type": "string", "description": "Content to analyze"},
				},
				"required": []string{"content"},
			},
		},
		{
			"name":        "get_taxonomy",
			"description": "Get the taxonomy of stored content",
			"inputSchema": map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			"name":        "mine_file",
			"description": "Mine a single file into the palace",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"file_path": map[string]any{"type": "string", "description": "File path to mine"},
					"wing":      map[string]any{"type": "string", "description": "Wing to store in"},
					"room":      map[string]any{"type": "string", "description": "Room to store in"},
				},
				"required": []string{"file_path"},
			},
		},
		{
			"name":        "batch_add",
			"description": "Add multiple drawers at once",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"wing":     map[string]any{"type": "string", "description": "Wing name"},
					"room":     map[string]any{"type": "string", "description": "Room name"},
					"contents": map[string]any{"type": "array", "description": "Contents to store"},
				},
				"required": []string{"wing", "room", "contents"},
			},
		},
		{
			"name":        "get_recent",
			"description": "Get recently added drawers",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"limit": map[string]any{"type": "integer", "description": "Max results"},
					"wing":  map[string]any{"type": "string", "description": "Optional wing filter"},
				},
			},
		},
		{
			"name":        "get_drawer",
			"description": "Get a specific drawer by ID",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{"type": "string", "description": "Drawer ID"},
				},
				"required": []string{"id"},
			},
		},
		{
			"name":        "update_drawer",
			"description": "Update an existing drawer",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":      map[string]any{"type": "string", "description": "Drawer ID"},
					"content": map[string]any{"type": "string", "description": "New content"},
				},
				"required": []string{"id", "content"},
			},
		},
		{
			"name":        "detect_room",
			"description": "Detect the appropriate room for content",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"wing":    map[string]any{"type": "string", "description": "Wing name"},
					"content": map[string]any{"type": "string", "description": "Content to analyze"},
				},
				"required": []string{"wing", "content"},
			},
		},
		{
			"name":        "store_layer",
			"description": "Store content in a specific memory layer",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"layer":   map[string]any{"type": "integer", "description": "Layer number (0-3)"},
					"wing":    map[string]any{"type": "string", "description": "Wing name"},
					"room":    map[string]any{"type": "string", "description": "Room name"},
					"content": map[string]any{"type": "string", "description": "Content to store"},
				},
				"required": []string{"layer", "wing", "content"},
			},
		},
	}
}

// executeTool executes a tool and returns the result
func (s *Server) executeTool(ctx context.Context, toolName string, args map[string]any) string {
	switch toolName {
	case "search":
		return s.toolSearch(ctx, args)
	case "wake_up":
		return s.toolWakeUp(ctx, args)
	case "add_drawer":
		return s.toolAddDrawer(ctx, args)
	case "get_status":
		return s.toolGetStatus(ctx, args)
	case "list_wings":
		return s.toolListWings(ctx, args)
	case "list_rooms":
		return s.toolListRooms(ctx, args)
	case "check_duplicate":
		return s.toolCheckDuplicate(ctx, args)
	case "get_layer_content":
		return s.toolGetLayerContent(ctx, args)
	case "delete_drawer":
		return s.toolDeleteDrawer(ctx, args)
	case "compress_drawer":
		return s.toolCompressDrawer(ctx, args)
	case "register_entity":
		return s.toolRegisterEntity(ctx, args)
	case "detect_entities":
		return s.toolDetectEntities(ctx, args)
	case "get_taxonomy":
		return s.toolGetTaxonomy(ctx, args)
	case "mine_file":
		return s.toolMineFile(ctx, args)
	case "batch_add":
		return s.toolBatchAdd(ctx, args)
	case "get_recent":
		return s.toolGetRecent(ctx, args)
	case "get_drawer":
		return s.toolGetDrawer(ctx, args)
	case "update_drawer":
		return s.toolUpdateDrawer(ctx, args)
	case "detect_room":
		return s.toolDetectRoom(ctx, args)
	case "store_layer":
		return s.toolStoreLayer(ctx, args)
	default:
		return fmt.Sprintf("Unknown tool: %s", toolName)
	}
}

// Tool implementations

func (s *Server) toolSearch(ctx context.Context, args map[string]any) string {
	query, _ := args["query"].(string)
	limit := 10
	if v, ok := args["limit"].(float64); ok {
		limit = int(v)
	}
	wing, _ := args["wing"].(string)
	room, _ := args["room"].(string)

	resp := s.search.Search(ctx, query, wing, room, limit)
	return searcher.FormatResults(resp.Results)
}

func (s *Server) toolWakeUp(ctx context.Context, args map[string]any) string {
	content, err := s.layers.WakeUp(ctx)
	if err != nil {
		return fmt.Sprintf("Error: %s", err)
	}
	return content
}

func (s *Server) toolAddDrawer(ctx context.Context, args map[string]any) string {
	wing, _ := args["wing"].(string)
	room, _ := args["room"].(string)
	content, _ := args["content"].(string)
	sourceFile, _ := args["source_file"].(string)
	addedBy, _ := args["added_by"].(string)
	if addedBy == "" {
		addedBy = "mcp"
	}

	// Check duplicate
	dup, err := s.search.CheckDuplicate(ctx, content, 0.9)
	if err == nil && dup.IsDuplicate {
		return fmt.Sprintf("Skipped: duplicate detected (similar content exists)")
	}

	doc := vector.Document{
		ID:      fmt.Sprintf("drawer_%s_%s_%d", wing, room, time.Now().UnixNano()),
		Content: content,
		Metadata: map[string]any{
			"wing":        wing,
			"room":        room,
			"source_file": sourceFile,
			"added_by":    addedBy,
			"filed_at":    time.Now().Format(time.RFC3339),
		},
	}

	if err := s.store.Add(ctx, []vector.Document{doc}); err != nil {
		return fmt.Sprintf("Error: %s", err)
	}

	return fmt.Sprintf("Added drawer %s in %s/%s", doc.ID, wing, room)
}

func (s *Server) toolGetStatus(ctx context.Context, args map[string]any) string {
	stats, err := s.store.GetStats(ctx)
	if err != nil {
		return fmt.Sprintf("Error: %s", err)
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Total drawers: %d\n", stats.TotalDocuments))
	builder.WriteString(fmt.Sprintf("Total wings: %d\n", stats.TotalWings))
	builder.WriteString(fmt.Sprintf("Total rooms: %d\n", stats.TotalRooms))
	builder.WriteString(fmt.Sprintf("Storage size: %.2f MB\n", float64(stats.StorageSize)/1024/1024))

	return builder.String()
}

func (s *Server) toolListWings(ctx context.Context, args map[string]any) string {
	wings, err := s.search.GetWings(ctx)
	if err != nil {
		return fmt.Sprintf("Error: %s", err)
	}

	return strings.Join(wings, "\n")
}

func (s *Server) toolListRooms(ctx context.Context, args map[string]any) string {
	wing, _ := args["wing"].(string)

	rooms, err := s.search.GetRooms(ctx, wing)
	if err != nil {
		return fmt.Sprintf("Error: %s", err)
	}

	return strings.Join(rooms, "\n")
}

func (s *Server) toolCheckDuplicate(ctx context.Context, args map[string]any) string {
	content, _ := args["content"].(string)
	threshold := 0.9
	if v, ok := args["threshold"].(float64); ok {
		threshold = v
	}

	resp, err := s.search.CheckDuplicate(ctx, content, threshold)
	if err != nil {
		return mapToString(map[string]any{"error": err.Error()})
	}
	return mapToString(map[string]any{
		"is_duplicate": resp.IsDuplicate,
		"matches":      len(resp.Matches),
	})
}

func (s *Server) toolGetLayerContent(ctx context.Context, args map[string]any) string {
	layer := 0
	if v, ok := args["layer"].(float64); ok {
		layer = int(v)
	}
	query, _ := args["query"].(string)

	results, err := s.layers.Retrieve(ctx, layers.Layer(layer), query)
	if err != nil {
		return fmt.Sprintf("Error: %s", err)
	}

	return searcher.FormatResults(results)
}

func (s *Server) toolDeleteDrawer(ctx context.Context, args map[string]any) string {
	id, _ := args["id"].(string)

	if err := s.store.Delete(ctx, id); err != nil {
		return fmt.Sprintf("Error: %s", err)
	}

	return fmt.Sprintf("Deleted drawer %s", id)
}

func (s *Server) toolCompressDrawer(ctx context.Context, args map[string]any) string {
	// Placeholder - would integrate with dialect package
	return "Compression not yet implemented"
}

func (s *Server) toolRegisterEntity(ctx context.Context, args map[string]any) string {
	// Placeholder - would integrate with entity package
	return "Entity registration not yet implemented"
}

func (s *Server) toolDetectEntities(ctx context.Context, args map[string]any) string {
	// Placeholder - would integrate with entity package
	return "Entity detection not yet implemented"
}

func (s *Server) toolGetTaxonomy(ctx context.Context, args map[string]any) string {
	return mapToString(map[string]any{"taxonomy": map[string]map[string]int{}})
}

func (s *Server) toolMineFile(ctx context.Context, args map[string]any) string {
	// Placeholder - would integrate with miner package
	return "File mining not yet implemented"
}

func (s *Server) toolBatchAdd(ctx context.Context, args map[string]any) string {
	wing, _ := args["wing"].(string)
	room, _ := args["room"].(string)
	contents, _ := args["contents"].([]any)

	var docs []vector.Document
	for i, content := range contents {
		if c, ok := content.(string); ok {
			doc := vector.Document{
				ID:      fmt.Sprintf("drawer_%s_%s_%d", wing, room, i),
				Content: c,
				Metadata: map[string]any{
					"wing":     wing,
					"room":     room,
					"added_by": "mcp",
					"filed_at": time.Now().Format(time.RFC3339),
				},
			}
			docs = append(docs, doc)
		}
	}

	if err := s.store.Add(ctx, docs); err != nil {
		return fmt.Sprintf("Error: %s", err)
	}

	return fmt.Sprintf("Added %d drawers", len(docs))
}

func (s *Server) toolGetRecent(ctx context.Context, args map[string]any) string {
	// Placeholder - would need to implement in store
	return "Recent retrieval not yet implemented"
}

func (s *Server) toolGetDrawer(ctx context.Context, args map[string]any) string {
	id, _ := args["id"].(string)

	doc, err := s.store.Get(ctx, id)
	if err != nil {
		return fmt.Sprintf("Error: %s", err)
	}

	if doc == nil {
		return "Drawer not found"
	}

	return doc.Content
}

func (s *Server) toolUpdateDrawer(ctx context.Context, args map[string]any) string {
	id, _ := args["id"].(string)
	content, _ := args["content"].(string)

	doc, err := s.store.Get(ctx, id)
	if err != nil {
		return fmt.Sprintf("Error: %s", err)
	}

	if doc == nil {
		return "Drawer not found"
	}

	doc.Content = content
	if err := s.store.Add(ctx, []vector.Document{*doc}); err != nil {
		return fmt.Sprintf("Error: %s", err)
	}

	return fmt.Sprintf("Updated drawer %s", id)
}

func (s *Server) toolDetectRoom(ctx context.Context, args map[string]any) string {
	wing, _ := args["wing"].(string)
	content, _ := args["content"].(string)

	// Use searcher's GetRooms to get available rooms
	rooms, err := s.search.GetRooms(context.Background(), wing)
	if err != nil {
		return "general"
	}

	// Simple keyword-based detection
	contentLower := strings.ToLower(content)
	bestRoom := "general"
	bestScore := 0

	for _, room := range rooms {
		score := strings.Count(contentLower, strings.ToLower(room))
		if score > bestScore {
			bestScore = score
			bestRoom = room
		}
	}

	return bestRoom
}

func (s *Server) toolStoreLayer(ctx context.Context, args map[string]any) string {
	layer := 0
	if v, ok := args["layer"].(float64); ok {
		layer = int(v)
	}
	wing, _ := args["wing"].(string)
	room, _ := args["room"].(string)
	content, _ := args["content"].(string)

	if err := s.layers.Store(ctx, layers.Layer(layer), wing, room, content); err != nil {
		return fmt.Sprintf("Error: %s", err)
	}

	return fmt.Sprintf("Stored in L%d, %s/%s", layer, wing, room)
}

// readResource reads a resource by URI
func (s *Server) readResource(ctx context.Context, uri string) string {
	// Parse URI: mempalace://wing/<wing> or mempalace://wing/<wing>/room/<room>
	parts := strings.Split(strings.TrimPrefix(uri, "mempalace://"), "/")

	if len(parts) < 2 {
		return "Invalid URI"
	}

	switch parts[0] {
	case "wing":
		if len(parts) >= 4 && parts[2] == "room" {
			// Get room content
			wing := parts[1]
			room := parts[3]
			drawers, err := s.store.Search(ctx, "", wing, room, 100)
			if err != nil {
				return fmt.Sprintf("Error: %s", err)
			}

			var builder strings.Builder
			for _, d := range drawers {
				builder.WriteString(d.Content)
				builder.WriteString("\n---\n")
			}
			return builder.String()
		} else if len(parts) >= 2 {
			// Get wing content
			wing := parts[1]
			drawers, err := s.store.Search(ctx, "", wing, "", 100)
			if err != nil {
				return fmt.Sprintf("Error: %s", err)
			}

			var builder strings.Builder
			for _, d := range drawers {
				builder.WriteString(d.Content)
				builder.WriteString("\n---\n")
			}
			return builder.String()
		}
	}

	return "Resource not found"
}

// sendError sends an error response
func (s *Server) sendError(encoder *json.Encoder, code string, message string) {
	err := encoder.Encode(MCPResponse{
		Error: &MCPError{
			Code:    -32000,
			Message: message,
			Data:    code,
		},
	})
	if err != nil {
		slog.Warn("failed to send error", "error", err)
	}
}

// Helper function
func mapToString(m map[string]any) string {
	result, _ := json.Marshal(m)
	return string(result)
}

// MCP types

type MCPRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type MCPResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id,omitempty"`
	Result  any       `json:"result,omitempty"`
	Error   *MCPError `json:"error,omitempty"`
}

type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}
