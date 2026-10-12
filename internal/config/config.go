package config

import (
	_ "embed"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

//go:embed default.yaml
var defaultYAML []byte

type Config struct {
	GitHub struct {
		APIURL     string `yaml:"api_url"`
		APIVersion string `yaml:"api_version"`
	} `yaml:"github"`
	Search struct {
		Profile         string        `yaml:"profile"`
		MaxDepth        int           `yaml:"max_depth"`
		MaxQueries      int           `yaml:"max_queries"`
		MaxResults      int           `yaml:"max_results"`
		Concurrency     int           `yaml:"concurrency"`
		Timeout         time.Duration `yaml:"-"`
		TimeoutText     string        `yaml:"timeout"`
		IncludeForks    bool          `yaml:"include_forks"`
		IncludeArchived bool          `yaml:"include_archived"`
	} `yaml:"search"`
	Extractors struct {
		Domains   bool `yaml:"domains"`
		URLs      bool `yaml:"urls"`
		Endpoints bool `yaml:"endpoints"`
		Cloud     bool `yaml:"cloud"`
		Email     bool `yaml:"email"`
	} `yaml:"extractors"`
	InterestingFiles     []string `yaml:"interesting_files"`
	SecretCandidateFiles []string `yaml:"secret_candidate_files"`
	CustomQueries        []struct {
		Type   string `yaml:"type"`
		Query  string `yaml:"query"`
		Reason string `yaml:"reason"`
	} `yaml:"custom_queries"`
}

func Load(path string) (Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(defaultYAML, &cfg); err != nil {
		return cfg, fmt.Errorf("parse embedded defaults: %w", err)
	}
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return cfg, fmt.Errorf("read config: %w", err)
		}
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return cfg, fmt.Errorf("parse config: %w", err)
		}
	}
	var err error
	cfg.Search.Timeout, err = time.ParseDuration(cfg.Search.TimeoutText)
	if err != nil {
		return cfg, fmt.Errorf("invalid search.timeout: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func (cfg Config) Validate() error {
	if cfg.Search.MaxQueries < 1 || cfg.Search.MaxResults < 1 || cfg.Search.Concurrency < 1 {
		return fmt.Errorf("max_queries, max_results, and concurrency must be positive")
	}
	if cfg.Search.MaxDepth < 0 || cfg.Search.Timeout <= 0 {
		return fmt.Errorf("max_depth must be non-negative and timeout must be positive")
	}
	switch cfg.Search.Profile {
	case "quick", "standard", "deep":
	default:
		return fmt.Errorf("unknown profile %q", cfg.Search.Profile)
	}
	return nil
}
