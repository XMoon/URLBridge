//go:build linux

package guest

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const (
	DesktopFileName = "urlbridge-browser.desktop"
)

var schemeHandlerMIMETypes = []string{
	"x-scheme-handler/http",
	"x-scheme-handler/https",
}

type InstallOptions struct {
	HostBaseURL      string
	Token            string
	BrowserPath      string
	ConfigPath       string
	OpenSettingsPage bool
	TimeoutSeconds   int
}

type SchemeHandlerStatus struct {
	DesktopFilePath string
	HTTPDefault     string
	HTTPSDefault    string
}

func Install(opts InstallOptions) error {
	if opts.TimeoutSeconds <= 0 {
		opts.TimeoutSeconds = 3
	}

	installDir, err := InstallDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return fmt.Errorf("create install dir: %w", err)
	}

	if err := copyDistributionBinary(ControllerBinaryName); err != nil {
		return err
	}
	if err := copyDistributionBinary(BrowserBinaryName); err != nil {
		return err
	}

	configPath := strings.TrimSpace(opts.ConfigPath)
	if configPath == "" {
		configPath, err = ConfigPath()
		if err != nil {
			return err
		}
	}

	if err := SaveConfig(Config{
		HostBaseURL:           opts.HostBaseURL,
		Token:                 opts.Token,
		RequestTimeoutSeconds: opts.TimeoutSeconds,
		BrowserPath:           opts.BrowserPath,
	}, configPath); err != nil {
		return err
	}

	browserPath, err := BrowserBinaryPath()
	if err != nil {
		return err
	}

	if err := registerBrowser(browserPath, configPath); err != nil {
		return err
	}

	return nil
}

func URLSchemeHandlerStatus() SchemeHandlerStatus {
	status := SchemeHandlerStatus{
		HTTPDefault:  queryDefaultSchemeHandler("x-scheme-handler/http"),
		HTTPSDefault: queryDefaultSchemeHandler("x-scheme-handler/https"),
	}

	applicationsDir, err := applicationsDir()
	if err == nil {
		status.DesktopFilePath = filepath.Join(applicationsDir, DesktopFileName)
	}

	return status
}

func Uninstall() error {
	var errs []string

	if err := unregisterBrowser(); err != nil {
		errs = append(errs, err.Error())
	}

	configPath, err := PlatformDefaultConfigPath()
	if err == nil {
		if removeErr := os.Remove(configPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			errs = append(errs, removeErr.Error())
		}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}

	return nil
}

func queryDefaultSchemeHandler(mimeType string) string {
	path, err := exec.LookPath("xdg-mime")
	if err == nil {
		out, err := exec.Command(path, "query", "default", mimeType).Output()
		if err == nil {
			return strings.TrimSpace(string(out))
		}
	}

	mimeAppsPath, err := mimeAppsPath()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(mimeAppsPath)
	if err != nil {
		return ""
	}

	return mimeAppsDefault(data, mimeType)
}

func copyDistributionBinary(name string) error {
	source, err := distributionBinaryPath(name)
	if err != nil {
		return err
	}

	destinationDir, err := InstallDir()
	if err != nil {
		return err
	}

	destination := filepath.Join(destinationDir, name)
	if err := copyFile(source, destination); err != nil {
		return err
	}

	return os.Chmod(destination, 0o755)
}

func distributionBinaryPath(name string) (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable path: %w", err)
	}

	if name == ControllerBinaryName {
		return executable, nil
	}

	dir := filepath.Dir(executable)
	candidates := []string{filepath.Join(dir, name)}

	if suffix := controllerVariantSuffix(filepath.Base(executable)); suffix != "" {
		ext := filepath.Ext(name)
		stem := strings.TrimSuffix(name, ext)
		candidates = append([]string{filepath.Join(dir, stem+suffix+ext)}, candidates...)
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("required binary %q not found next to %q", name, executable)
}

func controllerVariantSuffix(executableName string) string {
	ext := filepath.Ext(ControllerBinaryName)
	defaultStem := strings.TrimSuffix(ControllerBinaryName, ext)
	executableStem := strings.TrimSuffix(executableName, filepath.Ext(executableName))

	if !strings.HasPrefix(executableStem, defaultStem) {
		return ""
	}

	return strings.TrimPrefix(executableStem, defaultStem)
}

