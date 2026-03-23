//go:build windows

package guest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/xmoon/urlbridge/internal/bridge"
)

func ForwardURL(rawURL, configPath string) error {
	cfg, _, err := LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	targetURL, err := bridge.NormalizeURL(rawURL)
	if err != nil {
		return err
	}

	openEndpoint, err := endpointURL(cfg.HostBaseURL, "/open")
	if err != nil {
		return err
	}

	body, err := json.Marshal(bridge.OpenRequest{
		URL:    targetURL,
		Source: "windows-vm",
	})
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	client := &http.Client{Timeout: time.Duration(cfg.RequestTimeoutSeconds) * time.Second}
	req, err := http.NewRequest(http.MethodPost, openEndpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.Token != "" {
		req.Header.Set("X-URLBridge-Token", cfg.Token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send to host: %w", err)
	}
	defer resp.Body.Close()

	var openResp bridge.OpenResponse
	if err := json.NewDecoder(resp.Body).Decode(&openResp); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		if openResp.Message == "" {
			openResp.Message = resp.Status
		}
		return fmt.Errorf("host refused request: %s", openResp.Message)
	}

	if !openResp.OK {
		if openResp.Message == "" {
			openResp.Message = "host did not accept the request"
		}
		return fmt.Errorf(openResp.Message)
	}

	return nil
}
