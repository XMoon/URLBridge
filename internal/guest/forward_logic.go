package guest

import (
	"fmt"
	"strings"
)

type hostForwardFailure struct {
	Err          error
	PopupMessage string
}

type browserCandidate struct {
	Name string
	Path string
}

func classifyHostForwardFailure(statusCode int, statusText, responseMessage string, transportErr error) hostForwardFailure {
	if transportErr != nil {
		return hostForwardFailure{Err: transportErr}
	}

	message := strings.TrimSpace(responseMessage)
	if message == "" && statusCode != 0 && statusCode != 200 {
		message = strings.TrimSpace(statusText)
	}
	if message == "" {
		message = "host did not accept the request"
	}

	return hostForwardFailure{
		Err:          fmt.Errorf("host refused request: %s", message),
		PopupMessage: message,
	}
}

func handleHostForwardFailure(failure hostForwardFailure, notify func(string), fallback func() error) error {
	if failure.Err == nil {
		return nil
	}

	if failure.PopupMessage != "" && notify != nil {
		notify(failure.PopupMessage)
	}

	if fallback == nil {
		return failure.Err
	}

	if err := fallback(); err != nil {
		return fmt.Errorf("%v; local browser fallback failed: %w", failure.Err, err)
	}

	return nil
}

func orderedBrowserCandidates(configuredPath string, detected []browserCandidate) []browserCandidate {
	trimmed := strings.TrimSpace(configuredPath)
	if trimmed != "" {
		return []browserCandidate{{
			Name: "configured browser",
			Path: trimmed,
		}}
	}

	seen := make(map[string]struct{}, len(detected))
	ordered := make([]browserCandidate, 0, len(detected))
	for _, candidate := range detected {
		path := strings.TrimSpace(candidate.Path)
		if path == "" {
			continue
		}

		key := strings.ToLower(path)
		if _, exists := seen[key]; exists {
			continue
		}

		seen[key] = struct{}{}
		ordered = append(ordered, browserCandidate{
			Name: candidate.Name,
			Path: path,
		})
	}

	return ordered
}
