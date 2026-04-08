// Package mempalace provides a public Go API for MemPalace operations.
//
// MemPalace is a local AI memory system that stores everything and makes it
// findable through semantic search. This package allows Go projects to
// integrate MemPalace functionality directly without using MCP.
//
// # Basic Usage
//
// Create a Palace instance and use it for storing and searching:
//
//	ctx := context.Background()
//	palace, err := mempalace.NewWithDefaults(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer palace.Close()
//
//	// Search for content
//	result, err := palace.Search(ctx, "database connection error",
//	    mempalace.WithLimit(10))
//
//	// Search with a precomputed vector
//	result, err = palace.SearchByVector(ctx, embeddingVector,
//	    mempalace.WithWing("myproject"),
//	    mempalace.WithLimit(10))
//
// # Configuration Options
//
// Use options to customize the Palace:
//
//	palace, err := mempalace.New(ctx,
//	    mempalace.WithOpenAI("sk-...", "", "text-embedding-3-small"),
//	    mempalace.WithPalacePath("/data/mypalace"),
//	    mempalace.WithChunkSize(1000, 200, 100),
//	)
//
// # Adding Content
//
// Store content with wing/room organization:
//
//	result, err := palace.Add(ctx, "Important decision: We chose PostgreSQL",
//	    mempalace.WithWingForAdd("myproject"),
//	    mempalace.WithRoomForAdd("decisions"),
//	    mempalace.WithMetadata(map[string]any{"priority": "high"}),
//	)
//
// # Memory Layers
//
// MemPalace uses a 4-layer memory stack:
//
//	- L0 (Identity): Core identity, essential story, critical preferences
//	- L1 (Essential): Project context, current goals, recent decisions
//	- L2 (On-Demand): Search-based retrieval when needed
//	- L3 (Deep Search): Comprehensive search with full context
//
// Get wake-up context (L0 + L1):
//
//	wakeUp, err := palace.WakeUp(ctx)
//
// # Mining Project Files
//
// Mine files from a project directory:
//
//	result, err := palace.Mine(ctx, "/path/to/myproject",
//	    mempalace.WithWingOverride("myproject"),
//	)
//
// # Mining Conversations
//
// Mine conversation exports (Claude, ChatGPT, etc.):
//
//	result, err := palace.MineConversations(ctx, "/path/to/exports",
//	    mempalace.WithWingOverride("conversations"),
//	)
package mempalace