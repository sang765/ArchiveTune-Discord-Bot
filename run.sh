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

  # Always preserve the live configuration and all known caches.
  rm -f -- "$source_root/config.yaml"
  rm -rf -- "$source_root/.tools"
  chmod -R u+rwX "$source_root" 2>/dev/null || true

  # Snapshot the old root before the overlay. This allows stale-file cleanup only after
  # the new source is safely present, so a protected cache cannot abort the update.
  local entry base
  local old_entries=()
  shopt -s nullglob dotglob
  old_entries=(./* ./.??*)
  shopt -u dotglob nullglob

  log "Overlaying source while preserving config.yaml and caches..."
  if ! cp -a "$source_root"/. .; then
    log "ZIP overlay failed; keeping the current source unchanged where possible."
    rm -rf -- "$stage"
    return 0
  fi

  # Remove stale entries that were present before the overlay but are absent from the ZIP.
  # Live config, run.sh, and all known caches are always preserved.
  for entry in "${old_entries[@]}"; do
    [[ -e "$entry" ]] || continue
    base="$(basename "$entry")"
    case "$base" in
      .|..|run.sh|config.yaml|.tools|go|.cache)
        continue
        ;;
    esac
    [[ -e "$source_root/$base" ]] && continue
    if ! rm -rf -- "$entry"; then
      log "Warning: could not remove stale entry ${entry}; continuing with the updated source."
    fi
  done

  # Remove every root ZIP after a successful overlay when permissions allow it.
  for entry in ./*.zip; do
    [[ -e "$entry" ]] || continue
    rm -f -- "$entry" 2>/dev/null || log "Warning: could not remove ${entry}."
  done
  chmod 755 run.sh 2>/dev/null || true
  [[ -f install-go.sh ]] && chmod 755 install-go.sh 2>/dev/null || true
  [[ -f install-media-tools.sh ]] && chmod 755 install-media-tools.sh 2>/dev/null || true
  [[ -f discord-forum-bot ]] && chmod 755 discord-forum-bot 2>/dev/null || true

  rm -rf -- "$stage"
  log "ZIP update applied successfully."
}

apply_zip_update

PROJECT_ROOT="$(pwd -P)"
CONFIG_FILE_PATH="${CONFIG_FILE:-${PROJECT_ROOT}/config.yaml}"
BOT_BINARY="${BOT_BINARY:-./discord-forum-bot}"
GO_PACKAGE_PATH="${GO_PACKAGE:-./cmd/bot}"
GO_WORK_ROOT="${GO_WORK_ROOT:-${PROJECT_ROOT}/.tools/go-work}"
if [[ "${GO_WORK_ROOT}" != /* ]]; then
  GO_WORK_ROOT="${PROJECT_ROOT}/${GO_WORK_ROOT#./}"
fi
GO_MODULE_CACHE="${GO_WORK_ROOT}/pkg/mod"
GO_BUILD_CACHE="${GO_WORK_ROOT}/build-cache"
BUILD_FINGERPRINT_FILE="${PROJECT_ROOT}/.tools/.discord-forum-bot-build-fingerprint"

source_fingerprint() {
  if ! command -v sha256sum >/dev/null 2>&1; then
    printf '%s\n' "no-sha256sum"
    return 0
  fi
  {
    [[ -f go.mod ]] && sha256sum go.mod
    [[ -f go.sum ]] && sha256sum go.sum
    if [[ -d cmd ]]; then
      find cmd -type f -print0 | sort -z | while IFS= read -r -d '' file; do sha256sum "$file"; done
    fi
    if [[ -d internal ]]; then
      find internal -type f -print0 | sort -z | while IFS= read -r -d '' file; do sha256sum "$file"; done
    fi
  } | sha256sum | awk '{print $1}'
}

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
  mkdir -p "${GO_MODULE_CACHE}" "${GO_BUILD_CACHE}"
  current_fingerprint="$(source_fingerprint)"
  previous_fingerprint=""
  [[ -f "${BUILD_FINGERPRINT_FILE}" ]] && previous_fingerprint="$(cat "${BUILD_FINGERPRINT_FILE}")"

  if [[ -x "${BOT_BINARY}" && "$current_fingerprint" == "$previous_fingerprint" ]]; then
    log "Source unchanged; reusing existing binary ${BOT_BINARY}."
  else
    log "Building ${BOT_BINARY} from ${GO_PACKAGE_PATH} with ${GO_BIN}..."
    GOPATH="${GO_WORK_ROOT}" GOMODCACHE="${GO_MODULE_CACHE}" GOCACHE="${GO_BUILD_CACHE}" \
      GOTOOLCHAIN=local CGO_ENABLED=0 "${GO_BIN}" build \
      -trimpath \
      -ldflags='-s -w' \
      -o "${BOT_BINARY}.tmp" \
      "${GO_PACKAGE_PATH}"
    mv -f "${BOT_BINARY}.tmp" "${BOT_BINARY}"
    chmod 755 "${BOT_BINARY}" 2>/dev/null || true
    printf '%s\n' "$current_fingerprint" > "${BUILD_FINGERPRINT_FILE}"
    log "Build completed; dependency and build caches are stored under ${GO_WORK_ROOT}."
  fi
else
  log "Go compiler unavailable; using the existing binary ${BOT_BINARY}."
fi

MEDIA_ROOT="${MEDIA_TOOLS_ROOT:-${PROJECT_ROOT}/.tools/media}"
YTDLP_BIN="${YTDLP_BIN:-}"
FFMPEG_BIN="${FFMPEG_BIN:-}"
if [[ -z "${YTDLP_BIN}" ]] && command -v yt-dlp >/dev/null 2>&1; then
  YTDLP_BIN="$(command -v yt-dlp)"
fi
if [[ -z "${FFMPEG_BIN}" ]] && command -v ffmpeg >/dev/null 2>&1; then
  FFMPEG_BIN="$(command -v ffmpeg)"
fi
if [[ "${AUTO_INSTALL_MEDIA_TOOLS:-1}" == "1" && -x ./install-media-tools.sh && ( ! -x "${YTDLP_BIN}" || ! -x "${FFMPEG_BIN}" ) ]]; then
  log "yt-dlp or ffmpeg not found; installing local media tools..."
  if ! ./install-media-tools.sh >/dev/null; then
    log "Warning: media tool installation failed; .ytd will report that media tools are unavailable."
  fi
fi
if [[ "${AUTO_INSTALL_MEDIA_TOOLS:-1}" == "1" && -x "${MEDIA_ROOT}/yt-dlp" ]]; then
  YTDLP_BIN="${MEDIA_ROOT}/yt-dlp"
fi
if [[ "${AUTO_INSTALL_MEDIA_TOOLS:-1}" == "1" && -x "${MEDIA_ROOT}/ffmpeg/ffmpeg" ]]; then
  FFMPEG_BIN="${MEDIA_ROOT}/ffmpeg/ffmpeg"
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

exec env CONFIG_FILE="${CONFIG_FILE_PATH}" YTDLP_BIN="${YTDLP_BIN}" FFMPEG_BIN="${FFMPEG_BIN}" MEDIA_WORK_DIR="${MEDIA_WORK_DIR:-${PROJECT_ROOT}/.tools/media-work}" "${BOT_BINARY}"
