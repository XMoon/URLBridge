//go:build windows

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/xmoon/urlbridge/internal/guest"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}

	switch os.Args[1] {
	case "install":
		install(os.Args[2:])
	case "discover":
		discover(os.Args[2:])
	case "status":
		status(os.Args[2:])
	case "open-default-settings":
		if err := guest.OpenDefaultAppsSettings(); err != nil {
			fatal(err)
		}
	case "uninstall":
		if err := guest.Uninstall(); err != nil {
			fatal(err)
		}
		fmt.Println("URL Bridge unregistered. Installed binaries were left in place.")
	case "help", "-h", "--help":
		usage()
	default:
		usage()
		os.Exit(2)
	}
}

func install(args []string) {
	fs := flag.NewFlagSet("install", flag.ExitOnError)
	fs.SetOutput(io.Discard)
	configPath := fs.String("config", "", "path to config.yaml")
	hostURL := fs.String("host-url", "", "base URL of the host service, e.g. http://10.0.2.2:38495")
	token := fs.String("token", "", "optional shared secret expected by the host")
	noOpenSettings := fs.Bool("no-open-settings", false, "do not open the Windows Default Apps page after registration")
	timeout := fs.Int("timeout", 0, "HTTP timeout in seconds")
	fs.Parse(args)

	cfg, resolvedConfigPath, err := guest.LoadConfigForInstall(*configPath)
	if err != nil {
		fatal(err)
	}

	visited := visitedFlags(fs)
	if visited["host-url"] {
		cfg.HostBaseURL = *hostURL
	}
	if visited["token"] {
		cfg.Token = *token
	}
	if visited["timeout"] {
		cfg.RequestTimeoutSeconds = *timeout
	}
	if err := cfg.Normalize(); err != nil {
		fatal(err)
	}

	resolved, err := guest.ResolveInstallTarget(
		cfg.HostBaseURL,
		cfg.Token,
		time.Duration(cfg.RequestTimeoutSeconds)*time.Second,
	)
	if err != nil {
		fatal(err)
	}

	cfg.HostBaseURL = resolved.BaseURL
	cfg.Token = resolved.Token

	if err := guest.Install(guest.InstallOptions{
		HostBaseURL:      cfg.HostBaseURL,
		Token:            cfg.Token,
		BrowserPath:      cfg.BrowserPath,
		ConfigPath:       resolvedConfigPath,
		OpenSettingsPage: !*noOpenSettings,
		TimeoutSeconds:   cfg.RequestTimeoutSeconds,
	}); err != nil {
		fatal(err)
	}

	fmt.Println("URL Bridge was registered for this Windows user.")
	fmt.Printf("Host selection: %s (%s)\n", resolved.BaseURL, resolved.Resolved)
	fmt.Printf("Config file: %s\n", resolvedConfigPath)
	fmt.Println("Next step: in Default Apps, set URL Bridge as the handler for HTTP and HTTPS.")

	if err := guest.HealthCheck(cfg); err != nil {
		fmt.Printf("Host connectivity check: %v\n", err)
	} else {
		fmt.Println("Host connectivity check: OK")
	}
}

func discover(args []string) {
	fs := flag.NewFlagSet("discover", flag.ExitOnError)
	fs.SetOutput(io.Discard)
	configPath := fs.String("config", "", "path to config.yaml")
	timeout := fs.Int("timeout", 0, "discovery timeout in seconds")
	fs.Parse(args)

	cfgState, err := guest.LoadConfigState(*configPath)
	if err != nil {
		fatal(err)
	}
	if *configPath != "" && !cfgState.Found {
		fatal(fmt.Errorf("config file not found: %s", cfgState.Path))
	}

	cfg := cfgState.Config
	if visitedFlags(fs)["timeout"] {
		cfg.RequestTimeoutSeconds = *timeout
	}
	if err := cfg.Normalize(); err != nil {
		fatal(err)
	}

	candidates, err := guest.DiscoverHosts(time.Duration(cfg.RequestTimeoutSeconds) * time.Second)
	if err != nil {
		fatal(err)
	}

	for _, candidate := range candidates {
		fmt.Printf("Host: %s\n", candidate.BaseURL)
		fmt.Printf("  Source: %s\n", candidate.Source)
		fmt.Printf("  Name: %s\n", candidate.HostNameOrFallback())
		fmt.Printf("  Token required: %t\n", candidate.TokenRequired)
	}
}

func status(args []string) {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	fs.SetOutput(io.Discard)
	configPath := fs.String("config", "", "path to config.yaml")
	fs.Parse(args)

	cfg, resolvedConfigPath, err := guest.LoadConfig(*configPath)
	if err != nil {
		fatal(fmt.Errorf("load status: %w", err))
	}

	installDir, _ := guest.InstallDir()
	fmt.Printf("Install dir: %s\n", installDir)
	fmt.Printf("Config file: %s\n", resolvedConfigPath)
	fmt.Printf("Host service: %s\n", cfg.HostBaseURL)
	if cfg.Token != "" {
		fmt.Println("Token: configured")
	} else {
		fmt.Println("Token: not configured")
	}

	if err := guest.HealthCheck(cfg); err != nil {
		fmt.Printf("Host connectivity: %v\n", err)
		return
	}

	fmt.Println("Host connectivity: OK")
}

func usage() {
	fmt.Println(`URL Bridge guest controller

Usage:
  urlbridge-guestctl.exe install [--config PATH] [--host-url http://HOST:38495] [--token TOKEN]
  urlbridge-guestctl.exe discover [--config PATH]
  urlbridge-guestctl.exe status [--config PATH]
  urlbridge-guestctl.exe open-default-settings
  urlbridge-guestctl.exe uninstall`)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func visitedFlags(fs *flag.FlagSet) map[string]bool {
	visited := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) {
		visited[f.Name] = true
	})
	return visited
}
