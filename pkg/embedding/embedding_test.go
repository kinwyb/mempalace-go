package embedding

import (
	"context"
	"testing"
)

func TestMockEmbedder(t *testing.T) {
	embedder := NewMockEmbedder(768)

	if embedder.Dimension() != 768 {
		t.Errorf("Expected dimension 768, got %d", embedder.Dimension())
	}

	if embedder.Model() != "mock" {
		t.Errorf("Expected model 'mock', got %s", embedder.Model())
	}
}

func TestMockEmbedderEmbed(t *testing.T) {
	embedder := NewMockEmbedder(128)

	ctx := context.Background()
	embedding, err := embedder.Embed(ctx, "test text")

	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}

	if len(embedding) != 128 {
		t.Errorf("Expected embedding length 128, got %d", len(embedding))
	}

	// Verify embedding values are not all zeros
	allZero := true
	for _, v := range embedding {
		if v != 0 {
			allZero = false
			break
		}
	}

	if allZero {
		t.Error("Embedding should not be all zeros")
	}
}

func TestMockEmbedderEmbedBatch(t *testing.T) {
	embedder := NewMockEmbedder(64)

	ctx := context.Background()
	texts := []string{"text1", "text2", "text3"}
	embeddings, err := embedder.EmbedBatch(ctx, texts)

	if err != nil {
		t.Fatalf("EmbedBatch failed: %v", err)
	}

	if len(embeddings) != 3 {
		t.Errorf("Expected 3 embeddings, got %d", len(embeddings))
	}

	for i, emb := range embeddings {
		if len(emb) != 64 {
			t.Errorf("Embedding %d: expected length 64, got %d", i, len(emb))
		}
	}
}

func TestOllamaEmbedderCreation(t *testing.T) {
	embedder := NewOllamaEmbedder("http://localhost:11434", "nomic-embed-text")

	if embedder.Model() != "nomic-embed-text" {
		t.Errorf("Expected model 'nomic-embed-text', got %s", embedder.Model())
	}

	if embedder.Dimension() != 768 {
		t.Errorf("Expected dimension 768, got %d", embedder.Dimension())
	}
}

// Note: Testing OllamaEmbedder.Embed requires a running Ollama server
// Integration tests would be needed for full coverage
