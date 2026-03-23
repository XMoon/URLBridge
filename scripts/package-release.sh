#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
DIST_DIR="$ROOT_DIR/dist"
README_FILE="$ROOT_DIR/README.md"
PACKAGE_OUTPUT_DIR="${PACKAGE_OUTPUT_DIR:-}"

if [[ -z "$PACKAGE_OUTPUT_DIR" ]]; then
  echo "PACKAGE_OUTPUT_DIR must be set. This script is intended to run from CI." >&2
  exit 1
fi

PACKAGE_DIR="$PACKAGE_OUTPUT_DIR"

require_file() {
  local path="$1"
  if [[ ! -f "$path" ]]; then
    echo "Missing required file: $path" >&2
    exit 1
  fi
}

create_zip() {
  local source_dir="$1"
  local output_file="$2"

  if command -v zip >/dev/null 2>&1; then
    (cd "$source_dir" && zip -qr "$output_file" .)
    return
  fi

  python3 - "$source_dir" "$output_file" <<'PY'
import os
import sys
import zipfile

source_dir, output_file = sys.argv[1], sys.argv[2]

with zipfile.ZipFile(output_file, "w", zipfile.ZIP_DEFLATED) as archive:
    for root, _, files in os.walk(source_dir):
        for name in files:
            path = os.path.join(root, name)
            arcname = os.path.relpath(path, source_dir)
            archive.write(path, arcname)
PY
}

stage_bundle() {
  local bundle_name="$1"
  local archive_kind="$2"
  shift 2

  local stage_dir
  stage_dir="$(mktemp -d)"
  mkdir -p "$stage_dir/$bundle_name"

  while [[ $# -gt 0 ]]; do
    local source="$1"
    local target_name="$2"
    shift 2

    require_file "$source"
    cp "$source" "$stage_dir/$bundle_name/$target_name"
  done

  if [[ "$archive_kind" == "tar.gz" ]]; then
    tar -C "$stage_dir" -czf "$PACKAGE_DIR/$bundle_name.tar.gz" "$bundle_name"
  else
    create_zip "$stage_dir/$bundle_name" "$PACKAGE_DIR/$bundle_name.zip"
  fi

  rm -rf "$stage_dir"
}

mkdir -p "$PACKAGE_DIR"

stage_bundle \
  "urlbridge-host-linux-amd64" "tar.gz" \
  "$DIST_DIR/urlbridge-host-linux-amd64" "urlbridge-host" \
  "$SCRIPT_DIR/install-host.sh" "install-host.sh" \
  "$README_FILE" "README.md"

stage_bundle \
  "urlbridge-host-linux-arm64" "tar.gz" \
  "$DIST_DIR/urlbridge-host-linux-arm64" "urlbridge-host" \
  "$SCRIPT_DIR/install-host.sh" "install-host.sh" \
  "$README_FILE" "README.md"

stage_bundle \
  "urlbridge-host-darwin-amd64" "tar.gz" \
  "$DIST_DIR/urlbridge-host-darwin-amd64" "urlbridge-host" \
  "$SCRIPT_DIR/install-host.sh" "install-host.sh" \
  "$README_FILE" "README.md"

stage_bundle \
  "urlbridge-host-darwin-arm64" "tar.gz" \
  "$DIST_DIR/urlbridge-host-darwin-arm64" "urlbridge-host" \
  "$SCRIPT_DIR/install-host.sh" "install-host.sh" \
  "$README_FILE" "README.md"

stage_bundle \
  "urlbridge-host-windows-amd64" "zip" \
  "$DIST_DIR/urlbridge-host-windows-amd64.exe" "urlbridge-host.exe" \
  "$SCRIPT_DIR/install-host.ps1" "install-host.ps1" \
  "$README_FILE" "README.md"

stage_bundle \
  "urlbridge-host-windows-arm64" "zip" \
  "$DIST_DIR/urlbridge-host-windows-arm64.exe" "urlbridge-host.exe" \
  "$SCRIPT_DIR/install-host.ps1" "install-host.ps1" \
  "$README_FILE" "README.md"

stage_bundle \
  "urlbridge-guest-windows-amd64" "zip" \
  "$DIST_DIR/urlbridge-browser.exe" "urlbridge-browser.exe" \
  "$DIST_DIR/urlbridge-guestctl.exe" "urlbridge-guestctl.exe" \
  "$SCRIPT_DIR/install-guest.ps1" "install-guest.ps1" \
  "$README_FILE" "README.md"

stage_bundle \
  "urlbridge-guest-windows-arm64" "zip" \
  "$DIST_DIR/urlbridge-browser-arm64.exe" "urlbridge-browser.exe" \
  "$DIST_DIR/urlbridge-guestctl-arm64.exe" "urlbridge-guestctl.exe" \
  "$SCRIPT_DIR/install-guest.ps1" "install-guest.ps1" \
  "$README_FILE" "README.md"

echo "Created release bundles under $PACKAGE_DIR"
