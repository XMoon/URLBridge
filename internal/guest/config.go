package guest

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/xmoon/urlbridge/internal/bridge"
)

const (
	ConfigFileName    = "config.yaml"
	RegisteredAppName = bridge.AppName
)

type Config struct {
	HostBaseURL           string `yaml:"host_base_url,omitempty"`
	Token                 string `yaml:"token,omitempty"`
	RequestTimeoutSeconds int    `yaml:"request_timeout_seconds,omitempty"`
	BrowserPath           string `yaml:"browser_path,omitempty"`
}

type ConfigState struct {
	Config     Config
	Path       string
	Found      bool
	Explicit   bool
	Candidates []string
}

func defaultConfig() Config {
	return Config{
		RequestTimeoutSeconds: 3,
	}
}

func ConfigPath() (string, error) {
	return PlatformDefaultConfigPath()
}

func LoadConfigState(explicitPath string) (ConfigState, error) {
	lookup, err := resolveConfigLookup(explicitPath)
	if err != nil {
		return ConfigState{}, err
	}

	state := ConfigState{
		Config:     defaultConfig(),
		Path:       lookup.Path,
		Found:      lookup.Found,
		Explicit:   lookup.Explicit,
		Candidates: lookup.Candidates,
	}
	if !lookup.Found {
		return state, nil
	}

	if err := bridge.DecodeYAMLFile(lookup.Path, &state.Config); err != nil {
		return ConfigState{}, fmt.Errorf("load guest config: %w", err)
	}

	return state, nil
}

func LoadConfig(explicitPath string) (Config, string, error) {
	state, err := LoadConfigState(explicitPath)
	if err != nil {
		return Config{}, "", err
	}
	if !state.Found {
		return Config{}, "", missingConfigError(state)
	}
	if err := state.Config.NormalizeForRuntime(); err != nil {
		return Config{}, "", err
	}

	return state.Config, state.Path, nil
}

func LoadConfigForInstall(explicitPath string) (Config, string, error) {
	state, err := LoadConfigState(explicitPath)
	if err != nil {
		return Config{}, "", err
	}

	if state.Path != "" {
		return state.Config, state.Path, nil
	}

	defaultPath, err := PlatformDefaultConfigPath()
	if err != nil {
		return Config{}, "", err
	}

	return state.Config, defaultPath, nil
}

func SaveConfig(cfg Config, path string) error {
	if path == "" {
		return fmt.Errorf("config path is required")
	}
	if err := cfg.NormalizeForRuntime(); err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := bridge.EncodeYAML(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}

func (c *Config) Normalize() error {
	c.HostBaseURL = strings.TrimSpace(c.HostBaseURL)
	c.Token = strings.TrimSpace(c.Token)
	c.BrowserPath = strings.TrimSpace(c.BrowserPath)

	if c.RequestTimeoutSeconds <= 0 {
		return fmt.Errorf("request timeout must be greater than zero")
	}

	return nil
}

func (c *Config) NormalizeForRuntime() error {
	if err := c.Normalize(); err != nil {
		return err
	}
	if c.HostBaseURL == "" {
		return fmt.Errorf("host base url is required")
	}

	normalized, err := normalizeBaseURL(c.HostBaseURL)
	if err != nil {
		return err
	}
	c.HostBaseURL = normalized

	return nil
}

func resolveConfigLookup(explicitPath string) (bridge.ConfigLookup, error) {
	executable, err := os.Executable()
	if err != nil {
		executable = ""
	}

	defaultPath, err := PlatformDefaultConfigPath()
	if err != nil {
		return bridge.LookupConfigPath(explicitPath, executable, ConfigFileName, nil)
	}

	return bridge.LookupConfigPath(explicitPath, executable, ConfigFileName, []string{defaultPath})
}

func missingConfigError(state ConfigState) error {
	if state.Explicit && state.Path != "" {
		return fmt.Errorf("config file not found: %s", state.Path)
	}
	if len(state.Candidates) == 0 {
		return fmt.Errorf("config file not found")
	}
	return fmt.Errorf("config file not found; searched: %s", strings.Join(state.Candidates, ", "))
}

func normalizeBaseURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("host base url is required")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("parse host base url: %w", err)
	}

	switch parsed.Scheme {
	case "http", "https":
	default:
		return "", fmt.Errorf("host base url must use http or https")
	}

	if parsed.Host == "" {
		return "", fmt.Errorf("host base url must include host:port")
	}

	if parsed.Path == "" {
		parsed.Path = "/"
	}

	return parsed.String(), nil
}

func endpointURL(baseURL, path string) (string, error) {
	normalized, err := normalizeBaseURL(baseURL)
	if err != nil {
		return "", err
	}

	base, err := url.Parse(normalized)
	if err != nil {
		return "", err
	}

	rel, err := url.Parse(path)
	if err != nil {
		return "", err
	}

	return base.ResolveReference(rel).String(), nil
}
