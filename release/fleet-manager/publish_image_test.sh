#!/usr/bin/env bash

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TEST_DIRECTORY="$(mktemp -d)"
trap 'rm -rf "${TEST_DIRECTORY}"' EXIT

mkdir -p "${TEST_DIRECTORY}/bin"
printf '{"schemaVersion":2}\n' >"${TEST_DIRECTORY}/fleet-manager.oci.tar"
cat >"${TEST_DIRECTORY}/bin/skopeo" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

if [ "${1:-}" = inspect ] && [ "${2:-}" = --raw ]; then
  cat "${TEST_OCI_ARCHIVE}"
  exit 0
fi
if [ "${1:-}" = inspect ]; then
  echo "temporary registry connection failure" >&2
  exit 1
fi
if [ "${1:-}" = copy ]; then
  printf 'copy\n' >>"${TEST_COPY_LOG}"
  exit 0
fi
exit 2
EOF
chmod +x "${TEST_DIRECTORY}/bin/skopeo"

export TEST_OCI_ARCHIVE="${TEST_DIRECTORY}/fleet-manager.oci.tar"
export TEST_COPY_LOG="${TEST_DIRECTORY}/copy.log"
if PATH="${TEST_DIRECTORY}/bin:${PATH}" \
  bash "${REPO_ROOT}/release/fleet-manager/publish-image.sh" \
    v1.2.3 "${TEST_OCI_ARCHIVE}" >/dev/null 2>&1; then
  echo "Expected a registry lookup error to stop publication" >&2
  exit 1
fi
if [ -e "${TEST_COPY_LOG}" ]; then
  echo "Registry lookup errors must not publish an image" >&2
  exit 1
fi

cat >"${TEST_DIRECTORY}/bin/docker" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$@" >"${TEST_DOCKER_LOG}"
cat >"${TEST_DOCKER_STDIN}"
EOF
chmod +x "${TEST_DIRECTORY}/bin/docker"

export TEST_DOCKER_LOG="${TEST_DIRECTORY}/docker.log"
export TEST_DOCKER_STDIN="${TEST_DIRECTORY}/docker.stdin"
export REGISTRY_AUTH_FILE="${TEST_DIRECTORY}/auth/auth.json"
export SKOPEO_CONTAINER_IMAGE="quay.io/skopeo/stable@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
mkdir "$(dirname "${REGISTRY_AUTH_FILE}")"
printf 'test-token' | PATH="${TEST_DIRECTORY}/bin:${PATH}" \
  bash "${REPO_ROOT}/release/fleet-manager/run-skopeo.sh" \
    login example.com --username release-user --password-stdin

grep -Fx -- "--interactive" "${TEST_DOCKER_LOG}" >/dev/null
grep -Fx -- "${PWD}:${PWD}:ro" "${TEST_DOCKER_LOG}" >/dev/null
grep -Fx -- "$(dirname "${REGISTRY_AUTH_FILE}"):$(dirname "${REGISTRY_AUTH_FILE}")" "${TEST_DOCKER_LOG}" >/dev/null
grep -Fx -- "${SKOPEO_CONTAINER_IMAGE}" "${TEST_DOCKER_LOG}" >/dev/null
grep -Fx -- "login" "${TEST_DOCKER_LOG}" >/dev/null
grep -Fx -- "--password-stdin" "${TEST_DOCKER_LOG}" >/dev/null
printf 'test-token' | cmp - "${TEST_DOCKER_STDIN}"

echo "Fleet Manager image publication tests passed."
