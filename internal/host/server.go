package host

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/xmoon/urlbridge/internal/bridge"
)

const tokenHeader = "X-URLBridge-Token"

type Config struct {
	ListenAddr string
	Token      string
	Discovery  bool
	Logger     *log.Logger
}

func Serve(ctx context.Context, cfg Config) error {
	logger := cfg.Logger
	if logger == nil {
		return errors.New("logger is required")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		hostname, _ := os.Hostname()
		writeJSON(w, http.StatusOK, bridge.HealthResponse{
			OK:            true,
			Name:          bridge.AppName,
			Version:       bridge.DefaultVersion,
			HostName:      hostname,
			TokenRequired: cfg.Token != "",
		})
	})

	mux.HandleFunc("GET /open", func(w http.ResponseWriter, r *http.Request) {
		if cfg.Token != "" && subtleTrim(r.Header.Get(tokenHeader)) != cfg.Token {
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

		if err := openAndLog(r.Context(), logger, r.RemoteAddr, target, "query"); err != nil {
			writeJSON(w, http.StatusBadGateway, bridge.OpenResponse{
				OK:      false,
				Message: err.Error(),
			})
			return
		}

		writeJSON(w, http.StatusOK, bridge.OpenResponse{OK: true})
	})

	mux.HandleFunc("POST /open", func(w http.ResponseWriter, r *http.Request) {
		if cfg.Token != "" && subtleTrim(r.Header.Get(tokenHeader)) != cfg.Token {
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

		if err := openAndLog(r.Context(), logger, r.RemoteAddr, target, source); err != nil {
			writeJSON(w, http.StatusBadGateway, bridge.OpenResponse{
				OK:      false,
				Message: err.Error(),
			})
			return
		}

		writeJSON(w, http.StatusOK, bridge.OpenResponse{OK: true})
	})

	server := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           loggingMiddleware(logger, mux),
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
			Logger:     logger,
			Token:      cfg.Token,
		}); err != nil {
			return err
		}
	}

	logger.Printf("URL Bridge host listening on %s", cfg.ListenAddr)
	if cfg.Token == "" {
		logger.Printf("warning: no token configured; only use this on a trusted VM network")
	}

	for _, candidate := range CandidateBaseURLs(cfg.ListenAddr) {
		logger.Printf("guest can use: %s", candidate)
	}

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func loggingMiddleware(logger *log.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Printf("%s %s from %s in %s", r.Method, r.URL.Path, r.RemoteAddr, time.Since(start).Round(time.Millisecond))
	})
}

func openAndLog(ctx context.Context, logger *log.Logger, remoteAddr, targetURL, source string) error {
	logger.Printf("opening %s from %s (%s)", targetURL, source, remoteAddr)
	if err := OpenBrowser(ctx, targetURL); err != nil {
		return fmt.Errorf("open browser: %w", err)
	}
	return nil
}

func subtleTrim(s string) string {
	return strings.TrimSpace(s)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
