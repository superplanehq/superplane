#!/bin/sh
# docker-entrypoint.sh — fetch fleet-manager JSON config from S3 or SSM Parameter Store
# (if FM_CONFIG_S3_URI or FM_CONFIG_SSM_PARAM is set) then exec the binary.
# Runs as the container's default user.
#
# Environment variables (pick one):
#   FM_CONFIG_S3_URI     — S3 URI to fetch config from, e.g. s3://bucket/path/config.json
#   FM_CONFIG_SSM_PARAM  — SSM parameter name, e.g. /superplane/fleet-manager/config
#                          Supports SecureString (decrypted automatically via task role).
#
# Common:
#   FM_CONFIG_FILE       — local path to write the fetched config (default /tmp/fleet-manager/config.json)
#   AWS_REGION           — required when fetching from S3 or SSM
set -eu

CONFIG_PATH="${FM_CONFIG_FILE:-/tmp/fleet-manager/config.json}"
mkdir -p "$(dirname "$CONFIG_PATH")"

if [ -n "${FM_CONFIG_SSM_PARAM:-}" ]; then
  echo "fleet-manager: fetching config from SSM parameter ${FM_CONFIG_SSM_PARAM}"
  aws ssm get-parameter \
    --name "${FM_CONFIG_SSM_PARAM}" \
    --with-decryption \
    --query Parameter.Value \
    --output text > "${CONFIG_PATH}"
  chmod 600 "${CONFIG_PATH}"
  echo "fleet-manager: config written to ${CONFIG_PATH}"
  export FM_CONFIG_FILE="${CONFIG_PATH}"
elif [ -n "${FM_CONFIG_S3_URI:-}" ]; then
  echo "fleet-manager: fetching config from ${FM_CONFIG_S3_URI}"
  aws s3 cp "${FM_CONFIG_S3_URI}" "${CONFIG_PATH}"
  chmod 600 "${CONFIG_PATH}"
  echo "fleet-manager: config written to ${CONFIG_PATH}"
  export FM_CONFIG_FILE="${CONFIG_PATH}"
fi

exec /fleet-manager "$@"
