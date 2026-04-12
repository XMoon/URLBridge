//go:build windows

package guest

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	InstallDirName       = "URLBridge"
	BrowserBinaryName    = "urlbridge-browser.exe"
	ControllerBinaryName = "urlbridge-guestctl.exe"
	HTTPProgID           = "URLBridge.Url.Http"
	HTTPSProgID          = "URLBridge.Url.Https"
)

func InstallDir() (string, error) {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		return "", fmt.Errorf("LOCALAPPDATA is not set")
	}
	return filepath.Join(localAppData, InstallDirName), nil
}

func PlatformDefaultConfigPath() (string, error) {
	installDir, err := InstallDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(installDir, ConfigFileName), nil
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
