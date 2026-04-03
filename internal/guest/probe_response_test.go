package guest

import (
	"testing"

	"github.com/xmoon/urlbridge/internal/bridge"
)

func TestParseHealthResponseAcceptsURLBridgeHealthPayload(t *testing.T) {
	t.Parallel()

	body := []byte(bridge.MustJSON(bridge.HealthResponse{
		OK:            true,
		Name:          bridge.AppName,
		TokenRequired: true,
		HostName:      "host-a",
	}))

	health, err := parseHealthResponse(200, body)
	if err != nil {
		t.Fatalf("parse health response: %v", err)
	}
	if health.HostName != "host-a" {
		t.Fatalf("host name got %q want %q", health.HostName, "host-a")
	}
	if !health.TokenRequired {
		t.Fatalf("expected token_required=true")
	}
}

func TestParseProbeResponseAcceptsHealthyProbe(t *testing.T) {
	t.Parallel()

	body := []byte(bridge.MustJSON(bridge.ProbeResponse{
		OK:            true,
		Name:          bridge.AppName,
		Authenticated: true,
	}))

	if err := parseProbeResponse(200, body); err != nil {
		t.Fatalf("parse probe response: %v", err)
	}
}

func TestParseProbeResponseDetectsAuthenticationFailure(t *testing.T) {
	t.Parallel()

	body := []byte(bridge.MustJSON(bridge.ProbeResponse{
		OK:            false,
		Name:          bridge.AppName,
		TokenRequired: true,
		Message:       "invalid token",
	}))

	err := parseProbeResponse(401, body)
	if err == nil {
		t.Fatalf("expected error")
	}
	if err.Error() != "host rejected authentication: invalid token" {
		t.Fatalf("error got %q", err.Error())
	}
}

func TestParseProbeResponseRejectsUnexpectedService(t *testing.T) {
	t.Parallel()

	err := parseProbeResponse(200, []byte(`{"ok":true,"name":"something-else"}`))
	if err == nil {
		t.Fatalf("expected error")
	}
	if err.Error() != "host is reachable but is not a URL Bridge service" {
		t.Fatalf("error got %q", err.Error())
	}
}
