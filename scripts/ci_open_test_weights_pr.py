#!/usr/bin/env python3
"""Open or update the weekly test weight PR through the GitHub API."""

import base64
import json
import os
import sys
import urllib.error
import urllib.parse
import urllib.request
from datetime import datetime, timezone
from pathlib import Path

REPO = "superplanehq/superplane"
API = f"https://api.github.com/repos/{REPO}"
FILES = ("scripts/ci_e2e_test_weights.tsv", "scripts/ci_unit_package_weights.tsv")
TITLE = "chore: Refresh CI test weights"
BODY = (
    "## Summary\n\n"
    "Refresh shard weights from recent successful main runs. "
    "This keeps worker load balanced as test times change. "
    "The test discovery guard continues to check test coverage.\n"
)


def api(method, path, data=None):
    token = os.environ["GITHUB_RELEASE_TOKEN"]
    body = json.dumps(data).encode() if data is not None else None
    request = urllib.request.Request(
        API + path,
        data=body,
        method=method,
        headers={
            "Accept": "application/vnd.github+json",
            "Authorization": f"Bearer {token}",
            "Content-Type": "application/json",
            "User-Agent": "SuperPlane-CI",
            "X-GitHub-Api-Version": "2022-11-28",
        },
    )
    try:
        with urllib.request.urlopen(request, timeout=30) as response:
            return json.load(response)
    except urllib.error.HTTPError as error:
        detail = error.read().decode(errors="replace")
        raise RuntimeError(f"GitHub {method} {path}: HTTP {error.code}: {detail}") from error


def main():
    if not os.environ.get("GITHUB_RELEASE_TOKEN"):
        raise ValueError("GITHUB_RELEASE_TOKEN is required")
    query = urllib.parse.urlencode({"state": "open", "base": "main", "per_page": "100"})
    pulls = [pull for pull in api("GET", f"/pulls?{query}")
             if pull["title"] == TITLE and pull["head"]["repo"]["full_name"] == REPO]
    if len(pulls) > 1:
        raise ValueError("multiple open test weight PRs")
    branch = pulls[0]["head"]["ref"] if pulls else "ci/refresh-test-weights"
    main_ref = api("GET", "/git/ref/heads/main")
    main_sha = main_ref["object"]["sha"]
    main_commit = api("GET", f"/git/commits/{main_sha}")

    entries = []
    for path in FILES:
        content = Path(path).read_bytes()
        blob = api("POST", "/git/blobs", {
            "content": base64.b64encode(content).decode(), "encoding": "base64"
        })
        entries.append({"path": path, "mode": "100644", "type": "blob", "sha": blob["sha"]})
    tree = api("POST", "/git/trees", {"base_tree": main_commit["tree"]["sha"], "tree": entries})
    commit = api("POST", "/git/commits", {
        "message": "chore: Refresh CI test weights\n\nUpdate measured shard weights as test times change.\n\nSigned-off-by: SuperPlane CI <ci@superplanehq.com>",
        "tree": tree["sha"],
        "parents": [main_sha],
        "author": {"name": "SuperPlane CI", "email": "ci@superplanehq.com"},
        "committer": {"name": "SuperPlane CI", "email": "ci@superplanehq.com"},
    })

    if pulls:
        api("PATCH", f"/git/refs/heads/{urllib.parse.quote(branch, safe='/')}", {
            "sha": commit["sha"], "force": True
        })
        print(f"Updated {pulls[0]['html_url']}")
        return

    try:
        api("POST", "/git/refs", {"ref": f"refs/heads/{branch}", "sha": commit["sha"]})
    except RuntimeError as error:
        if "HTTP 422" not in str(error):
            raise
        branch = "ci/refresh-test-weights-" + datetime.now(timezone.utc).strftime("%Y%m%d-%H%M%S")
        api("POST", "/git/refs", {"ref": f"refs/heads/{branch}", "sha": commit["sha"]})

    pull = api("POST", "/pulls", {"title": TITLE, "head": branch, "base": "main", "body": BODY})
    print(f"Created {pull['html_url']}")


if __name__ == "__main__":
    try:
        main()
    except (ValueError, RuntimeError, KeyError, OSError) as error:
        print(f"test weight PR: {error}", file=sys.stderr)
        sys.exit(1)
