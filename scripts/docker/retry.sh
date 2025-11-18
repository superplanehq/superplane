#!/usr/bin/env bash
set -euo pipefail

# Simple retry helper with exponential backoff.
# Usage: retry.sh <max_attempts> <initial_sleep>
#        <command> [args ...]
# Example: retry.sh 6 2s go install example.com/foo@latest

if [[ $# -lt 3 ]]; then
  echo "Usage: $0 <max_attempts> <initial_sleep> <command> [args...]" >&2
  exit 2
fi

attempts=$1
sleep_dur=$2
shift 2

for ((i=1; i<=attempts; i++)); do
  if "$@"; then
    exit 0
  fi

  status=$?
  if [[ $i -lt attempts ]]; then
    echo "Command failed (exit $status). Attempt $i/$attempts. Retrying in $sleep_dur..." >&2
    sleep "$sleep_dur"
    # exponential backoff: double the sleep duration if it ends with 's'
    if [[ "$sleep_dur" =~ ^([0-9]+)s$ ]]; then
      secs=${BASH_REMATCH[1]}
      sleep_dur="$((secs*2))s"
    fi
  else
    echo "Command failed after $attempts attempts (exit $status): $*" >&2
    exit $status
  fi
done

