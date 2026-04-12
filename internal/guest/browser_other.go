//go:build !windows && !linux

package guest

import "fmt"

func OpenLocalBrowser(targetURL string, cfg Config, configPath string, notify func(string)) error {
	return fmt.Errorf("local browser fallback is not supported on this platform")
}
