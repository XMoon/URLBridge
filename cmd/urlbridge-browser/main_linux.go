//go:build linux

package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/xmoon/urlbridge/internal/guest"
)

func main() {
	fs := flag.NewFlagSet("urlbridge-browser", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	configPath := fs.String("config", "", "path to config.yaml")
	if err := fs.Parse(os.Args[1:]); err != nil {
		guest.ShowErrorDialog("URL Bridge", err.Error())
		return
	}

	args := fs.Args()
	if len(args) < 1 {
		guest.ShowErrorDialog("URL Bridge", "No URL was provided. Set URL Bridge as the handler for HTTP/HTTPS and try again.")
		return
	}
	if len(args) > 1 {
		guest.ShowErrorDialog("URL Bridge", fmt.Sprintf("Expected a single URL argument, got %d.", len(args)))
		return
	}

	target := args[0]
	if err := guest.ForwardURLWithNotice(target, *configPath, func(message string) {
		guest.ShowErrorDialog("URL Bridge", message)
	}); err != nil {
		guest.ShowErrorDialog("URL Bridge", err.Error())
	}
}
