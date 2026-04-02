package guest

import (
	"fmt"
	"strings"
)

type browserFallbackRuntime struct {
	ConfiguredPath      string
	ConfigPath          string
	Notify              func(string)
	ConfirmReplacement  func(string, browserCandidate) bool
	DetectCandidates    func() []browserCandidate
	StartBrowserProcess func(string, string) error
	SaveBrowserPath     func(string) error
}

func runBrowserFallback(targetURL string, runtime browserFallbackRuntime) error {
	if runtime.StartBrowserProcess == nil {
		return fmt.Errorf("open local browser: browser launcher is not configured")
	}

	configuredPath := strings.TrimSpace(runtime.ConfiguredPath)
	if configuredPath == "" {
		return openDetectedBrowser(targetURL, runtime)
	}

	if err := runtime.StartBrowserProcess(configuredPath, targetURL); err == nil {
		return nil
	} else {
		failures := []string{fmt.Sprintf("configured browser (%s): %v", configuredPath, err)}
		candidates := excludeBrowserCandidatePath(detectedBrowserCandidates(runtime.DetectCandidates), configuredPath)
		if len(candidates) == 0 {
			failures = append(failures, "no alternative local browser found; configure browser_path or install Chrome/Edge")
			return openLocalBrowserError(failures)
		}

		for _, candidate := range candidates {
			if runtime.ConfirmReplacement != nil && !runtime.ConfirmReplacement(configuredPath, candidate) {
				failures = append(failures, fmt.Sprintf("replacement %s (%s): user declined", candidate.Name, candidate.Path))
				continue
			}

			if err := runtime.StartBrowserProcess(candidate.Path, targetURL); err != nil {
				failures = append(failures, fmt.Sprintf("replacement %s (%s): %v", candidate.Name, candidate.Path, err))
				continue
			}

			runtime.notifyBrowserPathSaveFailure(candidate, runtime.persistBrowserPath(candidate.Path))
			return nil
		}

		return openLocalBrowserError(failures)
	}
}

func openDetectedBrowser(targetURL string, runtime browserFallbackRuntime) error {
	candidates := detectedBrowserCandidates(runtime.DetectCandidates)
	if len(candidates) == 0 {
		return fmt.Errorf("no local browser found; configure browser_path or install Chrome/Edge")
	}

	var failures []string
	for _, candidate := range candidates {
		if err := runtime.StartBrowserProcess(candidate.Path, targetURL); err != nil {
			failures = append(failures, fmt.Sprintf("%s (%s): %v", candidate.Name, candidate.Path, err))
			continue
		}

		runtime.notifyBrowserPathSaveFailure(candidate, runtime.persistBrowserPath(candidate.Path))
		return nil
	}

	return openLocalBrowserError(failures)
}

func detectedBrowserCandidates(detect func() []browserCandidate) []browserCandidate {
	if detect == nil {
		return nil
	}
	return orderedBrowserCandidates("", detect())
}

func (runtime browserFallbackRuntime) persistBrowserPath(browserPath string) error {
	if runtime.SaveBrowserPath == nil {
		return nil
	}
	return runtime.SaveBrowserPath(strings.TrimSpace(browserPath))
}

func (runtime browserFallbackRuntime) notifyBrowserPathSaveFailure(candidate browserCandidate, err error) {
	if err == nil || runtime.Notify == nil {
		return
	}

	if runtime.ConfigPath != "" {
		runtime.Notify(fmt.Sprintf("Opened %s locally, but failed to save browser_path to %s: %v", candidate.Name, runtime.ConfigPath, err))
		return
	}

	runtime.Notify(fmt.Sprintf("Opened %s locally, but failed to save browser_path: %v", candidate.Name, err))
}

func openLocalBrowserError(failures []string) error {
	return fmt.Errorf("open local browser: %s", strings.Join(failures, "; "))
}
