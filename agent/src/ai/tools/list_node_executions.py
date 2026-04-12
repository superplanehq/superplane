from typing import Any, cast

from pydantic_ai import RunContext

from ai.agent_deps import AgentDeps
from ai.models import NodeExecution
from ai.tools.support import tool_debug, tool_failure


class ListNodeExecutions:
    name = "list_node_executions"
    description = (
        "List recent executions for a node (state, result, messages, timestamps).\n\n"
        "Use for run failures and execution outcomes. Optional results filters "
        "API enum values such as RESULT_FAILED. Default limit is 10."
    )

    @staticmethod
    def label(_ctx: RunContext[AgentDeps]) -> str:
        return "Listing node executions"

    @staticmethod
    def run(
        ctx: RunContext[AgentDeps],
        node_id: str,
        limit: int = 10,
        results: list[str] | None = None,
    ) -> list[NodeExecution | dict[str, Any]]:
        try:
            executions = ctx.deps.client.list_node_executions(
                ctx.deps.canvas_id,
                node_id,
                limit=limit,
                results=results,
            )
            return cast(list[NodeExecution | dict[str, Any]], executions)
        except Exception as error:
            tool_debug(f"list_node_executions failed for {node_id}: {error}")
            return [tool_failure("list_node_executions", str(error), node_id=node_id)]
