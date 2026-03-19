import time

from pydantic_ai import Agent, RunContext

from ai.deps import (
    AgentDeps,
    color,
    elapsed_since_question_started,
    humanize_bytes,
    print_waiting_for_model_message,
)
from ai.models import CanvasAnswer, NodeDetails


def register(agent: Agent[AgentDeps, CanvasAnswer]) -> None:
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
            timestamp = color(elapsed_since_question_started(ctx), "90")
            tool_label = color("[tool]", "36")
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
            timestamp = color(elapsed_since_question_started(ctx), "90")
            tool_label = color("[tool]", "36")
            response_bytes = len(details.model_dump_json().encode("utf-8"))
            print(
                f"{timestamp} {tool_label} -> "
                f"node={details.node.name or details.node.id} "
                f"recent_events={len(details.recent_events)} "
                f"tool_elapsed_ms={elapsed_ms:.1f} "
                f"response_size={humanize_bytes(response_bytes)} "
                f"({response_bytes} bytes)",
                flush=True,
            )
            print_waiting_for_model_message(ctx)
        return details
