package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/shriya0_4/graphmigrate/cmd/config"
	neo4jdriver "github.com/shriya0_4/graphmigrate/cmd/driver"
	"github.com/shriya0_4/graphmigrate/cmd/internal/logger"
	"github.com/spf13/cobra"
)

// globalFlags holds the values of flags shared across all subcommands.
type globalFlags struct {
	verbose bool
	dryRun  bool
	dir     string
}

var flags globalFlags

// shared state passed down to subcommands
var (
	cfg *config.Config
	drv neo4j.DriverWithContext
	log *logger.Logger
)

// rootCmd is the base command — running graphmigrate with no subcommand prints help.
var rootCmd = &cobra.Command{
	Use:   "graphmigrate",
	Short: "A versioned Neo4j schema migration tool",
	Long: `GraphMigrate manages and applies versioned Cypher migrations
against a Neo4j database — think Flyway, but for graphs.

Migrations live in a directory as numbered .cypher files:
  0001_add_genre_nodes.cypher
  0001_add_genre_nodes.down.cypher

Each migration is applied exactly once and tracked in the database.`,
	Version: "1.0.0",
	// PersistentPreRunE runs before every subcommand — sets up logger, config, driver.
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// skip setup for the unlock command (it has its own minimal setup)
		log = logger.New(flags.verbose)

		var err error
		cfg, err = config.Load()
		if err != nil {
			return fmt.Errorf("configuration error: %w", err)
		}

		// override MigrationsDir if --dir flag was set
		if flags.dir != "" {
			cfg.MigrationsDir = flags.dir
		}
		cfg.Verbose = flags.verbose
		cfg.DryRun = flags.dryRun

		ctx := context.Background()
		drv, err = neo4jdriver.Connect(ctx, cfg)
		if err != nil {
			return err
		}

		log.Info("connected to neo4j", "url", cfg.URL, "database", cfg.Database)
		return nil
	},
}

// Execute is the entry point called from main.go.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Global persistent flags available to all subcommands
	rootCmd.PersistentFlags().BoolVarP(&flags.verbose, "verbose", "v", false,
		"enable debug logging (shows Cypher statements, timing details)")
	rootCmd.PersistentFlags().BoolVarP(&flags.dryRun, "dry-run", "n", false,
		"preview what would be applied/rolled back without making any changes")
	rootCmd.PersistentFlags().StringVarP(&flags.dir, "dir", "d", "",
		"path to migrations directory (overrides MIGRATIONS_DIR env var)")
}
