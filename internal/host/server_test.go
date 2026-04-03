package host

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xmoon/urlbridge/internal/bridge"
)

func TestNewHandlerProbeRequiresValidToken(t *testing.T) {
	t.Parallel()

	handler, err := newHandler(Config{
		Token:  "secret",
		Logger: log.New(io.Discard, "", 0),
	})
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("status got %d want %d", resp.Code, http.StatusUnauthorized)
	}

	var probe bridge.ProbeResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &probe); err != nil {
		t.Fatalf("decode probe response: %v", err)
	}
	if probe.Name != bridge.AppName {
		t.Fatalf("probe name got %q want %q", probe.Name, bridge.AppName)
	}
	if !probe.TokenRequired {
		t.Fatalf("expected token_required=true")
	}
	if probe.Authenticated {
		t.Fatalf("expected authenticated=false")
	}
	if probe.Message != "invalid token" {
		t.Fatalf("message got %q want %q", probe.Message, "invalid token")
	}
}

func TestNewHandlerProbeAcceptsValidToken(t *testing.T) {
	t.Parallel()

	handler, err := newHandler(Config{
		Token:  "secret",
		Logger: log.New(io.Discard, "", 0),
	})
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.Header.Set(tokenHeader, "secret")
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status got %d want %d", resp.Code, http.StatusOK)
	}

	var probe bridge.ProbeResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &probe); err != nil {
		t.Fatalf("decode probe response: %v", err)
	}
	if !probe.OK || !probe.Authenticated {
		t.Fatalf("expected authenticated probe response, got %#v", probe)
	}
}

func TestLoggableURLRedactsSensitivePartsByDefault(t *testing.T) {
	t.Parallel()

	got := loggableURL("https://user:pass@example.com/path?q=token#frag", false)
	if got != "https://example.com/path" {
		t.Fatalf("redacted url got %q want %q", got, "https://example.com/path")
	}
}

func TestLoggableURLPreservesFullURLWhenEnabled(t *testing.T) {
	t.Parallel()

	raw := "https://user:pass@example.com/path?q=token#frag"
	if got := loggableURL(raw, true); got != raw {
		t.Fatalf("full url got %q want %q", got, raw)
	}
}
