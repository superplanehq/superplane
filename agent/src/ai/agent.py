from dataclasses import dataclass, field
from pathlib import Path
from typing import Any, Literal, cast

from pydantic_ai import Agent, RunContext
from pydantic_ai.models.test import TestModel

from ai.config import config
from ai.models import (
    CanvasAnswer,
    CanvasProposal,
    CanvasQuestionRequest,
    CanvasShape,
    CanvasSummary,
    NodeDetails,
    NodeEvent,
    NodeExecution,
)
from ai.patterns import get_decision_pattern as get_markdown_pattern
from ai.patterns import list_decision_patterns as list_markdown_patterns
from ai.patterns import search_decision_patterns as search_markdown_patterns
from ai.superplane_client import SuperplaneClient

CatalogListKind = Literal["components", "triggers"]


def _catalog_list_cache_key(
    kind: CatalogListKind,
    provider: str | None,
    query: str | None,
) -> tuple[str, str, str]:
    p = provider.strip().lower() if isinstance(provider, str) else ""
    q = query.strip().lower() if isinstance(query, str) else ""
    return (kind, p, q)


def _clone_catalog_list_rows(rows: list[dict[str, Any]]) -> list[dict[str, Any]]:
    """Detach cached rows so callers cannot mutate the in-session cache."""
    out: list[dict[str, Any]] = []
    for row in rows:
        cloned = dict(row)
        ocn = cloned.get("output_channel_names")
        if isinstance(ocn, list):
            cloned["output_channel_names"] = list(ocn)
        out.append(cloned)
    return out


@dataclass
class AgentDeps:
    client: SuperplaneClient
    canvas_id: str
    canvas_cache: dict[str, CanvasSummary] = field(default_factory=dict)
    catalog_list_cache: dict[tuple[str, str, str], list[dict[str, Any]]] = field(
        default_factory=dict
    )


def _get_cached_catalog_list(
    deps: AgentDeps,
    kind: CatalogListKind,
    provider: str | None,
    query: str | None,
) -> list[dict[str, Any]] | None:
    key = _catalog_list_cache_key(kind, provider, query)
    hit = deps.catalog_list_cache.get(key)
    if hit is None:
        return None
    return _clone_catalog_list_rows(hit)


def _put_cached_catalog_list(
    deps: AgentDeps,
    kind: CatalogListKind,
    provider: str | None,
    query: str | None,
    rows: list[dict[str, Any]],
) -> None:
    key = _catalog_list_cache_key(kind, provider, query)
    deps.catalog_list_cache[key] = _clone_catalog_list_rows(rows)


def load_system_prompt() -> str:
    return (Path(__file__).with_name("system_prompt.txt")).read_text(encoding="utf-8").strip()


def build_prompt(payload: CanvasQuestionRequest) -> str:
    return payload.question


def _tool_failure(
    tool_name: str,
    message: str,
    *,
    code: str | None = None,
    **context: Any,
) -> dict[str, Any]:
    """Uniform tool error / failure dict for the model (and logging).

    Always includes ``__tool_error__`` (human-readable message) and
    ``__tool_name__``. Optional ``__tool_error_code__`` for stable categories.
    Extra keyword arguments are copied when not None (e.g. pattern_id, name).
    """
    payload: dict[str, Any] = {
        "__tool_error__": message,
        "__tool_name__": tool_name,
    }
    if code is not None:
        payload["__tool_error_code__"] = code
    for key, value in context.items():
        if value is not None:
            payload[key] = value
    return payload


