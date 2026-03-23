package bridge

import (
	"path/filepath"
	"testing"
)

func TestConfigCandidatePathsFromOrderAndDedup(t *testing.T) {
	t.Parallel()

	got, err := ConfigCandidatePathsFrom(
		filepath.Join(string(filepath.Separator), "work", "urlbridge"),
		filepath.Join(string(filepath.Separator), "opt", "urlbridge", "urlbridge-host"),
		"config.yaml",
		[]string{
			filepath.Join(string(filepath.Separator), "home", "user", ".config", "urlbridge", "config.yaml"),
			filepath.Join(string(filepath.Separator), "opt", "urlbridge", "config.yaml"),
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{
		filepath.Join(string(filepath.Separator), "work", "urlbridge", "config.yaml"),
		filepath.Join(string(filepath.Separator), "opt", "urlbridge", "config.yaml"),
		filepath.Join(string(filepath.Separator), "home", "user", ".config", "urlbridge", "config.yaml"),
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
