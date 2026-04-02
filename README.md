# URL Bridge

[English](README.md) | [简体中文](README.zh-CN.md)

URL Bridge lets a Windows 10/11 virtual machine act like a lightweight browser shim: when code, chat tools, documents, or other apps inside the VM open an `http://` or `https://` link, the URL is forwarded to a service running on the host, and the host opens the link in its own default browser.

## What this repo contains

- `urlbridge-host`: cross-platform host service for Windows, Linux, and macOS.
- `urlbridge-browser.exe`: the Windows URL handler that gets registered for `http` and `https`.
- `urlbridge-guestctl.exe`: Windows helper used to install, inspect, and unregister the guest-side integration.

## Why there are two Windows binaries

The browser handler itself is built as a GUI process so clicking a link inside the VM does not pop a console window. The controller stays as a normal CLI so installation and diagnostics remain straightforward.

## How it works

1. Run `urlbridge-host` on the host OS.
2. Inside the Windows VM, run `urlbridge-guestctl.exe install`, either with `--host-url`, or with `--token` if the host requires authentication.
3. Windows opens the Default Apps page.
4. Set `HTTP` and `HTTPS` to `URL Bridge`.
5. From then on, clicking a URL in the VM sends it to the host service, which opens the host browser.

## Discovery

URL Bridge supports guest-side auto-discovery:

- The host can answer UDP discovery on port `38496`.
- The Windows guest also probes common VM host addresses such as `10.0.2.2`, `10.0.3.2`, and the current default gateway.

## Important Windows behavior

On Windows 10/11, third-party applications cannot silently replace the user's default browser selection. URL Bridge therefore registers itself as a candidate handler and opens the system's Default Apps page so the user can explicitly assign `HTTP` and `HTTPS` to it.

Microsoft documentation used for this behavior:

- <https://learn.microsoft.com/en-us/windows/win32/shell/default-programs>
- <https://learn.microsoft.com/en-us/windows/apps/develop/launch/launch-default-apps-settings>

## Build

```bash
make fmt
make test
make build-all
```

Local build artifacts are written to `dist/`.

## Configuration files

URL Bridge now supports `config.yaml` for both the host and the Windows guest. When `--config PATH` is not provided, config files are searched in this order:

1. the current working directory
2. the executable directory
3. the platform default path

Default paths:

- Host on Windows: `%LOCALAPPDATA%\URLBridgeHost\config.yaml`
- Guest on Windows: `%LOCALAPPDATA%\URLBridge\config.yaml`
- Host on Linux: `$XDG_CONFIG_HOME/urlbridge/config.yaml` or `~/.config/urlbridge/config.yaml`, then `/etc/urlbridge/config.yaml`
- Host on macOS: `~/Library/Application Support/URLBridge/config.yaml`, then `/etc/urlbridge/config.yaml`

Precedence is always: built-in defaults, then `config.yaml`, then explicitly provided CLI flags.

Older `config.json` files are no longer read or migrated.

Example host config:

```yaml
listen_addr: "0.0.0.0:38495"
token: "YOUR_TOKEN"
discovery: true
log_path: "/path/to/host.log" # optional; use "" to disable file logging
```

Example guest config:

```yaml
host_base_url: "http://10.0.2.2:38495/"
token: "YOUR_TOKEN"
request_timeout_seconds: 3
browser_path: "C:/Program Files/Google/Chrome/Application/chrome.exe" # optional
```

## Host usage

Generate a token if you want the VM-to-host hop to be authenticated. Use the host binary that matches your OS and CPU, for example on Linux x64:

```bash
./dist/urlbridge-host-linux-amd64 token
```

Run the host service with the binary that matches the host machine:

```bash
./dist/urlbridge-host-linux-amd64 --listen 0.0.0.0:38495 --token YOUR_TOKEN
```

Or point it at a config file:

```bash
./dist/urlbridge-host-linux-amd64 --config ./config.yaml
```

Notes:

