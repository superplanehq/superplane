from typing import Literal

from pydantic_ai import Agent
from pydantic_ai.models.test import TestModel

from ai.deps import AgentDeps
from ai.models import CanvasAnswer, CanvasQuestionRequest
from ai.tools import register_tools


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
            "Use get_canvas_shape first for structure/topology questions. "
            "Full canvas details are gated: call request_canvas_details with a brief "
            "reason before calling get_canvas. "
            "Use get_canvas at most once per answer unless the user asks to refresh "
            "or use a different canvas. "
            "Keep responses short by default (about 6-10 lines) unless the user asks "
            "for deep detail."
        ),
    )
    register_tools(agent)
    return agent
