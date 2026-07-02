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

generated_openapi_spec_missing() {
  [ ! -f "api/swagger/superplane.swagger.json" ]
}

generated_frontend_client_missing() {
  [ ! -f "web_src/src/api-client/index.ts" ] ||
    [ ! -f "web_src/src/api-client/sdk.gen.ts" ] ||
    [ ! -f "web_src/src/api-client/types.gen.ts" ]
}

generated_release_build_inputs_missing() {
  generated_protobuf_missing ||
    generated_openapi_spec_missing ||
    generated_frontend_client_missing
}

require_docker_buildx() {
  require_command docker "Install Docker and ensure it is available on PATH."

  if docker buildx version >/dev/null 2>&1; then
    return
  fi

  echo "Error: Docker Buildx is required. Install the Docker Buildx plugin before running this release build." >&2
  exit 1
}

require_generated_artifact_prerequisites() {
  if ! generated_release_build_inputs_missing; then
    return
  fi

  require_command make "Install make/build-essential before generating release build artifacts."

  if docker compose version >/dev/null 2>&1; then
    return
  fi

  echo "Error: Docker Compose is required because generated release build artifacts are missing and make dev.up must run." >&2
  exit 1
}

require_release_image_build_prerequisites() {
  require_docker_buildx
  require_generated_artifact_prerequisites
}
