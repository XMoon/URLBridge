package host

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
)

func browserCommand(goos, target string) (string, []string, error) {
	switch goos {
	case "windows":
		return "rundll32.exe", []string{"url.dll,FileProtocolHandler", target}, nil
	case "darwin":
		return "open", []string{target}, nil
	case "linux":
		if _, err := exec.LookPath("xdg-open"); err == nil {
			return "xdg-open", []string{target}, nil
		}
		if _, err := exec.LookPath("gio"); err == nil {
			return "gio", []string{"open", target}, nil
		}
		return "", nil, fmt.Errorf("neither xdg-open nor gio is available")
	default:
		return "", nil, fmt.Errorf("unsupported host OS: %s", goos)
	}
}

func OpenBrowser(ctx context.Context, target string) error {
	name, args, err := browserCommand(runtime.GOOS, target)
	if err != nil {
		return err
	}

	return exec.CommandContext(ctx, name, args...).Run()
}
