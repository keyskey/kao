#!/usr/bin/env sh

set -eu

REPO="keyskey/kao"
BINARY_NAME="${BINARY_NAME:-kao}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "required command not found: $1" >&2
    exit 1
  fi
}

detect_os() {
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  case "$os" in
    darwin|linux) echo "$os" ;;
    *)
      echo "unsupported OS: $os (supported: darwin, linux)" >&2
      exit 1
      ;;
  esac
}

detect_arch() {
  arch="$(uname -m)"
  case "$arch" in
    x86_64|amd64) echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *)
      echo "unsupported architecture: $arch (supported: amd64, arm64)" >&2
      exit 1
      ;;
  esac
}

resolve_version() {
  if [ -n "${VERSION:-}" ]; then
    echo "$VERSION"
    return
  fi

  need_cmd curl
  version="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' | sed -n '1p')"
  if [ -z "$version" ]; then
    echo "could not resolve latest release version; set VERSION=vX.Y.Z and retry" >&2
    exit 1
  fi
  echo "$version"
}

install_binary() {
  need_cmd curl
  need_cmd tar
  need_cmd mktemp

  os="$(detect_os)"
  arch="$(detect_arch)"
  version="$(resolve_version)"
  version_no_v="${version#v}"
  archive="${BINARY_NAME}_${version_no_v}_${os}_${arch}.tar.gz"
  download_url="https://github.com/${REPO}/releases/download/${version}/${archive}"

  tmp_dir="$(mktemp -d)"
  trap 'rm -rf "$tmp_dir"' EXIT INT TERM

  echo "downloading ${download_url}"
  curl -fL -o "${tmp_dir}/${archive}" "$download_url"
  tar -xzf "${tmp_dir}/${archive}" -C "$tmp_dir"

  if [ ! -f "${tmp_dir}/${BINARY_NAME}" ]; then
    echo "binary not found in archive: ${BINARY_NAME}" >&2
    exit 1
  fi

  mkdir -p "$INSTALL_DIR"
  if command -v install >/dev/null 2>&1; then
    install -m 0755 "${tmp_dir}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
  else
    cp "${tmp_dir}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
    chmod 0755 "${INSTALL_DIR}/${BINARY_NAME}"
  fi

  echo "installed ${BINARY_NAME} to ${INSTALL_DIR}/${BINARY_NAME}"
  echo "run: ${BINARY_NAME} --help"
}

install_binary
