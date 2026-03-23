package host

import (
	"path/filepath"
	"testing"
)

func TestPlatformDefaultConfigPathsForLinux(t *testing.T) {
	t.Parallel()

	got := platformDefaultConfigPathsFor("linux", func(key string) string {
		if key == "XDG_CONFIG_HOME" {
			return "/tmp/xdg"
		}
		return ""
	}, "/home/user")

	want := []string{
		filepath.Join("/tmp/xdg", "urlbridge", ConfigFileName),
		filepath.Join(string(filepath.Separator), "etc", "urlbridge", ConfigFileName),
	}

	if len(got) != len(want) {
		t.Fatalf("got %d paths want %d: %#v", len(got), len(want), got)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("path %d got %q want %q", i, got[i], want[i])
		}
	}
}

func TestPlatformDefaultConfigPathsForDarwin(t *testing.T) {
	t.Parallel()

	got := platformDefaultConfigPathsFor("darwin", func(string) string { return "" }, "/Users/alice")

	want := []string{
		filepath.Join("/Users/alice", "Library", "Application Support", "URLBridge", ConfigFileName),
		filepath.Join(string(filepath.Separator), "etc", "urlbridge", ConfigFileName),
	}

	if len(got) != len(want) {
		t.Fatalf("got %d paths want %d: %#v", len(got), len(want), got)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("path %d got %q want %q", i, got[i], want[i])
		}
	}
}
