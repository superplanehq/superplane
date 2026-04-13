from pydantic_ai import RunContext

from ai.agent_deps import AgentDeps
from ai.memory.store import get_canvas_memory_markdown


class GetCanvasMemory:
    name = "get_canvas_memory"
    description = (
        "Load saved Markdown notes for this canvas.\n\n"
        "Call when prior decisions, preferences, or context from earlier sessions may matter. "
        "Returns empty string if none are stored. Skip when not relevant to avoid extra work."
    )

    @staticmethod
    def label(ctx: RunContext[AgentDeps]) -> str:
        return "Looking up previous decisions"

    @staticmethod
    def run(ctx: RunContext[AgentDeps]) -> str:
        store = ctx.deps.session_store
        if store is None:
            return ""
        return get_canvas_memory_markdown(store, ctx.deps.canvas_id)
