from pathlib import Path
from typing import Any, Literal

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
from ai.proposal_configuration_coerce import coerce_canvas_answer_proposal
from ai.proposal_configuration_validate import validate_proposal_operations
from ai.skills import skill_index_markdown
from ai.tools import default_tools

__all__ = [
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
        if answer.proposal is None or not answer.proposal.operations:
            return answer

        cache_key = f"{ctx.deps.canvas_id}:{ctx.deps.canvas_version_id or 'inspect'}"
        canvas = ctx.deps.canvas_cache.get(cache_key)
        schema_cache: dict[str, list[dict[str, Any]] | None] = {}
        coerced = coerce_canvas_answer_proposal(
            ctx.deps.client,
            answer,
            canvas,
            schema_cache=schema_cache,
        )

        errors = validate_proposal_operations(
            ctx.deps.client,
            list(coerced.proposal.operations),  # type: ignore[union-attr]
            canvas,
            schema_cache=schema_cache,
        )
        if errors:
            raise ModelRetry(
                "The proposal has invalid node configuration. "
                "Fix these errors and try again:\n" + "\n".join(f"- {e}" for e in errors)
            )
        return coerced

    return agent
