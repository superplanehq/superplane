from typing import Any

from pydantic_ai import RunContext

from ai.agent_deps import AgentDeps
from ai.tools.support import tool_debug, tool_failure


class DescribeComponent:
    name = "describe_component"
    description = "Describe one component including configuration fields and output channels."

    @staticmethod
    def label(_ctx: RunContext[AgentDeps]) -> str:
        return "Describe component"

    @staticmethod
    def run(ctx: RunContext[AgentDeps], name: str) -> dict[str, Any]:
        try:
            return ctx.deps.client.describe_component(name)
        except Exception as error:
            tool_debug(f"describe_component failed for {name}: {error}")
            return tool_failure("describe_component", str(error), name=name)
