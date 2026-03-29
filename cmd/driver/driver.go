package driver

import (
	"context"
	"fmt"
	"os"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type Config struct {
	url      string
	username string
	password string
}

func LoadConfig() Config {
	return Config{url: os.Getenv("NEO4J_url"),
		username: os.Getenv("NEO4J_username"),
		password: os.Getenv("NEO4J_password")}
}

func Connect(ctx context.Context, cfg Config) (neo4j.DriverWithContext, error) {
	driver, err := neo4j.NewDriverWithContext(
		cfg.url,
		neo4j.BasicAuth(cfg.username, cfg.password, ""),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create a driver:%w", err)
	}
	err = driver.VerifyConnectivity(ctx)
	if err != nil {
		return nil, fmt.Errorf("driver failed to connect with neo4j:%w", err)
	}
	fmt.Println("connected to neo4j!")
	return driver, nil
}
