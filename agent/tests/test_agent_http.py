import asyncio
import uuid
from collections.abc import AsyncIterator
from contextlib import asynccontextmanager
from types import SimpleNamespace
from typing import Any
from unittest.mock import AsyncMock, MagicMock

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
from ai.agent_stream_context import AgentStreamRequest
from ai.persisted_run_recorder import PersistedRunRecorder
from ai.session_store import AgentChatNotFoundError
from ai.usage_limit_checker import NoopUsageLimitChecker
from ai.usage_publisher import NoopUsagePublisher
from ai.web import create_app


class FakeModelRequestNode:
    def __init__(self, events: list[Any]) -> None:
        self._events = events

    @asynccontextmanager
    async def stream(self, ctx: Any) -> AsyncIterator[AsyncIterator[Any]]:
        async def iterator() -> AsyncIterator[Any]:
            for event in self._events:
                yield event

        yield iterator()


class FakeCallToolsNode:
    def __init__(self, events: list[Any]) -> None:
        self._events = events

    @asynccontextmanager
    async def stream(self, ctx: Any) -> AsyncIterator[AsyncIterator[Any]]:
        async def iterator() -> AsyncIterator[Any]:
            for event in self._events:
                yield event

        yield iterator()


class FakeRunResult:
    def __init__(self, output: Any, usage: Any, messages: list[Any]) -> None:
        self.output = output
        self._usage = usage
        self._messages = messages

    def usage(self) -> Any:
        return self._usage

    def new_messages(self) -> list[Any]:
        return list(self._messages)


class FakeAgentRun:
    def __init__(self, nodes: list[Any], usage: Any, result: FakeRunResult | None = None) -> None:
        self.ctx = SimpleNamespace()
        self._nodes = nodes
        self._usage = usage
        self._final_result = result
        self._index = 0
        self.result: FakeRunResult | None = None

    @property
    def next_node(self) -> Any:
        return self._nodes[self._index]

    async def next(self, node: Any) -> Any:
        self._index += 1
        if self._index >= len(self._nodes):
            self.result = self._final_result
            return SimpleNamespace()
        return self._nodes[self._index]

    def usage(self) -> Any:
        return self._usage


class FakeIterAgent:
    def __init__(
        self, agent_run: FakeAgentRun, captured_kwargs: dict[str, Any] | None = None
    ) -> None:
        self._agent_run = agent_run
        self._captured_kwargs = captured_kwargs

    @asynccontextmanager
    async def iter(self, **kwargs: Any) -> AsyncIterator[FakeAgentRun]:
        if self._captured_kwargs is not None:
            self._captured_kwargs.update(kwargs)
        yield self._agent_run


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


