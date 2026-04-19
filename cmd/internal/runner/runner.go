package runner

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/shriya0_4/graphmigrate/cmd/db"
	"github.com/shriya0_4/graphmigrate/cmd/internal/loader"
	"github.com/shriya0_4/graphmigrate/cmd/internal/logger"
	"github.com/shriya0_4/graphmigrate/migrations"
)

// Runner orchestrates migration execution against a Neo4j database.
type Runner struct {
	drv    neo4j.DriverWithContext
	dbName string
	dryRun bool
	log    *logger.Logger
}

// New creates a new Runner.
func New(drv neo4j.DriverWithContext, dbName string, dryRun bool, log *logger.Logger) *Runner {
	return &Runner{
		drv:    drv,
		dbName: dbName,
		dryRun: dryRun,
		log:    log,
	}
}

// RunPending executes all pending migrations in version order.
func (r *Runner) RunPending(ctx context.Context, pending []migration.Migration) (int, error) {
	if len(pending) == 0 {
		return 0, nil
	}

	if !r.dryRun {
		if err := db.AcquireLock(ctx, r.drv, r.dbName); err != nil {
			return 0, err
		}
		defer func() {
			if err := db.ReleaseLock(ctx, r.drv, r.dbName); err != nil {
				r.log.Error("failed to release migration lock", "error", err)
			}
		}()
	}

	applied := 0
	for _, m := range pending {
		r.log.MigrationRun(m.Version, m.Name, r.dryRun)
		start := time.Now()

		if err := r.runOne(ctx, m); err != nil {
			r.log.MigrationFail(m.Version, m.Name, err)
			return applied, fmt.Errorf("migration %04d_%s failed: %w", m.Version, m.Name, err)
		}

		elapsed := fmt.Sprintf("%dms", time.Since(start).Milliseconds())
		r.log.MigrationDone(m.Version, m.Name, elapsed, r.dryRun)
		applied++
	}

	return applied, nil
}

// isSchemaMigration returns true if any statement is a DDL schema operation.
// Neo4j forbids mixing schema modifications (CREATE/DROP INDEX/CONSTRAINT)
// with data writes in the same transaction — we detect this upfront so the
// runner can use separate transactions for each.
func isSchemaMigration(stmts []string) bool {
	ddlPrefixes := []string{
		"CREATE INDEX",
		"DROP INDEX",
		"CREATE CONSTRAINT",
		"DROP CONSTRAINT",
		"CREATE FULLTEXT",
		"CREATE RANGE",
		"CREATE LOOKUP",
		"CREATE TEXT",
		"CREATE POINT",
		"CREATE VECTOR",
	}
	for _, stmt := range stmts {
		upper := strings.ToUpper(strings.TrimSpace(stmt))
		for _, prefix := range ddlPrefixes {
			if strings.HasPrefix(upper, prefix) {
				return true
			}
		}
	}
	return false
}

// runOne executes a single migration's Cypher statements.
//
// Neo4j does not allow schema modifications (CREATE/DROP INDEX/CONSTRAINT)
// and data writes in the same transaction. When a migration contains DDL,
// each statement runs in its own transaction (Neo4j requires this for schema
// ops anyway), then the history record is written in a final separate
// transaction.
//
// For pure data migrations, all statements + the history record run in one
// atomic transaction.
func (r *Runner) runOne(ctx context.Context, m migration.Migration) error {
	stmts, err := loader.ReadStatements(m.Filepath)
	if err != nil {
		return err
	}

	checksum, err := loader.Checksum(m.Filepath)
	if err != nil {
		return err
	}

	if r.dryRun {
		r.log.Debug("dry-run: would execute statements", "count", len(stmts), "migration", m.Name)
		for i, s := range stmts {
			r.log.Debug(fmt.Sprintf("  statement %d: %s", i+1, truncate(s, 80)))
		}
		return nil
	}

	if isSchemaMigration(stmts) {
		return r.runSchemaOne(ctx, m, stmts, checksum)
	}
	return r.runDataOne(ctx, m, stmts, checksum)
}

// runSchemaOne handles migrations that contain DDL (CREATE/DROP INDEX/CONSTRAINT).
// Each statement runs in its own transaction. History is recorded in a final
// separate transaction after all DDL commits successfully.
func (r *Runner) runSchemaOne(ctx context.Context, m migration.Migration, stmts []string, checksum string) error {
	r.log.Debug("schema migration — using per-statement transactions", "migration", m.Name)

	for i, stmt := range stmts {
		r.log.Debug("executing schema statement", "index", i+1, "migration", m.Name)

		sess := r.drv.NewSession(ctx, neo4j.SessionConfig{DatabaseName: r.dbName})
		_, execErr := sess.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			_, err := tx.Run(ctx, stmt, nil)
			return nil, err
		})
		sess.Close(ctx)

		if execErr != nil {
			return fmt.Errorf("statement %d failed: %w\nCypher: %s", i+1, execErr, truncate(stmt, 200))
		}
	}

	// All DDL committed — now record history in its own separate transaction.
	sess := r.drv.NewSession(ctx, neo4j.SessionConfig{DatabaseName: r.dbName})
	defer sess.Close(ctx)

	_, err := sess.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return nil, db.RecordApplied(ctx, tx, m, checksum)
	})
	return err
}

