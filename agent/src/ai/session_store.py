import os
import uuid
from dataclasses import dataclass
from datetime import UTC, datetime
from typing import Any

import psycopg
from psycopg.rows import dict_row
from psycopg.types.json import Jsonb
from pydantic_ai.messages import (
    ModelMessage,
    ModelMessagesTypeAdapter,
    ModelRequest,
    ModelResponse,
    RetryPromptPart,
    TextPart,
    ToolReturnPart,
    UserPromptPart,
)


def _utcnow() -> datetime:
    return datetime.now(UTC)


def _from_db_time(value: datetime | str) -> datetime:
    if isinstance(value, datetime):
        return value.astimezone(UTC)
    return datetime.fromisoformat(value).astimezone(UTC)


def _format_tool_label(tool_name: str) -> str:
    normalized = tool_name.strip().lower()
    label_by_tool = {
        "get_canvas_shape": "Reading canvas structure",
        "get_canvas_details": "Reading canvas details",
        "list_available_blocks": "Listing available components",
    }
    if normalized in label_by_tool:
        return label_by_tool[normalized]

    words = normalized.replace("_", " ").replace("-", " ").strip()
    if not words:
        return "Running tool"
    return words[:1].upper() + words[1:]


def _user_content_to_text(content: Any) -> str:
    if isinstance(content, str):
        return content
    if not isinstance(content, list):
        return ""

    text_parts: list[str] = []
    for part in content:
        if isinstance(part, str):
            text_parts.append(part)
    return "\n".join(part for part in text_parts if part)


def _message_timestamp(message: ModelMessage) -> datetime:
    timestamp = getattr(message, "timestamp", None)
    if isinstance(timestamp, datetime):
        return timestamp.astimezone(UTC)
    return _utcnow()


def _serialize_model_message(message: ModelMessage) -> dict[str, Any]:
    payload = ModelMessagesTypeAdapter.dump_python([message], mode="json")
    if not payload or not isinstance(payload[0], dict):
        raise ValueError("Failed to serialize model message.")
    return payload[0]


def _deserialize_model_message(payload: Any) -> ModelMessage:
    messages = ModelMessagesTypeAdapter.validate_python([payload])
    if not messages:
        raise ValueError("Failed to deserialize model message.")
    return messages[0]


@dataclass(frozen=True)
class StoredAgentChat:
    id: str
    org_id: str
    user_id: str
    canvas_id: str
    initial_message: str | None
    created_at: datetime
    updated_at: datetime


@dataclass(frozen=True)
class StoredAgentChatMessageRecord:
    id: str
    chat_id: str
    message: dict[str, Any]
    sort_index: int
    created_at: datetime
    updated_at: datetime


@dataclass(frozen=True)
class StoredAgentChatMessage:
    id: str
    chat_id: str
    role: str
    content: str
    tool_call_id: str | None
    tool_status: str | None
    created_at: datetime


@dataclass(frozen=True)
class SessionStoreConfig:
    host: str
    port: int
    dbname: str
    user: str
    password: str
    sslmode: str
    application_name: str

    @classmethod
    def from_env(cls) -> "SessionStoreConfig":
        host = (os.getenv("DB_HOST") or "db").strip()
        port = int((os.getenv("DB_PORT") or "5432").strip())
        dbname = (os.getenv("DB_NAME") or "").strip()
        user = (os.getenv("DB_USERNAME") or "").strip()
        password = (os.getenv("DB_PASSWORD") or "").strip()
        sslmode = (os.getenv("DB_SSLMODE") or "disable").strip() or "disable"
        application_name = (os.getenv("APPLICATION_NAME") or "superplane-agent").strip() or "superplane-agent"

        missing_fields = [
            name
            for name, value in (
                ("DB_NAME", dbname),
                ("DB_USERNAME", user),
                ("DB_PASSWORD", password),
            )
            if not value
        ]
        if missing_fields:
            joined = ", ".join(missing_fields)
            raise ValueError(f"Missing required agent database settings: {joined}")

        return cls(
            host=host,
            port=port,
            dbname=dbname,
            user=user,
            password=password,
            sslmode=sslmode,
            application_name=application_name,
        )


class AgentChatNotFoundError(Exception):
    pass


