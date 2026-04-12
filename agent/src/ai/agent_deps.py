from dataclasses import dataclass, field
from typing import Any, Literal

from ai.models import CanvasSummary
from ai.session_store import SessionStore
from ai.superplane_client import SuperplaneClient

CatalogListKind = Literal["components", "triggers"]

@dataclass
class AgentDeps:
    client: SuperplaneClient
    canvas_id: str
    session_store: SessionStore | None = None
    canvas_cache: dict[str, CanvasSummary] = field(default_factory=dict)
    catalog_list_cache: dict[tuple[str, str, str], list[dict[str, Any]]] = field(
        default_factory=dict
    )

def _catalog_list_cache_key(
    kind: CatalogListKind,
    provider: str | None,
    query: str | None,
) -> tuple[str, str, str]:
    p = provider.strip().lower() if isinstance(provider, str) else ""
    q = query.strip().lower() if isinstance(query, str) else ""
    return (kind, p, q)


def _clone_catalog_list_rows(rows: list[dict[str, Any]]) -> list[dict[str, Any]]:
    """Detach cached rows so callers cannot mutate the in-session cache."""
    out: list[dict[str, Any]] = []
    for row in rows:
        cloned = dict(row)
        ocn = cloned.get("output_channel_names")
        if isinstance(ocn, list):
            cloned["output_channel_names"] = list(ocn)
        out.append(cloned)
    return out

def _get_cached_catalog_list(
    deps: AgentDeps,
    kind: CatalogListKind,
    provider: str | None,
    query: str | None,
) -> list[dict[str, Any]] | None:
    key = _catalog_list_cache_key(kind, provider, query)
    hit = deps.catalog_list_cache.get(key)
    if hit is None:
        return None
    return _clone_catalog_list_rows(hit)


def _put_cached_catalog_list(
    deps: AgentDeps,
    kind: CatalogListKind,
    provider: str | None,
    query: str | None,
    rows: list[dict[str, Any]],
) -> None:
    key = _catalog_list_cache_key(kind, provider, query)
    deps.catalog_list_cache[key] = _clone_catalog_list_rows(rows)
