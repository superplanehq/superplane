import os
import sys
import time
from dataclasses import dataclass, field

from pydantic_ai import RunContext

from ai.models import CanvasShape, CanvasSummary
from ai.superplane_client import SuperplaneClient


@dataclass
class AgentDeps:
    client: SuperplaneClient
    default_canvas_id: str | None = None
    show_tool_calls: bool = True
    question_started_at: float | None = None
    waiting_message_printed: bool = False
    canvas_cache: dict[str, CanvasSummary] = field(default_factory=dict)
    canvas_shape_cache: dict[str, CanvasShape] = field(default_factory=dict)
    allow_canvas_details: bool = False


def use_color() -> bool:
    return sys.stdout.isatty() and not os.getenv("NO_COLOR")


def color(text: str, ansi_code: str) -> str:
    if not use_color():
        return text
    return f"\033[{ansi_code}m{text}\033[0m"


def format_elapsed_seconds(elapsed_seconds: float) -> str:
    return f"{elapsed_seconds:7.3f}s"


def humanize_bytes(size_in_bytes: int) -> str:
    size = float(size_in_bytes)
    units = ("B", "KiB", "MiB", "GiB", "TiB")
    unit_index = 0
    while size >= 1024 and unit_index < len(units) - 1:
        size /= 1024
        unit_index += 1
    if unit_index == 0:
        return f"{int(size)} {units[unit_index]}"
    return f"{size:.1f} {units[unit_index]}"


def elapsed_since_question_started(ctx: RunContext[AgentDeps]) -> str:
    started_at = ctx.deps.question_started_at
    if started_at is None:
        return format_elapsed_seconds(0.0)
    return format_elapsed_seconds(time.perf_counter() - started_at)


def print_waiting_for_model_message(ctx: RunContext[AgentDeps]) -> None:
    if ctx.deps.waiting_message_printed:
        return
    timestamp = color(elapsed_since_question_started(ctx), "90")
    status = color("[status]", "33")
    print(
        f"{timestamp} {status} Tools completed. Generating final answer...",
        flush=True,
    )
    ctx.deps.waiting_message_printed = True
