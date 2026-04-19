package loader

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	migration "github.com/shriya0_4/graphmigrate/migrations"
)

// Load scans the given directory for migration files, parses their names into
// Migration structs, pairs them with their .down counterparts, sorts by version,
// and validates there are no duplicate version numbers.
//
// Expected filename format: 0001_name_here.cypher
// Down file format:         0001_name_here.down.cypher
func Load(dir string) ([]migration.Migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("cannot read migrations directory %q: %w", dir, err)
	}

	// First pass: collect all .down files so we can pair them
	downFiles := make(map[int]string) // version → full path
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".down.cypher") {
			version, err := parseVersion(name)
			if err != nil {
				continue // skip unparseable names
			}
			downFiles[version] = filepath.Join(dir, name)
		}
	}

	// Second pass: collect all .up files (ending in .cypher but NOT .down.cypher)
	seen := make(map[int]bool)
	var migrations []migration.Migration

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()

		// skip down files and non-cypher files
		if strings.HasSuffix(name, ".down.cypher") || !strings.HasSuffix(name, ".cypher") {
			continue
		}

		version, err := parseVersion(name)
		if err != nil {
			return nil, fmt.Errorf("invalid migration filename %q: %w", name, err)
		}

		slug, err := parseSlug(name)
		if err != nil {
			return nil, fmt.Errorf("invalid migration filename %q: %w", name, err)
		}

		if seen[version] {
			return nil, fmt.Errorf("duplicate migration version %04d — check your migrations directory", version)
		}
		seen[version] = true

		m := migration.Migration{
			Version:  version,
			Name:     slug,
			Filepath: filepath.Join(dir, name),
			DownPath: downFiles[version], // empty string if no .down file
		}
		migrations = append(migrations, m)
	}

	// Sort ascending by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// Checksum computes the MD5 hash of a migration file's contents.
// Used to detect if a migration file was modified after being applied.
func Checksum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("cannot open file for checksum %q: %w", path, err)
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("failed to hash file %q: %w", path, err)
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// ReadStatements reads a .cypher file and splits it into individual Cypher
// statements on semicolons. Empty statements are discarded.
func ReadStatements(path string) ([]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read migration file %q: %w", path, err)
	}

	// strip comment lines (lines starting with //)
	lines := strings.Split(string(raw), "\n")
	var filtered []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") || trimmed == "" {
			continue
		}
		filtered = append(filtered, line)
	}

	// split on semicolons to get individual statements
	parts := strings.Split(strings.Join(filtered, "\n"), ";")
	var stmts []string
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s != "" {
			stmts = append(stmts, s)
		}
	}

	if len(stmts) == 0 {
		return nil, fmt.Errorf("migration file %q contains no executable Cypher statements", path)
	}

	return stmts, nil
}

// parseVersion extracts the leading version number from a filename.
// e.g. "0004_rename_released.cypher" → 4
func parseVersion(filename string) (int, error) {
	parts := strings.SplitN(filename, "_", 2)
	if len(parts) < 2 {
		return 0, fmt.Errorf("expected format NNNN_name.cypher, got %q", filename)
	}
	v, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("version prefix %q is not a number in %q", parts[0], filename)
	}
	return v, nil
}

// parseSlug extracts the human-readable name from a filename.
// e.g. "0004_rename_released.cypher" → "rename_released"
// e.g. "0004_rename_released.down.cypher" → "rename_released"
func parseSlug(filename string) (string, error) {
	// strip extension(s)
	name := strings.TrimSuffix(filename, ".down.cypher")
	name = strings.TrimSuffix(name, ".cypher")

	parts := strings.SplitN(name, "_", 2)
	if len(parts) < 2 || parts[1] == "" {
		return "", fmt.Errorf("no slug found in filename %q", filename)
	}
	return parts[1], nil
}
