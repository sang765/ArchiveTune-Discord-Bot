#!/usr/bin/env bash
set -Eeuo pipefail

cd "$(dirname "$0")"

log() {
  echo "[dependency-installer] $*" >&2
}

usage() {
  cat >&2 <<'EOF'
Usage: ./install-dependencies.sh [all|go|media]

Installs the requested cached dependencies under .tools.
The default mode is all.
EOF
}

download_file() {
  local url="$1"
  local output="$2"
  local max_time="$3"

  if command -v curl >/dev/null 2>&1; then
    curl --fail --location --retry 3 --connect-timeout 20 --max-time "$max_time" --silent --show-error -o "$output" "$url"
  elif command -v wget >/dev/null 2>&1; then
    wget --https-only --tries=3 --timeout=20 --quiet -O "$output" "$url"
  else
    log "ERROR: curl or wget is required to download dependencies."
    return 1
  fi
}

install_go() {
  local go_version="${GO_VERSION:-1.22.2}"
  local go_root="${GO_ROOT:-.tools/go}"
  local arch="${GO_ARCH:-amd64}"
  local platform="linux-${arch}"
  local archive="go${go_version}.${platform}.tar.gz"
  local download_url="https://go.dev/dl/${archive}"

  if [[ -x "${go_root}/bin/go" ]]; then
    log "Go already exists at ${go_root}/bin/go; reusing it."
    return 0
  fi
  if ! command -v tar >/dev/null 2>&1; then
    log "ERROR: tar is required to extract Go."
    return 1
  fi

  local tmp_dir
  tmp_dir="$(mktemp -d .go-install.XXXXXX)"
  cleanup_go() {
    rm -rf -- "${tmp_dir:-}"
  }
  trap cleanup_go RETURN

  log "Downloading Go ${go_version} for ${platform}..."
  download_file "$download_url" "${tmp_dir}/${archive}" 300

  log "Extracting Go into ${go_root}..."
  rm -rf -- "$go_root"
  mkdir -p -- "$go_root"
  tar -xzf "${tmp_dir}/${archive}" -C "$tmp_dir"
  mv -- "${tmp_dir}/go"/* "$go_root"/
  rmdir -- "${tmp_dir}/go"

  if [[ ! -x "${go_root}/bin/go" ]]; then
    log "ERROR: Go installation did not produce ${go_root}/bin/go."
    return 1
  fi
  log "Go installed at ${go_root}/bin/go."
  trap - RETURN
  cleanup_go
}

is_elf_binary() {
  [[ -x "$1" ]] || return 1
  [[ "$(head -c 4 "$1" | od -An -tx1 | tr -d ' \n')" == "7f454c46" ]]
}

install_media() {
  local media_root="${MEDIA_TOOLS_ROOT:-$(pwd -P)/.tools/media}"
  local ytdlp_path="${media_root}/yt-dlp"
  local ffmpeg_root="${media_root}/ffmpeg"
  local ffmpeg_path="${ffmpeg_root}/ffmpeg"
  local ffprobe_path="${ffmpeg_root}/ffprobe"
  local ytdlp_url="${YTDLP_URL:-https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux}"
  local ffmpeg_url="${FFMPEG_URL:-https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-amd64-static.tar.xz}"

  mkdir -p -- "$media_root"
  if ! is_elf_binary "$ytdlp_path"; then
    log "Downloading yt-dlp standalone binary..."
    local ytdlp_tmp="${ytdlp_path}.tmp"
    download_file "$ytdlp_url" "$ytdlp_tmp" 180
    chmod 755 "$ytdlp_tmp"
    if ! is_elf_binary "$ytdlp_tmp"; then
      rm -f -- "$ytdlp_tmp"
      log "ERROR: yt-dlp download is not a Linux standalone binary."
      return 1
    fi
    mv -f -- "$ytdlp_tmp" "$ytdlp_path"
  else
    log "yt-dlp already exists at ${ytdlp_path}; reusing it."
  fi

  if [[ ! -x "$ffmpeg_path" || ! -x "$ffprobe_path" ]]; then
    if ! command -v tar >/dev/null 2>&1; then
      log "ERROR: tar is required to extract ffmpeg."
      return 1
    fi
    log "Downloading static ffmpeg and ffprobe..."
    local archive="${media_root}/ffmpeg.tar.xz"
    local extract="${media_root}/.ffmpeg-extract"
    rm -rf -- "$extract"
    mkdir -p -- "$extract"
    download_file "$ffmpeg_url" "$archive" 300
    tar -xJf "$archive" -C "$extract"
    local found_root
    found_root="$(find "$extract" -mindepth 1 -maxdepth 1 -type d -print -quit)"
    if [[ -z "$found_root" || ! -x "${found_root}/ffmpeg" || ! -x "${found_root}/ffprobe" ]]; then
      rm -rf -- "$extract" "$archive"
      log "ERROR: downloaded ffmpeg archive has an unexpected layout."
      return 1
    fi
    rm -rf -- "$ffmpeg_root"
    mv -- "$found_root" "$ffmpeg_root"
    rm -rf -- "$extract" "$archive"
  else
    log "ffmpeg and ffprobe already exist under ${ffmpeg_root}; reusing them."
  fi
}

mode="${1:-all}"
case "$mode" in
  all)
    install_go
    install_media
    ;;
  go)
    install_go
    ;;
  media)
    install_media
    ;;
  -h|--help)
    usage
    ;;
  *)
    usage
    exit 2
    ;;
esac
