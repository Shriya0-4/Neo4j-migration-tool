package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/shriya0_4/graphmigrate/cmd/config"
	"github.com/shriya0_4/graphmigrate/cmd/db"
	"github.com/shriya0_4/graphmigrate/cmd/driver"

	"github.com/shriya0_4/graphmigrate/cmd/internal/loader"
	"github.com/shriya0_4/graphmigrate/cmd/internal/logger"
	"github.com/shriya0_4/graphmigrate/cmd/internal/runner"
	"github.com/shriya0_4/graphmigrate/migrations"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Apply all pending migrations to the database",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		// ✅ Load config
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		// ✅ Create driver
		drv, err := driver.Connect(ctx, cfg)
		if err != nil {
			return err
		}
		defer drv.Close(ctx)

		// ✅ Logger
		log := logger.New(false)

		// ✅ Ensure constraints
		if err := db.EnsureConstraints(ctx, drv, cfg.Database); err != nil {
			return err
		}

		// ✅ Load migrations
		migrations, err := loader.Load(cfg.MigrationsDir)
		if err != nil {
			return err
		}

		// ✅ Get applied
		applied, err := db.GetApplied(ctx, drv, cfg.Database)
		if err != nil {
			return err
		}

		// ✅ Checksum validation
		for _, m := range migrations {
			if a, ok := applied[m.Version]; ok {
				currentChecksum, err := loader.Checksum(m.Filepath)
				if err != nil {
					log.Warn("could not compute checksum for %s: %v", m.Name, err)
					continue
				}
				if a.Checksum != "" && currentChecksum != a.Checksum {
					log.ChecksumMismatch(m.Version, m.Name)
				}
			}
		}

		// ✅ Filter pending
		var pending []migration.Migration
		for _, m := range migrations {
			if _, ok := applied[m.Version]; !ok {
				pending = append(pending, m)
			}
		}

		log.MigrationStart(len(migrations), len(pending))

		// ✅ Run migrations
		r := runner.New(drv, cfg.Database, cfg.DryRun, log)
		count, err := r.RunPending(ctx, pending)
		if err != nil {
			return err
		}

		log.Summary(count, cfg.DryRun)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
}
