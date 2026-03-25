import os
import sqlite3
import threading
import uuid
from dataclasses import dataclass
from datetime import UTC, datetime
from pathlib import Path


def _utcnow() -> datetime:
    return datetime.now(UTC)


def _to_db_time(value: datetime) -> str:
    return value.astimezone(UTC).isoformat()


def _from_db_time(value: str) -> datetime:
    return datetime.fromisoformat(value)


@dataclass(frozen=True)
class StoredAgent:
    id: str
    org_id: str
    user_id: str
    canvas_id: str
    initial_message: str | None
    message_history_json: str | None
    created_at: datetime
    updated_at: datetime


@dataclass(frozen=True)
class StoredAgentMessage:
    id: str
    agent_id: str
    role: str
    content: str
    tool_call_id: str | None
    tool_status: str | None
    sort_index: int
    created_at: datetime
    updated_at: datetime


class AgentNotFoundError(Exception):
    pass


class SessionStore:
    def __init__(self, db_path: str | None = None) -> None:
        configured_path = (db_path or os.getenv("AGENT_DB_PATH") or "/app/tmp/agent.sqlite3").strip()
        self._db_path = Path(configured_path)
        self._db_path.parent.mkdir(parents=True, exist_ok=True)
        self._schema_lock = threading.Lock()
        self._ensure_schema()

    def _connect(self) -> sqlite3.Connection:
        conn = sqlite3.connect(self._db_path, check_same_thread=False)
        conn.row_factory = sqlite3.Row
        return conn

    def _ensure_schema(self) -> None:
        with self._schema_lock:
            with self._connect() as conn:
                conn.execute(
                    """
                    CREATE TABLE IF NOT EXISTS agents (
                        id TEXT PRIMARY KEY,
                        org_id TEXT NOT NULL,
                        user_id TEXT NOT NULL,
                        canvas_id TEXT NOT NULL,
                        initial_message TEXT,
                        message_history_json TEXT,
                        created_at TEXT NOT NULL,
                        updated_at TEXT NOT NULL
                    )
                    """
                )
                conn.execute(
                    """
                    CREATE INDEX IF NOT EXISTS idx_agents_owner_canvas_created
                    ON agents (org_id, user_id, canvas_id, created_at DESC)
                    """
                )
                conn.execute(
                    """
                    CREATE TABLE IF NOT EXISTS agent_messages (
                        id TEXT PRIMARY KEY,
                        agent_id TEXT NOT NULL,
                        role TEXT NOT NULL,
                        content TEXT NOT NULL,
                        tool_call_id TEXT,
                        tool_status TEXT,
                        sort_index INTEGER NOT NULL,
                        created_at TEXT NOT NULL,
                        updated_at TEXT NOT NULL,
                        FOREIGN KEY(agent_id) REFERENCES agents(id) ON DELETE CASCADE,
                        UNIQUE(agent_id, sort_index)
                    )
                    """
                )
                conn.execute(
                    """
                    CREATE INDEX IF NOT EXISTS idx_agent_messages_agent_sort
                    ON agent_messages (agent_id, sort_index)
                    """
                )
                conn.commit()

    def create_agent(self, org_id: str, user_id: str, canvas_id: str, agent_id: str | None = None) -> StoredAgent:
        now = _utcnow()
        agent = StoredAgent(
            id=agent_id or str(uuid.uuid4()),
            org_id=org_id,
            user_id=user_id,
            canvas_id=canvas_id,
            initial_message=None,
            message_history_json=None,
            created_at=now,
            updated_at=now,
        )

        with self._connect() as conn:
            conn.execute(
                """
                INSERT INTO agents (
                    id, org_id, user_id, canvas_id, initial_message, message_history_json, created_at, updated_at
                ) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
                """,
                (
                    agent.id,
                    agent.org_id,
                    agent.user_id,
                    agent.canvas_id,
                    agent.initial_message,
                    agent.message_history_json,
                    _to_db_time(agent.created_at),
                    _to_db_time(agent.updated_at),
                ),
            )
            conn.commit()

        return agent

    def list_agents(self, org_id: str, user_id: str, canvas_id: str) -> list[StoredAgent]:
        with self._connect() as conn:
            rows = conn.execute(
                """
                SELECT *
                FROM agents
                WHERE org_id = ? AND user_id = ? AND canvas_id = ?
                ORDER BY created_at DESC
                """,
                (org_id, user_id, canvas_id),
            ).fetchall()

        return [self._row_to_agent(row) for row in rows]

    def describe_agent(self, org_id: str, user_id: str, canvas_id: str, agent_id: str) -> StoredAgent:
        agent = self.get_agent(agent_id)
        if agent.org_id != org_id or agent.user_id != user_id or agent.canvas_id != canvas_id:
            raise AgentNotFoundError(agent_id)

        return agent

    def get_agent(self, agent_id: str) -> StoredAgent:
        with self._connect() as conn:
            row = conn.execute(
                """
                SELECT *
                FROM agents
                WHERE id = ?
                LIMIT 1
                """,
                (agent_id,),
            ).fetchone()

        if row is None:
            raise AgentNotFoundError(agent_id)

        return self._row_to_agent(row)

    def list_messages(self, org_id: str, user_id: str, canvas_id: str, agent_id: str) -> list[StoredAgentMessage]:
        self.describe_agent(org_id, user_id, canvas_id, agent_id)

        with self._connect() as conn:
            rows = conn.execute(
                """
                SELECT *
                FROM agent_messages
                WHERE agent_id = ?
                ORDER BY sort_index ASC
                """,
                (agent_id,),
            ).fetchall()

        return [self._row_to_message(row) for row in rows]

    def load_message_history_json(self, agent_id: str) -> str | None:
        with self._connect() as conn:
            row = conn.execute(
                "SELECT message_history_json FROM agents WHERE id = ? LIMIT 1",
                (agent_id,),
            ).fetchone()

        if row is None:
            raise AgentNotFoundError(agent_id)
        value = row["message_history_json"]
        return value if isinstance(value, str) and value else None

    def save_message_history_json(self, agent_id: str, history_json: str) -> None:
        now = _to_db_time(_utcnow())
        with self._connect() as conn:
            conn.execute(
                """
                UPDATE agents
                SET message_history_json = ?, updated_at = ?
                WHERE id = ?
                """,
                (history_json, now, agent_id),
            )
            conn.commit()

    def create_message(
        self,
        agent_id: str,
        role: str,
        content: str,
        tool_call_id: str | None = None,
        tool_status: str | None = None,
    ) -> StoredAgentMessage:
        now = _utcnow()
        message = StoredAgentMessage(
            id=str(uuid.uuid4()),
            agent_id=agent_id,
            role=role,
            content=content,
            tool_call_id=tool_call_id,
            tool_status=tool_status,
            sort_index=self._next_sort_index(agent_id),
            created_at=now,
            updated_at=now,
        )

        with self._connect() as conn:
            conn.execute(
                """
                INSERT INTO agent_messages (
                    id, agent_id, role, content, tool_call_id, tool_status, sort_index, created_at, updated_at
                ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
                """,
                (
                    message.id,
                    message.agent_id,
                    message.role,
                    message.content,
                    message.tool_call_id,
                    message.tool_status,
                    message.sort_index,
                    _to_db_time(message.created_at),
                    _to_db_time(message.updated_at),
                ),
            )
            conn.execute("UPDATE agents SET updated_at = ? WHERE id = ?", (_to_db_time(now), agent_id))
            conn.commit()

        return message

    def append_message_content(self, message_id: str, content_delta: str) -> None:
        if not content_delta:
            return

        now = _to_db_time(_utcnow())
        with self._connect() as conn:
            conn.execute(
                """
                UPDATE agent_messages
                SET content = content || ?, updated_at = ?
                WHERE id = ?
                """,
                (content_delta, now, message_id),
            )
            conn.commit()

    def update_message(
        self,
        message_id: str,
        *,
        content: str | None = None,
        tool_status: str | None = None,
    ) -> None:
        assignments: list[str] = []
        values: list[str] = []
        if content is not None:
            assignments.append("content = ?")
            values.append(content)
        if tool_status is not None:
            assignments.append("tool_status = ?")
            values.append(tool_status)
        if not assignments:
            return

        assignments.append("updated_at = ?")
        values.append(_to_db_time(_utcnow()))
        values.append(message_id)

        with self._connect() as conn:
            conn.execute(
                f"UPDATE agent_messages SET {', '.join(assignments)} WHERE id = ?",
                values,
            )
            conn.commit()

    def set_initial_message_if_missing(self, agent_id: str, initial_message: str) -> None:
        if not initial_message.strip():
            return

        now = _to_db_time(_utcnow())
        with self._connect() as conn:
            conn.execute(
                """
                UPDATE agents
                SET initial_message = COALESCE(NULLIF(initial_message, ''), ?), updated_at = ?
                WHERE id = ?
                """,
                (initial_message.strip(), now, agent_id),
            )
            conn.commit()

    def _next_sort_index(self, agent_id: str) -> int:
        with self._connect() as conn:
            row = conn.execute(
                "SELECT COALESCE(MAX(sort_index), 0) AS max_sort_index FROM agent_messages WHERE agent_id = ?",
                (agent_id,),
            ).fetchone()

        max_sort_index = row["max_sort_index"]
        return int(max_sort_index) + 1

    def _row_to_agent(self, row: sqlite3.Row) -> StoredAgent:
        return StoredAgent(
            id=str(row["id"]),
            org_id=str(row["org_id"]),
            user_id=str(row["user_id"]),
            canvas_id=str(row["canvas_id"]),
            initial_message=str(row["initial_message"]) if row["initial_message"] is not None else None,
            message_history_json=str(row["message_history_json"]) if row["message_history_json"] is not None else None,
            created_at=_from_db_time(str(row["created_at"])),
            updated_at=_from_db_time(str(row["updated_at"])),
        )

    def _row_to_message(self, row: sqlite3.Row) -> StoredAgentMessage:
        return StoredAgentMessage(
            id=str(row["id"]),
            agent_id=str(row["agent_id"]),
            role=str(row["role"]),
            content=str(row["content"]),
            tool_call_id=str(row["tool_call_id"]) if row["tool_call_id"] is not None else None,
            tool_status=str(row["tool_status"]) if row["tool_status"] is not None else None,
            sort_index=int(row["sort_index"]),
            created_at=_from_db_time(str(row["created_at"])),
            updated_at=_from_db_time(str(row["updated_at"])),
        )
