//go:build windows

package guest

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type InstallOptions struct {
	HostBaseURL      string
	Token            string
	BrowserPath      string
	ConfigPath       string
	OpenSettingsPage bool
	TimeoutSeconds   int
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

	notifyAssociationChanged()
	time.Sleep(500 * time.Millisecond)

	if opts.OpenSettingsPage {
		if err := OpenDefaultAppsSettings(); err != nil {
			return err
		}
	}

	return nil
}

func Uninstall() error {
	var errs []string

	for _, key := range []string{
		`HKCU\Software\Classes\` + HTTPProgID,
		`HKCU\Software\Classes\` + HTTPSProgID,
		`HKCU\Software\` + RegisteredAppName,
	} {
		if err := regDeleteTree(key); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if err := regDeleteValue(`HKCU\Software\RegisteredApplications`, RegisteredAppName); err != nil {
		errs = append(errs, err.Error())
	}

	configPath, err := PlatformDefaultConfigPath()
	if err == nil {
		if removeErr := os.Remove(configPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			errs = append(errs, removeErr.Error())
		}
	}

	notifyAssociationChanged()

	if len(errs) > 0 {
		return fmt.Errorf(strings.Join(errs, "; "))
	}

	return nil
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
	return copyFile(source, destination)
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
	commandValue := fmt.Sprintf(`"%s" --config "%s" "%%1"`, browserPath, configPath)
	iconValue := fmt.Sprintf(`%s,0`, browserPath)
	capabilitiesKey := `HKCU\Software\` + RegisteredAppName + `\Capabilities`

	if err := regAdd(`HKCU\Software\Classes\`+HTTPProgID, "", "REG_SZ", bridgeDescription("HTTP")); err != nil {
		return err
	}
	if err := regAdd(`HKCU\Software\Classes\`+HTTPProgID, "FriendlyTypeName", "REG_SZ", "URL Bridge HTTP Link"); err != nil {
		return err
	}
	if err := regAdd(`HKCU\Software\Classes\`+HTTPProgID, "URL Protocol", "REG_SZ", ""); err != nil {
		return err
	}
	if err := regAdd(`HKCU\Software\Classes\`+HTTPProgID+`\DefaultIcon`, "", "REG_SZ", iconValue); err != nil {
		return err
	}
	if err := regAdd(`HKCU\Software\Classes\`+HTTPProgID+`\shell\open\command`, "", "REG_SZ", commandValue); err != nil {
		return err
	}

	if err := regAdd(`HKCU\Software\Classes\`+HTTPSProgID, "", "REG_SZ", bridgeDescription("HTTPS")); err != nil {
		return err
	}
	if err := regAdd(`HKCU\Software\Classes\`+HTTPSProgID, "FriendlyTypeName", "REG_SZ", "URL Bridge HTTPS Link"); err != nil {
		return err
	}
	if err := regAdd(`HKCU\Software\Classes\`+HTTPSProgID, "URL Protocol", "REG_SZ", ""); err != nil {
		return err
	}
	if err := regAdd(`HKCU\Software\Classes\`+HTTPSProgID+`\DefaultIcon`, "", "REG_SZ", iconValue); err != nil {
		return err
	}
	if err := regAdd(`HKCU\Software\Classes\`+HTTPSProgID+`\shell\open\command`, "", "REG_SZ", commandValue); err != nil {
		return err
	}

	if err := regAdd(capabilitiesKey, "ApplicationName", "REG_SZ", RegisteredAppName); err != nil {
		return err
	}
	if err := regAdd(capabilitiesKey, "ApplicationDescription", "REG_SZ", "Forward links clicked inside the VM to a browser on the host computer."); err != nil {
		return err
	}
	if err := regAdd(capabilitiesKey+`\UrlAssociations`, "http", "REG_SZ", HTTPProgID); err != nil {
		return err
	}
	if err := regAdd(capabilitiesKey+`\UrlAssociations`, "https", "REG_SZ", HTTPSProgID); err != nil {
		return err
	}

	if err := regAdd(`HKCU\Software\RegisteredApplications`, RegisteredAppName, "REG_SZ", `Software\`+RegisteredAppName+`\Capabilities`); err != nil {
		return err
	}

	return nil
}

func bridgeDescription(scheme string) string {
	return fmt.Sprintf("URL Bridge %s link bridge", scheme)
}

func regAdd(path, valueName, valueType, data string) error {
	args := []string{"add", path, "/f", "/t", valueType}
	if valueName == "" {
		args = append(args, "/ve")
	} else {
		args = append(args, "/v", valueName)
	}
	args = append(args, "/d", data)

	output, err := exec.Command("reg", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("reg add %s: %w: %s", path, err, strings.TrimSpace(string(output)))
	}

	return nil
}

func regDeleteTree(path string) error {
	output, err := exec.Command("reg", "delete", path, "/f").CombinedOutput()
	if err != nil {
		text := strings.TrimSpace(string(output))
		if strings.Contains(strings.ToLower(text), "unable to find") {
			return nil
		}
		return fmt.Errorf("reg delete %s: %w: %s", path, err, text)
	}
	return nil
}

func regDeleteValue(path, valueName string) error {
	output, err := exec.Command("reg", "delete", path, "/f", "/v", valueName).CombinedOutput()
	if err != nil {
		text := strings.TrimSpace(string(output))
		if strings.Contains(strings.ToLower(text), "unable to find") {
			return nil
		}
		return fmt.Errorf("reg delete value %s/%s: %w: %s", path, valueName, err, text)
	}
	return nil
}

func notifyAssociationChanged() {
	shell32 := syscall.NewLazyDLL("shell32.dll")
	proc := shell32.NewProc("SHChangeNotify")
	const (
		shcneAssocChanged = 0x08000000
		shcnfDWORD        = 0x0003
		shcnfFlush        = 0x1000
	)
	_, _, _ = proc.Call(uintptr(shcneAssocChanged), uintptr(shcnfDWORD|shcnfFlush), 0, 0)
}
