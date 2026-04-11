"""Retain references to fire-and-forget asyncio tasks (e.g. post-run memory merge)."""

from __future__ import annotations

import asyncio
from typing import Any

_tasks: set[asyncio.Task[Any]] = set()


def register_background_task(task: asyncio.Task[Any]) -> None:
    _tasks.add(task)
    task.add_done_callback(_tasks.discard)
