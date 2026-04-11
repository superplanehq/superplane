"""Memory curator agent: refresh the canvas markdown memory from a completed run."""

from __future__ import annotations

from pydantic_ai import Agent

from ai.config import config
from ai.session_store import SessionStore

from .store import get_canvas_memory_markdown, set_canvas_memory_markdown

_SYSTEM_PROMPT = (
    "You maintain a single Markdown executive summary for a SuperPlane canvas: what matters "
    "for future work on that canvas.\n"
    "Prioritize: infrastructure and systems tied to this canvas (integrations, services, "
    "environments, credentials or endpoints only when they are stable facts), important "
    "constraints or context, and explicit decisions or commitments from the conversation.\n"
    "Write like a short exec summary—tight, scannable, no chat transcript. Drop small talk, "
    "step-by-step narration, and anything that will not help later turns.\n"
    "You will receive the previous document (possibly empty), the user's latest message, "
    "and the assistant's reply from that turn.\n"
    "Output ONLY the full updated Markdown document. No preamble and no markdown code fences "
    "around the whole doc.\n"
    "Merge in new durable facts; keep prior content only while it stays relevant; remove "
    "noise and merge duplicates into one line.\n"
    "Use ## headings sparingly when they clarify structure. Stay under about 4000 characters; "
    "compress and summarize rather than growing long."
)


def build_memory_curator_agent(model: str) -> Agent[None, str]:
    return Agent(model=model, system_prompt=_SYSTEM_PROMPT, output_type=str)


async def curate_canvas_memory_markdown(
    *,
    store: SessionStore,
    canvas_id: str,
    model: str,
    user_question: str,
    assistant_reply: str,
) -> None:
    previous = get_canvas_memory_markdown(store, canvas_id)
    agent = build_memory_curator_agent(model)
    user_prompt = (
        "## Previous notes\n\n"
        f"{previous or '(none)'}\n\n"
        "## User message\n\n"
        f"{user_question}\n\n"
        "## Assistant reply\n\n"
        f"{assistant_reply or '(none)'}\n"
    )
    try:
        result = await agent.run(user_prompt)
    except Exception as error:
        if config.debug:
            print(f"[web] canvas memory curator model run failed: {error}", flush=True)
        return
    output = result.output
    if isinstance(output, str) and output.strip():
        set_canvas_memory_markdown(store, canvas_id, output)
