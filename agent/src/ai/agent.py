from pathlib import Path
from typing import Literal

from pydantic_ai import Agent, ModelRetry, RunContext
from pydantic_ai.models.anthropic import AnthropicModelSettings
from pydantic_ai.models.test import TestModel

from ai.agent_deps import (
    AgentContextState,
    AgentDeps,
    CatalogListKind,
    _catalog_list_cache_key,
    _clone_catalog_list_rows,
    _get_cached_catalog_list,
    _put_cached_catalog_list,
)
from ai.models import CanvasAnswer, CanvasQuestionRequest
from ai.skills import skill_index_markdown
from ai.tools import default_tools

__all__ = [
    "Agent",
    "AgentContextState",
    "AgentDeps",
    "CatalogListKind",
    "build_agent",
    "build_prompt",
    "load_system_prompt",
    "_catalog_list_cache_key",
    "_clone_catalog_list_rows",
    "_get_cached_catalog_list",
    "_put_cached_catalog_list",
]


def load_system_prompt() -> str:
    base = (Path(__file__).with_name("system_prompt.txt")).read_text(encoding="utf-8").strip()
    return base + skill_index_markdown()


def build_prompt(payload: CanvasQuestionRequest) -> str:
    return payload.question


def build_agent(model: str | Literal["test"] = "test") -> Agent[AgentDeps, CanvasAnswer]:
    resolved_model: str | TestModel
    if model == "test":
        resolved_model = TestModel()
    else:
        resolved_model = model

    agent: Agent[AgentDeps, CanvasAnswer] = Agent(
        model=resolved_model,
        output_type=CanvasAnswer,
        system_prompt=load_system_prompt(),
        output_retries=5,
        model_settings=AnthropicModelSettings(
            parallel_tool_calls=True,
            anthropic_cache_instructions="1h",
            anthropic_cache_tool_definitions="1h",
            anthropic_cache_messages=True,
        ),
        tools=default_tools,
    )

    @agent.output_validator
    def validate_answer_proposal(
        ctx: RunContext[AgentDeps],
        answer: CanvasAnswer,
    ) -> CanvasAnswer:
        if answer.proposal is None:
            return answer

        canvas_version_id = ctx.deps.canvas_version_id
        if not canvas_version_id:
            raise ModelRetry(
                "Proposal validation requires a canvas version id. "
                "Return an answer without proposal when no build context is available."
            )

        try:
            ctx.deps.client.validate_canvas_version_changeset(
                canvas_id=ctx.deps.canvas_id,
                canvas_version_id=canvas_version_id,
                changeset=answer.proposal.changeset,
            )
        except (RuntimeError, ValueError) as error:
            raise ModelRetry(
                "The proposal changeset failed server-side validation. "
                f"Update the changeset and retry. Error: {error}"
            ) from error

        return answer

    return agent