class SessionStore:
    def __init__(self, config: SessionStoreConfig | None = None) -> None:
        self._config = config or SessionStoreConfig.from_env()

    def _connect(self) -> psycopg.Connection[Any]:
        return psycopg.connect(
            host=self._config.host,
            port=self._config.port,
            dbname=self._config.dbname,
            user=self._config.user,
            password=self._config.password,
            sslmode=self._config.sslmode,
            application_name=self._config.application_name,
            row_factory=dict_row,
        )

    def create_agent_chat(self, org_id: str, user_id: str, canvas_id: str, chat_id: str | None = None) -> StoredAgentChat:
        now = _utcnow()
        chat = StoredAgentChat(
            id=chat_id or str(uuid.uuid4()),
            org_id=org_id,
            user_id=user_id,
            canvas_id=canvas_id,
            initial_message=None,
            created_at=now,
            updated_at=now,
        )

        with self._connect() as conn, conn.cursor() as cur:
            cur.execute(
                """
                INSERT INTO agent_chats (
                    id, org_id, user_id, canvas_id, initial_message, created_at, updated_at
                ) VALUES (%s, %s, %s, %s, %s, %s, %s)
                """,
                (
                    chat.id,
                    chat.org_id,
                    chat.user_id,
                    chat.canvas_id,
                    chat.initial_message,
                    chat.created_at,
                    chat.updated_at,
                ),
            )

        return chat

    def list_agent_chats(self, org_id: str, user_id: str, canvas_id: str) -> list[StoredAgentChat]:
        with self._connect() as conn, conn.cursor() as cur:
            cur.execute(
                """
                SELECT *
                FROM agent_chats
                WHERE org_id = %s AND user_id = %s AND canvas_id = %s
                ORDER BY created_at DESC
                """,
                (org_id, user_id, canvas_id),
            )
            rows = cur.fetchall()

        return [self._row_to_agent_chat(row) for row in rows]

    def describe_agent_chat(self, org_id: str, user_id: str, canvas_id: str, chat_id: str) -> StoredAgentChat:
        chat = self.get_agent_chat(chat_id)
        if chat.org_id != org_id or chat.user_id != user_id or chat.canvas_id != canvas_id:
            raise AgentChatNotFoundError(chat_id)

        return chat

    def get_agent_chat(self, chat_id: str) -> StoredAgentChat:
        with self._connect() as conn, conn.cursor() as cur:
            cur.execute(
                """
                SELECT *
                FROM agent_chats
                WHERE id = %s
                LIMIT 1
                """,
                (chat_id,),
            )
            row = cur.fetchone()

        if row is None:
            raise AgentChatNotFoundError(chat_id)

        return self._row_to_agent_chat(row)

    def count_chat_model_messages(self, chat_id: str) -> int:
        with self._connect() as conn, conn.cursor() as cur:
            cur.execute(
                "SELECT COUNT(*) AS message_count FROM agent_chat_messages WHERE chat_id = %s",
                (chat_id,),
            )
            row = cur.fetchone()
        return int(row["message_count"]) if row is not None else 0

    def list_agent_chat_message_records(self, chat_id: str) -> list[StoredAgentChatMessageRecord]:
        with self._connect() as conn, conn.cursor() as cur:
            cur.execute(
                """
                SELECT *
                FROM agent_chat_messages
                WHERE chat_id = %s
                ORDER BY sort_index ASC
                """,
                (chat_id,),
            )
            rows = cur.fetchall()

        return [self._row_to_message_record(row) for row in rows]

    def list_agent_chat_messages(self, org_id: str, user_id: str, canvas_id: str, chat_id: str) -> list[StoredAgentChatMessage]:
        self.describe_agent_chat(org_id, user_id, canvas_id, chat_id)
        records = self.list_agent_chat_message_records(chat_id)

        flattened: list[StoredAgentChatMessage] = []
        for record in records:
            flattened.extend(self._flatten_message_record(record))
        return flattened

    def load_agent_chat_message_history(self, chat_id: str) -> list[ModelMessage]:
        records = self.list_agent_chat_message_records(chat_id)
        return [_deserialize_model_message(record.message) for record in records]

    def create_agent_chat_model_message(self, chat_id: str, message: ModelMessage) -> StoredAgentChatMessageRecord:
        now = _utcnow()
        serialized_message = _serialize_model_message(message)
        created_at = _message_timestamp(message)

        with self._connect() as conn, conn.cursor() as cur:
            cur.execute("SELECT id FROM agent_chats WHERE id = %s FOR UPDATE", (chat_id,))
            if cur.fetchone() is None:
                raise AgentChatNotFoundError(chat_id)

            cur.execute(
                """
                SELECT COALESCE(MAX(sort_index), 0) AS max_sort_index
                FROM agent_chat_messages
                WHERE chat_id = %s
                """,
                (chat_id,),
            )
            row = cur.fetchone()
            next_sort_index = int(row["max_sort_index"]) + 1 if row is not None else 1

            record = StoredAgentChatMessageRecord(
                id=str(uuid.uuid4()),
                chat_id=chat_id,
                message=serialized_message,
                sort_index=next_sort_index,
                created_at=created_at,
                updated_at=now,
            )

            cur.execute(
                """
                INSERT INTO agent_chat_messages (
                    id, chat_id, message, sort_index, created_at, updated_at
                ) VALUES (%s, %s, %s, %s, %s, %s)
                """,
                (
                    record.id,
                    record.chat_id,
                    Jsonb(record.message),
                    record.sort_index,
                    record.created_at,
                    record.updated_at,
                ),
            )
            cur.execute("UPDATE agent_chats SET updated_at = %s WHERE id = %s", (now, chat_id))

        return record

    def update_agent_chat_model_message(self, message_id: str, message: ModelMessage) -> None:
        now = _utcnow()
        serialized_message = _serialize_model_message(message)
        created_at = _message_timestamp(message)

        with self._connect() as conn, conn.cursor() as cur:
            cur.execute(
                """
                UPDATE agent_chat_messages
                SET message = %s, created_at = %s, updated_at = %s
                WHERE id = %s
                RETURNING chat_id
                """,
                (Jsonb(serialized_message), created_at, now, message_id),
            )
            row = cur.fetchone()
            if row is not None:
                cur.execute("UPDATE agent_chats SET updated_at = %s WHERE id = %s", (now, row["chat_id"]))

    def replace_agent_chat_messages_after(
        self,
        chat_id: str,
        preserved_message_count: int,
        messages: list[ModelMessage],
    ) -> None:
        now = _utcnow()
        with self._connect() as conn, conn.cursor() as cur:
            cur.execute("SELECT id FROM agent_chats WHERE id = %s FOR UPDATE", (chat_id,))
            if cur.fetchone() is None:
                raise AgentChatNotFoundError(chat_id)

            cur.execute(
                "DELETE FROM agent_chat_messages WHERE chat_id = %s AND sort_index > %s",
                (chat_id, preserved_message_count),
            )

            sort_index = preserved_message_count
            for message in messages:
                sort_index += 1
                serialized_message = _serialize_model_message(message)
                created_at = _message_timestamp(message)
                cur.execute(
                    """
                    INSERT INTO agent_chat_messages (
                        id, chat_id, message, sort_index, created_at, updated_at
                    ) VALUES (%s, %s, %s, %s, %s, %s)
                    """,
                    (
                        str(uuid.uuid4()),
                        chat_id,
                        Jsonb(serialized_message),
                        sort_index,
                        created_at,
                        now,
                    ),
                )

            cur.execute("UPDATE agent_chats SET updated_at = %s WHERE id = %s", (now, chat_id))

    def set_initial_chat_message_if_missing(self, chat_id: str, initial_message: str) -> None:
        if not initial_message.strip():
            return

        now = _utcnow()
        with self._connect() as conn, conn.cursor() as cur:
            cur.execute(
                """
                UPDATE agent_chats
                SET initial_message = COALESCE(NULLIF(initial_message, ''), %s), updated_at = %s
                WHERE id = %s
                """,
                (initial_message.strip(), now, chat_id),
            )

    def _flatten_message_record(self, record: StoredAgentChatMessageRecord) -> list[StoredAgentChatMessage]:
        model_message = _deserialize_model_message(record.message)
        flattened: list[StoredAgentChatMessage] = []

        if isinstance(model_message, ModelRequest):
            for index, part in enumerate(model_message.parts):
                if isinstance(part, UserPromptPart):
                    content = _user_content_to_text(part.content)
                    if not content:
                        continue
                    flattened.append(
                        StoredAgentChatMessage(
                            id=f"{record.id}:{index}",
                            chat_id=record.chat_id,
                            role="user",
                            content=content,
                            tool_call_id=None,
                            tool_status=None,
                            created_at=record.created_at,
                        )
                    )
                    continue

                if isinstance(part, ToolReturnPart):
                    flattened.append(
                        StoredAgentChatMessage(
                            id=f"{record.id}:{index}",
                            chat_id=record.chat_id,
                            role="tool",
                            content=_format_tool_label(part.tool_name),
                            tool_call_id=part.tool_call_id,
                            tool_status="completed",
                            created_at=record.created_at,
                        )
                    )
                    continue

                if isinstance(part, RetryPromptPart) and part.tool_name:
                    flattened.append(
                        StoredAgentChatMessage(
                            id=f"{record.id}:{index}",
                            chat_id=record.chat_id,
                            role="tool",
                            content=_format_tool_label(part.tool_name),
                            tool_call_id=part.tool_call_id,
                            tool_status="completed",
                            created_at=record.created_at,
                        )
                    )

            return flattened

        if isinstance(model_message, ModelResponse):
            assistant_parts = [part.content for part in model_message.parts if isinstance(part, TextPart) and part.content]
            if assistant_parts:
                flattened.append(
                    StoredAgentChatMessage(
                        id=record.id,
                        chat_id=record.chat_id,
                        role="assistant",
                        content="".join(assistant_parts),
                        tool_call_id=None,
                        tool_status=None,
                        created_at=record.created_at,
                    )
                )

        return flattened

    def _row_to_agent_chat(self, row: dict[str, Any]) -> StoredAgentChat:
        return StoredAgentChat(
            id=str(row["id"]),
            org_id=str(row["org_id"]),
            user_id=str(row["user_id"]),
            canvas_id=str(row["canvas_id"]),
            initial_message=str(row["initial_message"]) if row["initial_message"] is not None else None,
            created_at=_from_db_time(row["created_at"]),
            updated_at=_from_db_time(row["updated_at"]),
        )

    def _row_to_message_record(self, row: dict[str, Any]) -> StoredAgentChatMessageRecord:
        payload = row["message"]
        if not isinstance(payload, dict):
            raise ValueError("Stored agent chat message payload must be a JSON object.")

        return StoredAgentChatMessageRecord(
            id=str(row["id"]),
            chat_id=str(row["chat_id"]),
            message=payload,
            sort_index=int(row["sort_index"]),
            created_at=_from_db_time(row["created_at"]),
            updated_at=_from_db_time(row["updated_at"]),
        )