def test_stream_agent_run_circuit_break_still_records_usage_and_publishes_totals(
    monkeypatch: pytest.MonkeyPatch,
    sqlite_store: Any,
) -> None:
    org_id = str(uuid.uuid4())
    user_id = str(uuid.uuid4())
    canvas_id = str(uuid.uuid4())
    chat = sqlite_store.create_agent_chat(org_id, user_id, canvas_id)

    publisher = MagicMock()
    limit_checker = SimpleNamespace(check_agent_token_limit=AsyncMock())
    request = SimpleNamespace(
        app=SimpleNamespace(
            state=SimpleNamespace(
                session_store=sqlite_store,
                publisher=publisher,
                limit_checker=limit_checker,
            )
        ),
        headers={},
    )

    claims = SimpleNamespace(org_id=org_id, subject=user_id)
    deps = SimpleNamespace(
        client=SimpleNamespace(),
        canvas_id=canvas_id,
        canvas_version_id=None,
        canvas_cache={},
    )
    usage = SimpleNamespace(
        input_tokens=90,
        output_tokens=30,
        total_tokens=120,
        cache_read_tokens=0,
        cache_write_tokens=0,
        requests=3,
        tool_calls=3,
    )
    original_update_run_usage = sqlite_store.update_run_usage
    update_run_usage_calls: list[dict[str, Any]] = []

    class FakeToolCallPart:
        def __init__(self, tool_name: str, args: dict[str, Any], tool_call_id: str) -> None:
            self.tool_name = tool_name
            self.args = args
            self.tool_call_id = tool_call_id

    class FakeToolCallEvent:
        def __init__(self, part: FakeToolCallPart) -> None:
            self.part = part

    class FakeToolResultEvent:
        def __init__(self, result: ToolReturnPart, *, content: str | None = None) -> None:
            self.result = result
            self.content = content

    def tracking_update_run_usage(**kwargs: Any) -> None:
        update_run_usage_calls.append(dict(kwargs))
        original_update_run_usage(**kwargs)

    tool_events: list[Any] = []
    args = {"integration_id": "integration-123", "type": "repository"}
    content = [
        {
            "__tool_empty__": True,
            "message": "No resources found",
        }
    ]
    for index in range(3):
        tool_events.append(
            FakeToolCallEvent(
                FakeToolCallPart(
                    tool_name="list_integration_resources",
                    args=args,
                    tool_call_id=f"call-{index}",
                )
            )
        )
        tool_events.append(
            FakeToolResultEvent(
                ToolReturnPart(
                    tool_name="list_integration_resources",
                    content=content,
                    tool_call_id=f"call-{index}",
                )
            )
        )

    agent_run = FakeAgentRun(
        nodes=[FakeCallToolsNode(tool_events)],
        usage=usage,
    )

    async def fake_curate_canvas_memory_markdown(**kwargs: Any) -> None:
        return None

    monkeypatch.setattr(
        agent_web, "_resolve_agent_context", lambda chat_id, request: (claims, chat)
    )
    monkeypatch.setattr(agent_web, "_build_deps", lambda *args, **kwargs: deps)
    monkeypatch.setattr(agent_web, "build_agent", lambda *args, **kwargs: FakeIterAgent(agent_run))
    monkeypatch.setattr(sqlite_store, "update_run_usage", tracking_update_run_usage)
    monkeypatch.setattr(agent_web, "ModelRequestNode", FakeModelRequestNode)
    monkeypatch.setattr(agent_web, "CallToolsNode", FakeCallToolsNode)
    monkeypatch.setattr(agent_web, "FunctionToolCallEvent", FakeToolCallEvent)
    monkeypatch.setattr(agent_web, "FunctionToolResultEvent", FakeToolResultEvent)
    monkeypatch.setattr(
        agent_web,
        "format_tool_display_label",
        lambda *args, **kwargs: "List integration resources",
    )
    monkeypatch.setattr(agent_web, "register_background_task", lambda task: None)
    monkeypatch.setattr(
        agent_web,
        "curate_canvas_memory_markdown",
        fake_curate_canvas_memory_markdown,
    )

    async def consume_events() -> list[dict[str, Any]]:
        events: list[dict[str, Any]] = []
        async for event in agent_web._stream_agent_run(
            chat.id,
            AgentStreamRequest(question="Inspect repositories", model="gpt-5-mini"),
            request,  # type: ignore[arg-type]
        ):
            events.append(event)
        return events

    events = asyncio.run(consume_events())

    final_answer_event = next(event for event in events if event["type"] == "final_answer")
    assert "repeating the same discovery step" in final_answer_event["output"]["answer"].lower()
    assert final_answer_event["usage"]["input_tokens"] == 90
    assert final_answer_event["usage"]["output_tokens"] == 30
    assert final_answer_event["usage"]["total_tokens"] == 120
    assert len(update_run_usage_calls) == 1

    persisted_chat = sqlite_store.get_agent_chat(chat.id)
    assert persisted_chat.total_input_tokens == 90
    assert persisted_chat.total_output_tokens == 30
    assert persisted_chat.total_tokens == 120

    org_usage = sqlite_store.get_org_usage(org_id)
    assert org_usage.total_input_tokens == 90
    assert org_usage.total_output_tokens == 30
    assert org_usage.total_tokens == 120

    publisher.publish_agent_run_finished.assert_called_once_with(
        organization_id=org_id,
        chat_id=chat.id,
        model="gpt-5-mini",
        input_tokens=90,
        output_tokens=30,
        total_tokens=120,
    )

    event_types = [event["type"] for event in events]
    assert "final_answer" in event_types
    assert event_types[-2:] == ["run_completed", "done"]


