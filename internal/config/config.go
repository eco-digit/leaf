// Package config handles loading and validating runtime configuration.
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Addr string `yaml:"addr"`
	} `yaml:"server"`

	Prometheus struct {
		URL      string `yaml:"url"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"prometheus"`

	Infrastructure struct {
		InfraPath   string `yaml:"infra_path"`
		ProfilePath string `yaml:"profile_path"`
	} `yaml:"infrastructure"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	if cfg.Prometheus.URL == "" {
		return nil, fmt.Errorf("prometheus.url must be set in %s", path)
	}
	if cfg.Infrastructure.InfraPath == "" {
		return nil, fmt.Errorf("infrastructure.infra_path must be set in %s", path)
	}
	if cfg.Infrastructure.ProfilePath == "" {
		return nil, fmt.Errorf("infrastructure.profile_path must be set in %s", path)
	}
	return &cfg, nil
}
