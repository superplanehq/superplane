"""Publishes agent run finished events to RabbitMQ for the saas usage service."""

from __future__ import annotations

import threading
from collections.abc import Callable

import pika  # type: ignore[import-untyped]
import pika.exceptions  # type: ignore[import-untyped]

from ai.config import config
from private import agents_pb2

AGENT_EXCHANGE = "superplane.agent-exchange"
AGENT_RUN_FINISHED_ROUTING_KEY = "agent-run-finished"


class _Publisher:
    """Thread-safe RabbitMQ publisher that reuses a single connection."""

    def __init__(self) -> None:
        self._lock = threading.Lock()
        self._connection: pika.BlockingConnection | None = None
        self._channel: pika.adapters.blocking_connection.BlockingChannel | None = None

    def publish(self, body: bytes) -> None:
        with self._lock:
            try:
                channel = self._ensure_channel()
                channel.basic_publish(
                    exchange=AGENT_EXCHANGE,
                    routing_key=AGENT_RUN_FINISHED_ROUTING_KEY,
                    body=body,
                )
            except Exception:
                self._close()
                raise

    def _ensure_channel(self) -> pika.adapters.blocking_connection.BlockingChannel:
        has_connection = self._connection is not None and self._connection.is_open
        has_channel = self._channel is not None and self._channel.is_open
        if has_connection and has_channel:
            return self._channel

        self._close()

        rabbitmq_url = config.rabbitmq_url
        params = pika.URLParameters(rabbitmq_url)
        self._connection = pika.BlockingConnection(params)
        self._channel = self._connection.channel()
        self._channel.exchange_declare(exchange=AGENT_EXCHANGE, exchange_type="topic", durable=True)
        return self._channel

    def _close(self) -> None:
        try:
            if self._connection and not self._connection.is_closed:
                self._connection.close()
        except Exception:
            pass
        self._connection = None
        self._channel = None


_publisher = _Publisher()
_executor = threading.Thread  # just used for typing reference below


def _publish_in_background(task: Callable[[], None]) -> None:
    thread = threading.Thread(target=task, daemon=True)
    thread.start()


def publish_agent_run_finished(
    organization_id: str,
    chat_id: str,
    model: str,
    input_tokens: int,
    output_tokens: int,
    total_tokens: int,
) -> None:
    """Publish an agent-run-finished message to RabbitMQ.

    Runs in a background thread to avoid blocking the async event loop.
    Fails silently with a log message if RabbitMQ is unavailable.
    """
    rabbitmq_url = config.rabbitmq_url
    if not rabbitmq_url:
        return

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
    body = message.SerializeToString()

    def _do_publish() -> None:
        try:
            _publisher.publish(body)
        except Exception as error:
            print(f"[web] failed to publish agent run finished: {error}", flush=True)

    _publish_in_background(_do_publish)
