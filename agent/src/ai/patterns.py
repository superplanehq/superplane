from __future__ import annotations

import os
import re
from dataclasses import dataclass
from pathlib import Path

_TOKEN_RE = re.compile(r"[a-z0-9]+")
_KEYWORDS_PREFIX = "keywords:"


@dataclass(frozen=True)
class DecisionPattern:
    id: str
    title: str
    path: Path
    content: str
    keywords: set[str]


def _resolve_pattern_dir() -> Path:
    env_value = os.getenv("AGENT_PATTERN_DIR", "").strip()
    if env_value:
        env_dir = Path(env_value).expanduser()
        return env_dir

    # Default to <repo>/agent/patterns
    return Path(__file__).resolve().parents[2] / "patterns"


def _tokenize(value: str) -> set[str]:
    tokens = _TOKEN_RE.findall(value.lower())
    return {token for token in tokens if len(token) > 1}


def _extract_title(content: str, fallback: str) -> str:
    for line in content.splitlines():
        stripped = line.strip()
        if stripped.startswith("# "):
            title = stripped[2:].strip()
            if title:
                return title
    return fallback


def _extract_keywords(content: str) -> set[str]:
    keywords: set[str] = set()
    for line in content.splitlines():
        stripped = line.strip()
        if stripped.lower().startswith(_KEYWORDS_PREFIX):
            _, _, rest = stripped.partition(":")
            for part in rest.split(","):
                cleaned = part.strip().lower()
                if cleaned:
                    keywords.update(_tokenize(cleaned))
    return keywords


def _build_snippet(content: str, max_chars: int = 260) -> str:
    for paragraph in content.split("\n\n"):
        cleaned = " ".join(paragraph.split())
        if cleaned and not cleaned.startswith("#"):
            return cleaned[:max_chars].rstrip()
    return " ".join(content.split())[:max_chars].rstrip()


def load_decision_patterns(pattern_dir: Path | None = None) -> list[DecisionPattern]:
    resolved_dir = pattern_dir or _resolve_pattern_dir()
    if not resolved_dir.exists() or not resolved_dir.is_dir():
        return []

    patterns: list[DecisionPattern] = []
    for file_path in sorted(resolved_dir.glob("*.md")):
        try:
            content = file_path.read_text(encoding="utf-8")
        except OSError:
            continue
        title = _extract_title(content, fallback=file_path.stem.replace("-", " "))
        keywords = _extract_keywords(content)
        keywords.update(_tokenize(file_path.stem))
        keywords.update(_tokenize(title))
        patterns.append(
            DecisionPattern(
                id=file_path.stem,
                title=title,
                path=file_path,
                content=content,
                keywords=keywords,
            )
        )
    return patterns


def list_decision_patterns(
    pattern_dir: Path | None = None,
) -> list[dict[str, str]]:
    items: list[dict[str, str]] = []
    for pattern in load_decision_patterns(pattern_dir):
        items.append(
            {
                "id": pattern.id,
                "title": pattern.title,
                "path": str(pattern.path),
            }
        )
    return items


def search_decision_patterns(
    query: str,
    limit: int = 3,
    pattern_dir: Path | None = None,
) -> list[dict[str, str | int]]:
    normalized_query = query.strip().lower()
    if not normalized_query:
        return []

    query_tokens = _tokenize(normalized_query)
    if not query_tokens:
        return []

    matches: list[tuple[int, DecisionPattern]] = []
    for pattern in load_decision_patterns(pattern_dir):
        content_tokens = _tokenize(pattern.content)
        keyword_hits = len(query_tokens.intersection(pattern.keywords))
        content_hits = len(query_tokens.intersection(content_tokens))
        phrase_bonus = 4 if normalized_query in pattern.content.lower() else 0
        score = (keyword_hits * 3) + content_hits + phrase_bonus
        if score <= 0:
            continue
        matches.append((score, pattern))

    ranked = sorted(matches, key=lambda item: (-item[0], item[1].id))[: max(limit, 1)]
    results: list[dict[str, str | int]] = []
    for score, pattern in ranked:
        results.append(
            {
                "id": pattern.id,
                "title": pattern.title,
                "path": str(pattern.path),
                "score": score,
                "snippet": _build_snippet(pattern.content),
            }
        )
    return results


def get_decision_pattern(
    pattern_id: str,
    pattern_dir: Path | None = None,
) -> dict[str, str] | None:
    normalized_pattern_id = pattern_id.strip().lower()
    if not normalized_pattern_id:
        return None

    for pattern in load_decision_patterns(pattern_dir):
        if pattern.id.lower() != normalized_pattern_id:
            continue
        return {
            "id": pattern.id,
            "title": pattern.title,
            "path": str(pattern.path),
            "content": pattern.content,
        }
    return None
