package driver

import (
	"context"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// struct to hold the neo4j config data
type Config struct {
	url      string
	username string
	password string
}

// load the config details from env
func LoadConfig() Config {

	err := godotenv.Load()
	if err != nil {
		fmt.Errorf("error loading env")
	}
	cfg := Config{
		url:      os.Getenv("NEO4J_URL"),
		username: os.Getenv("NEO4J_USERNAME"),
		password: os.Getenv("NEO4J_PASSWORD"),
	}
	return cfg
}

// initialise a neo4j driver
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
