//go:build windows

package guest

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func OpenLocalBrowser(targetURL string, cfg Config, configPath string, notify func(string)) error {
	return runBrowserFallback(targetURL, browserFallbackRuntime{
		ConfiguredPath: cfg.BrowserPath,
		ConfigPath:     configPath,
		Notify:         notify,
		ConfirmReplacement: func(failedPath string, candidate browserCandidate) bool {
			return ShowYesNoDialog("URL Bridge", replacementBrowserPrompt(failedPath, candidate))
		},
		DetectCandidates:    detectLocalBrowserCandidates,
		StartBrowserProcess: startBrowserProcess,
		SaveBrowserPath: func(browserPath string) error {
			updated := cfg
			updated.BrowserPath = browserPath
			return SaveConfig(updated, configPath)
		},
	})
}

func detectLocalBrowserCandidates() []browserCandidate {
	var candidates []browserCandidate
	candidates = append(candidates, browserInstallCandidates("Chrome", "chrome.exe", []string{
		`HKCU\Software\Microsoft\Windows\CurrentVersion\App Paths\chrome.exe`,
		`HKLM\Software\Microsoft\Windows\CurrentVersion\App Paths\chrome.exe`,
	}, chromeInstallPaths())...)
	candidates = append(candidates, browserInstallCandidates("Edge", "msedge.exe", []string{
		`HKCU\Software\Microsoft\Windows\CurrentVersion\App Paths\msedge.exe`,
		`HKLM\Software\Microsoft\Windows\CurrentVersion\App Paths\msedge.exe`,
	}, edgeInstallPaths())...)
	return candidates
}

func browserInstallCandidates(name, executable string, registryKeys, installPaths []string) []browserCandidate {
	var candidates []browserCandidate

	for _, key := range registryKeys {
		if path, err := readAppPathFromRegistry(key); err == nil {
			candidates = append(candidates, browserCandidate{Name: name, Path: path})
		}
	}

	for _, path := range installPaths {
		if path == "" {
			continue
		}
		if _, err := os.Stat(path); err == nil {
			candidates = append(candidates, browserCandidate{Name: name, Path: path})
		}
	}

	if lookPath, err := exec.LookPath(executable); err == nil {
		candidates = append(candidates, browserCandidate{Name: name, Path: lookPath})
	}

	return orderedBrowserCandidates("", candidates)
}

func chromeInstallPaths() []string {
	return commonBrowserInstallPaths("Google", "Chrome", "chrome.exe")
}

func edgeInstallPaths() []string {
	return commonBrowserInstallPaths("Microsoft", "Edge", "msedge.exe")
}

func commonBrowserInstallPaths(vendor, product, executable string) []string {
	var candidates []string

	for _, root := range []string{
		strings.TrimSpace(os.Getenv("ProgramFiles")),
		strings.TrimSpace(os.Getenv("ProgramFiles(x86)")),
		strings.TrimSpace(os.Getenv("LocalAppData")),
	} {
		if root == "" {
			continue
		}
		candidates = append(candidates, filepath.Join(root, vendor, product, "Application", executable))
	}

	return candidates
}

func readAppPathFromRegistry(key string) (string, error) {
	path, err := readRegistryStringValue(key, "")
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("registry key %s did not contain an executable path", key)
	}
	return path, nil
}

func startBrowserProcess(browserPath, targetURL string) error {
	cmd := exec.Command(browserPath, targetURL)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Start()
}

func replacementBrowserPrompt(failedPath string, candidate browserCandidate) string {
	return fmt.Sprintf(
		"The configured local browser failed to start:\n%s\n\nUse %s for this link and update browser_path?\n%s",
		strings.TrimSpace(failedPath),
		candidate.Name,
		candidate.Path,
	)
}