def test_stream_agent_run_normal_completion_records_usage_once(
    monkeypatch: pytest.MonkeyPatch,
    sqlite_store: Any,
) -> None:
    org_id = str(uuid.uuid4())
    user_id = str(uuid.uuid4())
    canvas_id = str(uuid.uuid4())
    chat = sqlite_store.create_agent_chat(org_id, user_id, canvas_id)

    publisher = MagicMock()
    limit_checker = SimpleNamespace(check_agent_token_limit=AsyncMock())
    request = SimpleNamespace(
        app=SimpleNamespace(
            state=SimpleNamespace(
                session_store=sqlite_store,
                publisher=publisher,
                limit_checker=limit_checker,
            )
        ),
        headers={},
    )

    claims = SimpleNamespace(org_id=org_id, subject=user_id)
    deps = SimpleNamespace(
        client=SimpleNamespace(),
        canvas_id=canvas_id,
        canvas_version_id=None,
        canvas_cache={},
    )
    usage = SimpleNamespace(
        input_tokens=55,
        output_tokens=13,
        total_tokens=68,
        cache_read_tokens=0,
        cache_write_tokens=0,
        requests=1,
        tool_calls=0,
    )
    result = FakeRunResult(
        output={"answer": "All set"},
        usage=usage,
        messages=[ModelResponse(parts=[TextPart("All set")])],
    )
    agent_run = FakeAgentRun(
        nodes=[FakeModelRequestNode([])],
        usage=usage,
        result=result,
    )
    original_update_run_usage = sqlite_store.update_run_usage
    update_run_usage_calls: list[dict[str, Any]] = []

    def tracking_update_run_usage(**kwargs: Any) -> None:
        update_run_usage_calls.append(dict(kwargs))
        original_update_run_usage(**kwargs)

    async def fake_curate_canvas_memory_markdown(**kwargs: Any) -> None:
        return None

    monkeypatch.setattr(
        agent_web, "_resolve_agent_context", lambda chat_id, request: (claims, chat)
    )
    monkeypatch.setattr(agent_web, "_build_deps", lambda *args, **kwargs: deps)
    monkeypatch.setattr(agent_web, "build_agent", lambda *args, **kwargs: FakeIterAgent(agent_run))
    monkeypatch.setattr(sqlite_store, "update_run_usage", tracking_update_run_usage)
    monkeypatch.setattr(agent_web, "ModelRequestNode", FakeModelRequestNode)
    monkeypatch.setattr(agent_web, "CallToolsNode", FakeCallToolsNode)
    monkeypatch.setattr(agent_web, "register_background_task", lambda task: None)
    monkeypatch.setattr(
        agent_web,
        "curate_canvas_memory_markdown",
        fake_curate_canvas_memory_markdown,
    )

    async def consume_events() -> list[dict[str, Any]]:
        events: list[dict[str, Any]] = []
        async for event in agent_web._stream_agent_run(
            chat.id,
            AgentStreamRequest(question="Summarize the canvas", model="gpt-5-mini"),
            request,  # type: ignore[arg-type]
        ):
            events.append(event)
        return events

    events = asyncio.run(consume_events())

    final_answer_event = next(event for event in events if event["type"] == "final_answer")
    assert final_answer_event["output"]["answer"] == "All set"
    assert final_answer_event["usage"]["input_tokens"] == 55
    assert final_answer_event["usage"]["output_tokens"] == 13
    assert final_answer_event["usage"]["total_tokens"] == 68
    assert len(update_run_usage_calls) == 1

    publisher.publish_agent_run_finished.assert_called_once_with(
        organization_id=org_id,
        chat_id=chat.id,
        model="gpt-5-mini",
        input_tokens=55,
        output_tokens=13,
        total_tokens=68,
    )

    persisted_chat = sqlite_store.get_agent_chat(chat.id)
    assert persisted_chat.total_input_tokens == 55
    assert persisted_chat.total_output_tokens == 13
    assert persisted_chat.total_tokens == 68

    event_types = [event["type"] for event in events]
    assert "model_delta" in event_types
    assert event_types[-2:] == ["run_completed", "done"]
