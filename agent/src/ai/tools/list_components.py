from typing import Any

from pydantic_ai import RunContext

from ai.agent_deps import (
    AgentDeps,
    _clone_catalog_list_rows,
    _get_cached_catalog_list,
    _put_cached_catalog_list,
)
from ai.tools.support import tool_debug, tool_error_entry


class ListComponents:
    name = "list_components"
    description = (
        "List components (compact catalog rows).\n\n"
        "Returns name, label, description, provider, output_channel_names. "
        "For configuration fields and types needed in proposals, "
        "call describe_component on the chosen name. "
        "Prefer a single list call per request with provider/query; "
        "reuse prior results when possible."
    )

    @staticmethod
    def label(_ctx: RunContext[AgentDeps]) -> str:
        return "List components"

    @staticmethod
    def run(
        ctx: RunContext[AgentDeps],
        provider: str | None = None,
        query: str | None = None,
    ) -> list[dict[str, Any]]:
        try:
            cached = _get_cached_catalog_list(ctx.deps, "components", provider, query)
            if cached is not None:
                return cached
            rows = ctx.deps.client.list_components(provider=provider, query=query)
            _put_cached_catalog_list(ctx.deps, "components", provider, query, rows)
            return _clone_catalog_list_rows(rows)
        except Exception as error:
            tool_debug(f"list_components failed: {error}")
            return [tool_error_entry("list_components", error)]
