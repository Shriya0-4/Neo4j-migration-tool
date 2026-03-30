package main

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/shriya0_4/graphmigrate/cmd/cypher"
	"github.com/shriya0_4/graphmigrate/cmd/driver"
)

func main() {
	//create context(manage lifecycle,timeouts across db connection)
	ctx := context.Background()
	//load config data
	cfg := driver.LoadConfig()
	fmt.Println("config:", cfg)
	// initilaise a neo4j driver
	drv, err := driver.Connect(ctx, cfg)

	if err != nil {
		fmt.Errorf("Failed to connect to DB:%w", err)
	}
	//cleanup for closing the driver when application exists
	defer drv.Close(ctx)
	// a new session to run the queries
	session := drv.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: "movies",
	})
	//run ad cleanup after session closes
	defer session.Close(ctx)

	// get the query to be executed form query layer
	query := cypher.Testquery()
	//execute query in a read transaction
	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		//run the query
		fmt.Println("Query:", query)
		res, err := tx.Run(ctx, query, nil)
		if err != nil {
			return nil, err
		}
		// store results in a map
		var data []map[string]any
		//iterate and store the results
		for res.Next(ctx) {
			data = append(data, res.Record().AsMap())
		}
		return data, res.Err()
	})
	if err != nil {
		fmt.Errorf("error:%w", err)
	}
	fmt.Println("result:", result)
}
