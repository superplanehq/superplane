import pytest
from pydantic import ValidationError

from ai.agent_stream_context import (
    AgentContext,
    AgentStreamRequest,
    build_agent_context_state,
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


def test_build_agent_context_state() -> None:
    empty = build_agent_context_state(None)
    assert empty == build_agent_context_state(None)
    assert empty.enabled is False
    assert empty.mode == "inspect"
    assert empty.canvas_version is None

    disabled = build_agent_context_state(AgentContext(enabled=False, mode="build", canvas_version="x"))
    assert disabled.enabled is False
    assert disabled.canvas_version == "x"

    inspect_on = build_agent_context_state(AgentContext(enabled=True, mode="inspect"))
    assert inspect_on.enabled is True
    assert inspect_on.mode == "inspect"
    assert inspect_on.canvas_version is None

    build_on = build_agent_context_state(
        AgentContext(enabled=True, mode="build", canvas_version="d-v"),
    )
    assert build_on.enabled is True
    assert build_on.mode == "build"
    assert build_on.canvas_version == "d-v"
