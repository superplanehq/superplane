from dataclasses import dataclass, field
from typing import Any
from typing import Literal

from pydantic_ai import Agent, RunContext
from pydantic_ai.models.test import TestModel

from ai.models import CanvasAnswer, CanvasQuestionRequest, CanvasSummary
from ai.superplane_client import SuperplaneClient


@dataclass
class AgentDeps:
    client: SuperplaneClient
    canvas_id: str
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
            "Use tools to fetch real canvas and catalog data before answering. "
            "Be concise and factual. Return citations when possible. "
            "When the user asks for canvas edits, include a structured proposal with "
            "operations that can be applied in the UI. Supported operation types are: "
            "add_node, connect_nodes, disconnect_nodes, update_node_config, and delete_node. "
            "Use exact block names from catalog tools, include node references by nodeId "
            "for existing nodes, and keep operation order executable. "
            "Do not invent unknown fields or operation types. "
            "Use get_canvas at most once per answer unless the user asks to refresh "
            "or use a different canvas. "
            "Keep responses short by default (about 6-10 lines) unless the user asks "
            "for deep detail."
        ),
    )

    @agent.tool
    def get_canvas(ctx: RunContext[AgentDeps]) -> CanvasSummary:
        """Fetch the current request canvas summary (nodes/edges)."""
        resolved_canvas_id = ctx.deps.canvas_id
        cached_summary = ctx.deps.canvas_cache.get(resolved_canvas_id)
        if cached_summary is not None:
            return cached_summary

        summary = ctx.deps.client.describe_canvas(resolved_canvas_id)
        ctx.deps.canvas_cache[resolved_canvas_id] = summary
        return summary

    @agent.tool
    def list_components(
        ctx: RunContext[AgentDeps],
        provider: str | None = None,
        query: str | None = None,
    ) -> list[dict[str, Any]]:
        """List available components; optionally filter by provider or text query."""
        return ctx.deps.client.list_components(provider=provider, query=query)

    @agent.tool
    def describe_component(ctx: RunContext[AgentDeps], name: str) -> dict[str, Any]:
        """Describe one component including configuration fields and output channels."""
        return ctx.deps.client.describe_component(name)

    @agent.tool
    def list_triggers(
        ctx: RunContext[AgentDeps],
        provider: str | None = None,
        query: str | None = None,
    ) -> list[dict[str, Any]]:
        """List available triggers; optionally filter by provider or text query."""
        return ctx.deps.client.list_triggers(provider=provider, query=query)

    @agent.tool
    def describe_trigger(ctx: RunContext[AgentDeps], name: str) -> dict[str, Any]:
        """Describe one trigger including configuration fields and required flags."""
        return ctx.deps.client.describe_trigger(name)

    @agent.tool
    def list_org_integrations(ctx: RunContext[AgentDeps]) -> list[dict[str, Any]]:
        """List integrations connected to the current organization."""
        return ctx.deps.client.list_org_integrations()

    @agent.tool
    def list_integration_resources(
        ctx: RunContext[AgentDeps],
        integration_id: str,
        type: str,
        parameters: dict[str, str] | None = None,
    ) -> list[dict[str, Any]]:
        """List selectable resources for an org integration resource type."""
        return ctx.deps.client.list_integration_resources(
            integration_id=integration_id,
            type=type,
            parameters=parameters,
        )

    return agent
