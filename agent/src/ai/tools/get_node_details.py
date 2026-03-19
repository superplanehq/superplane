from pydantic_ai import RunContext

from ai.deps import AgentDeps
from ai.models import NodeDetails


def get_node_details(
    ctx: RunContext[AgentDeps],
    node_id: str,
    canvas_id: str,
    include_recent_events: bool = True,
) -> NodeDetails:
    details = ctx.deps.client.get_node_details(
        canvas_id=canvas_id,
        node_id=node_id,
        include_recent_events=include_recent_events,
    )
    return details