func copyFile(source, destination string) error {
	if filepath.Clean(source) == filepath.Clean(destination) {
		return nil
	}

	src, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open source binary: %w", err)
	}
	defer src.Close()

	tmp := destination + ".tmp"
	dst, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("create destination binary: %w", err)
	}

	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		return fmt.Errorf("copy binary: %w", err)
	}

	if err := dst.Close(); err != nil {
		return fmt.Errorf("close destination binary: %w", err)
	}

	if err := os.Remove(destination); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove previous destination binary: %w", err)
	}

	if err := os.Rename(tmp, destination); err != nil {
		return fmt.Errorf("move binary into place: %w", err)
	}

	return nil
}

func registerBrowser(browserPath, configPath string) error {
	applicationsDir, err := applicationsDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(applicationsDir, 0o755); err != nil {
		return fmt.Errorf("create applications dir: %w", err)
	}

	desktopPath := filepath.Join(applicationsDir, DesktopFileName)
	if err := os.WriteFile(desktopPath, []byte(desktopFileContent(browserPath, configPath)), 0o644); err != nil {
		return fmt.Errorf("write desktop entry: %w", err)
	}

	refreshDesktopDatabase(applicationsDir)

	if err := setDefaultSchemeHandlers(DesktopFileName); err != nil {
		return err
	}

	return nil
}

func unregisterBrowser() error {
	var errs []string

	applicationsDir, err := applicationsDir()
	if err == nil {
		if removeErr := os.Remove(filepath.Join(applicationsDir, DesktopFileName)); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			errs = append(errs, removeErr.Error())
		}
		refreshDesktopDatabase(applicationsDir)
	}

	if err := removeDefaultSchemeHandlers(DesktopFileName); err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}

	return nil
}

func applicationsDir() (string, error) {
	dataHome := strings.TrimSpace(os.Getenv("XDG_DATA_HOME"))
	if dataHome == "" {
		home, err := os.UserHomeDir()
		if err != nil || strings.TrimSpace(home) == "" {
			return "", fmt.Errorf("user data directory is unavailable")
		}
		dataHome = filepath.Join(home, ".local", "share")
	}

	return filepath.Join(dataHome, "applications"), nil
}

func mimeAppsPath() (string, error) {
	configHome := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME"))
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil || strings.TrimSpace(home) == "" {
			return "", fmt.Errorf("user config directory is unavailable")
		}
		configHome = filepath.Join(home, ".config")
	}

	return filepath.Join(configHome, "mimeapps.list"), nil
}

func desktopFileContent(browserPath, configPath string) string {
	return fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=URL Bridge
Comment=Forward links clicked inside the VM to the host browser
Exec=%s --config %s %%u
Icon=web-browser
Terminal=false
NoDisplay=true
MimeType=x-scheme-handler/http;x-scheme-handler/https;
Categories=Network;WebBrowser;
`, desktopExecField(browserPath), desktopExecField(configPath))
}

func desktopExecField(value string) string {
	var b strings.Builder
	for _, r := range value {
		switch r {
		case ' ', '\t', '\n', '"', '\'', '\\', '`', '$':
			b.WriteByte('\\')
		case '%':
			b.WriteByte('%')
		}
		b.WriteRune(r)
	}
	return b.String()
}

func refreshDesktopDatabase(applicationsDir string) {
	path, err := exec.LookPath("update-desktop-database")
	if err != nil {
		return
	}

	_ = exec.Command(path, applicationsDir).Run()
}

func setDefaultSchemeHandlers(desktopFileName string) error {
	path, err := exec.LookPath("xdg-mime")
	if err == nil {
		var errs []string
		for _, mimeType := range schemeHandlerMIMETypes {
			if err := exec.Command(path, "default", desktopFileName, mimeType).Run(); err != nil {
				errs = append(errs, fmt.Sprintf("%s: %v", mimeType, err))
			}
		}
		if len(errs) == 0 {
			return nil
		}

		if err := updateMimeAppsDefaults(desktopFileName); err != nil {
			return fmt.Errorf("xdg-mime default failed (%s); fallback update mimeapps list: %w", strings.Join(errs, "; "), err)
		}
		return nil
	}

	return updateMimeAppsDefaults(desktopFileName)
}

func updateMimeAppsDefaults(desktopFileName string) error {
	path, err := mimeAppsPath()
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read mimeapps list: %w", err)
	}

	updated := upsertMimeAppsDefaults(data, desktopFileName)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create mimeapps dir: %w", err)
	}

	return os.WriteFile(path, updated, 0o644)
}

func removeDefaultSchemeHandlers(desktopFileName string) error {
	path, err := mimeAppsPath()
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read mimeapps list: %w", err)
	}

	updated := removeMimeAppsDefaults(data, desktopFileName)
	return os.WriteFile(path, updated, 0o644)
}

