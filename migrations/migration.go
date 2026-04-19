package migration

import "time"

// Migration represents a single versioned migration file pair.
// Each migration has an .up (forward) and optional .down (rollback) Cypher file.
type Migration struct {
	Version  int    // numeric prefix, e.g. 1 from "0001_add_genre_nodes.cypher"
	Name     string // slug, e.g. "add_genre_nodes"
	Filepath string // absolute path to the .up.cypher file
	DownPath string // absolute path to .down.cypher (empty if none exists)
}

// AppliedMigration represents a migration record stored in Neo4j.
type AppliedMigration struct {
	Version   int
	Name      string
	AppliedAt time.Time
	Checksum  string // MD5 of the .cypher file at time of application
}

// Status describes the current state of a migration.
type Status struct {
	Migration
	Applied   bool
	AppliedAt string // formatted time string, empty if not applied
	Checksum  string // stored checksum (empty if not applied)
}
