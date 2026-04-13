from unittest.mock import MagicMock

from ai.memory.store import get_canvas_memory_markdown, set_canvas_memory_markdown


def test_memory_store_delegates_to_session_store() -> None:
    store = MagicMock()
    store.get_canvas_memory_markdown.return_value = "## Notes\n\nhello"
    assert get_canvas_memory_markdown(store, "00000000-0000-0000-0000-000000000001") == (
        "## Notes\n\nhello"
    )
    store.get_canvas_memory_markdown.assert_called_once_with("00000000-0000-0000-0000-000000000001")

    set_canvas_memory_markdown(store, "00000000-0000-0000-0000-000000000002", "  x  ")
    store.set_canvas_memory_markdown.assert_called_once_with(
        "00000000-0000-0000-0000-000000000002",
        "  x  ",
    )
