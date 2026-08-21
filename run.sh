#!/usr/bin/env bash
set -Eeuo pipefail

cd "$(dirname "$0")"

log() {
  echo "[startup] $*"
}

apply_zip_update() {
  if [[ "${AUTO_UPDATE_ZIP:-1}" != "1" ]]; then
    return 0
  fi

  shopt -s nullglob
  local zip_files=(./*.zip)
  shopt -u nullglob
  if (( ${#zip_files[@]} == 0 )); then
    return 0
  fi

  if ! command -v unzip >/dev/null 2>&1; then
    log "ZIP archive detected, but unzip is unavailable; continuing without an update."
    return 0
  fi

  local update_zip="${zip_files[0]}"
  local candidate
  for candidate in "${zip_files[@]}"; do
    if [[ "$candidate" -nt "$update_zip" ]]; then
      update_zip="$candidate"
    fi
  done

  log "ZIP update detected: ${update_zip}"

  local stage
  mkdir -p .tools
  stage="$(mktemp -d .tools/.zip-update.XXXXXX)"

  if ! unzip -tqq "$update_zip"; then
    log "ZIP validation failed; keeping the current source unchanged."
    rm -rf -- "$stage"
    return 0
  fi

  if ! unzip -Z1 "$update_zip" | awk '
    {
      gsub(/\\/, "/", $0)
      if ($0 ~ /^\// || $0 ~ /^[A-Za-z]:\//) bad=1
      count=split($0, parts, "/")
      for (i=1; i<=count; i++) if (parts[i] == "..") bad=1
    }
    END { exit bad }
  '; then
    log "ZIP contains an unsafe path; keeping the current source unchanged."
    rm -rf -- "$stage"
    return 0
  fi

  if ! unzip -q "$update_zip" -d "$stage"; then
    log "ZIP extraction failed; keeping the current source unchanged."
    rm -rf -- "$stage"
    return 0
  fi

  local source_root="$stage"
  local top_entries=()
  mapfile -t top_entries < <(find "$stage" -mindepth 1 -maxdepth 1 -printf '%f\n' | sort)
  if (( ${#top_entries[@]} == 1 )) && [[ -d "$stage/${top_entries[0]}" ]]; then
    source_root="$stage/${top_entries[0]}"
  fi

  if [[ ! -f "$source_root/go.mod" && ! -f "$source_root/discord-forum-bot" && ! -f "$source_root/run.sh" ]]; then
    log "ZIP does not look like a bot source or binary archive; keeping the current source unchanged."
    rm -rf -- "$stage"
    return 0
  fi

  # Always preserve the live configuration and the cached local Go toolchain.
  rm -f -- "$source_root/config.yaml"
  rm -rf -- "$source_root/.tools"

  log "Replacing source files while preserving run.sh, config.yaml, and .tools..."
  find . -mindepth 1 -maxdepth 1 \
    ! -name 'run.sh' \
    ! -name 'config.yaml' \
    ! -name '.tools' \
    -exec rm -rf -- {} +
  cp -a "$source_root"/. .

  # Remove every root ZIP after a successful replacement.
  find . -maxdepth 1 -type f -iname '*.zip' -delete
  chmod 755 run.sh 2>/dev/null || true
  [[ -f install-go.sh ]] && chmod 755 install-go.sh 2>/dev/null || true
  [[ -f discord-forum-bot ]] && chmod 755 discord-forum-bot 2>/dev/null || true

  rm -rf -- "$stage"
  log "ZIP update applied successfully."
}

apply_zip_update

CONFIG_FILE_PATH="${CONFIG_FILE:-./config.yaml}"
BOT_BINARY="${BOT_BINARY:-./discord-forum-bot}"
GO_PACKAGE_PATH="${GO_PACKAGE:-./cmd/bot}"

GO_BIN=""
if command -v go >/dev/null 2>&1; then
  GO_BIN="$(command -v go)"
elif [[ -x .tools/go/bin/go ]]; then
  GO_BIN=".tools/go/bin/go"
elif [[ -x ./install-go.sh ]]; then
  log "Go compiler not found; installing a local Go toolchain..."
  GO_BIN="$(./install-go.sh)"
fi

if [[ -n "${GO_BIN}" && -f go.mod ]]; then
  log "Building ${BOT_BINARY} from ${GO_PACKAGE_PATH} with ${GO_BIN}..."
  GOTOOLCHAIN=local CGO_ENABLED=0 "${GO_BIN}" build \
    -trimpath \
    -ldflags='-s -w' \
    -o "${BOT_BINARY}.tmp" \
    "${GO_PACKAGE_PATH}"
  mv -f "${BOT_BINARY}.tmp" "${BOT_BINARY}"
  chmod 755 "${BOT_BINARY}" 2>/dev/null || true
else
  log "Go compiler unavailable; using the existing binary ${BOT_BINARY}."
fi

if [[ ! -f "${BOT_BINARY}" ]]; then
  log "ERROR: ${BOT_BINARY} does not exist."
  log "Upload a prebuilt Linux binary or include install-go.sh in the container."
  exit 1
fi

if [[ ! -x "${BOT_BINARY}" ]]; then
  log "ERROR: ${BOT_BINARY} is not executable."
  log "Ask the panel administrator to set its mode to 755."
  exit 1
fi

exec env CONFIG_FILE="${CONFIG_FILE_PATH}" "${BOT_BINARY}"