// runDataOne handles pure data migrations — all statements and the history
// record run in a single atomic transaction.
func (r *Runner) runDataOne(ctx context.Context, m migration.Migration, stmts []string, checksum string) error {
	session := r.drv.NewSession(ctx, neo4j.SessionConfig{DatabaseName: r.dbName})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		for i, stmt := range stmts {
			r.log.Debug("executing statement", "index", i+1, "migration", m.Name)
			if _, err := tx.Run(ctx, stmt, nil); err != nil {
				return nil, fmt.Errorf("statement %d failed: %w\nCypher: %s", i+1, err, truncate(stmt, 200))
			}
		}
		if err := db.RecordApplied(ctx, tx, m, checksum); err != nil {
			return nil, err
		}
		return nil, nil
	})
	return err
}

// Rollback executes .down.cypher files from latest applied down to targetVersion.
func (r *Runner) Rollback(ctx context.Context, applied []migration.Migration, targetVersion int) (int, error) {
	var toRollback []migration.Migration
	for i := len(applied) - 1; i >= 0; i-- {
		if applied[i].Version > targetVersion {
			toRollback = append(toRollback, applied[i])
		}
	}

	if len(toRollback) == 0 {
		return 0, fmt.Errorf("no migrations to roll back above version %04d", targetVersion)
	}

	r.log.RollbackStart(len(toRollback))

	if !r.dryRun {
		if err := db.AcquireLock(ctx, r.drv, r.dbName); err != nil {
			return 0, err
		}
		defer func() {
			if err := db.ReleaseLock(ctx, r.drv, r.dbName); err != nil {
				r.log.Error("failed to release migration lock", "error", err)
			}
		}()
	}

	rolledBack := 0
	for _, m := range toRollback {
		if m.DownPath == "" {
			return rolledBack, fmt.Errorf(
				"migration %04d_%s has no .down.cypher file — cannot roll back",
				m.Version, m.Name,
			)
		}

		r.log.MigrationRun(m.Version, m.Name, r.dryRun)
		start := time.Now()

		if err := r.rollbackOne(ctx, m); err != nil {
			r.log.MigrationFail(m.Version, m.Name, err)
			return rolledBack, fmt.Errorf("rollback of %04d_%s failed: %w", m.Version, m.Name, err)
		}

		elapsed := fmt.Sprintf("%dms", time.Since(start).Milliseconds())
		r.log.MigrationDone(m.Version, m.Name, elapsed, r.dryRun)
		rolledBack++
	}

	return rolledBack, nil
}

// rollbackOne executes a single .down.cypher, using per-statement transactions
// for schema DDL just like runOne does. Deletes the history record afterwards.
func (r *Runner) rollbackOne(ctx context.Context, m migration.Migration) error {
	stmts, err := loader.ReadStatements(m.DownPath)
	if err != nil {
		return err
	}

	if r.dryRun {
		r.log.Debug("dry-run rollback: would execute down statements", "count", len(stmts), "migration", m.Name)
		return nil
	}

	if isSchemaMigration(stmts) {
		// Run each DDL statement in its own transaction
		for i, stmt := range stmts {
			r.log.Debug("executing schema rollback statement", "index", i+1, "migration", m.Name)
			sess := r.drv.NewSession(ctx, neo4j.SessionConfig{DatabaseName: r.dbName})
			_, execErr := sess.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
				_, err := tx.Run(ctx, stmt, nil)
				return nil, err
			})
			sess.Close(ctx)
			if execErr != nil {
				return fmt.Errorf("rollback statement %d failed: %w\nCypher: %s", i+1, execErr, truncate(stmt, 200))
			}
		}
		// Delete history record in its own transaction
		sess := r.drv.NewSession(ctx, neo4j.SessionConfig{DatabaseName: r.dbName})
		defer sess.Close(ctx)
		_, err = sess.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			return nil, db.DeleteRecord(ctx, tx, m.Version)
		})
		return err
	}

	// Pure data rollback — all in one transaction
	session := r.drv.NewSession(ctx, neo4j.SessionConfig{DatabaseName: r.dbName})
	defer session.Close(ctx)

	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		for i, stmt := range stmts {
			r.log.Debug("executing rollback statement", "index", i+1, "migration", m.Name)
			if _, err := tx.Run(ctx, stmt, nil); err != nil {
				return nil, fmt.Errorf("rollback statement %d failed: %w\nCypher: %s", i+1, err, truncate(stmt, 200))
			}
		}
		return nil, db.DeleteRecord(ctx, tx, m.Version)
	})
	return err
}

// truncate shortens a string for log display.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
