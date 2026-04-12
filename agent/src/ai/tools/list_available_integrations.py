from typing import Any

from pydantic_ai import RunContext

from ai.agent_deps import AgentDeps
from ai.tools.support import tool_debug, tool_error_entry


class ListAvailableIntegrations:
    name = "list_available_integrations"
    description = "List available provider integrations from catalog metadata."

    @staticmethod
    def label(ctx: RunContext[AgentDeps]) -> str:
        return "Listing available integrations"

    @staticmethod
    def run(ctx: RunContext[AgentDeps]) -> list[dict[str, Any]]:
        try:
            return ctx.deps.client.list_available_integrations()
        except Exception as error:
            tool_debug(f"list_available_integrations failed: {error}")
            return [tool_error_entry("list_available_integrations", error)]
