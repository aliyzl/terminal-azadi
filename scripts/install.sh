#!/bin/sh
# Install script for azad - Beautiful terminal VPN client
# Usage: curl -sSL https://raw.githubusercontent.com/aliyzl/terminal-azadi/master/scripts/install.sh | sh
set -e

REPO="aliyzl/terminal-azadi"
BINARY="azad"

# --- OS Detection ---
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  darwin|linux) ;;
  *) echo "Error: Unsupported OS: $OS (only darwin and linux are supported)" >&2; exit 1 ;;
esac

# --- Arch Detection ---
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  arm64)   ARCH="arm64" ;;
  *) echo "Error: Unsupported architecture: $ARCH (only amd64 and arm64 are supported)" >&2; exit 1 ;;
esac

echo "Detected platform: ${OS}/${ARCH}"

# --- Version Detection ---
VERSION=$(curl -sSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
if [ -z "$VERSION" ]; then
  echo "Error: Failed to detect latest version from GitHub API" >&2
  exit 1
fi
VERSION_NUM="${VERSION#v}"
echo "Latest version: ${VERSION}"

# --- Download ---
FILENAME="${BINARY}_${VERSION_NUM}_${OS}_${ARCH}.tar.gz"
CHECKSUMS="checksums.txt"
DOWNLOAD_BASE="https://github.com/${REPO}/releases/download/${VERSION}"

echo "Downloading ${FILENAME}..."
curl -sSL -o "/tmp/${FILENAME}" "${DOWNLOAD_BASE}/${FILENAME}"
curl -sSL -o "/tmp/${CHECKSUMS}" "${DOWNLOAD_BASE}/${CHECKSUMS}"

# --- Checksum Verification ---
EXPECTED=$(grep "${FILENAME}" "/tmp/${CHECKSUMS}" | awk '{print $1}')
if [ -z "$EXPECTED" ]; then
  echo "Error: Could not find checksum for ${FILENAME} in checksums.txt" >&2
  exit 1
fi

if [ "$OS" = "darwin" ]; then
  ACTUAL=$(shasum -a 256 "/tmp/${FILENAME}" | awk '{print $1}')
else
  ACTUAL=$(sha256sum "/tmp/${FILENAME}" | awk '{print $1}')
fi

if [ "$EXPECTED" != "$ACTUAL" ]; then
  echo "Error: Checksum verification failed" >&2
  echo "  Expected: ${EXPECTED}" >&2
  echo "  Actual:   ${ACTUAL}" >&2
  exit 1
fi
echo "Checksum verified"

# --- Installation ---
tar -xzf "/tmp/${FILENAME}" -C /tmp "${BINARY}"

INSTALL_DIR=""
warn_path=0

if [ -d /usr/local/bin ] && [ -w /usr/local/bin ]; then
  INSTALL_DIR="/usr/local/bin"
  install -m 755 "/tmp/${BINARY}" "${INSTALL_DIR}/"
elif command -v sudo >/dev/null 2>&1; then
  INSTALL_DIR="/usr/local/bin"
  echo "Installing to ${INSTALL_DIR} (requires sudo)..."
  sudo install -m 755 "/tmp/${BINARY}" "${INSTALL_DIR}/"
else
  INSTALL_DIR="$HOME/.local/bin"
  mkdir -p "$INSTALL_DIR"
  install -m 755 "/tmp/${BINARY}" "${INSTALL_DIR}/"
fi

# Check if install dir is in PATH
case ":$PATH:" in
  *":${INSTALL_DIR}:"*) ;;
  *) warn_path=1 ;;
esac

# --- Cleanup ---
rm -f "/tmp/${FILENAME}" "/tmp/${CHECKSUMS}" "/tmp/${BINARY}"

# --- Success ---
echo ""
echo "Successfully installed ${BINARY} ${VERSION} to ${INSTALL_DIR}/${BINARY}"
if [ "$warn_path" = "1" ]; then
  echo ""
  echo "NOTE: ${INSTALL_DIR} is not in your PATH. Add it with:"
  echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
  echo ""
  echo "Add the line above to your shell profile (~/.bashrc, ~/.zshrc, etc.) to make it permanent."
fi
echo ""
echo "Run '${BINARY} --help' to get started"
