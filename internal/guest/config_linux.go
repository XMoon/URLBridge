//go:build linux

package guest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	InstallDirName       = "urlbridge-guest"
	BrowserBinaryName    = "urlbridge-browser"
	ControllerBinaryName = "urlbridge-guestctl"
)

func InstallDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return "", fmt.Errorf("user home directory is unavailable")
	}
	return filepath.Join(home, ".local", "lib", InstallDirName), nil
}

func PlatformDefaultConfigPath() (string, error) {
	configHome := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME"))
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil || strings.TrimSpace(home) == "" {
			return "", fmt.Errorf("user config directory is unavailable")
		}
		configHome = filepath.Join(home, ".config")
	}
	return filepath.Join(configHome, InstallDirName, ConfigFileName), nil
}

func BrowserBinaryPath() (string, error) {
	installDir, err := InstallDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(installDir, BrowserBinaryName), nil
}

func ControllerBinaryPath() (string, error) {
	installDir, err := InstallDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(installDir, ControllerBinaryName), nil
}
