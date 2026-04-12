from typing import Any

from pydantic_ai import RunContext

from ai.agent_deps import AgentDeps
from ai.models import CanvasShape
from ai.tools.support import tool_debug, tool_failure


class GetCanvasShape:
    name = "get_canvas_shape"
    description = (
        "Compact canvas topology using node display names (no edge channels).\n\n"
        "Use for explaining graph shape in human-readable form. For edits and "
        "exact ids or subscription channels, call get_canvas instead. Prefer at "
        "most one of get_canvas or get_canvas_shape per answer unless the user "
        "asks to refresh."
    )

    @staticmethod
    def label(_ctx: RunContext[AgentDeps]) -> str:
        return "Reading canvas structure"

    @staticmethod
    def run(ctx: RunContext[AgentDeps]) -> CanvasShape | dict[str, Any]:
        try:
            return ctx.deps.client.get_canvas_shape(ctx.deps.canvas_id)
        except Exception as error:
            tool_debug(f"get_canvas_shape failed: {error}")
            return tool_failure("get_canvas_shape", str(error))
