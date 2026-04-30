from typing import Any

from pydantic_ai import RunContext

from ai.agent_deps import AgentDeps
from ai.skills import get_agent_skill
from ai.tools.support import tool_debug, tool_failure


class LoadAgentSkill:
    name = "load_agent_skill"
    description = "Load full markdown content for an agent skill by name."

    @staticmethod
    def label(_ctx: RunContext[AgentDeps], skill_name: str) -> str:
        return f"Reading the {skill_name} skill"

    @staticmethod
    def run(_ctx: RunContext[AgentDeps], skill_name: str) -> dict[str, Any]:
        try:
            skill = get_agent_skill(skill_name=skill_name)
            if skill is None:
                return tool_failure(
                    "load_agent_skill",
                    "skill not found",
                    code="skill_not_found",
                    skill_name=skill_name,
                )
            return skill
        except Exception as error:
            tool_debug(f"load_agent_skill failed for {skill_name}: {error}")
            return tool_failure("load_agent_skill", str(error), skill_name=skill_name)
