package config

import (
	"es-index-pruner/pkg/utils"
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v3"
)

// Config structure that holds all configuration
type Config struct {
	Cluster struct {
		URL        string `yaml:"url"`
		Username   string `yaml:"username"`
		Password   string `yaml:"password"`
		CACertPath string `yaml:"ca_cert_path"`
		SkipVerify bool   `yaml:"skip_tls_verify"`
	} `yaml:"cluster"`
	Aliases      []Alias `yaml:"aliases"`
	PollInterval int     `yaml:"poll_interval"`
}

// Alias structure to hold alias-specific configurations
type Alias struct {
	Name    string `yaml:"name"`
	MaxSize string `yaml:"max_size"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	// Override config with environment variables if present
	overrideWithEnvVars(&cfg)

	return &cfg, nil
}

func overrideWithEnvVars(cfg *Config) {
	if url := os.Getenv("ES_CLUSTER_URL"); url != "" {
		cfg.Cluster.URL = url
	}
	if username := os.Getenv("ES_USERNAME"); username != "" {
		cfg.Cluster.Username = username
	}
	if password := os.Getenv("ES_PASSWORD"); password != "" {
		cfg.Cluster.Password = password
	}
	if caCertPath := os.Getenv("ES_CA_CERT_PATH"); caCertPath != "" {
		cfg.Cluster.CACertPath = caCertPath
	}
	if skipVerify := os.Getenv("ES_SKIP_TLS_VERIFY"); skipVerify != "" {
		cfg.Cluster.SkipVerify = skipVerify == "true"
	}
}

func (c *Config) Validate() error {
	if c.Cluster.URL == "" {
		return fmt.Errorf("missing cluster URL")
	}
	if c.Cluster.Username == "" || c.Cluster.Password == "" {
		return fmt.Errorf("missing cluster credentials")
	}
	if len(c.Aliases) == 0 {
		return fmt.Errorf("no aliases defined")
	}
	for _, alias := range c.Aliases {
		_, err := utils.ParseSize(alias.MaxSize)
		if err != nil {
			return fmt.Errorf("error parsing size for alias '%s': %v", alias.Name, err)
		}
	}
	return nil
}
