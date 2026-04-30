from __future__ import annotations

from pathlib import Path


def _skills_root() -> Path:
    return Path(__file__).resolve().parents[2] / "skills"


def _split_front_matter(raw: str) -> tuple[dict[str, str], str]:
    text = raw.lstrip()
    if not text.startswith("---"):
        return {}, raw
    _, rest = text.split("---", 1)
    if "\n---" not in rest:
        return {}, raw
    fm, body = rest.split("\n---", 1)
    meta: dict[str, str] = {}
    for line in fm.strip().splitlines():
        line = line.strip()
        if ":" in line:
            key, _, value = line.partition(":")
            meta[key.strip()] = value.strip()
    return meta, body.lstrip("\n")


def _title_from_body(body: str) -> str:
    for line in body.splitlines():
        stripped = line.strip()
        if stripped.startswith("# "):
            return stripped[2:].strip()
    return ""


def _skill_description(meta: dict[str, str], body: str) -> str:
    desc = meta.get("description", "").strip()
    if desc:
        return desc
    return _title_from_body(body) or "(no description)"


def _iter_skill_files() -> list[Path]:
    root = _skills_root()
    if not root.is_dir():
        return []
    return sorted(root.glob("*.md"))


def skill_index_markdown() -> str:
    lines: list[str] = []
    for path in _iter_skill_files():
        raw = path.read_text(encoding="utf-8")
        meta, body = _split_front_matter(raw)
        skill_name = meta.get("name", path.stem)
        desc = _skill_description(meta, body)
        lines.append(f"- **{skill_name}**: {desc}")
    if not lines:
        return ""
    return "\n\n## Available skills\n\n" + "\n".join(lines)


def get_agent_skill(skill_name: str) -> dict[str, str] | None:
    normalized = skill_name.strip().lower()

    if not normalized:
        return None
    for path in _iter_skill_files():
        if path.stem.lower() != normalized:
            continue
        raw = path.read_text(encoding="utf-8")
        meta, body = _split_front_matter(raw)
        return {
            "id": path.stem,
            "description": _skill_description(meta, body),
            "path": str(path),
            "content": body,
        }
    return None
