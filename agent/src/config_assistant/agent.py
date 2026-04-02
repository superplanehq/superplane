import os
from pathlib import Path
from typing import Literal

from pydantic_ai import Agent
from pydantic_ai.models.test import TestModel

from config_assistant.models import FieldSuggestOutput


def load_system_prompt() -> str:
    return (Path(__file__).with_name("system_prompt.txt")).read_text(encoding="utf-8").strip()


def default_config_assistant_model() -> str:
    raw = (os.getenv("CONFIG_ASSISTANT_AI_MODEL") or os.getenv("AI_MODEL") or "test").strip()
    return raw if raw else "test"


def build_config_assistant_agent(
    model: str | Literal["test"] = "test",
) -> Agent[None, FieldSuggestOutput]:
    resolved_model: str | TestModel
    if model == "test":
        resolved_model = TestModel()
    else:
        resolved_model = model

    return Agent(
        model=resolved_model,
        output_type=FieldSuggestOutput,
        system_prompt=load_system_prompt(),
    )


def build_user_prompt(*, instruction: str, field_context_json: str, node_id: str) -> str:
    ctx = field_context_json.strip() if field_context_json else "{}"
    return (
        f"Node id: {node_id}\n\n"
        f"Field context (JSON):\n{ctx}\n\n"
        f"User instruction:\n{instruction.strip()}"
    )
