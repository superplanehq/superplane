from pydantic_ai import Agent, RunContext

from ai.deps import AgentDeps, color, elapsed_since_question_started
from ai.models import CanvasAnswer


def register(agent: Agent[AgentDeps, CanvasAnswer]) -> None:
    @agent.tool
    def request_canvas_details(ctx: RunContext[AgentDeps], reason: str = "") -> str:
        if ctx.deps.show_tool_calls:
            timestamp = color(elapsed_since_question_started(ctx), "90")
            status_label = color("[status]", "33")
            printable_reason = reason.strip() or "no reason provided"
            print(
                f"{timestamp} {status_label} request_canvas_details(reason={printable_reason})",
                flush=True,
            )
        ctx.deps.allow_canvas_details = True
        return "Full canvas details enabled for this question."
