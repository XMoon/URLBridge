package main

import (
	"os"
	"path/filepath"
	"strings"
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

func TestParseRuntimeConfigGeneratesTokenInDefaultConfig(t *testing.T) {
	dir := t.TempDir()
	xdgConfigHome := filepath.Join(dir, "xdg")

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalWD); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	}()

	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", xdgConfigHome)

	cfg, err := parseRuntimeConfig(nil)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}

	if !cfg.TokenGenerated {
		t.Fatalf("expected token to be generated")
	}
	if strings.TrimSpace(cfg.Token) == "" {
		t.Fatalf("expected generated token to be set")
	}

	wantConfigPath := filepath.Join(xdgConfigHome, "urlbridge", "config.yaml")
	if cfg.ConfigPath != wantConfigPath {
		t.Fatalf("config path got %q want %q", cfg.ConfigPath, wantConfigPath)
	}

	data, err := os.ReadFile(wantConfigPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "token:") {
		t.Fatalf("expected token to be persisted, got %q", content)
	}
	if !strings.Contains(content, cfg.Token) {
		t.Fatalf("expected generated token to be written to config")
	}
}
