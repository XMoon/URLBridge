#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

usage() {
  cat <<'EOF'
URL Bridge Linux guest installer

Usage:
  ./install-guest.sh [--config PATH] [--host-url http://HOST:38495] [--token TOKEN] [--timeout SECONDS]
EOF
}

detect_binary_name() {
  local os arch
  os="$(uname -s)"
  arch="$(uname -m)"

  case "$os/$arch" in
    Linux/x86_64)
      echo "urlbridge-guestctl-linux-amd64"
      ;;
    Linux/aarch64|Linux/arm64)
      echo "urlbridge-guestctl-linux-arm64"
      ;;
    *)
      echo "Unsupported guest platform: $os/$arch" >&2
      exit 1
      ;;
  esac
}

find_guestctl() {
  local binary_name="$1"
  local candidates=(
    "$SCRIPT_DIR/urlbridge-guestctl"
    "$SCRIPT_DIR/$binary_name"
    "$ROOT_DIR/urlbridge-guestctl"
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

case "${1:-}" in
  -h|--help|help)
    usage
    exit 0
    ;;
esac

BINARY_NAME="$(detect_binary_name)"
GUESTCTL="$(find_guestctl "$BINARY_NAME")"

exec "$GUESTCTL" install "$@"
