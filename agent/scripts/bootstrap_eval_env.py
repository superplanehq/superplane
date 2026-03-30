#!/usr/bin/env python3
"""Provision org-scoped eval resources and print env lines for agent/.env.

Uses cookie-based session auth (owner setup or password login), then creates or
reuses a canvas and service account via the public HTTP API.

Typical use (inside the agent container, app reachable at http://app:8000):

  uv run --group dev python scripts/bootstrap_eval_env.py

See agent/.env.example and agent/README.md for required env vars.
"""

from __future__ import annotations

import argparse
import json
import os
import re
import sys
from pathlib import Path
from typing import Any

import httpx

DEFAULT_CANVAS_NAME = "Agent evals"
DEFAULT_SA_NAME = "agent-evals"

# Keys written by this script (order for appended block when key missing).
_MERGE_KEYS: tuple[str, ...] = (
    "EVAL_ORG_ID",
    "EVAL_CANVAS_ID",
    "SUPERPLANE_API_TOKEN",
    "SUPERPLANE_BASE_URL",
)

_ENV_LINE_KEY = re.compile(r"^\s*([A-Za-z_][A-Za-z0-9_]*)\s*=")


def _escape_env_double_quoted(value: str) -> str:
    return value.replace("\\", "\\\\").replace('"', '\\"')


def merge_env_file(path: Path, updates: dict[str, str]) -> None:
    """Update or insert keys in a .env file; preserve other lines and order."""
    path = path.expanduser().resolve()
    keys_set = set(updates.keys())
    raw = path.read_text(encoding="utf-8") if path.exists() else ""
    lines = raw.splitlines()
    out: list[str] = []
    placed: set[str] = set()

    for line in lines:
        match = _ENV_LINE_KEY.match(line)
        if match:
            key = match.group(1)
            if key in keys_set:
                if key not in placed:
                    out.append(f'{key}="{_escape_env_double_quoted(updates[key])}"')
                    placed.add(key)
                # drop duplicate definitions of the same key
                continue
        out.append(line)

    missing = [k for k in _MERGE_KEYS if k in updates and k not in placed]
    if missing:
        if out and out[-1].strip() != "":
            out.append("")
        out.append("# --- Agent evals (from scripts/bootstrap_eval_env.py) ---")
        for key in missing:
            out.append(f'{key}="{_escape_env_double_quoted(updates[key])}"')
        out.append("")

    text = "\n".join(out)
    if not text.endswith("\n"):
        text += "\n"
    tmp = path.with_name(path.name + ".tmp")
    tmp.write_text(text, encoding="utf-8")
    os.replace(tmp, path)


def _die(msg: str, *, detail: str | None = None) -> None:
    print(msg, file=sys.stderr)
    if detail:
        print(detail, file=sys.stderr)
    raise SystemExit(1)


def _env(name: str, default: str = "") -> str:
    return os.getenv(name, default).strip()


def _org_headers(org_id: str) -> dict[str, str]:
    return {"x-organization-id": org_id}


def _try_owner_setup(client: httpx.Client) -> tuple[bool, str | None]:
    """POST /api/v1/setup-owner when all owner fields are set.

    Returns (authenticated_via_setup, organization_id_or_none).
    """
    email = _env("EVAL_BOOTSTRAP_OWNER_EMAIL")
    password = _env("EVAL_BOOTSTRAP_OWNER_PASSWORD")
    first = _env("EVAL_BOOTSTRAP_OWNER_FIRST_NAME")
    last = _env("EVAL_BOOTSTRAP_OWNER_LAST_NAME")
    if not (email and password and first and last):
        return False, None

    body = {
        "email": email,
        "first_name": first,
        "last_name": last,
        "password": password,
        "smtp_enabled": False,
    }
    response = client.post("/api/v1/setup-owner", json=body)
    if response.status_code == 200:
        data = response.json()
        org_id = data.get("organization_id")
        if not org_id or not isinstance(org_id, str):
            _die("setup-owner succeeded but response missing organization_id", detail=response.text)
        return True, org_id
    if response.status_code == 409:
        return False, None
    if response.status_code == 404:
        # Owner setup disabled (e.g. non-dev); use password login instead.
        return False, None
    _die(
        f"setup-owner failed (HTTP {response.status_code})",
        detail=response.text[:2000],
    )


