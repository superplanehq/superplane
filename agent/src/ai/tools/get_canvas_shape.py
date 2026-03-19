import time

from pydantic_ai import Agent, RunContext

from ai.deps import (
    AgentDeps,
    color,
    elapsed_since_question_started,
    humanize_bytes,
    print_waiting_for_model_message,
)
from ai.models import CanvasAnswer, CanvasShape


def register(agent: Agent[AgentDeps, CanvasAnswer]) -> None:
    @agent.tool
    def get_canvas_shape(ctx: RunContext[AgentDeps], canvas_id: str | None = None) -> CanvasShape:
        resolved_canvas_id = (canvas_id or ctx.deps.default_canvas_id or "").strip()
        if not resolved_canvas_id:
            raise ValueError("canvas_id is required.")

        started_at = time.perf_counter()
        if ctx.deps.show_tool_calls:
            timestamp = color(elapsed_since_question_started(ctx), "90")
            tool_label = color("[tool]", "36")
            print(
                f"{timestamp} {tool_label} get_canvas_shape(canvas_id={resolved_canvas_id})",
                flush=True,
            )

        cached_shape = ctx.deps.canvas_shape_cache.get(resolved_canvas_id)
        if cached_shape is not None:
            shape = cached_shape
            elapsed_ms = (time.perf_counter() - started_at) * 1000
            if ctx.deps.show_tool_calls:
                timestamp = color(elapsed_since_question_started(ctx), "90")
                tool_label = color("[tool]", "36")
                response_bytes = len(shape.model_dump_json().encode("utf-8"))
                print(
                    f"{timestamp} {tool_label} -> cache_hit=true "
                    f"nodes={shape.node_count} edges={shape.edge_count} "
                    f"tool_elapsed_ms={elapsed_ms:.1f} "
                    f"response_size={humanize_bytes(response_bytes)} "
                    f"({response_bytes} bytes)",
                    flush=True,
                )
                print_waiting_for_model_message(ctx)
            return shape

        shape = ctx.deps.client.get_canvas_shape(resolved_canvas_id)
        ctx.deps.canvas_shape_cache[resolved_canvas_id] = shape
        elapsed_ms = (time.perf_counter() - started_at) * 1000
        if ctx.deps.show_tool_calls:
            timestamp = color(elapsed_since_question_started(ctx), "90")
            tool_label = color("[tool]", "36")
            response_bytes = len(shape.model_dump_json().encode("utf-8"))
            print(
                f"{timestamp} {tool_label} -> cache_hit=false "
                f"nodes={shape.node_count} edges={shape.edge_count} "
                f"tool_elapsed_ms={elapsed_ms:.1f} "
                f"response_size={humanize_bytes(response_bytes)} "
                f"({response_bytes} bytes)",
                flush=True,
            )
            print_waiting_for_model_message(ctx)
        return shape
