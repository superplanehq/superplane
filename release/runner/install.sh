#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RUNNER_BINARY="${SCRIPT_DIR}/runner"
RUNNER_API_URL="${RUNNER_API_URL:-}"
RUNNER_REGISTRATION_TOKEN="${RUNNER_REGISTRATION_TOKEN:-}"
RUNNER_TAGS="${RUNNER_TAGS:-}"
RUNNER_TAGS_FROM_EC2_METADATA="${RUNNER_TAGS_FROM_EC2_METADATA:-}"

fail() {
  echo "Error: $*" >&2
  exit 1
}

validate_environment_value() {
  local name="$1"
  local value="$2"

  [ -n "${value}" ] || fail "${name} is required"
  case "${value}" in
    *$'\n'* | *$'\r'*) fail "${name} must not contain newlines" ;;
  esac
}

quote_environment_value() {
  local value="$1"

  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  printf '"%s"' "${value}"
}

[ "$(id -u)" -eq 0 ] || fail "run install.sh as root"
[ -f "${RUNNER_BINARY}" ] || fail "runner binary is missing from the release bundle"
command -v systemctl >/dev/null 2>&1 || fail "systemd is required"
validate_environment_value "RUNNER_API_URL" "${RUNNER_API_URL}"
validate_environment_value "RUNNER_REGISTRATION_TOKEN" "${RUNNER_REGISTRATION_TOKEN}"
if [ -n "${RUNNER_TAGS}" ]; then
  validate_environment_value "RUNNER_TAGS" "${RUNNER_TAGS}"
fi
if [ -n "${RUNNER_TAGS_FROM_EC2_METADATA}" ]; then
  validate_environment_value "RUNNER_TAGS_FROM_EC2_METADATA" "${RUNNER_TAGS_FROM_EC2_METADATA}"
fi

install -d -m 0755 /usr/local/bin
install -m 0755 "${RUNNER_BINARY}" /usr/local/bin/superplane-runner

install -d -m 0700 /var/lib/superplane-runner
{
  printf 'RUNNER_API_URL='
  quote_environment_value "${RUNNER_API_URL}"
  printf '\nRUNNER_REGISTRATION_TOKEN='
  quote_environment_value "${RUNNER_REGISTRATION_TOKEN}"
  printf '\nRUNNER_TAGS='
  quote_environment_value "${RUNNER_TAGS}"
  printf '\nRUNNER_TAGS_FROM_EC2_METADATA='
  quote_environment_value "${RUNNER_TAGS_FROM_EC2_METADATA}"
  printf '\n'
} > /etc/default/superplane-runner
chmod 0600 /etc/default/superplane-runner

cat > /etc/systemd/system/superplane-runner.service <<'UNITEOF'
[Unit]
Description=SuperPlane Runner
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=ubuntu
Group=ubuntu
EnvironmentFile=/etc/default/superplane-runner
WorkingDirectory=/home/ubuntu
ExecStart=/usr/local/bin/superplane-runner
Restart=no
ExecStopPost=+/bin/sh -c 'sleep 5; /usr/sbin/poweroff'

[Install]
WantedBy=multi-user.target
UNITEOF

systemctl daemon-reload
systemctl enable --now superplane-runner.service
