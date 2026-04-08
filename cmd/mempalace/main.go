// Package main provides the CLI entry point for MemPalace
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/kinwyb/mempalace-go/internal/config"
	"github.com/kinwyb/mempalace-go/internal/convominer"
	"github.com/kinwyb/mempalace-go/internal/layers"
	"github.com/kinwyb/mempalace-go/internal/mcp"
	"github.com/kinwyb/mempalace-go/internal/miner"
	"github.com/kinwyb/mempalace-go/internal/onboarding"
	"github.com/kinwyb/mempalace-go/internal/palace"
	"github.com/kinwyb/mempalace-go/internal/searcher"
	"github.com/kinwyb/mempalace-go/internal/split"
	"github.com/kinwyb/mempalace-go/pkg/embedding"
	"github.com/kinwyb/mempalace-go/pkg/vector"
)

var (
	// Version information
	version = "1.0.0"

	// Global flags
	palacePath string
	verbose    bool
)

// Root command
var rootCmd = &cobra.Command{
	Use:   "mempalace",
	Short: "MemPalace - A local AI memory system",
	Long: `MemPalace is a local AI memory system that stores everything
and makes it findable through semantic search.

Commands:
  mempalace init <dir>                  Detect rooms from your folder structure
  mempalace mine <dir>                  Mine project files (default)
  mempalace mine <dir> --mode convos    Mine conversation exports
  mempalace search "query"              Find anything, exact words
  mempalace wake-up                     Show L0 + L1 wake-up context
  mempalace status                      Show what's been filed
  mempalace mcp                         Start MCP server`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Setup logging
		logLevel := slog.LevelInfo
		if verbose {
			logLevel = slog.LevelDebug
		}
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: logLevel,
		})))
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&palacePath, "palace", "", "Where the palace lives (default: ~/.mempalace/palace)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// Initialize store and config
func getStoreAndConfig(ctx context.Context) (*config.Config, vector.Store, error) {
	cfg, err := config.Load(config.GetConfigPath())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Override palace path if provided
	if palacePath != "" {
		cfg.PalacePath = palacePath
	}

	// Ensure palace path exists
	if err := cfg.EnsurePalacePath(); err != nil {
		return nil, nil, fmt.Errorf("failed to create palace directory: %w", err)
	}

	// Create embedder based on provider
	var embedder embedding.Embedder
	switch cfg.EmbeddingProvider {
	case "openai":
		embedder = embedding.NewOpenAIEmbedder(cfg.OpenAIAPIKey, cfg.EmbeddingAPIBase, cfg.EmbeddingModel)
	default:
		// Default to ollama
		embedder = embedding.NewOllamaEmbedder(cfg.OllamaHost, cfg.EmbeddingModel)
	}

	// Create vector store
	dbPath := cfg.PalacePath + "/palace.db"
	store, err := vector.NewSQLiteStore(dbPath, embedder)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create store: %w", err)
	}

	// Initialize store
	if err := store.Initialize(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to initialize store: %w", err)
	}

	return cfg, store, nil
}

// init command - detect rooms from folder structure
var initCmd = &cobra.Command{
	Use:   "init <directory>",
	Short: "Detect rooms from your folder structure",
	Long:  `Analyze a directory structure to suggest wing/room configuration.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		cfg, store, err := getStoreAndConfig(ctx)
		if err != nil {
			return err
		}
		defer store.Close()

		dir := args[0]
		m := miner.New(cfg, store)
		projConfig, err := m.LoadProjectConfig(dir)
		if err != nil {
			return err
		}

		fmt.Printf("\n=== Detected Configuration for %s ===\n\n", dir)
		fmt.Printf("Wing: %s\n\n", projConfig.Wing)

		if len(projConfig.Rooms) > 0 {
			fmt.Println("Rooms:")
			for _, room := range projConfig.Rooms {
				fmt.Printf("  - %s: %s\n", room.Name, room.Description)
				if len(room.Keywords) > 0 {
					fmt.Printf("    Keywords: %s\n", strings.Join(room.Keywords, ", "))
				}
			}
		} else {
			fmt.Println("No rooms detected. Creating default 'general' room.")
		}

		fmt.Printf("\nYou can create a mempalace.yaml in the directory to customize.\n")
		return nil
	},
}

// mine command - mine files into the palace
var mineMode string
var mineDryRun bool
var mineLimit int
var mineWing string

var mineCmd = &cobra.Command{
	Use:   "mine <directory>",
	Short: "Mine files into the palace",
	Long:  `Mine project files or conversation exports into the memory palace.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		cfg, store, err := getStoreAndConfig(ctx)
		if err != nil {
			return err
		}
		defer store.Close()

		dir := args[0]

		if mineMode == "convos" {
			// Mine conversation files
			m := convominer.New(store)
			m.SetDryRun(mineDryRun)
			m.SetLimit(mineLimit)
			m.SetExtractMode("exchange")

			return m.Mine(ctx, dir, mineWing)
		}

		// Mine project files
		m := miner.New(cfg, store)
		m.SetDryRun(mineDryRun)
		m.SetLimit(mineLimit)

		return m.Mine(ctx, dir, mineWing)
	},
}

