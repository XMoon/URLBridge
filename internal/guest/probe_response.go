package guest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/xmoon/urlbridge/internal/bridge"
)

func parseHealthResponse(statusCode int, body []byte) (bridge.HealthResponse, error) {
	if statusCode != http.StatusOK {
		return bridge.HealthResponse{}, fmt.Errorf("probe host returned %s", http.StatusText(statusCode))
	}

	var health bridge.HealthResponse
	if err := json.Unmarshal(body, &health); err != nil {
		return bridge.HealthResponse{}, fmt.Errorf("unexpected health response")
	}
	if !health.OK || health.Name != bridge.AppName {
		return bridge.HealthResponse{}, fmt.Errorf("unexpected health response")
	}

	return health, nil
}

func parseProbeResponse(statusCode int, body []byte) error {
	var probe bridge.ProbeResponse
	if err := json.Unmarshal(body, &probe); err != nil {
		return fmt.Errorf("host is reachable but is not a URL Bridge service")
	}
	if probe.Name != bridge.AppName {
		return fmt.Errorf("host is reachable but is not a URL Bridge service")
	}

	switch statusCode {
	case http.StatusOK:
		if probe.OK {
			return nil
		}
		return fmt.Errorf("host probe failed: %s", probeMessage(probe.Message, "unexpected probe failure"))
	case http.StatusUnauthorized:
		return fmt.Errorf("host rejected authentication: %s", probeMessage(probe.Message, "invalid token"))
	default:
		return fmt.Errorf("host probe failed: %s", probeMessage(probe.Message, http.StatusText(statusCode)))
	}
}

func probeMessage(message, fallback string) string {
	trimmed := strings.TrimSpace(message)
	if trimmed != "" {
		return trimmed
	}
	if strings.TrimSpace(fallback) != "" {
		return fallback
	}
	return "unknown error"
}
