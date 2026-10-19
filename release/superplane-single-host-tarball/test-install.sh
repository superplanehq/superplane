#!/usr/bin/env bash

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TEMPLATES_DIR="${REPO_ROOT}/release/superplane-single-host-tarball/templates"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

REAL_DOCKER="$(command -v docker)"

WORK_DIR="${TMP_DIR}/work"
STUB_BIN="${TMP_DIR}/bin"
mkdir -p "${WORK_DIR}" "${STUB_BIN}"

cp "${TEMPLATES_DIR}/install.sh" "${WORK_DIR}/install.sh"
cp "${TEMPLATES_DIR}/superplane-logo.txt" "${WORK_DIR}/superplane-logo.txt"
cp "${TEMPLATES_DIR}/docker-compose.yml" "${WORK_DIR}/docker-compose.yml"

# Stub docker so the installer stops before pulling images and starting
# containers. The env file must already hold the owner values at that point.
printf '#!/usr/bin/env bash\nexit 0\n' > "${STUB_BIN}/docker"
chmod +x "${STUB_BIN}/docker"

run_installer() {
  (
    cd "${WORK_DIR}"
    PATH="${STUB_BIN}:/usr/local/bin:/usr/bin:/bin" "$@"
  )
}

assert_env_value() {
  local name="$1"
  local expected="$2"
  local actual

  actual="$(cd "${WORK_DIR}" && OIDC_KEYS_HOST_PATH=./oidc OIDC_KEYS_CONTAINER_PATH=/app/oidc-keys \
    "${REAL_DOCKER}" compose -f docker-compose.resolved.yml config \
    | sed -n "s/^      ${name}: //p" | head -n 1)"

  # docker compose config escapes a literal dollar sign as $$.
  expected="${expected//\$/\$\$}"

  if [[ "${actual}" != "${expected}" ]]; then
    echo "${name} parsed as ${actual@Q}, expected ${expected@Q}" >&2
    exit 1
  fi
}

sed 's/__SUPERPLANE_VERSION__/v0.0.0/' "${WORK_DIR}/docker-compose.yml" > "${WORK_DIR}/docker-compose.resolved.yml"

#
# Exported owner values are written into the env file and survive docker
# compose parsing unchanged, including password special characters.
#

OWNER_PASSWORD='P$ss "w0rd"\#1'

run_installer env \
  SUPERPLANE_OWNER_EMAIL='owner@example.com' \
  SUPERPLANE_OWNER_PASSWORD="${OWNER_PASSWORD}" \
  SUPERPLANE_OWNER_FIRST_NAME='Owner' \
  SUPERPLANE_OWNER_LAST_NAME='User' \
  SUPERPLANE_OWNER_ORG_NAME='Acme Corp' \
  bash install.sh <<< 'superplane.example.com' >/dev/null

grep -q '^SUPERPLANE_OWNER_EMAIL="owner@example.com"$' "${WORK_DIR}/superplane.env"

assert_env_value SUPERPLANE_OWNER_EMAIL 'owner@example.com'
assert_env_value SUPERPLANE_OWNER_PASSWORD "${OWNER_PASSWORD}"
assert_env_value SUPERPLANE_OWNER_FIRST_NAME 'Owner'
assert_env_value SUPERPLANE_OWNER_LAST_NAME 'User'
assert_env_value SUPERPLANE_OWNER_ORG_NAME 'Acme Corp'

#
# A trailing backslash in the password survives quoting.
#

rm -f "${WORK_DIR}/superplane.env"
run_installer env SUPERPLANE_OWNER_PASSWORD='Abcdefg1\' bash install.sh <<< 'superplane.example.com' >/dev/null
assert_env_value SUPERPLANE_OWNER_PASSWORD 'Abcdefg1\'

#
# Partial values pass through so the server can report what is missing.
#

rm -f "${WORK_DIR}/superplane.env"
run_installer env SUPERPLANE_OWNER_EMAIL='owner@example.com' bash install.sh <<< 'superplane.example.com' >/dev/null

if ! grep -q '^SUPERPLANE_OWNER_EMAIL="owner@example.com"$' "${WORK_DIR}/superplane.env"; then
  echo "partial export did not write SUPERPLANE_OWNER_EMAIL" >&2
  exit 1
fi

if grep -q '^SUPERPLANE_OWNER_PASSWORD=' "${WORK_DIR}/superplane.env"; then
  echo "partial export wrote SUPERPLANE_OWNER_PASSWORD" >&2
  exit 1
fi

#
# Without exports the env file keeps only the commented examples.
#

rm -f "${WORK_DIR}/superplane.env"
run_installer bash install.sh <<< 'superplane.example.com' >/dev/null

if grep -q '^SUPERPLANE_OWNER_' "${WORK_DIR}/superplane.env"; then
  echo "installer wrote owner values without exports" >&2
  exit 1
fi
