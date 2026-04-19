package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/shriya0_4/graphmigrate/cmd/db"
	"github.com/shriya0_4/graphmigrate/cmd/internal/loader"
	"github.com/shriya0_4/graphmigrate/migrations"
	"github.com/shriya0_4/graphmigrate/cmd/internal/runner"
	"github.com/spf13/cobra"
)

var rollbackTarget int
var forceRollback bool

var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Roll back applied migrations to a target version",
	Long: `Roll back migrations from the latest applied version down to (but not including) --to.

Example: if versions 1,2,3,4,5 are applied and you run:
  graphmigrate rollback --to 3

Versions 5 and 4 will be rolled back (their .down.cypher files executed).
Version 3 remains applied.

Requires .down.cypher files for every migration being rolled back.
Use --force to skip the confirmation prompt.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		defer drv.Close(ctx)

		if rollbackTarget < 0 {
			return fmt.Errorf("--to must be a non-negative version number (use 0 to roll back everything)")
		}

		// Load migration files (needed for DownPath)
		allMigrations, err := loader.Load(cfg.MigrationsDir)
		if err != nil {
			return err
		}

		// Build a map for quick lookup by version
		migrationByVersion := make(map[int]migration.Migration)
		for _, m := range allMigrations {
			migrationByVersion[m.Version] = m
		}

		// Fetch applied versions from Neo4j
		appliedMap, err := db.GetApplied(ctx, drv, cfg.Database)
		if err != nil {
			return err
		}

		// Build the list of applied migrations that are above the target, with DownPath filled in
		var toRollback []migration.Migration
		for _, m := range allMigrations {
			if _, ok := appliedMap[m.Version]; ok && m.Version > rollbackTarget {
				toRollback = append(toRollback, m)
			}
		}

		if len(toRollback) == 0 {
			fmt.Printf("\n  Nothing to roll back above version %04d.\n\n", rollbackTarget)
			return nil
		}

		// Preview what will be rolled back
		fmt.Printf("\n  The following migration(s) will be rolled back:\n")
		for i := len(toRollback) - 1; i >= 0; i-- {
			m := toRollback[i]
			downStatus := "✓ has .down.cypher"
			if m.DownPath == "" {
				downStatus = "✗ NO .down.cypher — will fail"
			}
			fmt.Printf("    ← %04d  %-42s  %s\n", m.Version, m.Name, downStatus)
		}
		fmt.Println()

		// Confirm unless --force or --dry-run
		if !forceRollback && !cfg.DryRun {
			fmt.Print("  Continue? [y/N]: ")
			reader := bufio.NewReader(os.Stdin)
			input, _ := reader.ReadString('\n')
			if !strings.EqualFold(strings.TrimSpace(input), "y") {
				fmt.Println("  Aborted.")
				return nil
			}
		}

		r := runner.New(drv, cfg.Database, cfg.DryRun, log)
		count, err := r.Rollback(ctx, toRollback, rollbackTarget)
		if err != nil {
			return err
		}

		log.Summary(count, cfg.DryRun)
		if !cfg.DryRun && count > 0 {
			fmt.Printf("  Database is now at version %04d.\n\n", rollbackTarget)
		}

		return nil
	},
}

func init() {
	rollbackCmd.Flags().IntVar(&rollbackTarget, "to", -1,
		"target version to roll back to (required) — migrations above this version will be rolled back")
	rollbackCmd.Flags().BoolVar(&forceRollback, "force", false,
		"skip the confirmation prompt")
	_ = rollbackCmd.MarkFlagRequired("to")

	rootCmd.AddCommand(rollbackCmd)
}
