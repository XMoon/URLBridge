package host

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"

	"github.com/xmoon/urlbridge/internal/winutil"
)

func browserCommand(goos, target string) (string, []string, error) {
	return browserCommandWithLookPath(goos, target, exec.LookPath)
}

func browserCommandWithLookPath(goos, target string, lookPath func(string) (string, error)) (string, []string, error) {
	switch goos {
	case "darwin":
		return "open", []string{target}, nil
	case "linux":
		if _, err := lookPath("xdg-open"); err == nil {
			return "xdg-open", []string{target}, nil
		}
		if _, err := lookPath("gio"); err == nil {
			return "gio", []string{"open", target}, nil
		}
		return "", nil, fmt.Errorf("neither xdg-open nor gio is available")
	default:
		return "", nil, fmt.Errorf("unsupported host OS: %s", goos)
	}
}

func OpenBrowser(ctx context.Context, target string) error {
	if runtime.GOOS == "windows" {
		if err := ctx.Err(); err != nil {
			return err
		}
		return winutil.ShellOpen(target)
	}

	name, args, err := browserCommand(runtime.GOOS, target)
	if err != nil {
		return err
	}

	return exec.CommandContext(ctx, name, args...).Run()
}
