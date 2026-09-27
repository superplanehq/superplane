#!/usr/bin/env bash
# Download task files listed in attachments/manifest.json.
# Verify size and checksum when the manifest includes them.
# A download failure marks that file failed and continues so the agent can start.
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
FETCH_ATTEMPTS = 3

with open(manifest_path, encoding="utf-8") as handle:
    manifest = json.load(handle)


def mark_failed(item, reason, message):
    item["status"] = "failed"
    item["reason"] = reason
    print(message, file=sys.stderr)


def download(url, tmp):
    last_error = None
    for attempt in range(1, FETCH_ATTEMPTS + 1):
        try:
            subprocess.run(["curl", "-fsSL", "-o", tmp, url], check=True)
            return ""
        except subprocess.CalledProcessError as err:
            last_error = err
            if os.path.isfile(tmp):
                os.remove(tmp)
            if attempt < FETCH_ATTEMPTS:
                print(f"download attempt {attempt} failed; retrying: {err}", file=sys.stderr)
    return str(last_error)


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
    error = download(url, tmp)
    if error:
        mark_failed(item, "download_failed", f"download failed for {dest_name}: {error}")
        continue

    size = os.path.getsize(tmp)
    digest = hashlib.sha256(open(tmp, "rb").read()).hexdigest()
    if expected_size and size != expected_size:
        os.remove(tmp)
        mark_failed(item, "size_mismatch", f"{dest_name} size {size} does not match {expected_size}")
        continue
    if expected_checksum and digest != expected_checksum:
        os.remove(tmp)
        mark_failed(item, "checksum_mismatch", f"{dest_name} checksum mismatch")
        continue
    os.replace(tmp, dest)
    item["status"] = "downloaded"
    print(f"downloaded {dest_name} ({size} bytes)")

with open(manifest_path, "w", encoding="utf-8") as handle:
    json.dump(manifest, handle, indent=2)
    handle.write("\n")
PY
