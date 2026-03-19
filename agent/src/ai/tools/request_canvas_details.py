from pydantic_ai import RunContext

from ai.deps import AgentDeps


def request_canvas_details(ctx: RunContext[AgentDeps], reason: str = "") -> str:
    ctx.deps.allow_canvas_details = True
    return "Full canvas details enabled for this question."
