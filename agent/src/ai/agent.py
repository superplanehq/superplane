import os
import sys
import time
from dataclasses import dataclass, field
from typing import Literal

from pydantic_ai import Agent, RunContext
from pydantic_ai.models.test import TestModel

from ai.models import CanvasAnswer, CanvasQuestionRequest, CanvasShape, CanvasSummary, NodeDetails
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


def _use_color() -> bool:
    return sys.stdout.isatty() and not os.getenv("NO_COLOR")


def _color(text: str, ansi_code: str) -> str:
    if not _use_color():
        return text
    return f"\033[{ansi_code}m{text}\033[0m"


def _format_elapsed_seconds(elapsed_seconds: float) -> str:
    return f"{elapsed_seconds:7.3f}s"


def _humanize_bytes(size_in_bytes: int) -> str:
    size = float(size_in_bytes)
    units = ("B", "KiB", "MiB", "GiB", "TiB")
    unit_index = 0
    while size >= 1024 and unit_index < len(units) - 1:
        size /= 1024
        unit_index += 1
    if unit_index == 0:
        return f"{int(size)} {units[unit_index]}"
    return f"{size:.1f} {units[unit_index]}"


def _elapsed_since_question_started(ctx: RunContext[AgentDeps]) -> str:
    started_at = ctx.deps.question_started_at
    if started_at is None:
        return _format_elapsed_seconds(0.0)
    return _format_elapsed_seconds(time.perf_counter() - started_at)


def _print_waiting_for_model_message(ctx: RunContext[AgentDeps]) -> None:
    if ctx.deps.waiting_message_printed:
        return
    timestamp = _color(_elapsed_since_question_started(ctx), "90")
    status = _color("[status]", "33")
    print(
        f"{timestamp} {status} Tools completed. Generating final answer...",
        flush=True,
    )
    ctx.deps.waiting_message_printed = True


def build_prompt(payload: CanvasQuestionRequest) -> str:
    return payload.question


