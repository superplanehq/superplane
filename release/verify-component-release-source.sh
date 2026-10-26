#!/usr/bin/env bash

set -euo pipefail

if [ "$#" -ne 2 ]; then
  echo "Usage: release/verify-component-release-source.sh <component> <version>" >&2
  exit 1
fi

COMPONENT="$1"
VERSION="$2"
REPOSITORY="${GITHUB_REPOSITORY:-superplanehq/superplane}"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TAG="$(node "${REPO_ROOT}/release/component-release.js" tag "${COMPONENT}" "${VERSION}")"

required_environment=(
  SEMAPHORE_GIT_SHA
  SEMAPHORE_GIT_BRANCH
  SEMAPHORE_PIPELINE_PROMOTION
  SEMAPHORE_WORKFLOW_ID
  SEMAPHORE_PIPELINE_ID
)
for name in "${required_environment[@]}"; do
  if [ -z "${!name:-}" ]; then
    echo "Error: ${name} is required" >&2
    exit 1
  fi
done
SOURCE_BRANCH="${SEMAPHORE_GIT_BRANCH}"
if [ "${COMPONENT}" = "superplane" ]; then
  bash "${REPO_ROOT}/release/verify-superplane-release-branch.sh" "${SOURCE_BRANCH}"
elif [ "${SOURCE_BRANCH}" != "main" ]; then
  echo "Error: component releases must originate from main" >&2
  exit 1
fi
if [ "${SEMAPHORE_PIPELINE_PROMOTION}" != "true" ]; then
  echo "Error: component releases must run as a manual promotion" >&2
  exit 1
fi
if [ "$(git -C "${REPO_ROOT}" rev-parse HEAD)" != "${SEMAPHORE_GIT_SHA}" ]; then
  echo "Error: checkout does not match SEMAPHORE_GIT_SHA" >&2
  exit 1
fi
worktree_changes="$(git -C "${REPO_ROOT}" status --porcelain --untracked-files=all)"
if [ -n "${worktree_changes}" ]; then
  echo "Error: release checkout contains changes outside ${SEMAPHORE_GIT_SHA}:" >&2
  echo "${worktree_changes}" >&2
  exit 1
fi

SOURCE_REF="origin/${SOURCE_BRANCH}"
git -C "${REPO_ROOT}" fetch --quiet --tags origin "${SOURCE_BRANCH}"
if ! git -C "${REPO_ROOT}" merge-base --is-ancestor \
  "${SEMAPHORE_GIT_SHA}" "${SOURCE_REF}"; then
  echo "Error: ${SEMAPHORE_GIT_SHA} is no longer reachable from ${SOURCE_REF}" >&2
  exit 1
fi
if git -C "${REPO_ROOT}" ls-remote --exit-code --tags origin \
  "refs/tags/${TAG}" >/dev/null 2>&1; then
  echo "Error: tag ${TAG} already exists" >&2
  exit 1
fi

status_json="$(mktemp)"
release_json="$(mktemp)"
trap 'rm -f "${status_json}" "${release_json}"' EXIT
curl --fail --silent --show-error \
  --header "Accept: application/vnd.github+json" \
  --header "X-GitHub-Api-Version: 2022-11-28" \
  "https://api.github.com/repos/${REPOSITORY}/commits/${SEMAPHORE_GIT_SHA}/status" \
  >"${status_json}"
if ! jq -e \
  'any(.statuses[]; .context == "ci/semaphoreci/push: CI" and .state == "success")' \
  "${status_json}" >/dev/null; then
  echo "Error: required successful GitHub CI status is missing for ${SEMAPHORE_GIT_SHA}" >&2
  exit 1
fi

encoded_tag="$(node -e 'process.stdout.write(encodeURIComponent(process.argv[1]))' "${TAG}")"
release_status="$(
  curl --silent --show-error \
    --output "${release_json}" \
    --write-out '%{http_code}' \
    --header "Accept: application/vnd.github+json" \
    --header "X-GitHub-Api-Version: 2022-11-28" \
    "https://api.github.com/repos/${REPOSITORY}/releases/tags/${encoded_tag}"
)"
if [ "${release_status}" = "200" ]; then
  echo "Error: GitHub release ${TAG} already exists" >&2
  exit 1
fi
if [ "${release_status}" != "404" ]; then
  echo "Error: GitHub release lookup returned HTTP ${release_status}" >&2
  cat "${release_json}" >&2
  exit 1
fi

echo "Release source ${SEMAPHORE_GIT_SHA} is eligible for ${TAG}."
