from pydantic_ai import RunContext

from ai.deps import AgentDeps
from ai.models import CanvasShape


def get_canvas_shape(ctx: RunContext[AgentDeps], canvas_id: str | None = None) -> CanvasShape:
    resolved_canvas_id = (canvas_id or ctx.deps.default_canvas_id or "").strip()
    if not resolved_canvas_id:
        raise ValueError("canvas_id is required.")

    cached_shape = ctx.deps.canvas_shape_cache.get(resolved_canvas_id)
    if cached_shape is not None:
        return cached_shape

    shape = ctx.deps.client.get_canvas_shape(resolved_canvas_id)
    ctx.deps.canvas_shape_cache[resolved_canvas_id] = shape
    return shape
