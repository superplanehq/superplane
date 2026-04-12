from typing import Any

from pydantic_ai import RunContext

from ai.agent_deps import AgentDeps
from ai.patterns import get_decision_pattern as get_markdown_pattern
from ai.tools.support import tool_debug, tool_failure


class GetDecisionPattern:
    name = "get_decision_pattern"
    description = "Fetch full markdown content for one decision pattern by id."

    @staticmethod
    def label(_ctx: RunContext[AgentDeps], pattern_id: str) -> str:
        return f"Get decision pattern ({pattern_id})"

    @staticmethod
    def run(_ctx: RunContext[AgentDeps], pattern_id: str) -> dict[str, Any]:
        try:
            pattern = get_markdown_pattern(pattern_id=pattern_id)
            if pattern is None:
                return tool_failure(
                    "get_decision_pattern",
                    "pattern not found",
                    code="pattern_not_found",
                    pattern_id=pattern_id,
                )
            return pattern
        except Exception as error:
            tool_debug(f"get_decision_pattern failed for {pattern_id}: {error}")
            return tool_failure("get_decision_pattern", str(error), pattern_id=pattern_id)
