#!/bin/sh
# docker-entrypoint.sh — fetch fleet-manager JSON config from S3 (if FM_CONFIG_S3_URI
# is set) then exec the binary. Runs as the container's default user.
#
# Environment variables:
#   FM_CONFIG_S3_URI   — S3 URI to fetch config from, e.g. s3://bucket/path/config.json
#                        When empty the binary is started directly (bind-mount path or
#                        FM_CONFIG_FILE must already exist).
#   FM_CONFIG_FILE     — local path to write the fetched config (default /tmp/fleet-manager/config.json)
#   AWS_REGION         — required when FM_CONFIG_S3_URI is set (used by aws s3 cp)
set -eu

if [ -n "${FM_CONFIG_S3_URI:-}" ]; then
  echo "fleet-manager: fetching config from ${FM_CONFIG_S3_URI}"
  CONFIG_PATH="${FM_CONFIG_FILE:-/tmp/fleet-manager/config.json}"
  mkdir -p "$(dirname "$CONFIG_PATH")"
  aws s3 cp "${FM_CONFIG_S3_URI}" "${CONFIG_PATH}"
  chmod 600 "${CONFIG_PATH}"
  echo "fleet-manager: config written to ${CONFIG_PATH}"
  export FM_CONFIG_FILE="${CONFIG_PATH}"
fi

exec /fleet-manager "$@"
