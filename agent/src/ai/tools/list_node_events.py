from typing import Any, cast

from pydantic_ai import RunContext

from ai.agent_deps import AgentDeps
from ai.models import NodeEvent
from ai.tools.support import tool_debug, tool_failure


class ListNodeEvents:
    name = "list_node_events"
    description = (
        "List recent events for a canvas node (payload history / activity).\n\n"
        "Use when the user needs more event rows than get_node_details includes, "
        "or when only events matter. Keep limit modest (default 10)."
    )

    @staticmethod
    def label(
        ctx: RunContext[AgentDeps],
        node_id: str,
        limit: int = 10,
    ) -> str:
        return "Listing node events"

    @staticmethod
    def run(
        ctx: RunContext[AgentDeps],
        node_id: str,
        limit: int = 10,
    ) -> list[NodeEvent | dict[str, Any]]:
        try:
            events = ctx.deps.client.list_node_events(
                ctx.deps.canvas_id,
                node_id,
                limit=limit,
            )
            return cast(list[NodeEvent | dict[str, Any]], events)
        except Exception as error:
            tool_debug(f"list_node_events failed for {node_id}: {error}")
            return [tool_failure("list_node_events", str(error), node_id=node_id)]
