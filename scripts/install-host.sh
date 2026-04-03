#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

LISTEN_ADDR="0.0.0.0:38495"
TOKEN=""
INSTALL_DIR="${HOME}/.local/lib/urlbridge"
BIN_DIR="${HOME}/.local/bin"
CONFIG_PATH=""
LISTEN_SET=0
TOKEN_SET=0
NO_AUTOSTART=0
DISCOVERY="true"
LOG_PATH=""
LOG_PATH_PRESENT=0
LOG_FULL_URLS="false"
LOG_FULL_URLS_PRESENT=0

usage() {
  cat <<'EOF'
URL Bridge host installer

Usage:
  ./install-host.sh [--listen 0.0.0.0:38495] [--token TOKEN] [--install-dir DIR] [--config PATH] [--no-autostart]
EOF
}

read_yaml_value() {
  local key="$1"
  local file="$2"
  local line value
  line="$(grep -m1 -E "^[[:space:]]*${key}:[[:space:]]*.*$" "$file" || true)"
  if [[ -z "$line" ]]; then
    return 1
  fi

  value="${line#*:}"
  value="$(printf '%s' "$value" | sed -E 's/^[[:space:]]*//; s/[[:space:]]*$//')"

  if [[ "$value" == \'*\' ]]; then
    value="${value#\'}"
    value="${value%\'}"
    value="${value//\'\'/\'}"
  fi

  printf '%s' "$value"
}

yaml_quote() {
  printf "%s" "$1" | sed "s/'/''/g"
}

normalize_bool() {
  local value="${1,,}"
  case "$value" in
    true|false)
      printf '%s' "$value"
      ;;
    *)
      echo "invalid discovery value in config: $1" >&2
      exit 2
      ;;
  esac
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --listen)
      LISTEN_ADDR="$2"
      LISTEN_SET=1
      shift 2
      ;;
    --token)
      TOKEN="$2"
      TOKEN_SET=1
      shift 2
      ;;
    --install-dir)
      INSTALL_DIR="$2"
      shift 2
      ;;
    --config)
      CONFIG_PATH="$2"
      shift 2
      ;;
    --no-autostart)
      NO_AUTOSTART=1
      shift
      ;;
    -h|--help|help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

detect_binary_name() {
  local os arch
  os="$(uname -s)"
  arch="$(uname -m)"

  case "$os/$arch" in
    Linux/x86_64)
      echo "urlbridge-host-linux-amd64"
      ;;
    Linux/aarch64|Linux/arm64)
      echo "urlbridge-host-linux-arm64"
      ;;
    Darwin/x86_64)
      echo "urlbridge-host-darwin-amd64"
      ;;
    Darwin/arm64)
      echo "urlbridge-host-darwin-arm64"
      ;;
    *)
      echo "Unsupported host platform: $os/$arch" >&2
      exit 1
      ;;
  esac
}

find_source_binary() {
  local binary_name="$1"
  local candidates=(
    "$SCRIPT_DIR/urlbridge-host"
    "$SCRIPT_DIR/$binary_name"
    "$ROOT_DIR/urlbridge-host"
    "$ROOT_DIR/$binary_name"
    "$ROOT_DIR/dist/$binary_name"
  )

  local candidate
  for candidate in "${candidates[@]}"; do
    if [[ -f "$candidate" ]]; then
      echo "$candidate"
      return 0
    fi
  done

  echo "Could not find $binary_name next to the installer or under dist/" >&2
  exit 1
}

start_for_current_session() {
  local runner="$1"
  nohup "$runner" >/dev/null 2>&1 </dev/null &
}

BINARY_NAME="$(detect_binary_name)"
SOURCE_BINARY="$(find_source_binary "$BINARY_NAME")"

mkdir -p "$INSTALL_DIR" "$BIN_DIR"

if [[ -z "$CONFIG_PATH" ]]; then
  CONFIG_PATH="$INSTALL_DIR/config.yaml"
fi

INSTALLED_BINARY="$INSTALL_DIR/urlbridge-host"
RUNNER_SCRIPT="$INSTALL_DIR/run-host.sh"
BIN_LINK="$BIN_DIR/urlbridge-host"

cp "$SOURCE_BINARY" "$INSTALLED_BINARY"
chmod +x "$INSTALLED_BINARY"

