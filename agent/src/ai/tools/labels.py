import inspect
import json
from types import SimpleNamespace
from typing import Any, cast

from pydantic_ai import RunContext

from ai.agent_deps import AgentDeps
from ai.tools.registry import TOOLS_BY_NAME


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
    """Resolve `cls.label(...)` using the same arguments as `run` (from model `args`)."""
    key = (tool_name or "").strip()
    cls = TOOLS_BY_NAME.get(key)
    if cls is None:
        return key or "tool"

    parsed = normalize_tool_args(args)
    sig = inspect.signature(cls.run)
    params = sig.parameters
    if not params:
        return cast(str, cls.name)

    first_name = next(iter(params))
    filtered = {k: v for k, v in parsed.items() if k in params and k != first_name}
    ctx = _dummy_run_context(deps)
    try:
        bound = sig.bind_partial(ctx, **filtered)
        bound.apply_defaults()
        return cast(str, cls.label(*bound.args, **bound.kwargs))
    except Exception:
        return cast(str, cls.name)
