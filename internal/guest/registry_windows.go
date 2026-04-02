//go:build windows

package guest

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

const (
	registryTypeSZ       = 1
	registryTypeExpandSZ = 2
)

type registryLocation struct {
	RootName string
	RootKey  syscall.Handle
	SubKey   string
}

var (
	advapi32               = syscall.NewLazyDLL("advapi32.dll")
	kernel32               = syscall.NewLazyDLL("kernel32.dll")
	procRegCreateKeyExW    = advapi32.NewProc("RegCreateKeyExW")
	procRegDeleteTreeW     = advapi32.NewProc("RegDeleteTreeW")
	procRegDeleteValueW    = advapi32.NewProc("RegDeleteValueW")
	procRegSetValueExW     = advapi32.NewProc("RegSetValueExW")
	procExpandEnvStringsW  = kernel32.NewProc("ExpandEnvironmentStringsW")
	registryRootKeyHandles = map[string]syscall.Handle{
		"HKCR":                syscall.HKEY_CLASSES_ROOT,
		"HKEY_CLASSES_ROOT":   syscall.HKEY_CLASSES_ROOT,
		"HKCU":                syscall.HKEY_CURRENT_USER,
		"HKEY_CURRENT_USER":   syscall.HKEY_CURRENT_USER,
		"HKLM":                syscall.HKEY_LOCAL_MACHINE,
		"HKEY_LOCAL_MACHINE":  syscall.HKEY_LOCAL_MACHINE,
		"HKU":                 syscall.HKEY_USERS,
		"HKEY_USERS":          syscall.HKEY_USERS,
		"HKCC":                syscall.HKEY_CURRENT_CONFIG,
		"HKEY_CURRENT_CONFIG": syscall.HKEY_CURRENT_CONFIG,
	}
)

