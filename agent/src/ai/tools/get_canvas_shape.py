from pydantic_ai import RunContext

from ai.deps import AgentDeps
from ai.models import CanvasShape


def get_canvas_shape(ctx: RunContext[AgentDeps], canvas_id: str) -> CanvasShape:
    cached_shape = ctx.deps.canvas_shape_cache.get(canvas_id)
    if cached_shape is not None:
        return cached_shape

    shape = ctx.deps.client.get_canvas_shape(canvas_id)
    ctx.deps.canvas_shape_cache[canvas_id] = shape
    return shape
