package main

import (
	"es-index-pruner/pkg/config"
	"es-index-pruner/pkg/elasticsearch"
	"es-index-pruner/pkg/utils"
	"flag"
	"time"
)

func main() {
	logger := utils.SetupLogger()

	dryRun := flag.Bool("dry-run", false, "Enable dry-run mode")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig("./config.yaml")
	if err != nil {
		logger.Panicf("error loading configuration: %v", err)
	}

	err = cfg.Validate()
	if err != nil {
		logger.Panicf("error validating configuration: %v", err)
	}

	// Create a new Elasticsearch client
	esClient, err := elasticsearch.NewClient(cfg, logger)
	if err != nil {
		logger.Panicf("error creating ES client: %v", err)
	}
	esClient.DryRun = *dryRun

	// Run every 120 secs
	esClient.StartMonitoring(1 * time.Second)

}
