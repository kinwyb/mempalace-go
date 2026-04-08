package mempalace_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/kinwyb/mempalace-go/pkg/embedding"
	"github.com/kinwyb/mempalace-go/pkg/mempalace"
)

// tempDir creates a unique temporary directory for each example.
func tempDir(name string) string {
	dir := filepath.Join(os.TempDir(), "mempalace-examples", name, fmt.Sprintf("%d", os.Getpid()))
	os.RemoveAll(dir)
	os.MkdirAll(dir, 0755)
	return dir
}

// ExampleNew demonstrates creating a Palace with default configuration.
func ExampleNew() {
	ctx := context.Background()

	// Create a palace with mock embedder for testing
	palace, err := mempalace.New(ctx,
		mempalace.WithPalacePath(tempDir("new")),
		mempalace.WithEmbedder(embedding.NewMockEmbedder(768)),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer palace.Close()

	fmt.Println("Palace created successfully")
	// Output: Palace created successfully
}

// ExamplePalace_Search demonstrates searching for content.
func ExamplePalace_Search() {
	ctx := context.Background()

	palace, err := mempalace.New(ctx,
		mempalace.WithPalacePath(tempDir("search")),
		mempalace.WithEmbedder(embedding.NewMockEmbedder(768)),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer palace.Close()

	// Add some content first
	_, err = palace.Add(ctx, "Go is a programming language designed at Google",
		mempalace.WithWingForAdd("tech"),
		mempalace.WithRoomForAdd("languages"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Search for content
	result, err := palace.Search(ctx, "programming",
		mempalace.WithWing("tech"),
		mempalace.WithLimit(5),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d results\n", result.Total)
	// Output: Found 1 results
}

// ExamplePalace_Add demonstrates adding content to the palace.
func ExamplePalace_Add() {
	ctx := context.Background()

	palace, err := mempalace.New(ctx,
		mempalace.WithPalacePath(tempDir("add")),
		mempalace.WithEmbedder(embedding.NewMockEmbedder(768)),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer palace.Close()

	// Add content with metadata
	result, err := palace.Add(ctx, "Decision: Use PostgreSQL for the main database",
		mempalace.WithWingForAdd("myproject"),
		mempalace.WithRoomForAdd("decisions"),
		mempalace.WithMetadata(map[string]any{
			"priority": "high",
			"date":     "2024-01-15",
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Stored in %s/%s\n", result.Wing, result.Room)
	// Output: Stored in myproject/decisions
}

// ExamplePalace_WakeUp demonstrates getting wake-up context.
func ExamplePalace_WakeUp() {
	ctx := context.Background()

	palace, err := mempalace.New(ctx,
		mempalace.WithPalacePath(tempDir("wakeup")),
		mempalace.WithEmbedder(embedding.NewMockEmbedder(768)),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer palace.Close()

	// Store identity information in L0
	err = palace.StoreInLayer(ctx, mempalace.L0,
		"I am a Go developer who prefers clean architecture",
		mempalace.WithWingForLayer("identity"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Get wake-up context (L0 + L1)
	wakeUp, err := palace.WakeUp(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Wake-up context available")
	if len(wakeUp) > 0 {
		fmt.Println("Has content")
	}
	// Output:
	// Wake-up context available
	// Has content
}

// ExamplePalace_GetStats demonstrates getting palace statistics.
func ExamplePalace_GetStats() {
	ctx := context.Background()

	palace, err := mempalace.New(ctx,
		mempalace.WithPalacePath(tempDir("stats")),
		mempalace.WithEmbedder(embedding.NewMockEmbedder(768)),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer palace.Close()

	// Add some content
	_, err = palace.Add(ctx, "First document",
		mempalace.WithWingForAdd("wing1"),
		mempalace.WithRoomForAdd("room1"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Get statistics
	stats, err := palace.GetStats(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Total documents: %d\n", stats.TotalDocuments)
	// Output: Total documents: 1
}

// ExampleLayer demonstrates memory layer constants.
func ExampleLayer() {
	fmt.Println(mempalace.L0.String())
	fmt.Println(mempalace.L1.String())
	fmt.Println(mempalace.L2.String())
	fmt.Println(mempalace.L3.String())
	// Output:
	// L0-Identity
	// L1-Essential
	// L2-OnDemand
	// L3-DeepSearch
}

// ExampleChunkText demonstrates text chunking.
func ExampleChunkText() {
	content := "This is a long paragraph about databases. " +
		"PostgreSQL is a powerful open-source database. " +
		"It supports advanced features like JSON and full-text search."

	chunks := mempalace.ChunkText(content, 50, 10, 20)

	fmt.Printf("Created %d chunks\n", len(chunks))
	// Output: Created 4 chunks
}