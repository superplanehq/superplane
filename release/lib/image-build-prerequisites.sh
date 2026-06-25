#!/usr/bin/env bash

require_command() {
  local command_name="$1"
  local install_hint="$2"

  if command -v "${command_name}" >/dev/null 2>&1; then
    return
  fi

  echo "Error: '${command_name}' is required. ${install_hint}" >&2
  exit 1
}

generated_protobuf_missing() {
  [ ! -f "pkg/protos/me/me.pb.go" ] || [ ! -f "pkg/protos/me/me.pb.gw.go" ]
}

require_docker_buildx() {
  require_command docker "Install Docker and ensure it is available on PATH."

  if docker buildx version >/dev/null 2>&1; then
    return
  fi

  echo "Error: Docker Buildx is required. Install the Docker Buildx plugin before running this release build." >&2
  exit 1
}

require_protobuf_generation_prerequisites() {
  if ! generated_protobuf_missing; then
    return
  fi

  require_command make "Install make/build-essential before generating protobuf files."

  if docker compose version >/dev/null 2>&1; then
    return
  fi

  echo "Error: Docker Compose is required because generated protobuf files are missing and make dev.up must run." >&2
  exit 1
}

require_release_image_build_prerequisites() {
  require_docker_buildx
  require_protobuf_generation_prerequisites
}
