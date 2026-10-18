#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

mkdir -p "${TMP_DIR}/bin"
cat >"${TMP_DIR}/bin/skopeo" <<'SHIM'
#!/usr/bin/env bash
set -euo pipefail

reference=""
for argument in "$@"; do
  if [[ "${argument}" == docker://* ]]; then
    reference="${argument#docker://}"
  fi
done
mode="${TEST_MODE}"

if [ "${mode}" = missing-arm64 ] && [[ "${reference}" == *:v1.2.3 ]]; then
  cat <<'JSON'
{"manifests":[
  {"digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","platform":{"os":"linux","architecture":"amd64"}},
  {"digest":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","annotations":{"vnd.docker.reference.type":"attestation-manifest"}}
]}
JSON
  exit 0
fi

if [[ "${reference}" == *:v1.2.3 ]]; then
  if [[ "$*" == *"--raw"* ]]; then
    cat <<'JSON'
{"manifests":[
  {"digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","platform":{"os":"linux","architecture":"amd64"}},
  {"digest":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","platform":{"os":"linux","architecture":"arm64"}},
  {"digest":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","annotations":{"vnd.docker.reference.type":"attestation-manifest"}},
  {"digest":"sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd","annotations":{"vnd.docker.reference.type":"attestation-manifest"}}
]}
JSON
    exit 0
  fi
  if [ "${mode}" = old-revision ]; then
    printf '%s\n' '{"org.opencontainers.image.source":"https://github.com/superplanehq/superplane","org.opencontainers.image.version":"v1.2.3","org.opencontainers.image.revision":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}'
    exit 0
  fi
  printf '%s\n' '{"org.opencontainers.image.source":"https://github.com/superplanehq/superplane","org.opencontainers.image.version":"v1.2.3","org.opencontainers.image.revision":"cccccccccccccccccccccccccccccccccccccccc"}'
  exit 0
fi

if [ "${mode}" = foreign-digest ] && [[ "${reference}" == *-amd64 ]]; then
  printf '%s\n' '{"manifests":[{"digest":"sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"}]}'
  exit 0
fi

if [[ "${reference}" == *-amd64 ]]; then
  printf '%s\n' '{"manifests":[{"digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},{"digest":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"}]}'
  exit 0
fi

if [[ "${reference}" == *-arm64 ]]; then
  printf '%s\n' '{"manifests":[{"digest":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},{"digest":"sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"}]}'
  exit 0
fi

echo "unexpected skopeo invocation: $*" >&2
exit 1
SHIM
chmod +x "${TMP_DIR}/bin/skopeo"

export PATH="${TMP_DIR}/bin:${PATH}"
export SKOPEO_CONTAINER_IMAGE=""

TEST_MODE=ok bash "${SCRIPT_DIR}/verify-image.sh" \
  ghcr.io/superplanehq/superplane:v1.2.3 \
  v1.2.3 \
  cccccccccccccccccccccccccccccccccccccccc \
  ghcr.io/superplanehq/superplane:v1.2.3-amd64 \
  ghcr.io/superplanehq/superplane:v1.2.3-arm64 >/dev/null

TEST_MODE=old-revision bash "${SCRIPT_DIR}/verify-image.sh" \
  ghcr.io/superplanehq/superplane:v1.2.3 \
  v1.2.3 \
  cccccccccccccccccccccccccccccccccccccccc \
  ghcr.io/superplanehq/superplane:v1.2.3-amd64 \
  ghcr.io/superplanehq/superplane:v1.2.3-arm64 >/dev/null

if TEST_MODE=ok bash "${SCRIPT_DIR}/verify-image.sh" \
  ghcr.io/superplanehq/superplane:v1.2.3 \
  v9.9.9 \
  cccccccccccccccccccccccccccccccccccccccc \
  ghcr.io/superplanehq/superplane:v1.2.3-amd64 \
  ghcr.io/superplanehq/superplane:v1.2.3-arm64 \
  >/dev/null 2>"${TMP_DIR}/stderr"; then
  echo "Image labels for the wrong version must fail" >&2
  exit 1
fi
grep -q 'labels do not match v9.9.9' "${TMP_DIR}/stderr"

if TEST_MODE=missing-arm64 bash "${SCRIPT_DIR}/verify-image.sh" \
  ghcr.io/superplanehq/superplane:v1.2.3 \
  v1.2.3 \
  cccccccccccccccccccccccccccccccccccccccc \
  ghcr.io/superplanehq/superplane:v1.2.3-amd64 \
  ghcr.io/superplanehq/superplane:v1.2.3-arm64 >/dev/null 2>&1; then
  echo "A final index without linux/arm64 must fail" >&2
  exit 1
fi

if TEST_MODE=foreign-digest bash "${SCRIPT_DIR}/verify-image.sh" \
  ghcr.io/superplanehq/superplane:v1.2.3 \
  v1.2.3 \
  cccccccccccccccccccccccccccccccccccccccc \
  ghcr.io/superplanehq/superplane:v1.2.3-amd64 \
  ghcr.io/superplanehq/superplane:v1.2.3-arm64 >/dev/null 2>&1; then
  echo "A final index without the prepared architecture descriptor must fail" >&2
  exit 1
fi

echo "SuperPlane image index tests passed."
