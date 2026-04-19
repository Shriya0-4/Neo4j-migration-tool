package db

import (
	"context"
	"fmt"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	migration "github.com/shriya0_4/graphmigrate/migrations"
)

const (
	// labelMigration is the Neo4j node label for applied migration records.
	labelMigration = "SchemaMigration"
	// labelLock is the Neo4j node label used to prevent concurrent runs.
	labelLock = "MigrationLock"
)

// EnsureConstraints creates the uniqueness constraint on SchemaMigration nodes.
// Safe to call on every startup — uses IF NOT EXISTS.
func EnsureConstraints(ctx context.Context, drv neo4j.DriverWithContext, dbName string) error {
	session := drv.NewSession(ctx, neo4j.SessionConfig{DatabaseName: dbName})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx,
			`CREATE CONSTRAINT schema_migration_version_unique IF NOT EXISTS
             FOR (m:SchemaMigration) REQUIRE m.version IS UNIQUE`,
			nil,
		)
		return nil, err
	})
	if err != nil {
		return fmt.Errorf("failed to ensure migration constraints: %w", err)
	}
	return nil
}

// GetApplied returns all applied migrations stored in Neo4j, keyed by version.
func GetApplied(ctx context.Context, drv neo4j.DriverWithContext, dbName string) (map[int]migration.AppliedMigration, error) {
	session := drv.NewSession(ctx, neo4j.SessionConfig{DatabaseName: dbName})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx,
			`MATCH (m:SchemaMigration)
             RETURN m.version AS version, m.name AS name,
                    m.appliedAt AS appliedAt, m.checksum AS checksum
             ORDER BY m.version ASC`,
			nil,
		)
		if err != nil {
			return nil, err
		}

		applied := make(map[int]migration.AppliedMigration)
		for res.Next(ctx) {
			rec := res.Record()
			version, _ := rec.Get("version")
			name, _ := rec.Get("name")
			appliedAt, _ := rec.Get("appliedAt")
			checksum, _ := rec.Get("checksum")

			v := int(version.(int64))

			var t time.Time
			if appliedAt != nil {
				if neo4jTime, ok := appliedAt.(neo4j.LocalDateTime); ok {
					t = neo4jTime.Time()
				}
			}

			applied[v] = migration.AppliedMigration{
				Version:   v,
				Name:      name.(string),
				AppliedAt: t,
				Checksum:  checksumStr(checksum),
			}
		}
		return applied, res.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch applied migrations: %w", err)
	}

	return result.(map[int]migration.AppliedMigration), nil
}

// RecordApplied writes a SchemaMigration node after a successful migration run.
func RecordApplied(ctx context.Context, tx neo4j.ManagedTransaction, m migration.Migration, checksum string) error {
	_, err := tx.Run(ctx,
		`CREATE (m:SchemaMigration {
            version:   $version,
            name:      $name,
            appliedAt: localdatetime(),
            checksum:  $checksum
        })`,
		map[string]any{
			"version":  m.Version,
			"name":     m.Name,
			"checksum": checksum,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to record migration %04d: %w", m.Version, err)
	}
	return nil
}

// DeleteRecord removes the SchemaMigration node for a given version (used in rollback).
func DeleteRecord(ctx context.Context, tx neo4j.ManagedTransaction, version int) error {
	_, err := tx.Run(ctx,
		`MATCH (m:SchemaMigration {version: $version}) DELETE m`,
		map[string]any{"version": version},
	)
	if err != nil {
		return fmt.Errorf("failed to delete migration record for version %04d: %w", version, err)
	}
	return nil
}

// AcquireLock attempts to create the MigrationLock node.
// Returns an error if the lock already exists (another run is in progress).
func AcquireLock(ctx context.Context, drv neo4j.DriverWithContext, dbName string) error {
	session := drv.NewSession(ctx, neo4j.SessionConfig{DatabaseName: dbName})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// Check if lock already exists
		res, err := tx.Run(ctx,
			`MATCH (l:MigrationLock) RETURN l.lockedAt AS lockedAt LIMIT 1`,
			nil,
		)
		if err != nil {
			return nil, err
		}
		if res.Next(ctx) {
			lockedAt, _ := res.Record().Get("lockedAt")
			return nil, fmt.Errorf("migration lock already held (locked at: %v)", lockedAt)
		}

		// Acquire the lock
		_, err = tx.Run(ctx,
			`CREATE (:MigrationLock {lockedAt: localdatetime()})`,
			nil,
		)
		return nil, err
	})

	if err != nil {
		return fmt.Errorf("cannot acquire migration lock: %w", err)
	}
	return nil
}

// ReleaseLock removes the MigrationLock node.
func ReleaseLock(ctx context.Context, drv neo4j.DriverWithContext, dbName string) error {
	session := drv.NewSession(ctx, neo4j.SessionConfig{DatabaseName: dbName})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx, `MATCH (l:MigrationLock) DELETE l`, nil)
		return nil, err
	})
	if err != nil {
		return fmt.Errorf("failed to release migration lock: %w", err)
	}
	return nil
}

// checksumStr safely converts a Neo4j value to string checksum.
func checksumStr(v any) string {
	if v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}
