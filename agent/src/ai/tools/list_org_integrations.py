from typing import Any

from pydantic_ai import RunContext

from ai.agent_deps import AgentDeps
from ai.tools.support import tool_debug, tool_error_entry


class ListOrgIntegrations:
    name = "list_org_integrations"
    description = "List integrations connected to the current organization."

    @staticmethod
    def label(ctx: RunContext[AgentDeps]) -> str:
        return "List org integrations"

    @staticmethod
    def run(ctx: RunContext[AgentDeps]) -> list[dict[str, Any]]:
        try:
            return ctx.deps.client.list_org_integrations()
        except Exception as error:
            tool_debug(f"list_org_integrations failed: {error}")
            return [tool_error_entry("list_org_integrations", error)]
