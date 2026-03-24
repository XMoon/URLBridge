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

func OpenLocalBrowser(targetURL, configuredPath string) error {
	candidates := orderedBrowserCandidates(configuredPath, detectLocalBrowserCandidates())
	if len(candidates) == 0 {
		return fmt.Errorf("no local browser found; configure browser_path or install Chrome/Edge")
	}

	var failures []string
	for _, candidate := range candidates {
		if err := startBrowserProcess(candidate.Path, targetURL); err != nil {
			failures = append(failures, fmt.Sprintf("%s (%s): %v", candidate.Name, candidate.Path, err))
			continue
		}
		return nil
	}

	return fmt.Errorf("open local browser: %s", strings.Join(failures, "; "))
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
	output, err := exec.Command("reg", "query", key, "/ve").CombinedOutput()
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(strings.ReplaceAll(string(output), "\r\n", "\n"), "\n") {
		fields := strings.Fields(line)
		for idx, field := range fields {
			if strings.HasPrefix(field, "REG_") && idx+1 < len(fields) {
				return strings.Join(fields[idx+1:], " "), nil
			}
		}
	}

	return "", fmt.Errorf("registry key %s did not contain an executable path", key)
}

func startBrowserProcess(browserPath, targetURL string) error {
	cmd := exec.Command(browserPath, targetURL)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Start()
}
