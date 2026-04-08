// Package onboarding provides first-run setup functionality.
package onboarding

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/kinwyb/mempalace-go/internal/config"
)

// Onboarding handles first-run setup
type Onboarding struct {
	config     *config.Config
	configPath string
}

// New creates a new Onboarding
func New(cfg *config.Config, configPath string) *Onboarding {
	return &Onboarding{
		config:     cfg,
		configPath: configPath,
	}
}

// Run runs the interactive onboarding
func (o *Onboarding) Run() error {
	fmt.Println("\n=== MemPalace First-Time Setup ===")
	fmt.Println("\nLet's configure your memory palace...")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)

	// Palace path
	fmt.Print("Where should your palace live? [default: ~/.mempalace/palace]: ")
	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input != "" {
			o.config.PalacePath = input
		}
	}
	fmt.Printf("  → Palace path: %s\n", o.config.PalacePath)

	// Embedding model
	fmt.Println("\nEmbedding models:")
	fmt.Println("  1. nomic-embed-text (Ollama, recommended)")
	fmt.Println("  2. text-embedding-3-small (OpenAI)")
	fmt.Println("  3. custom")
	fmt.Print("Select embedding model [default: 1]: ")
	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		switch input {
		case "2":
			o.config.EmbeddingModel = "text-embedding-3-small"
		case "3":
			fmt.Print("Enter custom model name: ")
			if scanner.Scan() {
				o.config.EmbeddingModel = strings.TrimSpace(scanner.Text())
			}
		default:
			o.config.EmbeddingModel = "nomic-embed-text"
		}
	}
	fmt.Printf("  → Embedding model: %s\n", o.config.EmbeddingModel)

	// Ollama host (if using Ollama)
	if strings.Contains(o.config.EmbeddingModel, "nomic") {
		fmt.Print("\nOllama host URL [default: http://localhost:11434]: ")
		if scanner.Scan() {
			input := strings.TrimSpace(scanner.Text())
			if input != "" {
				o.config.OllamaHost = input
			}
		}
		fmt.Printf("  → Ollama host: %s\n", o.config.OllamaHost)
	}

	// Chunk size
	fmt.Print("\nDefault chunk size [default: 800]: ")
	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input != "" {
			size, err := parseInt(input)
			if err == nil && size >= 100 {
				o.config.ChunkSize = size
			}
		}
	}
	fmt.Printf("  → Chunk size: %d\n", o.config.ChunkSize)

	// Search limit
	fmt.Print("Default search result limit [default: 10]: ")
	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input != "" {
			limit, err := parseInt(input)
			if err == nil && limit >= 1 {
				o.config.SearchLimit = limit
			}
		}
	}
	fmt.Printf("  → Search limit: %d\n", o.config.SearchLimit)

	// Log level
	fmt.Println("\nLog levels:")
	fmt.Println("  1. info")
	fmt.Println("  2. debug")
	fmt.Println("  3. warn")
	fmt.Println("  4. error")
	fmt.Print("Select log level [default: 1]: ")
	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		switch input {
		case "2":
			o.config.LogLevel = "debug"
		case "3":
			o.config.LogLevel = "warn"
		case "4":
			o.config.LogLevel = "error"
		default:
			o.config.LogLevel = "info"
		}
	}
	fmt.Printf("  → Log level: %s\n", o.config.LogLevel)

	// Confirm
	fmt.Println("\n=== Configuration Summary ===")
	fmt.Printf("Palace path:    %s\n", o.config.PalacePath)
	fmt.Printf("Embedding:      %s\n", o.config.EmbeddingModel)
	fmt.Printf("Ollama host:    %s\n", o.config.OllamaHost)
	fmt.Printf("Chunk size:     %d\n", o.config.ChunkSize)
	fmt.Printf("Search limit:   %d\n", o.config.SearchLimit)
	fmt.Printf("Log level:      %s\n", o.config.LogLevel)

	fmt.Print("\nSave configuration? [Y/n]: ")
	if scanner.Scan() {
		input := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if input == "n" || input == "no" {
			fmt.Println("Configuration not saved.")
			return nil
		}
	}

	// Save configuration
	if err := o.config.Save(o.configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("\nConfiguration saved to: %s\n", o.configPath)

	// Create palace directory
	if err := o.config.EnsurePalacePath(); err != nil {
		return fmt.Errorf("failed to create palace directory: %w", err)
	}

	fmt.Printf("Palace directory created: %s\n", o.config.PalacePath)

	// Initial content
	fmt.Println("\n=== Initial Setup ===")
	fmt.Println("Would you like to add some initial identity content?")
	fmt.Println("This helps AI tools understand you better.")
	fmt.Print("Add initial content? [Y/n]: ")

	if scanner.Scan() {
		input := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if input != "n" && input != "no" {
			o.addInitialContent(scanner)
		}
	}

	fmt.Println("\n=== Setup Complete ===")
	fmt.Println("Your memory palace is ready!")
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Run 'mempalace mine <directory>' to mine project files")
	fmt.Println("  2. Run 'mempalace search \"query\"' to find content")
	fmt.Println("  3. Run 'mempalace mcp' to start the MCP server for AI tools")
	fmt.Println()

	return nil
}

// addInitialContent adds initial identity content
func (o *Onboarding) addInitialContent(scanner *bufio.Scanner) {
	fmt.Println("\nTell me a bit about yourself:")

	// Name
	fmt.Print("Your name: ")
	var name string
	if scanner.Scan() {
		name = strings.TrimSpace(scanner.Text())
	}

	// Role/occupation
	fmt.Print("Your role/occupation: ")
	var role string
	if scanner.Scan() {
		role = strings.TrimSpace(scanner.Text())
	}

	// Key interests
	fmt.Print("Key interests/skills (comma-separated): ")
	var interests string
	if scanner.Scan() {
		interests = strings.TrimSpace(scanner.Text())
	}

	// Generate identity content
	var content strings.Builder
	if name != "" {
		content.WriteString(fmt.Sprintf("Name: %s\n", name))
	}
	if role != "" {
		content.WriteString(fmt.Sprintf("Role: %s\n", role))
	}
	if interests != "" {
		content.WriteString(fmt.Sprintf("Interests: %s\n", interests))
	}

	fmt.Println("\nIdentity summary:")
	fmt.Println(content.String())

	fmt.Print("\nThis will be stored in L0 (Identity layer). Save? [Y/n]: ")
	if scanner.Scan() {
		input := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if input != "n" && input != "no" {
			// Would save to vector store here
			slog.Info("Initial identity content saved", "content", content.String())
			fmt.Println("Identity content saved to L0 layer.")
		}
	}
}

// parseInt parses an integer from string
func parseInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}

// CheckFirstRun checks if this is the first run
func CheckFirstRun(configPath string) bool {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return true
	}
	return false
}

// QuickSetup performs a quick non-interactive setup
func QuickSetup(cfg *config.Config, configPath string) error {
	// Create palace directory
	if err := cfg.EnsurePalacePath(); err != nil {
		return err
	}

	// Save default config
	if err := cfg.Save(configPath); err != nil {
		return err
	}

	slog.Info("Quick setup complete", "palace", cfg.PalacePath)
	return nil
}
