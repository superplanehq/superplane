import time

from pydantic_ai import Agent, RunContext

from ai.deps import (
    AgentDeps,
    color,
    elapsed_since_question_started,
    humanize_bytes,
    print_waiting_for_model_message,
)
from ai.models import CanvasAnswer, CanvasSummary


def register(agent: Agent[AgentDeps, CanvasAnswer]) -> None:
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
            timestamp = color(elapsed_since_question_started(ctx), "90")
            tool_label = color("[tool]", "36")
            print(
                f"{timestamp} {tool_label} get_canvas(canvas_id={resolved_canvas_id})",
                flush=True,
            )

        cached_summary = ctx.deps.canvas_cache.get(resolved_canvas_id)
        if cached_summary is not None:
            summary = cached_summary
            elapsed_ms = (time.perf_counter() - started_at) * 1000
            if ctx.deps.show_tool_calls:
                timestamp = color(elapsed_since_question_started(ctx), "90")
                tool_label = color("[tool]", "36")
                response_bytes = len(summary.model_dump_json().encode("utf-8"))
                print(
                    f"{timestamp} {tool_label} -> cache_hit=true "
                    f"nodes={len(summary.nodes)} edges={len(summary.edges)} "
                    f"tool_elapsed_ms={elapsed_ms:.1f} "
                    f"response_size={humanize_bytes(response_bytes)} "
                    f"({response_bytes} bytes)",
                    flush=True,
                )
                print_waiting_for_model_message(ctx)
            return summary

        summary = ctx.deps.client.describe_canvas(resolved_canvas_id)
        ctx.deps.canvas_cache[resolved_canvas_id] = summary
        elapsed_ms = (time.perf_counter() - started_at) * 1000
        if ctx.deps.show_tool_calls:
            timestamp = color(elapsed_since_question_started(ctx), "90")
            tool_label = color("[tool]", "36")
            response_bytes = len(summary.model_dump_json().encode("utf-8"))
            print(
                f"{timestamp} {tool_label} -> cache_hit=false "
                f"nodes={len(summary.nodes)} "
                f"edges={len(summary.edges)} tool_elapsed_ms={elapsed_ms:.1f} "
                f"response_size={humanize_bytes(response_bytes)} "
                f"({response_bytes} bytes)",
                flush=True,
            )
            print_waiting_for_model_message(ctx)
        return summary
