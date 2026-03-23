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
	ListenAddr string `yaml:"listen_addr"`
	Token      string `yaml:"token,omitempty"`
	Discovery  bool   `yaml:"discovery"`
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

func (c *FileConfig) Normalize() error {
	c.ListenAddr = strings.TrimSpace(c.ListenAddr)
	c.Token = strings.TrimSpace(c.Token)

	if c.ListenAddr == "" {
		return fmt.Errorf("listen address is required")
	}

	return nil
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
