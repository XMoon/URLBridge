//go:build linux

package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
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
	case "uninstall":
		if err := guest.Uninstall(); err != nil {
			fatal(err)
		}
		fmt.Println("URL Bridge was unregistered from xdg-open. Installed binaries were left in place.")
	case "help", "-h", "--help":
		usage()
	default:
		usage()
		os.Exit(2)
	}
}

func install(args []string) {
	fs, parseOutput := newCommandFlagSet("install")
	configPath := fs.String("config", "", "path to config.yaml")
	hostURL := fs.String("host-url", "", "base URL of the host service, e.g. http://10.0.2.2:38495")
	token := fs.String("token", "", "optional shared secret expected by the host")
	timeout := fs.Int("timeout", 0, "HTTP timeout in seconds")
	if !parseCommandFlags(fs, parseOutput, args, installUsage) {
		return
	}

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
		HostBaseURL:    cfg.HostBaseURL,
		Token:          cfg.Token,
		BrowserPath:    cfg.BrowserPath,
		ConfigPath:     resolvedConfigPath,
		TimeoutSeconds: cfg.RequestTimeoutSeconds,
	}); err != nil {
		fatal(err)
	}

	fmt.Println("URL Bridge was registered as the xdg-open handler for this Linux user.")
	fmt.Printf("Host selection: %s (%s)\n", resolved.BaseURL, resolved.Resolved)
	fmt.Printf("Config file: %s\n", resolvedConfigPath)
	printSchemeHandlerStatus()
	fmt.Println("Next step: run `xdg-open https://example.com` inside the VM to verify the handler.")

	if err := guest.HealthCheck(cfg); err != nil {
		fmt.Printf("Host connectivity check: %v\n", err)
	} else {
		fmt.Println("Host connectivity check: OK")
	}
}

func discover(args []string) {
	fs, parseOutput := newCommandFlagSet("discover")
	configPath := fs.String("config", "", "path to config.yaml")
	timeout := fs.Int("timeout", 0, "discovery timeout in seconds")
	if !parseCommandFlags(fs, parseOutput, args, discoverUsage) {
		return
	}

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
	fs, parseOutput := newCommandFlagSet("status")
	configPath := fs.String("config", "", "path to config.yaml")
	if !parseCommandFlags(fs, parseOutput, args, statusUsage) {
		return
	}

	cfg, resolvedConfigPath, err := guest.LoadConfig(*configPath)
	if err != nil {
		fatal(fmt.Errorf("load status: %w", err))
	}

	installDir, _ := guest.InstallDir()
	fmt.Printf("Install dir: %s\n", installDir)
	fmt.Printf("Config file: %s\n", resolvedConfigPath)
	printSchemeHandlerStatus()
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

func printSchemeHandlerStatus() {
	status := guest.URLSchemeHandlerStatus()
	if status.DesktopFilePath != "" {
		fmt.Printf("Desktop entry: %s\n", status.DesktopFilePath)
	}
	fmt.Printf("XDG http handler: %s\n", configuredOrUnset(status.HTTPDefault))
	fmt.Printf("XDG https handler: %s\n", configuredOrUnset(status.HTTPSDefault))
}

func configuredOrUnset(value string) string {
	if strings.TrimSpace(value) == "" {
		return "not configured"
	}
	return value
}

func usage() {
	fmt.Println(`URL Bridge guest controller

Usage:
  urlbridge-guestctl install [--config PATH] [--host-url http://HOST:38495] [--token TOKEN]
  urlbridge-guestctl discover [--config PATH]
  urlbridge-guestctl status [--config PATH]
  urlbridge-guestctl uninstall`)
}

func installUsage() {
	fmt.Println(`Usage:
  urlbridge-guestctl install [--config PATH] [--host-url http://HOST:38495] [--token TOKEN] [--timeout SECONDS]`)
}

func discoverUsage() {
	fmt.Println(`Usage:
  urlbridge-guestctl discover [--config PATH] [--timeout SECONDS]`)
}

func statusUsage() {
	fmt.Println(`Usage:
  urlbridge-guestctl status [--config PATH]`)
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

func newCommandFlagSet(name string) (*flag.FlagSet, *bytes.Buffer) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.Usage = func() {}

	var output bytes.Buffer
	fs.SetOutput(&output)
	return fs, &output
}

func parseCommandFlags(fs *flag.FlagSet, output *bytes.Buffer, args []string, usage func()) bool {
	err := fs.Parse(args)
	if err == nil {
		return true
	}
	if errors.Is(err, flag.ErrHelp) {
		usage()
		return false
	}

	message := strings.TrimSpace(output.String())
	if message == "" {
		message = err.Error()
	}
	fmt.Fprintln(os.Stderr, message)
	usage()
	os.Exit(2)
	return false
}
