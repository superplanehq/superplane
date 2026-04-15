import json
import logging
import uuid
from collections.abc import Iterator
from contextlib import contextmanager
from dataclasses import dataclass, replace
from datetime import UTC, datetime
from typing import Any

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
from sqlalchemy import delete, func, select
from sqlalchemy.dialects.postgresql import insert as pg_insert
from sqlalchemy.engine import Engine
from sqlalchemy.orm import Session, sessionmaker

from ai.config import config
from db.db import build_engine, build_session_factory
from db.models import (
    AgentCanvasMarkdownMemory,
    AgentChat,
    AgentChatMessage,
    AgentChatRun,
)

logger = logging.getLogger(__name__)

_SUPERPLANE_TOOL_DISPLAY_LABEL_KEY = "superplane_display_label"


def _utcnow() -> datetime:
    return datetime.now(UTC)


def _from_db_time(value: datetime) -> datetime:
    """Normalize a timezone-aware datetime to UTC.

    Postgres always returns aware datetimes for ``timestamptz`` columns, so this
    is a simple timezone conversion."""
    return value.astimezone(UTC)


def _likely_output_tool_name(tool_name: str | None) -> bool:
    if not isinstance(tool_name, str):
        return False

    return tool_name.strip().lower() in {"final_result", "return_canvasanswer", "canvasanswer"}


def _flatten_tool_message_label(tool_name: str | None, metadata: Any) -> str:
    if isinstance(metadata, dict):
        label = metadata.get(_SUPERPLANE_TOOL_DISPLAY_LABEL_KEY)
        if isinstance(label, str) and label.strip():
            return label.strip()
    cleaned = (tool_name or "").strip()
    return cleaned or "tool"


def apply_tool_display_labels_to_messages(
    messages: list[ModelMessage],
    labels_by_call_id: dict[str, str],
) -> list[ModelMessage]:
    """Attach UI labels to tool return parts before persisting (replay uses metadata on read)."""
    if not labels_by_call_id:
        return messages

    out: list[ModelMessage] = []
    for message in messages:
        if not isinstance(message, ModelRequest):
            out.append(message)
            continue

        new_parts: list[Any] = []
        changed = False
        for part in message.parts:
            if isinstance(part, ToolReturnPart) and not _likely_output_tool_name(part.tool_name):
                label = labels_by_call_id.get(part.tool_call_id)
                if isinstance(label, str) and label.strip():
                    existing_meta = part.metadata if isinstance(part.metadata, dict) else {}
                    merged_meta = {
                        **existing_meta,
                        _SUPERPLANE_TOOL_DISPLAY_LABEL_KEY: label.strip(),
                    }
                    new_parts.append(replace(part, metadata=merged_meta))
                    changed = True
                    continue
            new_parts.append(part)

        if changed:
            out.append(replace(message, parts=new_parts))
        else:
            out.append(message)

    return out


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


def _extract_output_tool_answer(payload: dict[str, Any]) -> str:
    parts = payload.get("parts")
    if not isinstance(parts, list):
        return ""

    for part in reversed(parts):
        if not isinstance(part, dict):
            continue
        if part.get("part_kind") != "tool-call":
            continue
        if not _likely_output_tool_name(part.get("tool_name")):
            continue

        args = part.get("args")
        if isinstance(args, str):
            try:
                args = json.loads(args)
            except json.JSONDecodeError:
                continue

        if not isinstance(args, dict):
            continue

        answer = args.get("answer")
        if isinstance(answer, str) and answer:
            return answer

    return ""


@dataclass(frozen=True)
class StoredAgentChat:
    id: str
    org_id: str
    user_id: str
    canvas_id: str
    initial_message: str | None
    created_at: datetime
    updated_at: datetime
    total_input_tokens: int = 0
    total_output_tokens: int = 0
    total_tokens: int = 0


@dataclass(frozen=True)
class StoredAgentChatUsage:
    total_input_tokens: int
    total_output_tokens: int
    total_tokens: int


