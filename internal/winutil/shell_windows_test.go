//go:build windows

package winutil

import "testing"

func TestShellExecuteError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		code    uintptr
		wantErr string
	}{
		{name: "success", code: 33},
		{name: "no association", code: 31, wantErr: "there is no application associated with the target"},
		{name: "unknown", code: 1, wantErr: "ShellExecuteW failed with code 1"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := shellExecuteError(tt.code)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("expected error")
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("got %q want %q", err.Error(), tt.wantErr)
			}
		})
	}
}