func upsertMimeAppsDefaults(data []byte, desktopFileName string) []byte {
	defaults := schemeHandlerDefaults(desktopFileName)
	remaining := schemeHandlerDefaultKeys(defaults)
	lines := splitConfigLines(string(data))

	var out []string
	inDefaults := false
	foundDefaultsSection := false

	appendRemaining := func() {
		for _, key := range remaining {
			out = append(out, fmt.Sprintf("%s=%s", key, defaults[key]))
		}
		remaining = nil
	}

	for _, line := range lines {
		sectionName, isSection := configSectionName(line)
		if isSection {
			if inDefaults {
				appendRemaining()
			}
			inDefaults = sectionName == "Default Applications"
			if inDefaults {
				foundDefaultsSection = true
			}
			out = append(out, line)
			continue
		}

		if inDefaults {
			if key, _, ok := strings.Cut(line, "="); ok {
				key = strings.TrimSpace(key)
				if _, exists := defaults[key]; exists {
					if containsString(remaining, key) {
						out = append(out, fmt.Sprintf("%s=%s", key, defaults[key]))
						remaining = removeString(remaining, key)
					}
					continue
				}
			}
		}

		out = append(out, line)
	}

	if inDefaults {
		appendRemaining()
	}

	if !foundDefaultsSection {
		if len(out) > 0 && out[len(out)-1] != "" {
			out = append(out, "")
		}
		out = append(out, "[Default Applications]")
		remaining = schemeHandlerDefaultKeys(defaults)
		appendRemaining()
	}

	return []byte(strings.Join(out, "\n") + "\n")
}

func removeMimeAppsDefaults(data []byte, desktopFileName string) []byte {
	defaults := schemeHandlerDefaults(desktopFileName)
	lines := splitConfigLines(string(data))

	var out []string
	inDefaults := false

	for _, line := range lines {
		sectionName, isSection := configSectionName(line)
		if isSection {
			inDefaults = sectionName == "Default Applications"
			out = append(out, line)
			continue
		}

		if inDefaults {
			if key, value, ok := strings.Cut(line, "="); ok {
				key = strings.TrimSpace(key)
				if _, exists := defaults[key]; exists {
					filteredValue, changed := removeDesktopFromMimeValue(value, desktopFileName)
					if changed {
						if strings.TrimSpace(filteredValue) != "" {
							out = append(out, fmt.Sprintf("%s=%s", key, filteredValue))
						}
						continue
					}
				}
			}
		}

		out = append(out, line)
	}

	return []byte(strings.Join(out, "\n") + "\n")
}

func mimeAppsDefault(data []byte, mimeType string) string {
	lines := splitConfigLines(string(data))
	inDefaults := false

	for _, line := range lines {
		sectionName, isSection := configSectionName(line)
		if isSection {
			inDefaults = sectionName == "Default Applications"
			continue
		}
		if !inDefaults {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) != mimeType {
			continue
		}

		return firstDesktopInMimeValue(value)
	}

	return ""
}

func schemeHandlerDefaults(desktopFileName string) map[string]string {
	defaults := make(map[string]string, len(schemeHandlerMIMETypes))
	for _, mimeType := range schemeHandlerMIMETypes {
		defaults[mimeType] = desktopFileName
	}
	return defaults
}

func schemeHandlerDefaultKeys(defaults map[string]string) []string {
	keys := make([]string, 0, len(defaults))
	for key := range defaults {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func firstDesktopInMimeValue(value string) string {
	for _, part := range strings.Split(value, ";") {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func splitConfigLines(data string) []string {
	normalized := strings.ReplaceAll(data, "\r\n", "\n")
	normalized = strings.TrimRight(normalized, "\n")
	if normalized == "" {
		return nil
	}
	return strings.Split(normalized, "\n")
}

func configSectionName(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "[") || !strings.HasSuffix(trimmed, "]") {
		return "", false
	}
	return strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(trimmed, "["), "]")), true
}

func removeDesktopFromMimeValue(value, desktopFileName string) (string, bool) {
	parts := strings.Split(value, ";")
	filtered := make([]string, 0, len(parts))
	changed := false

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		if trimmed == desktopFileName {
			changed = true
			continue
		}
		filtered = append(filtered, trimmed)
	}

	if !changed {
		return value, false
	}
	if len(filtered) == 0 {
		return "", true
	}

	return strings.Join(filtered, ";") + ";", true
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func removeString(values []string, needle string) []string {
	filtered := values[:0]
	for _, value := range values {
		if value == needle {
			continue
		}
		filtered = append(filtered, value)
	}
	return filtered
}
