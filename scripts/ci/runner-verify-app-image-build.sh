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
MODULES="authorization,organizations,integrations,secrets,users,groups,roles,me,configuration,components,actions,triggers,widgets,blueprints,canvases,canvas_folders,service_accounts,agents,usage"
REST_API_MODULES="authorization,organizations,integrations,secrets,users,groups,roles,me,configuration,actions,triggers,widgets,blueprints,canvases,canvas_folders,service_accounts,agents"

cleanup() {
  if [[ -n "${TMP_WORK:-}" && -d "$TMP_WORK" ]]; then
    rm -rf "$TMP_WORK"
  fi
}
trap cleanup EXIT

need_proto_gen() {
  [[ "${SKIP_PROTO_GEN:-0}" == "1" ]] && return 1
  return 0
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

prepare_workdir() {
  if [[ -n "${VERIFY_WORK_DIR:-}" ]]; then
    cd "$VERIFY_WORK_DIR"
    return
  fi

  if [[ -f Dockerfile && -f go.mod ]]; then
    echo "==> Using current directory as repo root"
    if [[ -n "${GIT_REF:-}" ]]; then
      git fetch origin --depth 1 "${GIT_REF}" 2>/dev/null || git fetch origin "${GIT_REF}"
      git checkout "${GIT_REF}"
    fi
    return
  fi

  TMP_WORK="$(mktemp -d)"
  echo "==> Cloning ${REPO_URL} into ${TMP_WORK}"
  git clone --depth 1 "$REPO_URL" "$TMP_WORK/repo"
  cd "$TMP_WORK/repo"
  if [[ -n "${GIT_REF:-}" ]]; then
    git fetch origin --depth 1 "${GIT_REF}"
    git checkout "${GIT_REF}"
  fi
}

main() {
  prepare_workdir
  if need_proto_gen; then
    run_proto_gen
  else
    echo "==> Skipping proto gen (SKIP_PROTO_GEN=1)"
  fi
  run_image_build
}

main "$@"
