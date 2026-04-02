//go:build !windows

package winutil

import "fmt"

func ShellOpen(target string) error {
	return fmt.Errorf("shell open %q: only supported on Windows", target)
}
