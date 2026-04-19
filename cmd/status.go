package cmd

import (
	"context"
	"fmt"

	"github.com/shriya0_4/graphmigrate/cmd/db"
	"github.com/shriya0_4/graphmigrate/cmd/internal/loader"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the current status of all migrations",
	Long: `Print a table showing every migration file and whether it has been applied.

Applied migrations show their timestamp and a checksum status.
A ⚠ symbol indicates the migration file was modified after being applied.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		defer drv.Close(ctx)

		migrations, err := loader.Load(cfg.MigrationsDir)
		if err != nil {
			return err
		}

		applied, err := db.GetApplied(ctx, drv, cfg.Database)
		if err != nil {
			return err
		}

		log.StatusHeader()

		for _, m := range migrations {
			if a, ok := applied[m.Version]; ok {
				appliedAt := a.AppliedAt.Format("2006-01-02 15:04:05")

				// Checksum check
				mismatch := ""
				currentChecksum, err := loader.Checksum(m.Filepath)
				if err == nil && a.Checksum != "" && currentChecksum != a.Checksum {
					mismatch = " ⚠"
				}

				log.StatusRow(m.Version, m.Name, "applied", appliedAt+mismatch)
			} else {
				log.StatusRow(m.Version, m.Name, "pending", "—")
			}
		}

		// Footer summary
		pendingCount := 0
		for _, m := range migrations {
			if _, ok := applied[m.Version]; !ok {
				pendingCount++
			}
		}
		fmt.Printf("\n  %d applied, %d pending\n\n", len(applied), pendingCount)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
