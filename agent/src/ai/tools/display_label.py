from __future__ import annotations

import inspect
import json
from types import SimpleNamespace
from typing import Any, cast

from pydantic_ai import RunContext

from ai.agent_deps import AgentDeps
from ai.tools.describe_component import DescribeComponent
from ai.tools.describe_trigger import DescribeTrigger
from ai.tools.get_canvas import GetCanvas
from ai.tools.get_canvas_memory import GetCanvasMemory
from ai.tools.get_canvas_shape import GetCanvasShape
from ai.tools.get_decision_pattern import GetDecisionPattern
from ai.tools.get_node_details import GetNodeDetails
from ai.tools.load_agent_skill import LoadAgentSkill
from ai.tools.list_available_integrations import ListAvailableIntegrations
from ai.tools.list_components import ListComponents
from ai.tools.list_decision_patterns import ListDecisionPatterns
from ai.tools.list_integration_resources import ListIntegrationResources
from ai.tools.list_node_events import ListNodeEvents
from ai.tools.list_node_executions import ListNodeExecutions
from ai.tools.list_org_integrations import ListOrgIntegrations
from ai.tools.list_triggers import ListTriggers
from ai.tools.search_decision_patterns import SearchDecisionPatterns

CANVAS_TOOL_CLASSES: tuple[type[Any], ...] = (
    GetCanvas,
    GetCanvasMemory,
    ListDecisionPatterns,
    SearchDecisionPatterns,
    GetDecisionPattern,
    LoadAgentSkill,
    ListComponents,
    DescribeComponent,
    ListTriggers,
    DescribeTrigger,
    ListOrgIntegrations,
    ListAvailableIntegrations,
    ListIntegrationResources,
    GetCanvasShape,
    GetNodeDetails,
    ListNodeEvents,
    ListNodeExecutions,
)

TOOLS_BY_NAME: dict[str, type[Any]] = {cls.name: cls for cls in CANVAS_TOOL_CLASSES}


def _fallback_tool_label(tool_name: str) -> str:
    normalized = tool_name.strip().lower()
    words = normalized.replace("_", " ").replace("-", " ").strip()
    if not words:
        return "Running tool"
    return words[0].upper() + words[1:]


def normalize_tool_args(raw: Any) -> dict[str, Any]:
    if isinstance(raw, dict):
        return dict(raw)
    if isinstance(raw, str):
        try:
            parsed = json.loads(raw)
        except json.JSONDecodeError:
            return {}
        return dict(parsed) if isinstance(parsed, dict) else {}
    return {}


def _dummy_run_context(deps: AgentDeps) -> RunContext[AgentDeps]:
    return cast(RunContext[AgentDeps], SimpleNamespace(deps=deps))


def format_tool_display_label(tool_name: str, args: Any, deps: AgentDeps) -> str:
    cls = TOOLS_BY_NAME.get(tool_name.strip())
    if cls is None:
        return _fallback_tool_label(tool_name)

    parsed = normalize_tool_args(args)
    try:
        sig = inspect.signature(cls.run)
        if not sig.parameters:
            return _fallback_tool_label(tool_name)
        first_name = next(iter(sig.parameters))
        filtered = {k: v for k, v in parsed.items() if k in sig.parameters and k != first_name}
        ctx = _dummy_run_context(deps)
        bound = sig.bind_partial(ctx, **filtered)
        bound.apply_defaults()
        return cls.label(*bound.args, **bound.kwargs)
    except Exception:
        return _fallback_tool_label(tool_name)


def format_tool_display_label_without_deps(tool_name: str, args: Any, canvas_id: str) -> str:
    """For persisted chat replay: labels must not rely on a live Superplane client."""
    placeholder_client = cast(Any, object())
    deps = AgentDeps(client=placeholder_client, canvas_id=canvas_id, session_store=None)
    return format_tool_display_label(tool_name, args, deps)
