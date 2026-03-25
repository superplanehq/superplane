import os
from dataclasses import dataclass, field
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
            "Be concise and factual."
            "Use list_available_integrations to verify provider availability when needed. "
            "Do not block proposing provider nodes just because org integrations are missing; "
            "it is valid to add nodes first and configure integration bindings later. "
            "When integration is missing, still provide the node proposal and mention setup as a follow-up. "
            "When a request sounds like a known workflow archetype, call list_decision_patterns or search_decision_patterns first. "
            "After selecting a pattern, always call get_decision_pattern and follow the retrieved pattern content as closely as possible. "
            "If a relevant pattern is found, follow it as closely as possible and only deviate when required by the user's explicit request or current canvas constraints. "
            "Do not claim a provider is unavailable unless a catalog tool succeeds and clearly shows no matches. "
            "If catalog tools fail, state that availability could not be verified and proceed with a best-effort proposal. "
            "When the user asks for canvas edits, include a structured proposal with "
            "operations that can be applied in the UI. Supported operation types are: "
            "add_node, connect_nodes, disconnect_nodes, update_node_config, and delete_node. "
            "Use exact block names from catalog tools, include node references by nodeId "
            "for existing nodes, and keep operation order executable. "
            "Do not invent unknown fields or operation types. "
            "In proposals, expression fields use this model: $ is the message chain—"
            "a map of upstream node outputs keyed by each node's name on the canvas "
            "(use keyed access like $['Node name']...); it is not the run-start event object. "
            "root() is the original event that started the run; "
            "when that payload nests under data, "
            "use root().data.... previous() refers to upstream output. "
            "Never use $.data. for run-start payload fields; use root().data. or the correct path "
            "under root() instead. "
            "Use get_canvas at most once per answer unless the user asks to refresh or "
            "use a different canvas. "
            "Keep responses short by default (about 3-5 lines) unless the user asks for "
            "deep analysis. "
            "If a tool returns an error payload, continue with other tools and provide the "
            "best-effort proposal instead of aborting. "
            "Common patterns: "
            "- if the user says 'pull-request comments' it maps to 'github.onPRComment'"
        ),
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
        """List available components; optionally filter by provider or text query."""
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
        """List available triggers; optionally filter by provider or text query."""
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
