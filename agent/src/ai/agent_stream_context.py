"""Stream request body models and UI ``agent_context`` mapping for the HTTP agent."""

from typing import Literal, Self

from pydantic import BaseModel, ConfigDict, Field, model_validator

from ai.agent import AgentContextState
from ai.config import config
from ai.text import normalize_optional


class AgentContext(BaseModel):
    """Structured editor state sent with each agent stream request (from the UI)."""

    model_config = ConfigDict(extra="forbid")

    enabled: bool = False
    mode: Literal["inspect", "build"] = "inspect"
    canvas_version: str | None = Field(
        default=None,
        max_length=200,
        description="Draft canvas version id for tool reads; only used when mode is build.",
    )

    @model_validator(mode="after")
    def canvas_version_matches_mode(self) -> Self:
        version = normalize_optional(self.canvas_version)
        if self.mode == "inspect" and version is not None:
            raise ValueError("canvas_version must not be set when mode is inspect")
        if self.enabled and self.mode == "build" and version is None:
            raise ValueError("canvas_version is required when enabled is true and mode is build")
        return self


class AgentStreamRequest(BaseModel):
    question: str = Field(min_length=1, max_length=2000)
    model: str = Field(
        default=config.ai_model,
        min_length=1,
        max_length=200,
    )
    base_url: str | None = None
    agent_context: AgentContext | None = Field(
        default=None,
        description=(
            "Dedicated agent contract: enabled, inspect vs build; canvas_version only for build."
        ),
    )


def build_agent_context_state(ctx: AgentContext | None) -> AgentContextState:
    if ctx is None:
        return AgentContextState()
    return AgentContextState(
        enabled=ctx.enabled,
        mode=ctx.mode,
        canvas_version=normalize_optional(ctx.canvas_version),
    )
