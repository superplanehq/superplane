import asyncio
import argparse
import socket
from types import SimpleNamespace
from typing import Any
from unittest.mock import MagicMock

import pytest
from pydantic_ai.messages import ModelRequest, ModelResponse, TextPart, ToolReturnPart, UserPromptPart

from ai.models import CanvasQuestionRequest
import ai.repl_web as repl_web
from ai.session_store import AgentChatNotFoundError
from ai.web import WebServer, WebServerConfig
import repl.main as repl_main
from repl.main import _parse_stream_event, _resolve_stream_url, _stream_repl_answer


def _next_free_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind(("127.0.0.1", 0))
        return int(sock.getsockname()[1])


@pytest.fixture(autouse=True)
def _stub_agent_persistence(monkeypatch: pytest.MonkeyPatch) -> None:
    fake_store = MagicMock()
    fake_store.get_agent_chat.side_effect = AgentChatNotFoundError("missing")
    fake_store.create_agent_chat.side_effect = lambda org_id, user_id, canvas_id, chat_id=None: SimpleNamespace(
        id=chat_id or "chat-123",
        org_id=org_id,
        user_id=user_id,
        canvas_id=canvas_id,
    )
    fake_store.count_chat_model_messages.return_value = 0
    fake_store.load_agent_chat_message_history.return_value = []
    fake_store.set_initial_chat_message_if_missing.return_value = None
    fake_store.create_agent_chat_model_message.side_effect = lambda chat_id, message: SimpleNamespace(
        id="message-123",
    )
    fake_store.update_agent_chat_model_message.return_value = None
    fake_store.replace_agent_chat_messages_after.return_value = None
    monkeypatch.setattr(repl_web, "SessionStore", MagicMock(return_value=fake_store))
    fake_grpc_server = MagicMock()
    fake_grpc_server.start.return_value = None
    fake_grpc_server.stop.return_value = None
    monkeypatch.setattr(repl_web.InternalAgentServer, "from_env", MagicMock(return_value=fake_grpc_server))


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
        stream_url=f"http://127.0.0.1:{port}",
        payload=payload,
        model="test",
    )

    server.stop()

    assert answer == "success (no tool calls)"
    captured = capsys.readouterr()
    assert "success (no tool calls)" in captured.out


def test_stream_repl_answer_uses_passed_stream_url(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    requested_urls: list[str] = []

    class FakeResponse:
        def __init__(self) -> None:
            self._lines = iter(
                [
                    b'data: {"type":"final_answer","output":"ok"}\n',
                    b'data: {"type":"done"}\n',
                    b"",
                ]
            )

        def __enter__(self) -> "FakeResponse":
            return self

        def __exit__(self, exc_type: object, exc: object, tb: object) -> None:
            return None

        def readline(self) -> bytes:
            return next(self._lines)

    def fake_urlopen(request: Any, timeout: int) -> FakeResponse:
        requested_urls.append(request.full_url)
        assert timeout == 30
        return FakeResponse()

    monkeypatch.setattr(repl_main, "urlopen", fake_urlopen)

    answer = _stream_repl_answer(
        stream_url="http://agent:8090/agents/chats/chat-123/stream",
        payload=CanvasQuestionRequest(question="hello from repl"),
        model="anthropic:claude-sonnet-4-6",
        token="session-token-123",
    )

    assert answer == "ok"
    assert requested_urls == ["http://agent:8090/agents/chats/chat-123/stream"]


def test_web_server_start_raises_when_port_is_already_in_use() -> None:
    port = _next_free_port()
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    sock.bind(("127.0.0.1", port))
    sock.listen()

    server = WebServer(WebServerConfig(host="127.0.0.1", port=port))
    with pytest.raises(RuntimeError, match="Failed to start REPL web server") as exc_info:
        server.start()

    sock.close()
    assert isinstance(exc_info.value.__cause__, SystemExit)


def test_stream_agent_run_excludes_current_prompt_from_loaded_message_history(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    persisted_messages = [ModelRequest(parts=[UserPromptPart("Earlier question")])]
    captured_run_kwargs: dict[str, Any] = {}
    chat = SimpleNamespace(
        id="chat-123",
        org_id="test-org",
        user_id="test-user",
        canvas_id="canvas-123",
    )
    store = MagicMock()
    store.get_agent_chat.return_value = chat
    store.count_chat_model_messages.side_effect = lambda chat_id: len(persisted_messages)
    store.load_agent_chat_message_history.side_effect = lambda chat_id: list(persisted_messages)
    store.set_initial_chat_message_if_missing.return_value = None

    def fake_create_agent_chat_model_message(chat_id: str, message: ModelRequest) -> SimpleNamespace:
        persisted_messages.append(message)
        return SimpleNamespace(id=f"message-{len(persisted_messages)}")

    async def fake_run_stream_events(agent: Any, **kwargs: Any):
        captured_run_kwargs.update(kwargs)
        if False:
            yield None

    store.create_agent_chat_model_message.side_effect = fake_create_agent_chat_model_message
    monkeypatch.setattr(repl_web, "_run_stream_events", fake_run_stream_events)

    request = SimpleNamespace(
        app=SimpleNamespace(state=SimpleNamespace(session_store=store)),
        headers={},
    )

    async def consume_events() -> list[dict[str, Any]]:
        events: list[dict[str, Any]] = []
        async for event in repl_web._stream_agent_run(
            "chat-123",
            repl_web.ReplStreamRequest(question="Current question", model="test"),
            request,
        ):
            events.append(event)
        return events

    events = asyncio.run(consume_events())

    assert [event["type"] for event in events] == ["run_started", "run_completed", "done"]
    assert captured_run_kwargs["user_prompt"] == "Current question"
    assert "message_history" in captured_run_kwargs
    history = list(captured_run_kwargs["message_history"])
    assert len(history) == 1
    assert history[0].parts[0].content == "Earlier question"
    assert all(
        not isinstance(part, UserPromptPart) or part.content != "Current question"
        for message in history
        for part in message.parts
    )


def test_persisted_run_recorder_does_not_duplicate_final_assistant_message() -> None:
    store = MagicMock()
    store.count_chat_model_messages.return_value = 0
    store.create_agent_chat_model_message.side_effect = [
        SimpleNamespace(id="user-message-1"),
        SimpleNamespace(id="tool-message-1"),
    ]

    recorder = repl_web.PersistedRunRecorder(store, "chat-123", "Current question")
    recorder.tool_finished(
        SimpleNamespace(
            result=ToolReturnPart(
                tool_name="get_canvas",
                content={"ok": True},
                tool_call_id="tool-call-123",
            ),
            content=None,
        )
    )
    recorder.save_authoritative_messages([ModelResponse(parts=[TextPart("Final answer")])])
    recorder.set_assistant_content("Final answer")

    assert store.create_agent_chat_model_message.call_count == 2
    store.replace_agent_chat_messages_after.assert_called_once()
    store.update_agent_chat_model_message.assert_not_called()


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
        stream_url: str,
        payload: CanvasQuestionRequest,
        model: str,
        token: str | None = None,
    ) -> str:
        stream_calls.append(
            {
                "stream_url": stream_url,
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
            "stream_url": "http://agent:8090/agents/chats/chat-123/stream",
            "canvas_id": "canvas-123",
            "model": "anthropic:claude-sonnet-4-6",
            "token": "session-token-123",
        }
    ]
