#!/usr/bin/env bash
#
# Fast static security checks. No ESLint. No Node import of tool configs.

set -euo pipefail

cd "$(dirname "$0")/.."
root="${1:-.}"

bash ./scripts/check_tool_config_guard.sh "$root"
bash ./scripts/check_ioc_markers.sh "$root"
bash ./scripts/check_install_hooks.sh "$root"

echo "Fast security checks passed."
