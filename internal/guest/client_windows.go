//go:build windows

package guest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/xmoon/urlbridge/internal/bridge"
)

func ForwardURL(rawURL, configPath string) error {
	return ForwardURLWithNotice(rawURL, configPath, nil)
}

func ForwardURLWithNotice(rawURL, configPath string, notify func(string)) error {
	cfg, _, err := LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	targetURL, err := bridge.NormalizeURL(rawURL)
	if err != nil {
		return err
	}

	failure := forwardToHost(targetURL, cfg)
	if failure.Err == nil {
		return nil
	}

	return handleHostForwardFailure(failure, notify, func() error {
		return OpenLocalBrowser(targetURL, cfg.BrowserPath)
	})
}

func forwardToHost(targetURL string, cfg Config) hostForwardFailure {
	openEndpoint, err := endpointURL(cfg.HostBaseURL, "/open")
	if err != nil {
		return hostForwardFailure{Err: err}
	}

	body, err := json.Marshal(bridge.OpenRequest{
		URL:    targetURL,
		Source: "windows-vm",
	})
	if err != nil {
		return hostForwardFailure{Err: fmt.Errorf("marshal request: %w", err)}
	}

	client := &http.Client{Timeout: time.Duration(cfg.RequestTimeoutSeconds) * time.Second}
	req, err := http.NewRequest(http.MethodPost, openEndpoint, bytes.NewReader(body))
	if err != nil {
		return hostForwardFailure{Err: fmt.Errorf("create request: %w", err)}
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.Token != "" {
		req.Header.Set("X-URLBridge-Token", cfg.Token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return classifyHostForwardFailure(0, "", "", fmt.Errorf("send to host: %w", err))
	}
	defer resp.Body.Close()

	var openResp bridge.OpenResponse
	if err := json.NewDecoder(resp.Body).Decode(&openResp); err != nil {
		return classifyHostForwardFailure(0, "", "", fmt.Errorf("decode response: %w", err))
	}

	if resp.StatusCode != http.StatusOK {
		return classifyHostForwardFailure(resp.StatusCode, resp.Status, strings.TrimSpace(openResp.Message), nil)
	}

	if !openResp.OK {
		return classifyHostForwardFailure(resp.StatusCode, resp.Status, strings.TrimSpace(openResp.Message), nil)
	}

	return hostForwardFailure{}
}
