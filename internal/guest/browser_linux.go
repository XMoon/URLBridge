//go:build linux

package guest

import (
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
)

func OpenLocalBrowser(targetURL string, cfg Config, configPath string, notify func(string)) error {
	return runBrowserFallback(targetURL, browserFallbackRuntime{
		ConfiguredPath:   cfg.BrowserPath,
		ConfigPath:       configPath,
		Notify:           notify,
		DetectCandidates: detectLocalBrowserCandidates,
		StartBrowserProcess: func(browserPath, targetURL string) error {
			return startBrowserProcess(browserPath, targetURL)
		},
		SaveBrowserPath: func(browserPath string) error {
			updated := cfg
			updated.BrowserPath = browserPath
			return SaveConfig(updated, configPath)
		},
	})
}

func detectLocalBrowserCandidates() []browserCandidate {
	var candidates []browserCandidate
	for _, candidate := range []struct {
		name       string
		executable string
	}{
		{name: "Chrome", executable: "google-chrome-stable"},
		{name: "Chrome", executable: "google-chrome"},
		{name: "Chromium", executable: "chromium"},
		{name: "Chromium", executable: "chromium-browser"},
		{name: "Edge", executable: "microsoft-edge-stable"},
		{name: "Edge", executable: "microsoft-edge"},
		{name: "Brave", executable: "brave-browser"},
		{name: "Firefox", executable: "firefox"},
		{name: "Vivaldi", executable: "vivaldi"},
	} {
		path, err := exec.LookPath(candidate.executable)
		if err != nil {
			continue
		}
		candidates = append(candidates, browserCandidate{Name: candidate.name, Path: path})
	}

	return orderedBrowserCandidates("", candidates)
}

func startBrowserProcess(browserPath, targetURL string) error {
	if loopsBackToURLBridge(browserPath) {
		return fmt.Errorf("%s would route back to URL Bridge; configure browser_path to a real browser", browserPath)
	}

	cmd := exec.Command(browserPath, targetURL)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Start()
}

func loopsBackToURLBridge(path string) bool {
	switch strings.ToLower(filepath.Base(strings.TrimSpace(path))) {
	case BrowserBinaryName, "xdg-open", "gio", "gnome-open", "kde-open", "kde-open5":
		return true
	default:
		return false
	}
}
