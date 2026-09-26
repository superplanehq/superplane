#!/usr/bin/env bash
# Download task files listed in attachments/manifest.json.
# Verify size and checksum when the manifest includes them.
set -euo pipefail

attachments="${SUPERPLANE_TASK_DIR:?}/attachments"
manifest="$attachments/manifest.json"
if [ ! -f "$manifest" ]; then
  printf 'attachments/manifest.json is missing.\n' >&2
  exit 1
fi

mkdir -p "$attachments"
export ATTACHMENTS_DIR="$attachments"
export MANIFEST_PATH="$manifest"

python3 - <<'PY'
import hashlib
import json
import os
import subprocess
import sys

attachments = os.environ["ATTACHMENTS_DIR"]
manifest_path = os.environ["MANIFEST_PATH"]
with open(manifest_path, encoding="utf-8") as handle:
    manifest = json.load(handle)

changed = False
for item in manifest.get("files") or []:
    url = (item.get("url") or "").strip()
    dest_name = (item.get("dest") or "").strip()
    if not url or not dest_name or "/" in dest_name or dest_name in (".", ".."):
        print(f"invalid attachment dest {dest_name!r}", file=sys.stderr)
        sys.exit(1)
    dest = os.path.join(attachments, dest_name)
    expected_size = int(item.get("size_bytes") or 0)
    expected_checksum = (item.get("checksum") or "").strip().lower()

    if os.path.isfile(dest):
        digest = hashlib.sha256(open(dest, "rb").read()).hexdigest()
        size = os.path.getsize(dest)
        if expected_checksum and digest != expected_checksum:
            os.remove(dest)
        elif expected_size and size != expected_size:
            os.remove(dest)
        else:
            item["status"] = item.get("status") or "downloaded"
            continue

    tmp = dest + ".part"
    try:
        subprocess.run(
            ["curl", "-fsSL", "-o", tmp, url],
            check=True,
        )
    except subprocess.CalledProcessError as err:
        print(f"download failed for {dest_name}: {err}", file=sys.stderr)
        sys.exit(1)

    size = os.path.getsize(tmp)
    digest = hashlib.sha256(open(tmp, "rb").read()).hexdigest()
    if expected_size and size != expected_size:
        os.remove(tmp)
        print(f"{dest_name} size {size} does not match {expected_size}", file=sys.stderr)
        sys.exit(1)
    if expected_checksum and digest != expected_checksum:
        os.remove(tmp)
        print(f"{dest_name} checksum mismatch", file=sys.stderr)
        sys.exit(1)
    os.replace(tmp, dest)
    item["status"] = "downloaded"
    changed = True
    print(f"downloaded {dest_name} ({size} bytes)")

if changed or True:
    with open(manifest_path, "w", encoding="utf-8") as handle:
        json.dump(manifest, handle, indent=2)
        handle.write("\n")
PY
