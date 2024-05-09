package elasticsearch

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"es-index-pruner/pkg/config"
	"es-index-pruner/pkg/utils"
	"fmt"
	"github.com/elastic/go-elasticsearch/v8"
	"go.uber.org/zap"
	"net/http"
	"sort"
	"time"
)

type ESClient struct {
	Client *elasticsearch.Client
	DryRun bool
	Alias  ESAlias

	config config.Config
	logger *zap.SugaredLogger
}

type ESAlias struct {
	Name      string
	Indices   []ESIndex
	TotalSize int64
}

type ESIndex struct {
	Name string
	Size int64
}

func NewClient(cfg *config.Config, logger *zap.SugaredLogger) (*ESClient, error) {
	esCfg := elasticsearch.Config{
		Addresses: []string{cfg.Cluster.URL},
		Username:  cfg.Cluster.Username,
		Password:  cfg.Cluster.Password,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // This disables SSL certificate verification
			},
			MaxIdleConnsPerHost:   10,
			ResponseHeaderTimeout: time.Second * 10,
		},
	}
	es, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		return nil, err
	}
	return &ESClient{Client: es, config: *cfg, logger: logger}, nil
}

func (c *ESClient) StartMonitoring(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			for _, aliasConfig := range c.config.Aliases {
				err := c.ProcessAlias(aliasConfig)
				if err != nil {
					c.logger.With("alias", aliasConfig.Name).Errorf("Error processing alias: %v", err)
				}
			}
		}
	}
}

func (c *ESClient) GetAlias(ctx context.Context, aliasName string) (*ESAlias, error) {
	// Get all indices for the alias
	res, err := c.Client.Indices.GetAlias(
		c.Client.Indices.GetAlias.WithContext(ctx),
		c.Client.Indices.GetAlias.WithName(aliasName),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get indices for alias %s: %v", aliasName, err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("error response when getting indices for alias %s: %s", c.Alias.Name, res.String())
	}

	// Parse the response to get a list of indices
	var r map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	alias := ESAlias{
		Name:    aliasName,
		Indices: make([]ESIndex, 0, len(r)),
	}

	for index := range r {
		alias.Indices = append(alias.Indices, ESIndex{Name: index})
	}

	// Calculate total size of indices
	err = c.populateSizeDetails(ctx, &alias)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate total size for indices under alias %s: %v", aliasName, err)
	}

	return &alias, nil
}

func (c *ESClient) ProcessAlias(alias config.Alias) error {
	ctx := context.Background()

	aliasInfo, err := c.GetAlias(ctx, alias.Name)
	if err != nil {
		return fmt.Errorf("failed to get alias %s: %v", alias.Name, err)
	}

	// Parse max size from config
	maxSizeBytes, err := utils.ParseSize(alias.MaxSize)
	if err != nil {
		return fmt.Errorf("invalid max size format: %v", err)
	}

	// If total size exceeds max size, prune indices
	if aliasInfo.TotalSize > maxSizeBytes {
		c.logger.With(
			"alias", aliasInfo.Name,
			"total_size", aliasInfo.TotalSize,
			"max_size", maxSizeBytes,
		).Warnf("Total size exceeds max size for alias")
		if err := c.pruneIndicesByMaxSize(ctx, aliasInfo.Indices, aliasInfo.TotalSize, maxSizeBytes); err != nil {
			return fmt.Errorf("failed to prune indices: %v", err)
		}
	} else {
		c.logger.With(
			"alias", aliasInfo.Name,
			"total_size", aliasInfo.TotalSize,
			"max_size", maxSizeBytes,
		).Infof("Total size is within max size for alias")
	}

	return nil
}

func (c *ESClient) populateSizeDetails(ctx context.Context, alias *ESAlias) error {
	var totalSize int64
	for i := range alias.Indices {
		index := &alias.Indices[i] // Work directly with the reference to modify the slice element

		res, err := c.Client.Indices.Stats(
			c.Client.Indices.Stats.WithContext(ctx),
			c.Client.Indices.Stats.WithIndex(index.Name),
			c.Client.Indices.Stats.WithMetric("store"),
		)
		if err != nil {
			return fmt.Errorf("error getting stats for index %s: %v", index, err)
		}
		defer res.Body.Close()

		if res.IsError() {
			return fmt.Errorf("error response when getting stats for index %s: %s", index, res.String())
		}

		var r map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
			return fmt.Errorf("error parsing stats response: %v", err)
		}

		// Navigate to the indices data, then to the specific index
		if indicesData, found := r["indices"]; found {
			if indexData, found := indicesData.(map[string]interface{})[index.Name]; found {
				if totalData, found := indexData.(map[string]interface{})["total"]; found {
					if storeData, found := totalData.(map[string]interface{})["store"]; found {
						if size, found := storeData.(map[string]interface{})["size_in_bytes"]; found {
							intSize := int64(size.(float64))
							// Update the index size
							index.Size = intSize
							totalSize += intSize
							continue
						}
					}
				}
			}
		}
		fmt.Printf("Data for index %s not in expected format or missing key parts\n", index)
	}
	alias.TotalSize = totalSize
	return nil
}

func (c *ESClient) pruneIndicesByMaxSize(ctx context.Context, indices []ESIndex, currentSize, maxSize int64) error {
	sort.Slice(indices, func(i, j int) bool {
		return indices[i].Name < indices[j].Name
	})

	for _, index := range indices {
		if currentSize <= maxSize {
			break
		}

		if c.DryRun {
			fmt.Printf("Would delete index %s\n", index)
			currentSize -= index.Size
			continue
		} else {
			res, err := c.Client.Indices.Delete([]string{index.Name},
				c.Client.Indices.Delete.WithContext(ctx),
			)
			if err != nil {
				return fmt.Errorf("error deleting index %s: %v", index, err)
			}
			if res.IsError() {
				return fmt.Errorf("error response when deleting index %s: %s", index, res.String())
			}

			fmt.Printf("Deleted index %s\n", index)
			// Assume deletion frees the size of the index (adjust as needed based on actual index size)
			currentSize -= index.Size
		}

		if currentSize <= maxSize {
			break
		}
		fmt.Printf("Total size after deleting index %s: %d\n", index, currentSize)
	}

	return nil
}
