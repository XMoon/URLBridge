package host

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const hostLogFileName = "host.log"

func NewLogger(stdout io.Writer, cfg FileConfig) (*log.Logger, io.Closer, error) {
	if stdout == nil {
		stdout = io.Discard
	}

	path, enabled, required := resolveLogFilePath(cfg.LogPath)
	if !enabled {
		return log.New(stdout, "", log.LstdFlags), nil, nil
	}

	file, err := openLogFile(path)
	if err != nil {
		baseLogger := log.New(stdout, "", log.LstdFlags)
		if required {
			return nil, nil, fmt.Errorf("open log file %q: %w", path, err)
		}

		baseLogger.Printf("warning: failed to open default log file %s: %v; continuing with stdout only", path, err)
		return baseLogger, nil, nil
	}

	return log.New(io.MultiWriter(stdout, file), "", log.LstdFlags), file, nil
}

func resolveLogFilePath(value OptionalString) (path string, enabled bool, required bool) {
	if value.IsSet() {
		trimmed := strings.TrimSpace(value.Value())
		if trimmed == "" {
			return "", false, false
		}
		return trimmed, true, true
	}

	return defaultLogPath(), true, false
}

func defaultLogPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}

	return platformDefaultLogPathFor(runtime.GOOS, os.Getenv, home, os.TempDir())
}

func platformDefaultLogPathFor(goos string, getenv func(string) string, home, tempDir string) string {
	switch goos {
	case "windows":
		localAppData := strings.TrimSpace(getenv("LOCALAPPDATA"))
		if localAppData != "" {
			return filepath.Join(localAppData, "URLBridgeHost", hostLogFileName)
		}
		return filepath.Join(tempDir, "URLBridgeHost", hostLogFileName)
	case "linux":
		xdgStateHome := strings.TrimSpace(getenv("XDG_STATE_HOME"))
		if xdgStateHome != "" {
			return filepath.Join(xdgStateHome, "urlbridge", hostLogFileName)
		}
		if home != "" {
			return filepath.Join(home, ".local", "state", "urlbridge", hostLogFileName)
		}
		return filepath.Join(tempDir, "urlbridge", hostLogFileName)
	case "darwin":
		if home != "" {
			return filepath.Join(home, "Library", "Logs", "URLBridge", hostLogFileName)
		}
		return filepath.Join(tempDir, "urlbridge", hostLogFileName)
	default:
		return filepath.Join(tempDir, "urlbridge", hostLogFileName)
	}
}

func openLogFile(path string) (*os.File, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}

	return file, nil
}
