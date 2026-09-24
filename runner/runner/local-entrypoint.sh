#!/bin/sh
set -eu

if [ -z "${RUNNER_REGISTRATION_TOKEN:-}" ] && [ -z "${RUNNER_ACCESS_TOKEN:-}" ]; then
  if [ -z "${BROKER_AUTH_TOKEN:-}" ]; then
    echo "BROKER_AUTH_TOKEN is required to mint a registration token" >&2
    exit 1
  fi
  if [ -z "${RUNNER_FLEET_ID:-}" ]; then
    echo "RUNNER_FLEET_ID is required" >&2
    exit 1
  fi
  RUNNER_REGISTRATION_TOKEN="$(mint-runner-registration -fleet "${RUNNER_FLEET_ID}" -secret "${BROKER_AUTH_TOKEN}")"
  export RUNNER_REGISTRATION_TOKEN
fi

exec runner
