from pydantic_ai import RunContext

from ai.deps import AgentDeps
from ai.models import NodeDetails


def get_node_details(
    ctx: RunContext[AgentDeps],
    node_id: str,
    canvas_id: str | None = None,
    include_recent_events: bool = True,
) -> NodeDetails:
    resolved_canvas_id = (canvas_id or ctx.deps.default_canvas_id or "").strip()
    if not resolved_canvas_id:
        raise ValueError("canvas_id is required.")

    details = ctx.deps.client.get_node_details(
        canvas_id=resolved_canvas_id,
        node_id=node_id,
        include_recent_events=include_recent_events,
    )
    return details
