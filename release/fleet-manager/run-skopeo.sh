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
  auth_directory="$(dirname "${REGISTRY_AUTH_FILE}")"
  if [ ! -d "${auth_directory}" ]; then
    echo "Error: registry auth directory does not exist: ${auth_directory}" >&2
    exit 1
  fi
  docker_args+=(
    --volume "${auth_directory}:${auth_directory}"
    --env "REGISTRY_AUTH_FILE=${REGISTRY_AUTH_FILE}"
  )
fi

exec docker "${docker_args[@]}" "${SKOPEO_CONTAINER_IMAGE}" "$@"