- `0.0.0.0` is useful because the VM usually reaches the host over a virtual NIC instead of loopback.
- The service prints candidate host URLs based on the host machine's IPv4 interfaces.
- Discovery is enabled by default on UDP `38496`; disable it with `--discovery=false`.
- Host logs always go to stdout, and by default also to a file. Set `log_path: ""` to disable file logging.
- Default host log paths are `%LOCALAPPDATA%\URLBridgeHost\host.log` on Windows, `$XDG_STATE_HOME/urlbridge/host.log` or `~/.local/state/urlbridge/host.log` on Linux, and `~/Library/Logs/URLBridge/host.log` on macOS.
- On Linux, URL Bridge uses `xdg-open`, then falls back to `gio open`.
- On macOS it uses `open`, and on Windows it uses the Windows shell/default URL handler.

### Host install scripts

From the repo root:

```bash
./scripts/install-host.sh
```

On Windows host:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\install-host.ps1
```

Both installers:

- detect the correct host binary
- install it into a per-user location
- write or reuse `config.yaml`
- generate a token if you do not provide one
- start the host immediately
- configure user-level autostart when possible
- launch the host with `--config <path>`

## Guest usage

Copy these two files into the Windows VM:

- `dist/urlbridge-browser.exe`
- `dist/urlbridge-guestctl.exe`

If the guest VM is Windows on ARM, use `dist/urlbridge-browser-arm64.exe` and `dist/urlbridge-guestctl-arm64.exe` instead.

Then install:

```powershell
.\urlbridge-guestctl.exe install --host-url http://10.0.2.2:38495 --token YOUR_TOKEN
```

Or let the VM auto-discover the host:

```powershell
.\urlbridge-guestctl.exe install
```

You can inspect what the VM can find before installing:

```powershell
.\urlbridge-guestctl.exe discover
```

Common host URL choices:

- VirtualBox NAT often exposes the host as `http://10.0.2.2:38495`.
- Host-only or bridged networking usually uses the host's actual LAN or host-only IP.
- The host OS does not matter here; the VM only needs network reachability to the host service.

After installation:

- Windows opens the general Default Apps page for compatibility with both Windows 10 and Windows 11.
- Set `HTTP` and `HTTPS` to `URL Bridge`.
- If the host cannot be reached within the configured timeout, the VM falls back to a local browser. When `browser_path` is empty, URL Bridge auto-detects Chrome first, then Edge, opens the first one that works, and saves it to the config for later reuse.
- If the saved `browser_path` later stops launching, URL Bridge re-detects local browsers and asks whether it should replace the stored path before opening the link with a new browser.

Useful guest commands:

```powershell
.\urlbridge-guestctl.exe install --config .\config.yaml --host-url http://10.0.2.2:38495
.\urlbridge-guestctl.exe discover
.\urlbridge-guestctl.exe discover --config .\config.yaml
.\urlbridge-guestctl.exe status
.\urlbridge-guestctl.exe status --config .\config.yaml
.\urlbridge-guestctl.exe open-default-settings
.\urlbridge-guestctl.exe uninstall
```

### Guest install script

From the repo root or from a packaged guest bundle:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\install-guest.ps1
```

The guest installer writes the config to `%LOCALAPPDATA%\URLBridge\config.yaml` by default and registers the Windows URL handler with an explicit `--config` argument. It keeps any existing `browser_path` value and uses a 3-second host timeout unless you override it. If `browser_path` is still empty, the first real local-browser fallback will detect and cache one automatically. You can still pass `-HostUrl`, `-Token`, or `-ConfigPath` if you prefer a fully explicit setup.

## Current limitations

- URL Bridge forwards only `http` and `https`.
- It does not proxy cookies, session state, or browser profiles from the VM.
- The host service expects the VM to be able to reach the host over the chosen virtual network.
- On Windows 10/11, the final default-app assignment still requires user interaction.
- UDP discovery depends on the VM network mode; if broadcast is blocked, use a reachable NAT alias like `10.0.2.2` with `--host-url`, or pass the host URL explicitly.
