package driver

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/shriya0_4/graphmigrate/cmd/config"
)

// Connect creates and verifies a Neo4j driver using the given config.
func Connect(ctx context.Context, cfg *config.Config) (neo4j.DriverWithContext, error) {
	drv, err := neo4j.NewDriverWithContext(
		cfg.URL,
		neo4j.BasicAuth(cfg.Username, cfg.Password, ""),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create neo4j driver: %w", err)
	}

	if err := drv.VerifyConnectivity(ctx); err != nil {
		return nil, fmt.Errorf("neo4j connectivity check failed (is Neo4j running at %s?): %w", cfg.URL, err)
	}

	return drv, nil
}
