#!/usr/bin/env bash
# Verify SuperPlane app image builds on a runner host (EC2) or locally.
# Full docker buildx (linux/amd64), no registry push — for #4693 / future daily builds.
#
# Env:
#   REPO_URL          default https://github.com/superplanehq/superplane.git
#   GIT_REF           branch or SHA to build (default: current HEAD when in-repo)
#   ARCH              default amd64
#   DEV_BASE_IMAGE    default ghcr.io/superplanehq/superplane-dev-base:app-latest
#   VERIFY_WORK_DIR   use this checkout instead of cloning (local dev)
#   SKIP_PROTO_GEN    set to 1 to skip (not recommended; generated protos are not in git)
set -euo pipefail
IFS=$'\n\t'

ARCH="${ARCH:-amd64}"
REPO_URL="${REPO_URL:-https://github.com/superplanehq/superplane.git}"
DEV_BASE_IMAGE="${DEV_BASE_IMAGE:-ghcr.io/superplanehq/superplane-dev-base:app-latest}"
LOCAL_TAG="${LOCAL_TAG:-superplane:runner-verify}"

cleanup() {
  if [[ -n "${TMP_WORK:-}" && -d "$TMP_WORK" ]]; then
    rm -rf "$TMP_WORK"
  fi
}
trap cleanup EXIT

need_codegen() {
  [[ "${SKIP_PROTO_GEN:-0}" == "1" ]] && return 1
  return 0
}

make_var() {
  local name="$1"
  awk -v name="$name" '
    $1 == name && $2 == ":=" {
      sub("^[^:]+:= ?", "")
      print
      exit
    }
  ' Makefile
}

load_codegen_modules() {
  MODULES="$(make_var MODULES)"
  REST_API_MODULES="$(make_var REST_API_MODULES)"
  if [[ -z "$MODULES" || -z "$REST_API_MODULES" ]]; then
    echo "Could not read MODULES and REST_API_MODULES from Makefile" >&2
    exit 1
  fi
}

run_proto_gen() {
  echo "==> Generating protobuf (dev-base container, no dev.up)"
  docker pull "$DEV_BASE_IMAGE"
  docker run --rm \
    -v "$(pwd):/app" \
    -w /app \
    -u "$(id -u):$(id -g)" \
    "$DEV_BASE_IMAGE" \
    bash -lc "/app/scripts/protoc.sh ${MODULES} && /app/scripts/protoc_gateway.sh ${REST_API_MODULES}"
}

OPENAPI_GENERATOR_IMAGE="openapitools/openapi-generator-cli:v7.13.0"

run_openapi_spec_gen() {
  echo "==> Generating OpenAPI spec (dev-base container)"
  docker run --rm \
    -v "$(pwd):/app" \
    -w /app \
    -u "$(id -u):$(id -g)" \
    "$DEV_BASE_IMAGE" \
    bash -lc "/app/scripts/protoc_openapi_spec.sh ${REST_API_MODULES}"
}

run_openapi_go_client_gen() {
  echo "==> Generating OpenAPI Go client (openapi-generator-cli)"
  rm -rf pkg/openapi_client
  docker run --rm \
    -v "$(pwd):/local" \
    -u "$(id -u):$(id -g)" \
    "$OPENAPI_GENERATOR_IMAGE" generate \
    -i /local/api/swagger/superplane.swagger.json \
    -g go \
    -o /local/pkg/openapi_client \
    --additional-properties=packageName=openapi_client,enumClassPrefix=true,isGoSubmodule=true,withGoMod=false
  rm -rf pkg/openapi_client/test pkg/openapi_client/docs pkg/openapi_client/api \
         pkg/openapi_client/.travis.yml pkg/openapi_client/README.md pkg/openapi_client/git_push.sh
  docker run --rm \
    -v "$(pwd):/app" \
    -w /app \
    -u "$(id -u):$(id -g)" \
    "$DEV_BASE_IMAGE" \
    bash -lc "find pkg/openapi_client -name '*.go' -print0 | xargs -0 gofmt -s -w"
}

run_openapi_web_client_gen() {
  echo "==> Generating OpenAPI web (TypeScript) client (dev-base container)"
  rm -rf web_src/src/api-client
  docker run --rm \
    -v "$(pwd):/app" \
    -w /app/web_src \
    -u "$(id -u):$(id -g)" \
    -e HOME=/tmp \
    -e NPM_CONFIG_CACHE=/tmp/.npm \
    "$DEV_BASE_IMAGE" \
    bash -lc "npm install --prefer-offline && npm run generate:api && npx prettier --log-level silent --write 'src/api-client/**/*.{ts,tsx}'"
}

run_image_build() {
  echo "==> docker buildx build (linux/${ARCH}, no push)"
  docker buildx build \
    --platform "linux/${ARCH}" \
    --progress=plain \
    --provenance=false \
    --cache-from "$DEV_BASE_IMAGE" \
    -t "$LOCAL_TAG" \
    --load \
    -f Dockerfile \
    .
  echo "==> OK: image loaded locally as ${LOCAL_TAG} (not pushed)"
}

checkout_ref() {
  if [[ -z "${GIT_REF:-}" ]]; then
    return
  fi
  git fetch origin --depth 1 "${GIT_REF}" 2>/dev/null || git fetch origin "${GIT_REF}"
  git checkout "${GIT_REF}"
}

prepare_workdir() {
  if [[ -n "${VERIFY_WORK_DIR:-}" ]]; then
    cd "$VERIFY_WORK_DIR"
    return
  fi
  if [[ -f Dockerfile && -f go.mod ]]; then
    echo "==> Using current directory as repo root"
    checkout_ref
    return
  fi
  TMP_WORK="$(mktemp -d)"
  echo "==> Cloning ${REPO_URL} (ref: ${GIT_REF:-default}) into ${TMP_WORK}"
  git clone --depth 1 "$REPO_URL" "$TMP_WORK/repo"
  cd "$TMP_WORK/repo"
  checkout_ref
}

main() {
  prepare_workdir
  load_codegen_modules
  if need_codegen; then
    run_proto_gen
    run_openapi_spec_gen
    run_openapi_go_client_gen
    run_openapi_web_client_gen
  else
    echo "==> Skipping codegen (SKIP_PROTO_GEN=1)"
  fi
  run_image_build
}

main "$@"
