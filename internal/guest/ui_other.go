//go:build !windows && !linux

package guest

import "fmt"

func OpenDefaultAppsSettings() error {
	return fmt.Errorf("opening default app settings is not supported on this platform")
}

func ShowErrorDialog(title, message string) {}

func ShowYesNoDialog(title, message string) bool {
	return false
}

func UnsupportedArgumentError(raw string) error {
	return fmt.Errorf("URL Bridge only supports http/https URLs, got %q", raw)
}
