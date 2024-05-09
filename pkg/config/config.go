package config

import (
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

// Config structure that holds all configuration
type Config struct {
	Cluster struct {
		URL      string `yaml:"url"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"cluster"`
	Aliases []Alias `yaml:"aliases"`
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

	return &cfg, nil
}
