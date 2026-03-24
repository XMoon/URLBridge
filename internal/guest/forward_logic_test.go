package guest

import (
	"errors"
	"strings"
	"testing"
)

func TestOrderedBrowserCandidatesPrefersConfiguredPath(t *testing.T) {
	t.Parallel()

	got := orderedBrowserCandidates(` C:\Custom\Browser.exe `, []browserCandidate{
		{Name: "Chrome", Path: `C:\Program Files\Google\Chrome\Application\chrome.exe`},
		{Name: "Edge", Path: `C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`},
	})

	if len(got) != 1 {
		t.Fatalf("got %d candidates want 1", len(got))
	}
	if got[0].Name != "configured browser" {
		t.Fatalf("candidate name got %q want %q", got[0].Name, "configured browser")
	}
	if got[0].Path != `C:\Custom\Browser.exe` {
		t.Fatalf("candidate path got %q want %q", got[0].Path, `C:\Custom\Browser.exe`)
	}
}

func TestOrderedBrowserCandidatesKeepsChromeBeforeEdge(t *testing.T) {
	t.Parallel()

	got := orderedBrowserCandidates("", []browserCandidate{
		{Name: "Chrome", Path: `C:\Chrome\chrome.exe`},
		{Name: "Chrome", Path: `C:\Chrome\chrome.exe`},
		{Name: "Edge", Path: `C:\Edge\msedge.exe`},
	})

	if len(got) != 2 {
		t.Fatalf("got %d candidates want 2", len(got))
	}
	if got[0].Name != "Chrome" || got[0].Path != `C:\Chrome\chrome.exe` {
		t.Fatalf("first candidate got %#v", got[0])
	}
	if got[1].Name != "Edge" || got[1].Path != `C:\Edge\msedge.exe` {
		t.Fatalf("second candidate got %#v", got[1])
	}
}

func TestHandleHostForwardFailureNetworkErrorFallsBackSilently(t *testing.T) {
	t.Parallel()

	var notices []string
	fallbackCalls := 0
	err := handleHostForwardFailure(hostForwardFailure{
		Err: errors.New("send to host: context deadline exceeded"),
	}, func(message string) {
		notices = append(notices, message)
	}, func() error {
		fallbackCalls++
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notices) != 0 {
		t.Fatalf("got notices %v want none", notices)
	}
	if fallbackCalls != 1 {
		t.Fatalf("fallback calls got %d want 1", fallbackCalls)
	}
}

func TestHandleHostForwardFailureHostResponseShowsNoticeThenFallsBack(t *testing.T) {
	t.Parallel()

	failure := classifyHostForwardFailure(401, "401 Unauthorized", "invalid token", nil)

	var notices []string
	fallbackCalls := 0
	err := handleHostForwardFailure(failure, func(message string) {
		notices = append(notices, message)
	}, func() error {
		fallbackCalls++
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notices) != 1 || notices[0] != "invalid token" {
		t.Fatalf("got notices %v want [invalid token]", notices)
	}
	if fallbackCalls != 1 {
		t.Fatalf("fallback calls got %d want 1", fallbackCalls)
	}
}

func TestHandleHostForwardFailureReturnsCombinedErrorWhenFallbackFails(t *testing.T) {
	t.Parallel()

	failure := classifyHostForwardFailure(500, "500 Internal Server Error", "host exploded", nil)

	err := handleHostForwardFailure(failure, nil, func() error {
		return errors.New("browser not found")
	})
	if err == nil {
		t.Fatalf("expected error")
	}

	text := err.Error()
	if !strings.Contains(text, "host exploded") {
		t.Fatalf("expected combined error to mention host failure, got %q", text)
	}
	if !strings.Contains(text, "local browser fallback failed: browser not found") {
		t.Fatalf("expected combined error to mention local fallback failure, got %q", text)
	}
}
