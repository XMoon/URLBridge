//go:build windows

package guest

import (
	"fmt"
	"os/exec"
	"syscall"
	"unsafe"
)

func OpenDefaultAppsSettings() error {
	target := "ms-settings:defaultapps"
	return exec.Command("cmd", "/c", "start", "", target).Start()
}

func ShowErrorDialog(title, message string) {
	showDialog(title, message, 0x00000010)
}

func ShowYesNoDialog(title, message string) bool {
	return showDialog(title, message, 0x00000024) == 6
}

func showDialog(title, message string, flags uintptr) uintptr {
	user32 := syscall.NewLazyDLL("user32.dll")
	messageBox := user32.NewProc("MessageBoxW")
	result, _, _ := messageBox.Call(
		0,
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(message))),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(title))),
		flags,
	)
	return result
}

func UnsupportedArgumentError(raw string) error {
	return fmt.Errorf("URL Bridge only supports http/https URLs, got %q", raw)
}
