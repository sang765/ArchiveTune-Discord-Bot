#!/usr/bin/env bash
set -Eeuo pipefail

GO_VERSION="${GO_VERSION:-1.22.2}"
GO_ROOT="${GO_ROOT:-.tools/go}"
ARCH="${GO_ARCH:-amd64}"
PLATFORM="linux-${ARCH}"
ARCHIVE="go${GO_VERSION}.${PLATFORM}.tar.gz"
DOWNLOAD_URL="https://go.dev/dl/${ARCHIVE}"

log() {
  echo "[go-installer] $*" >&2
}

cd "$(dirname "$0")"
mkdir -p "$(dirname "$GO_ROOT")"

if [[ -x "${GO_ROOT}/bin/go" ]]; then
  printf '%s\n' "${GO_ROOT}/bin/go"
  exit 0
fi

if ! command -v curl >/dev/null 2>&1 && ! command -v wget >/dev/null 2>&1; then
  log "ERROR: curl or wget is required to download Go."
  exit 1
fi
if ! command -v tar >/dev/null 2>&1; then
  log "ERROR: tar is required to extract Go."
  exit 1
fi

TMP_DIR="$(mktemp -d .go-install.XXXXXX)"
cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

log "Downloading Go ${GO_VERSION} for ${PLATFORM}..."
if command -v curl >/dev/null 2>&1; then
  curl --fail --location --retry 3 --silent --show-error "$DOWNLOAD_URL" -o "${TMP_DIR}/${ARCHIVE}"
else
  wget --https-only --tries=3 --quiet "$DOWNLOAD_URL" -O "${TMP_DIR}/${ARCHIVE}"
fi

log "Extracting Go into ${GO_ROOT}..."
rm -rf "$GO_ROOT"
mkdir -p "$GO_ROOT"
tar -xzf "${TMP_DIR}/${ARCHIVE}" -C "$TMP_DIR"
# The official archive contains a top-level directory named go.
mv "${TMP_DIR}/go"/* "$GO_ROOT"/
rmdir "${TMP_DIR}/go"

if [[ ! -x "${GO_ROOT}/bin/go" ]]; then
  log "ERROR: Go installation did not produce ${GO_ROOT}/bin/go."
  exit 1
fi

log "Go installed at ${GO_ROOT}/bin/go."
printf '%s\n' "${GO_ROOT}/bin/go"
