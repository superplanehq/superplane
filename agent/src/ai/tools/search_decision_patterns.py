from typing import Any

from pydantic_ai import RunContext

from ai.agent_deps import AgentDeps
from ai.patterns import search_decision_patterns as search_markdown_patterns
from ai.tools.support import tool_debug, tool_error_entry


class SearchDecisionPatterns:
    name = "search_decision_patterns"
    description = "Search markdown decision patterns relevant to a workflow request."

    @staticmethod
    def label(_ctx: RunContext[AgentDeps], query: str, limit: int = 3) -> str:
        preview = query.strip()
        if len(preview) > 48:
            preview = f"{preview[:45]}…"
        return f'Searching for patterns about {preview}'

    @staticmethod
    def run(
        _ctx: RunContext[AgentDeps],
        query: str,
        limit: int = 3,
    ) -> list[dict[str, Any]]:
        try:
            return search_markdown_patterns(query=query, limit=limit)
        except Exception as error:
            tool_debug(f"search_decision_patterns failed: {error}")
            return [tool_error_entry("search_decision_patterns", error)]
