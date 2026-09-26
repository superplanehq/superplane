#!/bin/bash
# Bake a static whisper-cli and the checksummed ggml-tiny.bin model.
# ffmpeg and ffprobe must already be on PATH (install.sh / Dockerfile.local).
# Tasks must not download models.
set -euxo pipefail

WHISPER_VERSION="${WHISPER_VERSION:-v1.9.2}"
WHISPER_REPO="${WHISPER_REPO:-https://github.com/ggml-org/whisper.cpp.git}"
WHISPER_MODEL_URL="${WHISPER_MODEL_URL:-https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-tiny.bin}"
WHISPER_MODEL_SHA256="${WHISPER_MODEL_SHA256:-be07e048e1e599ad46341c8d2a135645097a538221678b7acdd1b1919c6e1b21}"
PREFIX="${PREFIX:-/usr/local}"
MODEL_DIR="${MODEL_DIR:-/usr/local/share/whisper}"

if ! command -v ffmpeg >/dev/null 2>&1 || ! command -v ffprobe >/dev/null 2>&1; then
  printf 'ffmpeg and ffprobe must be installed before install-media-tools.sh.\n' >&2
  exit 1
fi

ffmpeg -version >/dev/null
ffprobe -version >/dev/null

WORKDIR=$(mktemp -d)
trap 'rm -rf "$WORKDIR"' EXIT
cd "$WORKDIR"

git clone --depth 1 --branch "$WHISPER_VERSION" "$WHISPER_REPO" whisper.cpp
cmake -S whisper.cpp -B whisper.cpp/build \
  -DCMAKE_BUILD_TYPE=Release \
  -DBUILD_SHARED_LIBS=OFF \
  -DWHISPER_BUILD_EXAMPLES=ON \
  -DWHISPER_SDL2=OFF \
  -DWHISPER_CURL=OFF
cmake --build whisper.cpp/build --config Release -j"$(nproc)" --target whisper-cli

CLI=""
for candidate in whisper.cpp/build/bin/whisper-cli whisper.cpp/build/whisper-cli; do
  if [ -x "$candidate" ]; then
    CLI=$candidate
    break
  fi
done
if [ -z "$CLI" ]; then
  printf 'whisper-cli binary was not built.\n' >&2
  exit 1
fi

install -D -m 755 "$CLI" "${PREFIX}/bin/whisper-cli"
mkdir -p "$MODEL_DIR"
curl -fsSL "$WHISPER_MODEL_URL" -o "${MODEL_DIR}/ggml-tiny.bin"
echo "${WHISPER_MODEL_SHA256}  ${MODEL_DIR}/ggml-tiny.bin" | sha256sum -c -

command -v whisper-cli
whisper-cli --help >/dev/null
test -s "${MODEL_DIR}/ggml-tiny.bin"
