//go:build linux

package guest

import (
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/xmoon/urlbridge/internal/bridge"
)

const (
	linuxRouteFlagUp      = 0x1
	linuxRouteFlagGateway = 0x2
)

func defaultGatewayURLs() []string {
	ips, err := defaultGatewayIPv4s()
	if err != nil {
		return nil
	}

	return gatewayBaseURLs(ips)
}

func defaultGatewayIPv4s() ([]net.IP, error) {
	data, err := os.ReadFile("/proc/net/route")
	if err != nil {
		return nil, err
	}

	return parseLinuxRouteGateways(string(data))
}

func parseLinuxRouteGateways(data string) ([]net.IP, error) {
	var ips []net.IP

	for lineNumber, line := range strings.Split(data, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if lineNumber == 0 && strings.EqualFold(fields[0], "Iface") {
			continue
		}
		if len(fields) < 4 {
			continue
		}
		if fields[1] != "00000000" {
			continue
		}

		flags, err := strconv.ParseUint(fields[3], 16, 32)
		if err != nil {
			return nil, fmt.Errorf("parse route flags %q: %w", fields[3], err)
		}
		if flags&linuxRouteFlagUp == 0 || flags&linuxRouteFlagGateway == 0 {
			continue
		}

		gateway, err := parseLinuxRouteIPv4(fields[2])
		if err != nil {
			return nil, err
		}
		if gateway != nil {
			ips = append(ips, gateway)
		}
	}

	return ips, nil
}

func parseLinuxRouteIPv4(value string) (net.IP, error) {
	hexValue, err := strconv.ParseUint(value, 16, 32)
	if err != nil {
		return nil, fmt.Errorf("parse route gateway %q: %w", value, err)
	}
	if hexValue == 0 {
		return nil, nil
	}

	return net.IPv4(byte(hexValue), byte(hexValue>>8), byte(hexValue>>16), byte(hexValue>>24)).To4(), nil
}

func gatewayBaseURLs(ips []net.IP) []string {
	normalized := normalizeIPv4GatewayStrings(ips)
	urls := make([]string, 0, len(normalized))
	for _, ip := range normalized {
		urls = append(urls, formatCandidateBaseURL(ip, bridge.DefaultPort))
	}
	return urls
}

func normalizeIPv4GatewayStrings(ips []net.IP) []string {
	seen := make(map[string]struct{}, len(ips))
	normalized := make([]string, 0, len(ips))

	for _, ip := range ips {
		ipv4 := ip.To4()
		if ipv4 == nil {
			continue
		}

		text := ipv4.String()
		if _, exists := seen[text]; exists {
			continue
		}
		seen[text] = struct{}{}
		normalized = append(normalized, text)
	}

	sort.Strings(normalized)
	return normalized
}
