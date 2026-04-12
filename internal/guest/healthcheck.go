package guest

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

func HealthCheck(cfg Config) error {
	if err := cfg.NormalizeForRuntime(); err != nil {
		return err
	}

	client := &http.Client{Timeout: time.Duration(cfg.RequestTimeoutSeconds) * time.Second}
	target, err := endpointURL(cfg.HostBaseURL, "/probe")
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	if cfg.Token != "" {
		req.Header.Set("X-URLBridge-Token", cfg.Token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("reach host: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 32*1024))
	if err != nil {
		return fmt.Errorf("read probe response: %w", err)
	}

	if err := parseProbeResponse(resp.StatusCode, body); err != nil {
		return err
	}

	return nil
}
