package host

import (
	"errors"
	"testing"
)

func TestBrowserCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		goos     string
		lookPath func(string) (string, error)
		want     string
		wantArgs []string
		wantErr  bool
	}{
		{
			name:     "darwin",
			goos:     "darwin",
			want:     "open",
			wantArgs: []string{"https://example.com"},
		},
		{
			name: "linux xdg-open",
			goos: "linux",
			lookPath: func(name string) (string, error) {
				if name == "xdg-open" {
					return "/usr/bin/xdg-open", nil
				}
				return "", errors.New("not found")
			},
			want:     "xdg-open",
			wantArgs: []string{"https://example.com"},
		},
		{
			name: "linux gio fallback",
			goos: "linux",
			lookPath: func(name string) (string, error) {
				if name == "gio" {
					return "/usr/bin/gio", nil
				}
				return "", errors.New("not found")
			},
			want:     "gio",
			wantArgs: []string{"open", "https://example.com"},
		},
		{
			name: "unsupported",
			goos: "plan9",
			lookPath: func(string) (string, error) {
				return "", errors.New("not found")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			lookPath := tt.lookPath
			if lookPath == nil {
				lookPath = func(string) (string, error) { return "", errors.New("not found") }
			}

			got, gotArgs, err := browserCommandWithLookPath(tt.goos, "https://example.com", lookPath)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
			if len(gotArgs) != len(tt.wantArgs) {
				t.Fatalf("args got %v want %v", gotArgs, tt.wantArgs)
			}
			for idx := range tt.wantArgs {
				if gotArgs[idx] != tt.wantArgs[idx] {
					t.Fatalf("args got %v want %v", gotArgs, tt.wantArgs)
				}
			}
		})
	}
}
