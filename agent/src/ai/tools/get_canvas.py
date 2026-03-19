from pydantic_ai import RunContext

from ai.deps import AgentDeps
from ai.models import CanvasSummary


def get_canvas(ctx: RunContext[AgentDeps], canvas_id: str | None = None) -> CanvasSummary:
    resolved_canvas_id = (canvas_id or ctx.deps.default_canvas_id or "").strip()
    if not resolved_canvas_id:
        raise ValueError("canvas_id is required.")
    if not ctx.deps.allow_canvas_details:
        raise ValueError(
            "Full canvas details are locked. "
            "Call request_canvas_details with a brief reason first."
        )

    cached_summary = ctx.deps.canvas_cache.get(resolved_canvas_id)
    if cached_summary is not None:
        return cached_summary

    summary = ctx.deps.client.describe_canvas(resolved_canvas_id)
    ctx.deps.canvas_cache[resolved_canvas_id] = summary
    return summary
