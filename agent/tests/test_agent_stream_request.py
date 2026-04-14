import pytest
from pydantic import ValidationError

from ai.web import (
    AgentContext,
    AgentStreamRequest,
    compose_agent_user_prompt,
    derive_agent_editor_fields,
)


def test_agent_context_defaults() -> None:
    ctx = AgentContext()
    assert ctx.enabled is False
    assert ctx.mode == "inspect"
    assert ctx.canvas_version is None


def test_agent_context_inspect_rejects_canvas_version() -> None:
    with pytest.raises(ValidationError):
        AgentContext(enabled=True, mode="inspect", canvas_version="x")


def test_agent_context_build_requires_canvas_version() -> None:
    with pytest.raises(ValidationError):
        AgentContext(enabled=True, mode="build")


def test_agent_context_build_with_version_ok() -> None:
    ctx = AgentContext(enabled=True, mode="build", canvas_version="ver-1")
    assert ctx.canvas_version == "ver-1"


def test_agent_stream_request_nested_agent_context() -> None:
    body = AgentStreamRequest(
        question="Hello",
        model="test",
        agent_context=AgentContext(enabled=True, mode="build", canvas_version="v-d"),
    )
    assert body.agent_context is not None
    assert body.agent_context.mode == "build"


def test_derive_agent_editor_fields_disabled() -> None:
    assert derive_agent_editor_fields(None) == (None, None)
    assert derive_agent_editor_fields(AgentContext(enabled=False)) == (None, None)


def test_derive_agent_editor_fields_inspect_and_build() -> None:
    assert derive_agent_editor_fields(AgentContext(enabled=True, mode="inspect")) == (None, "inspect")
    assert derive_agent_editor_fields(
        AgentContext(enabled=True, mode="build", canvas_version="d-v"),
    ) == ("d-v", "build")


def test_compose_agent_user_prompt_includes_build_context() -> None:
    out = compose_agent_user_prompt("Do X", "build")
    assert out.startswith("[Editor context:")
    assert out.endswith("Do X")


def test_compose_agent_user_prompt_omits_prefix_when_surface_unset() -> None:
    assert compose_agent_user_prompt("Hi", None) == "Hi"
