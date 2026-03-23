package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseRuntimeConfigCLIOverridesFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("listen_addr: '127.0.0.1:9999'\ndiscovery: true\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := parseRuntimeConfig([]string{
		"--config", configPath,
		"--listen", "0.0.0.0:38495",
		"--discovery=false",
	})
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}

	if cfg.ListenAddr != "0.0.0.0:38495" {
		t.Fatalf("listen addr got %q want %q", cfg.ListenAddr, "0.0.0.0:38495")
	}

	if cfg.Discovery {
		t.Fatalf("expected discovery override to be false")
	}
}
