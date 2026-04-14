import asyncio
from collections.abc import AsyncIterator
from types import SimpleNamespace
from typing import Any
from unittest.mock import MagicMock

import pytest
from pydantic_ai.messages import (
    ModelRequest,
    ModelResponse,
    TextPart,
    ToolReturnPart,
    UserPromptPart,
)
from starlette.testclient import TestClient

import ai.web as agent_web
from ai.persisted_run_recorder import PersistedRunRecorder
from ai.session_store import AgentChatNotFoundError
from ai.usage_limit_checker import NoopUsageLimitChecker
from ai.usage_publisher import NoopUsagePublisher
from ai.agent_stream_context import AgentStreamRequest
from ai.web import create_app


@pytest.fixture(autouse=True)
def _stub_agent_persistence(monkeypatch: pytest.MonkeyPatch) -> None:
    fake_store = MagicMock()
    fake_store.get_agent_chat.side_effect = AgentChatNotFoundError("missing")
    fake_store.create_agent_chat.side_effect = lambda org_id, user_id, canvas_id, chat_id=None: (
        SimpleNamespace(
            id=chat_id or "chat-123",
            org_id=org_id,
            user_id=user_id,
            canvas_id=canvas_id,
        )
    )
    fake_store.count_chat_model_messages.return_value = 0
    fake_store.load_agent_chat_message_history.return_value = []
    fake_store.set_initial_chat_message_if_missing.return_value = None
    fake_store.create_agent_chat_run.side_effect = lambda chat_id, model: "run-123"
    fake_store.create_agent_chat_model_message.side_effect = lambda chat_id, message, run_id=None: (
        SimpleNamespace(id="message-123")
    )
    fake_store.update_agent_chat_model_message.return_value = None
    fake_store.replace_agent_chat_messages_after.return_value = None
    monkeypatch.setattr(agent_web, "SessionStore", MagicMock(return_value=fake_store))
    fake_grpc_server = MagicMock()
    fake_grpc_server.start.return_value = None
    fake_grpc_server.stop.return_value = None
    monkeypatch.setattr(
        agent_web.InternalAgentServer,  # type: ignore[attr-defined]
        "from_env",
        MagicMock(return_value=fake_grpc_server),
    )


def test_health_returns_200_with_empty_body() -> None:
    with TestClient(create_app()) as client:
        response = client.get("/health")
        assert response.status_code == 200
        assert response.content == b""


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

    def fake_create_agent_chat_model_message(
        chat_id: str, message: ModelRequest, run_id: str | None = None
    ) -> SimpleNamespace:
        persisted_messages.append(message)
        return SimpleNamespace(id=f"message-{len(persisted_messages)}")

    async def fake_run_stream_events(agent: Any, **kwargs: Any) -> AsyncIterator[None]:
        captured_run_kwargs.update(kwargs)
        if False:
            yield None

    store.create_agent_chat_run.side_effect = lambda chat_id, model: "run-456"
    store.create_agent_chat_model_message.side_effect = fake_create_agent_chat_model_message
    monkeypatch.setattr(agent_web, "_run_stream_events", fake_run_stream_events)

    request = SimpleNamespace(
        app=SimpleNamespace(
            state=SimpleNamespace(
                session_store=store,
                publisher=NoopUsagePublisher(),
                limit_checker=NoopUsageLimitChecker(),
            )
        ),
        headers={},
    )

    async def consume_events() -> list[dict[str, Any]]:
        events: list[dict[str, Any]] = []
        async for event in agent_web._stream_agent_run(
            "chat-123",
            AgentStreamRequest(question="Current question", model="test"),
            request,  # type: ignore[arg-type]
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

    recorder = PersistedRunRecorder(store, "chat-123", "run-789", "Current question")
    recorder.tool_finished(
        SimpleNamespace(  # type: ignore[arg-type]
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
