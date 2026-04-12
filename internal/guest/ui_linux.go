//go:build linux

package guest

import (
	"fmt"
	"os"
	"os/exec"
)

func OpenDefaultAppsSettings() error {
	return fmt.Errorf("opening default app settings is not supported on Linux; use install to register the xdg-open handler")
}

func ShowErrorDialog(title, message string) {
	if notifyDesktop(title, message) {
		return
	}
	fmt.Fprintf(os.Stderr, "%s: %s\n", title, message)
}

func ShowYesNoDialog(title, message string) bool {
	ShowErrorDialog(title, message)
	return false
}

func notifyDesktop(title, message string) bool {
	path, err := exec.LookPath("notify-send")
	if err != nil {
		return false
	}

	return exec.Command(path, title, message).Start() == nil
}

func UnsupportedArgumentError(raw string) error {
	return fmt.Errorf("URL Bridge only supports http/https URLs, got %q", raw)
}
