"""Publishes agent run finished events to RabbitMQ for the saas usage service."""

from __future__ import annotations

import queue
import threading
from typing import Protocol

import pika  # type: ignore[import-untyped]
import pika.exceptions  # type: ignore[import-untyped]

from private import agents_pb2

AGENT_EXCHANGE = "superplane.agent-exchange"
AGENT_RUN_FINISHED_ROUTING_KEY = "agent-run-finished"


class AgentUsagePublisher(Protocol):
    def publish_agent_run_finished(
        self,
        organization_id: str,
        chat_id: str,
        model: str,
        input_tokens: int,
        output_tokens: int,
        total_tokens: int,
    ) -> None: ...

    def close(self) -> None: ...


class _Connection:
    """Manages a single pika connection and channel, owned by one thread."""

    def __init__(self) -> None:
        self._conn: pika.BlockingConnection | None = None
        self._chan: pika.adapters.blocking_connection.BlockingChannel | None = None

    @property
    def is_open(self) -> bool:
        return self._conn is not None and self._conn.is_open

    def ensure(self, rabbitmq_url: str) -> pika.adapters.blocking_connection.BlockingChannel:
        if self.is_open and self._chan is not None and self._chan.is_open:
            return self._chan

        self.close()
        params = pika.URLParameters(rabbitmq_url)
        self._conn = pika.BlockingConnection(params)
        self._chan = self._conn.channel()
        self._chan.exchange_declare(exchange=AGENT_EXCHANGE, exchange_type="topic", durable=True)
        return self._chan

    def heartbeat(self) -> None:
        if self._conn is not None and self._conn.is_open:
            self._conn.process_data_events(time_limit=0)

    def close(self) -> None:
        try:
            if self._conn is not None and not self._conn.is_closed:
                self._conn.close()
        except Exception:
            pass
        self._conn = None
        self._chan = None


class UsagePublisher:
    """RabbitMQ publisher that processes all publishes on a single dedicated thread.

    This avoids pika's thread-safety issues (BlockingConnection must only be
    used from the thread that created it) and keeps the connection alive with
    heartbeats between publishes.
    """

    def __init__(self, rabbitmq_url: str) -> None:
        self._rabbitmq_url = rabbitmq_url
        self._queue: queue.Queue[bytes] = queue.Queue()
        self._stopped = threading.Event()
        self._thread = threading.Thread(target=self._run, daemon=True)
        self._thread.start()

    def publish_agent_run_finished(
        self,
        organization_id: str,
        chat_id: str,
        model: str,
        input_tokens: int,
        output_tokens: int,
        total_tokens: int,
    ) -> None:
        """Enqueue a publish. Non-blocking, returns immediately."""
        if total_tokens <= 0:
            return

        message = agents_pb2.AgentRunFinishedMessage(  # type: ignore[attr-defined]
            organization_id=organization_id,
            chat_id=chat_id,
            model=model,
            input_tokens=input_tokens,
            output_tokens=output_tokens,
            total_tokens=total_tokens,
        )
        self._queue.put(message.SerializeToString())

    def close(self) -> None:
        self._stopped.set()
        self._thread.join(timeout=5.0)

    def _run(self) -> None:
        """Dedicated publisher thread — owns the pika connection."""
        conn = _Connection()

        while not self._stopped.is_set() or not self._queue.empty():
            body = self._dequeue(conn)
            if body is not None:
                self._publish(conn, body)

        conn.close()

    def _dequeue(self, conn: _Connection) -> bytes | None:
        try:
            return self._queue.get(timeout=1.0)
        except queue.Empty:
            self._process_heartbeat(conn)
            return None

    def _process_heartbeat(self, conn: _Connection) -> None:
        try:
            conn.heartbeat()
        except Exception:
            conn.close()

    def _publish(self, conn: _Connection, body: bytes) -> None:
        try:
            channel = conn.ensure(self._rabbitmq_url)
            channel.basic_publish(
                exchange=AGENT_EXCHANGE,
                routing_key=AGENT_RUN_FINISHED_ROUTING_KEY,
                body=body,
            )
        except Exception as error:
            print(f"[web] failed to publish agent run finished: {error}", flush=True)
            conn.close()


class NoopUsagePublisher:
    """No-op publisher used when RABBITMQ_URL is not configured."""

    def publish_agent_run_finished(
        self,
        organization_id: str,
        chat_id: str,
        model: str,
        input_tokens: int,
        output_tokens: int,
        total_tokens: int,
    ) -> None:
        pass

    def close(self) -> None:
        pass
