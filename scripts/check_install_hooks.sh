#!/usr/bin/env bash
#
# Fail if root or web_src package.json grows a lifecycle install hook.
# Those hooks run on npm install in CI (including make dev.setup).

set -euo pipefail

cd "$(dirname "$0")/.."
root="${1:-.}"

echo "==> Install hook scan"
echo "    Forbids preinstall, install, and postinstall on root and web_src."

python3 - "$root" <<'PY'
import json
import sys
from pathlib import Path

root = Path(sys.argv[1])
hook_names = ("preinstall", "install", "postinstall")
manifests = [root / "package.json", root / "web_src" / "package.json"]
failed = False
checked = 0

for path in manifests:
    if not path.is_file():
        print(f"SKIP  {path} (missing)")
        continue
    checked += 1
    data = json.loads(path.read_text())
    scripts = data.get("scripts") or {}
    found = [name for name in hook_names if name in scripts]
    if found:
        print(f"FAIL  {path}")
        print(f"      hooks: {', '.join(found)}")
        failed = True
    else:
        print(f"OK    {path}")

if failed:
    print("==> Install hook scan: FAIL")
    sys.exit(1)

print(f"==> Install hook scan: PASS ({checked} manifests)")
PY
