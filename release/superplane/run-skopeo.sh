#!/usr/bin/env bash

set -euo pipefail

exec bash "$(dirname "$0")/../fleet-manager/run-skopeo.sh" "$@"
