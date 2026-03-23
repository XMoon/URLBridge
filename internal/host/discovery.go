package host

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/xmoon/urlbridge/internal/bridge"
)

type DiscoveryConfig struct {
	ListenAddr string
	Port       int
	Logger     *log.Logger
	Token      string
}

func StartDiscovery(ctx context.Context, cfg DiscoveryConfig) error {
	if cfg.Port == 0 {
		cfg.Port = bridge.DiscoveryPort
	}
	if cfg.Logger == nil {
		return fmt.Errorf("logger is required")
	}

	httpPort, err := extractPort(cfg.ListenAddr)
	if err != nil {
		return fmt.Errorf("extract HTTP port: %w", err)
	}

	conn, err := net.ListenUDP("udp4", &net.UDPAddr{Port: cfg.Port})
	if err != nil {
		return fmt.Errorf("listen for discovery: %w", err)
	}

	cfg.Logger.Printf("URL Bridge discovery listening on udp/%d", cfg.Port)

	go func() {
		defer conn.Close()

		<-ctx.Done()
		_ = conn.SetReadDeadline(time.Now())
	}()

	go discoveryLoop(conn, discoveryRuntime{
		httpPort: httpPort,
		logger:   cfg.Logger,
		token:    cfg.Token,
	})

	return nil
}

type discoveryRuntime struct {
	httpPort int
	logger   *log.Logger
	token    string
}

func discoveryLoop(conn *net.UDPConn, rt discoveryRuntime) {
	buffer := make([]byte, 64*1024)

	for {
		n, addr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				return
			}
			if rt.logger != nil {
				rt.logger.Printf("discovery read error: %v", err)
			}
			return
		}

		var req bridge.DiscoveryRequest
		if err := json.Unmarshal(buffer[:n], &req); err != nil {
			continue
		}
		if req.App != bridge.AppName {
			continue
		}

		payload, err := json.Marshal(rt.response())
		if err != nil {
			if rt.logger != nil {
				rt.logger.Printf("discovery marshal error: %v", err)
			}
			continue
		}

		if _, err := conn.WriteToUDP(payload, addr); err != nil {
			if rt.logger != nil {
				rt.logger.Printf("discovery write error to %s: %v", addr, err)
			}
			continue
		}

		if rt.logger != nil {
			rt.logger.Printf("answered discovery from %s", addr)
		}
	}
}

func (rt discoveryRuntime) response() bridge.DiscoveryResponse {
	hostname, _ := os.Hostname()

	return bridge.DiscoveryResponse{
		OK:                true,
		Name:              bridge.AppName,
		HostName:          hostname,
		HTTPPort:          rt.httpPort,
		TokenRequired:     rt.token != "",
		CandidateBaseURLs: CandidateBaseURLs(fmt.Sprintf("0.0.0.0:%d", rt.httpPort)),
	}
}

func extractPort(listenAddr string) (int, error) {
	_, port, err := net.SplitHostPort(listenAddr)
	if err != nil {
		return 0, err
	}

	addr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort("127.0.0.1", port))
	if err != nil {
		return 0, err
	}

	return addr.Port, nil
}
