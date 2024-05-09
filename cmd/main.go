package main

import (
	"es-index-pruner/pkg/config"
	"es-index-pruner/pkg/elasticsearch"
	"flag"
	"fmt"
	"time"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "Enable dry-run mode")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig("./config.yaml")
	if err != nil {
		panic(fmt.Errorf("error loading configuration: %w", err))
	}

	// Create a new Elasticsearch client
	esClient, err := elasticsearch.NewClient(cfg)
	if err != nil {
		panic(fmt.Errorf("error creating ES client: %w", err))
	}
	esClient.DryRun = *dryRun

	// Run every 120 secs
	esClient.StartMonitoring(120 * time.Second)

}
