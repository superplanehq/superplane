"""Human-readable labels for agent tool calls (stream UI and chat history)."""

import json
from typing import Any

_LABEL_BY_TOOL: dict[str, str] = {
    "get_canvas": "Reading canvas",
    "get_canvas_memory": "Loading canvas notes",
    "get_canvas_shape": "Reading canvas structure",
    "get_canvas_details": "Reading canvas details",
    "get_node_details": "Reading node details",
    "list_node_events": "Listing node events",
    "list_node_executions": "Listing node executions",
    "list_available_blocks": "Listing available components",
    "list_components": "Listing available components",
}


def _coerce_args(args: Any) -> dict[str, Any] | None:
    if isinstance(args, dict):
        return args
    if isinstance(args, str) and args.strip():
        try:
            parsed = json.loads(args)
        except json.JSONDecodeError:
            return None
        return parsed if isinstance(parsed, dict) else None
    return None


def format_tool_display_label(tool_name: str | None, args: Any = None) -> str:
    normalized = (tool_name or "").strip().lower()
    d = _coerce_args(args)

    if normalized in _LABEL_BY_TOOL:
        base = _LABEL_BY_TOOL[normalized]
    else:
        words = normalized.replace("_", " ").replace("-", " ").strip()
        base = (words[:1].upper() + words[1:]) if words else "Running tool"

    if d and normalized in {"get_node_details", "list_node_events", "list_node_executions"}:
        raw = d.get("node_id")
        if isinstance(raw, str) and raw.strip():
            nid = raw.strip()
            display = nid if len(nid) <= 16 else f"{nid[:8]}…"
            return f"{base} ({display})"

    return base
