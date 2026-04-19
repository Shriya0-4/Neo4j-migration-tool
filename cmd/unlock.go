package cmd

import (
	"context"
	"fmt"

	"github.com/shriya0_4/graphmigrate/cmd/db"

	"github.com/spf13/cobra"
)

var unlockCmd = &cobra.Command{
	Use:   "unlock",
	Short: "Remove a stale migration lock from the database",
	Long: `If a migration run was interrupted (process killed, crash, etc.),
the MigrationLock node may not have been cleaned up.

This command removes it so future migrations can proceed.
Only run this if you are sure no other migration is currently running.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		defer drv.Close(ctx)

		if err := db.ReleaseLock(ctx, drv, cfg.Database); err != nil {
			return fmt.Errorf("failed to release lock: %w", err)
		}

		log.Info("migration lock released successfully")
		fmt.Println("\n  ✓ Migration lock removed. You may now run migrations.\n")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(unlockCmd)
}
