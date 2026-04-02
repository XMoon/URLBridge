//go:build windows

package guest

import (
	"net"
	"testing"

	"github.com/xmoon/urlbridge/internal/bridge"
)

func TestNormalizeIPv4GatewayStrings(t *testing.T) {
	t.Parallel()

	got := normalizeIPv4GatewayStrings([]net.IP{
		net.ParseIP("192.168.56.1"),
		net.ParseIP("10.0.2.2"),
		net.ParseIP("192.168.56.1"),
		net.ParseIP("fe80::1"),
		nil,
	})

	want := []string{"10.0.2.2", "192.168.56.1"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for idx := range want {
		if got[idx] != want[idx] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestGatewayBaseURLs(t *testing.T) {
	t.Parallel()

	got := gatewayBaseURLs([]net.IP{
		net.ParseIP("10.0.3.2"),
		net.ParseIP("10.0.2.2"),
	})

	want := []string{
		formatCandidateBaseURL("10.0.2.2", bridge.DefaultPort),
		formatCandidateBaseURL("10.0.3.2", bridge.DefaultPort),
	}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for idx := range want {
		if got[idx] != want[idx] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}
