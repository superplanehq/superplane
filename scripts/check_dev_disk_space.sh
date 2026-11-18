#!/usr/bin/env bash
set -euo pipefail

# Cold runner builds need room for dependencies and intermediate layers.
minimum_free_kb=$((15 * 1000 * 1000 * 1000 / 1024))

recovery_instructions() {
  echo 'Increase Docker disk space or remove unused images manually, then run make dev.up again.' >&2
  echo 'Database volumes were not changed.' >&2
}

available_kb() {
  local output available
  # Measure Docker's filesystem, including on Docker Desktop. This image has
  # no VOLUME declarations. The probe does not mount any host paths or volumes.
  if ! output=$(docker run --rm --read-only --network none busybox:1.37 df -Pk / 2>&1); then
    printf '%s\n' "$output" >&2
    # A full disk can prevent pulling the probe image or creating its container.
    # Return 2 so the caller can attempt cache cleanup before retrying the probe.
    case "$output" in
      *'no space left on device'*) return 2 ;;
    esac
    echo 'Cannot measure Docker disk space. Check that Docker is running.' >&2
    return 1
  fi
  # A first pull can print progress before df produces the root filesystem row.
  available=$(printf '%s\n' "$output" | awk '$NF == "/" { print $4 }')
  case "$available" in
    ''|*[!0-9]*) echo 'Cannot read available Docker disk space.' >&2; return 1 ;;
  esac
  printf '%s\n' "$available"
}

if free_kb=$(available_kb); then
  if [ "$free_kb" -ge "$minimum_free_kb" ]; then
    exit 0
  fi
else
  probe_status=$?
  if [ "$probe_status" -ne 2 ]; then
    recovery_instructions
    exit 1
  fi
fi

echo 'Docker has less than 15 GB free. Removing unused build cache...'
# Use the current engine, even if BUILDX_BUILDER selects a different builder.
# Build-cache pruning never removes containers, images, or database volumes.
context=$(docker context show)
if ! docker buildx prune --builder "$context" --all --force \
  --max-used-space 5GB --min-free-space 15GB; then
  recovery_instructions
  exit 1
fi

if ! free_kb=$(available_kb); then
  recovery_instructions
  exit 1
fi
if [ "$free_kb" -lt "$minimum_free_kb" ]; then
  echo "Docker still has only $((free_kb * 1024 / 1000 / 1000)) MB free; 15 GB is required." >&2
  recovery_instructions
  exit 1
fi

echo 'Docker has at least 15 GB free. Continuing startup.'
