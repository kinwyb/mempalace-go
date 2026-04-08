package mempalace

// Layer represents a memory layer in the 4-layer stack.
type Layer int

const (
	// L0 - Identity layer: core identity, essential story, critical preferences.
	// Always active, provides the foundation for all interactions.
	L0 Layer = 0
	// L1 - Essential layer: project context, current goals, recent decisions.
	// Context window layer, loaded at wake-up.
	L1 Layer = 1
	// L2 - On-Demand layer: search-based retrieval when needed.
	// Query-triggered retrieval for specific topics.
	L2 Layer = 2
	// L3 - Deep Search layer: comprehensive search with full context.
	// Maximum search scope for thorough investigation.
	L3 Layer = 3
)

// String returns the name of the layer.
func (l Layer) String() string {
	switch l {
	case L0:
		return "L0-Identity"
	case L1:
		return "L1-Essential"
	case L2:
		return "L2-OnDemand"
	case L3:
		return "L3-DeepSearch"
	default:
		return "Unknown"
	}
}

// LayerInfo contains information about a layer.
type LayerInfo struct {
	Name        string
	Description string
	MaxTokens   int
	Priority    int
	Rooms       []string
}

// DefaultLayerConfigs returns the default configuration for all layers.
func DefaultLayerConfigs() map[Layer]LayerInfo {
	return map[Layer]LayerInfo{
		L0: {
			Name:        "Identity",
			Description: "Core identity, essential story, critical preferences",
			MaxTokens:   500,
			Priority:    100,
			Rooms:       []string{"identity", "about_me", "core", "preferences"},
		},
		L1: {
			Name:        "Essential Story",
			Description: "Project context, current goals, recent decisions",
			MaxTokens:   1000,
			Priority:    80,
			Rooms:       []string{"project_context", "goals", "recent", "active"},
		},
		L2: {
			Name:        "On-Demand",
			Description: "Search-based retrieval when needed",
			MaxTokens:   2000,
			Priority:    50,
			Rooms:       []string{},
		},
		L3: {
			Name:        "Deep Search",
			Description: "Comprehensive search with full context",
			MaxTokens:   5000,
			Priority:    20,
			Rooms:       []string{},
		},
	}
}

// LayerOption is a functional option for layer operations.
type LayerOption func(*layerConfig)

type layerConfig struct {
	wing     string
	room     string
	priority int
}

// WithWingForLayer specifies the wing for layer storage.
func WithWingForLayer(wing string) LayerOption {
	return func(lc *layerConfig) {
		lc.wing = wing
	}
}

// WithRoomForLayer specifies the room for layer storage.
func WithRoomForLayer(room string) LayerOption {
	return func(lc *layerConfig) {
		lc.room = room
	}
}

// WithPriority sets the priority for layer content.
func WithPriority(priority int) LayerOption {
	return func(lc *layerConfig) {
		lc.priority = priority
	}
}