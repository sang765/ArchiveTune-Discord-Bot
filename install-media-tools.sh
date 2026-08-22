#!/usr/bin/env bash
set -Eeuo pipefail

cd "$(dirname "$0")"

MEDIA_ROOT="${MEDIA_TOOLS_ROOT:-$(pwd -P)/.tools/media}"
YTDLP_PATH="${MEDIA_ROOT}/yt-dlp"
FFMPEG_ROOT="${MEDIA_ROOT}/ffmpeg"
FFMPEG_PATH="${FFMPEG_ROOT}/ffmpeg"
FFPROBE_PATH="${FFMPEG_ROOT}/ffprobe"
YTDLP_URL="${YTDLP_URL:-https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux}"
FFMPEG_URL="${FFMPEG_URL:-https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-amd64-static.tar.xz}"

mkdir -p "${MEDIA_ROOT}"

is_elf_binary() {
  [[ -x "$1" ]] || return 1
  [[ "$(head -c 4 "$1" | od -An -tx1 | tr -d ' \n')" == "7f454c46" ]]
}

if ! is_elf_binary "${YTDLP_PATH}"; then
  echo "[media] Downloading yt-dlp..." >&2
  tmp="${YTDLP_PATH}.tmp"
  curl --fail --location --silent --show-error --retry 3 --connect-timeout 20 --max-time 180 -o "${tmp}" "${YTDLP_URL}"
  chmod 755 "${tmp}"
  if ! is_elf_binary "${tmp}"; then
    rm -f "${tmp}"
    echo "[media] ERROR: yt-dlp download is not a Linux standalone binary." >&2
    exit 1
  fi
  mv -f "${tmp}" "${YTDLP_PATH}"
fi

if [[ ! -x "${FFMPEG_PATH}" || ! -x "${FFPROBE_PATH}" ]]; then
  echo "[media] Downloading static ffmpeg..." >&2
  archive="${MEDIA_ROOT}/ffmpeg.tar.xz"
  extract="${MEDIA_ROOT}/.ffmpeg-extract"
  rm -rf "${extract}"
  mkdir -p "${extract}"
  curl --fail --location --silent --show-error --retry 3 --connect-timeout 20 --max-time 300 -o "${archive}" "${FFMPEG_URL}"
  tar -xJf "${archive}" -C "${extract}"
  found_root="$(find "${extract}" -mindepth 1 -maxdepth 1 -type d -print -quit)"
  if [[ -z "${found_root}" || ! -x "${found_root}/ffmpeg" || ! -x "${found_root}/ffprobe" ]]; then
    echo "[media] ERROR: downloaded ffmpeg archive has an unexpected layout." >&2
    exit 1
  fi
  rm -rf "${FFMPEG_ROOT}"
  mv "${found_root}" "${FFMPEG_ROOT}"
  rm -rf "${extract}" "${archive}"
fi

printf '%s\n' "${YTDLP_PATH}" "${FFMPEG_PATH}" "${FFPROBE_PATH}"
