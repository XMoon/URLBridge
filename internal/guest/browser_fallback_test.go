package guest

import (
	"errors"
	"strings"
	"testing"
)

func TestRunBrowserFallbackDetectsOnceWhenBrowserPathEmptyAndPersistsSuccessfulCandidate(t *testing.T) {
	t.Parallel()

	detectCalls := 0
	var startCalls []string
	var savedPaths []string

	err := runBrowserFallback("https://example.com", browserFallbackRuntime{
		ConfigPath: "/tmp/urlbridge/config.yaml",
		DetectCandidates: func() []browserCandidate {
			detectCalls++
			return []browserCandidate{
				{Name: "Chrome", Path: `C:\Chrome\chrome.exe`},
				{Name: "Edge", Path: `C:\Edge\msedge.exe`},
			}
		},
		StartBrowserProcess: func(browserPath, targetURL string) error {
			startCalls = append(startCalls, browserPath)
			return nil
		},
		SaveBrowserPath: func(browserPath string) error {
			savedPaths = append(savedPaths, browserPath)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if detectCalls != 1 {
		t.Fatalf("detect calls got %d want 1", detectCalls)
	}
	if len(startCalls) != 1 || startCalls[0] != `C:\Chrome\chrome.exe` {
		t.Fatalf("start calls got %v want [C:\\Chrome\\chrome.exe]", startCalls)
	}
	if len(savedPaths) != 1 || savedPaths[0] != `C:\Chrome\chrome.exe` {
		t.Fatalf("saved paths got %v want [C:\\Chrome\\chrome.exe]", savedPaths)
	}
}

func TestRunBrowserFallbackUsesConfiguredPathWithoutDetection(t *testing.T) {
	t.Parallel()

	detectCalls := 0
	var startCalls []string
	saveCalls := 0
	confirmCalls := 0

	err := runBrowserFallback("https://example.com", browserFallbackRuntime{
		ConfiguredPath: ` C:\Custom\Browser.exe `,
		DetectCandidates: func() []browserCandidate {
			detectCalls++
			return []browserCandidate{{Name: "Chrome", Path: `C:\Chrome\chrome.exe`}}
		},
		ConfirmReplacement: func(failedPath string, candidate browserCandidate) bool {
			confirmCalls++
			return true
		},
		StartBrowserProcess: func(browserPath, targetURL string) error {
			startCalls = append(startCalls, browserPath)
			return nil
		},
		SaveBrowserPath: func(browserPath string) error {
			saveCalls++
			return nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if detectCalls != 0 {
		t.Fatalf("detect calls got %d want 0", detectCalls)
	}
	if confirmCalls != 0 {
		t.Fatalf("confirm calls got %d want 0", confirmCalls)
	}
	if saveCalls != 0 {
		t.Fatalf("save calls got %d want 0", saveCalls)
	}
	if len(startCalls) != 1 || startCalls[0] != `C:\Custom\Browser.exe` {
		t.Fatalf("start calls got %v want [C:\\Custom\\Browser.exe]", startCalls)
	}
}

func TestRunBrowserFallbackPromptsAndPersistsReplacementAfterConfiguredBrowserFails(t *testing.T) {
	t.Parallel()

	var confirmed []string
	var startCalls []string
	var savedPaths []string

	err := runBrowserFallback("https://example.com", browserFallbackRuntime{
		ConfiguredPath: `C:\Broken\Browser.exe`,
		ConfirmReplacement: func(failedPath string, candidate browserCandidate) bool {
			confirmed = append(confirmed, failedPath+"->"+candidate.Path)
			return true
		},
		DetectCandidates: func() []browserCandidate {
			return []browserCandidate{{Name: "Chrome", Path: `C:\Chrome\chrome.exe`}}
		},
		StartBrowserProcess: func(browserPath, targetURL string) error {
			startCalls = append(startCalls, browserPath)
			if browserPath == `C:\Broken\Browser.exe` {
				return errors.New("file missing")
			}
			return nil
		},
		SaveBrowserPath: func(browserPath string) error {
			savedPaths = append(savedPaths, browserPath)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(confirmed) != 1 || confirmed[0] != `C:\Broken\Browser.exe->C:\Chrome\chrome.exe` {
		t.Fatalf("confirmed got %v want [C:\\Broken\\Browser.exe->C:\\Chrome\\chrome.exe]", confirmed)
	}
	if len(startCalls) != 2 {
		t.Fatalf("start calls got %v want 2 entries", startCalls)
	}
	if startCalls[0] != `C:\Broken\Browser.exe` || startCalls[1] != `C:\Chrome\chrome.exe` {
		t.Fatalf("start calls got %v", startCalls)
	}
	if len(savedPaths) != 1 || savedPaths[0] != `C:\Chrome\chrome.exe` {
		t.Fatalf("saved paths got %v want [C:\\Chrome\\chrome.exe]", savedPaths)
	}
}

func TestRunBrowserFallbackSkipsDeclinedReplacementAndKeepsSearching(t *testing.T) {
	t.Parallel()

	var confirmed []string
	var startCalls []string
	var savedPaths []string

	err := runBrowserFallback("https://example.com", browserFallbackRuntime{
		ConfiguredPath: `C:\Broken\Browser.exe`,
		ConfirmReplacement: func(failedPath string, candidate browserCandidate) bool {
			confirmed = append(confirmed, candidate.Path)
			return candidate.Path == `C:\Edge\msedge.exe`
		},
		DetectCandidates: func() []browserCandidate {
			return []browserCandidate{
				{Name: "Chrome", Path: `C:\Chrome\chrome.exe`},
				{Name: "Edge", Path: `C:\Edge\msedge.exe`},
			}
		},
		StartBrowserProcess: func(browserPath, targetURL string) error {
			startCalls = append(startCalls, browserPath)
			if browserPath == `C:\Broken\Browser.exe` {
				return errors.New("file missing")
			}
			return nil
		},
		SaveBrowserPath: func(browserPath string) error {
			savedPaths = append(savedPaths, browserPath)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(confirmed) != 2 || confirmed[0] != `C:\Chrome\chrome.exe` || confirmed[1] != `C:\Edge\msedge.exe` {
		t.Fatalf("confirmed got %v", confirmed)
	}
	if len(startCalls) != 2 || startCalls[0] != `C:\Broken\Browser.exe` || startCalls[1] != `C:\Edge\msedge.exe` {
		t.Fatalf("start calls got %v want [broken edge]", startCalls)
	}
	if len(savedPaths) != 1 || savedPaths[0] != `C:\Edge\msedge.exe` {
		t.Fatalf("saved paths got %v want [C:\\Edge\\msedge.exe]", savedPaths)
	}
}

func TestRunBrowserFallbackExcludesFailedConfiguredPathFromRediscovery(t *testing.T) {
	t.Parallel()

	var confirmed []string
	var startCalls []string

	err := runBrowserFallback("https://example.com", browserFallbackRuntime{
		ConfiguredPath: `C:\Chrome\chrome.exe`,
		ConfirmReplacement: func(failedPath string, candidate browserCandidate) bool {
			confirmed = append(confirmed, candidate.Path)
			return true
		},
		DetectCandidates: func() []browserCandidate {
			return []browserCandidate{
				{Name: "Chrome", Path: ` c:\chrome\chrome.exe `},
				{Name: "Edge", Path: `C:\Edge\msedge.exe`},
			}
		},
		StartBrowserProcess: func(browserPath, targetURL string) error {
			startCalls = append(startCalls, browserPath)
			if browserPath == `C:\Chrome\chrome.exe` {
				return errors.New("file missing")
			}
			return nil
		},
		SaveBrowserPath: func(browserPath string) error {
			return nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(confirmed) != 1 || confirmed[0] != `C:\Edge\msedge.exe` {
		t.Fatalf("confirmed got %v want [C:\\Edge\\msedge.exe]", confirmed)
	}
	if len(startCalls) != 2 || startCalls[0] != `C:\Chrome\chrome.exe` || startCalls[1] != `C:\Edge\msedge.exe` {
		t.Fatalf("start calls got %v want [chrome edge]", startCalls)
	}
}

func TestRunBrowserFallbackNotifiesWhenSaveFailsAfterSuccessfulOpen(t *testing.T) {
	t.Parallel()

	var notices []string

	err := runBrowserFallback("https://example.com", browserFallbackRuntime{
		ConfigPath: "/tmp/urlbridge/config.yaml",
		Notify: func(message string) {
			notices = append(notices, message)
		},
		DetectCandidates: func() []browserCandidate {
			return []browserCandidate{{Name: "Chrome", Path: `C:\Chrome\chrome.exe`}}
		},
		StartBrowserProcess: func(browserPath, targetURL string) error {
			return nil
		},
		SaveBrowserPath: func(browserPath string) error {
			return errors.New("disk full")
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(notices) != 1 {
		t.Fatalf("notices got %v want 1 entry", notices)
	}
	if !strings.Contains(notices[0], "failed to save browser_path") {
		t.Fatalf("expected notice to mention save failure, got %q", notices[0])
	}
	if !strings.Contains(notices[0], "/tmp/urlbridge/config.yaml") {
		t.Fatalf("expected notice to mention config path, got %q", notices[0])
	}
	if !strings.Contains(notices[0], "disk full") {
		t.Fatalf("expected notice to mention save error, got %q", notices[0])
	}
}

func TestRunBrowserFallbackReturnsCombinedErrorWhenAllRepairOptionsAreDeclined(t *testing.T) {
	t.Parallel()

	err := runBrowserFallback("https://example.com", browserFallbackRuntime{
		ConfiguredPath: `C:\Broken\Browser.exe`,
		ConfirmReplacement: func(failedPath string, candidate browserCandidate) bool {
			return false
		},
		DetectCandidates: func() []browserCandidate {
			return []browserCandidate{{Name: "Chrome", Path: `C:\Chrome\chrome.exe`}}
		},
		StartBrowserProcess: func(browserPath, targetURL string) error {
			if browserPath == `C:\Broken\Browser.exe` {
				return errors.New("file missing")
			}
			return nil
		},
	})
	if err == nil {
		t.Fatalf("expected error")
	}

	text := err.Error()
	if !strings.Contains(text, `configured browser (C:\Broken\Browser.exe): file missing`) {
		t.Fatalf("expected error to mention configured browser failure, got %q", text)
	}
	if !strings.Contains(text, `replacement Chrome (C:\Chrome\chrome.exe): user declined`) {
		t.Fatalf("expected error to mention declined replacement, got %q", text)
	}
}
