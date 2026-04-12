from pydantic_ai import RunContext

from ai.agent_deps import AgentDeps
from ai.models import CanvasSummary


class GetCanvas:
    name = "get_canvas"
    description = "Fetch the current request canvas summary (nodes/edges)."

    @staticmethod
    def label(ctx: RunContext[AgentDeps]) -> str:
        return "Reading canvas"

    @staticmethod
    def run(ctx: RunContext[AgentDeps]) -> CanvasSummary:
        resolved_canvas_id = ctx.deps.canvas_id
        cached_summary = ctx.deps.canvas_cache.get(resolved_canvas_id)
        if cached_summary is not None:
            return cached_summary

        summary = ctx.deps.client.describe_canvas(resolved_canvas_id)
        ctx.deps.canvas_cache[resolved_canvas_id] = summary
        return summary
