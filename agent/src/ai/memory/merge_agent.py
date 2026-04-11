"""Second-pass agent: merge a completed run into the canvas markdown memory document."""

from __future__ import annotations

from pydantic_ai import Agent

from ai.config import config
from ai.session_store import SessionStore

from .store import get_canvas_memory_markdown, set_canvas_memory_markdown

_SYSTEM_PROMPT = (
    "You maintain one Markdown document of durable notes for a SuperPlane canvas "
    "(user preferences, decisions, naming, integrations, things to remember across chats).\n"
    "You will receive the previous document (possibly empty), the user's latest message, "
    "and the assistant's reply from that turn.\n"
    "Output ONLY the full updated Markdown document. No preamble or markdown code fences "
    "around the whole doc.\n"
    "Merge in new useful facts; keep prior content when still relevant; drop noise and "
    "duplication.\n"
    "Use ## headings for sections when helpful. Prefer staying under about 4000 characters; "
    "summarize if needed."
)


def build_memory_merge_agent(model: str) -> Agent[None, str]:
    return Agent(model=model, system_prompt=_SYSTEM_PROMPT, output_type=str)


async def merge_canvas_memory_markdown(
    *,
    store: SessionStore,
    canvas_id: str,
    model: str,
    user_question: str,
    assistant_reply: str,
) -> None:
    previous = get_canvas_memory_markdown(store, canvas_id)
    agent = build_memory_merge_agent(model)
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
            print(f"[web] canvas memory merge model run failed: {error}", flush=True)
        return
    output = result.output
    if isinstance(output, str) and output.strip():
        set_canvas_memory_markdown(store, canvas_id, output)