if [[ -f "$CONFIG_PATH" ]]; then
  if [[ "$LISTEN_SET" -eq 0 ]]; then
    EXISTING_LISTEN="$(read_yaml_value "listen_addr" "$CONFIG_PATH" || true)"
    if [[ -n "$EXISTING_LISTEN" ]]; then
      LISTEN_ADDR="$EXISTING_LISTEN"
    fi
  fi

  if [[ "$TOKEN_SET" -eq 0 ]]; then
    EXISTING_TOKEN="$(read_yaml_value "token" "$CONFIG_PATH" || true)"
    if [[ -n "$EXISTING_TOKEN" ]]; then
      TOKEN="$EXISTING_TOKEN"
    fi
  fi

  EXISTING_DISCOVERY="$(read_yaml_value "discovery" "$CONFIG_PATH" || true)"
  if [[ -n "$EXISTING_DISCOVERY" ]]; then
    DISCOVERY="$(normalize_bool "$EXISTING_DISCOVERY")"
  fi

  if read_yaml_value "log_path" "$CONFIG_PATH" >/dev/null; then
    LOG_PATH="$(read_yaml_value "log_path" "$CONFIG_PATH")"
    LOG_PATH_PRESENT=1
  fi

  if read_yaml_value "log_full_urls" "$CONFIG_PATH" >/dev/null; then
    LOG_FULL_URLS="$(normalize_bool "$(read_yaml_value "log_full_urls" "$CONFIG_PATH")")"
    LOG_FULL_URLS_PRESENT=1
  fi
fi

if [[ -z "$TOKEN" ]]; then
  TOKEN="$("$INSTALLED_BINARY" token)"
fi

mkdir -p "$(dirname "$CONFIG_PATH")"

LISTEN_ADDR_YAML="$(yaml_quote "$LISTEN_ADDR")"
TOKEN_YAML="$(yaml_quote "$TOKEN")"

cat >"$CONFIG_PATH" <<EOF
listen_addr: '$LISTEN_ADDR_YAML'
token: '$TOKEN_YAML'
discovery: $DISCOVERY
EOF

if [[ "$LOG_PATH_PRESENT" -eq 1 ]]; then
  LOG_PATH_YAML="$(yaml_quote "$LOG_PATH")"
  printf "log_path: '%s'\n" "$LOG_PATH_YAML" >>"$CONFIG_PATH"
fi

if [[ "$LOG_FULL_URLS_PRESENT" -eq 1 ]]; then
  printf "log_full_urls: %s\n" "$LOG_FULL_URLS" >>"$CONFIG_PATH"
fi

INSTALLED_BINARY_Q="$(printf '%q' "$INSTALLED_BINARY")"
CONFIG_PATH_Q="$(printf '%q' "$CONFIG_PATH")"

cat >"$RUNNER_SCRIPT" <<EOF
#!/usr/bin/env bash
set -euo pipefail
exec $INSTALLED_BINARY_Q --config $CONFIG_PATH_Q
EOF
chmod +x "$RUNNER_SCRIPT"
ln -sf "$RUNNER_SCRIPT" "$BIN_LINK"

AUTOSTART_STATUS="disabled"
STARTED_NOW=0

case "$(uname -s)" in
  Linux)
    if [[ "$NO_AUTOSTART" -eq 0 ]]; then
      SERVICE_DIR="${HOME}/.config/systemd/user"
      SERVICE_FILE="$SERVICE_DIR/urlbridge-host.service"
      mkdir -p "$SERVICE_DIR"

      cat >"$SERVICE_FILE" <<EOF
[Unit]
Description=URL Bridge Host Service

[Service]
ExecStart=$RUNNER_SCRIPT
Restart=on-failure
RestartSec=2

[Install]
WantedBy=default.target
EOF

      if command -v systemctl >/dev/null 2>&1 && systemctl --user show-environment >/dev/null 2>&1; then
        systemctl --user daemon-reload
        systemctl --user enable --now urlbridge-host.service >/dev/null
        AUTOSTART_STATUS="systemd user service enabled"
      else
        start_for_current_session "$RUNNER_SCRIPT"
        STARTED_NOW=1
        AUTOSTART_STATUS="systemd user session unavailable; started only for the current session"
      fi
    else
      start_for_current_session "$RUNNER_SCRIPT"
      STARTED_NOW=1
      AUTOSTART_STATUS="started for the current session only"
    fi
    ;;
  Darwin)
    if [[ "$NO_AUTOSTART" -eq 0 ]]; then
      LAUNCH_DIR="${HOME}/Library/LaunchAgents"
      PLIST_FILE="$LAUNCH_DIR/com.xmoon.urlbridge.host.plist"
      mkdir -p "$LAUNCH_DIR"

      cat >"$PLIST_FILE" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>com.xmoon.urlbridge.host</string>
  <key>ProgramArguments</key>
  <array>
    <string>$RUNNER_SCRIPT</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
</dict>
</plist>
EOF

      launchctl unload "$PLIST_FILE" >/dev/null 2>&1 || true
      launchctl load "$PLIST_FILE" >/dev/null
      AUTOSTART_STATUS="LaunchAgent installed"
    else
      start_for_current_session "$RUNNER_SCRIPT"
      STARTED_NOW=1
      AUTOSTART_STATUS="started for the current session only"
    fi
    ;;
esac

echo "URL Bridge host installed."
echo "Installed binary: $INSTALLED_BINARY"
echo "Runner script: $RUNNER_SCRIPT"
echo "Config file: $CONFIG_PATH"
echo "Listen address: $LISTEN_ADDR"
echo "Token: $TOKEN"
echo "Autostart: $AUTOSTART_STATUS"
