#!/usr/bin/env bash

set -euo pipefail

if [[ -z "${RABBITMQ_ADDITIONAL_VHOSTS:-}" ]]; then
  exit 0
fi

# Run in the background so RabbitMQ can continue booting.
(
  for _ in $(seq 1 90); do
    if rabbitmq-diagnostics -q ping >/dev/null 2>&1; then
      break
    fi
    sleep 1
  done

  IFS=',' read -r -a vhosts <<< "${RABBITMQ_ADDITIONAL_VHOSTS}"
  for raw_vhost in "${vhosts[@]}"; do
    vhost="$(echo "${raw_vhost}" | tr -d '[:space:]')"
    if [[ -z "${vhost}" ]]; then
      continue
    fi

    rabbitmqctl add_vhost "${vhost}" || true
    rabbitmqctl set_permissions -p "${vhost}" guest ".*" ".*" ".*"
  done
) &