@dataclass(frozen=True)
class StoredAgentChatMessageRecord:
    id: str
    chat_id: str
    message_index: int
    message: dict[str, Any]
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
        missing_fields = [
            name
            for name, value in (
                ("DB_NAME", config.db_name),
                ("DB_USERNAME", config.db_username),
                ("DB_PASSWORD", config.db_password),
            )
            if not value
        ]
        if missing_fields:
            joined = ", ".join(missing_fields)
            raise ValueError(f"Missing required agent database settings: {joined}")

        return cls(
            host=config.db_host,
            port=config.db_port,
            dbname=config.db_name,
            user=config.db_username,
            password=config.db_password,
            sslmode=config.db_sslmode,
            application_name=config.application_name,
        )


class AgentChatNotFoundError(Exception):
    pass


class SessionStore:
    def __init__(self, config: SessionStoreConfig | None = None) -> None:
        self._config = config or SessionStoreConfig.from_env()
        self._engine: Engine = build_engine(
            host=self._config.host,
            port=self._config.port,
            dbname=self._config.dbname,
            user=self._config.user,
            password=self._config.password,
            sslmode=self._config.sslmode,
            application_name=self._config.application_name,
        )
        self._session_factory: sessionmaker[Session] = build_session_factory(self._engine)

    @contextmanager
    def _session(self) -> Iterator[Session]:
        """Provide a short-lived session scope. Callers that mutate data must
        wrap their work in ``with session.begin():`` so that changes are
        committed atomically. Read-only callers may use the session directly.
        Rollback on failure is handled by ``session.begin()``; ``session.close()``
        releases the connection back to the pool."""
        session = self._session_factory()
        try:
            yield session
        finally:
            session.close()

    @staticmethod
    def _lock_chat(session: Session, chat_id: uuid.UUID) -> AgentChat | None:
        """SELECT ... FOR UPDATE on a chat row. Returns None if not found."""
        return session.execute(
            select(AgentChat).where(AgentChat.id == chat_id).with_for_update()
        ).scalar_one_or_none()

    def close(self) -> None:
        self._engine.dispose()

    # ---- canvas memory ----

    def get_canvas_memory_markdown(self, canvas_id: str) -> str:
        with self._session() as session:
            row = session.get(AgentCanvasMarkdownMemory, uuid.UUID(canvas_id))
            if row is None:
                return ""
            body = row.markdown_body
            return body if isinstance(body, str) else ""

    def set_canvas_memory_markdown(self, canvas_id: str, body: str) -> None:
        text = body.strip()
        cid = uuid.UUID(canvas_id)
        with self._session() as session:
            with session.begin():
                stmt = pg_insert(AgentCanvasMarkdownMemory).values(
                    canvas_id=cid,
                    markdown_body=text,
                    updated_at=func.now(),
                )
                stmt = stmt.on_conflict_do_update(
                    index_elements=[AgentCanvasMarkdownMemory.canvas_id],
                    set_={
                        "markdown_body": stmt.excluded.markdown_body,
                        "updated_at": func.now(),
                    },
                )
                session.execute(stmt)

    # ---- agent chats ----

    def create_agent_chat(
        self, org_id: str, user_id: str, canvas_id: str, chat_id: str | None = None
    ) -> StoredAgentChat:
        now = _utcnow()
        row = AgentChat(
            id=uuid.UUID(chat_id) if chat_id else uuid.uuid4(),
            org_id=uuid.UUID(org_id),
            user_id=uuid.UUID(user_id),
            canvas_id=uuid.UUID(canvas_id),
            initial_message=None,
            created_at=now,
            updated_at=now,
            total_input_tokens=0,
            total_output_tokens=0,
            total_tokens=0,
        )

        with self._session() as session:
            with session.begin():
                session.add(row)
            return self._to_stored_chat(row)

    def list_agent_chats(self, org_id: str, user_id: str, canvas_id: str) -> list[StoredAgentChat]:
        stmt = (
            select(AgentChat)
            .where(
                AgentChat.org_id == uuid.UUID(org_id),
                AgentChat.user_id == uuid.UUID(user_id),
                AgentChat.canvas_id == uuid.UUID(canvas_id),
            )
            .order_by(AgentChat.created_at.desc())
        )

        with self._session() as session:
            rows = session.scalars(stmt).all()
            return [self._to_stored_chat(row) for row in rows]

    def describe_agent_chat(
        self, org_id: str, user_id: str, canvas_id: str, chat_id: str
    ) -> StoredAgentChat:
        chat = self.get_agent_chat(chat_id)
        if chat.org_id != org_id or chat.user_id != user_id or chat.canvas_id != canvas_id:
            raise AgentChatNotFoundError(chat_id)

        return chat

    def get_agent_chat(self, chat_id: str) -> StoredAgentChat:
        with self._session() as session:
            row = session.get(AgentChat, uuid.UUID(chat_id))
            if row is None:
                raise AgentChatNotFoundError(chat_id)
            return self._to_stored_chat(row)

    # ---- chat message counts ----

    def count_chat_model_messages(self, chat_id: str) -> int:
        stmt = (
            select(func.count())
            .select_from(AgentChatMessage)
            .where(AgentChatMessage.chat_id == uuid.UUID(chat_id))
        )

        with self._session() as session:
            result = session.execute(stmt).scalar()
            return int(result) if result is not None else 0

    # ---- chat message records ----

    def list_agent_chat_message_records(self, chat_id: str) -> list[StoredAgentChatMessageRecord]:
        stmt = (
            select(AgentChatMessage)
            .where(AgentChatMessage.chat_id == uuid.UUID(chat_id))
            .order_by(AgentChatMessage.message_index.asc())
        )

        with self._session() as session:
            rows = session.scalars(stmt).all()
            return [self._to_message_record(row) for row in rows]

    def list_agent_chat_messages(
        self, org_id: str, user_id: str, canvas_id: str, chat_id: str
    ) -> list[StoredAgentChatMessage]:
        self.describe_agent_chat(org_id, user_id, canvas_id, chat_id)
        records = self.list_agent_chat_message_records(chat_id)

        flattened: list[StoredAgentChatMessage] = []
        for record in records:
            try:
                flattened.extend(self._flatten_message_record(record))
            except Exception:
                logger.warning(
                    "failed to flatten chat message record chat_id=%s message_id=%s",
                    chat_id,
                    record.id,
                    exc_info=True,
                )
        return flattened

    def load_agent_chat_message_history(self, chat_id: str) -> list[ModelMessage]:
        records = self.list_agent_chat_message_records(chat_id)
        history: list[ModelMessage] = []
        for record in records:
            try:
                history.append(_deserialize_model_message(record.message))
            except Exception:
                logger.warning(
                    "failed to deserialize chat history record chat_id=%s message_id=%s",
                    chat_id,
                    record.id,
                    exc_info=True,
                )
        return history

    def create_agent_chat_model_message(
        self, chat_id: str, message: ModelMessage, run_id: str | None = None
    ) -> StoredAgentChatMessageRecord:
        now = _utcnow()
        serialized_message = _serialize_model_message(message)
        created_at = _message_timestamp(message)
        message_id = uuid.uuid4()
        cid = uuid.UUID(chat_id)

        with self._session() as session:
            with session.begin():
                chat = self._lock_chat(session, cid)
                if chat is None:
                    raise AgentChatNotFoundError(chat_id)

                message_index = self._next_message_index(session, cid)
                row = AgentChatMessage(
                    id=message_id,
                    chat_id=cid,
                    run_id=uuid.UUID(run_id) if run_id else None,
                    message_index=message_index,
                    message=serialized_message,
                    created_at=created_at,
                    updated_at=now,
                )
                session.add(row)
                chat.updated_at = now

        return StoredAgentChatMessageRecord(
            id=str(message_id),
            chat_id=chat_id,
            message_index=message_index,
            message=serialized_message,
            created_at=created_at,
            updated_at=now,
        )

    def update_agent_chat_model_message(self, message_id: str, message: ModelMessage) -> None:
        now = _utcnow()
        serialized_message = _serialize_model_message(message)
        created_at = _message_timestamp(message)
        mid = uuid.UUID(message_id)

        with self._session() as session:
            with session.begin():
                row = session.get(AgentChatMessage, mid)
                if row is None:
                    logger.warning(
                        "update_agent_chat_model_message: message %s not found, skipping", mid
                    )
                    return
                row.message = serialized_message
                row.created_at = created_at
                row.updated_at = now

                chat = session.get(AgentChat, row.chat_id)
                if chat is not None:
                    chat.updated_at = now

    def replace_agent_chat_messages_after(
        self,
        chat_id: str,
        preserved_message_count: int,
        messages: list[ModelMessage],
        run_id: str | None = None,
    ) -> None:
        now = _utcnow()
        cid = uuid.UUID(chat_id)

        with self._session() as session:
            with session.begin():
                chat = self._lock_chat(session, cid)
                if chat is None:
                    raise AgentChatNotFoundError(chat_id)

                session.execute(
                    delete(AgentChatMessage).where(
                        AgentChatMessage.chat_id == cid,
                        AgentChatMessage.message_index >= preserved_message_count,
                    )
                )

                for offset, msg in enumerate(messages):
                    serialized_message = _serialize_model_message(msg)
                    created_at = _message_timestamp(msg)
                    session.add(
                        AgentChatMessage(
                            id=uuid.uuid4(),
                            chat_id=cid,
                            run_id=uuid.UUID(run_id) if run_id else None,
                            message_index=preserved_message_count + offset,
                            message=serialized_message,
                            created_at=created_at,
                            updated_at=now,
                        )
                    )

                chat.updated_at = now

    def set_initial_chat_message_if_missing(self, chat_id: str, initial_message: str) -> None:
        if not initial_message.strip():
            return

        now = _utcnow()
        cid = uuid.UUID(chat_id)
        with self._session() as session:
            with session.begin():
                chat = self._lock_chat(session, cid)
                if chat is None:
                    return
                if not chat.initial_message:
                    chat.initial_message = initial_message.strip()
                    chat.updated_at = now

    # ---- chat runs ----

    def list_distinct_org_ids(self) -> list[str]:
        with self._cursor() as cur:
            cur.execute("SELECT DISTINCT org_id FROM agent_chats")
            return [str(row["org_id"]) for row in cur.fetchall()]

    def delete_expired_chats_for_org(
        self, org_id: str, retention_days: int, batch_size: int = 500
    ) -> int:
        total = 0
        while True:
            with self._cursor() as cur:
                # The outer updated_at re-check guards against a concurrent
                # update (e.g. a new message) that commits between the subquery
                # materializing IDs and the DELETE locking rows. Under READ
                # COMMITTED, PostgreSQL re-evaluates only the outer WHERE after
                # acquiring the row lock, so without this the chat could be
                # deleted despite having just been refreshed.
                cur.execute(
                    """
                    DELETE FROM agent_chats
                    WHERE id IN (
                        SELECT id FROM agent_chats
                        WHERE org_id = %s
                          AND updated_at < NOW() - make_interval(days => %s)
                        LIMIT %s
                    )
                    AND updated_at < NOW() - make_interval(days => %s)
                    """,
                    (org_id, retention_days, batch_size, retention_days),
                )
                deleted = int(cur.rowcount)
                total += deleted
                if deleted < batch_size:
                    break
        return total

    def create_agent_chat_run(self, chat_id: str, model: str) -> str:
        run_id = uuid.uuid4()
        now = _utcnow()
        cid = uuid.UUID(chat_id)

        with self._session() as session:
            with session.begin():
                chat = self._lock_chat(session, cid)
                if chat is None:
                    raise AgentChatNotFoundError(chat_id)

                session.add(
                    AgentChatRun(
                        id=run_id,
                        chat_id=cid,
                        model=model,
                        created_at=now,
                    )
                )

        return str(run_id)

    def update_run_usage(
        self,
        run_id: str,
        input_tokens: int,
        output_tokens: int,
        cache_read_tokens: int,
        cache_write_tokens: int,
        total_tokens: int,
    ) -> None:
        now = _utcnow()
        rid = uuid.UUID(run_id)

        with self._session() as session:
            with session.begin():
                run = session.get(AgentChatRun, rid)
                if run is None:
                    return

                run.input_tokens = input_tokens
                run.output_tokens = output_tokens
                run.cache_read_tokens = cache_read_tokens
                run.cache_write_tokens = cache_write_tokens
                run.total_tokens = total_tokens
                # Flush so the run's updated tokens are visible to the SUM() below.
                session.flush()

                chat = self._lock_chat(session, run.chat_id)
                if chat is None:
                    return

                totals = session.execute(
                    select(
                        func.coalesce(func.sum(AgentChatRun.input_tokens), 0),
                        func.coalesce(func.sum(AgentChatRun.output_tokens), 0),
                        func.coalesce(func.sum(AgentChatRun.total_tokens), 0),
                    ).where(AgentChatRun.chat_id == run.chat_id)
                ).one()

                chat.total_input_tokens = int(totals[0])
                chat.total_output_tokens = int(totals[1])
                chat.total_tokens = int(totals[2])
                chat.updated_at = now

    # ---- org usage ----

    def get_org_usage(self, org_id: str) -> StoredAgentChatUsage:
        stmt = select(
            func.coalesce(func.sum(AgentChat.total_input_tokens), 0),
            func.coalesce(func.sum(AgentChat.total_output_tokens), 0),
            func.coalesce(func.sum(AgentChat.total_tokens), 0),
        ).where(AgentChat.org_id == uuid.UUID(org_id))

        with self._session() as session:
            row = session.execute(stmt).one()

        return StoredAgentChatUsage(
            total_input_tokens=int(row[0]),
            total_output_tokens=int(row[1]),
            total_tokens=int(row[2]),
        )

    # ---- internal helpers ----

    def _flatten_message_record(
        self, record: StoredAgentChatMessageRecord
    ) -> list[StoredAgentChatMessage]:
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
                    if _likely_output_tool_name(part.tool_name):
                        continue
                    tool_label = _flatten_tool_message_label(part.tool_name, part.metadata)
                    flattened.append(
                        StoredAgentChatMessage(
                            id=f"{record.id}:{index}",
                            chat_id=record.chat_id,
                            role="tool",
                            content=tool_label,
                            tool_call_id=part.tool_call_id,
                            tool_status="completed",
                            created_at=record.created_at,
                        )
                    )
                    continue

                if isinstance(part, RetryPromptPart) and part.tool_name:
                    if _likely_output_tool_name(part.tool_name):
                        continue
                    tool_label = _flatten_tool_message_label(
                        part.tool_name, getattr(part, "metadata", None)
                    )
                    flattened.append(
                        StoredAgentChatMessage(
                            id=f"{record.id}:{index}",
                            chat_id=record.chat_id,
                            role="tool",
                            content=tool_label,
                            tool_call_id=part.tool_call_id,
                            tool_status="completed",
                            created_at=record.created_at,
                        )
                    )

            return flattened

        if isinstance(model_message, ModelResponse):
            assistant_parts = [
                part.content
                for part in model_message.parts
                if isinstance(part, TextPart) and part.content
            ]
            assistant_content = "".join(assistant_parts)
            if not assistant_content:
                assistant_content = _extract_output_tool_answer(record.message)
            if assistant_content:
                flattened.append(
                    StoredAgentChatMessage(
                        id=str(record.id),
                        chat_id=record.chat_id,
                        role="assistant",
                        content=assistant_content,
                        tool_call_id=None,
                        tool_status=None,
                        created_at=record.created_at,
                    )
                )

        return flattened

    @staticmethod
    def _to_stored_chat(row: AgentChat) -> StoredAgentChat:
        return StoredAgentChat(
            id=str(row.id),
            org_id=str(row.org_id),
            user_id=str(row.user_id),
            canvas_id=str(row.canvas_id),
            initial_message=str(row.initial_message) if row.initial_message is not None else None,
            created_at=_from_db_time(row.created_at),
            updated_at=_from_db_time(row.updated_at),
            total_input_tokens=int(row.total_input_tokens or 0),
            total_output_tokens=int(row.total_output_tokens or 0),
            total_tokens=int(row.total_tokens or 0),
        )

    @staticmethod
    def _to_message_record(row: AgentChatMessage) -> StoredAgentChatMessageRecord:
        payload = row.message
        if not isinstance(payload, dict):
            raise ValueError("Stored agent chat message payload must be a JSON object.")

        return StoredAgentChatMessageRecord(
            id=str(row.id),
            chat_id=str(row.chat_id),
            message_index=int(row.message_index),
            message=payload,
            created_at=_from_db_time(row.created_at),
            updated_at=_from_db_time(row.updated_at),
        )

    @staticmethod
    def _next_message_index(session: Session, chat_id: uuid.UUID) -> int:
        stmt = select(func.coalesce(func.max(AgentChatMessage.message_index), -1) + 1).where(
            AgentChatMessage.chat_id == chat_id
        )
        result = session.execute(stmt).scalar()
        return int(result) if result is not None else 0
