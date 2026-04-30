"""Records tool names invoked during eval case runs, keyed by case input string.

``Dataset.evaluate`` may run cases concurrently; this registry is keyed by the unique
case ``inputs`` string (duplicate inputs are rejected elsewhere in the eval harness).
"""

from __future__ import annotations

from threading import Lock

_lock = Lock()
_tool_calls_by_question: dict[str, list[str]] = {}


def record_tool_call(question: str, tool_name: str | None) -> None:
    if not tool_name:
        return
    with _lock:
        _tool_calls_by_question.setdefault(question, []).append(tool_name)


def count_tool_calls(question: str, tool_name: str) -> int:
    with _lock:
        return _tool_calls_by_question.get(question, []).count(tool_name)


def clear_tool_call_registry() -> None:
    with _lock:
        _tool_calls_by_question.clear()
