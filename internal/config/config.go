// Package config handles loading and validating Leaf's configuration files.
package config

import (
	"gopkg.in/yaml.v3"
	"log"
	"os"
)

type Config struct {
	Prometheus struct {
		URL      string `yaml:"url"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"prometheus"`
	Metrics []string `yaml:"metrics"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.Prometheus.URL == "" {
		log.Fatal("Prometheus URL must be defined in config.yaml")
	}
	return &cfg, nil
}