def _tool_error_entry(tool_name: str, error: Exception) -> dict[str, Any]:
    return _tool_failure(tool_name, str(error))


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
        if config.debug:
            print(f"[web][agent] {message}", flush=True)

    @agent.tool
    def validate_proposal(
        _ctx: RunContext[AgentDeps],
        proposal: CanvasProposal,
    ) -> dict[str, Any]:
        """Validate and normalize a draft canvas proposal against live catalog schemas.

        Call this with the same structured proposal you plan to include in your final
        answer when the user asked for canvas edits. The server applies the same
        coercion and type checks as the workflow UI. On success, copy the returned
        ``proposal`` into your final ``CanvasAnswer``. On failure, read ``errors``,
        fix configurations (use describe_component / describe_trigger), and call again.
        """
        return {
            "ok": True,
            "proposal": proposal.model_dump(mode="json", by_alias=True),
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
            return [_tool_error_entry("list_decision_patterns", error)]

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
    def get_decision_pattern(_ctx: RunContext[AgentDeps], pattern_id: str) -> dict[str, Any]:
        """Fetch full markdown content for one decision pattern by id."""
        try:
            pattern = get_markdown_pattern(pattern_id=pattern_id)
            if pattern is None:
                return _tool_failure(
                    "get_decision_pattern",
                    "pattern not found",
                    code="pattern_not_found",
                    pattern_id=pattern_id,
                )
            return pattern
        except Exception as error:
            _tool_debug(f"get_decision_pattern failed for {pattern_id}: {error}")
            return _tool_failure("get_decision_pattern", str(error), pattern_id=pattern_id)

    @agent.tool
    def list_components(
        ctx: RunContext[AgentDeps],
        provider: str | None = None,
        query: str | None = None,
    ) -> list[dict[str, Any]]:
        """List components (compact catalog rows).

        Returns name, label, description, provider, output_channel_names.
        For configuration fields and types needed in proposals,
        call describe_component on the chosen name.
        Prefer a single list call per request with provider/query;
        reuse prior results when possible.
        """
        try:
            cached = _get_cached_catalog_list(ctx.deps, "components", provider, query)
            if cached is not None:
                return cached
            rows = ctx.deps.client.list_components(provider=provider, query=query)
            _put_cached_catalog_list(ctx.deps, "components", provider, query, rows)
            return _clone_catalog_list_rows(rows)
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
            return _tool_failure("describe_component", str(error), name=name)

    @agent.tool
    def list_triggers(
        ctx: RunContext[AgentDeps],
        provider: str | None = None,
        query: str | None = None,
    ) -> list[dict[str, Any]]:
        """List triggers (compact catalog rows).

        Returns name, label, description, provider.
        For configuration fields and types needed in proposals,
        call describe_trigger on the chosen name.
        Prefer a single list call per request with provider/query;
        reuse prior results when possible.
        """
        try:
            cached = _get_cached_catalog_list(ctx.deps, "triggers", provider, query)
            if cached is not None:
                return cached
            rows = ctx.deps.client.list_triggers(provider=provider, query=query)
            _put_cached_catalog_list(ctx.deps, "triggers", provider, query, rows)
            return _clone_catalog_list_rows(rows)
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
            return _tool_failure("describe_trigger", str(error), name=name)

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
        """List selectable resources for an org integration resource type.

        Returns [] without calling the API when integration_id or type
        is missing or blank. For results, both must be set: use
        describe_component / describe_trigger to read
        integration-resource field metadata for the correct type string.
        """
        if not isinstance(integration_id, str) or not integration_id.strip():
            _tool_debug("list_integration_resources skipped: empty integration_id")
            return []
        if not isinstance(type, str) or not type.strip():
            _tool_debug(
                "list_integration_resources skipped: empty type (resource type is required by API)"
            )
            return []
        try:
            return ctx.deps.client.list_integration_resources(
                integration_id=integration_id,
                type=type,
                parameters=parameters,
            )
        except Exception as error:
            _tool_debug(f"list_integration_resources failed: {error}")
            return [_tool_error_entry("list_integration_resources", error)]

    @agent.tool
    def get_canvas_shape(ctx: RunContext[AgentDeps]) -> CanvasShape | dict[str, Any]:
        """Compact canvas topology using node display names (no edge channels).

        Use for explaining graph shape in human-readable form. For edits and
        exact ids or subscription channels, call get_canvas instead. Prefer at
        most one of get_canvas or get_canvas_shape per answer unless the user
        asks to refresh.
        """
        try:
            return ctx.deps.client.get_canvas_shape(ctx.deps.canvas_id)
        except Exception as error:
            _tool_debug(f"get_canvas_shape failed: {error}")
            return _tool_failure("get_canvas_shape", str(error))

    @agent.tool
    def get_node_details(
        ctx: RunContext[AgentDeps],
        node_id: str,
        include_recent_events: bool = True,
    ) -> NodeDetails | dict[str, Any]:
        """Fetch one node's catalog identity, configuration, validation messages,
        integration binding summary, and recent events.

        Use for questions about a specific node's settings, errors, or activity.
        Request one node at a time unless the user explicitly asks for several;
        configuration payloads can be large.
        """
        try:
            return ctx.deps.client.get_node_details(
                ctx.deps.canvas_id,
                node_id,
                include_recent_events=include_recent_events,
            )
        except Exception as error:
            _tool_debug(f"get_node_details failed for {node_id}: {error}")
            return _tool_failure("get_node_details", str(error), node_id=node_id)

    @agent.tool
    def list_node_events(
        ctx: RunContext[AgentDeps],
        node_id: str,
        limit: int = 10,
    ) -> list[NodeEvent | dict[str, Any]]:
        """List recent events for a canvas node (payload history / activity).

        Use when the user needs more event rows than get_node_details includes,
        or when only events matter. Keep limit modest (default 10).
        """
        try:
            events = ctx.deps.client.list_node_events(
                ctx.deps.canvas_id,
                node_id,
                limit=limit,
            )
            return cast(list[NodeEvent | dict[str, Any]], events)
        except Exception as error:
            _tool_debug(f"list_node_events failed for {node_id}: {error}")
            return [_tool_failure("list_node_events", str(error), node_id=node_id)]

    @agent.tool
    def list_node_executions(
        ctx: RunContext[AgentDeps],
        node_id: str,
        limit: int = 10,
        results: list[str] | None = None,
    ) -> list[NodeExecution | dict[str, Any]]:
        """List recent executions for a node (state, result, messages, timestamps).

        Use for run failures and execution outcomes. Optional results filters
        API enum values such as RESULT_FAILED. Default limit is 10.
        """
        try:
            executions = ctx.deps.client.list_node_executions(
                ctx.deps.canvas_id,
                node_id,
                limit=limit,
                results=results,
            )
            return cast(list[NodeExecution | dict[str, Any]], executions)
        except Exception as error:
            _tool_debug(f"list_node_executions failed for {node_id}: {error}")
            return [_tool_failure("list_node_executions", str(error), node_id=node_id)]

    return agent
