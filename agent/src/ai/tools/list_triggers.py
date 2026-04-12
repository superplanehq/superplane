from typing import Any

from pydantic_ai import RunContext

from ai.agent_deps import (
    AgentDeps,
    _clone_catalog_list_rows,
    _get_cached_catalog_list,
    _put_cached_catalog_list,
)
from ai.tools.support import tool_debug, tool_error_entry


class ListTriggers:
    name = "list_triggers"
    description = (
        "List triggers (compact catalog rows).\n\n"
        "Returns name, label, description, provider. "
        "For configuration fields and types needed in proposals, "
        "call describe_trigger on the chosen name. "
        "Prefer a single list call per request with provider/query; "
        "reuse prior results when possible."
    )

    @staticmethod
    def label(
        ctx: RunContext[AgentDeps],
        provider: str | None = None,
        query: str | None = None,
    ) -> str:
        if isinstance(query, str) and query.strip():
            preview = query.strip()
            if len(preview) > 40:
                preview = f"{preview[:37]}…"
            return f'List triggers matching "{preview}"'
        if isinstance(provider, str) and provider.strip():
            return f"List triggers ({provider.strip()})"
        return "List triggers"

    @staticmethod
    def run(
        ctx: RunContext[AgentDeps],
        provider: str | None = None,
        query: str | None = None,
    ) -> list[dict[str, Any]]:
        try:
            cached = _get_cached_catalog_list(ctx.deps, "triggers", provider, query)
            if cached is not None:
                return cached
            rows = ctx.deps.client.list_triggers(provider=provider, query=query)
            _put_cached_catalog_list(ctx.deps, "triggers", provider, query, rows)
            return _clone_catalog_list_rows(rows)
        except Exception as error:
            tool_debug(f"list_triggers failed: {error}")
            return [tool_error_entry("list_triggers", error)]
