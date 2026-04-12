from typing import Any

from pydantic_ai import RunContext

from ai.agent_deps import AgentDeps
from ai.models import NodeDetails
from ai.tools.support import tool_debug, tool_failure


class GetNodeDetails:
    name = "get_node_details"
    description = (
        "Fetch one node's catalog identity, configuration, validation messages, "
        "integration binding summary, and recent events.\n\n"
        "Use for questions about a specific node's settings, errors, or activity. "
        "Request one node at a time unless the user explicitly asks for several; "
        "configuration payloads can be large."
    )

    @staticmethod
    def label(_ctx: RunContext[AgentDeps], node_id: str, _include_recent_events: bool = True) -> str:
        return f"Looking up details for {node_id}"

    @staticmethod
    def run(
        ctx: RunContext[AgentDeps],
        node_id: str,
        include_recent_events: bool = True,
    ) -> NodeDetails | dict[str, Any]:
        try:
            return ctx.deps.client.get_node_details(
                ctx.deps.canvas_id,
                node_id,
                include_recent_events=include_recent_events,
            )
        except Exception as error:
            tool_debug(f"get_node_details failed for {node_id}: {error}")
            return tool_failure("get_node_details", str(error), node_id=node_id)
