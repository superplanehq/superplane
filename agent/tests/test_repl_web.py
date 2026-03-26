import argparse
import socket
from types import SimpleNamespace

import pytest

from ai.models import CanvasQuestionRequest
from ai.web import WebServer, WebServerConfig
import repl.main as repl_main
from repl.main import _parse_stream_event, _resolve_stream_url, _stream_repl_answer


def _next_free_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind(("127.0.0.1", 0))
        return int(sock.getsockname()[1])


def test_parse_stream_event_accepts_valid_sse_line() -> None:
    event = _parse_stream_event(b'data: {"type":"model_delta","content":"hello"}\n')
    assert event == {"type": "model_delta", "content": "hello"}


def test_resolve_stream_url_uses_agent_chat_stream_path() -> None:
    assert _resolve_stream_url("http://agent:8090") == "http://agent:8090/agents/chats/local/stream"
    assert (
        _resolve_stream_url("http://agent:8090/agents/chats/chat-123/stream")
        == "http://agent:8090/agents/chats/chat-123/stream"
    )


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


def test_web_server_start_raises_when_port_is_already_in_use() -> None:
    port = _next_free_port()
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    sock.bind(("127.0.0.1", port))
    sock.listen()

    server = WebServer(WebServerConfig(host="127.0.0.1", port=port))
    with pytest.raises(RuntimeError, match="Failed to start REPL web server"):
        server.start()

    sock.close()


def test_main_non_test_mode_mints_agent_chat_session(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    session_calls: list[dict[str, str]] = []
    stream_calls: list[dict[str, str | None]] = []

    def fake_parse_args(self: argparse.ArgumentParser) -> SimpleNamespace:
        return SimpleNamespace(
            healthcheck=False,
            question="hello",
            interactive=False,
            canvas_id="canvas-123",
            base_url="http://app:8000",
            token="token-123",
            org_id="org-123",
            server=False,
            test_repl_web_host="127.0.0.1",
            test_repl_web_port=8090,
            repl_web_url=None,
            start_repl_web=False,
            start_test_repl_web=False,
            model="anthropic:claude-sonnet-4-6",
        )

    def fake_stream_repl_answer(
        web_url: str,
        payload: CanvasQuestionRequest,
        model: str,
        token: str | None = None,
    ) -> str:
        stream_calls.append(
            {
                "web_url": web_url,
                "canvas_id": payload.canvas_id,
                "model": model,
                "token": token,
            }
        )
        return "ok"

    def fake_create_agent_chat(
        base_url: str,
        api_token: str,
        org_id: str,
        canvas_id: str,
    ) -> tuple[str, str]:
        session_calls.append(
            {
                "base_url": base_url,
                "api_token": api_token,
                "org_id": org_id,
                "canvas_id": canvas_id,
            }
        )
        return "session-token-123", "http://agent:8090/agents/chats/chat-123/stream"

    monkeypatch.setattr(argparse.ArgumentParser, "parse_args", fake_parse_args)
    monkeypatch.setattr(repl_main, "_create_agent_chat", fake_create_agent_chat)
    monkeypatch.setattr(repl_main, "_stream_repl_answer", fake_stream_repl_answer)

    repl_main.main()

    assert session_calls == [
        {
            "base_url": "http://app:8000",
            "api_token": "token-123",
            "org_id": "org-123",
            "canvas_id": "canvas-123",
        }
    ]
    assert stream_calls == [
        {
            "web_url": "http://agent:8090/agents/chats/chat-123/stream",
            "canvas_id": "canvas-123",
            "model": "anthropic:claude-sonnet-4-6",
            "token": "session-token-123",
        }
    ]
