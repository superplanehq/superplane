from dataclasses import dataclass, field
from typing import Literal

from pydantic_ai import Agent, RunContext
from pydantic_ai.models.test import TestModel

from ai.models import CanvasAnswer, CanvasQuestionRequest, CanvasSummary
from ai.superplane_client import SuperplaneClient


@dataclass
class AgentDeps:
    client: SuperplaneClient
    canvas_cache: dict[str, CanvasSummary] = field(default_factory=dict)


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
        system_prompt=(
            "You answer questions about Superplane canvases. "
            "Use tools to fetch real canvas data before answering. "
            "Be concise and factual. Return citations when possible. "
            "Use get_canvas at most once per answer unless the user asks to refresh "
            "or use a different canvas. "
            "Keep responses short by default (about 6-10 lines) unless the user asks "
            "for deep detail."
        ),
    )

    @agent.tool
    def get_canvas(ctx: RunContext[AgentDeps], canvas_id: str) -> CanvasSummary:
        cached_summary = ctx.deps.canvas_cache.get(canvas_id)
        if cached_summary is not None:
            return cached_summary

        summary = ctx.deps.client.describe_canvas(canvas_id)
        ctx.deps.canvas_cache[canvas_id] = summary
        return summary

    return agent
