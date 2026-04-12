from typing import Any

from pydantic_ai import RunContext

from ai.agent_deps import AgentDeps
from ai.tools.support import tool_debug, tool_failure


class DescribeTrigger:
    name = "describe_trigger"
    description = "Describe one trigger including configuration fields and required flags."

    @staticmethod
    def label(_ctx: RunContext[AgentDeps]) -> str:
        return "Describe trigger"

    @staticmethod
    def run(ctx: RunContext[AgentDeps], name: str) -> dict[str, Any]:
        try:
            return ctx.deps.client.describe_trigger(name)
        except Exception as error:
            tool_debug(f"describe_trigger failed for {name}: {error}")
            return tool_failure("describe_trigger", str(error), name=name)
