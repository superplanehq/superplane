import os
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any, Literal

from pydantic_ai import Agent, RunContext
from pydantic_ai.models.test import TestModel

from ai.models import CanvasAnswer, CanvasQuestionRequest, CanvasSummary
from ai.patterns import get_decision_pattern as get_markdown_pattern
from ai.patterns import list_decision_patterns as list_markdown_patterns
from ai.patterns import search_decision_patterns as search_markdown_patterns
from ai.superplane_client import SuperplaneClient


@dataclass
class AgentDeps:
    client: SuperplaneClient
    canvas_id: str
    canvas_cache: dict[str, CanvasSummary] = field(default_factory=dict)


def load_system_prompt() -> str:
    return (Path(__file__).with_name("system_prompt.txt")).read_text(encoding="utf-8").strip()


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
    )

    def _tool_debug(message: str) -> None:
        if os.getenv("REPL_WEB_DEBUG", "").strip().lower() in {"1", "true", "yes", "on"}:
            print(f"[web][agent] {message}", flush=True)

    def _tool_error_entry(tool_name: str, error: Exception) -> dict[str, Any]:
        return {
            "__tool_error__": str(error),
            "__tool_name__": tool_name,
        }

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
    def list_decision_patterns(_ctx: RunContext[AgentDeps]) -> list[dict[str, Any]]:
        """List markdown decision patterns available to the agent."""
        try:
            return list_markdown_patterns()
        except Exception as error:
            _tool_debug(f"list_decision_patterns failed: {error}")
            return [{"error": str(error)}]

    @agent.tool
    def search_decision_patterns(
        _ctx: RunContext[AgentDeps],
        query: str,
        limit: int = 3,
    ) -> list[dict[str, Any]]:
        """Search markdown decision patterns relevant to a workflow request."""
        try:
            return search_markdown_patterns(query=query, limit=limit)
        except Exception as error:
            _tool_debug(f"search_decision_patterns failed: {error}")
            return [_tool_error_entry("search_decision_patterns", error)]

    @agent.tool
    def get_decision_pattern(
        _ctx: RunContext[AgentDeps], pattern_id: str
    ) -> dict[str, Any]:
        """Fetch full markdown content for one decision pattern by id."""
        try:
            pattern = get_markdown_pattern(pattern_id=pattern_id)
            if pattern is None:
                return {"id": pattern_id, "error": "pattern_not_found"}
            return pattern
        except Exception as error:
            _tool_debug(f"get_decision_pattern failed for {pattern_id}: {error}")
            return {"id": pattern_id, "error": str(error)}

    @agent.tool
    def list_components(
        ctx: RunContext[AgentDeps],
        provider: str | None = None,
        query: str | None = None,
    ) -> list[dict[str, Any]]:
        """List components (compact catalog rows: name, label, description, provider, output_channel_names).

        For configuration fields and types needed in proposals, call describe_component on the chosen name.
        Prefer a single list call per request with provider/query; reuse prior results when possible.
        """
        try:
            return ctx.deps.client.list_components(provider=provider, query=query)
        except Exception as error:
            _tool_debug(f"list_components failed: {error}")
            return [_tool_error_entry("list_components", error)]

    @agent.tool
    def describe_component(ctx: RunContext[AgentDeps], name: str) -> dict[str, Any]:
        """Describe one component including configuration fields and output channels."""
        try:
            return ctx.deps.client.describe_component(name)
        except Exception as error:
            _tool_debug(f"describe_component failed for {name}: {error}")
            return {"name": name, "error": str(error)}

    @agent.tool
    def list_triggers(
        ctx: RunContext[AgentDeps],
        provider: str | None = None,
        query: str | None = None,
    ) -> list[dict[str, Any]]:
        """List triggers (compact catalog rows: name, label, description, provider).

        For configuration fields and types needed in proposals, call describe_trigger on the chosen name.
        Prefer a single list call per request with provider/query; reuse prior results when possible.
        """
        try:
            return ctx.deps.client.list_triggers(provider=provider, query=query)
        except Exception as error:
            _tool_debug(f"list_triggers failed: {error}")
            return [_tool_error_entry("list_triggers", error)]

    @agent.tool
    def describe_trigger(ctx: RunContext[AgentDeps], name: str) -> dict[str, Any]:
        """Describe one trigger including configuration fields and required flags."""
        try:
            return ctx.deps.client.describe_trigger(name)
        except Exception as error:
            _tool_debug(f"describe_trigger failed for {name}: {error}")
            return {"name": name, "error": str(error)}

    @agent.tool
    def list_org_integrations(ctx: RunContext[AgentDeps]) -> list[dict[str, Any]]:
        """List integrations connected to the current organization."""
        try:
            return ctx.deps.client.list_org_integrations()
        except Exception as error:
            _tool_debug(f"list_org_integrations failed: {error}")
            return [_tool_error_entry("list_org_integrations", error)]

    @agent.tool
    def list_available_integrations(ctx: RunContext[AgentDeps]) -> list[dict[str, Any]]:
        """List available provider integrations from catalog metadata."""
        try:
            return ctx.deps.client.list_available_integrations()
        except Exception as error:
            _tool_debug(f"list_available_integrations failed: {error}")
            return [_tool_error_entry("list_available_integrations", error)]

    @agent.tool
    def list_integration_resources(
        ctx: RunContext[AgentDeps],
        integration_id: str,
        type: str,
        parameters: dict[str, str] | None = None,
    ) -> list[dict[str, Any]]:
        """List selectable resources for an org integration resource type."""
        try:
            return ctx.deps.client.list_integration_resources(
                integration_id=integration_id,
                type=type,
                parameters=parameters,
            )
        except Exception as error:
            _tool_debug(f"list_integration_resources failed: {error}")
            return [_tool_error_entry("list_integration_resources", error)]

    return agent
