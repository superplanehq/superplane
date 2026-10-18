#!/usr/bin/env bash

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TEST_DIRECTORY="$(mktemp -d)"
trap 'rm -rf "${TEST_DIRECTORY}"' EXIT

mkdir -p "${TEST_DIRECTORY}/bin"
cat >"${TEST_DIRECTORY}/bin/docker" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

reference=""
for argument in "$@"; do
  if [[ "${argument}" == docker://* ]]; then
    reference="${argument}"
  fi
done

printf '%s\n' "${reference}" >>"${TEST_REFERENCE_LOG}"
if [[ "${reference}" == *:v1.2.3@sha256:* ]]; then
  echo "Docker references with both a tag and digest are not supported" >&2
  exit 1
fi

if [[ " $* " == *" --raw "* ]]; then
  cat <<'JSON'
{"manifests":[
  {"platform":{"os":"linux","architecture":"amd64"}},
  {"platform":{"os":"linux","architecture":"arm64"}},
  {"annotations":{"vnd.docker.reference.type":"attestation-manifest"}}
]}
JSON
  exit 0
fi

if [[ " $* " == *" {{.Digest}} "* ]]; then
  printf '%s\n' "${TEST_IMAGE_DIGEST}"
  exit 0
fi

cat <<JSON
{"org.opencontainers.image.source":"https://github.com/superplanehq/superplane","org.opencontainers.image.version":"v1.2.3","org.opencontainers.image.revision":"${TEST_SOURCE_SHA}"}
JSON
EOF
chmod +x "${TEST_DIRECTORY}/bin/docker"

export SKOPEO_CONTAINER_IMAGE="quay.io/skopeo/stable@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
export TEST_IMAGE_DIGEST="sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
export TEST_REFERENCE_LOG="${TEST_DIRECTORY}/references.log"
export TEST_SOURCE_SHA="cccccccccccccccccccccccccccccccccccccccc"

PATH="${TEST_DIRECTORY}/bin:${PATH}" \
  bash "${REPO_ROOT}/release/fleet-manager/verify-image.sh" \
    ghcr.io/superplanehq/fleet-manager:v1.2.3 \
    "${TEST_IMAGE_DIGEST}" \
    v1.2.3 \
    "${TEST_SOURCE_SHA}" >/dev/null

grep -Fx -- "docker://ghcr.io/superplanehq/fleet-manager:v1.2.3" "${TEST_REFERENCE_LOG}" >/dev/null
if grep -F -- ":v1.2.3@${TEST_IMAGE_DIGEST}" "${TEST_REFERENCE_LOG}" >/dev/null; then
  echo "Digest verification must not combine a tag and digest" >&2
  exit 1
fi
if [ "$(grep -Fxc -- "docker://ghcr.io/superplanehq/fleet-manager@${TEST_IMAGE_DIGEST}" "${TEST_REFERENCE_LOG}")" -ne 3 ]; then
  echo "Digest verification must inspect the immutable digest reference" >&2
  exit 1
fi

echo "Fleet Manager image verification tests passed."
