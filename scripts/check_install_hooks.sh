#!/usr/bin/env bash
#
# Fail if root or web_src package.json grows a lifecycle install hook.
# Those hooks run on npm install in CI (including make dev.setup).

set -euo pipefail

cd "$(dirname "$0")/.."
root="${1:-.}"

python3 - "$root" <<'PY'
import json
import sys
from pathlib import Path

root = Path(sys.argv[1])
hook_names = ("preinstall", "install", "postinstall")
manifests = [root / "package.json", root / "web_src" / "package.json"]
failed = False

for path in manifests:
    if not path.is_file():
        continue
    data = json.loads(path.read_text())
    scripts = data.get("scripts") or {}
    found = [name for name in hook_names if name in scripts]
    if found:
        print(f"Forbidden install hook(s) in {path}: {', '.join(found)}", file=sys.stderr)
        failed = True

if failed:
    sys.exit(1)

print("Install hook scan passed.")
PY
