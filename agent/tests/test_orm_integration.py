"""Smoke tests verifying ORM model mappings work end-to-end via an in-memory SQLite database.

Limitations of the SQLite-backed approach:

- ``set_canvas_memory_markdown`` is not covered because it uses
  ``pg_insert(...).on_conflict_do_update(...)`` (PostgreSQL-specific). Its SQL
  structure is validated via a mock-based test in
  ``test_session_store.py::test_set_canvas_memory_markdown_executes_pg_upsert``.
- ``with_for_update()`` (SELECT ... FOR UPDATE) is silently ignored by SQLite,
  so row-level locking behaviour is not exercised by these tests.
"""

import uuid
from datetime import UTC, datetime

import pytest
from pydantic_ai.messages import ModelRequest, ModelResponse, TextPart, UserPromptPart
from sqlalchemy.orm import Session

from ai.models_db import (
    AgentCanvasMarkdownMemory,
    AgentChat,
    AgentChatMessage,
    AgentChatRun,
    Base,
)
from ai.session_store import (
    AgentChatNotFoundError,
    SessionStore,
)

# ---------------------------------------------------------------------------
# Low-level ORM model tests (direct session, no SessionStore)
# ---------------------------------------------------------------------------


def test_create_all_tables(sqlite_session: Session) -> None:
    tables = set(Base.metadata.tables.keys())
    assert "agent_chats" in tables
    assert "agent_chat_messages" in tables
    assert "agent_chat_runs" in tables
    assert "agent_canvas_markdown_memory" in tables


def test_agent_chat_roundtrip(sqlite_session: Session) -> None:
    now = datetime.now(UTC)
    chat_id = uuid.uuid4()
    chat = AgentChat(
        id=chat_id,
        org_id=uuid.uuid4(),
        user_id=uuid.uuid4(),
        canvas_id=uuid.uuid4(),
        created_at=now,
        updated_at=now,
    )
    sqlite_session.add(chat)
    sqlite_session.commit()

    loaded = sqlite_session.get(AgentChat, chat_id)
    assert loaded is not None
    assert loaded.id == chat_id
    assert loaded.total_input_tokens == 0
    assert loaded.total_output_tokens == 0
    assert loaded.total_tokens == 0
    assert loaded.initial_message is None


def test_chat_message_foreign_key(sqlite_session: Session) -> None:
    now = datetime.now(UTC)
    chat = AgentChat(
        id=uuid.uuid4(),
        org_id=uuid.uuid4(),
        user_id=uuid.uuid4(),
        canvas_id=uuid.uuid4(),
        created_at=now,
        updated_at=now,
    )
    sqlite_session.add(chat)
    sqlite_session.flush()

    msg = AgentChatMessage(
        id=uuid.uuid4(),
        chat_id=chat.id,
        message_index=0,
        message={"kind": "request", "parts": []},
        created_at=now,
        updated_at=now,
    )
    sqlite_session.add(msg)
    sqlite_session.commit()

    loaded = sqlite_session.get(AgentChatMessage, msg.id)
    assert loaded is not None
    assert loaded.chat_id == chat.id
    assert loaded.message == {"kind": "request", "parts": []}


def test_chat_run_foreign_key(sqlite_session: Session) -> None:
    now = datetime.now(UTC)
    chat = AgentChat(
        id=uuid.uuid4(),
        org_id=uuid.uuid4(),
        user_id=uuid.uuid4(),
        canvas_id=uuid.uuid4(),
        created_at=now,
        updated_at=now,
    )
    sqlite_session.add(chat)
    sqlite_session.flush()

    run = AgentChatRun(
        id=uuid.uuid4(),
        chat_id=chat.id,
        model="test-model",
        created_at=now,
    )
    sqlite_session.add(run)
    sqlite_session.commit()

    loaded = sqlite_session.get(AgentChatRun, run.id)
    assert loaded is not None
    assert loaded.model == "test-model"
    assert loaded.input_tokens == 0


