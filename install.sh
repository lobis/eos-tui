#!/usr/bin/env bash
set -euo pipefail

REPO="lobis/eos-tui"
BIN_NAME="eos-tui"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# ── platform detection ────────────────────────────────────────────────────────

OS="$(uname -s)"
ARCH="$(uname -m)"

case "${OS}" in
  Linux)  os="linux" ;;
  Darwin) os="macos" ;;
  *)
    echo "Unsupported OS: ${OS}" >&2
    exit 1
    ;;
esac

case "${ARCH}" in
  x86_64|amd64) arch="amd64" ;;
  aarch64|arm64) arch="arm64" ;;
  *)
    echo "Unsupported architecture: ${ARCH}" >&2
    exit 1
    ;;
esac

PLATFORM="${os}_${arch}"

# ── resolve latest release tag ────────────────────────────────────────────────

API="https://api.github.com/repos/${REPO}/releases/latest"
TAG="$(curl -fsSL "${API}" | grep '"tag_name"' | head -1 | cut -d'"' -f4)"

if [ -z "${TAG}" ]; then
  echo "Could not determine latest release tag." >&2
  exit 1
fi

BASE_URL="https://github.com/${REPO}/releases/download/${TAG}"
BINARY="${BIN_NAME}_${TAG}_${PLATFORM}"

# ── download binary + checksum ────────────────────────────────────────────────

TMP="$(mktemp -d)"
trap 'rm -rf "${TMP}"' EXIT

echo "Downloading ${BIN_NAME} ${TAG} (${PLATFORM})..."
curl -fsSL --progress-bar "${BASE_URL}/${BINARY}" -o "${TMP}/${BIN_NAME}"
curl -fsSL "${BASE_URL}/SHA256SUMS.txt" -o "${TMP}/SHA256SUMS.txt"

# ── verify checksum ───────────────────────────────────────────────────────────

EXPECTED="$(grep "${BINARY}" "${TMP}/SHA256SUMS.txt" | awk '{print $1}')"

if [ -z "${EXPECTED}" ]; then
  echo "No checksum found for ${BINARY} in SHA256SUMS.txt." >&2
  exit 1
fi

if command -v sha256sum &>/dev/null; then
  ACTUAL="$(sha256sum "${TMP}/${BIN_NAME}" | awk '{print $1}')"
elif command -v shasum &>/dev/null; then
  ACTUAL="$(shasum -a 256 "${TMP}/${BIN_NAME}" | awk '{print $1}')"
else
  echo "Warning: no sha256sum or shasum found, skipping checksum verification." >&2
  ACTUAL="${EXPECTED}"
fi

if [ "${ACTUAL}" != "${EXPECTED}" ]; then
  echo "Checksum mismatch!" >&2
  echo "  expected: ${EXPECTED}" >&2
  echo "  got:      ${ACTUAL}" >&2
  exit 1
fi

# ── install ───────────────────────────────────────────────────────────────────

chmod +x "${TMP}/${BIN_NAME}"

if [ -w "${INSTALL_DIR}" ]; then
  mv "${TMP}/${BIN_NAME}" "${INSTALL_DIR}/${BIN_NAME}"
else
  echo "Installing to ${INSTALL_DIR} (requires sudo)..."
  sudo mv "${TMP}/${BIN_NAME}" "${INSTALL_DIR}/${BIN_NAME}"
fi

echo "Installed ${BIN_NAME} ${TAG} to ${INSTALL_DIR}/${BIN_NAME}"
"${INSTALL_DIR}/${BIN_NAME}" --version
