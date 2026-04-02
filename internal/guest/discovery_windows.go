//go:build windows

package guest

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"syscall"
	"time"

	"github.com/xmoon/urlbridge/internal/bridge"
)

type DiscoveryCandidate struct {
	BaseURL       string
	Source        string
	HostName      string
	TokenRequired bool
}

type ResolvedInstall struct {
	BaseURL  string
	Token    string
	Resolved string
}

func ResolveInstallTarget(hostURL, token string, timeout time.Duration) (ResolvedInstall, error) {
	trimmedHostURL := strings.TrimSpace(hostURL)
	trimmedToken := strings.TrimSpace(token)

	if trimmedHostURL != "" {
		resolvedBaseURL, resolvedToken, err := resolveWithKnownHost(trimmedHostURL, trimmedToken)
		if err != nil {
			return ResolvedInstall{}, err
		}

		return ResolvedInstall{
			BaseURL:  resolvedBaseURL,
			Token:    resolvedToken,
			Resolved: "configured host URL",
		}, nil
	}

	candidates, err := DiscoverHosts(timeout)
	if err != nil {
		return ResolvedInstall{}, err
	}

	for _, candidate := range candidates {
		if candidate.TokenRequired && trimmedToken == "" {
			continue
		}

		return ResolvedInstall{
			BaseURL:  candidate.BaseURL,
			Token:    trimmedToken,
			Resolved: fmt.Sprintf("auto-discovered host via %s", candidate.Source),
		}, nil
	}

	return ResolvedInstall{}, fmt.Errorf("discovered URL Bridge hosts all require --token")
}

func resolveWithKnownHost(baseURL, token string) (string, string, error) {
	normalizedBaseURL, err := normalizeBaseURL(baseURL)
	if err != nil {
		return "", "", err
	}

	return normalizedBaseURL, token, nil
}

func DiscoverHosts(timeout time.Duration) ([]DiscoveryCandidate, error) {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	seen := map[string]DiscoveryCandidate{}
	var ordered []string

	record := func(candidate DiscoveryCandidate) {
		if candidate.BaseURL == "" {
			return
		}
		if _, exists := seen[candidate.BaseURL]; exists {
			return
		}
		seen[candidate.BaseURL] = candidate
		ordered = append(ordered, candidate.BaseURL)
	}

	udpTimeout := timeout / 2
	if udpTimeout < 1500*time.Millisecond {
		udpTimeout = 1500 * time.Millisecond
	}
	if udpTimeout > timeout {
		udpTimeout = timeout
	}

	for _, candidate := range discoverViaUDP(udpTimeout) {
		record(candidate)
	}

	probeTimeout := timeout / 4
	if probeTimeout < 1200*time.Millisecond {
		probeTimeout = 1200 * time.Millisecond
	}

	for _, candidate := range discoverViaProbes(probeTimeout) {
		record(candidate)
	}

	if len(ordered) == 0 {
		return nil, fmt.Errorf("no URL Bridge host discovered; try --host-url or make sure the host service is reachable")
	}

	results := make([]DiscoveryCandidate, 0, len(ordered))
	for _, key := range ordered {
		results = append(results, seen[key])
	}

	return results, nil
}

func discoverViaUDP(timeout time.Duration) []DiscoveryCandidate {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return nil
	}
	defer conn.Close()

	if err := enableUDPBroadcast(conn); err != nil {
		return nil
	}

	payload, err := json.Marshal(bridge.DiscoveryRequest{App: bridge.AppName})
	if err != nil {
		return nil
	}

	if _, err := conn.WriteToUDP(payload, &net.UDPAddr{IP: net.IPv4bcast, Port: bridge.DiscoveryPort}); err != nil {
		return nil
	}

	var results []DiscoveryCandidate
	buffer := make([]byte, 64*1024)
	deadline := time.Now().Add(timeout)

	for {
		if err := conn.SetReadDeadline(deadline); err != nil {
			break
		}

		n, addr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			break
		}

		var resp bridge.DiscoveryResponse
		if err := json.Unmarshal(buffer[:n], &resp); err != nil {
			continue
		}
		if !resp.OK || resp.Name != bridge.AppName || resp.HTTPPort <= 0 {
			continue
		}

		baseURL := formatCandidateBaseURL(addr.IP.String(), resp.HTTPPort)
		candidate, err := probeCandidate(baseURL, timeout)
		if err != nil {
			for _, alternate := range resp.CandidateBaseURLs {
				candidate, err = probeCandidate(alternate, timeout)
				if err == nil {
					candidate.Source = "udp-broadcast"
					if candidate.HostName == "" {
						candidate.HostName = resp.HostName
					}
					results = append(results, candidate)
					break
				}
			}
			continue
		}

		candidate.Source = "udp-broadcast"
		if candidate.HostName == "" {
			candidate.HostName = resp.HostName
		}
		results = append(results, candidate)
	}

	return results
}

func discoverViaProbes(timeout time.Duration) []DiscoveryCandidate {
	var urls []probeTarget
	for _, baseURL := range commonNATHostURLs() {
		urls = append(urls, probeTarget{BaseURL: baseURL, Source: "vm-nat"})
	}
	for _, baseURL := range defaultGatewayURLs() {
		urls = append(urls, probeTarget{BaseURL: baseURL, Source: "default-gateway"})
	}

	seen := map[string]struct{}{}
	var results []DiscoveryCandidate
	for _, target := range urls {
		if _, exists := seen[target.BaseURL]; exists {
			continue
		}
		seen[target.BaseURL] = struct{}{}

		candidate, err := probeCandidate(target.BaseURL, timeout)
		if err != nil {
			continue
		}
		candidate.Source = target.Source
		results = append(results, candidate)
	}

	return results
}

type probeTarget struct {
	BaseURL string
	Source  string
}

func probeCandidate(baseURL string, timeout time.Duration) (DiscoveryCandidate, error) {
	target, err := endpointURL(baseURL, "/healthz")
	if err != nil {
		return DiscoveryCandidate{}, err
	}

	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return DiscoveryCandidate{}, fmt.Errorf("create probe request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return DiscoveryCandidate{}, fmt.Errorf("probe host: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return DiscoveryCandidate{}, fmt.Errorf("probe host returned %s", resp.Status)
	}

	var health bridge.HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return DiscoveryCandidate{}, fmt.Errorf("decode health response: %w", err)
	}

	if !health.OK || health.Name != bridge.AppName {
		return DiscoveryCandidate{}, fmt.Errorf("unexpected health response")
	}

	return DiscoveryCandidate{
		BaseURL:       baseURL,
		HostName:      health.HostName,
		TokenRequired: health.TokenRequired,
	}, nil
}

func commonNATHostURLs() []string {
	return []string{
		formatCandidateBaseURL("10.0.2.2", bridge.DefaultPort),
		formatCandidateBaseURL("10.0.3.2", bridge.DefaultPort),
		formatCandidateBaseURL("192.168.56.1", bridge.DefaultPort),
	}
}

func enableUDPBroadcast(conn *net.UDPConn) error {
	raw, err := conn.SyscallConn()
	if err != nil {
		return err
	}

	var sockErr error
	if err := raw.Control(func(fd uintptr) {
		sockErr = syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1)
	}); err != nil {
		return err
	}

	return sockErr
}

func formatCandidateBaseURL(host string, port int) string {
	return fmt.Sprintf("http://%s", net.JoinHostPort(host, fmt.Sprintf("%d", port)))
}

func (c DiscoveryCandidate) HostNameOrFallback() string {
	if strings.TrimSpace(c.HostName) != "" {
		return c.HostName
	}
	return c.BaseURL
}
