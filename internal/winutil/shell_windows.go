//go:build windows

package winutil

import (
	"fmt"
	"syscall"
	"unsafe"
)

const (
	swShowNormal = 1
)

var (
	shell32           = syscall.NewLazyDLL("shell32.dll")
	procShellExecuteW = shell32.NewProc("ShellExecuteW")
)

func ShellOpen(target string) error {
	verbPtr, err := syscall.UTF16PtrFromString("open")
	if err != nil {
		return err
	}

	targetPtr, err := syscall.UTF16PtrFromString(target)
	if err != nil {
		return err
	}

	result, _, _ := procShellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(verbPtr)),
		uintptr(unsafe.Pointer(targetPtr)),
		0,
		0,
		swShowNormal,
	)
	if err := shellExecuteError(result); err != nil {
		return fmt.Errorf("shell open %q: %w", target, err)
	}

	return nil
}

func shellExecuteError(code uintptr) error {
	if code > 32 {
		return nil
	}

	switch code {
	case 0:
		return fmt.Errorf("the operating system is out of memory or resources")
	case 2:
		return fmt.Errorf("the specified file was not found")
	case 3:
		return fmt.Errorf("the specified path was not found")
	case 5:
		return fmt.Errorf("access was denied")
	case 8:
		return fmt.Errorf("the operating system is out of memory")
	case 26:
		return fmt.Errorf("a sharing violation occurred")
	case 27:
		return fmt.Errorf("the file association information is incomplete or invalid")
	case 28:
		return fmt.Errorf("the DDE operation timed out")
	case 29:
		return fmt.Errorf("the DDE transaction failed")
	case 30:
		return fmt.Errorf("the DDE operation is busy")
	case 31:
		return fmt.Errorf("there is no application associated with the target")
	case 32:
		return fmt.Errorf("the specified DLL was not found")
	default:
		return fmt.Errorf("ShellExecuteW failed with code %d", code)
	}
}
