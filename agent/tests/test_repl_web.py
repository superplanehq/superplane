import socket

import pytest

from ai.models import CanvasQuestionRequest
from ai.web import WebServer, WebServerConfig
from repl.main import _parse_stream_event, _stream_repl_answer


def _next_free_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind(("127.0.0.1", 0))
        return int(sock.getsockname()[1])


def test_parse_stream_event_accepts_valid_sse_line() -> None:
    event = _parse_stream_event(b'data: {"type":"model_delta","content":"hello"}\n')
    assert event == {"type": "model_delta", "content": "hello"}


def test_stream_repl_answer_reads_sse_response_end_to_end(
    capsys: pytest.CaptureFixture[str],
) -> None:
    port = _next_free_port()
    server = WebServer(WebServerConfig(host="127.0.0.1", port=port))
    server.start()

    payload = CanvasQuestionRequest(question="hello from repl")
    answer = _stream_repl_answer(
        web_url=f"http://127.0.0.1:{port}",
        payload=payload,
        model="test",
    )

    server.stop()

    assert answer == "success (no tool calls)"
    captured = capsys.readouterr()
    assert "success (no tool calls)" in captured.out
