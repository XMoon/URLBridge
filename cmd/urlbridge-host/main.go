package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/xmoon/urlbridge/internal/bridge"
	"github.com/xmoon/urlbridge/internal/host"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "token", "secret":
			token, err := bridge.RandomToken(16)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(token)
			return
		}
	}

	cfg, err := parseRuntimeConfig(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	logger := log.New(os.Stdout, "", log.LstdFlags)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := host.Serve(ctx, host.Config{
		ListenAddr: cfg.ListenAddr,
		Token:      cfg.Token,
		Discovery:  cfg.Discovery,
		Logger:     logger,
	}); err != nil {
		log.Fatal(err)
	}
}

func parseRuntimeConfig(args []string) (host.FileConfig, error) {
	fs := flag.NewFlagSet("urlbridge-host", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	configPath := fs.String("config", "", "path to config.yaml")
	listenAddr := fs.String("listen", "", "listen address for the host bridge service")
	token := fs.String("token", "", "shared secret expected from the Windows guest")
	discovery := fs.Bool("discovery", false, "enable UDP host discovery")
	if err := fs.Parse(args); err != nil {
		return host.FileConfig{}, err
	}

	cfg, _, err := host.LoadFileConfig(*configPath)
	if err != nil {
		return host.FileConfig{}, err
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

	if err := cfg.Normalize(); err != nil {
		return host.FileConfig{}, err
	}

	return cfg, nil
}

func visitedFlags(fs *flag.FlagSet) map[string]bool {
	visited := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) {
		visited[f.Name] = true
	})
	return visited
}
