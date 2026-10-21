#!/bin/sh

set -eu

if [ "${1:-}" = "sync-fleets" ] || [ "${1:-}" = "--version" ] || [ "${1:-}" = "--help" ]; then
  exec /usr/local/bin/fleet-manager "$@"
fi

/usr/local/bin/fleet-manager sync-fleets
exec /usr/local/bin/fleet-manager "$@"
