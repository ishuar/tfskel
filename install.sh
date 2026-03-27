#!/usr/bin/env bash
set -euo pipefail

# ─────────────────────────────────────────────
# tfskel installer — https://github.com/ishuar/tfskel
#
# Usage:
#   curl -fsSL <raw-url>/install.sh | bash
#   curl -fsSL <raw-url>/install.sh | TFSKEL_VERSION=0.7.0 bash
#   INSTALL_DIR=~/.local/bin bash install.sh
#
# Prefer downloading and inspecting before running:
#   curl -fsSL <raw-url>/install.sh -o install.sh
#   less install.sh
#   TFSKEL_VERSION=0.7.0 bash install.sh
#
# Environment variables:
#   TFSKEL_VERSION  — version to install (default: latest release tag)
#   INSTALL_DIR     — install destination  (default: ~/.local/bin)
#   TFSKEL_DEBUG    — set to 1 to enable bash debug tracing (set -x)
# ─────────────────────────────────────────────

if [[ "${TFSKEL_DEBUG:-0}" == "1" ]]; then
  set -x
fi

REPO="ishuar/tfskel"
INSTALL_DIR="${INSTALL_DIR:-${HOME}/.local/bin}"

# ── Colors ───────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
info()  { printf "${GREEN}[✓]${NC}  %s\n" "$*" >&2; }
warn()  { printf "${YELLOW}[!]${NC}  %s\n" "$*" >&2; }
error() { printf "${RED}[✗]${NC}  %s\n" "$*" >&2; exit 1; }

# ── Resolve version ───────────────────────────
resolve_version() {
  local v="${TFSKEL_VERSION:-}"

  if [[ -z "$v" ]]; then
    info "Fetching latest release tag..."
    v="$(curl -fsSL -o /dev/null -w '%{url_effective}' \
      "https://github.com/${REPO}/releases/latest" | grep -oE '[^/]+$')"
    [[ -n "$v" ]] || error "Could not determine latest release version."
  fi

  v="${v#v}"
  [[ "$v" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] \
    || error "Invalid version: '${v}'. Expected format: MAJOR.MINOR.PATCH (e.g. 0.7.0 or v0.7.0)"
  echo "$v"
}

# ── Detect OS / Arch ─────────────────────────
detect_os() {
  case "$(uname -s)" in
    Darwin)  echo "Darwin" ;;
    Linux)   echo "Linux"  ;;
    *) error "Unsupported OS: $(uname -s)" ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64)   echo "x86_64" ;;
    arm64|aarch64)  echo "arm64"  ;;
    *) error "Unsupported architecture: $(uname -m)" ;;
  esac
}

# ── Checksum verification ────────────────────
verify_checksum() {
  local file="$1" checksums="$2"
  local expected actual

  local name
  name="$(basename "$file")"
  expected="$(grep -F "$name" "$checksums" | grep -w "$name" | awk '{print $1}')"
  if [[ -z "$expected" ]]; then
    error "Checksum entry not found for ${name} — aborting."
  fi
  if [[ "$(echo "$expected" | wc -l)" -gt 1 ]]; then
    error "Multiple checksum entries found for ${name} — aborting."
  fi

  if command -v sha256sum &>/dev/null; then
    actual="$(sha256sum "$file" | awk '{print $1}')"
  elif command -v shasum &>/dev/null; then
    actual="$(shasum -a 256 "$file" | awk '{print $1}')"
  else
    error "sha256sum or shasum is required for checksum verification."
  fi

  [[ "$expected" == "$actual" ]] \
    || error "Checksum mismatch!\n  Expected : ${expected}\n  Got      : ${actual}"
  info "Checksum verified."
}

# ── Main ──────────────────────────────────────
main() {
  command -v curl &>/dev/null || error "curl is required but not found."
  command -v tar  &>/dev/null || error "tar is required but not found."

  local version os arch
  version="$(resolve_version)"
  os="$(detect_os)"
  arch="$(detect_arch)"

  local filename="tfskel_${version}_${os}_${arch}.tar.gz"
  local base_url="https://github.com/${REPO}/releases/download/v${version}"
  TMP="$(mktemp -d)"
  trap 'rm -rf "$TMP"' EXIT

  info "Installing tfskel v${version} (${os}/${arch})"

  curl -fsSL "${base_url}/${filename}" -o "${TMP}/${filename}"
  curl -fsSL "${base_url}/checksums.txt" -o "${TMP}/checksums.txt"
  verify_checksum "${TMP}/${filename}" "${TMP}/checksums.txt"

  tar -xzf "${TMP}/${filename}" -C "${TMP}" tfskel

  local dest="${INSTALL_DIR}/tfskel"
  if [[ -d "$INSTALL_DIR" && -w "$INSTALL_DIR" ]]; then
    mv "${TMP}/tfskel" "$dest"
  elif [[ -d "$INSTALL_DIR" ]]; then
    info "Installing to ${dest} (requires sudo)..."
    sudo mv "${TMP}/tfskel" "$dest"
  elif mkdir -p "$INSTALL_DIR" 2>/dev/null; then
    mv "${TMP}/tfskel" "$dest"
  else
    info "Creating ${INSTALL_DIR} and installing (requires sudo)..."
    sudo mkdir -p "$INSTALL_DIR"
    sudo mv "${TMP}/tfskel" "$dest"
  fi

  if [[ -x "$dest" ]]; then
    info "Installed $("$dest" --version) to ${dest}"
    if ! command -v tfskel &>/dev/null; then
      warn "tfskel is not in your PATH. Run: export PATH=\"${INSTALL_DIR}:\$PATH\""
    fi
  else
    error "Installation failed — ${dest} is not executable."
  fi
}

main "$@"
