"""Publishes agent run finished events to RabbitMQ for the saas usage service."""

from __future__ import annotations

import queue
import threading

import pika  # type: ignore[import-untyped]
import pika.exceptions  # type: ignore[import-untyped]

from private import agents_pb2

AGENT_EXCHANGE = "superplane.agent-exchange"
AGENT_RUN_FINISHED_ROUTING_KEY = "agent-run-finished"

_STOP_SENTINEL = object()


class UsagePublisher:
    """RabbitMQ publisher that processes all publishes on a single dedicated thread.

    This avoids pika's thread-safety issues (BlockingConnection must only be
    used from the thread that created it) and keeps the connection alive with
    heartbeats between publishes.
    """

    def __init__(self, rabbitmq_url: str) -> None:
        self._rabbitmq_url = rabbitmq_url
        self._queue: queue.Queue[object] = queue.Queue()
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
        self._queue.put(_STOP_SENTINEL)
        self._thread.join(timeout=5.0)

    def _run(self) -> None:
        """Dedicated publisher thread — owns the pika connection."""
        connection: pika.BlockingConnection | None = None
        channel: pika.adapters.blocking_connection.BlockingChannel | None = None

        while True:
            try:
                item = self._queue.get(timeout=1.0)
            except queue.Empty:
                # Process pika heartbeats while idle.
                if connection is not None and connection.is_open:
                    try:
                        connection.process_data_events(time_limit=0)
                    except Exception:
                        connection = None
                        channel = None
                continue

            if item is _STOP_SENTINEL:
                break

            body: bytes = item  # type: ignore[assignment]

            try:
                if connection is None or not connection.is_open:
                    connection, channel = self._connect()

                assert channel is not None
                channel.basic_publish(
                    exchange=AGENT_EXCHANGE,
                    routing_key=AGENT_RUN_FINISHED_ROUTING_KEY,
                    body=body,
                )
            except Exception as error:
                print(f"[web] failed to publish agent run finished: {error}", flush=True)
                connection = None
                channel = None

        if connection is not None and connection.is_open:
            try:
                connection.close()
            except Exception:
                pass

    def _connect(
        self,
    ) -> tuple[pika.BlockingConnection, pika.adapters.blocking_connection.BlockingChannel]:
        params = pika.URLParameters(self._rabbitmq_url)
        connection = pika.BlockingConnection(params)
        channel = connection.channel()
        channel.exchange_declare(exchange=AGENT_EXCHANGE, exchange_type="topic", durable=True)
        return connection, channel


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