func init() {
	mineCmd.Flags().StringVar(&mineMode, "mode", "files", "Mining mode (files or convos)")
	mineCmd.Flags().BoolVar(&mineDryRun, "dry-run", false, "Show what would be mined without filing")
	mineCmd.Flags().IntVar(&mineLimit, "limit", 0, "Limit number of files to process")
	mineCmd.Flags().StringVar(&mineWing, "wing", "", "Override wing name")
}

// search command - find content
var searchWing string
var searchRoom string
var searchLimit int

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Find anything, exact words",
	Long:  `Search the memory palace for relevant content.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		_, store, err := getStoreAndConfig(ctx)
		if err != nil {
			return err
		}
		defer store.Close()

		query := args[0]
		s := searcher.New(store)
		s.SetLimit(searchLimit)

		resp := s.Search(ctx, query, searchWing, searchRoom, searchLimit)

		if resp.Error != "" {
			return fmt.Errorf("search error: %s", resp.Error)
		}

		fmt.Println(searcher.FormatResults(resp.Results))

		if resp.Hint != "" {
			fmt.Printf("\nHint: %s\n", resp.Hint)
		}

		return nil
	},
}

func init() {
	searchCmd.Flags().StringVar(&searchWing, "wing", "", "Filter by wing")
	searchCmd.Flags().StringVar(&searchRoom, "room", "", "Filter by room")
	searchCmd.Flags().IntVarP(&searchLimit, "limit", "l", 10, "Max results")
}

// status command - show palace status
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show what's been filed",
	Long:  `Display current palace status and statistics.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		_, store, err := getStoreAndConfig(ctx)
		if err != nil {
			return err
		}
		defer store.Close()

		p := palace.New(store)
		stats, err := p.GetStats(ctx)
		if err != nil {
			return err
		}

		fmt.Println(palace.FormatStats(stats))
		return nil
	},
}

// wake-up command - show L0 + L1 context
var wakeUpCmd = &cobra.Command{
	Use:   "wake-up",
	Short: "Show L0 + L1 wake-up context",
	Long:  `Display the identity and essential context for current session.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		_, store, err := getStoreAndConfig(ctx)
		if err != nil {
			return err
		}
		defer store.Close()

		l := layers.New(store)
		content, err := l.WakeUp(ctx)
		if err != nil {
			return err
		}

		fmt.Println(content)
		return nil
	},
}

// mcp command - start MCP server
var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server for AI tool integration",
	Long:  `Start the Model Context Protocol server for AI tool integration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithCancel(context.Background())

		// Handle shutdown gracefully
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		cfg, store, err := getStoreAndConfig(ctx)
		if err != nil {
			cancel()
			return err
		}

		server := mcp.NewServer(cfg, store)

		// Start server in goroutine
		go func() {
			if err := server.Start(ctx); err != nil {
				slog.Error("MCP server error", "error", err)
			}
			cancel()
		}()

		// Wait for signal
		select {
		case <-sigChan:
			slog.Info("Received shutdown signal")
			server.Stop()
			store.Close()
			cancel()
		case <-ctx.Done():
			store.Close()
		}

		return nil
	},
}

// split command - split large files
var splitMinSessions int

var splitCmd = &cobra.Command{
	Use:   "split <file>",
	Short: "Split concatenated transcript mega-files into per-session files",
	Long:  `Split large transcript files into individual session files.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]

		s := split.NewSplitter()
		s.SetMinSessions(splitMinSessions)

		outputFiles, err := s.Split(filePath)
		if err != nil {
			return err
		}

		fmt.Printf("\nSplit %s into %d files:\n", filePath, len(outputFiles))
		for _, f := range outputFiles {
			fmt.Printf("  - %s\n", f)
		}

		return nil
	},
}

func init() {
	splitCmd.Flags().IntVar(&splitMinSessions, "min-sessions", 2, "Minimum sessions required")
}

// compress command - compress using AAAK dialect
var compressLevel int
var compressWing string
var compressRoom string

var compressCmd = &cobra.Command{
	Use:   "compress",
	Short: "Compress drawers using AAAK Dialect",
	Long:  `Compress content in drawers using the AAAK compression dialect.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		_, store, err := getStoreAndConfig(ctx)
		if err != nil {
			return err
		}
		defer store.Close()

		// Placeholder - dialect compression not fully implemented yet
		fmt.Println("AAAK compression is not yet fully implemented.")
		fmt.Println("This will compress content in the specified wing/room.")
		fmt.Printf("\nWing: %s\n", compressWing)
		fmt.Printf("Room: %s\n", compressRoom)
		fmt.Printf("Level: %d\n", compressLevel)

		return nil
	},
}

func init() {
	compressCmd.Flags().IntVar(&compressLevel, "level", 2, "Compression level (0-3)")
	compressCmd.Flags().StringVar(&compressWing, "wing", "", "Wing to compress")
	compressCmd.Flags().StringVar(&compressRoom, "room", "", "Room to compress")
	compressCmd.MarkFlagRequired("wing")
}

// setup command - run onboarding
var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Run first-time setup",
	Long:  `Run interactive first-time configuration setup.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.DefaultConfig()
		configPath := config.GetConfigPath()

		o := onboarding.New(cfg, configPath)
		return o.Run()
	},
}

// Add commands to root
func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(mineCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(wakeUpCmd)
	rootCmd.AddCommand(mcpCmd)
	rootCmd.AddCommand(splitCmd)
	rootCmd.AddCommand(compressCmd)
	rootCmd.AddCommand(setupCmd)
}
