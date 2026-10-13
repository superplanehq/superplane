#!/usr/bin/env bash

# Bake a static whisper-cli and the checksummed ggml-tiny.bin model.
# ffmpeg and ffprobe must already be on PATH. Tasks must not download models.
set -euxo pipefail

WHISPER_VERSION="${WHISPER_VERSION:-v1.9.2}"
WHISPER_REPO="${WHISPER_REPO:-https://github.com/ggml-org/whisper.cpp.git}"
WHISPER_MODEL_REVISION="${WHISPER_MODEL_REVISION:-5359861c739e955e79d9a303bcbc70fb988958b1}"
WHISPER_MODEL_URL="${WHISPER_MODEL_URL:-https://huggingface.co/ggerganov/whisper.cpp/resolve/${WHISPER_MODEL_REVISION}/ggml-tiny.bin}"
WHISPER_MODEL_SHA256="${WHISPER_MODEL_SHA256:-be07e048e1e599ad46341c8d2a135645097a538221678b7acdd1b1919c6e1b21}"
PREFIX="${PREFIX:-/usr/local}"
MODEL_DIR="${MODEL_DIR:-/usr/local/share/whisper}"

if ! command -v ffmpeg >/dev/null 2>&1 || ! command -v ffprobe >/dev/null 2>&1; then
  printf 'ffmpeg and ffprobe must be installed before install-media-tools.sh.\n' >&2
  exit 1
fi

ffmpeg -version >/dev/null
ffprobe -version >/dev/null

workdir="$(mktemp -d)"
trap 'rm -rf "${workdir}"' EXIT
cd "${workdir}"

git clone --depth 1 --branch "${WHISPER_VERSION}" "${WHISPER_REPO}" whisper.cpp

# Fleet CPUs can differ from the Packer build CPU. Disable native tuning so the
# baked binary does not require instruction sets that are absent at runtime.
cmake_args=(
  -DCMAKE_BUILD_TYPE=Release
  -DBUILD_SHARED_LIBS=OFF
  -DGGML_NATIVE=OFF
  -DWHISPER_BUILD_EXAMPLES=ON
  -DWHISPER_SDL2=OFF
  -DWHISPER_CURL=OFF
)
if [ "$(uname -m)" = "aarch64" ]; then
  # Apple Silicon and Graviton 2 (t4g) implement this portable ARM baseline.
  cmake_args+=(
    -DGGML_CPU_ARM_ARCH=armv8.2-a+dotprod+fp16
  )
fi

cmake -S whisper.cpp -B whisper.cpp/build "${cmake_args[@]}"
cmake --build whisper.cpp/build --config Release -j"$(nproc)" --target whisper-cli

whisper_cli=""
for candidate in whisper.cpp/build/bin/whisper-cli whisper.cpp/build/whisper-cli; do
  if [ -x "${candidate}" ]; then
    whisper_cli="${candidate}"
    break
  fi
done
if [ -z "${whisper_cli}" ]; then
  printf 'whisper-cli binary was not built.\n' >&2
  exit 1
fi

install -D -m 0755 "${whisper_cli}" "${PREFIX}/bin/whisper-cli"
mkdir -p "${MODEL_DIR}"
curl --fail --location --silent --show-error \
  "${WHISPER_MODEL_URL}" \
  --output "${MODEL_DIR}/ggml-tiny.bin"
echo "${WHISPER_MODEL_SHA256}  ${MODEL_DIR}/ggml-tiny.bin" | sha256sum -c -

command -v whisper-cli
whisper-cli --help >/dev/null
test -s "${MODEL_DIR}/ggml-tiny.bin"

bash "$(dirname "$0")/install-yt-dlp.sh"
