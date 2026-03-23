package bridge

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

const (
	DefaultPort    = 38495
	DiscoveryPort  = 38496
	DefaultVersion = "dev"
	AppName        = "URL Bridge"
)

type OpenRequest struct {
	URL    string `json:"url"`
	Source string `json:"source,omitempty"`
}

type OpenResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
}

type HealthResponse struct {
	OK            bool   `json:"ok"`
	Name          string `json:"name"`
	Version       string `json:"version"`
	HostName      string `json:"host_name,omitempty"`
	TokenRequired bool   `json:"token_required,omitempty"`
}

type DiscoveryRequest struct {
	App string `json:"app"`
}

type DiscoveryResponse struct {
	OK                bool     `json:"ok"`
	Name              string   `json:"name"`
	HostName          string   `json:"host_name,omitempty"`
	HTTPPort          int      `json:"http_port,omitempty"`
	TokenRequired     bool     `json:"token_required,omitempty"`
	CandidateBaseURLs []string `json:"candidate_base_urls,omitempty"`
	Message           string   `json:"message,omitempty"`
}

func NormalizeURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("url is required")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("parse url: %w", err)
	}

	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
	default:
		return "", fmt.Errorf("only http and https are supported")
	}

	if parsed.Host == "" {
		return "", fmt.Errorf("url host is required")
	}

	return parsed.String(), nil
}

func MustJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}

	return string(data)
}

func RandomToken(bytes int) (string, error) {
	if bytes <= 0 {
		bytes = 16
	}

	buf := make([]byte, bytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}

	return hex.EncodeToString(buf), nil
}
