//go:build !windows && !linux

package guest

import "fmt"

const (
	InstallDirName       = "urlbridge-guest"
	BrowserBinaryName    = "urlbridge-browser"
	ControllerBinaryName = "urlbridge-guestctl"
)

func InstallDir() (string, error) {
	return "", fmt.Errorf("guest install is not supported on this platform")
}

func PlatformDefaultConfigPath() (string, error) {
	return "", fmt.Errorf("guest config is not supported on this platform")
}

func BrowserBinaryPath() (string, error) {
	return "", fmt.Errorf("guest browser binary is not supported on this platform")
}

func ControllerBinaryPath() (string, error) {
	return "", fmt.Errorf("guest controller binary is not supported on this platform")
}
