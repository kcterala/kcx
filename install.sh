#!/bin/sh

set -e

KCX_VERSION=${KCX_VERSION:-"v1.0.0"}
INSTALL_DIR=${KCX_INSTALL_DIR:-"/usr/local/bin"}
REPO_URL="https://github.com/kcterala/kcx/releases/download"

detect_platform() {
  OS="$(uname | tr '[:upper:]' '[:lower:]')"
  ARCH="$(uname -m)"

  case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
  esac

  if [ "$OS" = "mingw64_nt" ] || [ "$OS" = "cygwin" ]; then
    OS="windows"
  fi

  echo "${OS}-${ARCH}"
}

download_binary() {
  PLATFORM=$(detect_platform)
  BINARY_NAME="kcx"
  [ "$PLATFORM" = "windows-amd64" ] && BINARY_NAME="kcx.exe"

  URL="${REPO_URL}/${KCX_VERSION}/kcx-${PLATFORM}"
  echo "Downloading $URL"

  TMP_FILE=$(mktemp)
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$URL" -o "$TMP_FILE"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$TMP_FILE" "$URL"
  else
    echo "Error: Neither curl nor wget is installed."; exit 1
  fi

  chmod +x "$TMP_FILE"
  sudo mv "$TMP_FILE" "$INSTALL_DIR/$BINARY_NAME"

  echo "✅ kcx installed to $INSTALL_DIR/$BINARY_NAME"
}

download_binary