def build_agent(model: str | Literal["test"] = "test") -> Agent[AgentDeps, CanvasAnswer]:
    resolved_model: str | TestModel
    if model == "test":
        resolved_model = TestModel()
    else:
        resolved_model = model

    agent: Agent[AgentDeps, CanvasAnswer] = Agent(
        model=resolved_model,
        output_type=CanvasAnswer,
        system_prompt=(
            "You answer questions about Superplane canvases. "
            "Use tools to fetch real canvas data before answering. "
            "Be concise and factual. Return citations when possible. "
            "Use get_canvas_shape first for structure/topology questions. "
            "Full canvas details are gated: call request_canvas_details with a brief "
            "reason before calling get_canvas. "
            "Use get_canvas at most once per answer unless the user asks to refresh "
            "or use a different canvas. "
            "Keep responses short by default (about 6-10 lines) unless the user asks "
            "for deep detail."
        ),
    )

    @agent.tool
    def get_canvas_shape(ctx: RunContext[AgentDeps], canvas_id: str | None = None) -> CanvasShape:
        resolved_canvas_id = (canvas_id or ctx.deps.default_canvas_id or "").strip()
        if not resolved_canvas_id:
            raise ValueError("canvas_id is required.")
        started_at = time.perf_counter()
        if ctx.deps.show_tool_calls:
            timestamp = _color(_elapsed_since_question_started(ctx), "90")
            tool_label = _color("[tool]", "36")
            print(
                f"{timestamp} {tool_label} get_canvas_shape(canvas_id={resolved_canvas_id})",
                flush=True,
            )
        cached_shape = ctx.deps.canvas_shape_cache.get(resolved_canvas_id)
        if cached_shape is not None:
            shape = cached_shape
            elapsed_ms = (time.perf_counter() - started_at) * 1000
            if ctx.deps.show_tool_calls:
                timestamp = _color(_elapsed_since_question_started(ctx), "90")
                tool_label = _color("[tool]", "36")
                response_bytes = len(shape.model_dump_json().encode("utf-8"))
                print(
                    f"{timestamp} {tool_label} -> cache_hit=true "
                    f"nodes={shape.node_count} edges={shape.edge_count} "
                    f"tool_elapsed_ms={elapsed_ms:.1f} "
                    f"response_size={_humanize_bytes(response_bytes)} "
                    f"({response_bytes} bytes)",
                    flush=True,
                )
                _print_waiting_for_model_message(ctx)
            return shape

        shape = ctx.deps.client.get_canvas_shape(resolved_canvas_id)
        ctx.deps.canvas_shape_cache[resolved_canvas_id] = shape
        elapsed_ms = (time.perf_counter() - started_at) * 1000
        if ctx.deps.show_tool_calls:
            timestamp = _color(_elapsed_since_question_started(ctx), "90")
            tool_label = _color("[tool]", "36")
            response_bytes = len(shape.model_dump_json().encode("utf-8"))
            print(
                f"{timestamp} {tool_label} -> cache_hit=false "
                f"nodes={shape.node_count} edges={shape.edge_count} "
                f"tool_elapsed_ms={elapsed_ms:.1f} "
                f"response_size={_humanize_bytes(response_bytes)} "
                f"({response_bytes} bytes)",
                flush=True,
            )
            _print_waiting_for_model_message(ctx)
        return shape

    @agent.tool
    def request_canvas_details(ctx: RunContext[AgentDeps], reason: str = "") -> str:
        if ctx.deps.show_tool_calls:
            timestamp = _color(_elapsed_since_question_started(ctx), "90")
            status_label = _color("[status]", "33")
            printable_reason = reason.strip() or "no reason provided"
            print(
                f"{timestamp} {status_label} request_canvas_details(reason={printable_reason})",
                flush=True,
            )
        ctx.deps.allow_canvas_details = True
        return "Full canvas details enabled for this question."

    @agent.tool
    def get_canvas(ctx: RunContext[AgentDeps], canvas_id: str | None = None) -> CanvasSummary:
        resolved_canvas_id = (canvas_id or ctx.deps.default_canvas_id or "").strip()
        if not resolved_canvas_id:
            raise ValueError("canvas_id is required.")
        if not ctx.deps.allow_canvas_details:
            raise ValueError(
                "Full canvas details are locked. "
                "Call request_canvas_details with a brief reason first."
            )
        started_at = time.perf_counter()
        if ctx.deps.show_tool_calls:
            timestamp = _color(_elapsed_since_question_started(ctx), "90")
            tool_label = _color("[tool]", "36")
            print(
                f"{timestamp} {tool_label} get_canvas(canvas_id={resolved_canvas_id})",
                flush=True,
            )
        cached_summary = ctx.deps.canvas_cache.get(resolved_canvas_id)
        if cached_summary is not None:
            summary = cached_summary
            elapsed_ms = (time.perf_counter() - started_at) * 1000
            if ctx.deps.show_tool_calls:
                timestamp = _color(_elapsed_since_question_started(ctx), "90")
                tool_label = _color("[tool]", "36")
                response_bytes = len(summary.model_dump_json().encode("utf-8"))
                print(
                    f"{timestamp} {tool_label} -> cache_hit=true "
                    f"nodes={len(summary.nodes)} edges={len(summary.edges)} "
                    f"tool_elapsed_ms={elapsed_ms:.1f} "
                    f"response_size={_humanize_bytes(response_bytes)} "
                    f"({response_bytes} bytes)",
                    flush=True,
                )
                _print_waiting_for_model_message(ctx)
            return summary

        summary = ctx.deps.client.describe_canvas(resolved_canvas_id)
        ctx.deps.canvas_cache[resolved_canvas_id] = summary
        elapsed_ms = (time.perf_counter() - started_at) * 1000
        if ctx.deps.show_tool_calls:
            timestamp = _color(_elapsed_since_question_started(ctx), "90")
            tool_label = _color("[tool]", "36")
            response_bytes = len(summary.model_dump_json().encode("utf-8"))
            print(
                f"{timestamp} {tool_label} -> cache_hit=false "
                f"nodes={len(summary.nodes)} "
                f"edges={len(summary.edges)} tool_elapsed_ms={elapsed_ms:.1f} "
                f"response_size={_humanize_bytes(response_bytes)} "
                f"({response_bytes} bytes)",
                flush=True,
            )
            _print_waiting_for_model_message(ctx)
        return summary

    @agent.tool
    def get_node_details(
        ctx: RunContext[AgentDeps],
        node_id: str,
        canvas_id: str | None = None,
        include_recent_events: bool = True,
    ) -> NodeDetails:
        resolved_canvas_id = (canvas_id or ctx.deps.default_canvas_id or "").strip()
        if not resolved_canvas_id:
            raise ValueError("canvas_id is required.")
        started_at = time.perf_counter()
        if ctx.deps.show_tool_calls:
            timestamp = _color(_elapsed_since_question_started(ctx), "90")
            tool_label = _color("[tool]", "36")
            print(
                f"{timestamp} {tool_label} get_node_details("
                f"canvas_id={resolved_canvas_id}, node_id={node_id}, "
                f"include_recent_events={include_recent_events})",
                flush=True,
            )
        details = ctx.deps.client.get_node_details(
            canvas_id=resolved_canvas_id,
            node_id=node_id,
            include_recent_events=include_recent_events,
        )
        elapsed_ms = (time.perf_counter() - started_at) * 1000
        if ctx.deps.show_tool_calls:
            timestamp = _color(_elapsed_since_question_started(ctx), "90")
            tool_label = _color("[tool]", "36")
            response_bytes = len(details.model_dump_json().encode("utf-8"))
            print(
                f"{timestamp} {tool_label} -> "
                f"node={details.node.name or details.node.id} "
                f"recent_events={len(details.recent_events)} "
                f"tool_elapsed_ms={elapsed_ms:.1f} "
                f"response_size={_humanize_bytes(response_bytes)} "
                f"({response_bytes} bytes)",
                flush=True,
            )
            _print_waiting_for_model_message(ctx)
        return details

    return agent
