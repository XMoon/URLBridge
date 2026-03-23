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
	user32 := syscall.NewLazyDLL("user32.dll")
	messageBox := user32.NewProc("MessageBoxW")
	_, _, _ = messageBox.Call(
		0,
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(message))),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(title))),
		0x00000010,
	)
}

func UnsupportedArgumentError(raw string) error {
	return fmt.Errorf("URL Bridge only supports http/https URLs, got %q", raw)
}
