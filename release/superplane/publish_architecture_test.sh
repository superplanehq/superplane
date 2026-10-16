#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

mkdir -p "${TMP_DIR}/bin"
printf '{"schemaVersion":2}\n' >"${TMP_DIR}/image.oci.tar"
PREPARED_DIGEST="sha256:$(sha256sum "${TMP_DIR}/image.oci.tar" | awk '{print $1}')"

cat >"${TMP_DIR}/bin/skopeo" <<'SHIM'
#!/usr/bin/env bash
set -euo pipefail

if [ "$1" = inspect ] && [ "$2" != --raw ]; then
  echo 'no image found in image index for architecture "amd64"' >&2
  exit 1
fi
if [ "$1" = inspect ] && [ "$2" = --raw ]; then
  reference=""
  for argument in "$@"; do
    reference="${argument}"
  done
  if [[ "${reference}" == oci-archive:* ]]; then
    cat "${TEST_ARCHIVE}"
    exit 0
  fi
  if [ "${TEST_MODE}" = collision ]; then
    printf 'different\n'
    exit 0
  fi
  if [ "${TEST_MODE}" = matching ] || [ -f "${TEST_PUBLISHED}" ]; then
    cat "${TEST_ARCHIVE}"
    exit 0
  fi
  echo 'manifest unknown' >&2
  exit 1
fi
if [ "$1" = copy ]; then
  printf 'copy\n' >>"${TEST_COPY_LOG}"
  : >"${TEST_PUBLISHED}"
  exit 0
fi
exit 2
SHIM
chmod +x "${TMP_DIR}/bin/skopeo"

export TEST_ARCHIVE="${TMP_DIR}/image.oci.tar"
export TEST_COPY_LOG="${TMP_DIR}/copy.log"
export TEST_PUBLISHED="${TMP_DIR}/published"
export SKOPEO_CONTAINER_IMAGE=""

TEST_MODE=missing PATH="${TMP_DIR}/bin:${PATH}" \
  bash "${SCRIPT_DIR}/publish-architecture.sh" v1.2.3 amd64 "${TEST_ARCHIVE}" \
  >"${TMP_DIR}/digest"
grep -Fx "${PREPARED_DIGEST}" "${TMP_DIR}/digest" >/dev/null
test "$(wc -l <"${TEST_COPY_LOG}")" -eq 1

rm -f "${TEST_PUBLISHED}"
TEST_MODE=matching PATH="${TMP_DIR}/bin:${PATH}" \
  bash "${SCRIPT_DIR}/publish-architecture.sh" v1.2.3 amd64 "${TEST_ARCHIVE}" >/dev/null
test "$(wc -l <"${TEST_COPY_LOG}")" -eq 1

different_digest="sha256:$(printf 'different\n' | sha256sum | awk '{print $1}')"
TEST_MODE=collision PATH="${TMP_DIR}/bin:${PATH}" \
  bash "${SCRIPT_DIR}/publish-architecture.sh" v1.2.3 amd64 "${TEST_ARCHIVE}" \
  >"${TMP_DIR}/digest"
grep -Fx "${different_digest}" "${TMP_DIR}/digest" >/dev/null
test "$(wc -l <"${TEST_COPY_LOG}")" -eq 1

echo "SuperPlane architecture publication tests passed."