def test_canvas_markdown_memory_roundtrip(sqlite_session: Session) -> None:
    now = datetime.now(UTC)
    canvas_id = uuid.uuid4()
    mem = AgentCanvasMarkdownMemory(
        canvas_id=canvas_id,
        markdown_body="## Notes\n\nhello world",
        updated_at=now,
    )
    sqlite_session.add(mem)
    sqlite_session.commit()

    loaded = sqlite_session.get(AgentCanvasMarkdownMemory, canvas_id)
    assert loaded is not None
    assert loaded.markdown_body == "## Notes\n\nhello world"


# ---------------------------------------------------------------------------
# SessionStore integration tests (via sqlite_store fixture)
# ---------------------------------------------------------------------------


def test_store_create_and_get_chat(sqlite_store: SessionStore) -> None:
    org_id = str(uuid.uuid4())
    user_id = str(uuid.uuid4())
    canvas_id = str(uuid.uuid4())

    chat = sqlite_store.create_agent_chat(org_id, user_id, canvas_id)
    assert chat.org_id == org_id
    assert chat.user_id == user_id
    assert chat.canvas_id == canvas_id

    loaded = sqlite_store.get_agent_chat(chat.id)
    assert loaded.id == chat.id
    assert loaded.org_id == org_id


def test_store_get_chat_not_found(sqlite_store: SessionStore) -> None:
    with pytest.raises(AgentChatNotFoundError):
        sqlite_store.get_agent_chat(str(uuid.uuid4()))


def test_store_list_agent_chats(sqlite_store: SessionStore) -> None:
    org_id = str(uuid.uuid4())
    user_id = str(uuid.uuid4())
    canvas_id = str(uuid.uuid4())

    sqlite_store.create_agent_chat(org_id, user_id, canvas_id)
    sqlite_store.create_agent_chat(org_id, user_id, canvas_id)

    chats = sqlite_store.list_agent_chats(org_id, user_id, canvas_id)
    assert len(chats) == 2


def test_store_describe_agent_chat(sqlite_store: SessionStore) -> None:
    org_id = str(uuid.uuid4())
    user_id = str(uuid.uuid4())
    canvas_id = str(uuid.uuid4())

    chat = sqlite_store.create_agent_chat(org_id, user_id, canvas_id)
    described = sqlite_store.describe_agent_chat(org_id, user_id, canvas_id, chat.id)
    assert described.id == chat.id

    with pytest.raises(AgentChatNotFoundError):
        sqlite_store.describe_agent_chat(org_id, user_id, str(uuid.uuid4()), chat.id)


def test_store_canvas_memory_read_returns_empty_for_missing(sqlite_store: SessionStore) -> None:
    assert sqlite_store.get_canvas_memory_markdown(str(uuid.uuid4())) == ""


def test_store_org_usage(sqlite_store: SessionStore) -> None:
    org_id = str(uuid.uuid4())
    usage = sqlite_store.get_org_usage(org_id)
    assert usage.total_input_tokens == 0
    assert usage.total_output_tokens == 0
    assert usage.total_tokens == 0


def test_store_close_disposes_engine(
    sqlite_store: SessionStore, monkeypatch: pytest.MonkeyPatch
) -> None:
    disposed = False

    original_dispose = sqlite_store._engine.dispose

    def tracking_dispose() -> None:
        nonlocal disposed
        disposed = True
        original_dispose()

    monkeypatch.setattr(sqlite_store._engine, "dispose", tracking_dispose)
    sqlite_store.close()
    assert disposed


# ---------------------------------------------------------------------------
# Write-heavy method tests
# ---------------------------------------------------------------------------


def _make_chat(store: SessionStore) -> str:
    chat = store.create_agent_chat(str(uuid.uuid4()), str(uuid.uuid4()), str(uuid.uuid4()))
    return chat.id


def test_store_create_agent_chat_model_message(sqlite_store: SessionStore) -> None:
    chat_id = _make_chat(sqlite_store)
    msg = ModelRequest(parts=[UserPromptPart("hello")])

    record = sqlite_store.create_agent_chat_model_message(chat_id, msg)
    assert record.chat_id == chat_id
    assert record.message_index == 0
    assert record.message["kind"] == "request"


