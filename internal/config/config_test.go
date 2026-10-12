package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCustomConfigOverlaysDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	data := []byte("search:\n  max_queries: 7\ncustom_queries:\n  - type: code\n    query: filename:custom.conf\n    reason: test\n")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Search.MaxQueries != 7 || cfg.Search.MaxResults != 100 || len(cfg.CustomQueries) != 1 {
		t.Fatalf("defaults were not overlaid correctly: %#v", cfg)
	}
}
