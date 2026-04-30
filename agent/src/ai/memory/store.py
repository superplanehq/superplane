"""Postgres-backed canvas notes; see table ``agent_canvas_markdown_memory`` and ``SessionStore``."""

from __future__ import annotations

from ai.session_store import SessionStore


def get_canvas_memory_markdown(store: SessionStore, canvas_id: str) -> str:
    return store.get_canvas_memory_markdown(canvas_id)


def set_canvas_memory_markdown(store: SessionStore, canvas_id: str, body: str) -> None:
    store.set_canvas_memory_markdown(canvas_id, body)
