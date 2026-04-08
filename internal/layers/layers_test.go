package layers

import (
	"testing"
)

func TestLayerConfigs(t *testing.T) {
	configs := LayerConfigs

	if len(configs) != 4 {
		t.Errorf("Expected 4 layer configs, got %d", len(configs))
	}

	// Verify L0 config
	l0 := configs[L0]
	if l0.Name != "Identity" {
		t.Errorf("L0 name should be 'Identity', got %s", l0.Name)
	}
	if l0.MaxTokens <= 0 {
		t.Error("L0 MaxTokens should be positive")
	}

	// Verify L1 config
	l1 := configs[L1]
	if l1.Name != "Essential Story" {
		t.Errorf("L1 name should be 'Essential Story', got %s", l1.Name)
	}

	// Verify L2 config
	l2 := configs[L2]
	if l2.Name != "On-Demand" {
		t.Errorf("L2 name should be 'On-Demand', got %s", l2.Name)
	}

	// Verify L3 config
	l3 := configs[L3]
	if l3.Name != "Deep Search" {
		t.Errorf("L3 name should be 'Deep Search', got %s", l3.Name)
	}
}

func TestLayerPriority(t *testing.T) {
	// L0 should have highest priority
	if LayerConfigs[L0].Priority <= LayerConfigs[L1].Priority {
		t.Error("L0 should have higher priority than L1")
	}

	// L1 should have higher priority than L2
	if LayerConfigs[L1].Priority <= LayerConfigs[L2].Priority {
		t.Error("L1 should have higher priority than L2")
	}

	// L2 should have higher priority than L3
	if LayerConfigs[L2].Priority <= LayerConfigs[L3].Priority {
		t.Error("L2 should have higher priority than L3")
	}
}

func TestLayerMaxTokens(t *testing.T) {
	// L0 should have smallest token limit
	if LayerConfigs[L0].MaxTokens >= LayerConfigs[L1].MaxTokens {
		t.Error("L0 should have smaller MaxTokens than L1")
	}

	// L3 should have largest token limit
	if LayerConfigs[L3].MaxTokens <= LayerConfigs[L2].MaxTokens {
		t.Error("L3 should have larger MaxTokens than L2")
	}
}

func TestAutoClassify(t *testing.T) {
	tests := []struct {
		content  string
		expected Layer
	}{
		{
			content:  "My name is John. I am a software engineer.",
			expected: L0,
		},
		{
			content:  "I am currently working on a new project.",
			expected: L0, // "i am" matches first
		},
		{
			content:  "This week we are developing a new feature.",
			expected: L1,
		},
		{
			content:  "Random technical notes and observations.",
			expected: L2,
		},
	}

	// Create Layers with nil store for testing classification logic
	l := &Layers{configs: LayerConfigs}

	for _, tt := range tests {
		result := l.AutoClassify(tt.content)
		if result != tt.expected {
			t.Errorf("AutoClassify(%q) = %d, want %d", tt.content[:min(30, len(tt.content))], result, tt.expected)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
