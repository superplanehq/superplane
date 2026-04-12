from ai.tools.display_label import format_tool_display_label_without_deps


def test_describe_component_label_includes_name() -> None:
    label = format_tool_display_label_without_deps(
        "describe_component",
        {"name": "slack_send_message"},
        "canvas-1",
    )
    assert label == 'Describe component "slack_send_message"'


def test_unknown_tool_falls_back_to_title_case() -> None:
    assert (
        format_tool_display_label_without_deps("some_future_tool", {}, "canvas-1")
        == "Some future tool"
    )
