//go:build windows

package guest

import (
	"syscall"
	"testing"
)

func TestParseRegistryPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		path       string
		wantRoot   string
		wantHandle syscall.Handle
		wantSubKey string
		wantErr    bool
	}{
		{
			name:       "current user",
			path:       `HKCU\Software\URLBridge`,
			wantRoot:   "HKCU",
			wantHandle: syscall.HKEY_CURRENT_USER,
			wantSubKey: `Software\URLBridge`,
		},
		{
			name:       "long root name",
			path:       ` HKEY_LOCAL_MACHINE\Software\Microsoft\Windows `,
			wantRoot:   "HKEY_LOCAL_MACHINE",
			wantHandle: syscall.HKEY_LOCAL_MACHINE,
			wantSubKey: `Software\Microsoft\Windows`,
		},
		{
			name:    "missing subkey",
			path:    `HKCU`,
			wantErr: true,
		},
		{
			name:    "unsupported root",
			path:    `HKNOPE\Software\Test`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseRegistryPath(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.RootName != tt.wantRoot {
				t.Fatalf("root got %q want %q", got.RootName, tt.wantRoot)
			}
			if got.RootKey != tt.wantHandle {
				t.Fatalf("handle got %v want %v", got.RootKey, tt.wantHandle)
			}
			if got.SubKey != tt.wantSubKey {
				t.Fatalf("subkey got %q want %q", got.SubKey, tt.wantSubKey)
			}
		})
	}
}
