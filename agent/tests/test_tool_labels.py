from ai.tool_labels import format_tool_display_label


def test_format_tool_display_label_known_tool() -> None:
    assert format_tool_display_label("get_canvas") == "Reading canvas"


def test_format_tool_display_label_unknown_tool() -> None:
    assert format_tool_display_label("some_other_tool") == "Some other tool"


def test_format_tool_display_label_node_id_suffix() -> None:
    assert (
        format_tool_display_label(
            "get_node_details",
            {"node_id": "node-123"},
        )
        == "Reading node details (node-123)"
    )


def test_format_tool_display_label_long_node_id_truncated() -> None:
    long_id = "x" * 20
    out = format_tool_display_label("list_node_events", {"node_id": long_id})
    assert out.startswith("Listing node events (")
    assert "…" in out
