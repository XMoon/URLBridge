//go:build linux

package guest

import (
	"strings"
	"testing"
)

func TestDesktopFileContentUsesFieldCodeAndEscapedPaths(t *testing.T) {
	t.Parallel()

	got := desktopFileContent(`/home/alice/URL Bridge/urlbridge-browser`, `/home/alice/config/urlbridge-guest/config.yaml`)

	if !strings.Contains(got, `Exec=/home/alice/URL\ Bridge/urlbridge-browser --config /home/alice/config/urlbridge-guest/config.yaml %u`) {
		t.Fatalf("desktop entry did not escape exec line correctly:\n%s", got)
	}
	if !strings.Contains(got, "MimeType=x-scheme-handler/http;x-scheme-handler/https;") {
		t.Fatalf("desktop entry did not declare URL scheme handlers:\n%s", got)
	}
}

func TestDesktopExecFieldEscapesSpecialCharacters(t *testing.T) {
	t.Parallel()

	got := desktopExecField(`/tmp/a "b" $c %d`)
	want := `/tmp/a\ \"b\"\ \$c\ %%d`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestUpsertMimeAppsDefaultsAddsDefaultApplicationsSection(t *testing.T) {
	t.Parallel()

	got := string(upsertMimeAppsDefaults([]byte("[Added Associations]\ntext/plain=vim.desktop;\n"), DesktopFileName))

	for _, want := range []string{
		"[Added Associations]",
		"text/plain=vim.desktop;",
		"[Default Applications]",
		"x-scheme-handler/http=urlbridge-browser.desktop",
		"x-scheme-handler/https=urlbridge-browser.desktop",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("updated mimeapps list missing %q:\n%s", want, got)
		}
	}
}

func TestUpsertMimeAppsDefaultsUpdatesExistingHandlers(t *testing.T) {
	t.Parallel()

	input := []byte(`[Default Applications]
x-scheme-handler/http=firefox.desktop
x-scheme-handler/https=firefox.desktop
text/plain=vim.desktop
`)
	got := string(upsertMimeAppsDefaults(input, DesktopFileName))

	for _, want := range []string{
		"x-scheme-handler/http=urlbridge-browser.desktop",
		"x-scheme-handler/https=urlbridge-browser.desktop",
		"text/plain=vim.desktop",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("updated mimeapps list missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "firefox.desktop") {
		t.Fatalf("old handler was not replaced:\n%s", got)
	}
}

func TestRemoveMimeAppsDefaultsLeavesUserChangesAlone(t *testing.T) {
	t.Parallel()

	input := []byte(`[Default Applications]
x-scheme-handler/http=urlbridge-browser.desktop
x-scheme-handler/https=firefox.desktop
text/plain=vim.desktop
`)
	got := string(removeMimeAppsDefaults(input, DesktopFileName))

	if strings.Contains(got, "x-scheme-handler/http=urlbridge-browser.desktop") {
		t.Fatalf("URL Bridge handler was not removed:\n%s", got)
	}
	if !strings.Contains(got, "x-scheme-handler/https=firefox.desktop") {
		t.Fatalf("non-URL Bridge handler was changed:\n%s", got)
	}
	if !strings.Contains(got, "text/plain=vim.desktop") {
		t.Fatalf("unrelated handler was changed:\n%s", got)
	}
}

func TestMimeAppsDefaultReturnsFirstDefaultApplication(t *testing.T) {
	t.Parallel()

	input := []byte(`[Added Associations]
x-scheme-handler/https=ignored.desktop;

[Default Applications]
x-scheme-handler/https=urlbridge-browser.desktop;firefox.desktop;
`)

	got := mimeAppsDefault(input, "x-scheme-handler/https")
	if got != "urlbridge-browser.desktop" {
		t.Fatalf("got %q want urlbridge-browser.desktop", got)
	}
}
