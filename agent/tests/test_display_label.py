from typing import Any, cast

from ai.agent_deps import AgentDeps
from ai.tools import format_tool_display_label


def _replay_deps(canvas_id: str) -> AgentDeps:
    return AgentDeps(client=cast(Any, object()), canvas_id=canvas_id, session_store=None)


def test_describe_component_label_includes_name() -> None:
    label = format_tool_display_label(
        "describe_component",
        {"name": "slack_send_message"},
        _replay_deps("canvas-1"),
    )
    assert label == 'Describe component "slack_send_message"'


def test_unknown_tool_falls_back_to_title_case() -> None:
    label = format_tool_display_label("some_future_tool", {}, _replay_deps("canvas-1"))
    assert label == "Some future tool"
