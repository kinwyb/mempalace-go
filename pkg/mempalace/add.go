package mempalace

import (
	"time"
)

// Document represents content to be stored.
type Document struct {
	ID       string
	Content  string
	Wing     string
	Room     string
	Source   string
	Metadata map[string]any
}

// AddResult represents the result of an add operation.
type AddResult struct {
	ID        string
	Wing      string
	Room      string
	CreatedAt time.Time
}

// AddOption is a functional option for add operations.
type AddOption func(*addConfig)

type addConfig struct {
	wing         string
	room         string
	source       string
	layer        Layer
	metadata     map[string]any
	checkDup     bool
	dupThreshold float64
	autoDetect   bool
	detectPath   string
}

// WithWingForAdd specifies the wing for content storage.
func WithWingForAdd(wing string) AddOption {
	return func(ac *addConfig) {
		ac.wing = wing
	}
}

// WithRoomForAdd specifies the room for content storage.
func WithRoomForAdd(room string) AddOption {
	return func(ac *addConfig) {
		ac.room = room
	}
}

// WithSource specifies the source file for the content.
func WithSource(source string) AddOption {
	return func(ac *addConfig) {
		ac.source = source
	}
}

// WithAutoDetect automatically detects wing and room from source path.
func WithAutoDetect(sourcePath string) AddOption {
	return func(ac *addConfig) {
		ac.autoDetect = true
		ac.detectPath = sourcePath
	}
}

// WithLayer specifies which memory layer to store in.
func WithLayer(layer Layer) AddOption {
	return func(ac *addConfig) {
		ac.layer = layer
	}
}

// WithMetadata adds custom metadata to the document.
func WithMetadata(meta map[string]any) AddOption {
	return func(ac *addConfig) {
		ac.metadata = meta
	}
}

// WithDuplicateCheck enables duplicate checking before adding.
func WithDuplicateCheck(threshold float64) AddOption {
	return func(ac *addConfig) {
		ac.checkDup = true
		ac.dupThreshold = threshold
	}
}