def _password_login(client: httpx.Client) -> None:
    email = _env("EVAL_BOOTSTRAP_EMAIL") or _env("EVAL_BOOTSTRAP_OWNER_EMAIL")
    password = _env("EVAL_BOOTSTRAP_PASSWORD") or _env("EVAL_BOOTSTRAP_OWNER_PASSWORD")
    if not email or not password:
        _die(
            "Password login required: set EVAL_BOOTSTRAP_EMAIL and EVAL_BOOTSTRAP_PASSWORD "
            "in agent/.env (or EVAL_BOOTSTRAP_OWNER_EMAIL / EVAL_BOOTSTRAP_OWNER_PASSWORD) "
            "when the instance is already initialized. See agent/.env.example."
        )
    response = client.post(
        "/login",
        data={"email": email, "password": password},
        headers={"Content-Type": "application/x-www-form-urlencoded"},
    )
    if response.status_code not in (200, 303, 302):
        _die(f"login failed (HTTP {response.status_code})", detail=response.text[:2000])


def _pick_org_id(client: httpx.Client, preferred: str) -> str:
    response = client.get("/organizations")
    if response.status_code != 200:
        _die(
            f"GET /organizations failed (HTTP {response.status_code})",
            detail=response.text[:2000],
        )
    orgs: list[dict[str, Any]] = response.json()
    if not orgs:
        _die("No organizations for this account; complete owner setup or use a valid login.")
    if preferred:
        for org in orgs:
            if org.get("id") == preferred:
                return preferred
        _die(
            f"EVAL_BOOTSTRAP_ORG_ID={preferred!r} not found in your organizations.",
            detail="Available: " + ", ".join(o.get("id", "?") for o in orgs),
        )
    org_id = orgs[0].get("id")
    if not org_id:
        _die("Unexpected /organizations response: missing id", detail=json.dumps(orgs)[:2000])
    return org_id


def _list_canvases(client: httpx.Client, org_id: str) -> list[dict[str, Any]]:
    response = client.get(
        "/api/v1/canvases",
        headers=_org_headers(org_id),
        params={"includeTemplates": "false"},
    )
    if response.status_code != 200:
        _die(f"list canvases failed (HTTP {response.status_code})", detail=response.text[:2000])
    data = response.json()
    raw = data.get("canvases")
    if not isinstance(raw, list):
        _die("Unexpected list canvases response", detail=json.dumps(data)[:2000])
    return raw


def _canvas_id_by_name(canvases: list[dict[str, Any]], name: str) -> str | None:
    for c in canvases:
        meta = c.get("metadata") or {}
        if meta.get("name") == name:
            cid = meta.get("id")
            if isinstance(cid, str):
                return cid
    return None


def _create_canvas(client: httpx.Client, org_id: str, name: str, description: str) -> str:
    payload = {
        "canvas": {
            "metadata": {"name": name, "description": description},
            "spec": {"nodes": [], "edges": []},
        }
    }
    response = client.post(
        "/api/v1/canvases",
        headers=_org_headers(org_id),
        json=payload,
    )
    if response.status_code == 409:
        return ""
    if response.status_code != 200:
        _die(f"create canvas failed (HTTP {response.status_code})", detail=response.text[:2000])
    data = response.json()
    canvas = data.get("canvas") or {}
    meta = canvas.get("metadata") or {}
    cid = meta.get("id")
    if not isinstance(cid, str):
        _die("create canvas: missing canvas.metadata.id", detail=json.dumps(data)[:2000])
    return cid


def _ensure_canvas(client: httpx.Client, org_id: str, name: str, description: str) -> str:
    canvases = _list_canvases(client, org_id)
    existing = _canvas_id_by_name(canvases, name)
    if existing:
        return existing
    new_id = _create_canvas(client, org_id, name, description)
    if new_id:
        return new_id
    canvases = _list_canvases(client, org_id)
    existing = _canvas_id_by_name(canvases, name)
    if existing:
        return existing
    _die("Canvas create returned conflict but canvas not found by name after list.")


def _list_service_accounts(client: httpx.Client, org_id: str) -> list[dict[str, Any]]:
    response = client.get("/api/v1/service-accounts", headers=_org_headers(org_id))
    if response.status_code != 200:
        _die(
            f"list service accounts failed (HTTP {response.status_code})",
            detail=response.text[:2000],
        )
    data = response.json()
    raw = data.get("serviceAccounts")
    if not isinstance(raw, list):
        _die("Unexpected list service accounts response", detail=json.dumps(data)[:2000])
    return raw


def _sa_id_by_name(accounts: list[dict[str, Any]], name: str) -> str | None:
    for sa in accounts:
        if sa.get("name") == name:
            sid = sa.get("id")
            if isinstance(sid, str):
                return sid
    return None


