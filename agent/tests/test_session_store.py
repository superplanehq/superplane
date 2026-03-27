from datetime import UTC, datetime

from ai.session_store import SessionStore, SessionStoreConfig, StoredAgentChatMessageRecord


def _build_store() -> SessionStore:
    return SessionStore(
        SessionStoreConfig(
            host="localhost",
            port=5432,
            dbname="test",
            user="test",
            password="test",
            sslmode="disable",
            application_name="test",
        )
    )


def test_flatten_message_record_exposes_output_tool_answer_as_assistant() -> None:
    now = datetime.now(UTC)
    store = _build_store()
    record = StoredAgentChatMessageRecord(
        id=119,
        chat_id="chat-123",
        message={
            "kind": "response",
            "parts": [
                {
                    "id": None,
                    "args": '{"answer":"Your canvas contains two nodes.","confidence":1}',
                    "part_kind": "tool-call",
                    "tool_name": "final_result",
                    "tool_call_id": "toolu_017y2m5nLgupxCUMW3q6zK3b",
                    "provider_name": None,
                    "provider_details": None,
                }
            ],
            "usage": {
                "details": {
                    "input_tokens": 2942,
                    "output_tokens": 407,
                    "cache_read_input_tokens": 0,
                    "cache_creation_input_tokens": 0,
                },
                "input_tokens": 2942,
                "output_tokens": 407,
                "cache_read_tokens": 0,
                "cache_write_tokens": 0,
                "input_audio_tokens": 0,
                "output_audio_tokens": 0,
                "cache_audio_read_tokens": 0,
            },
            "run_id": "43beced1-acec-43b2-abc0-261dc36608ab",
            "metadata": None,
            "timestamp": "2026-03-27T00:01:14.686075Z",
            "model_name": "claude-sonnet-4-6",
            "provider_url": "https://api.anthropic.com",
            "finish_reason": "tool_call",
            "provider_name": "anthropic",
            "provider_details": {
                "finish_reason": "tool_use",
            },
            "provider_response_id": "msg_01LyGq1Au5VTsXtf49SKt3Da",
        },
        created_at=now,
        updated_at=now,
    )

    messages = store._flatten_message_record(record)

    assert len(messages) == 1
    assert messages[0].id == "119"
    assert messages[0].role == "assistant"
    assert messages[0].content == "Your canvas contains two nodes."
    assert messages[0].tool_call_id is None
    assert messages[0].tool_status is None


def test_flatten_message_record_ignores_output_tool_returns() -> None:
    now = datetime.now(UTC)
    store = _build_store()
    record = StoredAgentChatMessageRecord(
        id=120,
        chat_id="chat-123",
        message={
            "kind": "request",
            "parts": [
                {
                    "content": {"name": "test"},
                    "outcome": "success",
                    "metadata": None,
                    "part_kind": "tool-return",
                    "timestamp": "2026-03-27T00:01:12.551360Z",
                    "tool_name": "get_canvas",
                    "tool_call_id": "toolu_012wJyWVffYDcQqR3Ne9W6FN",
                },
                {
                    "content": "Final result processed.",
                    "outcome": "success",
                    "metadata": None,
                    "part_kind": "tool-return",
                    "timestamp": "2026-03-27T00:01:19.265155Z",
                    "tool_name": "final_result",
                    "tool_call_id": "toolu_017y2m5nLgupxCUMW3q6zK3b",
                },
            ],
            "run_id": "43beced1-acec-43b2-abc0-261dc36608ab",
            "metadata": None,
            "timestamp": "2026-03-27T00:01:19.265172Z",
            "instructions": None,
        },
        created_at=now,
        updated_at=now,
    )

    messages = store._flatten_message_record(record)

    assert len(messages) == 1
    assert messages[0].role == "tool"
    assert messages[0].content == "Get canvas"
    assert messages[0].tool_call_id == "toolu_012wJyWVffYDcQqR3Ne9W6FN"
    assert messages[0].tool_status == "completed"


def test_list_agent_chat_messages_skips_unflattenable_records(monkeypatch) -> None:
    now = datetime.now(UTC)
    store = _build_store()
    records = [
        StoredAgentChatMessageRecord(
            id=1,
            chat_id="chat-123",
            message={
                "kind": "request",
                "parts": [
                    {
                        "content": "What is in my canvas?",
                        "part_kind": "user-prompt",
                        "timestamp": "2026-03-27T00:01:11.361230Z",
                    }
                ],
                "run_id": "run-1",
                "metadata": None,
                "timestamp": "2026-03-27T00:01:11.361347Z",
                "instructions": None,
            },
            created_at=now,
            updated_at=now,
        ),
        StoredAgentChatMessageRecord(
            id=2,
            chat_id="chat-123",
            message={"kind": "broken"},
            created_at=now,
            updated_at=now,
        ),
    ]

    monkeypatch.setattr(store, "describe_agent_chat", lambda org_id, user_id, canvas_id, chat_id: None)
    monkeypatch.setattr(store, "list_agent_chat_message_records", lambda chat_id: records)

    messages = store.list_agent_chat_messages("org-123", "user-123", "canvas-123", "chat-123")

    assert len(messages) == 1
    assert messages[0].role == "user"
    assert messages[0].content == "What is in my canvas?"
