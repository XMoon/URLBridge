package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/xmoon/urlbridge/internal/bridge"
	"github.com/xmoon/urlbridge/internal/host"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "token", "secret":
			token, err := bridge.RandomToken(16)
			if err != nil {
				return err
			}
			fmt.Println(token)
			return nil
		}
	}

	cfg, err := parseRuntimeConfig(os.Args[1:])
	if err != nil {
		return err
	}

	logger, closer, err := host.NewLogger(os.Stdout, cfg.FileConfig)
	if err != nil {
		return err
	}
	defer func() {
		if closer != nil {
			_ = closer.Close()
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if cfg.TokenGenerated {
		logger.Printf("generated a new host token and saved it to %s; check the config file to retrieve the token", cfg.ConfigPath)
	}

	if err := host.Serve(ctx, host.Config{
		ListenAddr: cfg.FileConfig.ListenAddr,
		Token:      cfg.FileConfig.Token,
		Discovery:  cfg.FileConfig.Discovery,
		Logger:     logger,
	}); err != nil {
		return err
	}

	return nil
}

type runtimeConfig struct {
	host.FileConfig
	ConfigPath     string
	TokenGenerated bool
}

func parseRuntimeConfig(args []string) (runtimeConfig, error) {
	fs := flag.NewFlagSet("urlbridge-host", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	configPath := fs.String("config", "", "path to config.yaml")
	listenAddr := fs.String("listen", "", "listen address for the host bridge service")
	token := fs.String("token", "", "shared secret expected from the Windows guest")
	discovery := fs.Bool("discovery", false, "enable UDP host discovery")
	if err := fs.Parse(args); err != nil {
		return runtimeConfig{}, err
	}

	cfg, resolvedConfigPath, err := host.LoadFileConfig(*configPath)
	if err != nil {
		return runtimeConfig{}, err
	}

	visited := visitedFlags(fs)
	if visited["listen"] {
		cfg.ListenAddr = *listenAddr
	}
	if visited["token"] {
		cfg.Token = *token
	}
	if visited["discovery"] {
		cfg.Discovery = *discovery
	}

	tokenGenerated := false
	if *configPath == "" && strings.TrimSpace(cfg.Token) == "" {
		generatedToken, err := bridge.RandomToken(16)
		if err != nil {
			return runtimeConfig{}, err
		}
		cfg.Token = generatedToken
		tokenGenerated = true

		if strings.TrimSpace(resolvedConfigPath) == "" {
			resolvedConfigPath, err = host.DefaultConfigPath()
			if err != nil {
				return runtimeConfig{}, err
			}
		}
	}

	if err := cfg.Normalize(); err != nil {
		return runtimeConfig{}, err
	}

	if tokenGenerated {
		if err := host.SaveFileConfig(cfg, resolvedConfigPath); err != nil {
			return runtimeConfig{}, err
		}
	}

	return runtimeConfig{
		FileConfig:     cfg,
		ConfigPath:     resolvedConfigPath,
		TokenGenerated: tokenGenerated,
	}, nil
}

func visitedFlags(fs *flag.FlagSet) map[string]bool {
	visited := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) {
		visited[f.Name] = true
	})
	return visited
}
