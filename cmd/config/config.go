package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration for GraphMigrate.
type Config struct {
	// Neo4j connection
	URL      string
	Username string
	Password string
	Database string

	// Migration settings
	MigrationsDir string

	// Runtime flags (set by CLI, not env)
	Verbose bool
	DryRun  bool
}

// Load reads configuration from a .env file and environment variables.
// Environment variables take precedence over .env file values.
func Load() (*Config, error) {
	// Load .env if it exists — don't fail if it doesn't
	_ = godotenv.Load()

	cfg := &Config{
		URL:           getEnv("NEO4J_URL", "bolt://localhost:7687"),
		Username:      getEnv("NEO4J_USERNAME", "neo4j"),
		Password:      getEnv("NEO4J_PASSWORD", ""),
		Database:      getEnv("NEO4J_DATABASE", "movies"),
		MigrationsDir: getEnv("MIGRATIONS_DIR", "./migrations"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validate checks required fields are present.
func (c *Config) validate() error {
	if c.URL == "" {
		return fmt.Errorf("NEO4J_URL is required")
	}
	if c.Username == "" {
		return fmt.Errorf("NEO4J_USERNAME is required")
	}
	if c.Password == "" {
		return fmt.Errorf("NEO4J_PASSWORD is required — set it in .env or environment")
	}
	return nil
}

// getEnv returns the env value or a fallback default.
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
