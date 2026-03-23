package host

import (
	"fmt"
	"net"
	"sort"
	"strings"
)

func CandidateBaseURLs(listenAddr string) []string {
	host, port, err := net.SplitHostPort(listenAddr)
	if err != nil {
		return nil
	}

	if port == "" {
		return nil
	}

	if host != "" && host != "0.0.0.0" && host != "::" {
		return []string{formatBaseURL(host, port)}
	}

	var urls []string
	seen := map[string]struct{}{}
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP == nil {
				continue
			}

			ip := ipNet.IP.To4()
			if ip == nil {
				continue
			}

			url := formatBaseURL(ip.String(), port)
			if _, exists := seen[url]; exists {
				continue
			}
			seen[url] = struct{}{}
			urls = append(urls, url)
		}
	}

	sort.Strings(urls)
	return urls
}

func formatBaseURL(host, port string) string {
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	return fmt.Sprintf("http://%s", net.JoinHostPort(host, port))
}
