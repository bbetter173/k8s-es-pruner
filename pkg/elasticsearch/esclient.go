package elasticsearch

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"es-index-pruner/pkg/config"
	"es-index-pruner/pkg/utils"
	"fmt"
	"go.uber.org/zap"
	"io/ioutil"
	"k8s.io/klog/v2"
	"net/http"
	"sort"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
)

type ESClient struct {
	Client *elasticsearch.Client
	DryRun bool
	config config.Config
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
	// Setup the TLS configuration
	if cfg.Cluster.SkipVerify {
		logger.Warn("TLS verification is disabled")
	}
	tlsConfig := &tls.Config{
		InsecureSkipVerify: cfg.Cluster.SkipVerify,
	}

	if cfg.Cluster.CACertPath != "" {
		caCert, err := ioutil.ReadFile(cfg.Cluster.CACertPath)
		if err != nil {
			return nil, fmt.Errorf("reading CA certificate: %v", err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		tlsConfig.RootCAs = caCertPool
		tlsConfig.InsecureSkipVerify = false
	}

	httpTransport := &http.Transport{
		TLSClientConfig:       tlsConfig,
		MaxIdleConnsPerHost:   10,
		ResponseHeaderTimeout: time.Second * 10,
	}

	esCfg := elasticsearch.Config{
		Addresses: []string{cfg.Cluster.URL},
		Username:  cfg.Cluster.Username,
		Password:  cfg.Cluster.Password,
		Transport: httpTransport,
	}

	client, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		return nil, fmt.Errorf("creating ES client: %v", err)
	}

	return &ESClient{Client: client, config: *cfg}, nil
}

// StartMonitoring simulates monitoring with detailed logs
func (c *ESClient) StartMonitoring(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		c.PruneConfiguredAliases(ctx)
	}
}

func (c *ESClient) PruneConfiguredAliases(ctx context.Context) {
	for _, aliasConfig := range c.config.Aliases {
		if err := c.ProcessAlias(ctx, aliasConfig); err != nil {
			klog.Errorf("Error processing alias %s: %v", aliasConfig.Name, err)
		}
	}
}

// GetAlias fetches alias details from Elasticsearch and computes sizes
func (c *ESClient) GetAlias(ctx context.Context, aliasName string) (*ESAlias, error) {
	res, err := c.Client.Indices.GetAlias(
		c.Client.Indices.GetAlias.WithContext(ctx),
		c.Client.Indices.GetAlias.WithName(aliasName),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get indices for alias %s: %v", aliasName, err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("error response when getting indices for alias %s: %s", aliasName, res.String())
	}

	var r map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	alias := ESAlias{Name: aliasName, Indices: make([]ESIndex, 0, len(r))}
	for index := range r {
		alias.Indices = append(alias.Indices, ESIndex{Name: index})
	}

	if err = c.populateSizeDetails(ctx, &alias); err != nil {
		return nil, err
	}

	return &alias, nil
}

// ProcessAlias handles alias processing based on max size configuration
func (c *ESClient) ProcessAlias(ctx context.Context, alias config.Alias) error {
	aliasInfo, err := c.GetAlias(ctx, alias.Name)
	if err != nil {
		return fmt.Errorf("failed to get alias %s: %v", alias.Name, err)
	}

	maxSizeBytes, err := utils.ParseSize(alias.MaxSize)
	if err != nil {
		return fmt.Errorf("invalid max size format: %v", err)
	}

	if aliasInfo.TotalSize > maxSizeBytes {
		klog.Warningf("Total size exceeds max size for alias - alias: %s, total_size: %d, max_size: %d",
			aliasInfo.Name, aliasInfo.TotalSize, maxSizeBytes)
		if err := c.pruneIndicesByMaxSize(ctx, aliasInfo.Indices, aliasInfo.TotalSize, maxSizeBytes); err != nil {
			return fmt.Errorf("failed to prune indices: %v", err)
		}
	} else {
		/*klog.Infof("Total size is within max size for alias - alias: %s, total_size: %d, max_size: %d",
		aliasInfo.Name, aliasInfo.TotalSize, maxSizeBytes)*/
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
		klog.Errorf("Data for index %s not in expected format or missing key parts", index)
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
			klog.Infof("Would delete index %s", index)
			currentSize -= index.Size
			continue
		} else {
			res, err := c.Client.Indices.Delete([]string{index.Name},
				c.Client.Indices.Delete.WithContext(ctx),
			)
			if err != nil {
				return fmt.Errorf("error deleting index %s: %v", index.Name, err)
			}
			if res.IsError() {
				return fmt.Errorf("error response when deleting index %s: %s", index.Name, res.String())
			}
			klog.Infof("Deleted index %s", index.Name)
			// Assume deletion frees the size of the index (adjust as needed based on actual index size)
			currentSize -= index.Size
		}

		if currentSize <= maxSize {
			break
		}
		klog.Infof("Total size after deleting index %s: %d", index.Name, currentSize)
	}

	return nil
}
