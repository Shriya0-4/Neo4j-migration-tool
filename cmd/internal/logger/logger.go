package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"
)

// ANSI color codes for terminal output
const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
	colorBold   = "\033[1m"
)

// Level constants mirroring slog levels for external use
const (
	LevelDebug = slog.LevelDebug
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
)

// prettyHandler is a custom slog.Handler that writes human-readable,
// colorized log lines to the given writer.
type prettyHandler struct {
	w       io.Writer
	level   slog.Level
	verbose bool
}

func (h *prettyHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *prettyHandler) Handle(_ context.Context, r slog.Record) error {
	ts := time.Now().Format("15:04:05")

	var levelStr string
	switch r.Level {
	case slog.LevelDebug:
		levelStr = colorGray + "DEBUG" + colorReset
	case slog.LevelInfo:
		levelStr = colorCyan + "INFO " + colorReset
	case slog.LevelWarn:
		levelStr = colorYellow + "WARN " + colorReset
	case slog.LevelError:
		levelStr = colorRed + "ERROR" + colorReset
	default:
		levelStr = "     "
	}

	line := fmt.Sprintf("%s%s%s  %s  %s\n",
		colorGray, ts, colorReset,
		levelStr,
		r.Message,
	)

	// append structured attrs if verbose
	if h.verbose {
		r.Attrs(func(a slog.Attr) bool {
			line += fmt.Sprintf("         %s%s%s=%v\n", colorGray, a.Key, colorReset, a.Value)
			return true
		})
	}

	_, err := fmt.Fprint(h.w, line)
	return err
}

func (h *prettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *prettyHandler) WithGroup(name string) slog.Handler {
	return h
}

// Logger wraps slog.Logger with convenience methods and migration-specific helpers.
type Logger struct {
	*slog.Logger
	verbose bool
}

// New creates a new Logger. verbose=true prints DEBUG logs and structured attrs.
func New(verbose bool) *Logger {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	handler := &prettyHandler{
		w:       os.Stdout,
		level:   level,
		verbose: verbose,
	}

	return &Logger{
		Logger:  slog.New(handler),
		verbose: verbose,
	}
}

// MigrationStart prints a banner-style header for a migration run.
func (l *Logger) MigrationStart(total, pending int) {
	fmt.Printf("\n%s%s GraphMigrate%s\n", colorBold, colorCyan, colorReset)
	fmt.Printf("%s────────────────────────────────────────%s\n", colorGray, colorReset)
	fmt.Printf("  Total migrations : %s%d%s\n", colorBold, total, colorReset)
	fmt.Printf("  Applied          : %s%d%s\n", colorGreen, total-pending, colorReset)
	fmt.Printf("  Pending          : %s%d%s\n", colorYellow, pending, colorReset)
	fmt.Printf("%s────────────────────────────────────────%s\n\n", colorGray, colorReset)
}

// MigrationRun prints a running migration line.
func (l *Logger) MigrationRun(version int, name string, dryRun bool) {
	prefix := colorGreen + "[RUN] " + colorReset
	if dryRun {
		prefix = colorYellow + "[DRY] " + colorReset
	}
	fmt.Printf("  %s %s%04d%s  %s\n", prefix, colorBold, version, colorReset, name)
}

// MigrationDone prints the completion checkmark + duration.
func (l *Logger) MigrationDone(version int, name string, elapsed string, dryRun bool) {
	marker := colorGreen + "✓" + colorReset
	if dryRun {
		marker = colorYellow + "~" + colorReset
	}
	fmt.Printf("  %s  %s%04d%s  %-45s %s%s%s\n",
		marker, colorBold, version, colorReset, name, colorGray, elapsed, colorReset)
}

// MigrationFail prints a failure line.
func (l *Logger) MigrationFail(version int, name string, err error) {
	fmt.Printf("  %s✗%s  %s%04d%s  %-45s %s%v%s\n",
		colorRed, colorReset, colorBold, version, colorReset, name, colorRed, err, colorReset)
}

// RollbackStart prints header for a rollback run.
func (l *Logger) RollbackStart(count int) {
	fmt.Printf("\n%s%s GraphMigrate — Rollback%s\n", colorBold, colorYellow, colorReset)
	fmt.Printf("%s────────────────────────────────────────%s\n", colorGray, colorReset)
	fmt.Printf("  Rolling back %s%d%s migration(s)\n\n", colorYellow, count, colorReset)
}

// Summary prints the final summary line.
func (l *Logger) Summary(applied int, dryRun bool) {
	fmt.Printf("\n%s────────────────────────────────────────%s\n", colorGray, colorReset)
	if dryRun {
		fmt.Printf("  %sDRY RUN complete — no changes made to the database.%s\n\n", colorYellow, colorReset)
		return
	}
	if applied == 0 {
		fmt.Printf("  %sDatabase is up to date. Nothing to run.%s\n\n", colorGreen, colorReset)
		return
	}
	fmt.Printf("  %s%d migration(s) applied successfully.%s\n\n", colorGreen, applied, colorReset)
}

// StatusHeader prints the header row for the status table.
func (l *Logger) StatusHeader() {
	fmt.Printf("\n%s%s GraphMigrate — Status%s\n", colorBold, colorCyan, colorReset)
	fmt.Printf("%s────────────────────────────────────────────────────────────────────%s\n", colorGray, colorReset)
	fmt.Printf("  %s%-8s  %-42s  %-10s  %s%s\n",
		colorBold, "VERSION", "NAME", "STATUS", "APPLIED AT", colorReset)
	fmt.Printf("%s────────────────────────────────────────────────────────────────────%s\n", colorGray, colorReset)
}

// StatusRow prints one row of the status table.
func (l *Logger) StatusRow(version int, name, status, appliedAt string) {
	var statusColored string
	switch status {
	case "applied":
		statusColored = colorGreen + "✓ applied " + colorReset
	case "pending":
		statusColored = colorYellow + "⧖ pending " + colorReset
	default:
		statusColored = status
	}
	fmt.Printf("  %-8d  %-42s  %s  %s\n", version, name, statusColored, appliedAt)
}

// Warn is a convenience wrapper for a bright yellow warning.
func (l *Logger) Warn(msg string, args ...any) {
	l.Logger.Warn(fmt.Sprintf(msg, args...))
}

// ChecksumMismatch prints a prominent checksum warning.
func (l *Logger) ChecksumMismatch(version int, name string) {
	fmt.Printf("\n  %s⚠ CHECKSUM MISMATCH%s  %04d_%s\n", colorYellow, colorReset, version, name)
	fmt.Printf("    This migration file was modified after being applied.\n")
	fmt.Printf("    This may indicate accidental edits. Proceed with caution.\n\n")
}

// LockWarning prints a migration-lock warning.
func (l *Logger) LockWarning() {
	fmt.Printf("\n  %s⚠ MIGRATION LOCK DETECTED%s\n", colorRed, colorReset)
	fmt.Printf("    Another migration is already in progress.\n")
	fmt.Printf("    If this is stale, run: %sgraphmigrate unlock%s\n\n", colorBold, colorReset)
}
