#!/usr/bin/env bash

set -euo pipefail

if [ -z "${SKOPEO_CONTAINER_IMAGE:-}" ]; then
  exec skopeo "$@"
fi

if [[ ! "${SKOPEO_CONTAINER_IMAGE}" =~ @sha256:[0-9a-f]{64}$ ]]; then
  echo "Error: SKOPEO_CONTAINER_IMAGE must use an immutable SHA-256 digest" >&2
  exit 1
fi

docker_args=(
  run
  --rm
  --interactive
  --volume "${PWD}:${PWD}:ro"
  --workdir "${PWD}"
)

if [ -n "${REGISTRY_AUTH_FILE:-}" ]; then
  if [ ! -f "${REGISTRY_AUTH_FILE}" ]; then
    echo "Error: registry auth file does not exist: ${REGISTRY_AUTH_FILE}" >&2
    exit 1
  fi
  docker_args+=(
    --volume "${REGISTRY_AUTH_FILE}:${REGISTRY_AUTH_FILE}"
    --env "REGISTRY_AUTH_FILE=${REGISTRY_AUTH_FILE}"
  )
fi

exec docker "${docker_args[@]}" "${SKOPEO_CONTAINER_IMAGE}" "$@"
