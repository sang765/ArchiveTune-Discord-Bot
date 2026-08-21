#!/usr/bin/env bash
set -Eeuo pipefail

cd "$(dirname "$0")"

CONFIG_FILE_PATH="${CONFIG_FILE:-./config.yaml}"
BOT_BINARY="${BOT_BINARY:-./discord-forum-bot}"
GO_PACKAGE_PATH="${GO_PACKAGE:-./cmd/bot}"

if command -v go >/dev/null 2>&1 && [[ -f go.mod ]]; then
  echo "[startup] Go detected; building ${BOT_BINARY} from ${GO_PACKAGE_PATH}..."
  GOTOOLCHAIN=local CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags='-s -w' \
    -o "${BOT_BINARY}.tmp" \
    "${GO_PACKAGE_PATH}"
  mv -f "${BOT_BINARY}.tmp" "${BOT_BINARY}"
  chmod 755 "${BOT_BINARY}" 2>/dev/null || true
else
  echo "[startup] Go compiler not found; using the existing binary ${BOT_BINARY}."
fi

if [[ ! -f "${BOT_BINARY}" ]]; then
  echo "[startup] ERROR: ${BOT_BINARY} does not exist."
  echo "[startup] Upload a prebuilt Linux binary or use a container image with Go installed."
  exit 1
fi

if [[ ! -x "${BOT_BINARY}" ]]; then
  echo "[startup] ERROR: ${BOT_BINARY} is not executable."
  echo "[startup] Ask the panel administrator to set its mode to 755."
  exit 1
fi

exec env CONFIG_FILE="${CONFIG_FILE_PATH}" "${BOT_BINARY}"
