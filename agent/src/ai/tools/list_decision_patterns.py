from typing import Any

from pydantic_ai import RunContext

from ai.agent_deps import AgentDeps
from ai.patterns import list_decision_patterns as list_markdown_patterns
from ai.tools.support import tool_debug, tool_error_entry


class ListDecisionPatterns:
    name = "list_decision_patterns"
    description = "List markdown decision patterns available to the agent."

    @staticmethod
    def label(_ctx: RunContext[AgentDeps]) -> str:
        return "Looking up existing patterns"

    @staticmethod
    def run(_ctx: RunContext[AgentDeps]) -> list[dict[str, Any]]:
        try:
            return list_markdown_patterns()
        except Exception as error:
            tool_debug(f"list_decision_patterns failed: {error}")
            return [tool_error_entry("list_decision_patterns", error)]
