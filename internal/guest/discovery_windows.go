//go:build windows

package guest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
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
	start := time.Now()
	deadline := start.Add(timeout)

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

	for _, candidate := range resolveUDPDiscoveryTargets(deadline, discoverViaUDP(udpDiscoveryDeadline(start, timeout))) {
		record(candidate)
	}

	for _, candidate := range discoverViaProbes(deadline) {
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

type udpDiscoveryTarget struct {
	BaseURLs []string
	HostName string
}

func discoverViaUDP(deadline time.Time) []udpDiscoveryTarget {
	if remainingUntil(deadline) <= 0 {
		return nil
	}

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

	var results []udpDiscoveryTarget
	buffer := make([]byte, 64*1024)

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

		results = append(results, udpDiscoveryTarget{
			BaseURLs: orderedCandidateBaseURLs(
				formatCandidateBaseURL(addr.IP.String(), resp.HTTPPort),
				resp.CandidateBaseURLs,
			),
			HostName: resp.HostName,
		})
	}

	return results
}

func resolveUDPDiscoveryTargets(deadline time.Time, targets []udpDiscoveryTarget) []DiscoveryCandidate {
	if len(targets) == 0 || remainingUntil(deadline) <= 0 {
		return nil
	}

	results := make([]probeResult, len(targets))
	var wg sync.WaitGroup
	for idx, target := range targets {
		wg.Add(1)
		go func(idx int, target udpDiscoveryTarget) {
			defer wg.Done()

			ctx, cancel := context.WithDeadline(context.Background(), deadline)
			defer cancel()

			for _, baseURL := range target.BaseURLs {
				candidate, err := probeCandidate(ctx, baseURL)
				if err == nil {
					candidate.Source = "udp-broadcast"
					if candidate.HostName == "" {
						candidate.HostName = target.HostName
					}
					results[idx] = probeResult{Candidate: candidate, OK: true}
					return
				}
				if ctx.Err() != nil {
					return
				}
			}
		}(idx, target)
	}
	wg.Wait()

	return orderedProbeResults(results)
}

func discoverViaProbes(deadline time.Time) []DiscoveryCandidate {
	var urls []probeTarget
	for _, baseURL := range commonNATHostURLs() {
		urls = append(urls, probeTarget{BaseURL: baseURL, Source: "vm-nat"})
	}
	for _, baseURL := range defaultGatewayURLs() {
		urls = append(urls, probeTarget{BaseURL: baseURL, Source: "default-gateway"})
	}

	return probeTargetsWithinDeadline(deadline, dedupeProbeTargets(urls))
}

type probeTarget struct {
	BaseURL string
	Source  string
}

type probeResult struct {
	Candidate DiscoveryCandidate
	OK        bool
}

func probeTargetsWithinDeadline(deadline time.Time, targets []probeTarget) []DiscoveryCandidate {
	if len(targets) == 0 || remainingUntil(deadline) <= 0 {
		return nil
	}

	results := make([]probeResult, len(targets))
	var wg sync.WaitGroup
	for idx, target := range targets {
		wg.Add(1)
		go func(idx int, target probeTarget) {
			defer wg.Done()

			ctx, cancel := context.WithDeadline(context.Background(), deadline)
			defer cancel()

			candidate, err := probeCandidate(ctx, target.BaseURL)
			if err != nil {
				return
			}
			candidate.Source = target.Source
			results[idx] = probeResult{Candidate: candidate, OK: true}
		}(idx, target)
	}
	wg.Wait()

	return orderedProbeResults(results)
}

func orderedProbeResults(results []probeResult) []DiscoveryCandidate {
	ordered := make([]DiscoveryCandidate, 0, len(results))
	for _, result := range results {
		if !result.OK {
			continue
		}
		ordered = append(ordered, result.Candidate)
	}
	return ordered
}

func dedupeProbeTargets(targets []probeTarget) []probeTarget {
	seen := make(map[string]struct{}, len(targets))
	ordered := make([]probeTarget, 0, len(targets))

	for _, target := range targets {
		if _, exists := seen[target.BaseURL]; exists {
			continue
		}
		seen[target.BaseURL] = struct{}{}
		ordered = append(ordered, target)
	}

	return ordered
}

func orderedCandidateBaseURLs(primary string, alternates []string) []string {
	ordered := make([]string, 0, len(alternates)+1)
	seen := make(map[string]struct{}, len(alternates)+1)

	for _, candidate := range append([]string{primary}, alternates...) {
		trimmed := strings.TrimSpace(candidate)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		ordered = append(ordered, trimmed)
	}

	return ordered
}

func udpDiscoveryDeadline(start time.Time, timeout time.Duration) time.Time {
	budget := timeout / 2
	if budget > 1500*time.Millisecond {
		budget = 1500 * time.Millisecond
	}
	if budget <= 0 || budget > timeout {
		budget = timeout
	}
	return start.Add(budget)
}

func remainingUntil(deadline time.Time) time.Duration {
	return time.Until(deadline)
}

func probeCandidate(ctx context.Context, baseURL string) (DiscoveryCandidate, error) {
	target, err := endpointURL(baseURL, "/healthz")
	if err != nil {
		return DiscoveryCandidate{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return DiscoveryCandidate{}, fmt.Errorf("create probe request: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return DiscoveryCandidate{}, fmt.Errorf("probe host: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 32*1024))
	if err != nil {
		return DiscoveryCandidate{}, fmt.Errorf("read health response: %w", err)
	}

	health, err := parseHealthResponse(resp.StatusCode, body)
	if err != nil {
		return DiscoveryCandidate{}, err
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
