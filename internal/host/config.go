package host

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/xmoon/urlbridge/internal/bridge"
)

const ConfigFileName = "config.yaml"

type FileConfig struct {
	ListenAddr  string         `yaml:"listen_addr"`
	Token       string         `yaml:"token,omitempty"`
	Discovery   bool           `yaml:"discovery"`
	LogPath     OptionalString `yaml:"log_path,omitempty"`
	LogFullURLs bool           `yaml:"log_full_urls,omitempty"`
}

func DefaultFileConfig() FileConfig {
	return FileConfig{
		ListenAddr: fmt.Sprintf("0.0.0.0:%d", bridge.DefaultPort),
		Discovery:  true,
	}
}

func LoadFileConfig(explicitPath string) (FileConfig, string, error) {
	executable, err := os.Executable()
	if err != nil {
		executable = ""
	}

	lookup, err := bridge.LookupConfigPath(explicitPath, executable, ConfigFileName, platformDefaultConfigPaths())
	if err != nil {
		return FileConfig{}, "", err
	}

	cfg := DefaultFileConfig()
	if !lookup.Found {
		if lookup.Explicit {
			return FileConfig{}, "", missingConfigError(lookup)
		}
		return cfg, "", nil
	}

	if err := bridge.DecodeYAMLFile(lookup.Path, &cfg); err != nil {
		return FileConfig{}, "", fmt.Errorf("load host config: %w", err)
	}

	return cfg, lookup.Path, nil
}

func SaveFileConfig(cfg FileConfig, path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("config path is required")
	}

	if err := cfg.Normalize(); err != nil {
		return err
	}

	data, err := bridge.EncodeYAML(cfg)
	if err != nil {
		return fmt.Errorf("encode host config: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write host config: %w", err)
	}

	return nil
}

func DefaultConfigPath() (string, error) {
	defaultPaths := platformDefaultConfigPaths()
	if len(defaultPaths) > 0 {
		return defaultPaths[0], nil
	}

	executable, err := os.Executable()
	if err != nil {
		executable = ""
	}

	candidates, err := bridge.ConfigCandidatePaths(executable, ConfigFileName, nil)
	if err != nil {
		return "", err
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("default config path is unavailable")
	}

	return candidates[0], nil
}

func (c *FileConfig) Normalize() error {
	c.ListenAddr = strings.TrimSpace(c.ListenAddr)
	c.Token = strings.TrimSpace(c.Token)
	c.LogPath.TrimSpace()

	if c.ListenAddr == "" {
		return fmt.Errorf("listen address is required")
	}

	return nil
}

type OptionalString struct {
	set   bool
	value string
}

func (s *OptionalString) UnmarshalYAML(unmarshal func(any) error) error {
	var value string
	if err := unmarshal(&value); err != nil {
		return err
	}

	s.set = true
	s.value = value
	return nil
}

func (s OptionalString) MarshalYAML() (any, error) {
	if !s.set {
		return nil, nil
	}
	return s.value, nil
}

func (s OptionalString) IsZero() bool {
	return !s.set
}

func (s OptionalString) IsSet() bool {
	return s.set
}

func (s OptionalString) Value() string {
	return s.value
}

func (s *OptionalString) TrimSpace() {
	if !s.set {
		return
	}
	s.value = strings.TrimSpace(s.value)
}

func platformDefaultConfigPaths() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}

	return platformDefaultConfigPathsFor(runtime.GOOS, os.Getenv, home)
}

func platformDefaultConfigPathsFor(goos string, getenv func(string) string, home string) []string {
	switch goos {
	case "windows":
		localAppData := strings.TrimSpace(getenv("LOCALAPPDATA"))
		if localAppData == "" {
			return nil
		}
		return []string{filepath.Join(localAppData, "URLBridgeHost", ConfigFileName)}
	case "linux":
		userPath := strings.TrimSpace(getenv("XDG_CONFIG_HOME"))
		if userPath != "" {
			return []string{
				filepath.Join(userPath, "urlbridge", ConfigFileName),
				filepath.Join(string(filepath.Separator), "etc", "urlbridge", ConfigFileName),
			}
		}
		if home == "" {
			return []string{
				filepath.Join(string(filepath.Separator), "etc", "urlbridge", ConfigFileName),
			}
		}
		return []string{
			filepath.Join(home, ".config", "urlbridge", ConfigFileName),
			filepath.Join(string(filepath.Separator), "etc", "urlbridge", ConfigFileName),
		}
	case "darwin":
		if home == "" {
			return []string{
				filepath.Join(string(filepath.Separator), "etc", "urlbridge", ConfigFileName),
			}
		}
		return []string{
			filepath.Join(home, "Library", "Application Support", "URLBridge", ConfigFileName),
			filepath.Join(string(filepath.Separator), "etc", "urlbridge", ConfigFileName),
		}
	default:
		return nil
	}
}

func missingConfigError(lookup bridge.ConfigLookup) error {
	if lookup.Path != "" && lookup.Explicit {
		return fmt.Errorf("config file not found: %s", lookup.Path)
	}
	if len(lookup.Candidates) == 0 {
		return fmt.Errorf("config file not found")
	}
	return fmt.Errorf("config file not found; searched: %s", strings.Join(lookup.Candidates, ", "))
}
