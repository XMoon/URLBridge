DIST_DIR := dist
GOCACHE ?= $(abspath ./.cache/go-build)

export GOCACHE

.PHONY: fmt test build-host build-guest build-all clean

fmt:
	gofmt -w .

test:
	go test ./...

build-host:
	mkdir -p $(DIST_DIR)
	go build -o $(DIST_DIR)/urlbridge-host ./cmd/urlbridge-host

build-guest:
	mkdir -p $(DIST_DIR)
	GOOS=windows GOARCH=amd64 go build -o $(DIST_DIR)/urlbridge-browser.exe -ldflags="-H=windowsgui" ./cmd/urlbridge-browser
	GOOS=windows GOARCH=amd64 go build -o $(DIST_DIR)/urlbridge-guestctl.exe ./cmd/urlbridge-guestctl

build-all:
	mkdir -p $(DIST_DIR)
	go build -o $(DIST_DIR)/urlbridge-host-linux-amd64 ./cmd/urlbridge-host
	GOOS=linux GOARCH=arm64 go build -o $(DIST_DIR)/urlbridge-host-linux-arm64 ./cmd/urlbridge-host
	GOOS=windows GOARCH=amd64 go build -o $(DIST_DIR)/urlbridge-host-windows-amd64.exe ./cmd/urlbridge-host
	GOOS=windows GOARCH=arm64 go build -o $(DIST_DIR)/urlbridge-host-windows-arm64.exe ./cmd/urlbridge-host
	GOOS=darwin GOARCH=amd64 go build -o $(DIST_DIR)/urlbridge-host-darwin-amd64 ./cmd/urlbridge-host
	GOOS=darwin GOARCH=arm64 go build -o $(DIST_DIR)/urlbridge-host-darwin-arm64 ./cmd/urlbridge-host
	GOOS=windows GOARCH=amd64 go build -o $(DIST_DIR)/urlbridge-browser.exe -ldflags="-H=windowsgui" ./cmd/urlbridge-browser
	GOOS=windows GOARCH=amd64 go build -o $(DIST_DIR)/urlbridge-guestctl.exe ./cmd/urlbridge-guestctl
	GOOS=windows GOARCH=arm64 go build -o $(DIST_DIR)/urlbridge-browser-arm64.exe -ldflags="-H=windowsgui" ./cmd/urlbridge-browser
	GOOS=windows GOARCH=arm64 go build -o $(DIST_DIR)/urlbridge-guestctl-arm64.exe ./cmd/urlbridge-guestctl

clean:
	rm -rf $(DIST_DIR) ./.cache