func parseRegistryPath(path string) (registryLocation, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return registryLocation{}, fmt.Errorf("registry path is empty")
	}

	rootName, subKey, found := strings.Cut(trimmed, `\`)
	if !found || strings.TrimSpace(subKey) == "" {
		return registryLocation{}, fmt.Errorf("registry path %q is missing a subkey", path)
	}

	normalizedRoot := strings.ToUpper(strings.TrimSpace(rootName))
	rootKey, ok := registryRootKeyHandles[normalizedRoot]
	if !ok {
		return registryLocation{}, fmt.Errorf("registry path %q uses unsupported root %q", path, rootName)
	}

	return registryLocation{
		RootName: normalizedRoot,
		RootKey:  rootKey,
		SubKey:   strings.TrimLeft(subKey, `\`),
	}, nil
}

func readRegistryStringValue(path, valueName string) (string, error) {
	location, err := parseRegistryPath(path)
	if err != nil {
		return "", err
	}

	key, err := openRegistryKey(location, syscall.KEY_QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer syscall.RegCloseKey(key)

	return readOpenRegistryStringValue(key, valueName)
}

func writeRegistryStringValue(path, valueName, data string) error {
	location, err := parseRegistryPath(path)
	if err != nil {
		return err
	}

	key, err := createRegistryKey(location, syscall.KEY_SET_VALUE|syscall.KEY_CREATE_SUB_KEY)
	if err != nil {
		return err
	}
	defer syscall.RegCloseKey(key)

	return setOpenRegistryStringValue(key, valueName, data)
}

func deleteRegistryTree(path string) error {
	location, err := parseRegistryPath(path)
	if err != nil {
		return err
	}

	return callRegDeleteTree(location.RootKey, location.SubKey)
}

func deleteRegistryValue(path, valueName string) error {
	location, err := parseRegistryPath(path)
	if err != nil {
		return err
	}

	key, err := openRegistryKey(location, syscall.KEY_SET_VALUE)
	if err != nil {
		if errorsIsRegistryNotFound(err) {
			return nil
		}
		return err
	}
	defer syscall.RegCloseKey(key)

	if err := callRegDeleteValue(key, valueName); err != nil {
		if errorsIsRegistryNotFound(err) {
			return nil
		}
		return err
	}

	return nil
}

func openRegistryKey(location registryLocation, access uint32) (syscall.Handle, error) {
	subKeyPtr, err := syscall.UTF16PtrFromString(location.SubKey)
	if err != nil {
		return 0, err
	}

	var key syscall.Handle
	if err := syscall.RegOpenKeyEx(location.RootKey, subKeyPtr, 0, access, &key); err != nil {
		return 0, err
	}

	return key, nil
}

func createRegistryKey(location registryLocation, access uint32) (syscall.Handle, error) {
	subKeyPtr, err := syscall.UTF16PtrFromString(location.SubKey)
	if err != nil {
		return 0, err
	}

	var key syscall.Handle
	result, _, _ := procRegCreateKeyExW.Call(
		uintptr(location.RootKey),
		uintptr(unsafe.Pointer(subKeyPtr)),
		0,
		0,
		0,
		uintptr(access),
		0,
		uintptr(unsafe.Pointer(&key)),
		0,
	)
	if errno := syscall.Errno(result); errno != 0 {
		return 0, errno
	}

	return key, nil
}

func readOpenRegistryStringValue(key syscall.Handle, valueName string) (string, error) {
	valueNamePtr, err := syscall.UTF16PtrFromString(valueName)
	if err != nil {
		return "", err
	}

	var valueType uint32
	var byteLen uint32
	if err := syscall.RegQueryValueEx(key, valueNamePtr, nil, &valueType, nil, &byteLen); err != nil {
		return "", err
	}

	if valueType != registryTypeSZ && valueType != registryTypeExpandSZ {
		return "", fmt.Errorf("unexpected registry value type %d", valueType)
	}
	if byteLen == 0 {
		return "", nil
	}

	data := make([]byte, byteLen)
	if err := syscall.RegQueryValueEx(key, valueNamePtr, nil, &valueType, &data[0], &byteLen); err != nil {
		return "", err
	}

	text := utf16BytesToString(data[:byteLen])
	if valueType == registryTypeExpandSZ {
		return expandEnvironmentStrings(text)
	}

	return text, nil
}

func setOpenRegistryStringValue(key syscall.Handle, valueName, data string) error {
	valueNamePtr, err := syscall.UTF16PtrFromString(valueName)
	if err != nil {
		return err
	}

	encoded, err := syscall.UTF16FromString(data)
	if err != nil {
		return err
	}

	result, _, _ := procRegSetValueExW.Call(
		uintptr(key),
		uintptr(unsafe.Pointer(valueNamePtr)),
		0,
		registryTypeSZ,
		uintptr(unsafe.Pointer(&encoded[0])),
		uintptr(len(encoded)*2),
	)
	if errno := syscall.Errno(result); errno != 0 {
		return errno
	}

	return nil
}

func callRegDeleteTree(rootKey syscall.Handle, subKey string) error {
	subKeyPtr, err := syscall.UTF16PtrFromString(subKey)
	if err != nil {
		return err
	}

	result, _, _ := procRegDeleteTreeW.Call(uintptr(rootKey), uintptr(unsafe.Pointer(subKeyPtr)))
	if errno := syscall.Errno(result); errno != 0 {
		return errno
	}

	return nil
}

func callRegDeleteValue(key syscall.Handle, valueName string) error {
	valueNamePtr, err := syscall.UTF16PtrFromString(valueName)
	if err != nil {
		return err
	}

	result, _, _ := procRegDeleteValueW.Call(uintptr(key), uintptr(unsafe.Pointer(valueNamePtr)))
	if errno := syscall.Errno(result); errno != 0 {
		return errno
	}

	return nil
}

func expandEnvironmentStrings(value string) (string, error) {
	if value == "" {
		return "", nil
	}

	src, err := syscall.UTF16PtrFromString(value)
	if err != nil {
		return "", err
	}

	size := uint32(len(value) + 1)
	if size < 64 {
		size = 64
	}

	for {
		buf := make([]uint16, size)
		result, _, callErr := procExpandEnvStringsW.Call(
			uintptr(unsafe.Pointer(src)),
			uintptr(unsafe.Pointer(&buf[0])),
			uintptr(size),
		)
		if result == 0 {
			if callErr != syscall.Errno(0) {
				return "", callErr
			}
			return "", fmt.Errorf("ExpandEnvironmentStringsW failed")
		}
		if uint32(result) <= size {
			return syscall.UTF16ToString(buf[:result]), nil
		}
		size = uint32(result)
	}
}

func utf16BytesToString(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	if len(data)%2 != 0 {
		data = data[:len(data)-1]
	}
	if len(data) == 0 {
		return ""
	}

	chars := unsafe.Slice((*uint16)(unsafe.Pointer(&data[0])), len(data)/2)
	return syscall.UTF16ToString(chars)
}

func errorsIsRegistryNotFound(err error) bool {
	return err == syscall.ERROR_FILE_NOT_FOUND || err == syscall.ERROR_PATH_NOT_FOUND
}