def _create_service_account(client: httpx.Client, org_id: str, name: str, description: str) -> str:
    payload = {
        "name": name,
        "description": description,
        "role": "org_admin",
    }
    response = client.post(
        "/api/v1/service-accounts",
        headers=_org_headers(org_id),
        json=payload,
    )
    if response.status_code != 200:
        _die(
            f"create service account failed (HTTP {response.status_code})",
            detail=response.text[:2000],
        )
    data = response.json()
    token = data.get("token")
    if not isinstance(token, str) or not token:
        _die("create service account: missing token in response", detail=json.dumps(data)[:2000])
    return token


def _regenerate_sa_token(client: httpx.Client, org_id: str, sa_id: str) -> str:
    response = client.post(
        f"/api/v1/service-accounts/{sa_id}/token",
        headers=_org_headers(org_id),
    )
    if response.status_code != 200:
        _die(
            f"regenerate service account token failed (HTTP {response.status_code})",
            detail=response.text[:2000],
        )
    data = response.json()
    token = data.get("token")
    if not isinstance(token, str) or not token:
        _die("regenerate token: missing token", detail=json.dumps(data)[:2000])
    return token


def _ensure_service_account_token(
    client: httpx.Client,
    org_id: str,
    name: str,
    description: str,
) -> str:
    accounts = _list_service_accounts(client, org_id)
    sa_id = _sa_id_by_name(accounts, name)
    if sa_id:
        return _regenerate_sa_token(client, org_id, sa_id)
    return _create_service_account(client, org_id, name, description)


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--base-url",
        default="",
        help="Override SUPERPLANE_BASE_URL (default from env or http://app:8000).",
    )
    out_group = parser.add_mutually_exclusive_group()
    out_group.add_argument(
        "-o",
        "--output-file",
        default="",
        help="Append eval env lines to this file (UTF-8). Does not remove duplicates.",
    )
    out_group.add_argument(
        "--merge-env",
        metavar="PATH",
        default="",
        help=(
            "Merge EVAL_ORG_ID, EVAL_CANVAS_ID, SUPERPLANE_API_TOKEN, SUPERPLANE_BASE_URL "
            "into this .env (update existing keys or append a marked block)."
        ),
    )
    args = parser.parse_args()

    base = (args.base_url or _env("SUPERPLANE_BASE_URL", "http://app:8000")).rstrip("/")
    if not base:
        _die("SUPERPLANE_BASE_URL is empty.")

    canvas_name = _env("EVAL_BOOTSTRAP_CANVAS_NAME", DEFAULT_CANVAS_NAME)
    sa_name = _env("EVAL_BOOTSTRAP_SERVICE_ACCOUNT_NAME", DEFAULT_SA_NAME)
    canvas_description = _env(
        "EVAL_BOOTSTRAP_CANVAS_DESCRIPTION",
        "Empty canvas used by agent eval runs (bootstrap_eval_env.py).",
    )
    sa_description = _env(
        "EVAL_BOOTSTRAP_SERVICE_ACCOUNT_DESCRIPTION",
        "API access for local agent eval runs.",
    )
    preferred_org = _env("EVAL_BOOTSTRAP_ORG_ID")

    with httpx.Client(base_url=base, follow_redirects=True, timeout=60.0) as client:
        via_setup, org_from_setup = _try_owner_setup(client)
        if not via_setup:
            _password_login(client)

        org_id = org_from_setup if via_setup else _pick_org_id(client, preferred_org)
        if via_setup and preferred_org and preferred_org != org_id:
            print(
                f"Note: using organization_id from setup-owner ({org_id}); "
                f"EVAL_BOOTSTRAP_ORG_ID={preferred_org!r} ignored.",
                file=sys.stderr,
            )

        canvas_id = _ensure_canvas(client, org_id, canvas_name, canvas_description)
        token = _ensure_service_account_token(client, org_id, sa_name, sa_description)

    lines = [
        "",
        "# --- Agent evals (from scripts/bootstrap_eval_env.py) ---",
        f'EVAL_ORG_ID="{org_id}"',
        f'EVAL_CANVAS_ID="{canvas_id}"',
        f'SUPERPLANE_API_TOKEN="{token}"',
        f'SUPERPLANE_BASE_URL="{base}"',
        "",
    ]
    text = "\n".join(lines)
    print(text)

    updates = {
        "EVAL_ORG_ID": org_id,
        "EVAL_CANVAS_ID": canvas_id,
        "SUPERPLANE_API_TOKEN": token,
        "SUPERPLANE_BASE_URL": base,
    }
    if args.merge_env:
        merge_path = Path(args.merge_env)
        merge_env_file(merge_path, updates)
        print(f"Merged eval keys into {merge_path}", file=sys.stderr)
    if args.output_file:
        path = os.path.expanduser(args.output_file)
        with open(path, "a", encoding="utf-8") as handle:
            handle.write(text)
        print(f"Appended to {path}", file=sys.stderr)


if __name__ == "__main__":
    main()
