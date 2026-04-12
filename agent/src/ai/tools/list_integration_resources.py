from typing import Any

from pydantic_ai import RunContext

from ai.agent_deps import AgentDeps
from ai.tools.support import tool_debug, tool_error_entry


class ListIntegrationResources:
    name = "list_integration_resources"
    description = (
        "List selectable resources for an org integration resource type.\n\n"
        "Returns [] without calling the API when integration_id or type "
        "is missing or blank. For results, both must be set: use "
        "describe_component / describe_trigger to read "
        "integration-resource field metadata for the correct type string."
    )

    @staticmethod
    def label(
        ctx: RunContext[AgentDeps],
        integration_id: str,
        type: str,
        parameters: dict[str, str] | None = None,
    ) -> str:
        resource_type = type.strip() if isinstance(type, str) else ""
        suffix = f" ({resource_type})" if resource_type else ""
        return f"List integration resources{suffix}"

    @staticmethod
    def run(
        ctx: RunContext[AgentDeps],
        integration_id: str,
        type: str,
        parameters: dict[str, str] | None = None,
    ) -> list[dict[str, Any]]:
        if not isinstance(integration_id, str) or not integration_id.strip():
            tool_debug("list_integration_resources skipped: empty integration_id")
            return []
        if not isinstance(type, str) or not type.strip():
            tool_debug(
                "list_integration_resources skipped: empty type (resource type is required by API)"
            )
            return []
        try:
            return ctx.deps.client.list_integration_resources(
                integration_id=integration_id,
                type=type,
                parameters=parameters,
            )
        except Exception as error:
            tool_debug(f"list_integration_resources failed: {error}")
            return [tool_error_entry("list_integration_resources", error)]
