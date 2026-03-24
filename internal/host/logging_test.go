package host

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFileConfigLogPathModes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		configYAML string
		assert     func(t *testing.T, cfg FileConfig)
	}{
		{
			name: "unset",
			configYAML: "listen_addr: '127.0.0.1:38495'\n" +
				"discovery: true\n",
			assert: func(t *testing.T, cfg FileConfig) {
				t.Helper()
				if cfg.LogPath.IsSet() {
					t.Fatalf("expected log_path to be unset")
				}
			},
		},
		{
			name: "explicit empty",
			configYAML: "listen_addr: '127.0.0.1:38495'\n" +
				"discovery: true\n" +
				"log_path: ''\n",
			assert: func(t *testing.T, cfg FileConfig) {
				t.Helper()
				if !cfg.LogPath.IsSet() {
					t.Fatalf("expected log_path to be set")
				}
				if cfg.LogPath.Value() != "" {
					t.Fatalf("log_path got %q want empty string", cfg.LogPath.Value())
				}
			},
		},
		{
			name: "custom path",
			configYAML: "listen_addr: '127.0.0.1:38495'\n" +
				"discovery: true\n" +
				"log_path: '/tmp/urlbridge/host.log'\n",
			assert: func(t *testing.T, cfg FileConfig) {
				t.Helper()
				if !cfg.LogPath.IsSet() {
					t.Fatalf("expected log_path to be set")
				}
				if cfg.LogPath.Value() != "/tmp/urlbridge/host.log" {
					t.Fatalf("log_path got %q want %q", cfg.LogPath.Value(), "/tmp/urlbridge/host.log")
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			configPath := filepath.Join(dir, "config.yaml")
			if err := os.WriteFile(configPath, []byte(tt.configYAML), 0o644); err != nil {
				t.Fatalf("write config: %v", err)
			}

			cfg, _, err := LoadFileConfig(configPath)
			if err != nil {
				t.Fatalf("load config: %v", err)
			}

			tt.assert(t, cfg)
		})
	}
}

func TestPlatformDefaultLogPathForLinux(t *testing.T) {
	t.Parallel()

	got := platformDefaultLogPathFor("linux", func(key string) string {
		if key == "XDG_STATE_HOME" {
			return "/tmp/state"
		}
		return ""
	}, "/home/user", "/tmp")

	want := filepath.Join("/tmp/state", "urlbridge", hostLogFileName)
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestPlatformDefaultLogPathForLinuxFallback(t *testing.T) {
	t.Parallel()

	got := platformDefaultLogPathFor("linux", func(string) string { return "" }, "", "/tmp")

	want := filepath.Join("/tmp", "urlbridge", hostLogFileName)
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestPlatformDefaultLogPathForDarwin(t *testing.T) {
	t.Parallel()

	got := platformDefaultLogPathFor("darwin", func(string) string { return "" }, "/Users/alice", "/tmp")

	want := filepath.Join("/Users/alice", "Library", "Logs", "URLBridge", hostLogFileName)
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestPlatformDefaultLogPathForWindows(t *testing.T) {
	t.Parallel()

	got := platformDefaultLogPathFor("windows", func(key string) string {
		if key == "LOCALAPPDATA" {
			return `C:\Users\alice\AppData\Local`
		}
		return ""
	}, "", `C:\Temp`)

	want := filepath.Join(`C:\Users\alice\AppData\Local`, "URLBridgeHost", hostLogFileName)
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