def test_store_create_message_increments_index(sqlite_store: SessionStore) -> None:
    chat_id = _make_chat(sqlite_store)

    r0 = sqlite_store.create_agent_chat_model_message(
        chat_id, ModelRequest(parts=[UserPromptPart("first")])
    )
    r1 = sqlite_store.create_agent_chat_model_message(
        chat_id, ModelResponse(parts=[TextPart("reply")])
    )
    assert r0.message_index == 0
    assert r1.message_index == 1


def test_store_create_message_raises_for_missing_chat(sqlite_store: SessionStore) -> None:
    with pytest.raises(AgentChatNotFoundError):
        sqlite_store.create_agent_chat_model_message(
            str(uuid.uuid4()), ModelRequest(parts=[UserPromptPart("boom")])
        )


def test_store_count_chat_model_messages(sqlite_store: SessionStore) -> None:
    chat_id = _make_chat(sqlite_store)
    assert sqlite_store.count_chat_model_messages(chat_id) == 0

    sqlite_store.create_agent_chat_model_message(chat_id, ModelRequest(parts=[UserPromptPart("a")]))
    sqlite_store.create_agent_chat_model_message(chat_id, ModelResponse(parts=[TextPart("b")]))
    assert sqlite_store.count_chat_model_messages(chat_id) == 2


def test_store_list_agent_chat_message_records(sqlite_store: SessionStore) -> None:
    chat_id = _make_chat(sqlite_store)
    sqlite_store.create_agent_chat_model_message(chat_id, ModelRequest(parts=[UserPromptPart("q")]))
    sqlite_store.create_agent_chat_model_message(chat_id, ModelResponse(parts=[TextPart("a")]))

    records = sqlite_store.list_agent_chat_message_records(chat_id)
    assert len(records) == 2
    assert records[0].message_index == 0
    assert records[1].message_index == 1


def test_store_update_agent_chat_model_message(sqlite_store: SessionStore) -> None:
    chat_id = _make_chat(sqlite_store)
    original = ModelResponse(parts=[TextPart("draft")])
    record = sqlite_store.create_agent_chat_model_message(chat_id, original)

    updated = ModelResponse(parts=[TextPart("final answer")])
    sqlite_store.update_agent_chat_model_message(record.id, updated)

    records = sqlite_store.list_agent_chat_message_records(chat_id)
    assert len(records) == 1
    assert records[0].message["parts"][0]["content"] == "final answer"


def test_store_update_message_noop_for_missing(sqlite_store: SessionStore) -> None:
    sqlite_store.update_agent_chat_model_message(
        str(uuid.uuid4()), ModelResponse(parts=[TextPart("ghost")])
    )


def test_store_replace_messages_raises_for_missing_chat(sqlite_store: SessionStore) -> None:
    with pytest.raises(AgentChatNotFoundError):
        sqlite_store.replace_agent_chat_messages_after(
            str(uuid.uuid4()), 0, [ModelResponse(parts=[TextPart("x")])]
        )


def test_store_replace_agent_chat_messages_after(sqlite_store: SessionStore) -> None:
    chat_id = _make_chat(sqlite_store)
    sqlite_store.create_agent_chat_model_message(
        chat_id, ModelRequest(parts=[UserPromptPart("keep me")])
    )
    sqlite_store.create_agent_chat_model_message(
        chat_id, ModelResponse(parts=[TextPart("old reply")])
    )
    assert sqlite_store.count_chat_model_messages(chat_id) == 2

    new_messages: list[ModelRequest | ModelResponse] = [
        ModelResponse(parts=[TextPart("new reply")]),
        ModelRequest(parts=[UserPromptPart("follow up")]),
    ]
    sqlite_store.replace_agent_chat_messages_after(chat_id, 1, new_messages)

    records = sqlite_store.list_agent_chat_message_records(chat_id)
    assert len(records) == 3
    assert records[0].message_index == 0
    assert records[1].message_index == 1
    assert records[1].message["parts"][0]["content"] == "new reply"
    assert records[2].message_index == 2


def test_store_create_agent_chat_run(sqlite_store: SessionStore) -> None:
    chat_id = _make_chat(sqlite_store)
    run_id = sqlite_store.create_agent_chat_run(chat_id, "claude-sonnet-4-6")
    assert run_id
    assert uuid.UUID(run_id)


