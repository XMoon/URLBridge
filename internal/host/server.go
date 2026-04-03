package host

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/xmoon/urlbridge/internal/bridge"
)

const tokenHeader = "X-URLBridge-Token"

type Config struct {
	ListenAddr  string
	Token       string
	Discovery   bool
	LogFullURLs bool
	Logger      *log.Logger
}

func Serve(ctx context.Context, cfg Config) error {
	handler, err := newHandler(cfg)
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	if cfg.Discovery {
		if err := StartDiscovery(ctx, DiscoveryConfig{
			ListenAddr: cfg.ListenAddr,
			Logger:     cfg.Logger,
			Token:      cfg.Token,
		}); err != nil {
			return err
		}
	}

	cfg.Logger.Printf("URL Bridge host listening on %s", cfg.ListenAddr)
	if cfg.Token == "" {
		cfg.Logger.Printf("warning: no token configured; only use this on a trusted VM network")
	}

	for _, candidate := range CandidateBaseURLs(cfg.ListenAddr) {
		cfg.Logger.Printf("guest can use: %s", candidate)
	}

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func newHandler(cfg Config) (http.Handler, error) {
	logger := cfg.Logger
	if logger == nil {
		return nil, errors.New("logger is required")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, healthResponse(cfg))
	})

	mux.HandleFunc("GET /probe", func(w http.ResponseWriter, r *http.Request) {
		status, resp := probeResponse(cfg, r.Header.Get(tokenHeader))
		writeJSON(w, status, resp)
	})

	mux.HandleFunc("GET /open", func(w http.ResponseWriter, r *http.Request) {
		if !tokenAuthorized(cfg.Token, r.Header.Get(tokenHeader)) {
			writeJSON(w, http.StatusUnauthorized, bridge.OpenResponse{
				OK:      false,
				Message: "invalid token",
			})
			return
		}

		target, err := bridge.NormalizeURL(r.URL.Query().Get("url"))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, bridge.OpenResponse{
				OK:      false,
				Message: err.Error(),
			})
			return
		}

		if err := openAndLog(r.Context(), logger, r.RemoteAddr, target, "query", cfg.LogFullURLs); err != nil {
			writeJSON(w, http.StatusBadGateway, bridge.OpenResponse{
				OK:      false,
				Message: err.Error(),
			})
			return
		}

		writeJSON(w, http.StatusOK, bridge.OpenResponse{OK: true})
	})

	mux.HandleFunc("POST /open", func(w http.ResponseWriter, r *http.Request) {
		if !tokenAuthorized(cfg.Token, r.Header.Get(tokenHeader)) {
			writeJSON(w, http.StatusUnauthorized, bridge.OpenResponse{
				OK:      false,
				Message: "invalid token",
			})
			return
		}

		defer r.Body.Close()

		var req bridge.OpenRequest
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32*1024))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, bridge.OpenResponse{
				OK:      false,
				Message: fmt.Sprintf("decode request: %v", err),
			})
			return
		}

		target, err := bridge.NormalizeURL(req.URL)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, bridge.OpenResponse{
				OK:      false,
				Message: err.Error(),
			})
			return
		}

		source := req.Source
		if source == "" {
			source = "guest"
		}

		if err := openAndLog(r.Context(), logger, r.RemoteAddr, target, source, cfg.LogFullURLs); err != nil {
			writeJSON(w, http.StatusBadGateway, bridge.OpenResponse{
				OK:      false,
				Message: err.Error(),
			})
			return
		}

		writeJSON(w, http.StatusOK, bridge.OpenResponse{OK: true})
	})

	return loggingMiddleware(logger, mux), nil
}

func loggingMiddleware(logger *log.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Printf("%s %s from %s in %s", r.Method, r.URL.Path, r.RemoteAddr, time.Since(start).Round(time.Millisecond))
	})
}

func openAndLog(ctx context.Context, logger *log.Logger, remoteAddr, targetURL, source string, logFullURLs bool) error {
	logger.Printf("opening %s from %s (%s)", loggableURL(targetURL, logFullURLs), source, remoteAddr)
	if err := OpenBrowser(ctx, targetURL); err != nil {
		return fmt.Errorf("open browser: %w", err)
	}
	return nil
}

func healthResponse(cfg Config) bridge.HealthResponse {
	hostname, _ := os.Hostname()
	return bridge.HealthResponse{
		OK:            true,
		Name:          bridge.AppName,
		Version:       bridge.DefaultVersion,
		HostName:      hostname,
		TokenRequired: cfg.Token != "",
	}
}

func probeResponse(cfg Config, presentedToken string) (int, bridge.ProbeResponse) {
	health := healthResponse(cfg)
	authenticated := tokenAuthorized(cfg.Token, presentedToken)
	resp := bridge.ProbeResponse{
		OK:            authenticated,
		Name:          health.Name,
		Version:       health.Version,
		HostName:      health.HostName,
		TokenRequired: health.TokenRequired,
		Authenticated: authenticated,
	}
	if !authenticated {
		resp.Message = "invalid token"
		return http.StatusUnauthorized, resp
	}

	return http.StatusOK, resp
}

func tokenAuthorized(expectedToken, presentedToken string) bool {
	if subtleTrim(expectedToken) == "" {
		return true
	}
	return subtleTrim(presentedToken) == expectedToken
}

func loggableURL(targetURL string, logFullURLs bool) string {
	if logFullURLs {
		return targetURL
	}

	parsed, err := url.Parse(targetURL)
	if err != nil {
		return targetURL
	}

	parsed.User = nil
	parsed.RawQuery = ""
	parsed.Fragment = ""
	parsed.ForceQuery = false
	return parsed.String()
}

func subtleTrim(s string) string {
	return strings.TrimSpace(s)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
