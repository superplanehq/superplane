from pydantic_ai import RunContext

from ai.agent_deps import AgentDeps
from ai.models import CanvasSummary


class GetCanvas:
    name = "get_canvas"
    description = "Fetch the current request canvas summary (nodes/edges)."

    @staticmethod
    def label(ctx: RunContext[AgentDeps]) -> str:
        return "Loading canvas details"

    @staticmethod
    def run(ctx: RunContext[AgentDeps]) -> CanvasSummary:
        resolved_canvas_id = ctx.deps.canvas_id
        version_id = ctx.deps.canvas_version_id
        cache_key = f"{resolved_canvas_id}:{version_id or 'live'}"
        cached_summary = ctx.deps.canvas_cache.get(cache_key)
        if cached_summary is not None:
            return cached_summary

        summary = ctx.deps.client.describe_editing_canvas(resolved_canvas_id, version_id)
        ctx.deps.canvas_cache[cache_key] = summary
        return summary