def test_store_create_run_raises_for_missing_chat(sqlite_store: SessionStore) -> None:
    with pytest.raises(AgentChatNotFoundError):
        sqlite_store.create_agent_chat_run(str(uuid.uuid4()), "test-model")


def test_store_update_run_usage(sqlite_store: SessionStore) -> None:
    chat_id = _make_chat(sqlite_store)
    run_id = sqlite_store.create_agent_chat_run(chat_id, "test-model")

    sqlite_store.update_run_usage(
        run_id=run_id,
        input_tokens=100,
        output_tokens=50,
        cache_read_tokens=10,
        cache_write_tokens=5,
        total_tokens=150,
    )

    chat = sqlite_store.get_agent_chat(chat_id)
    assert chat.total_input_tokens == 100
    assert chat.total_output_tokens == 50
    assert chat.total_tokens == 150


def test_store_update_run_usage_aggregates_multiple_runs(sqlite_store: SessionStore) -> None:
    chat_id = _make_chat(sqlite_store)
    run1 = sqlite_store.create_agent_chat_run(chat_id, "model-a")
    run2 = sqlite_store.create_agent_chat_run(chat_id, "model-b")

    sqlite_store.update_run_usage(
        run_id=run1,
        input_tokens=100,
        output_tokens=50,
        cache_read_tokens=0,
        cache_write_tokens=0,
        total_tokens=150,
    )
    sqlite_store.update_run_usage(
        run_id=run2,
        input_tokens=200,
        output_tokens=100,
        cache_read_tokens=0,
        cache_write_tokens=0,
        total_tokens=300,
    )

    chat = sqlite_store.get_agent_chat(chat_id)
    assert chat.total_input_tokens == 300
    assert chat.total_output_tokens == 150
    assert chat.total_tokens == 450


def test_store_set_initial_chat_message_if_missing(sqlite_store: SessionStore) -> None:
    chat_id = _make_chat(sqlite_store)

    sqlite_store.set_initial_chat_message_if_missing(chat_id, "  first question  ")
    chat = sqlite_store.get_agent_chat(chat_id)
    assert chat.initial_message == "first question"

    sqlite_store.set_initial_chat_message_if_missing(chat_id, "second question")
    chat = sqlite_store.get_agent_chat(chat_id)
    assert chat.initial_message == "first question"


def test_store_set_initial_chat_message_ignores_blank(sqlite_store: SessionStore) -> None:
    chat_id = _make_chat(sqlite_store)
    sqlite_store.set_initial_chat_message_if_missing(chat_id, "   ")
    chat = sqlite_store.get_agent_chat(chat_id)
    assert chat.initial_message is None


def test_store_load_agent_chat_message_history(sqlite_store: SessionStore) -> None:
    chat_id = _make_chat(sqlite_store)
    sqlite_store.create_agent_chat_model_message(
        chat_id, ModelRequest(parts=[UserPromptPart("hello")])
    )
    sqlite_store.create_agent_chat_model_message(
        chat_id, ModelResponse(parts=[TextPart("hi there")])
    )

    history = sqlite_store.load_agent_chat_message_history(chat_id)
    assert len(history) == 2
    assert isinstance(history[0], ModelRequest)
    assert isinstance(history[1], ModelResponse)


def test_store_list_agent_chat_messages_flattens(sqlite_store: SessionStore) -> None:
    """Exercises the full flatten pipeline (list_agent_chat_messages) end-to-end."""
    org = str(uuid.uuid4())
    user = str(uuid.uuid4())
    canvas = str(uuid.uuid4())
    chat = sqlite_store.create_agent_chat(org, user, canvas)

    sqlite_store.create_agent_chat_model_message(
        chat.id, ModelRequest(parts=[UserPromptPart("what is 2+2?")])
    )
    sqlite_store.create_agent_chat_model_message(chat.id, ModelResponse(parts=[TextPart("4")]))

    messages = sqlite_store.list_agent_chat_messages(org, user, canvas, chat.id)
    assert len(messages) == 2
    assert messages[0].role == "user"
    assert messages[0].content == "what is 2+2?"
    assert messages[1].role == "assistant"
    assert messages[1].content == "4"
